// Package main is the runnable companion to Appendix A. Each function
// demonstrates the slice of Go that the appendix section of the same
// name explains; run it and read the output next to the text:
//
//	$ go run ./book/examples/go-primer
//
// No API token needed.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ANCHOR: values
// half returns a result and an error; the caller checks the error before
// touching the result. This pair-return is Go's only error mechanism.
func half(n int) (int, error) {
	if n%2 != 0 {
		return 0, fmt.Errorf("%d is odd", n)
	}
	return n / 2, nil
}

func values() {
	h, err := half(10) // := declares h and err and infers their types
	if err != nil {
		panic(err) // no exceptions; panic is for bugs, not for control flow
	}
	fmt.Println("half(10) =", h)

	var count int        // declared without a value: the zero value, 0
	var name string      // ""
	var m map[string]int // nil — reads are safe, writes would panic
	fmt.Println(count, name == "", m["missing"])
}

// ANCHOR_END: values

// ANCHOR: structs
type Message struct {
	Role    string
	Content string
}

// Note carries a Message plus extras. The embedded field has no name: a
// Note answers for a Message's fields — n.Content works — without any
// inheritance behind it.
type Note struct {
	Message
	Tags []string
}

func structs() {
	n := Note{
		Message: Message{Role: "user", Content: "hello"},
		Tags:    []string{"greeting"},
	}
	fmt.Println(n.Content, n.Tags) // Content is promoted from the embedded Message
}

// ANCHOR_END: structs

// ANCHOR: methods
type Counter struct{ n int }

// A pointer receiver: the method can change the Counter it is called on.
func (c *Counter) Add() { c.n++ }

// A value receiver: the method sees a copy.
func (c Counter) Value() int { return c.n }

// An interface is satisfied by anything with the right methods; there is
// no "implements" declaration anywhere.
type Describable interface {
	Describe() string
}

// Describe has a pointer receiver, so *Counter satisfies Describable and
// plain Counter does not — which is why such values travel as &c.
func (c *Counter) Describe() string {
	return fmt.Sprintf("counter at %d", c.n)
}

func methods() {
	var c Counter
	c.Add()                // Go takes &c for you on a method call…
	var d Describable = &c // …but an interface assignment needs the & spelled out
	fmt.Println(d.Describe(), c.Value())
}

// ANCHOR_END: methods

// ANCHOR: typeswitch
func classify(d Describable) string {
	// A type assertion with comma-ok: the concrete value back out of the
	// interface, plus whether it really was that type.
	if c, ok := d.(*Counter); ok {
		return fmt.Sprintf("a counter holding %d", c.n)
	}
	// Or a switch over the concrete type. A nil interface matches only
	// the nil case — never a concrete type's case.
	switch v := d.(type) {
	case nil:
		return "nothing at all"
	default:
		return "something else: " + v.Describe()
	}
}

// ANCHOR_END: typeswitch

// ANCHOR: collections
func collections() {
	var list []string             // nil, and append works on nil…
	list = append(list, "a", "b") // …but returns the new slice: always reassign
	for i, s := range list {
		fmt.Println(i, s)
	}

	ages := make(map[string]int) // maps must be made before writing
	ages["ada"] = 36
	if age, ok := ages["ada"]; ok { // comma-ok: was the key present?
		fmt.Println("ada:", age)
	}
}

// ANCHOR_END: collections

// ANCHOR: closures
type Step struct {
	Name string
	Run  func(input string) string // a function is a value; a field can hold one
}

func closures() {
	suffix := "!"
	s := Step{
		Name: "shout",
		// The closure captures suffix itself, not a copy of it: change the
		// variable later and the closure sees the change.
		Run: func(input string) string { return strings.ToUpper(input) + suffix },
	}
	suffix = "!!"
	fmt.Println(s.Run("hello")) // HELLO!!
}

// ANCHOR_END: closures

// ANCHOR: concurrency
func concurrency() {
	results := make(chan string) // unbuffered: a send blocks until someone receives
	words := []string{"tree", "state", "signal"}
	for _, w := range words {
		go func() { // one goroutine per word; go returns immediately
			results <- strings.ToUpper(w)
		}()
	}
	for range words { // exactly as many receives as sends — or someone blocks forever
		fmt.Println(<-results)
	}
}

// ANCHOR_END: concurrency

// ANCHOR: context
type ctxKey string

func contextAndDefer() {
	defer fmt.Println("…and this prints on the way out") // runs when the function returns

	ctx := context.Background()                          // the root context: never done, carries nothing
	ctx = context.WithValue(ctx, ctxKey("user"), "ada")  // a derived context carrying one value
	if v, ok := ctx.Value(ctxKey("user")).(string); ok { // Value returns any; assert the type back
		fmt.Println("from context:", v)
	}
}

// ANCHOR_END: context

// ANCHOR: json
type Record struct {
	Ref      string         `json:"ref"` // the tag names the JSON field
	Messages []Message      `json:"messages"`
	Extra    map[string]any `json:"extra,omitempty"` // any: any type at all
}

func jsonRoundTrip() {
	r := Record{Ref: "tree-qa", Messages: []Message{{Role: "user", Content: "hi"}}}
	data, err := json.Marshal(r)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(data))

	var back Record
	if err := json.Unmarshal(data, &back); err != nil { // & so Unmarshal can write into back
		panic(err)
	}
	fmt.Println(back.Ref, back.Messages[0].Content)
}

// ANCHOR_END: json

func main() {
	values()
	structs()
	methods()
	fmt.Println(classify(&Counter{}))
	fmt.Println(classify(nil))
	collections()
	closures()
	concurrency()
	contextAndDefer()
	jsonRoundTrip()
}
