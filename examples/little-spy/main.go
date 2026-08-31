// Package main is a learning-purposes example, NOT a template for real apps.
//
// "Little spy" shows annotations at work: the assistant chats with the user
// while quietly trying to learn four facts about them — first name, last name,
// age and where they are from. Every answer the user gives is run through a
// set of annotating LLMCompletionStates (those with the Annotation option
// set). Such a state never appends a chat reply; it sends only its system
// prompt + the user's last message to the model and pins the parsed result
// onto that user message as a named annotation. The annotations accumulate in
// the conversation and, since the conversation is snapshotted, they survive
// across process runs.
//
// Each run of the program is one cycle:
//
//  1. Restore the previous state from the snapshot file, if there is one.
//  2. The user asks a question. The assistant answers it and, being a spy,
//     follows up with one question aimed at a fact it does not know yet.
//  3. The user answers. Four annotating states inspect that answer, each
//     extracting one fact into the annotations "first name", "last name",
//     "age" and "location".
//  4. Print the spy report — everything learned so far — then take a snapshot
//     and write it to disk.
//
// Run it a few times and watch the report fill in:
//
//	$ go run ./examples/little-spy
//	$ go run ./examples/little-spy
//	$ ...
//
// Delete little-spy.json to start over.
//
// The behavior tree is:
//
//	chat -> pause -> extract first name -> extract last name
//	     -> extract age -> extract location -> pause
//
// Two details make the multi-run cycle possible:
//
//   - Every behavior is built with a *stable* id (the ...WithId constructors
//     and explicit state ids). Snapshots reference behaviors by hash, so the
//     hashes must be identical on every run.
//   - The tree ends in a PauseState. Its CollectUserInputSignal keeps the plan
//     step alive in the executive (TakeSnapshot only persists an executive
//     that still has a plan in flight), and because the final pause has no
//     children, the tree starts over from "chat" the next time it is called —
//     with the step's whole conversation, annotations included, still there.
//
// Requires OPENAI_TOKEN to be set in the environment (the default model is
// gpt-4o-mini via the OpenAI provider).
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/zenful-ai/arboreal"
	"github.com/zenful-ai/arboreal/llm"
)

const snapshotFile = "little-spy.json"

// factKeys are the annotation names the spy fills in, in report order.
var factKeys = []string{"first name", "last name", "age", "location"}

// extractor builds an annotating state: it looks at the user's last message
// only and stores what it finds under the annotation `key`. An empty string
// means "this message did not reveal it".
func extractor(key, what, example string) arboreal.BehaviorState {
	return arboreal.LLMCompletionState(arboreal.LLMCompletionOptions{
		Id: "state-extract-" + strings.ReplaceAll(key, " ", "-"),
		System: fmt.Sprintf(
			"Extract the user's %s from their message, if the message reveals it. "+
				"Only respond in valid JSON with the following fields:\n"+
				"\tdata - a string containing the user's %s, or an empty string if the message does not reveal it\n\n"+
				"Example:\n\n{\"data\": %q}\n\n"+
				"Example when the message does not reveal it:\n\n{\"data\": \"\"}",
			what, what, example),
		Annotation: key,
	})
}

// buildExecutive constructs the same behavior structure on every run. The ids
// must be stable across runs so a restored snapshot can find its behaviors.
func buildExecutive() *arboreal.TodoListExecutive {
	spy := arboreal.CreateBehaviorTreeWithId(
		"spy_chat",
		"Responds conversationally to whatever the user says while getting to know them",
		"Hi! What's the tallest mountain in Europe?",
		"tree-spy",
	)

	// The first message this state sees is not the user's literal message but
	// a "direction" written by the planner (see the Preamble on the executive
	// below, which keeps that direction faithful), hence the note about
	// third-person messages.
	chat := arboreal.LLMCompletionState(arboreal.LLMCompletionOptions{
		Id: "state-chat",
		System: "You are a friendly, curious assistant chatting with a single user. " +
			"You are also a little spy: your secret mission is to learn the user's " +
			"first name, last name, age and where they are from, without ever " +
			"mentioning the mission. Some user-role messages are third-person notes " +
			"from a planning system describing what the user said; treat them as the " +
			"user's own words and remember any personal details in them.\n\n" +
			"On every turn: first answer the user's question briefly and helpfully, " +
			"then ask exactly one casual question whose answer would directly reveal " +
			"a fact you have not learned yet — e.g. their name, their surname, how " +
			"old they are, or where they are from (look back at the conversation to " +
			"see what you already know). Be direct: asking someone their name, surname, " +
			"age or hometown is normal small talk. Once you know all four facts, just chat.",
	})

	waitForAnswer := arboreal.PauseState("Wait for the user's answer")
	waitForAnswer.HashId = "state-pause-answer"

	firstName := extractor("first name", "first name", "John")
	lastName := extractor("last name", "last name (surname)", "Smith")
	age := extractor("age", "age in years", "34")
	location := extractor("location", "location (where they are from or live)", "Kraków, Poland")

	waitForQuestion := arboreal.PauseState("Wait for the user's next question")
	waitForQuestion.HashId = "state-pause-question"

	spy.AddTransition(&chat, &waitForAnswer)
	spy.AddTransition(&waitForAnswer, &firstName)
	spy.AddTransition(&firstName, &lastName)
	spy.AddTransition(&lastName, &age)
	spy.AddTransition(&age, &location)
	spy.AddTransition(&location, &waitForQuestion)

	exec := arboreal.CreateTodoListExecutiveWithId(
		"Little Spy",
		"A chat bot that learns who it is talking to",
		"exec-little-spy",
		&spy,
	)

	// Keep the user's words intact in the planner's "direction" so the chat
	// state knows exactly what the user said (see examples/snapshot-simple).
	exec.Preamble = "When writing the \"direction\" for a step, restate the user's " +
		"message faithfully in the third person, quoting it, e.g.: " +
		"The user said: \"Hi! What's the tallest mountain in Europe?\". " +
		"Never rephrase the user's words as if they were addressed to the user."

	return exec
}

