package arboreal

import (
	"context"
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
