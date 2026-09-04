// Package main is a learning-purposes example, NOT a template for real apps.
//
// The one-turn example again — two Calls, no RunLoop — but this time with a
// Trace attached to the context. A Trace is just chan *TraceMessage; every
// state, tree and executive sends begin/end messages (the only types this
// program triggers) into it if it finds one under the context key
// "arboreal_trace". The channel is unbuffered, so someone MUST drain it or
// the agent deadlocks mid-turn; that someone is the goroutine below.
//
// Requires OPENAI_TOKEN to be set in the environment.
package main

import (
	"context"
	"fmt"

	"github.com/zenful-ai/arboreal"
	"github.com/zenful-ai/arboreal/llm"
)

func buildExecutive() *arboreal.TodoListExecutive {
	chatBehavior := arboreal.CreateBehaviorTree(
		"chat_behavior",
		"A conversational bot",
		"<insert user's input here>",
	)

	chatState := arboreal.LLMCompletionState(arboreal.LLMCompletionOptions{
		Name: "chat", // Name reaches the trace; without it lines are anonymous
	})
	pauseState := arboreal.PauseState("Let user respond")
	chatBehavior.AddTransition(&chatState, &pauseState)

	return arboreal.CreateTodoListExecutive(
		"Chat Bot",
		"Simple chat bot",
		&chatBehavior,
	)
}

// ANCHOR: drain
func startDrain(trace arboreal.Trace) chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		for msg := range trace {
			line := fmt.Sprintf("%-10s %-14s %s", msg.Type, msg.Name, msg.Message)
			if msg.Signal != nil {
				line += fmt.Sprintf("  [signal: %s %q]", msg.Signal.Type, msg.Signal.Reason)
			}
			// Operations is declared for history changes but is never
			// populated today — see the tracing chapter's sharp edge.
			for _, op := range msg.Operations {
				line += fmt.Sprintf("\n           %s %s", op.Action, op.Type)
			}
			fmt.Println(line)
		}
	}()
	return done
}

// ANCHOR_END: drain

// ANCHOR: attach
func main() {
	exec := buildExecutive()

	trace := make(arboreal.Trace)
	done := startDrain(trace)

	// The key is a raw string — there is no exported helper for this.
	ctx := context.WithValue(context.Background(), "arboreal_trace", trace)

	var transcript arboreal.AnnotatedMessages
	transcript = turn(ctx, exec, transcript, "Hi, I'm Paul. Please remember my name.")
	transcript = turn(ctx, exec, transcript, "What is my name?")

	close(trace)
	<-done

	fmt.Println("---")
	for i, m := range transcript {
		fmt.Printf("[%d] %-9s %s\n", i, m.Role, m.Content)
	}
}

// ANCHOR_END: attach

func turn(ctx context.Context, exec *arboreal.TodoListExecutive,
	transcript arboreal.AnnotatedMessages, content string) arboreal.AnnotatedMessages {

	transcript = arboreal.AppendToMessages(transcript, llm.ChatCompletionMessage{
		Role:    llm.ChatMessageRoleUser,
		Content: content,
	})

	transcript, _ = exec.Call(ctx, transcript)
	return transcript
}
