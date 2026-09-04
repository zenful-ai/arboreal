# Design: *The Arboreal Book* — Parts I and II

**Date:** 2026-08-31
**Status:** approved in brainstorming, ready for implementation planning
**Scope of this spec:** front matter, Part I (top-down), Part II (bottom-up).
Parts III and IV are deferred; their outline is kept in the appendix so it is not lost.

## Goal

An engineering book, built with mdbook, that teaches engineers how to write
agentic workflows with Arboreal. The reader has already built agents with
another framework (LangGraph is the reference point) and has never seen
Arboreal. After Parts I and II they can design a behavior tree, drive it with
the executive, and predict what one user turn does at every layer of the stack.

The book documents the framework **as it is at the current commit**, sharp
edges included. Every pitfall gets a callout with a workaround; nothing is
described as working when it does not.

## Audience and assumptions

- Comfortable with Go (reads goroutines, interfaces, `context.Context` without
  explanation). Go-language asides are not part of the book.
- Has shipped an agent with LangGraph or similar; knows what a graph of nodes,
  shared state, interrupts, checkpointers, tool calling, and a ReAct loop are.
- Has an `OPENAI_TOKEN`. The default model everywhere is `gpt-4o-mini` via
  OpenAI, so that is the only credential the book requires.

## Pedagogical structure: the spiral

Neither pure top-down nor pure bottom-up works for this framework: the run
loop lives on the executive (top of the stack), so hello-world needs the top,
but explaining the executive honestly needs signals and pause/resume (bottom
of the stack). The book therefore makes two passes:

1. **Part I goes top-down at reading level.** Run the quickstart, then trace a
   single user turn through every layer, *naming* each piece without
   explaining its internals. That trace (Chapter 2) is the map for the rest of
   the book.
2. **Part II rebuilds from the bottom.** Messages → states → signals → trees →
   executive. Each chapter closes with a short "Back to the trace" section
   pointing at the exact lines of the Chapter 2 trace it just explained. The
   two passes meet in Chapter 8, which ends by re-reading the whole trace with
   nothing left unexplained.

## Chapter plan

Every chapter has the same shape:

1. **Why this exists** — the problem the piece solves, one or two paragraphs.
2. **Run it** — a runnable example, included from `examples/`, with expected
   output.
3. **How it works** — the mechanism, referencing identifiers and files
   (`BehaviorTree.Call` in `behavior.go`), never line numbers.
4. **Coming from LangGraph** — a short sidebar mapping the concept.
5. **Sharp edges** — verified pitfalls with workarounds. Omitted only if a
   chapter genuinely has none.
6. **Back to the trace** (Part II only) — which step of the Chapter 2 trace
   this chapter explained.
7. **Recap** — three to five bullets.

### Front matter

**Introduction.** What Arboreal is in one paragraph (Go framework; behavior
trees as directed graphs of LLM/Go states; a plan-and-execute executive on
top; snapshots for pause/resume). Who the book is for. What you need (Go
1.23+, `OPENAI_TOKEN`, `go get github.com/zenful-ai/arboreal`). How the book
is organized and why (the spiral). A note that the book describes the current
commit and marks pitfalls with **Sharp edge** callouts.

### Part I — The view from the top

**Chapter 1 — Quick start.** Run `examples/quickstart` and have a
conversation. Then walk the ~25 lines and *name* every piece: `BehaviorTree`,
`LLMCompletionState`, `PauseState`, `AddTransition`, `TodoListExecutive`,
`RunLoop`, `TerminalChannel`. No internals. Must cover the two things that
stop people on minute one: input is submitted with a lone `$` line (or
Ctrl-]), and `OPENAI_TOKEN` must be set. Sharp edge: the executive is required
even for one behavior — it is the run loop, not just the planner.

**Chapter 2 — Anatomy of one turn.** The spine of the book. A sequence
diagram and a numbered narrative of one user message through the quickstart:

1. `RunLoop` blocks in `Channel.Receive`; the message is appended to the
   executive's `History`.
2. `plan` is empty, so `Plan` runs: an LLM call turns the message into a JSON
   todo list of behavior *names* plus a free-text **direction** per step. Each
   step gets a `Copy()` of its behavior and its own small message list whose
   first message is the direction.
3. `Execute` fans the steps out, one goroutine each, calling
   `Behavior.Call(ctx, step.Messages)`.
4. Inside the step, `BehaviorTree.Call` walks the graph: the chat state calls
   the model and appends the reply; the pause state returns
   `CollectUserInputSignal`, so the tree exits and the signal propagates up.
