package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

// bedrockClient is the subset of the Bedrock runtime client the provider uses,
// so tests can substitute a fake.
type bedrockClient interface {
	Converse(context.Context, *bedrockruntime.ConverseInput, ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error)
	InvokeModel(context.Context, *bedrockruntime.InvokeModelInput, ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error)
}

// BedrockService talks to AWS Bedrock. Chat goes through the Converse API,
// which is model-agnostic; embeddings go through InvokeModel, whose body is
// model-specific (only Amazon Titan text embeddings for now).
//
// Credentials and region come from the standard AWS configuration chain
// (AWS_REGION, AWS_PROFILE, environment keys, SSO, instance roles...).
type BedrockService struct {
	client bedrockClient
}

func newBedrockService() (*BedrockService, error) {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, err
	}

	if cfg.Region == "" {
		return nil, errors.New("AWS region not set (set AWS_REGION or configure a profile region)")
	}

	return &BedrockService{client: bedrockruntime.NewFromConfig(cfg)}, nil
}

func (s *BedrockService) CreateChatCompletion(ctx context.Context, request *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	input, toolMap, err := bedrockConverseInput(request)
	if err != nil {
		return nil, err
	}

	output, err := s.client.Converse(ctx, input)
	if err != nil {
		return nil, err
	}

	return bedrockChatResponse(request.Model, output, toolMap)
}

// bedrockConverseInput maps a provider-neutral request onto a Converse call.
// It returns the lookup from the normalized tool names sent to Bedrock back to
// the names the caller knows them by.
func bedrockConverseInput(request *ChatCompletionRequest) (*bedrockruntime.ConverseInput, map[string]string, error) {
	if len(request.Messages) == 0 {
		return nil, nil, fmt.Errorf("No messages were passed in")
	}

	input := bedrockruntime.ConverseInput{
		ModelId: aws.String(ParseModelURI(request.Model).Name),
	}

	messages := request.Messages
	if messages[0].Role == ChatMessageRoleSystem {
		if messages[0].Content != "" {
			input.System = []types.SystemContentBlock{
				&types.SystemContentBlockMemberText{Value: messages[0].Content},
			}
		}
		messages = messages[1:]
	}

	// The executive only executes the first tool call of a response, so a
	// model asking for several in parallel gets a single result back. Converse
	// rejects a toolUse without a matching toolResult, so only tool calls that
	// were answered are replayed.
	answered := make(map[string]bool)
	for _, m := range messages {
		if m.Role == ChatMessageRoleFunction {
			answered[idFromMeta(&m)] = true
		}
	}

	for _, m := range messages {
		var role types.ConversationRole
		var content []types.ContentBlock

		switch m.Role {
		case ChatMessageRoleFunction:
			role = types.ConversationRoleUser
			content = append(content, &types.ContentBlockMemberToolResult{
				Value: types.ToolResultBlock{
					ToolUseId: aws.String(idFromMeta(&m)),
					Content: []types.ToolResultContentBlock{
						&types.ToolResultContentBlockMemberText{Value: m.Content},
					},
				},
			})
		case ChatMessageRoleAssistant:
			role = types.ConversationRoleAssistant
			if m.Content != "" {
				content = append(content, &types.ContentBlockMemberText{Value: m.Content})
			}
			for _, tool := range m.ToolCalls {
				id := idFromMeta(&tool)
				if !answered[id] {
					continue
				}

				arguments := tool.Arguments
				if arguments == nil {
					arguments = map[string]any{}
				}

				content = append(content, &types.ContentBlockMemberToolUse{
					Value: types.ToolUseBlock{
						ToolUseId: aws.String(id),
						Name:      aws.String(normalizeNamelike(tool.Name)),
						Input:     document.NewLazyDocument(arguments),
					},
				})
			}
		default:
			role = types.ConversationRoleUser
			if m.Content != "" {
				content = append(content, &types.ContentBlockMemberText{Value: m.Content})
			}
		}

		// Converse rejects blank text blocks and messages without content.
		if len(content) == 0 {
			continue
		}

		// Converse requires roles to alternate, so consecutive messages from
		// the same side (e.g. a tool result followed by a user message) are
		// merged into one turn.
		if n := len(input.Messages); n > 0 && input.Messages[n-1].Role == role {
			input.Messages[n-1].Content = append(input.Messages[n-1].Content, content...)
			continue
		}

		input.Messages = append(input.Messages, types.Message{
			Role:    role,
			Content: content,
		})
	}

	if len(input.Messages) == 0 {
		return nil, nil, fmt.Errorf("No non-empty messages were passed in")
	}

	var inference types.InferenceConfiguration
	if t, ok := request.Options["temperature"].(float32); ok {
		inference.Temperature = aws.Float32(t)
	}
	if t, ok := request.Options["max_tokens"].(int); ok {
		inference.MaxTokens = aws.Int32(int32(t))
	}
	if inference.Temperature != nil || inference.MaxTokens != nil {
		input.InferenceConfig = &inference
	}

	toolMap := make(map[string]string)
	if len(request.Tools) > 0 {
		var tools []types.Tool
		for _, t := range request.Tools {
			normalized := normalizeNamelike(t.Name)
			toolMap[normalized] = t.Name

			// The document encoder does not honor json tags, so the schema is
			// round-tripped through JSON into plain maps first.
			schema := map[string]any{"type": "object"}
			if t.InputSchema != nil {
				b, err := json.Marshal(t.InputSchema)
				if err != nil {
					return nil, nil, err
				}
				if err := json.Unmarshal(b, &schema); err != nil {
					return nil, nil, err
				}
				if schema["type"] == nil {
					schema["type"] = "object"
				}
			}

			tools = append(tools, &types.ToolMemberToolSpec{
				Value: types.ToolSpecification{
					Name:        aws.String(normalized),
					Description: aws.String(t.Description),
					InputSchema: &types.ToolInputSchemaMemberJson{Value: document.NewLazyDocument(schema)},
				},
			})
		}

		input.ToolConfig = &types.ToolConfiguration{Tools: tools}
	}

	return &input, toolMap, nil
}

