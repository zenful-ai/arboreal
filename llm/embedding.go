package llm

import (
	"encoding/binary"
	"math"
)

type Embedding []float32

func (e *Embedding) ToData() []byte {
	buf := make([]byte, len(*e)*4)
	for i, f := range *e {
		u := math.Float32bits(f)
		binary.LittleEndian.PutUint32(buf[i*4:], u)
	}
	return buf
}

type EmbeddingRequest struct {
	Model string
	Input string
}
