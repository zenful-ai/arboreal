// Package main is a learning-purposes example, NOT a template for real apps.
//
// It demonstrates persisting a conversation across separate process runs using
// TakeSnapshot and Snapshot.Restore. Each run performs a single interaction:
//
//  1. Ask the user for their next message on standard input.
//  2. If a snapshot file from a previous run exists, restore the executive's
//     state (history + in-flight plan) from it; otherwise start fresh, letting
//     the executive plan a todo list from scratch for the user's message.
//  3. Execute the plan once, print the assistant's response.
//  4. Take a snapshot and write it to disk.
//
// There is deliberately no loop: to continue the conversation, run the program
// again — the snapshot is what carries the state between runs. For example:
//
//	$ go run ./examples/snapshot-simple   # say: Hey, my name is John and I am a pirate
//	$ go run ./examples/snapshot-simple   # ask: What is my name and profession?
//
// The second run should answer "John" and "a pirate", proving the state
// survived the process exit.
//
// Two details make restoring possible:
//
//   - Every behavior is built with a *stable* id (the ...WithId constructors
//     and explicit state ids). Snapshots reference behaviors by hash, so the
//     hashes must be identical on every run.
//   - The behavior tree ends in a PauseState. Its CollectUserInputSignal keeps
//     the plan step alive in the executive, and TakeSnapshot only persists an
//     executive that still has a plan in flight.
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

const snapshotFile = "snapshot.json"

// buildExecutive constructs the same behavior structure on every run. The ids
// must be stable across runs so a restored snapshot can find its behaviors.
func buildExecutive() *arboreal.TodoListExecutive {
	chatBehavior := arboreal.CreateBehaviorTreeWithId(
		"chat",
		"Responds conversationally to whatever the user says, remembering the conversation so far",
		"Hey, my name is John and I am a pirate",
		"tree-chat",
	)

	// The system prompt matters here: the first message this state sees is not
	// the user's literal message but a "direction" written by the planner (see
	// the Preamble on the executive below, which keeps that direction
	// faithful). Tell the model how to read it, so facts about the user
	// survive as memory in later runs.
	chatState := arboreal.LLMCompletionState(arboreal.LLMCompletionOptions{
		Id: "state-chat",
		System: "You are a friendly assistant chatting with a single user. " +
			"Some user-role messages are third-person notes from a planning system " +
			"describing what the user said; treat any personal details in them (such as " +
			"the user's name or profession) as facts about the user, remember those " +
			"facts, and answer follow-up questions about them directly.",
	})
	pauseState := arboreal.PauseState("Wait for the user's next message")
	pauseState.HashId = "state-pause"

	chatBehavior.AddTransition(&chatState, &pauseState)

	exec := arboreal.CreateTodoListExecutiveWithId(
		"Snapshot Chat",
		"A chat bot whose conversation state is persisted between runs",
		"exec-snapshot-chat",
		&chatBehavior,
	)

	// By default the planner paraphrases the user's message into the step
	// "direction", which can garble who said what (e.g. turning "my name is
	// John" into a greeting addressed to John). Keep the user's words intact
	// so the chat state — and every later run restored from a snapshot — knows
	// exactly what the user originally said.
	exec.Preamble = "When writing the \"direction\" for a step, restate the user's " +
		"message faithfully in the third person, quoting it, e.g.: " +
		"The user said: \"Hey, my name is John and I am a pirate\". " +
		"Never rephrase the user's words as if they were addressed to the user."

	return exec
}

func main() {
	fmt.Print("Your message: ")
	question, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		panic(err)
	}
	question = strings.TrimSpace(question)

	exec := buildExecutive()

	// Restore the previous conversation, if one was snapshotted.
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

	// One interaction: add the user's message to the conversation and let the
	// executive do a single plan/execute pass. On a fresh start Call plans a
	// new todo list; on a restored run it feeds the message to the plan
	// already in flight.
	exec.History = arboreal.AppendToMessages(exec.History, llm.ChatCompletionMessage{
		Role:    llm.ChatMessageRoleUser,
		Content: question,
	})

	exec.History, _ = exec.Call(context.Background(), exec.History)

	fmt.Printf("\nAssistant: %s\n", exec.History.LastMessage().Content)

	// Persist the conversation for the next run.
	snap, err := arboreal.TakeSnapshot(exec)
	if err != nil {
		panic(err)
	}

	out, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		panic(err)
	}

	if err := os.WriteFile(snapshotFile, out, 0o644); err != nil {
		panic(err)
	}

	fmt.Printf("\nSnapshot written to %s\n", snapshotFile)
}
