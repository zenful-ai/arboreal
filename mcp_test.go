package arboreal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// recordingRoundTripper is a base transport that records that it ran and the
// Authorization header it received, then delegates to http.DefaultTransport so
// the real request still completes. It proves NewBearerTransport composes onto
// an existing transport rather than replacing it.
type recordingRoundTripper struct {
	called  bool
	gotAuth string
}

func (rt *recordingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.called = true
	rt.gotAuth = req.Header.Get("Authorization")
	return http.DefaultTransport.RoundTrip(req)
}

// TestNewBearerTransport_WrapsBaseAndAddsHeader covers the one non-obvious
// behavior of this code: NewBearerTransport layers onto an existing transport
// rather than replacing it.
func TestNewBearerTransport_WrapsBaseAndAddsHeader(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Echo-Authorization", r.Header.Get("Authorization"))
	}))
	defer ts.Close()

	spy := &recordingRoundTripper{}

	rt, err := NewBearerTransport("secret-token", spy)
	if err != nil {
		t.Fatalf("NewBearerTransport returned error: %v", err)
	}

	resp, err := (&http.Client{Transport: rt}).Get(ts.URL)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// The wrapped base transport ran (composition, not replacement)...
	if !spy.called {
		t.Fatal("base transport was not called; NewBearerTransport did not delegate to it")
	}
	// ...and it saw the request with the bearer header already attached.
	if spy.gotAuth != "Bearer secret-token" {
		t.Fatalf("base transport saw Authorization %q, want %q", spy.gotAuth, "Bearer secret-token")
	}
	// And the header reached the server end to end.
	if got := resp.Header.Get("X-Echo-Authorization"); got != "Bearer secret-token" {
		t.Fatalf("server saw Authorization %q, want %q", got, "Bearer secret-token")
	}
}

func TestAddStreamableHTTPServer_RegistersToolsAndSendsBearer(t *testing.T) {
	ctx := context.Background()

	// In-process MCP server exposing one tool.
	server := mcp.NewServer(&mcp.Implementation{Name: "test-server", Version: "v1.0.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "greet", Description: "say hi"},
		func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[map[string]any]) (*mcp.CallToolResultFor[any], error) {
			return &mcp.CallToolResultFor[any]{Content: []mcp.Content{&mcp.TextContent{Text: "hi"}}}, nil
		})

	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil)

	// Capture the Authorization header the server sees. Guarded by a mutex
	// because the streamable transport may hit the server from a background
	// goroutine (run the suite with -race to confirm safety).
	var (
		mu      sync.Mutex
		gotAuth string
	)
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		if a := r.Header.Get("Authorization"); a != "" {
			gotAuth = a
		}
		mu.Unlock()
		handler.ServeHTTP(w, r)
	}))
	defer httpServer.Close()

	mux := NewMCPClientMux()
	defer mux.Close()

	bearerClient, err := NewBearerHTTPClient("secret-token")
	if err != nil {
		t.Fatalf("NewBearerHTTPClient returned error: %v", err)
	}

	err = mux.AddStreamableHTTPServer(ctx, httpServer.URL, &StreamableHTTPOptions{
		HTTPClient: bearerClient,
	})
	if err != nil {
		t.Fatalf("AddStreamableHTTPServer returned error: %v", err)
	}

	// The remote tool is registered on the mux.
	var found bool
	for _, tool := range mux.Tools() {
		if tool.Name == "greet" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected tool %q to be registered; got %v", "greet", mux.Tools())
	}

	// The bearer token reached the server.
	mu.Lock()
	defer mu.Unlock()
	if gotAuth != "Bearer secret-token" {
		t.Fatalf("server received Authorization %q, want %q", gotAuth, "Bearer secret-token")
	}
}

func TestMCPClientContext(t *testing.T) {
	mux := NewMCPClientMux()

	// Round-trip: a mux stored with WithMCPClient is read back as the same mux.
	ctx := WithMCPClient(context.Background(), mux)
	if got, ok := MCPClientFromContext(ctx); !ok || got != mux {
		t.Fatalf("MCPClientFromContext after WithMCPClient = (%p, %v), want (%p, true)", got, ok, mux)
	}

	// Absent: an empty context yields (nil, false).
	if got, ok := MCPClientFromContext(context.Background()); ok || got != nil {
		t.Fatalf("MCPClientFromContext(empty) = (%v, %v), want (nil, false)", got, ok)
	}
}
