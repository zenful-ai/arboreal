package engine

import (
	"fmt"
	lua "github.com/yuin/gopher-lua"
	"github.com/zenful-ai/arboreal"
)

var arborealBehaviorInterface = LuaTypeExport{
	Name: "behavior",
	Methods: map[string]lua.LGFunction{
		"name":        arborealBehaviorNameGetter,
		"description": arborealBehaviorDescriptionGetter,
		"call":        arborealBehaviorCallFn,
		"copy":        arborealBehaviorCopy,
	},
	Index: true,
}

func arborealBehaviorFromLua(ud *lua.LUserData) (arboreal.Behavior, error) {
	if vv, ok := ud.Value.(arboreal.Behavior); ok {
		return vv, nil
	}

	return nil, fmt.Errorf("failed to convert userdata to arboreal behavior")
}

func checkArborealBehavior(l *lua.LState, n int) arboreal.Behavior {
	ud := l.CheckUserData(n)

	behavior, err := arborealBehaviorFromLua(ud)
	if err != nil {
		l.ArgError(1, err.Error())
		return nil
	}

	return behavior
}

func arborealBehaviorNameGetter(l *lua.LState) int {
	behavior := checkArborealBehavior(l, 1)

	l.Push(lua.LString(behavior.Name()))
	return 1
}

func arborealBehaviorDescriptionGetter(l *lua.LState) int {
	behavior := checkArborealBehavior(l, 1)

	l.Push(lua.LString(behavior.Description()))
	return 1
}

func arborealBehaviorCallFn(l *lua.LState) int {
	behavior := checkArborealBehavior(l, 1)

	messages := checkArborealAnnotatedMessages(l, 2)

	m, s := behavior.Call(l.Context(), *messages)

	l.Push(arborealAnnotatedMessagesToLua(l, m))

	if s == nil {
		l.Push(lua.LNil)
	} else {
		l.Push(arborealSignalToLua(l, s))
	}

	return 2
}

func arborealBehaviorCopy(l *lua.LState) int {
	behavior := checkArborealBehavior(l, 1)

	ud := l.NewUserData()
	ud.Value = behavior.Copy()

	switch behavior.(type) {
	case *arboreal.BehaviorTree:
		l.SetMetatable(ud, l.GetTypeMetatable("behavior_tree"))
	case *arboreal.BehaviorState:
		l.SetMetatable(ud, l.GetTypeMetatable("behavior"))
	case *arboreal.TodoListExecutive:
		l.SetMetatable(ud, l.GetTypeMetatable("planner"))
	}
	l.Push(ud)

	return 1
}

// Behavior Tree

var arborealBehaviorTree = LuaTypeExport{
	Name: "behavior_tree",
	Methods: map[string]lua.LGFunction{
		"name":        arborealBehaviorNameGetter,
		"description": arborealBehaviorDescriptionGetter,
		"client_id":   arborealBehaviorTreeClientIDGetterAndSetter,
		"call":        arborealBehaviorCallFn,
		"copy":        arborealBehaviorCopy,
		"add":         arborealBehaviorTreeAdd,
	},
	Index: true,
}

func arborealBehaviorTreeFromLua(ud *lua.LUserData) (*arboreal.BehaviorTree, error) {
	if vv, ok := ud.Value.(*arboreal.BehaviorTree); ok {
		return vv, nil
	}

	return nil, fmt.Errorf("failed to convert userdata to arboreal behavior tree")
}

func checkArborealBehaviorTree(l *lua.LState) *arboreal.BehaviorTree {
	ud := l.CheckUserData(1)

	behavior, err := arborealBehaviorTreeFromLua(ud)
	if err != nil {
		l.ArgError(1, err.Error())
		return nil
	}

	return behavior
}

func arborealBehaviorTreeAdd(l *lua.LState) int {
	tree := checkArborealBehaviorTree(l)

	if l.GetTop() == 2 {
		newState := checkArborealBehavior(l, 2)
		tree.AddState(newState)
	} else if l.GetTop() == 3 {
		fromState := checkArborealBehavior(l, 2)
		toState := checkArborealBehavior(l, 3)
		tree.AddTransition(fromState, toState)
	} else {
		l.ArgError(1, "expected either 1 or 2 arguments to add()")
	}

	return 0
}

func arborealBehaviorTreeClientIDGetterAndSetter(l *lua.LState) int {
	tree := checkArborealBehaviorTree(l)

	if l.GetTop() == 2 {
		tree.ClientID = l.CheckString(2)
	} else {
		l.Push(lua.LString(tree.ClientID))
		return 1
	}

	return 0
}
