// Package main is a learning-purposes example, NOT a template for real apps.
//
// It drives a BehaviorTree directly, with a hand-written loop, and no
// TodoListExecutive at all. The loop is the part of RunLoop that a tree
// needs from its caller: call the tree, show what it said, and when it
// pauses for input, collect a message and call it again.
//
// The tree is:
//
//	greet → ask → answer
//
// greet is a canned assistant message, ask is a PauseState, answer is an LLM
// state. One full pass therefore takes two calls: the first runs greet and
// stops at ask (CollectUserInputSignal); the second, made after the user
// typed a question, resumes at answer. After answer the tree's stack is
// empty, so the third call starts over at greet — you will see the greeting
// again every other turn. That is the rule: a paused tree resumes, a drained
// tree restarts. One consequence to watch for: a question typed right after
// an answer is met with the greeting, not an answer — that turn restarts the
// tree, and only the following turn reaches answer again. It is the tree's
// design, not a bug.
//
// Type a message and submit it with a line containing only "$" (or Ctrl-]).
// Submitting an empty message ends the session.
//
// Requires OPENAI_TOKEN to be set in the environment (the default model is
// gpt-4o-mini via the OpenAI provider).
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/zenful-ai/arboreal"
	"github.com/zenful-ai/arboreal/llm"
)

func main() {
	// ANCHOR: tree
	tree := arboreal.CreateBehaviorTree("qa", "Greets the user, waits for a question, answers it", "")

	greet := arboreal.CannedResponseState("Hello! Ask me one question.")
	ask := arboreal.PauseState("Wait for the user's question")
	answer := arboreal.LLMCompletionState(arboreal.LLMCompletionOptions{
		System: "Answer the user's latest question in at most two sentences.",
	})

	// CannedResponseState returns a *BehaviorState; the other two factories
	// return values, hence the &. All three satisfy Behavior via pointer.
	tree.AddTransition(greet, &ask)
	tree.AddTransition(&ask, &answer)
	// ANCHOR_END: tree

	// ANCHOR: loop
	var channel arboreal.TerminalChannel
	var history arboreal.AnnotatedMessages
	ctx := context.Background()

	for {
		var sig arboreal.Signal
		history, sig = tree.Call(ctx, history)

		// In this tree every call appends at most one assistant message
		// (greet, or the model's reply), so showing the last one is enough. A
		// tree with several replying states per call would need to show
		// everything appended since the previous call.
		if last := history.LastMessage(); last != nil && last.Role == llm.ChatMessageRoleAssistant {
			if err := channel.Send(&arboreal.ChannelMessage{Content: last.Content}); err != nil {
				log.Fatal(err)
			}
		}

		switch s := sig.(type) {
		case *arboreal.ErrorSignal:
			log.Fatal(s)
		case *arboreal.CollectUserInputSignal:
			fmt.Printf("(tree paused: %s)\n\n", s.Reason)
		case nil:
			// The tree ran to the end and its stack is empty, so the next
			// Call will restart from greet.
			fmt.Printf("(tree finished; the next call restarts it)\n\n")
		default:
			log.Fatalf("unexpected signal %T", s)
		}

		msg, err := channel.Receive()
		if err != nil {
			log.Fatal(err)
		}
		if msg.Content == "" {
			// TerminalChannel never returns an error, not even at EOF; an
			// empty message is the only way to notice the user is gone.
			fmt.Println("(bye)")
			return
		}
		history = arboreal.AppendToMessages(history, llm.ChatCompletionMessage{
			Role:    llm.ChatMessageRoleUser,
			Content: msg.Content,
		})
	}
	// ANCHOR_END: loop
}
