// Package main is a learning-purposes example, NOT a template for real apps.
//
// It calls a single LLMCompletionState against a model served by AWS Bedrock.
// The only difference from an OpenAI-backed state is the Model option: a
// "bedrock:" prefix routes the request to the Bedrock provider, and the rest of
// the URI is the Bedrock model ID (or inference profile ID) as AWS spells it.
//
// Requires AWS credentials and a region from the standard AWS configuration
// chain (e.g. AWS_PROFILE and AWS_REGION), with access to the model enabled in
// the Bedrock console. Set BEDROCK_MODEL to try another model from the catalog,
// e.g. BEDROCK_MODEL=us.anthropic.claude-haiku-4-5-20251001-v1:0.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/zenful-ai/arboreal"
	"github.com/zenful-ai/arboreal/llm"
)

func main() {
	model := llm.BedrockNovaLite
	if m := os.Getenv("BEDROCK_MODEL"); m != "" {
		model = llm.ProviderBedrock + ":" + m
	}

	state := arboreal.LLMCompletionState(arboreal.LLMCompletionOptions{
		Name:        "bedrock_assistant",
		Description: "Answers briefly through a Bedrock-hosted model",
		Model:       model,
		System:      "You are a terse assistant. Answer in one sentence.",
	})

	history := arboreal.AppendToMessages(nil, llm.ChatCompletionMessage{
		Role:    llm.ChatMessageRoleUser,
		Content: "What is a behavior tree?",
	})

	history, sig := state.Call(context.Background(), history)
	if err, ok := sig.(*arboreal.ErrorSignal); ok {
		log.Fatal(err)
	}

	fmt.Printf("model: %s\n\n", model)
	for i, m := range history {
		fmt.Printf("[%d] %-9s %s\n", i, m.Role, m.Content)
	}
}