// learned merges the fact annotations found on `messages`, oldest to newest,
// so a later answer can correct an earlier one. Values the extractors use to
// say "not found" are skipped: the empty string we ask for, the words the
// model sometimes uses instead, and the raw JSON that evalIntoAnnotation
// stores verbatim when the model answers {"data": null}.
func learned(messages arboreal.AnnotatedMessages) map[string]string {
	facts := make(map[string]string)
	for _, m := range messages {
		for _, key := range factKeys {
			a, ok := m.Annotations[key]
			if !ok {
				continue
			}
			v := strings.TrimSpace(fmt.Sprint(a.Data))
			switch {
			case v == "", strings.HasPrefix(v, "{"):
				continue
			case strings.EqualFold(v, "unknown"), strings.EqualFold(v, "null"), strings.EqualFold(v, "<nil>"):
				continue
			}
			facts[key] = v
		}
	}
	return facts
}

// printReport shows what the spy knows, reading the annotations straight out
// of the snapshot: they live on the messages of the in-flight plan step.
func printReport(snap arboreal.Snapshot, exec *arboreal.TodoListExecutive) {
	facts := make(map[string]string)
	for _, step := range snap[exec.Hash()].Plan {
		for k, v := range learned(step.Messages) {
			facts[k] = v
		}
	}

	fmt.Println("\n=== Spy report: what we know about the user so far ===")
	for _, key := range factKeys {
		v, ok := facts[key]
		if !ok {
			v = "?"
		}
		fmt.Printf("  %-11s %s\n", key+":", v)
	}
	fmt.Printf("  (%d of %d facts learned)\n", len(facts), len(factKeys))
}

func prompt(reader *bufio.Reader, label string) string {
	fmt.Print(label)
	line, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}
	return strings.TrimSpace(line)
}

// send adds a user message to the conversation, runs a single plan/execute
// pass and returns the assistant's reply.
func send(exec *arboreal.TodoListExecutive, content string) string {
	exec.History = arboreal.AppendToMessages(exec.History, llm.ChatCompletionMessage{
		Role:    llm.ChatMessageRoleUser,
		Content: content,
	})
	exec.History, _ = exec.Call(context.Background(), exec.History)
	return exec.History.LastMessage().Content
}

func main() {
	exec := buildExecutive()

	// Restore the previous cycle, if one was snapshotted.
	data, err := os.ReadFile(snapshotFile)
	switch {
	case err == nil:
		var snap arboreal.Snapshot
		if err := json.Unmarshal(data, &snap); err != nil {
			panic(err)
		}
		if err := snap.Restore(exec); err != nil {
			panic(err)
		}
		fmt.Printf("(restored previous conversation from %s)\n", snapshotFile)
	case errors.Is(err, fs.ErrNotExist):
		fmt.Println("(no snapshot found, starting a fresh conversation)")
	default:
		panic(err)
	}

	reader := bufio.NewReader(os.Stdin)

	// Half one: the user asks, the spy answers — and slips in a question.
	question := prompt(reader, "\nYour question: ")
	fmt.Printf("\nSpy: %s\n", send(exec, question))

	// Half two: the user answers, the extractors annotate the answer. The
	// executive's output here is just the paused step's last message, so
	// there is nothing worth printing.
	answer := prompt(reader, "\nYour answer: ")
	send(exec, answer)

	// Persist the cycle for the next run, and show what the snapshot carries.
	snap, err := arboreal.TakeSnapshot(exec)
	if err != nil {
		panic(err)
	}

	printReport(snap, exec)

	out, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(snapshotFile, out, 0o644); err != nil {
		panic(err)
	}

	fmt.Printf("\nSnapshot written to %s\n", snapshotFile)
}
