package main

import (
	"context"
	"log"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zenful-ai/arboreal"
	"github.com/zenful-ai/arboreal/llm"
)

type Empty struct{}

func getCurrentTimestamp(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[any]) (*mcp.CallToolResultFor[string], error) {
	return &mcp.CallToolResultFor[string]{
		Content: []mcp.Content{&mcp.TextContent{Text: time.Now().Format(time.RFC3339)}},
	}, nil
}

func main() {
	server := mcp.NewServer(&mcp.Implementation{Name: "ai.zenful", Version: "v1.0.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "ai.zenful/get_current_timestamp", Description: "Get the current timestamp"}, getCurrentTimestamp)

	x, y := mcp.NewInMemoryTransports()

	go func() {
		if err := server.Run(context.Background(), x); err != nil {
			log.Fatal(err)
		}
	}()

	mcpClient := arboreal.NewMCPClientMux()
	mcpClient.AddInMemoryServer(context.Background(), y)

	ctx := context.WithValue(context.Background(), "arboreal_mcp_client", mcpClient)

	state := arboreal.LLMCompletionState(arboreal.LLMCompletionOptions{
		Model:      llm.ClaudeHaiku,
		AllowTools: true,
	})

	var history arboreal.AnnotatedMessages
	var t arboreal.TerminalChannel

	for {
		var sig arboreal.Signal

		m, err := t.Receive()
		if err != nil {
			log.Fatal(err)
		}

		history = arboreal.AppendToMessages(history, llm.ChatCompletionMessage{
			Role:    llm.ChatMessageRoleUser,
			Content: m.Content,
		})

		history, sig = state.Call(ctx, history)

		if _, ok := sig.(*arboreal.ErrorSignal); ok {
			log.Fatal(sig.Description())
		}

		err = t.Send(&arboreal.ChannelMessage{
			Content: history.LastMessage().Content,
		})
		if err != nil {
			log.Fatal(err)
		}
	}
}
