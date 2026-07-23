package main

import "os"

func InitEnvFunctions() {
	if builtins == nil {
		builtins = make(map[string]BuiltinInfo)
	}

	ns := "env."

	Register(ns+"Get", "env", "key", "Liest eine Umgebungsvariable (z.B. 'PATH' oder 'TEMP') aus.", func(args []Value) Value {
		if len(args) < 1 {
			return ErrorVal("env.Get(key) benötigt einen Namen")
		}
		key := args[0].Str
		val := os.Getenv(key)
		return StrVal(val)
	})

	Register(ns+"Set", "env", "key, value", "Setzt eine Umgebungsvariable für den aktuellen Prozess. Gibt True bei Erfolg zurück.", func(args []Value) Value {
		if len(args) < 2 {
			return ErrorVal("env.Set(key, value) benötigt zwei Argumente")
		}
		err := os.Setenv(args[0].Str, args[1].Str)
		if err != nil {
			return ErrorVal("Fehler beim Setzen der Env-Variable: " + err.Error())
		}
		return BoolVal(true)
	})

	Register(ns+"All", "env", "-", "Gibt alle verfügbaren Umgebungsvariablen als Array im Format 'KEY=VALUE' zurück.", func(args []Value) Value {
		vars := os.Environ()
		return Value{Kind: KindArr, Arr: stringSliceToValueSlice(vars)}
	})
}
