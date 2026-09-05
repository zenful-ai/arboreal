package arboreal

import (
	"bytes"
	"context"
	"log"
	"strings"
	"sync"
	"testing"

	"github.com/zenful-ai/arboreal/llm"
)

func TestDefaultModelContext(t *testing.T) {
	const uri = "anthropic:claude-sonnet-4-20250514"

	// Round-trip: a URI stored with WithDefaultModel is read back unchanged.
	ctx := WithDefaultModel(context.Background(), uri)
	if got, ok := DefaultModelFromContext(ctx); !ok || got != uri {
		t.Fatalf("DefaultModelFromContext after WithDefaultModel = (%q, %v), want (%q, true)", got, ok, uri)
	}

	// Absent: an empty context yields ("", false).
	if got, ok := DefaultModelFromContext(context.Background()); ok || got != "" {
		t.Fatalf("DefaultModelFromContext(empty) = (%q, %v), want (\"\", false)", got, ok)
	}

	// Nil context is tolerated, matching MCPClientFromContext. A nil-valued
	// variable rather than a literal nil keeps staticcheck SA1012 quiet without
	// a suppression directive; the interface value is nil either way.
	var nilCtx context.Context
	if got, ok := DefaultModelFromContext(nilCtx); ok || got != "" {
		t.Fatalf("DefaultModelFromContext(nil) = (%q, %v), want (\"\", false)", got, ok)
	}

	// The default-model key does not collide with the MCP client key: both
	// values coexist in one context.
	ctx = WithMCPClient(ctx, NewMCPClientMux())
	if got, ok := DefaultModelFromContext(ctx); !ok || got != uri {
		t.Fatalf("DefaultModelFromContext after WithMCPClient = (%q, %v), want (%q, true)", got, ok, uri)
	}
}

func TestResolveModelURI(t *testing.T) {
	cases := []struct {
		name         string
		explicit     string
		ctxDefault   string
		wantURI      string
		wantFallback bool
	}{
		{
			name:       "explicit wins over context default",
			explicit:   "anthropic:explicit",
			ctxDefault: "anthropic:ctx",
			wantURI:    "anthropic:explicit",
		},
		{
			name:     "explicit alone",
			explicit: "anthropic:explicit",
			wantURI:  "anthropic:explicit",
		},
		{
			name:       "context default when explicit is empty",
			ctxDefault: "anthropic:ctx",
			wantURI:    "anthropic:ctx",
		},
		{
			name:         "fallback when both are empty",
			wantURI:      fallbackModelURI,
			wantFallback: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotURI, gotFallback := resolveModelURI(tc.explicit, tc.ctxDefault)
			if gotURI != tc.wantURI || gotFallback != tc.wantFallback {
				t.Fatalf("resolveModelURI(%q, %q) = (%q, %v), want (%q, %v)",
					tc.explicit, tc.ctxDefault, gotURI, gotFallback, tc.wantURI, tc.wantFallback)
			}
		})
	}
}

func TestFallbackModelURIIsTheHistoricalDefault(t *testing.T) {
	// This change must not alter what unconfigured code gets. The release that
	// removes the fallback deletes this constant and this test with it.
	if fallbackModelURI != llm.GPT4oMini {
		t.Fatalf("fallbackModelURI = %q, want llm.GPT4oMini (%q)", fallbackModelURI, llm.GPT4oMini)
	}
}

// captureLog redirects the standard logger into a buffer for the duration of
// the test and resets the once-only fallback warning so the test observes a
// fresh first emission regardless of what ran before it.
//
// Not safe for use with t.Parallel: it mutates process-global state (the
// standard logger and fallbackWarning). The returned buffer is written under
// the logger's mutex but read without one; join any goroutines that may log
// before reading it.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prevOut := log.Writer()
	prevFlags := log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	fallbackWarning = sync.Once{}
	t.Cleanup(func() {
		log.SetOutput(prevOut)
		log.SetFlags(prevFlags)
		fallbackWarning = sync.Once{}
	})
	return &buf
}

func TestModelURIFor(t *testing.T) {
	t.Run("explicit wins and does not warn", func(t *testing.T) {
		buf := captureLog(t)
		ctx := WithDefaultModel(context.Background(), "anthropic:ctx")
		if got := modelURIFor(ctx, "anthropic:explicit"); got != "anthropic:explicit" {
			t.Fatalf("modelURIFor = %q, want %q", got, "anthropic:explicit")
		}
		if buf.Len() != 0 {
			t.Fatalf("unexpected log output: %q", buf.String())
		}
	})

	t.Run("context default is used and does not warn", func(t *testing.T) {
		buf := captureLog(t)
		ctx := WithDefaultModel(context.Background(), "anthropic:ctx")
		if got := modelURIFor(ctx, ""); got != "anthropic:ctx" {
			t.Fatalf("modelURIFor = %q, want %q", got, "anthropic:ctx")
		}
		if buf.Len() != 0 {
			t.Fatalf("unexpected log output: %q", buf.String())
		}
	})

	t.Run("fallback warns exactly once per process", func(t *testing.T) {
		buf := captureLog(t)
		ctx := context.Background()

		if got := modelURIFor(ctx, ""); got != fallbackModelURI {
			t.Fatalf("modelURIFor = %q, want fallback %q", got, fallbackModelURI)
		}
		if got := modelURIFor(ctx, ""); got != fallbackModelURI {
			t.Fatalf("second modelURIFor = %q, want fallback %q", got, fallbackModelURI)
		}

		out := buf.String()
		const marker = "arboreal: no default model configured"
		if n := strings.Count(out, marker); n != 1 {
			t.Fatalf("fallback warning emitted %d times, want 1; output:\n%s", n, out)
		}
		for _, want := range []string{fallbackModelURI, "WithDefaultModel"} {
			if !strings.Contains(out, want) {
				t.Fatalf("warning should mention %q; got:\n%s", want, out)
			}
		}
	})

	t.Run("empty context default is treated as unset", func(t *testing.T) {
		buf := captureLog(t)
		ctx := WithDefaultModel(context.Background(), "")
		if got := modelURIFor(ctx, ""); got != fallbackModelURI {
			t.Fatalf("modelURIFor = %q, want fallback %q", got, fallbackModelURI)
		}
		if !strings.Contains(buf.String(), "no default model configured") {
			t.Fatalf("empty context default should warn; got %q", buf.String())
		}
	})
}