5. `Execute` sees the pause, keeps that step in `plan`, and — because there is
   exactly one step — uses its last message as the turn's output.
6. `RunLoop` appends the output to `History` and calls `Channel.Send`.
7. **Next turn:** `plan` is non-empty, so `Plan` is skipped; the new user
   message is appended to each pending step's messages and `Execute` runs
   again. The tree's stack is empty (the pause was a leaf), so the tree
   restarts from the chat state — on the step's ever-growing conversation.

Introduces the vocabulary used everywhere after: *turn, plan, step,
direction, signal, history, output, pause/resume*. Two facts get their own
callouts because they surprise every LangGraph engineer: (a) **the behavior
sees the planner's direction, not the user's words** — and the `Preamble`
trick from `examples/snapshot-simple` that keeps them faithful; (b) **there
are two conversations** — the executive's `History` (the transcript the user
sees) and each step's own `Messages` (what the model actually sees).

**Chapter 3 — If you come from LangGraph.** A Rosetta stone table and the
three mental shifts. Table rows: `StateGraph` ↔ `BehaviorTree`; node function
↔ `BehaviorState.Lambda`; typed `State` ↔ `AnnotatedMessages` + annotations;
`add_edge` ↔ `AddTransition`; `add_conditional_edges` ↔ *no equivalent* —
static edges plus `SkipSignal`/`TerminalSignal` from inside the node;
`interrupt()` ↔ `PauseState`/`CollectUserInputSignal`; checkpointer +
`thread_id` ↔ `TakeSnapshot`/`Restore` (with the stable-id requirement named
but detailed in Part III); `@tool`/`ToolNode` ↔ MCP tools only (Part III);
`create_react_agent` ↔ `TodoListExecutive`, which is plan-and-execute, not
ReAct; `Command`/`Send` ↔ none; streaming ↔ none. The three shifts: control
flow lives in the leaf; the agent layer is plan-then-execute with concurrent
steps; the tree is a stateful struct (not goroutine-safe, `Copy()` before
concurrent use).

### Part II — Building blocks, from the ground up

**Chapter 4 — Messages and annotations.** `AnnotatedMessage` embeds
`llm.ChatCompletionMessage` and adds a named `Annotations` map; the message
list is the blackboard. `GetAnnotation` searches newest-to-oldest (last write
wins) and falls back to meta-annotations `$last_message`, `$date`,
`$date_llm`. `AnnotationTemplate` syntax: `{{ name }}`, the conditional
multi-word form `{{ Label: name? }}`, and the `??` escape. `ExtraContext`
injects annotations into a system prompt. `AppendToMessages` vs mutating
`LastMessage().Annotations` in place. Example: `examples/annotations-probe`
(dumps the history before and after two annotating states). Sharp edges:
`Annotation.Data` is `any` and round-trips through JSON lossily (numbers
become `float64`); `__trace_annotations` is an internal key that can appear
in the map; the `__trace_annotations` breadcrumb is scrubbed by
`BehaviorState.Call`, not by you.

**Chapter 5 — Behaviors and states.** The `Behavior` interface as the one
contract (`Call(ctx, history) → (history, signal)`; everything is a
behavior, so composition nests). `BehaviorState`: one struct, pluggable
`Lambda`; `Call` is a uniform tracing envelope around it. The four factories:
`CannedResponseState`, `PauseState`, `BehaviorState{Lambda: …}` for arbitrary
Go, and `LLMCompletionState` in depth — templated `System`, model URIs
(`openai:…`, `anthropic:…`, default `gpt-4o-mini`), `Annotation` mode
(`evalIntoAnnotation`: sends *only* the system prompt and the last user
message, pins the parsed `{"data": …}` onto that user message, appends no
reply), `Terminal`, and `AllowTools` (named, deferred to Part III). Hash
identity: `HashId`, `Id`, and why two states must never share a hash. Example
(new): `examples/state-direct` — build one `LLMCompletionState` with a
templated system prompt (`{{ $date_llm }}`) and call it directly on a
hand-built history; no tree, no executive. Sharp edges: copying a state
struct copies its hash (the `examples/crm` `HashId` regeneration idiom);
`Annotation` mode stores the raw JSON string verbatim when the model returns
`{"data": null}`; the model default silently requires OpenAI.

