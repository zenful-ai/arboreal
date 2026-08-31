// Package main is a learning-purposes example, NOT a template for real apps.
//
// It calls a single LLMCompletionState directly — no behavior tree, no
// executive — to show what one state does to the conversation:
//
//  1. The System option is a template. {{ $date_llm }} is a built-in
//     meta-annotation that renders as "Today's date is: ...", so the model
//     knows the date without us hard-coding it.
//  2. Because System is set and the history has no system message yet, the
//     state prepends one. You can see the rendered prompt in the dump.
//  3. The model's reply is appended as an assistant message.
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
	// ANCHOR: state
	state := arboreal.LLMCompletionState(arboreal.LLMCompletionOptions{
		Name:        "date_aware_assistant",
		Description: "Answers briefly and knows today's date",
		System: "You are a terse assistant. {{ $date_llm }} " +
			"Answer in one sentence.",
	})
	// ANCHOR_END: state

	// ANCHOR: history
	history := arboreal.AppendToMessages(nil, llm.ChatCompletionMessage{
		Role:    llm.ChatMessageRoleUser,
		Content: "What day of the week is it today?",
	})
	// ANCHOR_END: history

	// ANCHOR: call
	history, sig := state.Call(context.Background(), history)
	if err, ok := sig.(*arboreal.ErrorSignal); ok {
		log.Fatal(err)
	}
	// ANCHOR_END: call

	// ANCHOR: dump
	for i, m := range history {
		fmt.Printf("[%d] %-9s %s\n", i, m.Role, m.Content)
	}
	// ANCHOR_END: dump
}
