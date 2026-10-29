package engine

import (
	"strings"
	"testing"

	lua "github.com/yuin/gopher-lua"
	"github.com/zenful-ai/arboreal"
)

func TestPlannerRefusesABehaviorThatIsNotAStep(t *testing.T) {
	err := TestInRuntime(`
local s = arboreal.state("a", "a", function(history) return history, nil end)
local p = arboreal.planner("p", "plans", s)
`)

	if err == nil {
		t.Fatal("arboreal.planner accepted a bare state as a behavior, want an argument error")
	}
	if !strings.Contains(err.Error(), "must be trees") {
		t.Fatalf("error = %q, want it to say a planner's behaviors must be trees", err)
	}
}

func TestPlannerAcceptsATree(t *testing.T) {
	err := TestInRuntime(`
local tree = arboreal.tree("chat", "chat with the user", "hi there")
tree:add(arboreal.state("a", "a", function(history) return history, nil end))
local p = arboreal.planner("p", "plans", tree)
`)

	if err != nil {
		t.Fatalf("arboreal.planner refused a tree: %v", err)
	}
}

func TestPlannerNamesTheOffendingArgument(t *testing.T) {
	err := TestInRuntime(`
local tree = arboreal.tree("chat", "chat with the user", "hi there")
tree:add(arboreal.state("a", "a", function(history) return history, nil end))
local inner = arboreal.planner("inner", "plans too")
local p = arboreal.planner("p", "plans", tree, inner)
`)

	if err == nil {
		t.Fatal("arboreal.planner accepted a nested planner as a behavior, want an argument error")
	}
	// A nested executive is a Behavior but not a Step, and the error must
	// point at it, not at the tree before it.
	if !strings.Contains(err.Error(), "bad argument #4") || !strings.Contains(err.Error(), "must be trees") {
		t.Fatalf("error = %q, want it to name argument #4 and say a planner's behaviors must be trees", err)
	}
}

// plannerInRuntime runs script on a test state and returns the state and
// the executive the script left in the global p, so a test can read from Go
// what the script set and set from Go what the script then reads.
func plannerInRuntime(t *testing.T, script string) (*lua.LState, *arboreal.TodoListExecutive) {
	t.Helper()
	l := testState()
	if err := l.DoString(script); err != nil {
		t.Fatalf("script failed: %v", err)
	}
	ud, ok := l.GetGlobal("p").(*lua.LUserData)
	if !ok {
		t.Fatalf("global p is %T, want the planner's userdata", l.GetGlobal("p"))
	}
	exec, err := arborealPlannerFromLua(ud)
	if err != nil {
		t.Fatal(err)
	}
	return l, exec
}

// TestPlannerPreamblesRoundTrip pins the two preamble accessors, preamble
// and summary_preamble: one argument reads, two set, and each reaches its
// own field on the Go executive, in both directions.
func TestPlannerPreamblesRoundTrip(t *testing.T) {
	l, exec := plannerInRuntime(t, `
local tree = arboreal.tree("chat", "chat with the user", "hi there")
tree:add(arboreal.state("a", "a", function(history) return history, nil end))
p = arboreal.planner("p", "plans", tree)
p:preamble("PLANNER")
p:summary_preamble("REPLY")
`)
	if exec.Preamble != "PLANNER" || exec.SummaryPreamble != "REPLY" {
		t.Fatalf("after the script, Preamble = %q and SummaryPreamble = %q, want PLANNER and REPLY", exec.Preamble, exec.SummaryPreamble)
	}

	exec.Preamble = "PLANNER FROM GO"
	exec.SummaryPreamble = "REPLY FROM GO"
	err := l.DoString(`
if p:preamble() ~= "PLANNER FROM GO" then error("preamble() = " .. p:preamble()) end
if p:summary_preamble() ~= "REPLY FROM GO" then error("summary_preamble() = " .. p:summary_preamble()) end
`)
	if err != nil {
		t.Fatalf("the getters did not read what Go set: %v", err)
	}
}
