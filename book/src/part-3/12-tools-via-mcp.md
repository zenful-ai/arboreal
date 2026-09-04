# Tools via MCP

## Why this exists

Every state in Parts I and II produced text and only text. `LLMCompletionOptions` has carried the way out all along: `AllowTools`, the one field of that struct this book has not yet used, opts a state into letting the model call tools mid-completion. Arboreal has exactly one tool mechanism, the Model Context Protocol. There is no "register a Go function" path — no registry, no reflection over signatures; a Go function becomes a tool by being served over MCP. That sounds heavier than it is: the MCP SDK ships an in-memory transport, so the server that makes your function a tool can live in the same process as the agent, a goroutine away, with no network in between. The machinery splits in two — `mcp.go` holds the client side, a mux over any number of servers plus the context plumbing that delivers it, and `state.go` holds the loop a state with `AllowTools` runs against that mux. This chapter reads both, then runs them.

## The mux

`MCPClientMux` in `mcp.go` is one MCP client over many servers. Every `Add*` call connects a session, asks the server for its tool list once, and files each tool into two maps: name to definition, name to session. `Tools()` flattens the definitions into one `[]llm.ChatTool`, the shape a completion request wants; `CallTool` looks up the session by tool name and forwards the call, or errors on a name no server claimed; `Close` closes every session — `defer` it next to the `Add`. Two servers exporting the same tool name collide silently: the later registration wins both maps.

Three ways in. `AddInMemoryServer` takes a transport — the client half of the pair `mcp.NewInMemoryTransports` returns, with the server's `Run` holding the other half in a goroutine — and is how a function in your own process becomes a tool. `AddStreamableHTTPServer` takes a base URL and a `*StreamableHTTPOptions`, whose one field, `HTTPClient`, is the auth seam: `NewBearerHTTPClient(token)` builds a client that stamps `Authorization: Bearer` onto every request, `NewBearerTransport` layers the same onto an `*http.Client` you already have, and `nil` means `http.DefaultClient`. `AddSSEServer` takes just a URL; it has no options struct and therefore no auth seam yet — its own `TODO` says so.

Then delivery. The mux travels in the context: `WithMCPClient(ctx, mux)` stores it under an unexported typed key, and `MCPClientFromContext` gets it back. This is the exported, typed pair that Chapter 11's raw `"arboreal_trace"` string is not — the same context-value habit, finished.

## The `AllowTools` loop

`LLMCompletionState`'s lambda in `state.go`, after the system-prompt work Chapter 5 covered, does one new thing when `options.AllowTools` is set: if `MCPClientFromContext` finds a mux, the request gets `request.Tools = client.Tools()`. Both conditions must hold — no mux, or no `AllowTools`, and the request simply carries no tools; neither absence is an error.

Then the completion becomes a loop. Ask the model; if the reply carries no tool calls, break. Otherwise execute the first call through `CallTool`, turn the result's first content into a string — a `TextContent`'s text, anything else marshaled to JSON — and build a `function`-role message carrying that string, the tool's name, and the call's `Meta`. The assistant reply that made the call and the result message are then appended to both lists the loop maintains: the history the state will return, and `request.Messages`, so the next round's model sees what its tool said. Then ask again. The reply that finally has no tool calls breaks the loop and is appended once, after the loop, with the usual `Terminal` handling — from the outside, a state that ran three model rounds returns exactly like one that ran one.

