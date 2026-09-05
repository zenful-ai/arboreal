package arboreal

import (
	"context"
	"testing"
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
