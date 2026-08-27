//go:build !windows
// +build !windows

package main

func InitRegistryFunctions() {
	if builtins == nil {
		builtins = make(map[string]BuiltinInfo)
	}

	ns := "reg."

	Register(ns+"Read", "", "", "", func(args []Value) Value {
		return StrVal("error: registry not available on this OS")
	})

	Register(ns+"Write", "", "", "", func(args []Value) Value {
		return StrVal("error: registry not available on this OS")
	})

	Register(ns+"Exists", "", "", "", func(args []Value) Value {
		return StrVal("error: registry not available on this OS")
	})

	Register(ns+"Delete", "", "", "", func(args []Value) Value {
		return StrVal("error: registry not available on this OS")
	})
	Register(ns+"WriteProtectedValue", "", "", "", func(args []Value) Value {
		return StrVal("error: registry not available on this OS")
	})
	Register(ns+"ReadProtectedValue", "", "", "", func(args []Value) Value {
		return StrVal("error: registry not available on this OS")
	})
	Register(ns+"ReadProtectedValueBytes", "", "", "", func(args []Value) Value {
		return StrVal("error: registry not available on this OS")
	})
}
