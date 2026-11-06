package llm

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/google/jsonschema-go/jsonschema"
)

type fakeBedrockClient struct {
	converseInput *bedrockruntime.ConverseInput
	converse      *bedrockruntime.ConverseOutput
	invokeInput   *bedrockruntime.InvokeModelInput
	invoke        *bedrockruntime.InvokeModelOutput
}

func (f *fakeBedrockClient) Converse(_ context.Context, in *bedrockruntime.ConverseInput, _ ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
	f.converseInput = in
	return f.converse, nil
}

func (f *fakeBedrockClient) InvokeModel(_ context.Context, in *bedrockruntime.InvokeModelInput, _ ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error) {
	f.invokeInput = in
	return f.invoke, nil
}

func TestParseModelURIBedrock(t *testing.T) {
	m := ParseModelURI(BedrockNovaLite)

	if m.Type != ProviderBedrock {
		t.Errorf("type = %q, want %q", m.Type, ProviderBedrock)
	}

	// Bedrock model IDs carry a version after a colon, which must survive.
	if m.Name != ModelBedrockNovaLite {
		t.Errorf("name = %q, want %q", m.Name, ModelBedrockNovaLite)
	}
}

func TestBedrockConverseInput(t *testing.T) {
	answered := ToolCall{Name: "catalog/lookup", Arguments: map[string]any{"title": "Dune"}}
	answered.SetMeta("id", "tool-1")
	unanswered := ToolCall{Name: "catalog/lookup", Arguments: map[string]any{"title": "Emma"}}
	unanswered.SetMeta("id", "tool-2")

	result := ChatCompletionMessage{Role: ChatMessageRoleFunction, Name: "catalog/lookup", Content: "in stock"}
	result.SetMeta("id", "tool-1")

	request := ChatCompletionRequest{
		Model: BedrockNovaLite,
		Messages: []ChatCompletionMessage{
			{Role: ChatMessageRoleSystem, Content: "Be terse."},
			{Role: ChatMessageRoleUser, Content: "Is Dune in stock?"},
			{Role: ChatMessageRoleAssistant, ToolCalls: []ToolCall{answered, unanswered}},
			result,
			{Role: ChatMessageRoleUser, Content: ""},
		},
		Tools: []ChatTool{{
			Type:        ChatToolTypeFunction,
			Name:        "catalog/lookup",
			Description: "Look a book up",
			InputSchema: &jsonschema.Schema{
				Properties: map[string]*jsonschema.Schema{"title": {Type: "string"}},
			},
		}},
		Options: map[string]any{"temperature": float32(0.2), "max_tokens": 512},
	}

	input, toolMap, err := bedrockConverseInput(&request)
	if err != nil {
		t.Fatal(err)
	}

	if got := aws.ToString(input.ModelId); got != ModelBedrockNovaLite {
		t.Errorf("model = %q, want %q", got, ModelBedrockNovaLite)
	}

	if len(input.System) != 1 || input.System[0].(*types.SystemContentBlockMemberText).Value != "Be terse." {
		t.Errorf("system = %#v", input.System)
	}

	// user, assistant (one answered toolUse, no blank text), user (toolResult;
	// the trailing blank user message is dropped).
	if len(input.Messages) != 3 {
		t.Fatalf("got %d messages, want 3", len(input.Messages))
	}

	assistant := input.Messages[1]
	if assistant.Role != types.ConversationRoleAssistant || len(assistant.Content) != 1 {
		t.Fatalf("assistant turn = %#v", assistant)
	}
	toolUse := assistant.Content[0].(*types.ContentBlockMemberToolUse).Value
	if aws.ToString(toolUse.ToolUseId) != "tool-1" || aws.ToString(toolUse.Name) != normalizeNamelike("catalog/lookup") {
		t.Errorf("toolUse = %#v", toolUse)
	}

	toolResult := input.Messages[2].Content[0].(*types.ContentBlockMemberToolResult).Value
	if input.Messages[2].Role != types.ConversationRoleUser || aws.ToString(toolResult.ToolUseId) != "tool-1" {
		t.Errorf("tool result turn = %#v", input.Messages[2])
	}

	if aws.ToFloat32(input.InferenceConfig.Temperature) != 0.2 || aws.ToInt32(input.InferenceConfig.MaxTokens) != 512 {
		t.Errorf("inference config = %#v", input.InferenceConfig)
	}

	spec := input.ToolConfig.Tools[0].(*types.ToolMemberToolSpec).Value
	b, err := spec.InputSchema.(*types.ToolInputSchemaMemberJson).Value.MarshalSmithyDocument()
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(b, &schema); err != nil {
		t.Fatal(err)
	}
	if schema["type"] != "object" || schema["properties"] == nil {
		t.Errorf("schema = %s", b)
	}

	if toolMap[normalizeNamelike("catalog/lookup")] != "catalog/lookup" {
		t.Errorf("toolMap = %v", toolMap)
	}
}

