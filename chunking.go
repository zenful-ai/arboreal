package arboreal

import (
	"context"
	"github.com/neurosnap/sentences/english"
	"github.com/zenful-ai/arboreal/llm"
	"strings"
)

type ChunkingStrategy interface {
	Chunk(s string) ([]string, error)
}

type SemanticChunker struct {
	Threshold float64
	sp        llm.ModelProvider
}

type Chunk struct {
	Start     int
	End       int
	Text      string
	Embedding string
}

func NewSemanticChunker(m llm.ModelProvider) *SemanticChunker {
	return &SemanticChunker{Threshold: .65, sp: m}
}

func (s *SemanticChunker) Chunk(c string) ([]Chunk, error) {
	tokenizer, err := english.NewSentenceTokenizer(nil)
	if err != nil {
		return nil, err
	}

	var chunks []Chunk
	var currentChunkText string

	var previousEmbedding llm.Embedding

	sentences := tokenizer.Tokenize(c)
	var chunkStart int
	for _, sentence := range sentences {
		embedding, err := s.sp.CreateEmbedding(context.Background(), &llm.EmbeddingRequest{
			Input: sentence.Text,
		})
		if err != nil {
			panic(err)
		}

		if previousEmbedding == nil {
			previousEmbedding = embedding
			currentChunkText += sentence.Text
			continue
		}

		similarity := CosineSimilarity(previousEmbedding, embedding)
		if similarity > s.Threshold {
			currentChunkText += sentence.Text
		} else {
			chunks = append(chunks, Chunk{
				Start: chunkStart,
				End:   sentence.End,
				Text:  currentChunkText,
			})

			currentChunkText = sentence.Text
			chunkStart = sentence.End + 1
			previousEmbedding = nil
			continue
		}

		previousEmbedding = embedding
	}

	if len(sentences) > 0 && strings.Trim(currentChunkText, " \t\n") != "" {
		chunks = append(chunks, Chunk{
			Start: chunkStart,
			End:   sentences[len(sentences)-1].End,
			Text:  currentChunkText,
		})
	}

	return chunks, nil
}