Two limits, both read straight off the source. First, `ToolCalls[0]` is hardcoded: a reply that makes several calls gets exactly one executed, and the extras survive only as unexecuted entries inside the appended assistant message — no result, no error. (The default OpenAI provider still speaks the legacy single-function-call API and can surface only one call per reply anyway; providers that do parallel calls hit this limit for real.) Second, the history question. Directly above the loop's exit test sits `// FIXME: Tool calls don't currently make it into the conversation history` — and further down the same loop another comment says the opposite, "Keep tool call and result in history so callers can inspect executed tools.", and the two appends beneath it do exactly that. The `FIXME` is stale; the code beneath it moved on. The dump in the next section is the proof.

## Run it

`examples/tools` is the whole mechanism in one file: one tool, one question, one `Call` — no executive, no tree, a state driven directly as in Chapter 5. The tool is an ordinary function:

```go
{{#include ../../../examples/tools/main.go:server}}
```

The wiring builds the server, connects a mux to it over the in-memory transport, and puts the mux in the context:

```go
{{#include ../../../examples/tools/main.go:wiring}}
```

And the state asks a question only the tool can answer:

```go
{{#include ../../../examples/tools/main.go:call}}
```

Run it with `go run ./examples/tools`; it needs `OPENAI_TOKEN`. The timestamp is whatever now is; the rest is stable. One run printed:

```text
[0] system    "Answer using the tools available to you."
[1] user      "What is the exact current timestamp?"
[2] assistant ""
[3] function  "2026-09-02T00:08:30+02:00"
[4] assistant "The exact current timestamp is 2026-09-02T00:08:30+02:00."
```

Five messages, four roles. `[0]` is the system prompt the state prepended; `[1]` is the caller's question. `[2]` is the tool call, and it prints as an empty string because the dump prints `Content` with `%q` and a tool-call reply carries no content — the call lives in the message's `ToolCalls` field, name and arguments, which the dump cannot show. `[3]` is the `function`-role result: the string the MCP server returned, filed under the tool's name in the message's `Name` field, also unprinted. `[4]` is the reply that had no tool calls, appended after the loop. Two model rounds produced these five messages; attach Chapter 11's trace and you would still see a single entering/leaving pair for the state — the extra round per tool call happens inside the state's one envelope, and nothing on the wire marks it.

## Coming from LangGraph

`AllowTools` plus a mux in the context is `bind_tools` plus a `ToolNode`, with one structural difference: the execute-and-ask-again loop lives inside the state, not in the graph. There is no tools node to wire and no conditional edge routing tool-call replies back to the model; a state with `AllowTools` is the whole agent-with-tools cycle in one node, and the rest of the tree never knows a tool ran. The other habit to unlearn is conflating this with planning. In LangGraph the canonical agent chooses its next action by calling tools; Arboreal's planner does not — choosing a behavior (Chapter 8) is a prompt-and-annotation affair at the layer above, and calling a tool is a mid-completion round trip inside one state. Different mechanisms at different layers, and the planner never sees the mux.

## Sharp edges

```admonish warning title="Sharp edge"
Only the first tool call of each reply is executed; parallel tool calls are dropped without an error and never produce a result — at most they ride along, unexecuted, inside the appended assistant message. Prompt accordingly, or expose only tools that make sense one at a time.
```

```admonish warning title="Sharp edge"
No mux in the context means no tools — silently. `AllowTools: true` with a missing `WithMCPClient` is a state that simply never offers tools; nothing fails. When a model "refuses" to use a tool, check the context first.
```

```admonish warning title="Sharp edge"
Tools reach the model only through a state that sets `AllowTools`. The planner, annotation-mode states, and every default state ignore the mux entirely.
```

## Back to the trace

The tool loop lives inside Chapter 2's step 4 — the tree walks its states, and one state's `Call` may now hold several model requests instead of one. Nothing else in that chapter's sequence moves: step 2's planner never sees the mux, step 3's fan-out is unchanged, step 5 triages the same signals. On the wire, Chapter 11 showed step 4 as one begin/end pair per state; it still is. Tools change what a state does, not what a turn is.

```admonish example title="Recap"
- One mechanism: MCP. A mux merges servers; the context delivers it; `AllowTools` opts a state in.
- In-memory transport = tools with no network; HTTP/SSE for real servers, bearer on HTTP only.
- One tool call per round; a missing mux is silent; `Close` the mux.
- Tool calls and results do land in the history — the `state.go` FIXME is stale.
```
