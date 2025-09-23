package engine

import (
	"fmt"
	lua "github.com/yuin/gopher-lua"
	"github.com/zenful-ai/arboreal"
)

var arborealSignal = LuaTypeExport{
	Name: "signal",
	Methods: map[string]lua.LGFunction{
		"description": arborealSignalDescriptionGetter,

		"error": arborealSignalConstructorError,
		"skip":  arborealSignalConstructContinuation,
		"user":  arborealSignalConstructUserCollectInput,
		"stop":  arborealSignalConstructTerminal,
	},
	Index: true,
}

func arborealSignalFromLua(ud *lua.LUserData) (arboreal.Signal, error) {
	if vv, ok := ud.Value.(arboreal.Signal); ok {
		return vv, nil
	}

	return nil, fmt.Errorf("failed to convert userdata to arboreal signal")
}

func arborealSignalToLua(l *lua.LState, s arboreal.Signal) lua.LValue {
	ud := l.NewUserData()
	ud.Value = s

	l.SetMetatable(ud, l.GetTypeMetatable("signal"))
	l.Push(ud)

	return ud
}

func checkArborealSignal(l *lua.LState, n int) arboreal.Signal {
	ud := l.CheckUserData(n)

	signal, err := arborealSignalFromLua(ud)
	if err != nil {
		l.ArgError(n, err.Error())
		return nil
	}

	return signal
}

func arborealSignalDescriptionGetter(l *lua.LState) int {
	signal := checkArborealSignal(l, 1)

	l.Push(lua.LString(signal.Description()))
	return 1
}

func arborealSignalConstructorError(l *lua.LState) int {
	message := l.CheckString(1)

	errorType := arboreal.StateErrorTypeUnknown
	if l.GetTop() == 2 {
		errorType = l.CheckString(2)
	}

	ud := l.NewUserData()
	ud.Value = &arboreal.ErrorSignal{
		ErrorMessage: message,
		ErrorType:    errorType,
	}

	l.SetMetatable(ud, l.GetTypeMetatable("signal"))
	l.Push(ud)

	return 1
}

func arborealSignalConstructContinuation(l *lua.LState) int {
	reason := l.CheckString(1)

	ud := l.NewUserData()
	ud.Value = &arboreal.SkipSignal{
		Reason: reason,
	}

	l.SetMetatable(ud, l.GetTypeMetatable("signal"))
	l.Push(ud)

	return 1
}

func arborealSignalConstructUserCollectInput(l *lua.LState) int {
	reason := l.CheckString(1)

	ud := l.NewUserData()
	ud.Value = &arboreal.CollectUserInputSignal{
		Reason: reason,
	}

	l.SetMetatable(ud, l.GetTypeMetatable("signal"))
	l.Push(ud)

	return 1
}

func arborealSignalConstructTerminal(l *lua.LState) int {
	reason := l.CheckString(1)

	ud := l.NewUserData()
	ud.Value = &arboreal.TerminalSignal{
		Reason: reason,
	}

	l.SetMetatable(ud, l.GetTypeMetatable("signal"))
	l.Push(ud)

	return 1
}
