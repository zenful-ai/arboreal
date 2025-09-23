package main

import (
	"fmt"
	"github.com/zenful-ai/arboreal"
	"github.com/zenful-ai/arboreal/llm"
	"io/ioutil"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: semchunks <file>")
	}

	file := os.Args[1]
	if file == "" {
		fmt.Println("Usage: semchunks <file>")
	}

	t, err := ioutil.ReadFile(file)
	if err != nil {
		panic(err)
	}

	text := string(t)

	chunker := arboreal.NewSemanticChunker(llm.OllamaService{})
	chunks, err := chunker.Chunk(text)
	if err != nil {
		panic(err)
	}

	for idx, chunk := range chunks {
		fmt.Printf("Chunk %d [%d...%d]\n%s\n\n", idx, chunk.Start, chunk.End, chunk.Text)
	}
}