func bedrockChatResponse(model string, output *bedrockruntime.ConverseOutput, toolMap map[string]string) (*ChatCompletionResponse, error) {
	message, ok := output.Output.(*types.ConverseOutputMemberMessage)
	if !ok {
		return nil, fmt.Errorf("unexpected Bedrock output type %T", output.Output)
	}

	response := ChatCompletionResponse{
		Model:     model,
		CreatedAt: time.Now(),
		Message: ChatCompletionMessage{
			Role: ChatMessageRoleAssistant,
		},
	}

	var text []string
	for _, block := range message.Value.Content {
		switch b := block.(type) {
		case *types.ContentBlockMemberText:
			text = append(text, b.Value)
		case *types.ContentBlockMemberToolUse:
			// Decoded through plain JSON so numbers come back as float64, as
			// they do from the other providers, rather than smithy numbers.
			var arguments map[string]any
			if b.Value.Input != nil {
				raw, err := b.Value.Input.MarshalSmithyDocument()
				if err != nil {
					return nil, err
				}
				if err := json.Unmarshal(raw, &arguments); err != nil {
					return nil, err
				}
			}

			name := aws.ToString(b.Value.Name)
			if original, ok := toolMap[name]; ok {
				name = original
			}

			t := ToolCall{
				Name:      name,
				Arguments: arguments,
			}
			t.SetMeta("id", aws.ToString(b.Value.ToolUseId))

			response.Message.ToolCalls = append(response.Message.ToolCalls, t)
		}
	}
	response.Message.Content = strings.Join(text, "\n")

	return &response, nil
}

type bedrockTitanEmbeddingRequest struct {
	InputText string `json:"inputText"`
}

type bedrockTitanEmbeddingResponse struct {
	Embedding []float32 `json:"embedding"`
}

func (s *BedrockService) CreateEmbedding(ctx context.Context, request *EmbeddingRequest) (Embedding, error) {
	model := ParseModelURI(request.Model).Name
	if model == "" {
		model = ModelBedrockTitanEmbedV2
	}

	if !strings.Contains(model, "amazon.titan-embed-text") {
		return nil, fmt.Errorf("unsupported Bedrock embedding model: %s (only Amazon Titan text embeddings are supported)", model)
	}

	body, err := json.Marshal(bedrockTitanEmbeddingRequest{InputText: request.Input})
	if err != nil {
		return nil, err
	}

	output, err := s.client.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(model),
		ContentType: aws.String("application/json"),
		Accept:      aws.String("application/json"),
		Body:        body,
	})
	if err != nil {
		return nil, err
	}

	var response bedrockTitanEmbeddingResponse
	if err := json.Unmarshal(output.Body, &response); err != nil {
		return nil, err
	}

	return response.Embedding, nil
}
