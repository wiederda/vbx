package main

import (
	"fmt"
	"strconv"
	"strings"
)

func InitConvertFunctions() {
	if builtins == nil {
		builtins = make(map[string]BuiltinInfo)
	}

	ns := "convert."

	// Kategorien für Faktor-basierte Einheiten
	categories := map[string]map[string]float64{
		"length": {
			"mm": 0.001, "cm": 0.01, "m": 1.0, "meter": 1.0, "km": 1000.0,
			"inch": 0.0254, "ft": 0.3048, "foot": 0.3048, "yard": 0.9144, "mile": 1609.344,
		},
		"weight": {
			"mg": 0.001, "g": 1.0, "gram": 1.0, "kg": 1000.0, "t": 1000000.0,
			"oz": 28.34952, "lb": 453.592, "pound": 453.592,
		},
		"volume": {
			"ml": 0.001, "l": 1.0, "liter": 1.0,
			"gal": 3.78541, "gallon": 3.78541,
		},
		"speed": {
			"kmh": 1.0, "mph": 1.609344,
		},
	}

	Register(ns+"Unit", "convert", "category, from, to, value", "Konvertiert Einheiten basierend auf Kategorien (Länge, Gewicht, Temp, etc.)", func(args []Value) Value {
		if len(args) < 4 {
			return ErrorVal("usage: convert.Unit(category, from, to, value)")
		}

		cat := strings.ToLower(ToString(args[0]))
		from := strings.ToUpper(ToString(args[1])) // Temp braucht oft Großbuchstaben (C, F, K)
		to := strings.ToUpper(ToString(args[2]))
		val := toNumVal(args[3])

		// --- SONDERFALL: TEMPERATUR (Intervallskala) ---
		if cat == "temp" || cat == "temperature" {
			// 1. Nach Kelvin
			var k float64
			switch from {
			case "C":
				k = val + 273.15
			case "F":
				k = (val-32.0)*5.0/9.0 + 273.15
			case "K":
				k = val
			case "R":
				k = val * 5.0 / 9.0
			default:
				return ErrorVal("unknown temp unit: " + from)
			}

			if k < 0 {
				return ErrorVal("temperature below absolute zero")
			}

			// 2. Von Kelvin zum Ziel
			var res float64
			switch to {
			case "C":
				res = k - 273.15
			case "F":
				res = (k-273.15)*9.0/5.0 + 32.0
			case "K":
				res = k
			case "R":
				res = k * 9.0 / 5.0
			default:
				return ErrorVal("unknown temp unit: " + to)
			}
			return NumVal(res)
		}

		// --- NORMALFALL: FAKTOR-BASIERT (Verhältnisskala) ---
		units, ok := categories[cat]
		if !ok {
			return ErrorVal("unknown category: " + cat)
		}

		fFactor, ok1 := units[strings.ToLower(from)]
		tFactor, ok2 := units[strings.ToLower(to)]

		if !ok1 || !ok2 {
			return ErrorVal("unit not found in category " + cat)
		}

		return NumVal((val * fFactor) / tFactor)
	})

	Register(ns+"Categories", "convert", "-", "Gibt alle Unit-Kategorien zurück", func(args []Value) Value {
		var res []Value
		for k := range categoryRegistry {
			res = append(res, StrVal(k))
		}
		return ArrVal(res)
	})

	Register(ns+"Units", "convert", "category", "Gibt alle Units einer Kategorie zurück", func(args []Value) Value {
		if len(args) < 1 {
			return ErrorVal("usage: convert.Units(category)")
		}

		cat := strings.ToLower(ToString(args[0]))
		units, ok := categories[cat]
		if !ok {
			return ErrorVal("unknown category: " + cat)
		}

		var res []Value
		for u := range units {
			res = append(res, StrVal(u))
		}
		return ArrVal(res)
	})

	// --- NEU: Hexadezimal / Dezimal ---
	Register(ns+"HexToDec", "convert", "hexStr", "Wandelt Hex (0x...) in Dezimal um.", func(args []Value) Value {
		if len(args) < 1 {
			return NumVal(0)
		}
		s := strings.TrimPrefix(strings.ToLower(args[0].Str), "0x")
		val, err := strconv.ParseInt(s, 16, 64)
		if err != nil {
			return ErrorVal("invalid hex: " + err.Error())
		}
		return NumVal(float64(val))
	})

	Register(ns+"DecToHex", "convert", "val", "Wandelt Dezimal in Hex um.", func(args []Value) Value {
		v := int64(toNumVal(args[0]))
		return StrVal(fmt.Sprintf("0x%X", v))
	})
}

type CategoryInfo struct {
	Name        string
	Type        string // "factor" oder "special"
	Description string
}

var categoryRegistry = map[string]CategoryInfo{
	"length": {"length", "factor", "Längeneinheiten"},
	"weight": {"weight", "factor", "Gewichtseinheiten"},
	"volume": {"volume", "factor", "Volumen"},
	"speed":  {"speed", "factor", "Geschwindigkeit"},
	"temp":   {"temp", "special", "Temperatur"},
}
