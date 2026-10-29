// Package main is a learning-purposes example, NOT a template for real apps.
//
// It shows why the executive's summarizer has a preamble of its own. The
// executive is a team's scheduling assistant with two behaviors, one moves
// a calendar event and one cancels it, and a request that needs both:
//
//	$ go run ./examples/summary-preamble
//	$ go run ./examples/summary-preamble -no-summary-preamble
//
// Both runs plan two steps, run them in parallel, change the calendar the
// same way, and hand the same two one-sentence confirmations to the
// summarizer, the model call that merges them into the reply (Chapter 8).
// Only the reply differs. The first run opens the summarizer prompt with
// SummaryPreamble and the reply is prose: "The weekly review has been moved
// to 15:00, and the vendor call has been cancelled." The second leaves
// SummaryPreamble unset, so the summarizer prompt opens with the planner's
// Preamble instead, as it did before the field existed, and the reply comes
// back as JSON, in a different shape each run: the two extracted objects in
// an array, or bare and comma-separated, or an object keyed by the
// behaviors' names:
//
//	[{"event": "weekly review", "time": "15:00"}, {"event": "vendor call"}]
//
// Nothing in that JSON came from the steps. The planner's Preamble is the
// only free-text control over the planner, so it carries the planner's
// mechanics: what each behavior's direction must say, the JSON shape each
// step extracts from it, the quoted-direction format, the []-for-out-of-
// bounds rule. In the summarizer prompt those sentences are the only ones
// that describe a shape, and a merged reply takes it. SummaryPreamble is
// the identity and tone alone, and reaches no other prompt (Chapter 15,
// Faithful Directions via Preamble).
//
// Each behavior has JSON under the hood: an annotation-mode state extracts
// the event (and the time) from the planner's direction as a small object,
// and a lambda state applies it to an in-memory calendar and confirms in a
// sentence. The summarizer never sees that JSON; it sees the sentences.
//
// Pass your own request as arguments to try other combinations. Requires
// OPENAI_TOKEN to be set in the environment; the model is gpt-4o-mini via
// the OpenAI provider, named on the context with WithDefaultModel.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/zenful-ai/arboreal"
	"github.com/zenful-ai/arboreal/llm"
)

// Calendar is the team's calendar, event name to time. It is guarded
// because the executive runs a turn's steps in parallel (Chapter 8), and
// this turn's two steps both write it.
type Calendar struct {
	mu     sync.Mutex
	events map[string]string
}

func NewCalendar(events map[string]string) *Calendar {
	c := &Calendar{events: make(map[string]string, len(events))}
	for name, at := range events {
		c.events[name] = at
	}
	return c
}

// find returns the stored name of the event the user called name, matched
// without regard to case, or "" when there is none.
func (c *Calendar) find(name string) string {
	for stored := range c.events {
		if strings.EqualFold(stored, strings.TrimSpace(name)) {
			return stored
		}
	}
	return ""
}

// Reschedule moves the event to at and returns the confirmation sentence.
func (c *Calendar) Reschedule(name, at string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	stored := c.find(name)
	if stored == "" {
		return fmt.Sprintf("There is no event named %s on the calendar.", strings.TrimSpace(name))
	}
	was := c.events[stored]
	c.events[stored] = at
	return fmt.Sprintf("%s moved from %s to %s.", stored, was, at)
}

// Cancel removes the event and returns the confirmation sentence.
func (c *Calendar) Cancel(name string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	stored := c.find(name)
	if stored == "" {
		return fmt.Sprintf("There is no event named %s on the calendar.", strings.TrimSpace(name))
	}
	delete(c.events, stored)
	return fmt.Sprintf("%s cancelled.", stored)
}

// String lists the events, one per line, in name order.
func (c *Calendar) String() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	names := make([]string, 0, len(c.events))
	for name := range c.events {
		names = append(names, name)
	}
	sort.Strings(names)
	var b strings.Builder
	for _, name := range names {
		fmt.Fprintf(&b, "  %s at %s\n", name, c.events[name])
	}
	return b.String()
}

// fields reads the object an annotation-mode state extracted. Annotation
// mode stores the model's {"data": {...}} parsed, so Data is the object;
// when the model fenced its JSON the reply did not parse and Data is the
// raw text, which is parsed here after the fences are stripped (Chapter 4).
// Anything else reads as no fields.
func fields(a *arboreal.Annotation) map[string]string {
	out := make(map[string]string)
	if a == nil {
		return out
	}
	object, ok := a.Data.(map[string]any)
	if !ok {
		text, isText := a.Data.(string)
		if !isText {
			return out
		}
		text = strings.TrimSpace(text)
		text = strings.TrimPrefix(text, "```json")
		text = strings.TrimPrefix(text, "```")
		text = strings.TrimSuffix(text, "```")
		var reply struct {
			Data map[string]any `json:"data"`
		}
		if err := json.Unmarshal([]byte(text), &reply); err != nil || reply.Data == nil {
			return out
		}
		object = reply.Data
	}
	for key, value := range object {
		if s, ok := value.(string); ok {
			out[key] = s
		}
	}
	return out
}

// applyState is the second state of each tree: it reads the object the
// extraction state pinned under name on the direction, applies it to the
// calendar through apply, and replies with the confirmation apply returns.
// A missing or unreadable pin still gets a sentence, from apply's handling
// of empty fields, so the step always finishes with a reply.
func applyState(id, name string, apply func(f map[string]string) string) arboreal.BehaviorState {
	return arboreal.BehaviorState{
		HashId:    id,
		StateName: "apply",
		Lambda: func(ctx context.Context, history arboreal.AnnotatedMessages) (arboreal.AnnotatedMessages, arboreal.Signal) {
			reply := apply(fields(history.GetAnnotation(name)))
			return arboreal.AppendToMessages(history, llm.ChatCompletionMessage{
				Role:    llm.ChatMessageRoleAssistant,
				Content: reply,
			}), nil
		},
	}
}

