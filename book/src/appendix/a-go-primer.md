# Appendix A — A Go primer

This appendix is not a Go tutorial — that takes a book of its own, and the
official [Tour of Go](https://go.dev/tour) is a good one. It is the Go this
book's examples actually use, for a reader who is fluent in some other
typed language and wants to read on without stopping at the syntax. Every
snippet below comes from `book/examples/go-primer`, one runnable, model-free
program; run it and read the output alongside the text:

```sh
go run ./book/examples/go-primer
```

What the examples do *not* use, you will not find here: generics, `select`,
the `sync` package, error wrapping. If you meet a construct in the book
that this appendix skips, that is a bug in the appendix.

## Reading a Go file

Every file opens with a `package` clause and its imports. `package main`
plus a `func main()` is an executable; everything under `examples/` is one.
Visibility is spelled with capitalization: `Behavior`, `Call` and `History`
are exported, `plan` and `executivePlannerPrompt` are private to their
package. There are no exceptions to learn beyond that. Formatting is not a
matter of taste — `gofmt` decides, and the tab indentation in every listing
is its doing. Four commands cover this book: `go run ./examples/<name>`,
`go build ./...`, `go vet ./...`, `go test ./...`.

## Values, zero values and errors

```go
{{#include ../../examples/go-primer/main.go:values}}
```

`:=` declares and infers; `var` declares without initializing, and every
type has a useful **zero value** — `0`, `""`, `nil` — that the language
hands you instead of an "uninitialized" error. Functions return multiple
values, and the idiomatic pair is result-plus-`error`: no exceptions, no
sum types, just a value you check with `if err != nil` before moving on.
You will read that line in every example in this book. `panic` aborts the
program with a stack trace; it is for broken invariants, not expected
failures — though this framework reaches for it more often than Go style
recommends, which is what several Sharp edges are about.

The book's central signature is a pair-return with no error in it:
`tree.Call(ctx, history)` hands back `(history, signal)`, and Chapter 6 is
the story of the second value.

## Structs and embedding

```go
{{#include ../../examples/go-primer/main.go:structs}}
```

Structs are declared field by field and built with **composite literals**
naming each field — `LLMCompletionOptions{System: "…"}` in the chapters,
where an omitted field silently takes its zero value (that is what "empty
options" means in Chapter 1). **Embedding** is the unnamed field:
the outer type answers for the inner type's fields and methods as if they
were its own. The framework's `AnnotatedMessage` embeds the LLM library's
`ChatCompletionMessage` this way (Chapter 4), which is why
`LastMessage().Content` works and why a message literal writes
`ChatCompletionMessage:` as a field name. It looks like inheritance; it is
only composition with promotion.

## Methods, pointers and interfaces

```go
{{#include ../../examples/go-primer/main.go:methods}}
```

Any named type can carry methods, and the **receiver** — the `(c *Counter)`
before the method name — decides what the method sees: a pointer receiver
can mutate, a value receiver gets a copy. The rule that bites: only `*T`
has the pointer-receiver methods, so when an interface demands them, you
must pass `&value`. An **interface** is satisfied implicitly — have the
methods, be the type; no declaration links `*BehaviorState` to `Behavior`
anywhere in the framework (Chapter 5). This pair of rules is the whole
answer to Chapter 1's "Why the ampersands?".

## Getting the type back out

```go
{{#include ../../examples/go-primer/main.go:typeswitch}}
```

An interface value remembers its concrete type, and two forms recover it:
the **type assertion** `d.(*Counter)` with comma-ok, and the **type
switch**. The framework runs on these: every signal comes back as a
`Signal` interface, and every layer decides what to do by matching
`*SkipSignal`, `*CollectUserInputSignal`, `*ErrorSignal` (Chapter 6 — and
why a signal returned by value matches nothing). Note the nil line: a nil
interface matches only `case nil`, so code that forgets that case simply
falls through — Chapter 14 meets a `nil` `Behavior` doing exactly that
inside `TakeSnapshot`.

## Slices and maps

```go
{{#include ../../examples/go-primer/main.go:collections}}
```

A slice is a view onto a shared backing array; `append` may move it, so
the result is always reassigned — the shape of `AppendToMessages` and of
`history, sig = tree.Call(…)` throughout the book. A nil slice is a fine
empty list. Maps must be `make`d before writing, reads on missing keys
return the zero value, and comma-ok distinguishes "absent" from "zero".
A named slice type can carry methods: `AnnotatedMessages` is a
`[]AnnotatedMessage` with `LastMessage()` and friends on it (Chapter 4).
`map[string]any` — `any` being the empty interface, satisfied by
everything — is what a snapshot deserializes into (Chapter 10).

## Functions are values

```go
{{#include ../../examples/go-primer/main.go:closures}}
```

Functions are first-class, and a **closure** captures the variables around
it by reference, not by value. Both halves matter here: a
`BehaviorState`'s entire behavior is one function-typed field, `Lambda`
(Chapter 5), and a lambda that captures a mutable variable shares it with
every copy of the state — which is Chapter 5's sharp edge about trees the
executive runs concurrently.

## Goroutines and channels

```go
{{#include ../../examples/go-primer/main.go:concurrency}}
```

`go f()` starts a **goroutine** — a cheap concurrent function call with no
handle and no return value; results come back over **channels**. Sends and
receives on an unbuffered channel block until the other side shows up,
which is both the synchronization and the trap: a channel nobody drains
stops the sender forever. The executive fans a plan out exactly like this
loop — one goroutine per step, results collected on a channel (Chapter 2,
step 3) — and the tracing channel of Chapter 11 is the trap in the wild:
subscribe and you must drain, or the agent deadlocks.

## `context.Context` and `defer`

```go
{{#include ../../examples/go-primer/main.go:context}}
```

A `context.Context` is the first parameter of nearly every function in the
book. It is a request-scoped bag: cancellation you can check, and values
keyed by your own types. The framework uses the values half — the trace
channel (Chapter 11) and the MCP mux (Chapter 12) both travel down to the
states inside a context, which is why nothing else in a program's plumbing
has to know about them. `defer` schedules a call for the function's
return, on every path; `defer mux.Close()` in Chapter 12 is the idiom.

## JSON

```go
{{#include ../../examples/go-primer/main.go:json}}
```

`encoding/json` marshals exported fields, under their own names unless a
backtick **struct tag** renames them — compare the lowercase `"ref"` with
the capitalized `"Role"` in the output, from the untagged `Message`.
`Unmarshal` needs a pointer to write into. This is the whole mechanism
behind snapshots (Chapter 10: a snapshot is a map you `json.Marshal`) and
behind `Plan` parsing the planner's JSON array (Chapter 8). One behavior
to know: unmarshaling into `any` yields `map[string]any` with no memory
of field order — Chapter 10's last sharp edge rests on it.

## Tests

Go's test runner needs no framework: any `TestXxx(t *testing.T)` function
in a `_test.go` file runs under `go test`, failing loudly via `t.Fatal` or
`t.Fatalf` and passing silently otherwise. Two idioms carry Chapter 13:
the **table test** — a slice of struct literals, one row per scenario,
ranged over with `t.Run` — and `t.Setenv`, which pins an environment
variable for one test (`examples/bookshelf` uses it to make sure a test
never spends a real token). Start with `examples/test` and
`examples/bookshelf`'s test files; they are the book's testing chapter in
executable form.

## One backtick, two meanings

Raw string literals are written in backticks and honor no escapes, which
is why prompts and templates in the framework read naturally across many
lines (`executivePlannerPrompt` in Chapter 2 is one). The same backticks
around a struct field are a tag, as in the JSON section above. Context
decides; the compiler is never confused, and now neither are you.
