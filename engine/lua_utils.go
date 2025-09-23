package engine

import (
	lua "github.com/yuin/gopher-lua"
)

func LuaToGo(lv lua.LValue) any {
	switch t := lv.(type) {
	case lua.LString:
		return string(t)
	case *lua.LString:
		return string(*t)
	case lua.LNumber:
		return float64(t)
	case *lua.LNumber:
		return float64(*t)
	case lua.LBool:
		return bool(t)
	case *lua.LBool:
		return bool(*t)
	case *lua.LTable:
		if t.RawGet(lua.LNumber(1.0)) == lua.LNil {
			m := make(map[string]any)
			t.ForEach(func(key, value lua.LValue) {
				m[key.String()] = LuaToGo(value)
			})
			return m
		} else {
			var arr []any
			t.ForEach(func(_, value lua.LValue) {
				arr = append(arr, LuaToGo(value))
			})
			return arr
		}
	case *lua.LNilType:
		return nil
	}

	return nil
}
