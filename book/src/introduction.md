# Introduction

Arboreal is a Go framework for agentic AI systems. Its unit of composition is the *behavior*: anything that takes a conversation and returns an updated conversation plus a signal. Behaviors are wired into behavior trees: directed graphs whose nodes are LLM calls, pauses, plain Go functions, or other trees. A tree is itself a behavior, which is what makes them compose. A plan-and-execute *executive* turns a user's request into a list of trees to run, then feeds later messages to the trees still waiting on the user. There is no separate state object to design: the conversation *is* the state, and every layer reads the message list and appends to it. Pausing to ask the user a question is not an exception or a callback but a first-class signal that travels back up the stack. Because the message list and the tree's position in its graph are both plain data, an executive mid-conversation can be snapshotted to JSON and restored in a different process.

## Who this book is for

This book is for engineers who have already built agents with LangGraph or a similar framework and are new to Arboreal. You know what a tool call is, you have wired up a graph with conditional edges, and you have probably fought a checkpointer at least once. What you need is a map from those concepts onto this codebase. This book is that map.

It assumes fluent Go: the chapters do not stop to explain goroutines, interfaces or `context.Context`. If Go is not yet fluent for you, Appendix A is a primer on exactly the Go the examples use — enough to read this book, not to learn the language. Every chapter has a "Coming from LangGraph" sidebar that maps the concept you already know onto the Arboreal one, and says where the mapping breaks down.

## What you need

- Go 1.23 or newer.
- `go get github.com/zenful-ai/arboreal`
- An OpenAI API key in `OPENAI_TOKEN`. Every example in this book uses the framework's default model, `gpt-4o-mini`, through the OpenAI provider, so this is the only credential you need. (A couple of examples — `examples/signals`, `examples/snapshot-edges` — need no key at all.)

All code in this book is included straight from the `examples/` directory of the repository and compiles with `go build ./...`. Run any of them with `go run ./examples/<name>`.

## How this book is organized

The book is a spiral: it passes over the framework twice, once from the top and once from the bottom.

Part I is the top-down pass. In Chapter 1 you run a chat bot and read the thirty-odd lines that make it up. In Chapter 2 you follow a single user message through every layer of the framework, from the executive's run loop down to the LLM call and back, naming each piece as it appears but not yet explaining it. Chapter 3 is a translation table from LangGraph, so you can attach each of those names to something you already know.

Part II is the bottom-up pass. It rebuilds the framework from the bottom of the stack up: messages and annotations (Chapter 4), behaviors and states (Chapter 5), signals (Chapter 6), behavior trees (Chapter 7), and finally the executive (Chapter 8). Each Part II chapter ends with a section called "Back to the trace" that points at the step of Chapter 2 it has just explained. Chapter 8 ends by re-reading the whole trace with nothing left unexplained.

Part III takes agents out of the chat loop: one turn at a time with no `RunLoop` (Chapter 9), snapshots that carry a paused conversation across a process boundary (Chapter 10), tracing (Chapter 11), tools over MCP (Chapter 12), and what can be tested without a model (Chapter 13).

Part IV spends all of it at once. Chapter 14 is a capstone: a single program that assembles every piece of the framework into an agent that handles one message per process, with a pause that outlives the exit. Chapter 15 is a cookbook — every pitfall in the book indexed by symptom, and the patterns the examples converged on indexed by intent — and it is the page to come back to.

Neither direction works on its own. A pure top-down book cannot get past hello world without hand-waving, and a pure bottom-up book cannot get to hello world at all: the run loop lives on the executive, which is the top of the stack, so even the smallest working program depends on the piece that is hardest to explain. But explaining the executive honestly requires signals and pause/resume, which are the bottom of the stack, because what the executive does with a finished step is decided by the signal it came back with. The spiral resolves the tension by letting you see everything working first and then building each piece in the order its dependencies allow.

## A note on sharp edges

This book documents the framework as it is at the current commit, not as it is intended to be. Where a name promises more than the code delivers, the book says so rather than describing the design goal. Where the current behavior will surprise or bite you, you will find a callout in the style shown below. Each names the identifier involved, describes what actually happens, and gives a workaround.

```admonish warning title="Sharp edge"
Callouts in this style mark verified pitfalls in the current implementation. Read them; they are the parts of the framework that cost the authors an afternoon.
```