// ANCHOR: build
// buildExecutive is the assistant over cal. withSummaryPreamble is the
// switch the example exists for: true is the program as it should be
// written, false is what every executive got before the field existed.
func buildExecutive(cal *Calendar, withSummaryPreamble bool) *arboreal.TodoListExecutive {
	reschedule := arboreal.CreateBehaviorTreeWithId(
		"reschedule_event",
		"Move an existing calendar event to a new time",
		"Move the weekly review to 15:00",
		"tree-reschedule",
	)
	extractMove := arboreal.LLMCompletionState(arboreal.LLMCompletionOptions{
		Id:   "state-extract-move",
		Name: "extract-move",
		System: "A scheduling request follows, possibly quoting the user. Extract the event " +
			"to move and its new time. Only respond in valid JSON with the following fields:\n" +
			"\tdata - an object with two string fields: event, the event's name as the user " +
			"called it, and time, the new time\n\n" +
			"Example:\n\n{\"data\": {\"event\": \"Weekly review\", \"time\": \"15:00\"}}",
		Annotation: "reschedule",
	})
	applyMove := applyState("state-apply-move", "reschedule", func(f map[string]string) string {
		return cal.Reschedule(f["event"], f["time"])
	})
	reschedule.AddTransition(&extractMove, &applyMove)

	cancel := arboreal.CreateBehaviorTreeWithId(
		"cancel_event",
		"Remove an existing calendar event",
		"Cancel the vendor call",
		"tree-cancel",
	)
	extractCancel := arboreal.LLMCompletionState(arboreal.LLMCompletionOptions{
		Id:   "state-extract-cancel",
		Name: "extract-cancel",
		System: "A scheduling request follows, possibly quoting the user. Extract the event " +
			"to cancel. Only respond in valid JSON with the following fields:\n" +
			"\tdata - an object with one string field: event, the event's name as the user called it\n\n" +
			"Example:\n\n{\"data\": {\"event\": \"Vendor call\"}}",
		Annotation: "cancel",
	})
	applyCancel := applyState("state-apply-cancel", "cancel", func(f map[string]string) string {
		return cal.Cancel(f["event"])
	})
	cancel.AddTransition(&extractCancel, &applyCancel)

	exec := arboreal.CreateTodoListExecutiveWithId(
		"Scheduler",
		"The scheduling assistant of a small team",
		"exec-scheduler",
		&reschedule, &cancel,
	)

	// The Preamble is the planner's: an identity sentence, then the
	// mechanics. What each behavior's direction must carry and the JSON
	// shape its step extracts from it, the quoted-direction format, the []
	// rule that is the only deliberate route to the OutOfBoundsHandler, and
	// the Re-plan guard. Every one of those is a planner instruction.
	exec.Preamble = "You are the scheduling assistant of a small team. " +
		"reschedule_event: use it when the user wants an event moved; its direction must name " +
		"the event and the new time, which the step extracts as {\"event\": ..., \"time\": ...}. " +
		"cancel_event: use it when the user wants an event removed; its direction must name " +
		"the event, which the step extracts as {\"event\": ...}. " +
		"When writing a step's direction, quote the user faithfully: The user wrote: \"...\". " +
		"If the message fits none of the behaviors, return an empty JSON array: []. " +
		"Never include a Re-plan step in the plan."

	// The SummaryPreamble is the reply's: the identity and the tone, nothing
	// about directions or JSON. Left unset, the Preamble above opens the
	// summarizer prompt instead, and the reply takes the only shape those
	// sentences describe.
	if withSummaryPreamble {
		exec.SummaryPreamble = "You are the scheduling assistant of a small team. " +
			"Reply to the user in prose, confirming each thing that was done."
	}

	exec.OutOfBoundsHandler = arboreal.CannedResponseState("I only move and cancel calendar events.")
	return exec
}

// ANCHOR_END: build

func main() {
	noSummary := flag.Bool("no-summary-preamble", false,
		"leave SummaryPreamble unset, so the planner's Preamble opens the summarizer prompt")
	flag.Parse()

	message := "Move the weekly review to 15:00 and cancel the vendor call."
	if flag.NArg() > 0 {
		message = strings.Join(flag.Args(), " ")
	}

	cal := NewCalendar(map[string]string{
		"Weekly review":   "14:00",
		"Vendor call":     "16:30",
		"Design critique": "11:00",
	})
	exec := buildExecutive(cal, !*noSummary)
	ctx := arboreal.WithDefaultModel(context.Background(), "openai:"+llm.ModelGPT4oMini)

	fmt.Println("Calendar before:")
	fmt.Print(cal)
	fmt.Printf("\nUser: %s\n", message)

	history := arboreal.AppendToMessages(nil, llm.ChatCompletionMessage{
		Role:    llm.ChatMessageRoleUser,
		Content: message,
	})
	result, sig := exec.Call(ctx, history)
	if es, ok := sig.(*arboreal.ErrorSignal); ok {
		log.Printf("the turn failed: %s", es.ErrorMessage)
		os.Exit(1)
	}

	fmt.Println("\nCalendar after:")
	fmt.Print(cal)
	fmt.Printf("\nReply:\n%s\n", result.LastMessage().Content)
}
