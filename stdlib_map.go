// ------------------------
// stdlib_map.go
// ------------------------

package main

import (
	"fmt"
)

// InitMapFunctions registriert Map-Funktionen
func InitMapFunctions() {
	if builtins == nil {
		builtins = make(map[string]BuiltinInfo)
	}

	ns := "map."

	// ------------------------
	// Create
	// ------------------------
	Register(ns+"Create", "map", "", "Erstellt eine leere Map", func(args []Value) Value {
		return Value{
			Kind: KindMap,
			Map:  make(map[string]Value),
		}
	})

	// ------------------------
	// Set (mutierend)
	// ------------------------
	Register(ns+"Set", "map", "map, key, value", "Setzt einen Wert in der Map (In-Place).", func(args []Value) Value {
		if len(args) < 3 || args[0].Kind != KindMap {
			return args[0]
		}

		m := args[0].Map
		key := ToString(args[1])

		m[key] = args[2]
		return args[0]
	})

	// ------------------------
	// Get
	// ------------------------
	Register(ns+"Get", "map", "map, key", "Liest einen Wert aus der Map", func(args []Value) Value {
		if len(args) < 2 || args[0].Kind != KindMap {
			return NullVal()
		}

		m := args[0].Map
		key := ToString(args[1])

		v, ok := m[key]
		if !ok {
			return NullVal()
		}

		return v
	})

	// ------------------------
	// ContainsKey
	// ------------------------
	Register(ns+"ContainsKey", "map", "map, key", "Prüft, ob ein Schlüssel existiert", func(args []Value) Value {
		if len(args) < 2 || args[0].Kind != KindMap {
			return BoolVal(false)
		}

		m := args[0].Map
		key := ToString(args[1])

		_, ok := m[key]
		return BoolVal(ok)
	})

	// ------------------------
	// Remove
	// ------------------------
	Register(ns+"Remove", "map", "map, key", "Entfernt einen Schlüssel aus der Map", func(args []Value) Value {
		if len(args) < 2 || args[0].Kind != KindMap {
			return args[0]
		}

		m := args[0].Map
		key := ToString(args[1])

		delete(m, key)
		return args[0]
	})

	// ------------------------
	// Keys
	// ------------------------
	Register(ns+"Keys", "map", "map", "Gibt alle Schlüssel als Array zurück", func(args []Value) Value {
		if len(args) < 1 || args[0].Kind != KindMap {
			return Value{Kind: KindArr, Arr: []Value{}}
		}

		m := args[0].Map
		keys := make([]Value, 0, len(m))

		for k := range m {
			keys = append(keys, StrVal(k))
		}

		return Value{Kind: KindArr, Arr: keys}
	})

	// ------------------------
	// Values
	// ------------------------
	Register(ns+"Values", "map", "map", "Gibt alle Werte als Array zurück", func(args []Value) Value {
		if len(args) < 1 || args[0].Kind != KindMap {
			return Value{Kind: KindArr, Arr: []Value{}}
		}

		m := args[0].Map
		values := make([]Value, 0, len(m))

		for _, v := range m {
			values = append(values, v)
		}

		return Value{Kind: KindArr, Arr: values}
	})

	// ------------------------
	// Count
	// ------------------------
	Register(ns+"Count", "map", "map", "Anzahl der Einträge in der Map", func(args []Value) Value {
		if len(args) < 1 || args[0].Kind != KindMap {
			return NumVal(0)
		}

		return NumVal(float64(len(args[0].Map)))
	})

	// ------------------------
	// Clear
	// ------------------------
	Register(ns+"Clear", "map", "map", "Leert die Map (mutierend)", func(args []Value) Value {
		if len(args) < 1 || args[0].Kind != KindMap {
			return args[0]
		}

		m := args[0].Map
		for k := range m {
			delete(m, k)
		}

		return args[0]
	})

	// ------------------------
	// Clone
	// ------------------------
	Register(ns+"Clone", "map", "map", "Erstellt eine Kopie der Map (shallow copy)", func(args []Value) Value {
		if len(args) < 1 || args[0].Kind != KindMap {
			return Value{}
		}

		src := args[0].Map
		dst := make(map[string]Value, len(src))

		for k, v := range src {
			dst[k] = v
		}

		return Value{
			Kind: KindMap,
			Map:  dst,
		}
	})

	// ------------------------
	// ToString (Debug)
	// ------------------------
	Register(ns+"ToString", "map", "map", "Gibt eine einfache String-Darstellung zurück", func(args []Value) Value {
		if len(args) < 1 || args[0].Kind != KindMap {
			return StrVal("{}")
		}

		m := args[0].Map
		out := "{"

		first := true
		for k, v := range m {
			if !first {
				out += ", "
			}
			first = false

			out += fmt.Sprintf("%s: %s", k, ToString(v))
		}

		out += "}"

		return StrVal(out)
	})
}
