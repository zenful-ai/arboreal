package main

import (
	"context"
	"testing"

	"github.com/zenful-ai/arboreal"
	"github.com/zenful-ai/arboreal/llm"
)

// ANCHOR: roundtrip
// TestSnapshotRoundTrip seeds a plan through Restore, runs one hermetic
// turn, snapshots, restores into a second executive, and runs the same
// turn there: same reply, no model anywhere.
// The pause sits at a leaf, so the step's nested tree snapshot is empty and
// tree-position rehydration is a no-op here: the round trip proves the
// history/plan/messages layer survives a snapshot, not a mid-walk cursor.
func TestSnapshotRoundTrip(t *testing.T) {
	ctx := context.Background()

	first := build("exec-edges")
	if err := craftSnapshot("exec-edges", "tree-greeter").Restore(first); err != nil {
		t.Fatal(err)
	}

	h := arboreal.AppendToMessages(first.History, llm.ChatCompletionMessage{
		Role: llm.ChatMessageRoleUser, Content: "Anyone home?",
	})
	h, _ = first.Call(ctx, h)

	snap, err := arboreal.TakeSnapshot(first)
	if err != nil {
		t.Fatal(err)
	}
	if len(snap) == 0 {
		t.Fatal("expected the paused executive to appear in the snapshot")
	}

	second := build("exec-edges")
	if err := snap.Restore(second); err != nil {
		t.Fatal(err)
	}

	h2 := arboreal.AppendToMessages(second.History, llm.ChatCompletionMessage{
		Role: llm.ChatMessageRoleUser, Content: "Still there?",
	})
	h2, _ = second.Call(ctx, h2)

	want := "Hello! I am a canned reply."
	if got := h2.LastMessage().Content; got != want {
		t.Fatalf("restored executive replied %q, want %q", got, want)
	}
}

// ANCHOR_END: roundtrip
