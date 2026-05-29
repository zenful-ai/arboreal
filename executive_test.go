package arboreal

import (
	"context"
	"errors"
	"testing"
)

// ctxCapturingBehavior is a Behavior that records the context it was called
// with, so a test can assert how the executive threads context down into the
// behavior tree. It returns a CollectUserInputSignal, which makes
// TodoListExecutive.Execute take its early-return path and avoids any LLM call.
type ctxCapturingBehavior struct {
	called      bool
	receivedCtx context.Context
}

func (b *ctxCapturingBehavior) Hash() string       { return "ctx-capturing-behavior" }
func (b *ctxCapturingBehavior) Name() string       { return "ctx-capturing-behavior" }
func (b *ctxCapturingBehavior) Description() string { return "records the context it is called with" }
func (b *ctxCapturingBehavior) Copy() Behavior      { return b }

func (b *ctxCapturingBehavior) Call(ctx context.Context, messages AnnotatedMessages) (AnnotatedMessages, Signal) {
	b.called = true
	b.receivedCtx = ctx
	return messages, &CollectUserInputSignal{}
}

// oneShotChannel delivers a single user message and then reports EOF, so
// RunLoop processes exactly one turn and then exits its loop.
type oneShotChannel struct {
	sent bool
}

func (c *oneShotChannel) AllocateID() string         { return "id" }
func (c *oneShotChannel) Send(*ChannelMessage) error { return nil }

func (c *oneShotChannel) Receive() (*ChannelMessage, error) {
	if c.sent {
		return nil, errors.New("eof")
	}
	c.sent = true
	return &ChannelMessage{Id: "1", Content: "Hello"}, nil
}

// TestRunLoopThreadsNonNilContext is the regression test for the nil-context
// panic. The RunLoop used to call e.Execute(nil, ...), so the behavior tree —
// and ultimately the LLM provider's select on ctx.Done() — received a nil
// context and crashed with a SIGSEGV on the very first user turn.
//
// We pre-seed the executive's plan with a behavior that captures the context it
// receives, drive a single turn through the real RunLoop, and assert the
// context that reached the behavior is non-nil. Pre-seeding the plan also means
// RunLoop skips the planning LLM call, and the behavior's CollectUserInputSignal
// makes Execute return before the summarizer LLM call — so the test is hermetic
// (no network required).
func TestRunLoopThreadsNonNilContext(t *testing.T) {
	behavior := &ctxCapturingBehavior{}

	exec := CreateTodoListExecutive("test exec", "exercises context threading", behavior)
	exec.plan = []*ExecGeneratedStep{
		{Behavior: behavior, Messages: AnnotatedMessages{}},
	}

	// RunLoop exits by returning the channel's terminating error.
	_ = exec.RunLoop(&oneShotChannel{})

	if !behavior.called {
		t.Fatal("behavior was never called; RunLoop did not reach the plan step")
	}
	if behavior.receivedCtx == nil {
		t.Fatal("behavior received a nil context: RunLoop is passing nil to Execute")
	}
}
