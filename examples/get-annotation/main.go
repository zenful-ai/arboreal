// A scratch probe (not a real example): see what GetAnnotation returns for a
// three-message history, without a model in the way.
//
// It shows the newest-first walk (a later write under the same name wins),
// that the result is a pointer to a copy, the three $ meta-annotations, and
// the nil cases. Nothing here needs OPENAI_TOKEN.
package main

import (
	"fmt"

	"github.com/zenful-ai/arboreal"
	"github.com/zenful-ai/arboreal/llm"
)

func main() {
	var history arboreal.AnnotatedMessages
	history = user(history, "Hi, I'm Joe and I like coffee.")
	history[0].Annotations["name"] = arboreal.Annotation{Data: "Joe"}
	history[0].Annotations["pref"] = arboreal.Annotation{Data: "coffee"}

	history = assistant(history, "Nice to meet you, Joe.")

	history = user(history, "Actually, make that tea.")
	history[2].Annotations["pref"] = arboreal.Annotation{Data: "tea"}

	// Newest-first: "pref" is on messages 0 and 2, and the later one wins.
	// "name" is only on message 0, so the walk reaches back to it.
	show(history, "pref") // tea
	show(history, "name") // Joe

	// The pointer is to a copy: writing through it does not touch the history.
	p := history.GetAnnotation("pref")
	p.Data = "cocoa"
	show(history, "pref") // tea

	// Meta-annotations, synthesized when nothing is stored under the name.
	show(history, "$last_message") // Actually, make that tea.
	show(history, "$date")         // 2026-09-01T19:11:54+02:00
	show(history, "$date_llm")     // Today's date is: Tue Sep 01 19:11:54 +0200 2026.

	// A stored annotation shadows a meta-annotation of the same name.
	history[2].Annotations["$date"] = arboreal.Annotation{Data: "1999-12-31T23:59:59Z"}
	show(history, "$date") // 1999-12-31T23:59:59Z

	// Missing names, plain or $-prefixed, are nil.
	show(history, "age")   // nil
	show(history, "$user") // nil
}

func show(history arboreal.AnnotatedMessages, name string) {
	a := history.GetAnnotation(name)
	if a == nil {
		fmt.Printf("%-14s -> nil\n", name)
		return
	}
	fmt.Printf("%-14s -> %#v\n", name, a.Data)
}

func user(h arboreal.AnnotatedMessages, content string) arboreal.AnnotatedMessages {
	return arboreal.AppendToMessages(h, llm.ChatCompletionMessage{Role: llm.ChatMessageRoleUser, Content: content})
}

func assistant(h arboreal.AnnotatedMessages, content string) arboreal.AnnotatedMessages {
	return arboreal.AppendToMessages(h, llm.ChatCompletionMessage{Role: llm.ChatMessageRoleAssistant, Content: content})
}
