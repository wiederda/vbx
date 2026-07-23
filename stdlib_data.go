// ------------------------
// stdlib_data.go
// ------------------------

package main

import (
	"fmt"
	"time"
)

var timers = make(map[string]time.Time)

func InitDataFunctions() {
	if builtins == nil {
		builtins = make(map[string]BuiltinInfo)
	}

	ns := "data."

	// Hilfsfunktion für den Zugriff auf die Faktoren aus convert.Unit
	// oder direktes Rechnen für maximale Performance

	// --- DATENMENGEN (SI 1000er) ---
	Register(ns+"ByteToKb", "data", "val", "Konvertiert Byte in Kilobyte (1000).", func(args []Value) Value {
		return NumVal(toFloatSafe(args, 0) / 1000.0)
	})
	Register(ns+"KbToMb", "data", "val", "Konvertiert Kilobyte in Megabyte (1000).", func(args []Value) Value {
		return NumVal(toFloatSafe(args, 0) / 1000.0)
	})
	Register(ns+"MbToGb", "data", "val", "Konvertiert Megabyte in Gigabyte (1000).", func(args []Value) Value {
		return NumVal(toFloatSafe(args, 0) / 1000.0)
	})
	Register(ns+"GbToTb", "data", "val", "Konvertiert Gigabyte in Terabyte (1000).", func(args []Value) Value {
		return NumVal(toFloatSafe(args, 0) / 1000.0)
	})

	// --- DATENMENGEN (Binär 1024er) ---
	Register(ns+"ByteToKiB", "data", "val", "Konvertiert Byte in Kibibyte (1024).", func(args []Value) Value {
		return NumVal(toFloatSafe(args, 0) / 1024.0)
	})
	Register(ns+"MiBToByte", "data", "val", "Konvertiert Mebibyte in Byte (1024^2).", func(args []Value) Value {
		return NumVal(toFloatSafe(args, 0) * 1024.0 * 1024.0)
	})

	// --- LEISTUNG ---
	Register(ns+"WattToKilowatt", "data", "val", "Konvertiert Watt in Kilowatt.", func(args []Value) Value {
		return NumVal(toFloatSafe(args, 0) / 1000.0)
	})

	Register(ns+"KilowattToWatt", "data", "val", "Konvertiert Kilowatt in Watt.", func(args []Value) Value {
		return NumVal(toFloatSafe(args, 0) * 1000.0)
	})

	// --- ZEIT (Wichtig für SQL-Laufzeiten) ---
	Register(ns+"MinutesToHours", "data", "val", "Konvertiert Minuten in Stunden.", func(args []Value) Value {
		return NumVal(toFloatSafe(args, 0) / 60.0)
	})
	Register(ns+"SecondsToDays", "data", "val", "Konvertiert Sekunden in Tage.", func(args []Value) Value {
		return NumVal(toFloatSafe(args, 0) / 86400.0)
	})

	Register(ns+"HoursToMinutes", "data", "val", "Konvertiert Stunden in Minuten.", func(args []Value) Value {
		return NumVal(toFloatSafe(args, 0) * 60.0)
	})

	Register(ns+"DaysToSeconds", "data", "val", "Konvertiert Tage in Sekunden.", func(args []Value) Value {
		return NumVal(toFloatSafe(args, 0) * 86400.0)
	})

	// NEU: Ein kleiner Formatierer für Log-Ausgaben
	Register(ns+"FormatSeconds", "data", "seconds", "Formatiert Sekunden in HH:MM:SS.", func(args []Value) Value {
		s := int(toFloatSafe(args, 0))
		h := s / 3600
		m := (s % 3600) / 60
		sec := s % 60
		return StrVal(fmt.Sprintf("%02d:%02d:%02d", h, m, sec))
	})

	// --- NEU: Stoppuhr für Performance-Messung ---
	Register(ns+"TimerStart", "data", "[name]", "Startet eine Zeitmessung.", func(args []Value) Value {
		name := "default"
		if len(args) > 0 {
			name = args[0].Str
		}
		timers[name] = time.Now()
		return BoolVal(true)
	})

	Register(ns+"TimerElapsed", "data", "[name]", "Gibt Sekunden seit Start zurück.", func(args []Value) Value {
		name := "default"
		if len(args) > 0 {
			name = args[0].Str
		}
		if start, ok := timers[name]; ok {
			return NumVal(time.Since(start).Seconds())
		}
		return NumVal(0)
	})
}