**Chapter 6 — Signals.** The five outcomes of a `Call`: `nil`, `SkipSignal`,
`TerminalSignal`, `CollectUserInputSignal`, `ErrorSignal` (with
`retryable`/`unrecoverable`/`unknown`/`lua_syntax` types; also a Go `error`).
The propagation table: what each does to the traversal stack, whether it
reaches the tree's caller, and what the caller receives (`Terminal` is
absorbed to `nil`; `CollectUserInput` propagates unchanged; `Error` bubbles
and aborts). Example (new): `examples/signals` — a tree of print-only Go
states, no LLM, no token needed; each run demonstrates one signal's effect on
the visit order. Sharp edges: **signals must be returned as pointers** — every
type switch in the framework (`BehaviorTree.Call`, `Execute`, `TraceForSignal`
in `trace.go`) matches `*ErrorSignal`, `*SkipSignal`, etc., so a value such as
`ErrorSignal{…}` or `SkipSignal{…}` (the form `examples/crm` returns) is not
recognized by the traversal and makes `TraceForSignal` panic with
`unknown Signal type`; a trailing `SkipSignal` can leak out as the tree's
return value.

**Chapter 7 — Behavior trees.** Opens with a one-page classical-BT primer
(leaves, Sequence, Selector, tick, blackboard — condensed from the deck's
Part 1) so the name stops misleading, then the six departures (from the
deck's Part 2): graph not tree; no composites; five signals not three
statuses; once per turn not per frame; leaves are LLM/Go; the blackboard is
the message list and per-node state lives on the struct. Mechanics:
`Graph[Behavior]` and `AddTransition` (auto-adds nodes; first node added is
the entry); insertion order is priority; the stack DFS in `BehaviorTree.Call`
with a worked diagram; `Traversed` makes execution DAG-shaped even if you
wire a cycle; restart vs resume keyed on whether `State` is empty;
pause-mid-tree (`chat → pause → chat2`). Patterns: Sequence (a chain),
Selector (siblings + `Skip`/`Terminal`, with the caveat that `Terminal`
clears the *whole* tree), branch-on-annotation via `Skip` in a Go state.
Examples: `examples/signals` again for the DFS order; (new)
`examples/tree-loop` — drive a `chat → pause → chat2` tree **directly**, with
a hand-written loop that appends user input when it sees
`CollectUserInputSignal`. This is the point where the bottom-up pass first
touches the top: the reader has rebuilt, by hand, what `RunLoop` does for
them. Sharp edges: children push in the opposite order on the pause path
(priority inverts after a pause when a pause node has several children);
`Call` on an empty tree panics; a tree is not goroutine-safe.

**Chapter 8 — The executive.** `TodoListExecutive` holds a *flat list* of
behaviors and is itself a behavior. `Plan`: the planner prompt (`Preamble`
templated against history, the behavior list rendered as `Name: Description`,
the `Example` shown as the sample direction), the JSON reply, name-based
lookup, the `Re-plan` tombstone, `fixJSON` retry, and what a step contains
(`Copy()` of the behavior, a one-message list with the direction, the
`raw_history`/`$context` annotations). `Execute`: concurrent fan-out on
copies, result triage by signal, paused steps re-stashed into `plan`, the
summarizer prompt vs the work-in-progress summarizer vs the single-step
shortcut, `OutOfBoundsHandler`, `MaxPlanDepth` and the re-plan recursion.
`Call` vs `RunLoop` as twins (nested behavior vs top-level channel-driven
agent); `History` vs step `Messages`. Examples: `examples/oneshot` (one
`Call`, no loop); (new) `examples/poetry` — an executive with two behaviors
(haiku, sonnet) and a `CannedResponseState` out-of-bounds handler, showing
the planner choosing between behaviors and rejecting off-topic input. Closes
with **"The trace, re-read"**: Chapter 2's seven steps restated with every
identifier now known. Sharp edges: planner failures `panic` (unparseable
plan, unknown step name, planning LLM error); behaviors are matched by
`Name()` case-sensitively while `re-plan` is matched case-insensitively;
`Output` is a side channel the author has marked `FIXME`; `Copy()` of an
executive dereferences a nil `OutOfBoundsHandler`.

## Examples

Reused as-is: `examples/quickstart`, `examples/oneshot`,
`examples/annotations-probe`, `examples/test` (the `chat → pause → chat2`
shape, referenced in Chapter 7). `examples/snapshot-simple` is cited in
Chapter 2 for the `Preamble` trick only.

New, all under `examples/`, each with the same header-comment style as the
existing learning examples ("learning-purposes example, NOT a template"):

| Example | Chapter | Needs token | Demonstrates |
|---|---|---|---|
| `state-direct` | 5 | yes | one `LLMCompletionState` called directly with a templated system prompt |
| `signals` | 6, 7 | **no** | print-only states; visit order under each signal; insertion-order priority |
| `tree-loop` | 7 | yes | a tree driven by a hand-written loop handling `CollectUserInputSignal` |
| `poetry` | 8 | yes | two-behavior executive + out-of-bounds handler |

`examples/signals` is deliberately hermetic so at least one example runs in
CI and in a reader's shell with no credentials.

## Book mechanics

- **Location:** `book/` at the repository root. `book/book.toml`,
  `book/src/SUMMARY.md`, chapters under `book/src/part-1/` and
  `book/src/part-2/`, front matter under `book/src/`. Build output
  `book/book/` is git-ignored.
- **Toolchain:** the installed `mdbook` (v0.5.4) and `mdbook-mermaid`
  preprocessor for diagrams (sequence diagram in Chapter 2, graph/DFS
  diagrams in Chapters 6–8). Both are already on this machine.
- **Code is never pasted.** Every listing is
  `{{#include ../../../examples/<name>/main.go:<anchor>}}` against a real
  package that `go build ./...` compiles. Anchors are `// ANCHOR: name` /
  `// ANCHOR_END: name` comments in the example source. This is the
  mechanism that prevents the drift the current README has (it documents a
  snapshot API that does not exist).
- **Callouts** use plain blockquotes with a bold lead, no extra preprocessor:
  `> **Sharp edge.** …`, `> **Coming from LangGraph.** …`.
- **References** are by identifier and file, never by line number.
- **Existing material** is the source, not the destination:
  `documentation/arboreal-behavior-trees.md`, `arboreal-behavior-state.md`,
  `executive.md`, `snapshots.md`, `arboreal-analysis.md` and the deck
  `presentation/arboreal-walkthrough.md` are mined for Chapters 2–8. They are
  left in place, untouched, in this scope.

## Verification

A book change is done when all of the following pass:

1. `go build ./... && go vet ./...` — every included example compiles.
2. `mdbook build book` exits 0 **and prints no warnings** (a missing include
   path is a warning, not an error, in mdbook, so the check greps the build
   log for `WARN`).
3. `examples/signals` runs with no token and its output matches the listing
   in Chapter 6.
4. Each token-requiring example has been run once by hand and its
   representative output pasted into the chapter as a fenced block marked as
   a sample (model output varies).
5. Every "Sharp edge" callout cites the identifier that exhibits it, and the
   claim has been re-checked against the source at writing time.

## Out of scope for this spec

- Parts III and IV (see appendix).
- Fixing any framework bug the book surfaces. The book documents; fixes are
  separate changes.
- Correcting `README.md` (the book will supersede it; a later change can
  point the README at the book).
- Publishing/hosting (GitHub Pages or otherwise).
- Rewriting or deleting `documentation/*.md`.

## Appendix — deferred outline (Parts III and IV)

Kept here so the shape agreed in brainstorming is not lost.

**Part III — Running agents.** 9 Channels (interface, `TerminalChannel`,
writing your own, Twilio as reference). 10 Snapshots (`TakeSnapshot`/
`Restore`, stable ids via `…WithId` and `HashId`, coherent only between turns,
persisted only with a plan in flight; `examples/snapshot-simple`,
`examples/little-spy`). 11 Tracing (`Trace` via context, message types, the
breadcrumb, must-drain, `ClientID`). 12 Tools via MCP (`MCPClientMux`,
transports, `WithMCPClient`, the `AllowTools` loop, first-tool-call-only
limit; `examples/tools`). 13 Models and providers (URIs, env vars, defaults,
provider status, `fixJSON`). 14 Memory and retrieval (`MemoryStore`,
sqlite-vec banks, `SemanticChunker`; `examples/crm`). 15 Testing agents
(stub behaviors, pre-seeded plans, one-shot channels; what cannot be stubbed).

**Part IV — Putting it together.** 16 Capstone (several behaviors,
annotations, a pause, an MCP tool, snapshot persistence). 17 Patterns and
pitfalls cookbook. 18 The Lua engine, short tour.

**Appendices.** API summary by file; known limitations distilled from
`documentation/production-readiness-gaps.md` and `tech-debt.md`; glossary;
LangGraph cheat sheet.
