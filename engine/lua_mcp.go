package engine

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	lua "github.com/yuin/gopher-lua"
	"github.com/zenful-ai/arboreal"
)

var mcpNamespaceExport = LuaTypeExport{
	Name: "mcp",
	Methods: map[string]lua.LGFunction{
		"call_tool": arborealToolCall,
	},
}

func arborealToolCall(l *lua.LState) int {
	client, ok := l.Context().Value("arboreal_mcp_client").(*arboreal.MCPClientMux)
	if !ok {
		l.RaiseError("No MCP client found")
		return 0
	}

	toolName := l.CheckString(1)
	arguments := l.CheckTable(2)

	callToolArgs := LuaToGo(arguments)

	res, err := client.CallTool(l.Context(), &mcp.CallToolParams{
		Name:      toolName,
		Arguments: callToolArgs,
	})

	if err != nil {
		l.RaiseError("Error calling tool: %w", err)
		return 0
	}

	resultTable := l.NewTable()

	resultTable.RawSetString("error", lua.LBool(res.IsError))

	content := l.NewTable()

	for _, c := range res.Content {
		switch t := c.(type) {
		case *mcp.TextContent:
			content.Append(lua.LString(t.Text))
		default:
			l.RaiseError("unsupported content type: %w", err)
		}
	}

	resultTable.RawSetString("content", content)
	l.Push(resultTable)

	return 1
}
