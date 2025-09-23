package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/jsonschema"
)

const anthropicVersion = "2023-06-01"

type anthropicAPIError struct {
	Type       string `json:"type"`
	InnerError struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

func (e anthropicAPIError) Error() string {
	return fmt.Sprintf("%s: %s", e.InnerError.Type, e.InnerError.Message)
}

type anthropicUsageObject struct {
	InputTokens   int `json:"input_tokens"`
	OutputTokens  int `json:"output_tokens"`
	ServerToolUse *struct {
		WebSearchResults int `json:"web_search_results"`
	} `json:"server_tool_use"`
	ServiceTier *string `json:"service_tier"`
}

type anthropicMessageResponse struct {
	Id           string                    `json:"id"`
	Model        string                    `json:"model"`
	Type         string                    `json:"type"`
	Role         string                    `json:"role"`
	Content      []anthropicMessageContent `json:"content"`
	StopReason   *string                   `json:"stop_reason"`
	StopSequence *string                   `json:"stop_sequence"`
	Usage        anthropicUsageObject      `json:"usage"`
}

type anthropicMessageRequest struct {
	Model     string                    `json:"model"`
	System    []anthropicMessageContent `json:"system,omitempty"`
	Messages  []anthropicMessage        `json:"messages"`
	Tools     []anthropicTool           `json:"tools,omitempty"`
	MaxTokens int                       `json:"max_tokens"`
	Stream    bool                      `json:"stream"`
}

type anthropicMessage struct {
	Role    string                    `json:"role"`
	Content []anthropicMessageContent `json:"content"`
}

type anthropicMessageContent struct {
	Type string `json:"type"`
	// Text
	Text string `json:"text,omitempty"`
	// Multimedia (images)
	MediaType string `json:"media_type,omitempty"`
	Data      []byte `json:"data,omitempty"`
	// Tool Use
	Id    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input *map[string]any `json:"input,omitempty"`
	// Tool Result
	ToolUseId string `json:"tool_use_id,omitempty"`
	Content   string `json:"content,omitempty"`
}

type anthropicTool struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	InputSchema jsonschema.Schema `json:"input_schema"`
}

type AnthropicService struct {
	apiKey string
}

func newAnthropicService() (*AnthropicService, error) {
	var service AnthropicService

	if service.apiKey = os.Getenv("ANTHROPIC_TOKEN"); service.apiKey == "" {
		return nil, errors.New("ANTHROPIC_TOKEN environment variable not set")
	}

	return &service, nil
}

func idFromMeta(m Meta) string {
	s, ok := m.GetMeta("id").(string)
	if !ok {
		return ""
	}
	return s
}

func (s *AnthropicService) messageRequest(ctx context.Context, request *anthropicMessageRequest) (*anthropicMessageResponse, error) {
	var response anthropicMessageResponse

	b, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	buff := bytes.NewBuffer(b)

	req, err := http.NewRequest(http.MethodPost, "https://api.anthropic.com/v1/messages", buff)
	if err != nil {
		return nil, err
	}

	req.Header.Add("x-api-key", s.apiKey)
	req.Header.Add("anthropic-version", anthropicVersion)
	req.Header.Add("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	b, err = io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		var e anthropicAPIError
		err = json.Unmarshal(b, &e)
		if err != nil {
			return nil, err
		}
		return nil, e
	}

	err = json.Unmarshal(b, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func (s *AnthropicService) CreateChatCompletion(ctx context.Context, request *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	modelURI := ParseModelURI(request.Model)
	aRequest := anthropicMessageRequest{
		Model:     modelURI.Name,
		MaxTokens: 1024,
		Stream:    false,
	}

	if request.Stream != nil {
		aRequest.Stream = *request.Stream
	}

	if len(request.Messages) == 0 {
		return nil, fmt.Errorf("No messages were passed in")
	}

	messages := request.Messages
	if messages[0].Role == ChatMessageRoleSystem {
		aRequest.System = []anthropicMessageContent{
			{
				Type: "text",
				Text: messages[0].Content,
			},
		}
		if len(messages) > 1 {
			messages = messages[1:]
		}
	}

	for _, m := range messages {
		if m.Role == ChatMessageRoleFunction {
			aRequest.Messages = append(aRequest.Messages, anthropicMessage{
				Role: ChatMessageRoleUser,
				Content: []anthropicMessageContent{
					{
						ToolUseId: idFromMeta(&m),
						Type:      "tool_result",
						Content:   m.Content,
					},
				},
			})
		} else {
			content := []anthropicMessageContent{
				{
					Type: "text",
					Text: m.Content,
				},
			}

			if len(m.ToolCalls) > 0 {
				for _, tool := range m.ToolCalls {
					content = append(content, anthropicMessageContent{
						Id:    idFromMeta(&tool),
						Type:  "tool_use",
						Name:  tool.Name,
						Input: &tool.Arguments,
					})
				}
			}

			aRequest.Messages = append(aRequest.Messages, anthropicMessage{
				Role:    m.Role,
				Content: content,
			})
		}
	}

	toolMap := make(map[string]string)
	if len(request.Tools) > 0 {
		for _, t := range request.Tools {
			normalized := normalizeNamelike(t.Name)
			toolMap[normalized] = t.Name

			if t.InputSchema != nil && t.InputSchema.Type == "" {
				t.InputSchema.Type = "object"
			}

			aRequest.Tools = append(aRequest.Tools, anthropicTool{
				Name:        normalized,
				Description: t.Description,
				InputSchema: *t.InputSchema,
			})
		}
	}

	aResponse, err := s.messageRequest(ctx, &aRequest)
	if err != nil {
		return nil, err
	}

	response := ChatCompletionResponse{
		Model:     aResponse.Model,
		CreatedAt: time.Now(),
		Message: ChatCompletionMessage{
			Role: aResponse.Role,
		},
	}

	// TODO: Support multi content
	for _, content := range aResponse.Content {
		switch content.Type {
		case "text":
			response.Message.Content = content.Text
		case "tool_use":
			t := ToolCall{
				Name:      toolMap[content.Name],
				Arguments: *content.Input,
			}

			t.SetMeta("id", content.Id)

			response.Message.ToolCalls = append(response.Message.ToolCalls, t)
		}
	}

	return &response, nil
}

func (s *AnthropicService) CreateEmbedding(ctx context.Context, request *EmbeddingRequest) (Embedding, error) {

	return nil, nil
}
