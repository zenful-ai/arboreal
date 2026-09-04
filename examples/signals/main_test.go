package main

import (
	"reflect"
	"testing"

	"github.com/zenful-ai/arboreal"
)

// ANCHOR: predicates
func isNil(sig arboreal.Signal) bool { return sig == nil }

func isSkip(sig arboreal.Signal) bool {
	_, ok := sig.(*arboreal.SkipSignal)
	return ok
}

func isError(sig arboreal.Signal) bool {
	_, ok := sig.(*arboreal.ErrorSignal)
	return ok
}

func isPause(sig arboreal.Signal) bool {
	_, ok := sig.(*arboreal.CollectUserInputSignal)
	return ok
}

// ANCHOR_END: predicates

func TestVisitOrder(t *testing.T) {
	// ANCHOR: cases
	cases := []struct {
		name    string
		signals map[string]arboreal.Signal
		calls   int
		// want holds the visit order of each call, wantSig a predicate for
		// the signal that same call handed back to the caller.
		want    [][]string
		wantSig []func(arboreal.Signal) bool
	}{
		{
			name:    "nil everywhere: depth-first in insertion order",
			calls:   1,
			want:    [][]string{{"root", "a", "a1", "a2", "b", "b1"}},
			wantSig: []func(arboreal.Signal) bool{isNil},
		},
		{
			name:    "Skip prunes a's subtree but not its sibling, and is absorbed",
			signals: map[string]arboreal.Signal{"a": &arboreal.SkipSignal{}},
			calls:   1,
			want:    [][]string{{"root", "a", "b", "b1"}},
			wantSig: []func(arboreal.Signal) bool{isNil},
		},
		{
			name:    "Terminal stops the whole tree and is absorbed to nil",
			signals: map[string]arboreal.Signal{"a1": &arboreal.TerminalSignal{}},
			calls:   1,
			want:    [][]string{{"root", "a", "a1"}},
			wantSig: []func(arboreal.Signal) bool{isNil},
		},
		{
			name:    "Error aborts the tree and propagates to the caller",
			signals: map[string]arboreal.Signal{"a1": &arboreal.ErrorSignal{ErrorMessage: "boom"}},
			calls:   1,
			want:    [][]string{{"root", "a", "a1"}},
			wantSig: []func(arboreal.Signal) bool{isError},
		},
		{
			name:    "CollectUserInput pauses; the next call resumes a's children in reverse",
			signals: map[string]arboreal.Signal{"a": &arboreal.CollectUserInputSignal{}},
			calls:   2,
			want:    [][]string{{"root", "a"}, {"a2", "a1", "b", "b1"}},
			wantSig: []func(arboreal.Signal) bool{isPause, isNil},
		},
		{
			name:    "a trailing Skip leaks out as the tree's return value",
			signals: map[string]arboreal.Signal{"b1": &arboreal.SkipSignal{}},
			calls:   1,
			want:    [][]string{{"root", "a", "a1", "a2", "b", "b1"}},
			wantSig: []func(arboreal.Signal) bool{isSkip},
		},
	}
	// ANCHOR_END: cases

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			results := exercise(tc.signals, tc.calls)

			var got [][]string
			for _, r := range results {
				got = append(got, r.visited)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("visit order = %v, want %v", got, tc.want)
			}

			if len(tc.wantSig) != len(results) {
				t.Fatalf("test case declares %d signal expectations for %d calls", len(tc.wantSig), len(results))
			}
			for i, r := range results {
				if !tc.wantSig[i](r.sig) {
					t.Errorf("call %d returned %T, which this scenario does not expect", i+1, r.sig)
				}
			}
		})
	}
}
