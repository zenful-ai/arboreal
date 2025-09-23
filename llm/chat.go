package llm

import (
	"time"
)

const (
	ChatMessageRoleSystem    = "system"
	ChatMessageRoleUser      = "user"
	ChatMessageRoleAssistant = "assistant"
	ChatMessageRoleFunction  = "function"

	ChatContentTypeText       = "text/plain"
	ChatContentTypeJSON       = "application/json"
	ChatContentTypePDF        = "application/pdf"
	ChatContentTypePowerpoint = "application/vnd.ms-powerpoint"
	ChatContentTypeZip        = "application/zip"
	ChatContentTypeExcel      = "application/vnd.ms-excel"
	ChatContentTypeHTML       = "text/html"
	ChatContentTypeMSDoc      = "application/msword"
)

// The meta interface serves as a useful way to pass around provider-specific implementation details.
// For example, Anthropic expects tool_use and tool_results to have matching Id strings.
type Meta interface {
	GetMeta(key string) any
	SetMeta(key string, value any)
}

type ChatContent struct {
	Type           string `json:"type"`
	Content        []byte `json:"content"`
	TextEquivalent string `json:"text_equivalent,omitempty"`
}

type ChatCompletionMessage struct {
	Meta         map[string]any `json:"-"`
	Context      string         `json:"context,omitempty"`
	Role         string         `json:"role"`
	Content      string         `json:"content,omitempty"`
	MultiContent []ChatContent  `json:"multi_content,omitempty"`
	ToolCalls    []ToolCall     `json:"tool_calls,omitempty"`
	Format       string         `json:"format,omitempty"`
	Name         string         `json:"name,omitempty"`
}

func (c *ChatCompletionMessage) GetMeta(key string) any {
	if c.Meta == nil {
		return nil
	}

	return c.Meta[key]
}

func (c *ChatCompletionMessage) SetMeta(key string, value any) {
	if c.Meta == nil {
		c.Meta = make(map[string]any)
	}

	c.Meta[key] = value
}

type ChatCompletionRequest struct {
	Model    string                  `json:"model"`
	Messages []ChatCompletionMessage `json:"messages"`
	Tools    []ChatTool              `json:"tools,omitempty"`
	Options  map[string]any          `json:"options,omitempty"`
	Format   string                  `json:"format,omitempty"`
	Stream   *bool                   `json:"stream"`
}

type ChatCompletionResponse struct {
	Model     string                `json:"model"`
	CreatedAt time.Time             `json:"created_at"`
	Message   ChatCompletionMessage `json:"message"`
}