func TestBedrockConverseInputMergesConsecutiveRoles(t *testing.T) {
	input, _, err := bedrockConverseInput(&ChatCompletionRequest{
		Model: BedrockNovaLite,
		Messages: []ChatCompletionMessage{
			{Role: ChatMessageRoleUser, Content: "Hi"},
			{Role: ChatMessageRoleUser, Content: "Anyone there?"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(input.Messages) != 1 || len(input.Messages[0].Content) != 2 {
		t.Errorf("messages = %#v", input.Messages)
	}

	if input.InferenceConfig != nil || input.ToolConfig != nil {
		t.Errorf("unexpected config: %#v %#v", input.InferenceConfig, input.ToolConfig)
	}
}

func TestBedrockCreateChatCompletion(t *testing.T) {
	client := &fakeBedrockClient{
		converse: &bedrockruntime.ConverseOutput{
			Output: &types.ConverseOutputMemberMessage{Value: types.Message{
				Role: types.ConversationRoleAssistant,
				Content: []types.ContentBlock{
					&types.ContentBlockMemberText{Value: "Let me check."},
					&types.ContentBlockMemberToolUse{Value: types.ToolUseBlock{
						ToolUseId: aws.String("tool-9"),
						Name:      aws.String(normalizeNamelike("catalog/lookup")),
						Input:     document.NewLazyDocument(map[string]any{"title": "Dune"}),
					}},
				},
			}},
		},
	}

	service := BedrockService{client: client}
	res, err := service.CreateChatCompletion(context.Background(), &ChatCompletionRequest{
		Model:    BedrockNovaLite,
		Messages: []ChatCompletionMessage{{Role: ChatMessageRoleUser, Content: "Is Dune in stock?"}},
		Tools:    []ChatTool{{Type: ChatToolTypeFunction, Name: "catalog/lookup"}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if res.Message.Role != ChatMessageRoleAssistant || res.Message.Content != "Let me check." {
		t.Errorf("message = %#v", res.Message)
	}

	if len(res.Message.ToolCalls) != 1 {
		t.Fatalf("got %d tool calls, want 1", len(res.Message.ToolCalls))
	}
	call := res.Message.ToolCalls[0]
	if call.Name != "catalog/lookup" || call.Arguments["title"] != "Dune" || idFromMeta(&call) != "tool-9" {
		t.Errorf("tool call = %#v (id %q)", call, idFromMeta(&call))
	}
}

func TestBedrockCreateEmbedding(t *testing.T) {
	client := &fakeBedrockClient{
		invoke: &bedrockruntime.InvokeModelOutput{Body: []byte(`{"embedding":[0.5,-1]}`)},
	}
	service := BedrockService{client: client}

	embedding, err := service.CreateEmbedding(context.Background(), &EmbeddingRequest{Input: "hello"})
	if err != nil {
		t.Fatal(err)
	}

	if len(embedding) != 2 || embedding[0] != 0.5 || embedding[1] != -1 {
		t.Errorf("embedding = %v", embedding)
	}

	if aws.ToString(client.invokeInput.ModelId) != ModelBedrockTitanEmbedV2 || string(client.invokeInput.Body) != `{"inputText":"hello"}` {
		t.Errorf("invoke input = %#v", client.invokeInput)
	}

	if _, err := service.CreateEmbedding(context.Background(), &EmbeddingRequest{Model: "bedrock:cohere.embed-english-v3"}); err == nil {
		t.Error("expected an error for a non-Titan embedding model")
	}
}
