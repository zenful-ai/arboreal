// A scratch probe (not a real example): understand how annotation extraction
// works by calling a BehaviorTree directly — no TodoListExecutive, no plan.
//
// The tree has two LLMCompletionStates, both with the Annotation option set.
// Such a state never appends a chat reply; instead it sends only its system
// prompt + the last user message to the model and pins the parsed result onto
// that user message as a named annotation.
//
// We print the history before and after the call to see exactly what changed.
package main

import (
	"context"
	"fmt"

	"github.com/zenful-ai/arboreal"
	"github.com/zenful-ai/arboreal/llm"
)

func main() {
	// The third argument, Example, is a sample direction for the executive's
	// planner. No executive runs this tree, so it is unused here; the user's
	// request is the message handed to tree.Call below.
	tree := arboreal.CreateBehaviorTree(
		"extract_facts",
		"Extracts facts about the user from their message",
		"",
	)

	extractName := arboreal.LLMCompletionState(arboreal.LLMCompletionOptions{
		System: "Extract the person's name from the user's message. " +
			"Only respond in valid JSON with the following fields:\n" +
			"\tdata - a string containing the person's name\n\n" +
			"Example:\n\n{\"data\": \"John\"}",
		Annotation: "name",
	})

	extractProfession := arboreal.LLMCompletionState(arboreal.LLMCompletionOptions{
		System: "Extract the person's profession from the user's message. " +
			"Only respond in valid JSON with the following fields:\n" +
			"\tdata - a string containing the person's profession\n\n" +
			"Example:\n\n{\"data\": \"carpenter\"}",
		Annotation: "profession",
	})

	tree.AddTransition(&extractName, &extractProfession)

	history := arboreal.AppendToMessages(nil, llm.ChatCompletionMessage{
		Role:    llm.ChatMessageRoleUser,
		Content: "Hi, I'm Joe and I work as a pirate on the Black Pearl.",
	})

	fmt.Println("=== history BEFORE tree.Call ===")
	dump(history)

	history, sig := tree.Call(context.Background(), history)
	fmt.Printf("\n(tree.Call returned signal: %v)\n\n", sig)

	fmt.Println("=== history AFTER tree.Call ===")
	dump(history)
}

func dump(history arboreal.AnnotatedMessages) {
	for i, m := range history {
		fmt.Printf("[%d] role=%-9s content=%q\n", i, m.Role, m.Content)
		for name, a := range m.Annotations {
			fmt.Printf("      annotation %-20q data=%#v\n", name, a.Data)
		}
	}
}
