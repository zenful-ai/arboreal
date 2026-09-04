// A scratch probe (not a real example): see what ExtraContext adds to a
// system prompt.
//
// One LLMCompletionState is called directly — no tree, no executive — on a
// history whose user message carries two annotations set by hand, standing
// in for what a lookup state would have written. The state lists those names
// in ExtraContext, and because it prepends its rendered system prompt to the
// history, the dump afterwards shows exactly what the model was sent.
//
// Requires OPENAI_TOKEN to be set in the environment.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/zenful-ai/arboreal"
	"github.com/zenful-ai/arboreal/llm"
)

func main() {
	answer := arboreal.LLMCompletionState(arboreal.LLMCompletionOptions{
		System: "You are a CRM assistant. Answer in one sentence " +
			"using only the extra context below.",
		// "membank" finds nothing on this history and is skipped.
		ExtraContext: []string{"name", "context", "membank"},
	})

	history := arboreal.AppendToMessages(nil, llm.ChatCompletionMessage{
		Role:    llm.ChatMessageRoleUser,
		Content: "When is Bob's daughter's birthday?",
	})
	history[0].Annotations["name"] = arboreal.Annotation{Data: "Bob Marley"}
	history[0].Annotations["context"] = arboreal.Annotation{
		Data: "Bob Marley has one daughter, Cedella, born on August 23rd.",
	}

	history, sig := answer.Call(context.Background(), history)
	if err, ok := sig.(*arboreal.ErrorSignal); ok {
		log.Fatal(err)
	}

	for i, m := range history {
		fmt.Printf("[%d] %-9s %q\n", i, m.Role, m.Content)
	}
}
