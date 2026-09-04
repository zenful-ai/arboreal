package arboreal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zenful-ai/arboreal/llm"
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
		func(ctx context.Context, req *mcp.CallToolRequest, _ map[string]any) (*mcp.CallToolResult, any, error) {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "hi"}}}, nil, nil
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

// titleQuery is the typed input of the tool used by
// TestTools_ConvertsInputSchema. The SDK infers its JSON schema from the
// struct: an object with one string property, "title".
type titleQuery struct {
	Title string `json:"title"`
}

// TestTools_ConvertsInputSchema covers the one behavior the v1 SDK forces on
// the mux: a tool's input schema arrives from the client session as a plain
// map[string]any and must reach llm.ChatTool as a typed *jsonschema.Schema.
func TestTools_ConvertsInputSchema(t *testing.T) {
	// The server goroutine is joined by the first (hence last-run) defer,
	// after mux.Close and cancel have ended the session, so the test cannot
	// leak it or hang on it.
	done := make(chan struct{})
	defer func() { <-done }()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := mcp.NewServer(&mcp.Implementation{Name: "schema-server", Version: "v1.0.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "lookup", Description: "look up a title"},
		func(ctx context.Context, req *mcp.CallToolRequest, q titleQuery) (*mcp.CallToolResult, any, error) {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: q.Title}}}, nil, nil
		})

	serverSide, clientSide := mcp.NewInMemoryTransports()
	go func() {
		defer close(done)
		_ = server.Run(ctx, serverSide)
	}()

	mux := NewMCPClientMux()
	defer mux.Close()
	if err := mux.AddInMemoryServer(ctx, clientSide); err != nil {
		t.Fatalf("AddInMemoryServer returned error: %v", err)
	}

	tools := mux.Tools()
	if len(tools) != 1 {
		t.Fatalf("got %d tools, want 1: %+v", len(tools), tools)
	}
	tool := tools[0]
	if tool.Name != "lookup" || tool.Type != llm.ChatToolTypeFunction {
		t.Fatalf("got tool %q of type %q, want %q of type %q", tool.Name, tool.Type, "lookup", llm.ChatToolTypeFunction)
	}
	if tool.Description != "look up a title" {
		t.Fatalf("got Description %q, want %q", tool.Description, "look up a title")
	}
	if tool.InputSchema == nil {
		t.Fatal("InputSchema is nil; the wire schema was not converted")
	}
	if tool.InputSchema.Type != "object" {
		t.Fatalf("InputSchema.Type = %q, want %q", tool.InputSchema.Type, "object")
	}
	prop, ok := tool.InputSchema.Properties["title"]
	if !ok {
		t.Fatalf("InputSchema.Properties has no %q entry: %+v", "title", tool.InputSchema.Properties)
	}
	if prop.Type != "string" {
		t.Fatalf("Properties[%q].Type = %q, want %q", "title", prop.Type, "string")
	}

	// The outgoing direction: llm/openai.go hands this schema to the provider
	// as FunctionDefinition.Parameters, so the converted value must serialize
	// as a structurally valid object schema (Schema.MarshalJSON validates it).
	out, err := json.Marshal(tool.InputSchema)
	if err != nil {
		t.Fatalf("marshal InputSchema: %v", err)
	}
	var wire map[string]any
	if err := json.Unmarshal(out, &wire); err != nil {
		t.Fatalf("marshaled InputSchema %s is not a JSON object: %v", out, err)
	}
	if wire["type"] != "object" {
		t.Fatalf("marshaled InputSchema %s has type %v, want %q", out, wire["type"], "object")
	}
	props, _ := wire["properties"].(map[string]any)
	if _, ok := props["title"]; !ok {
		t.Fatalf("marshaled InputSchema %s has no %q property", out, "title")
	}
}
