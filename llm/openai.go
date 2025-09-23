package llm

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
	"github.com/zenful-ai/arboreal/util"
)

type OpenAIService struct{}

func createOpenAIClient() (*openai.Client, error) {
	var token string

	if token = os.Getenv("OPENAI_TOKEN"); token == "" {
		return nil, errors.New("OPENAI_TOKEN environment variable not set")
	}

	client := openai.NewClient(token)

	return client, nil
}

func normalizeNamelike(name string) string {
	// TODO: Properly normalize to *only* the characters OpenAI allows ([a-zA-Z_])
	normalized := name
	parts := strings.Split(name, "/")
	if len(parts) >= 2 {
		normalized = parts[len(parts)-1]
	}

	normalized = strings.ReplaceAll(normalized, ".", "_")
	return normalized
}

func (s OpenAIService) CreateChatCompletion(c context.Context, request *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	var r = make(chan *ChatCompletionResponse)
	var e = make(chan error)

	go func() {
		client, err := createOpenAIClient()
		if err != nil {
			e <- err
			return
		}

		var functionLookup = make(map[string]string)
		var functions []openai.FunctionDefinition
		for _, tool := range request.Tools {
			if tool.Type == ChatToolTypeFunction {
				// Tool names may be namespaced, so let's regularize them, since OpenAI doesn't allow for most characters
				toolName := normalizeNamelike(tool.Name)
				functionLookup[toolName] = tool.Name

				functions = append(functions, openai.FunctionDefinition{
					Name:        toolName,
					Description: tool.Description,
					Parameters:  tool.InputSchema,
				})
			}
		}

		var messages []openai.ChatCompletionMessage
		for _, message := range request.Messages {
			m := openai.ChatCompletionMessage{
				Role:    message.Role,
				Content: message.Content,
				Name:    normalizeNamelike(message.Name),
			}

			if len(message.ToolCalls) > 0 {
				b, err := json.Marshal(message.ToolCalls[0].Arguments)
				if err != nil {
					e <- err
					return
				}

				m.FunctionCall = &openai.FunctionCall{
					Name:      normalizeNamelike(message.ToolCalls[0].Name),
					Arguments: string(b),
				}
			}

			messages = append(messages, m)
		}

		var temperature float32 = 0.7
		if t, ok := request.Options["temperature"].(float32); ok {
			temperature = t
		}

		var response openai.ChatCompletionResponse
		err = util.RetryWithBackoff(func() (ex error) {
			response, ex = client.CreateChatCompletion(
				context.Background(),
				openai.ChatCompletionRequest{
					Model:       ParseModelURI(request.Model).Name,
					Messages:    messages,
					Functions:   functions,
					Temperature: temperature,
				},
			)

			return
		}, 4)
		if err != nil {
			e <- err
			return
		}

		var toolCalls []ToolCall

		if response.Choices[0].Message.FunctionCall != nil {
			call := ToolCall{
				Name: functionLookup[response.Choices[0].Message.FunctionCall.Name],
			}

			err = json.Unmarshal([]byte(response.Choices[0].Message.FunctionCall.Arguments), &call.Arguments)
			if err != nil {
				e <- err
				return
			}

			toolCalls = append(toolCalls, call)
		}

		r <- &ChatCompletionResponse{
			Model:     request.Model,
			CreatedAt: time.Now(),
			Message: ChatCompletionMessage{
				Role:      ChatMessageRoleAssistant,
				Content:   response.Choices[0].Message.Content,
				ToolCalls: toolCalls,
			},
		}
	}()

	select {
	case <-c.Done():
		return nil, c.Err()
	case err := <-e:
		return nil, err
	case response := <-r:
		return response, nil
	}
}

func (s OpenAIService) CreateEmbedding(ctx context.Context, request *EmbeddingRequest) (Embedding, error) {
	client, err := createOpenAIClient()
	if err != nil {
		return Embedding{}, err
	}

	model := openai.SmallEmbedding3
	if request.Model != "" {
		model = openai.EmbeddingModel(
			ParseModelURI(request.Model).Name,
		)
	}

	r, err := client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input:          request.Input,
		Model:          model,
		EncodingFormat: openai.EmbeddingEncodingFormatFloat,
	})
	if err != nil {
		return Embedding{}, err
	}

	if len(r.Data) < 1 {
		return Embedding{}, errors.New("embedding is empty")
	}

	return r.Data[0].Embedding, nil
}
