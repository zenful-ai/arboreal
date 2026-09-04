package main

import (
	"reflect"
	"testing"

	"github.com/zenful-ai/arboreal"
)

// ANCHOR: helper
func annotated(role, content string, facts map[string]any) arboreal.AnnotatedMessage {
	m := arboreal.AnnotatedMessage{}
	m.Role = role
	m.Content = content
	m.Annotations = make(map[string]arboreal.Annotation)
	for k, v := range facts {
		m.Annotations[k] = arboreal.Annotation{Name: k, Data: v}
	}
	return m
}

// ANCHOR_END: helper

// ANCHOR: merge
func TestLearnedMergesAcrossMessages(t *testing.T) {
	history := arboreal.AnnotatedMessages{
		annotated("user", "I'm John", map[string]any{
			"first name": "John", "last name": "", "age": "", "location": "",
		}),
		annotated("assistant", "Nice to meet you!", nil),
		annotated("user", "Smith, from Kraków", map[string]any{
			"first name": "", "last name": "Smith", "age": "", "location": "Kraków",
		}),
	}

	got := learned(history)
	want := map[string]string{"first name": "John", "last name": "Smith", "location": "Kraków"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("learned() = %v, want %v", got, want)
	}
}

// ANCHOR_END: merge

func TestLearnedIgnoresUnknownAndKeepsLatest(t *testing.T) {
	history := arboreal.AnnotatedMessages{
		annotated("user", "I'm 34", map[string]any{"age": float64(34)}),
		annotated("user", "Actually 35", map[string]any{"age": "35"}),
		annotated("user", "hello", map[string]any{"age": "unknown", "location": "null", "first name": `{"data": null}`}),
	}

	got := learned(history)
	want := map[string]string{"age": "35"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("learned() = %v, want %v", got, want)
	}
}

func TestLearnedIgnoresForeignAnnotations(t *testing.T) {
	history := arboreal.AnnotatedMessages{
		annotated("user", "hi", map[string]any{"__trace_annotations": "first name,age", "plan": "[]"}),
	}
	if got := learned(history); len(got) != 0 {
		t.Fatalf("learned() = %v, want empty", got)
	}
}
