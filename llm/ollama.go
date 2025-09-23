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
)

type OllamaService struct{}

type OllamaEmbeddingReqeust struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type OllamaEmbeddingResponse struct {
	Embedding []float32 `json:"embedding"`
}

func (s OllamaService) CreateChatCompletion(c context.Context, request *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	var response ChatCompletionResponse

	return &response, nil
}

func (s OllamaService) CreateEmbedding(c context.Context, request *EmbeddingRequest) (Embedding, error) {
	var embedding Embedding

	serviceURL := os.Getenv("OLLAMA_SERVICE_URL")
	if serviceURL == "" {
		return embedding, errors.New("arboreal.CreateEmbedding: OLLAMA_SERVICE_URL environment variable not set")
	}

	model := "nomic-embed-text"
	if request.Model != "" {
		model = ParseModelURI(request.Model).Name
	}

	b, err := json.Marshal(OllamaEmbeddingReqeust{
		Model:  model,
		Prompt: request.Input,
	})
	if err != nil {
		return embedding, fmt.Errorf("arboreal.CreateEmbedding: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/embeddings", serviceURL), bytes.NewReader(b))
	if err != nil {
		return embedding, fmt.Errorf("arboreal.CreateEmbedding: %w", err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return embedding, fmt.Errorf("arboreal.CreateEmbedding: %w", err)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return embedding, fmt.Errorf("arboreal.CreateEmbedding: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return embedding, fmt.Errorf("arboreal.CreateEmbedding: %s", res.Status)
	}

	var response OllamaEmbeddingResponse
	if err = json.Unmarshal(body, &response); err != nil {
		return embedding, fmt.Errorf("arboreal.CreateEmbedding: %w", err)
	}

	embedding = response.Embedding

	return embedding, nil
}
