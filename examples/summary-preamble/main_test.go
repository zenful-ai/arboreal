package main

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/zenful-ai/arboreal"
	"github.com/zenful-ai/arboreal/llm"
)

func sampleCalendar() *Calendar {
	return NewCalendar(map[string]string{"Weekly review": "14:00", "Vendor call": "16:30"})
}

func TestRescheduleMovesTheEvent(t *testing.T) {
	cal := sampleCalendar()
	got := cal.Reschedule("weekly review", "15:00")
	if want := "Weekly review moved from 14:00 to 15:00."; got != want {
		t.Fatalf("Reschedule = %q, want %q", got, want)
	}
	if at := cal.events["Weekly review"]; at != "15:00" {
		t.Fatalf("the calendar has the weekly review at %q, want 15:00", at)
	}
}

func TestCancelRemovesTheEvent(t *testing.T) {
	cal := sampleCalendar()
	got := cal.Cancel("Vendor Call")
	if want := "Vendor call cancelled."; got != want {
		t.Fatalf("Cancel = %q, want %q", got, want)
	}
	if _, still := cal.events["Vendor call"]; still {
		t.Fatal("the vendor call is still on the calendar")
	}
}

func TestUnknownEventIsReported(t *testing.T) {
	cal := sampleCalendar()
	if got, want := cal.Reschedule("Standup", "09:00"), "There is no event named Standup on the calendar."; got != want {
		t.Fatalf("Reschedule = %q, want %q", got, want)
	}
	if got, want := cal.Cancel("Standup"), "There is no event named Standup on the calendar."; got != want {
		t.Fatalf("Cancel = %q, want %q", got, want)
	}
}

// TestFieldsReadsTheExtractedObject pins both shapes annotation mode can
// leave: the parsed object, and the raw text when the model fenced it.
func TestFieldsReadsTheExtractedObject(t *testing.T) {
	want := map[string]string{"event": "Weekly review", "time": "15:00"}

	parsed := &arboreal.Annotation{Data: map[string]any{"event": "Weekly review", "time": "15:00"}}
	if got := fields(parsed); !reflect.DeepEqual(got, want) {
		t.Fatalf("fields(parsed object) = %v, want %v", got, want)
	}

	fenced := &arboreal.Annotation{Data: "```json\n{\"data\": {\"event\": \"Weekly review\", \"time\": \"15:00\"}}\n```"}
	if got := fields(fenced); !reflect.DeepEqual(got, want) {
		t.Fatalf("fields(fenced text) = %v, want %v", got, want)
	}

	if got := fields(nil); len(got) != 0 {
		t.Fatalf("fields(nil) = %v, want empty", got)
	}
}

// TestApplyStateRepliesWithTheConfirmation runs the second state of a tree
// by hand, with the extraction already pinned on the direction, the way the
// first state would have left it. No model is involved.
func TestApplyStateRepliesWithTheConfirmation(t *testing.T) {
	cal := sampleCalendar()
	state := applyState("state-test", "reschedule", func(f map[string]string) string {
		return cal.Reschedule(f["event"], f["time"])
	})

	history := arboreal.AppendToMessages(nil, llm.ChatCompletionMessage{
		Role: llm.ChatMessageRoleUser, Content: "The user wrote: \"move the weekly review to 15:00\"",
	})
	history[0].Annotations["reschedule"] = arboreal.Annotation{
		Name: "reschedule", Data: map[string]any{"event": "weekly review", "time": "15:00"},
	}

	out, sig := state.Lambda(context.Background(), history)
	if sig != nil {
		t.Fatalf("signal = %v, want nil", sig)
	}
	last := out.LastMessage()
	if last.Role != llm.ChatMessageRoleAssistant || last.Content != "Weekly review moved from 14:00 to 15:00." {
		t.Fatalf("last message = %s %q, want the assistant's confirmation", last.Role, last.Content)
	}
}

// TestPreambleSplit pins what the example is about: the planner's Preamble
// carries the mechanics (the JSON shapes, the [] rule) and the
// SummaryPreamble carries none of them; the flag leaves the second unset.
func TestPreambleSplit(t *testing.T) {
	exec := buildExecutive(sampleCalendar(), true)
	if !strings.Contains(exec.Preamble, "[]") || !strings.Contains(exec.Preamble, "{") {
		t.Fatalf("Preamble = %q, want the [] rule and a JSON shape in it", exec.Preamble)
	}
	if exec.SummaryPreamble == "" || strings.ContainsAny(exec.SummaryPreamble, "[]{}") {
		t.Fatalf("SummaryPreamble = %q, want prose with no bracket in it", exec.SummaryPreamble)
	}

	bare := buildExecutive(sampleCalendar(), false)
	if bare.SummaryPreamble != "" {
		t.Fatalf("SummaryPreamble = %q with the flag, want empty", bare.SummaryPreamble)
	}
	if bare.Preamble != exec.Preamble {
		t.Fatal("the flag changed the planner's Preamble; it must only drop the SummaryPreamble")
	}
}
