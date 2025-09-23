package arboreal

import (
	"context"
	"fmt"
	"github.com/ncruces/go-sqlite3"
	"github.com/zenful-ai/arboreal/llm"
	"strings"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/ncruces"
)

type MemoryChunk struct {
	Distance  float64       `json:"distance"`
	Text      string        `json:"text"`
	Embedding llm.Embedding `json:"-"`
	Metadata  string        `json:"metadata"`
}

type MemoryStore struct {
	db       *sqlite3.Conn
	provider llm.ModelProvider
}

func CreateMemoryStore(db *sqlite3.Conn, provider llm.ModelProvider) *MemoryStore {
	return &MemoryStore{db: db, provider: provider}
}

func (m *MemoryStore) CreateMemoryBankIfNotExists(name string) error {
	stmt, _, err := m.db.Prepare(fmt.Sprintf(`create virtual table if not exists mb_%s using vec0( embedding float[768] distance_metric=cosine, chunk text, metadata text );`, name))
	if err != nil {
		return err
	}

	stmt.Step()

	return err
}

func (m *MemoryStore) Store(ctx context.Context, bank string, chunk string, metadata string) error {
	stmt, _, err := m.db.Prepare(fmt.Sprintf("insert into mb_%s (embedding, chunk, metadata) values (?, ?, ?);", bank))
	if err != nil {
		return err
	}

	embedding, err := m.provider.CreateEmbedding(ctx, &llm.EmbeddingRequest{
		Input: chunk,
	})
	if err != nil {
		return err
	}

	b, err := sqlite_vec.SerializeFloat32(embedding)
	if err != nil {
		return err
	}

	err = stmt.BindBlob(1, b)
	if err != nil {
		return err
	}

	err = stmt.BindText(2, chunk)
	if err != nil {
		return err
	}
	err = stmt.BindText(3, metadata)
	if err != nil {
		return err
	}
	err = stmt.Exec()
	if err != nil {
		return err
	}

	return nil
}

func (m *MemoryStore) StoreBatch(ctx context.Context, bank string, chunks []string, metadatas []string) error {
	if len(chunks) != len(metadatas) {
		return fmt.Errorf("chunks and metadatas must have the same length")
	}

	var err error
	tx := m.db.Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	stmt, _, err := m.db.Prepare(fmt.Sprintf(
		"INSERT INTO mb_%s (embedding, chunk, metadata) VALUES (?, ?, ?);", bank))
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	m.provider.CreateEmbedding(ctx, &llm.EmbeddingRequest{})

	for i, chunk := range chunks {
		embedding, err := m.provider.CreateEmbedding(ctx, &llm.EmbeddingRequest{
			Input: chunk,
		})
		if err != nil {
			tx.Rollback()
			return err
		}

		b, err := sqlite_vec.SerializeFloat32(embedding)
		if err != nil {
			tx.Rollback()
			return err
		}

		stmt.ClearBindings()
		stmt.BindBlob(1, b)
		stmt.BindText(2, chunk)
		stmt.BindText(3, metadatas[i])

		stmt.Step()
		if err := stmt.Err(); err != nil {
			tx.Rollback()
			return err
		}
		stmt.Reset()
	}

	return tx.Commit()
}

func (m *MemoryStore) Recall(ctx context.Context, bank, query, prefix string) ([]MemoryChunk, error) {
	stmt, _, err := m.db.Prepare(fmt.Sprintf(`
		SELECT
			distance,
			chunk,
			metadata
		FROM mb_%s
		WHERE embedding MATCH ?
		ORDER BY distance
		LIMIT 1000
	`, bank))
	if err != nil {
		return nil, err
	}

	embedding, err := m.provider.CreateEmbedding(ctx, &llm.EmbeddingRequest{
		Input: query,
	})
	if err != nil {
		return nil, err
	}

	b, err := sqlite_vec.SerializeFloat32(embedding)
	if err != nil {
		return nil, err
	}

	err = stmt.BindBlob(1, b)
	if err != nil {
		return nil, err
	}

	var results []MemoryChunk
	var charCount int

	for stmt.Step() {
		distance := stmt.ColumnFloat(0)
		chunk := stmt.ColumnText(1)
		metadata := stmt.ColumnText(2)

		if prefix != "" && !strings.HasPrefix(metadata, prefix) {
			continue
		}

		results = append(results, MemoryChunk{
			Distance: distance,
			Text:     chunk,
			Metadata: metadata,
		})
		charCount += len(chunk)
	}

	return results, nil
}
