//go:build windows
// +build windows

package main

import (
	"golang.org/x/sys/windows/registry"
)

func regRoot(root string) registry.Key {
	switch root {
	case "HKCU":
		return registry.CURRENT_USER
	case "HKLM":
		return registry.LOCAL_MACHINE
	default:
		return registry.CURRENT_USER
	}
}

// löscht einen Registry-Key rekursiv inkl. Unterkeys
func deleteKeyRecursive(root registry.Key, path string) error {
	k, err := registry.OpenKey(root, path, registry.ENUMERATE_SUB_KEYS|registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	// alle Unterkeys löschen
	subkeys, err := k.ReadSubKeyNames(-1)
	if err != nil {
		return err
	}
	for _, sub := range subkeys {
		if err := deleteKeyRecursive(k, sub); err != nil {
			return err
		}
	}

	// Key selbst löschen (muss nach dem Schließen passieren)
	k.Close()
	return registry.DeleteKey(root, path)
}

func InitRegistryFunctions() {
	if builtins == nil {
		builtins = make(map[string]BuiltinInfo)
	}

	ns := "reg."

	Register(ns+"Read", "reg", "root, path, name",
		"Liest einen String-Wert aus der Registry. 'root' kann 'HKCU' oder 'HKLM' sein.", func(args []Value) Value {
			if len(args) < 3 {
				return ErrorVal("reg.Read erwartet 3 Argumente")
			}
			root := regRoot(args[0].Str)
			keyPath := args[1].Str
			valueName := args[2].Str

			k, err := registry.OpenKey(root, keyPath, registry.QUERY_VALUE)
			if err != nil {
				return ErrorVal(err.Error())
			}
			defer k.Close()

			val, _, err := k.GetStringValue(valueName)
			if err != nil {
				return ErrorVal(err.Error())
			}
			return StrVal(val)
		})

	Register(ns+"Write", "reg", "root, path, name, data",
		"Schreibt einen String-Wert in die Registry. Erstellt den Key automatisch, falls er nicht existiert.", func(args []Value) Value {
			if len(args) < 4 {
				return ErrorVal("reg.Write erwartet 4 Argumente")
			}
			root := regRoot(args[0].Str)
			keyPath := args[1].Str
			valueName := args[2].Str
			valueData := args[3].Str

			k, _, err := registry.CreateKey(root, keyPath, registry.SET_VALUE)
			if err != nil {
				return ErrorVal(err.Error())
			}
			defer k.Close()

			err = k.SetStringValue(valueName, valueData)
			if err != nil {
				return ErrorVal(err.Error())
			}
			return StrVal("OK")
		})

	// reg.Exists
	Register(ns+"Exists", "reg", "root, path [, name]",
		"Prüft, ob ein Registry-Key oder ein spezifischer Wert existiert. Gibt true oder false zurück.", func(args []Value) Value {

			if len(args) < 2 || len(args) > 3 {
				return ErrorVal("reg.Exists erwartet 2 oder 3 Argumente")
			}

			if args[0].Kind != KindStr || args[1].Kind != KindStr {
				return ErrorVal("reg.Exists erwartet String-Argumente")
			}

			root := regRoot(args[0].Str)
			keyPath := args[1].Str

			// Key prüfen
			if len(args) == 2 {
				k, err := registry.OpenKey(root, keyPath, registry.QUERY_VALUE)
				if err != nil {
					return BoolVal(false)
				}
				k.Close()
				return BoolVal(true)
			}

			// Value prüfen
			if args[2].Kind != KindStr {
				return ErrorVal("ValueName muss String sein")
			}

			valueName := args[2].Str

			k, err := registry.OpenKey(root, keyPath, registry.QUERY_VALUE)
			if err != nil {
				return BoolVal(false)
			}
			defer k.Close()

			_, _, err = k.GetValue(valueName, nil)
			return BoolVal(err == nil)
		})

	Register(ns+"Delete", "reg", "root, path [, name]",
		"Löscht einen Wert oder einen ganzen Key rekursiv (inkl. aller Unterkeys!). Ohne 'name' wird der gesamte Pfad gelöscht. Vorsicht geboten.", func(args []Value) Value {

			if len(args) < 2 || len(args) > 3 {
				return ErrorVal("reg.Delete erwartet 2 oder 3 Argumente")
			}

			root := regRoot(args[0].Str)
			keyPath := args[1].Str

			// --- reg.Delete(root, keyPath) ---
			if len(args) == 2 {
				err := deleteKeyRecursive(root, keyPath)
				if err != nil {
					return ErrorVal(err.Error())
				}
				return StrVal("OK")
			}

			// --- reg.Delete(root, keyPath, valueName) ---
			valueName := args[2].Str

			k, err := registry.OpenKey(root, keyPath, registry.SET_VALUE)
			if err != nil {
				return ErrorVal(err.Error())
			}
			defer k.Close()

			err = k.DeleteValue(valueName)
			if err != nil {
				return ErrorVal(err.Error())
			}

			return StrVal("OK")
		})
}
