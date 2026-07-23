//go:build !windows

package main

import "fmt"

func InitWinFunctions() {
	if builtins == nil {
		builtins = make(map[string]BuiltinInfo)
	}

	ns := "win."

	// API existiert, liefert aber leere Werte
	Register(ns+"GetActiveTitle", "", "", "", func(args []Value) Value {
		return StrVal("")
	})
}

func registerEventModule(builtins map[string]func([]Value) Value, ns string) {

	// Eine kleine Hilfsfunktion für die Fehlermeldung
	notSupported := func(args []Value) Value {
		fmt.Println("\033[31m[ERROR]\033[0m Das 'event'-Modul ist nur unter Windows verfügbar.")
		fmt.Println("\033[90mLinux/macOS nutzen kein Windows Event Log.\033[0m")
		return BoolVal(false)
	}

	// Wir registrieren die Funktionen trotzdem, damit das Skript nicht mit
	// "Function not found" abstürzt, sondern kontrolliert meldet:
	builtins[ns+"GetEvent"] = notSupported
	builtins[ns+"SearchEventLog"] = notSupported
	builtins[ns+"FindEventID"] = notSupported
}

// Wird für Main gebraucht
func enableWindowsANSI() {
	// Auf Linux/macOS ist nichts zu tun
}
