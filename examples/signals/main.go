// Package main is a learning-purposes example, NOT a template for real apps.
//
// It shows how the five signals steer BehaviorTree.Call, using print-only Go
// states — no LLM, no API token. Every scenario builds the same small tree:
//
//	root → a → a1
//	         → a2
//	     → b → b1
//
// and tells exactly one state to return a particular signal. The program
// prints the order in which states ran and the signal the tree handed back
// to its caller. Run it with:
//
//	$ go run ./examples/signals
//
// Things to notice in the output:
//
//   - With no signals at all the walk is depth-first in insertion order:
//     root a a1 a2 b b1.
//   - SkipSignal prunes the subtree below the state that returned it; the
//     state's siblings still run.
//   - TerminalSignal stops the whole tree, and the caller sees nil — the
//     tree absorbs it.
//   - ErrorSignal stops the tree and is returned to the caller.
//   - CollectUserInputSignal stops this call but keeps the tree's stack, so
//     the next call resumes with the paused state's children. Watch the
//     order: a2 runs before a1, because the pause path pushes children in
//     the opposite order from the normal path.
//   - A SkipSignal from the very last state visited leaks out as the tree's
//     return value.
package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/zenful-ai/arboreal"
	"github.com/zenful-ai/arboreal/llm"
)

// ANCHOR: recorder
// recorder hands out states that do nothing except append their name to
// visited and return the signal they were given.
type recorder struct {
	visited []string
}

func (r *recorder) state(name string, sig arboreal.Signal) *arboreal.BehaviorState {
	return &arboreal.BehaviorState{
		// The tree keys its visited-set by Hash(), so every state needs a
		// distinct id. The name is a fine id here.
		HashId:    name,
		StateName: name,
		Lambda: func(ctx context.Context, history arboreal.AnnotatedMessages) (arboreal.AnnotatedMessages, arboreal.Signal) {
			r.visited = append(r.visited, name)
			return history, sig
		},
	}
}

// ANCHOR_END: recorder

// ANCHOR: build
// build wires the tree. signals maps a state name to the signal that state
// should return; states not in the map return nil ("done, carry on").
func build(r *recorder, signals map[string]arboreal.Signal) *arboreal.BehaviorTree {
	tree := arboreal.CreateBehaviorTree("signals", "Demonstrates how signals steer traversal", "")

	root := r.state("root", signals["root"])
	a := r.state("a", signals["a"])
	a1 := r.state("a1", signals["a1"])
	a2 := r.state("a2", signals["a2"])
	b := r.state("b", signals["b"])
	b1 := r.state("b1", signals["b1"])

	// The first state wired in becomes the entry point, and among a state's
	// children, insertion order is priority: a runs before b, a1 before a2.
	tree.AddTransition(root, a)
	tree.AddTransition(root, b)
	tree.AddTransition(a, a1)
	tree.AddTransition(a, a2)
	tree.AddTransition(b, b1)

	return &tree
}

// ANCHOR_END: build

// ANCHOR: exercise
type callResult struct {
	visited []string
	sig     arboreal.Signal
}

// exercise builds a fresh tree and calls it `calls` times, threading the
// history returned by each call into the next, and recording each call's
// visit order and returned signal.
func exercise(signals map[string]arboreal.Signal, calls int) []callResult {
	r := &recorder{}
	tree := build(r, signals)

	// A state must never see an empty history (BehaviorState.Call reads the
	// last message after the lambda runs), so seed it with one user message.
	history := arboreal.AppendToMessages(nil, llm.ChatCompletionMessage{
		Role:    llm.ChatMessageRoleUser,
		Content: "go",
	})

	var results []callResult
	for i := 0; i < calls; i++ {
		r.visited = nil
		var sig arboreal.Signal
		history, sig = tree.Call(context.Background(), history)
		// r.visited is reset to nil above, so each call appends into a
		// fresh slice and storing it directly needs no defensive copy.
		results = append(results, callResult{
			visited: r.visited,
			sig:     sig,
		})
	}
	return results
}

// ANCHOR_END: exercise

func describe(sig arboreal.Signal) string {
	switch s := sig.(type) {
	case nil:
		return "nil"
	case *arboreal.SkipSignal:
		return "SkipSignal"
	case *arboreal.TerminalSignal:
		return "TerminalSignal"
	case *arboreal.CollectUserInputSignal:
		return "CollectUserInputSignal(" + s.Reason + ")"
	case *arboreal.ErrorSignal:
		return "ErrorSignal(" + s.ErrorMessage + ")"
	default:
		return fmt.Sprintf("%T", sig)
	}
}

func run(title string, signals map[string]arboreal.Signal, calls int) {
	fmt.Printf("== %s\n", title)
	for i, r := range exercise(signals, calls) {
		// 18 is the width of the longest visit string, "root a a1 a2 b b1",
		// so the "returned" column lines up across every scenario.
		fmt.Printf("call %d: visited %-18s returned %s\n", i+1, strings.Join(r.visited, " "), describe(r.sig))
	}
	fmt.Println()
}

// ANCHOR: scenarios
func main() {
	run("nil everywhere: depth-first, insertion order", nil, 1)

	run("a returns Skip: a's subtree is pruned, its sibling b still runs",
		map[string]arboreal.Signal{"a": &arboreal.SkipSignal{}}, 1)

	run("a1 returns Terminal: the whole tree stops, the caller sees nil",
		map[string]arboreal.Signal{"a1": &arboreal.TerminalSignal{}}, 1)

	run("a1 returns Error: the tree aborts and the error propagates",
		map[string]arboreal.Signal{"a1": &arboreal.ErrorSignal{
			ErrorMessage: "boom",
			ErrorType:    arboreal.StateErrorTypeUnrecoverable,
		}}, 1)

	run("a returns CollectUserInput: pause now, resume on the next call",
		map[string]arboreal.Signal{"a": &arboreal.CollectUserInputSignal{Reason: "need input"}}, 2)

	run("b1 (the last state) returns Skip: the Skip leaks out of the tree",
		map[string]arboreal.Signal{"b1": &arboreal.SkipSignal{}}, 1)
}

// ANCHOR_END: scenarios
