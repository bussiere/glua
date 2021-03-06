package main

import (
	"github.com/gopherjs/gopherjs/js"
	lua "github.com/yuin/gopher-lua"
	"honnef.co/go/js/console"
)

func main() {
	run := func(code string) {
		L := lua.NewState()
		defer L.Close()
		if err := L.DoString(code); err != nil {
			console.Error("error during Lua script execution", err.Error())
		}
	}
	withGlobals := func(globals map[string]interface{}, code string) {
		L := lua.NewState()
		defer L.Close()

		for name, value := range globals {
			L.SetGlobal(name, lvalueFromInterface(L, value))
		}

		if err := L.DoString(code); err != nil {
			console.Error("error during Lua script execution", err.Error())
		}
	}

	if js.Module != js.Undefined {
		js.Module.Get("exports").Set("run", run)
		js.Module.Get("exports").Set("runWithGlobals", withGlobals)
	} else {
		js.Global.Set("glua", map[string]interface{}{
			"run":            run,
			"runWithGlobals": withGlobals,
		})
	}
}

func lvalueFromInterface(L *lua.LState, value interface{}) lua.LValue {
	switch val := value.(type) {
	case string:
		return lua.LString(val)
	case float64:
		return lua.LNumber(val)
	case bool:
		return lua.LBool(val)
	case map[string]interface{}:
		table := L.NewTable()
		for k, iv := range val {
			table.RawSetString(k, lvalueFromInterface(L, iv))
		}
		return table
	case func(...interface{}) *js.Object:
		fn := val
		return L.NewFunction(func(L *lua.LState) int {
			var args []interface{}

			for a := 1; ; a++ {
				arg := L.Get(a)
				if arg == lua.LNil {
					break
				}
				args = append(args, lvalueToInterface(arg))
			}

			jsreturn := fn(args...)

			if jsreturn == js.Undefined {
				return 0
			}

			L.Push(lvalueFromInterface(L, jsreturn.Interface()))
			return 1
		})
	default:
		return lua.LNil
	}
}

func lvalueToInterface(lvalue lua.LValue) interface{} {
	switch value := lvalue.(type) {
	case *lua.LTable:
		m := make(map[string]interface{}, value.Len())
		value.ForEach(func(k lua.LValue, v lua.LValue) {
			m[lua.LVAsString(k)] = lvalueToInterface(v)
		})
		return m
	case lua.LNumber:
		return float64(value)
	case lua.LString:
		return string(value)
	default:
		switch lvalue {
		case lua.LTrue:
			return true
		case lua.LFalse:
			return false
		case lua.LNil:
			return nil
		}
	}
	return nil
}
