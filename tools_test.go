package arboreal

import (
	"context"
	"strings"
	"testing"

	"github.com/zenful-ai/arboreal/llm"
)

func TestAllowedToolCall(t *testing.T) {
	cases := []struct {
		name  string
		tools []string
		call  string
		want  bool
	}{
		{"offered", []string{"alpha", "beta"}, "beta", true},
		{"not offered", []string{"alpha", "beta"}, "gamma", false},
		{"empty list allows everything", nil, "gamma", true},
		{"empty non-nil list allows everything", []string{}, "gamma", true},
		{"empty name is not offered", []string{"alpha", "beta"}, "", false},
		{"matching is exact", []string{"alpha", "beta"}, "Alpha", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := allowedToolCall(tc.tools, tc.call); got != tc.want {
				t.Fatalf("allowedToolCall(%v, %q) = %v, want %v", tc.tools, tc.call, got, tc.want)
			}
		})
	}
}

// callState builds an LLMCompletionState from options and calls it with a
// one-message history. It returns what the call returned. System is left
// empty by every caller so the state prepends nothing, which is what lets
// the tests assert the history came back unchanged.
func callState(t *testing.T, ctx context.Context, options LLMCompletionOptions) (AnnotatedMessages, AnnotatedMessages, Signal) {
	t.Helper()
	// An empty token proves no model was reached: the OpenAI provider reads
	// OPENAI_TOKEN at completion time and fails with its own message, which
	// none of the tool-block assertions below accept.
	t.Setenv("OPENAI_TOKEN", "")

	state := LLMCompletionState(options)
	history := AppendToMessages(nil, llm.ChatCompletionMessage{
		Role:    llm.ChatMessageRoleUser,
		Content: "hi",
	})
	got, sig := state.Call(ctx, history)
	return history, got, sig
}

// assertToolConfigError asserts the shape of every configuration error the
// tool block reports: a pointer *ErrorSignal (the book's ch. 6 rule), of
// type unrecoverable, whose message contains want, with the input history
// returned untouched.
func assertToolConfigError(t *testing.T, in, got AnnotatedMessages, sig Signal, want string) {
	t.Helper()
	errSig, ok := sig.(*ErrorSignal)
	if !ok {
		t.Fatalf("signal = %#v, want *ErrorSignal", sig)
	}
	if !strings.Contains(errSig.ErrorMessage, want) {
		t.Fatalf("ErrorMessage = %q, want it to contain %q", errSig.ErrorMessage, want)
	}
	if errSig.ErrorType != StateErrorTypeUnrecoverable {
		t.Fatalf("ErrorType = %q, want %q", errSig.ErrorType, StateErrorTypeUnrecoverable)
	}
	if len(got) != len(in) || got[len(got)-1].Content != in[len(in)-1].Content {
		t.Fatalf("history changed: got %d messages ending %q, want the %d given ending %q",
			len(got), got[len(got)-1].Content, len(in), in[len(in)-1].Content)
	}
}

func TestTools_WithoutAllowTools(t *testing.T) {
	in, got, sig := callState(t, context.Background(), LLMCompletionOptions{
		AllowTools: false,
		Tools:      []string{"alpha"},
	})
	assertToolConfigError(t, in, got, sig, "AllowTools is false")
}

func TestTools_NoMuxInContext(t *testing.T) {
	in, got, sig := callState(t, context.Background(), LLMCompletionOptions{
		AllowTools: true,
		Tools:      []string{"alpha"},
	})
	assertToolConfigError(t, in, got, sig, "no MCP client")
}

func TestTools_UnknownName(t *testing.T) {
	ctx := WithMCPClient(context.Background(), newThreeToolMux(t))
	in, got, sig := callState(t, ctx, LLMCompletionOptions{
		AllowTools: true,
		Tools:      []string{"alpha", "delta"},
	})
	assertToolConfigError(t, in, got, sig, `"delta"`)
}

// TestAllowTools_EmptyToolsNoMuxStaysSilent pins the backward-compatibility
// claim: AllowTools with an empty Tools and no mux is not a tool-block
// error. The call proceeds to the provider, which is where it fails here —
// on the empty token — so the error is the provider's, of the type the
// completion path has always used.
func TestAllowTools_EmptyToolsNoMuxStaysSilent(t *testing.T) {
	_, _, sig := callState(t, context.Background(), LLMCompletionOptions{
		AllowTools: true,
	})
	errSig, ok := sig.(*ErrorSignal)
	if !ok {
		t.Fatalf("signal = %#v, want *ErrorSignal from the provider", sig)
	}
	if errSig.ErrorMessage != "OPENAI_TOKEN environment variable not set" {
		t.Fatalf("ErrorMessage = %q, want the provider's token error", errSig.ErrorMessage)
	}
	if errSig.ErrorType != StateErrorTypeUnknown {
		t.Fatalf("ErrorType = %q, want %q (the completion path's type)", errSig.ErrorType, StateErrorTypeUnknown)
	}
}
