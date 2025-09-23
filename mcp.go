package arboreal

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

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
	toolMap      map[string]*mcp.Tool
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
		tool := llm.ChatTool{
			Type:        llm.ChatToolTypeFunction,
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		}

		tools = append(tools, tool)
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

func (m *MCPClientMux) addSessionMetadata(ctx context.Context, session *mcp.ClientSession) error {
	res, err := session.ListTools(ctx, nil)
	if err != nil {
		return err
	}

	for _, tool := range res.Tools {
		m.toolSessions[tool.Name] = session
		m.toolMap[tool.Name] = tool
	}

	return nil
}

func (m *MCPClientMux) AddInMemoryServer(ctx context.Context, transport mcp.Transport) error {
	session, err := m.client.Connect(ctx, transport)
	if err != nil {
		return err
	}

	m.sessions = append(m.sessions, session)
	return m.addSessionMetadata(ctx, session)
}

func (m *MCPClientMux) AddSSEServer(ctx context.Context, baseURL string) error {
	transport := mcp.NewSSEClientTransport(baseURL, nil)

	session, err := m.client.Connect(ctx, transport)
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
	m.toolMap = make(map[string]*mcp.Tool)

	return &m
}
