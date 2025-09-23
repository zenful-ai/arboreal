package engine

import (
	"context"
	"fmt"
	"net/http"

	"github.com/cjoudrey/gluahttp"
	luajson "github.com/layeh/gopher-json"
	lua "github.com/yuin/gopher-lua"
	"github.com/zenful-ai/arboreal"
)

type LuaTypeExport struct {
	Name        string
	Constructor lua.LGFunction
	Methods     map[string]lua.LGFunction
	Static      map[string]lua.LGFunction
	Index       bool
}

var luaTypeExportsV1 = []LuaTypeExport{
	arborealNamespaceExport,
	arborealBehaviorInterface,
	arborealBehaviorTree,
	arborealPlanner,
	arborealAnnotatedMessage,
	arborealSignal,
	arborealAnnotation,

	mcpNamespaceExport,
}

var exportMap = map[int][]LuaTypeExport{
	1: luaTypeExportsV1,
}

func registerLuaType(l *lua.LState, luaTypeExport LuaTypeExport) {
	typeMetatable := l.NewTypeMetatable(luaTypeExport.Name)
	l.SetGlobal(luaTypeExport.Name, typeMetatable)

	if luaTypeExport.Constructor != nil {
		l.SetField(typeMetatable, "new", l.NewFunction(luaTypeExport.Constructor))
	} else {
		for k, v := range luaTypeExport.Methods {
			l.SetField(typeMetatable, k, l.NewFunction(v))
		}
	}

	if luaTypeExport.Constructor != nil || luaTypeExport.Index {
		l.SetField(typeMetatable, "__index", l.SetFuncs(l.NewTable(), luaTypeExport.Methods))
	}

	if luaTypeExport.Static != nil {
		for k, v := range luaTypeExport.Static {
			l.SetField(typeMetatable, k, l.NewFunction(v))
		}
	}
}

const (
	CurrentRuntimeVersion = 1
)

type Runtime struct {
	l            *lua.LState
	entrypoint   arboreal.Behavior
	traceContext context.Context
	Trace        arboreal.Trace
}

func (r *Runtime) Entry() arboreal.Behavior {
	return r.entrypoint
}

func (r *Runtime) Context() context.Context {
	return r.traceContext
}

func (r *Runtime) Call(messages arboreal.AnnotatedMessages) (arboreal.AnnotatedMessages, arboreal.Signal) {
	return r.entrypoint.Call(r.traceContext, messages)
}

func getEntryPoint(l *lua.LState) (arboreal.Behavior, error) {
	var b arboreal.Behavior
	arborealGlobal := l.GetGlobal("arboreal")
	if arborealGlobal == nil {
		return nil, fmt.Errorf("missing arboreal namespace")
	}

	if lt, ok := arborealGlobal.(*lua.LTable); ok {
		luaEntry := lt.RawGetString("entry")
		if luaEntry == nil {
			return nil, fmt.Errorf("an entrypoint must be defined by the agent")
		}

		if ud, ok := luaEntry.(*lua.LUserData); ok {
			if b, ok = ud.Value.(arboreal.Behavior); ok {
				return b, nil
			} else {
				return nil, fmt.Errorf("the entrypoint must be an Arboreal Behavior")
			}
		} else {
			return nil, fmt.Errorf("entrypoint of the wrong type: must be an Arboreal Behavior")
		}
	} else {
		return nil, fmt.Errorf("arboreal namespace seems to be redefined")
	}
}

func TestInRuntime(script string) error {
	var err error

	l := lua.NewState()

	// HTTP requests module
	l.PreloadModule("http", gluahttp.NewHttpModule(&http.Client{}).Loader)
	// JSON support
	luajson.Preload(l)

	for _, export := range exportMap[CurrentRuntimeVersion] {
		registerLuaType(l, export)
	}

	// TODO: Handle cancel
	ctx, _ := context.WithCancel(context.WithValue(context.Background(), "id", arboreal.MonotonicIdGenerator("id-")))
	l.SetContext(ctx)

	if err = l.DoString(script); err != nil {
		return err
	}

	return nil
}

func (r *Runtime) NoTrace() {
	go func() {
		for {
			select {
			case _, ok := <-r.Trace:
				if !ok {
					return
				}
			}
		}
	}()
}

type RuntimeOptions struct {
	MCPProfile *string
	MCPClient  *arboreal.MCPClientMux
}

func InitializeRuntime(script string, version int, options *RuntimeOptions) (*Runtime, error) {
	var runtime Runtime
	var err error

	if options == nil {
		options = &RuntimeOptions{}
	}

	l := lua.NewState()

	// HTTP requests module
	l.PreloadModule("http", gluahttp.NewHttpModule(&http.Client{}).Loader)
	// JSON support
	luajson.Preload(l)

	for _, export := range exportMap[version] {
		registerLuaType(l, export)
	}

	// Trace execution context
	var trace = make(arboreal.Trace, 100)
	ctx := context.WithValue(context.Background(), "arboreal_trace", trace)
	runtime.Trace = trace
	runtime.traceContext = ctx

	// MCP Context
	if options.MCPClient != nil {
		ctx = context.WithValue(ctx, "arboreal_mcp_client", options.MCPClient)
	}

	// TODO: Handle cancel
	ctx, _ = context.WithCancel(context.WithValue(ctx, "id", arboreal.MonotonicIdGenerator("id-")))
	l.SetContext(ctx)

	if err = l.DoString(script); err != nil {
		return nil, err
	}

	runtime.entrypoint, err = getEntryPoint(l)
	if err != nil {
		return nil, err
	}

	return &runtime, nil
}
