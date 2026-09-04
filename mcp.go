package arboreal

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zenful-ai/arboreal/llm"
)

const (
	ProfileTypeEmulator = "emulator"
	ProfileTypeChat     = "chat"
	ProfileTypeEmbedded = "embedded"
)

type MCPProfile struct {
	Type    string      `json:"type"`
	Servers []MCPServer `json:"servers"`
}

func ProfilesForArtifact(artifact []byte) ([]MCPProfile, error) {
	r, err := zip.NewReader(bytes.NewReader(artifact), int64(len(artifact)))
	if err != nil {
		return nil, err
	}

	var profiles []MCPProfile
	for _, f := range r.File {
		if f.FileInfo().Name() == "profiles.json" {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()

			b, err := io.ReadAll(rc)
			if err != nil {
				return nil, err
			}

			err = json.Unmarshal(b, &profiles)
			if err != nil {
				return nil, err
			}
		}
	}

	return profiles, nil
}

const (
	MCPServerTypeSSE     = "sse"
	MCPServerTypeMemory  = "mem"
	MCPServerTypeCommand = "cmd"
)

type MCPServer struct {
	Type     string `json:"type"`
	Location string `json:"location"`
}

type MCPClientMux struct {
	client       *mcp.Client
	sessions     []*mcp.ClientSession
	toolSessions map[string]*mcp.ClientSession
	// toolMap holds each remote tool already converted to the llm package's
	// shape; the conversion (wire schema -> *jsonschema.Schema) happens once,
	// in addSessionMetadata, not on every Tools() call.
	toolMap map[string]llm.ChatTool
}

func (m *MCPClientMux) Close() error {
	var err error
	for _, session := range m.sessions {
		err = session.Close()
	}
	return err
}

func (m *MCPClientMux) Tools() []llm.ChatTool {
	var tools []llm.ChatTool

	for _, t := range m.toolMap {
		tools = append(tools, t)
	}

	return tools
}

