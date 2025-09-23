package engine

import (
	"fmt"
	lua "github.com/yuin/gopher-lua"
	"github.com/zenful-ai/arboreal"
)

var arborealAnnotation = LuaTypeExport{
	Name:        "annotation",
	Constructor: arborealNewAnnotation,
	Methods: map[string]lua.LGFunction{
		"value": arborealValueGetter,
	},
	Static: map[string]lua.LGFunction{
		"append": arborealAppendAnnotation,
		"find":   arborealFindAnnotation,
	},
}

func arborealAnnotationFromLua(ud *lua.LUserData) (*arboreal.Annotation, error) {
	if vv, ok := ud.Value.(*arboreal.Annotation); ok {
		return vv, nil
	}

	return nil, fmt.Errorf("failed to convert userdata to arboreal behavior")
}

func checkArborealAnnotation(l *lua.LState, n int) *arboreal.Annotation {
	ud := l.CheckUserData(n)

	annotation, err := arborealAnnotationFromLua(ud)
	if err != nil {
		l.ArgError(1, err.Error())
		return nil
	}

	return annotation
}

func arborealNewAnnotationFromTable(l *lua.LState, t *lua.LTable) *arboreal.Annotation {
	var annotation arboreal.Annotation

	t.ForEach(func(k, v lua.LValue) {
		if k.String() == "name" {
			annotation.Name = v.String()
		}

		if k.String() == "value" {
			switch t := v.Type(); t {
			case lua.LTString:
				annotation.Data = v.String()
			case lua.LTBool:
				annotation.Data = v.(lua.LBool)
			case lua.LTNumber:
				annotation.Data = float64(v.(lua.LNumber))
			case lua.LTUserData:
				annotation.Data = v
			default:
				l.RaiseError("unknown data type: %s", t)
			}
		}

		if k.String() == "description" {
			annotation.Explanation = v.String()
		}
	})

	return &annotation

}

func arborealNewAnnotation(l *lua.LState) int {
	annotation := arborealNewAnnotationFromTable(l, l.CheckTable(1))
	ud := l.NewUserData()
	ud.Value = annotation

	l.SetMetatable(ud, l.GetTypeMetatable("annotation"))
	l.Push(ud)

	return 1
}

func arborealValueGetter(l *lua.LState) int {
	annotation := checkArborealAnnotation(l, 1)

	if annotation == nil {
		l.Push(lua.LNil)
		return 1
	}

	switch t := annotation.Data.(type) {
	case lua.LValue:
		l.Push(t)
	case string:
		l.Push(lua.LString(t))
	case float64:
		l.Push(lua.LNumber(float64(t)))
	case bool:
		l.Push(lua.LBool(t))
	case nil:
		l.Push(lua.LNil)
	default:
		l.RaiseError("unknown annotation data type: %T", t)
		return 0
	}

	return 1
}

func arborealFindAnnotation(l *lua.LState) int {
	messages := checkArborealAnnotatedMessages(l, 1)
	name := l.CheckString(2)

	ud := l.NewUserData()
	ud.Value = messages.GetAnnotation(name)

	l.SetMetatable(ud, l.GetTypeMetatable("annotation"))
	l.Push(ud)

	return 1
}

func arborealAppendAnnotation(l *lua.LState) int {
	messages := checkArborealAnnotatedMessages(l, 1)

	var annotation *arboreal.Annotation
	var err error

	// Allow either annotation.new() or a plain table to create an annotation
	v := l.Get(2)
	if ud, ok := v.(*lua.LUserData); ok {
		annotation, err = arborealAnnotationFromLua(ud)
		if err != nil {
			l.ArgError(2, err.Error())
			return 0
		}
	} else if t, ok := v.(*lua.LTable); ok {
		annotation = arborealNewAnnotationFromTable(l, t)
	} else {
		l.ArgError(2, "annotation or table expected")
		return 0
	}

	if annotation == nil {
		l.ArgError(1, "annotation or table expected")
		return 0
	}

	if annotation.Name == "" {
		l.ArgError(2, "annotation name is required")
		return 0
	}

	m := *messages

	// If arboreal_trace is set in our context, then record annotation trace information
	ctx := l.Context()
	if ctx != nil && ctx.Value("arboreal_trace") != nil {
		m.AddTraceInformation(annotation.Name)
	}

	if m[len(m)-1].Annotations == nil {
		m[len(m)-1].Annotations = make(map[string]arboreal.Annotation)
	}

	m[len(m)-1].Annotations[annotation.Name] = *annotation

	l.Push(arborealAnnotatedMessagesToLua(l, m))
	return 1
}
