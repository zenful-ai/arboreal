package main

import (
	"context"

	"github.com/zenful-ai/arboreal"
)

func main() {
	// Create a behavior tree for a chat bot
	chatBehavior := arboreal.CreateBehaviorTree(
		"chat_behavior",
		"A conversational bot",
		"<insert user's input here>",
	)

	// Define states
	chatState := arboreal.LLMCompletionState(arboreal.LLMCompletionOptions{})
	pauseState := arboreal.PauseState("Let user respond")

	// Add transitions
	chatBehavior.AddTransition(&chatState, &pauseState)

	// Create executive
	exec := arboreal.CreateTodoListExecutive(
		"Chat Bot",
		"Simple chat bot",
		&chatBehavior,
	)

	// Run the bot
	err := exec.RunLoop(context.Background(), arboreal.TerminalChannel{})
	if err != nil {
		panic(err)
	}
}