func (m *MCPClientMux) CallTool(ctx context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error) {
	// First, find the correct session for the tool in question
	session, ok := m.toolSessions[params.Name]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", params.Name)
	}

	res, err := session.CallTool(ctx, params)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (m *MCPClientMux) AddProfilesOfType(t string, profiles []MCPProfile) error {
	var targetProfile MCPProfile
	for _, p := range profiles {
		if p.Type == t {
			targetProfile = p
			break
		}
	}

	for _, server := range targetProfile.Servers {
		switch server.Type {
		case "sse":
			err := m.AddSSEServer(context.Background(), server.Location)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// schemaFromTool converts a listed tool's input schema into the typed schema
// the llm package carries. Since go-sdk v1 the client side holds the default
// JSON decoding of the wire schema (normally a map[string]any; a boolean
// schema decodes to a bool), so the conversion is a JSON round trip, which is
// correct whatever the field holds; jsonschema.Schema's UnmarshalJSON handles
// the "type" keyword in both its string and array forms. A tool with no input
// schema yields nil, unchanged from v0.2.0, and the providers in llm are the
// ones that must cope with that (llm/anthropic.go currently does not).
func schemaFromTool(t *mcp.Tool) (*jsonschema.Schema, error) {
	if t.InputSchema == nil {
		return nil, nil
	}

	b, err := json.Marshal(t.InputSchema)
	if err != nil {
		return nil, fmt.Errorf("tool %q: marshal input schema: %w", t.Name, err)
	}

	var s jsonschema.Schema
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("tool %q: decode input schema: %w", t.Name, err)
	}

	return &s, nil
}

func (m *MCPClientMux) addSessionMetadata(ctx context.Context, session *mcp.ClientSession) error {
	res, err := session.ListTools(ctx, nil)
	if err != nil {
		return err
	}

	for _, tool := range res.Tools {
		schema, err := schemaFromTool(tool)
		if err != nil {
			return err
		}

		m.toolSessions[tool.Name] = session
		m.toolMap[tool.Name] = llm.ChatTool{
			Type:        llm.ChatToolTypeFunction,
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: schema,
		}
	}

	return nil
}

func (m *MCPClientMux) AddInMemoryServer(ctx context.Context, transport mcp.Transport) error {
	session, err := m.client.Connect(ctx, transport, nil)
	if err != nil {
		return err
	}

	m.sessions = append(m.sessions, session)
	return m.addSessionMetadata(ctx, session)
}

// StreamableHTTPOptions configures a Streamable HTTP MCP connection. It carries
// a single field today but is a struct so non-auth transport knobs can be added
// later without changing AddStreamableHTTPServer's signature.
type StreamableHTTPOptions struct {
	// HTTPClient is the client used for transport requests; its RoundTripper is
	// where authentication (and any other request customization) lives.
	// nil => http.DefaultClient. The *http.Client is the universal auth seam;
	// no auth scheme is privileged by this API.
	HTTPClient *http.Client
}

// AddStreamableHTTPServer connects to a remote MCP server over the Streamable
// HTTP transport and registers its tools on the mux. Pass opts.HTTPClient (e.g.
// the client from NewBearerHTTPClient(token)) to authenticate; nil opts / nil
// client uses http.DefaultClient.
func (m *MCPClientMux) AddStreamableHTTPServer(ctx context.Context, baseURL string, opts *StreamableHTTPOptions) error {
	var httpClient *http.Client
	if opts != nil {
		httpClient = opts.HTTPClient
	}

	transport := &mcp.StreamableClientTransport{
		Endpoint:   baseURL,
		HTTPClient: httpClient, // nil is valid; the SDK falls back to http.DefaultClient
	}

	session, err := m.client.Connect(ctx, transport, nil)
	if err != nil {
		return err
	}

	m.sessions = append(m.sessions, session)
	return m.addSessionMetadata(ctx, session)
}

// TODO: no auth seam — add an SSEOptions{HTTPClient} mirroring
// AddStreamableHTTPServer
func (m *MCPClientMux) AddSSEServer(ctx context.Context, baseURL string) error {
	transport := &mcp.SSEClientTransport{Endpoint: baseURL}

	session, err := m.client.Connect(ctx, transport, nil)
	if err != nil {
		return err
	}

	m.sessions = append(m.sessions, session)
	return m.addSessionMetadata(ctx, session)
}

func NewMCPClientMux() *MCPClientMux {
	var m MCPClientMux

	m.client = mcp.NewClient(&mcp.Implementation{
		Name:    "arboreal-client",
		Version: "v1.0.0",
	}, nil)
	m.toolSessions = make(map[string]*mcp.ClientSession)
	m.toolMap = make(map[string]llm.ChatTool)

	return &m
}

// contextKey is an unexported type for context keys defined in this package,
// preventing collisions with keys from other packages.
type contextKey string

const mcpClientContextKey contextKey = "arboreal_mcp_client"

// WithMCPClient returns a copy of ctx carrying the MCP client mux, ready to pass
// to RunLoop / Execute so the tool-calling state can reach it.
func WithMCPClient(ctx context.Context, mux *MCPClientMux) context.Context {
	return context.WithValue(ctx, mcpClientContextKey, mux)
}

// MCPClientFromContext returns the MCP client mux stored by WithMCPClient, if any.
func MCPClientFromContext(ctx context.Context) (*MCPClientMux, bool) {
	if ctx == nil {
		return nil, false
	}
	mux, ok := ctx.Value(mcpClientContextKey).(*MCPClientMux)
	return mux, ok
}

// bearerRoundTripper attaches "Authorization: Bearer <token>" to every request.
// base nil => http.DefaultTransport. The token is non-empty by construction:
// NewBearerTransport (and NewBearerHTTPClient, built on it) are the only ways to
// build one, and both reject an empty token.
type bearerRoundTripper struct {
	token string
	base  http.RoundTripper
}

func (b bearerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	base := b.base
	if base == nil {
		base = http.DefaultTransport
	}
	// Clone before mutating: a RoundTripper must not modify the input request.
	r := req.Clone(req.Context())
	r.Header.Set("Authorization", "Bearer "+b.token)
	return base.RoundTrip(r)
}

// NewBearerTransport wraps base in a RoundTripper that adds
// "Authorization: Bearer <token>" to every request, or returns an error if token
// is empty. base nil => http.DefaultTransport. Use it to layer bearer auth onto
// an existing *http.Client without discarding its current transport:
//
//	c.Transport, err = arboreal.NewBearerTransport(token, c.Transport)
func NewBearerTransport(token string, base http.RoundTripper) (http.RoundTripper, error) {
	if token == "" {
		return nil, errors.New("arboreal: bearer token must not be empty")
	}
	return bearerRoundTripper{token: token, base: base}, nil
}

// NewBearerHTTPClient returns a new *http.Client that adds a bearer token to
// every request, or an error if token is empty. Pass it as
// StreamableHTTPOptions.HTTPClient for bearer-authenticated MCP servers. Callers
// who already have a configured *http.Client should use NewBearerTransport to
// layer bearer auth onto it instead. Bearer has no special status in the connect
// API; other schemes supply their own *http.Client.
func NewBearerHTTPClient(token string) (*http.Client, error) {
	rt, err := NewBearerTransport(token, nil)
	if err != nil {
		return nil, err
	}
	return &http.Client{Transport: rt}, nil
}
