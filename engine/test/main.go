package main

import (
	"context"
	"fmt"
	"github.com/zenful-ai/arboreal"
	"github.com/zenful-ai/arboreal/engine"
	"github.com/zenful-ai/arboreal/llm"
	"io/ioutil"
	"os"
)

func main() {
	b, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}

	runtime, err := engine.InitializeRuntime(string(b), engine.CurrentRuntimeVersion, nil)
	if err != nil {
		panic(err)
	}

	var history arboreal.AnnotatedMessages
	c := arboreal.TerminalChannel{}

	for {
		m, err := c.Receive()

		if err != nil {
			panic(err)
		}

		var sig arboreal.Signal

		history = append(history, arboreal.AnnotatedMessage{
			ChatCompletionMessage: llm.ChatCompletionMessage{
				Role:    llm.ChatMessageRoleUser,
				Content: m.Content,
			},
			Annotations: make(map[string]arboreal.Annotation),
		})

		history, sig = runtime.Entry().Call(context.Background(), history)

		if e, ok := sig.(*arboreal.ErrorSignal); ok {
			fmt.Println("Error: ", e.Error())
			return
		}

		if history.LastMessage().Role == llm.ChatMessageRoleAssistant {
			c.Send(&arboreal.ChannelMessage{
				Content: history.LastMessage().Content,
			})
		}
	}

	return
}
