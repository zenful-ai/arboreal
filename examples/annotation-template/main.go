// A scratch probe (not a real example): see AnnotationTemplate render each of
// its three forms against a history, without a model in the way.
//
// LLMCompletionState runs the System option through exactly this code before
// every model call, so what prints here is what a prompt would say. We render
// one template twice: first with the "pref" annotation missing, then with it
// set, to show the multi-word blocks vanish and reappear as a whole.
package main

import (
	"fmt"
	"strings"

	"github.com/zenful-ai/arboreal"
	"github.com/zenful-ai/arboreal/llm"
)

const prompt = "Hello {{ name }}, welcome back. " +
	"{{ Visits so far: visits? }} " +
	"{{ Preference: pref? }} " +
	"{{ Sure?? pref? }}\n" +
	"The user just said: {{ $last_message }}"

func main() {
	history := arboreal.AppendToMessages(nil, llm.ChatCompletionMessage{
		Role:    llm.ChatMessageRoleUser,
		Content: "Something warm, please.",
	})
	history[0].Annotations["name"] = arboreal.Annotation{Data: "Joe"}
	history[0].Annotations["visits"] = arboreal.Annotation{Data: 3}

	fmt.Println("=== without pref ===")
	fmt.Println(render(prompt, history))

	history[0].Annotations["pref"] = arboreal.Annotation{Data: "tea"}

	fmt.Println("=== with pref ===")
	fmt.Println(render(prompt, history))
}

func render(text string, history arboreal.AnnotatedMessages) string {
	var tmpl arboreal.AnnotationTemplate
	if _, err := tmpl.Parse(text); err != nil {
		panic(err)
	}
	var out strings.Builder
	if err := tmpl.Execute(&out, history); err != nil {
		panic(err)
	}
	return out.String()
}
