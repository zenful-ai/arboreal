package llm

import (
	"github.com/modelcontextprotocol/go-sdk/jsonschema"
)

// Tools

const (
	ParameterTypeObject  = "object"
	ParameterTypeString  = "string"
	ParameterTypeInteger = "integer"
	ParameterTypeBoolean = "boolean"

	ChatToolTypeFunction = "function"
)

type ChatTool struct {
	Type        string             `json:"type"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	InputSchema *jsonschema.Schema `json:"input_schema"`
}

type ToolCall struct {
	Meta      map[string]any `json:"-"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

func (t *ToolCall) GetMeta(key string) any {
	if t.Meta == nil {
		return nil
	}

	return t.Meta[key]
}

func (t *ToolCall) SetMeta(key string, value any) {
	if t.Meta == nil {
		t.Meta = make(map[string]any)
	}

	t.Meta[key] = value
}
