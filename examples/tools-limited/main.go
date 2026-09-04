// Package main is a learning-purposes example, NOT a template for real apps.
//
// One MCP server exposing three tools, one LLMCompletionState that offers
// the model exactly one of them. The mux is connected to the whole server —
// its Tools() reports all three — but the state names the one it needs in
// LLMCompletionOptions.Tools, and that list is the only thing the model is
// sent and the only thing the tool loop will execute. The dump shows which
// tool ran, and the model saying it cannot do the rest.
//
// Requires OPENAI_TOKEN to be set in the environment (the default model is
// gpt-4o-mini via the OpenAI provider).
package main

import (
	"context"
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zenful-ai/arboreal"
	"github.com/zenful-ai/arboreal/llm"
)

func getCurrentTimestamp(ctx context.Context, req *mcp.CallToolRequest,
	_ any) (*mcp.CallToolResult, any, error) {
	return text(time.Now().Format(time.RFC3339))
}

func getWeather(ctx context.Context, req *mcp.CallToolRequest,
	_ any) (*mcp.CallToolResult, any, error) {
	return text("sunny, 21°C")
}

func deleteEverything(ctx context.Context, req *mcp.CallToolRequest,
	_ any) (*mcp.CallToolResult, any, error) {
	return text("everything deleted")
}

func text(s string) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: s}},
	}, nil, nil
}


func main() {
	server := mcp.NewServer(&mcp.Implementation{Name: "ai.zenful", Version: "v1.0.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ai.zenful/get_current_timestamp",
		Description: "Get the current timestamp",
	}, getCurrentTimestamp)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ai.zenful/get_weather",
		Description: "Get the current weather",
	}, getWeather)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ai.zenful/delete_everything",
		Description: "Delete all of the user's data",
	}, deleteEverything)

	serverSide, clientSide := mcp.NewInMemoryTransports()
	go func() {
		if err := server.Run(context.Background(), serverSide); err != nil {
			log.Fatal(err)
		}
	}()

	mux := arboreal.NewMCPClientMux()
	if err := mux.AddInMemoryServer(context.Background(), clientSide); err != nil {
		log.Fatal(err)
	}
	defer mux.Close()
	ctx := arboreal.WithMCPClient(context.Background(), mux)

	// The mux knows the whole server. offered is the list the state below
	// names; Select on it here prints the list the model will be sent.
	var all []string
	for _, t := range mux.Tools() {
		all = append(all, t.Name)
	}
	slices.Sort(all) // Tools() is in map order
	fmt.Printf("server exposes %d tools:\n", len(all))
	for _, name := range all {
		fmt.Printf("  %s\n", name)
	}

	offered := []string{"ai.zenful/get_current_timestamp"}
	selected, err := mux.Select(offered...)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("state offers %d:\n", len(selected))
	for _, t := range selected {
		fmt.Printf("  %s\n", t.Name)
	}

	// ANCHOR: call
	state := arboreal.LLMCompletionState(arboreal.LLMCompletionOptions{
		System: "Answer using only the tools available to you. If part of the " +
			"question needs a tool you do not have, say so instead of guessing.",
		AllowTools: true,
		Tools:      offered,
	})

	history := arboreal.AppendToMessages(nil, llm.ChatCompletionMessage{
		Role:    llm.ChatMessageRoleUser,
		Content: "What is the exact current timestamp, and what is the weather like?",
	})

	history, sig := state.Call(ctx, history)
	if err, ok := sig.(*arboreal.ErrorSignal); ok {
		log.Fatal(err.Description())
	}

	fmt.Println()
	for i, m := range history {
		// Name is set only on the function-role result: the tool that ran.
		fmt.Printf("[%d] %-9s %-32s %q\n", i, m.Role, m.Name, m.Content)
	}
}
