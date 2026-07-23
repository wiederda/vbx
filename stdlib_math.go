// ------------------------
// stdlib_math.go
// ------------------------

package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Hilfsfunktion: Holt ein Argument sicher ab und wandelt es um.
// Gibt einen Fehler-Value zurück, falls das Argument fehlt oder keine Zahl ist.
func getMathArg(args []Value, index int, funcName string) (float64, Value) {
	if index < 0 || index >= len(args) {
		return 0, ErrorVal(fmt.Sprintf("%s: Fehlendes Argument an Position %d", funcName, index+1))
	}

	v := args[index]
	// Wir nutzen deine Logik aus toFloatSafe, geben aber Fehler zurück statt stiller 0
	switch v.Kind {
	case KindNum:
		return v.Num, Value{}
	case KindStr:
		s := strings.TrimSpace(v.Str)
		s = strings.ReplaceAll(s, ",", ".")
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f, Value{}
		}
	}
	return 0, ErrorVal(fmt.Sprintf("%s: Argument %d ist keine gültige Zahl", funcName, index+1))
}

func InitMathFunctions() {
	if builtins == nil {
		builtins = make(map[string]BuiltinInfo)
	}

	ns := "math."

	// Registrier-Helfer für 1 Argument (Sin, Cos, Abs, ...)
	// Jetzt mit 'desc' (Beschreibung) Parameter
	reg1 := func(name string, desc string, f func(float64) float64) {
		Register(ns+name, "math", "n", desc, func(args []Value) Value {
			val, err := getMathArg(args, 0, name)
			if err.Kind == KindError {
				return err
			}
			return NumVal(f(val))
		})
	}

	// Registrier-Helfer für 2 Argumente (Min, Max, Pow, ...)
	// Jetzt mit 'params' (Argumentnamen) und 'desc' (Beschreibung)
	reg2 := func(name string, params string, desc string, f func(float64, float64) float64) {
		Register(ns+name, "math", params, desc, func(args []Value) Value {
			a, errA := getMathArg(args, 0, name)
			if errA.Kind == KindError {
				return errA
			}
			b, errB := getMathArg(args, 1, name)
			if errB.Kind == KindError {
				return errB
			}
			return NumVal(f(a, b))
		})
	}

	// --- Standard Funktionen ---
	reg1("Abs", "Gibt den absoluten Wert (Betrag) einer Zahl zurück.", math.Abs)
	reg1("Sin", "Berechnet den Sinus (Radiant).", math.Sin)
	reg1("Cos", "Berechnet den Cosinus (Radiant).", math.Cos)
	reg1("Tan", "Berechnet den Tangens (Radiant).", math.Tan)
	reg1("Asin", "Berechnet den Arkussinus.", math.Asin)
	reg1("Acos", "Berechnet den Arkuscosinus.", math.Acos)
	reg1("Atan", "Berechnet den Arkustangens.", math.Atan)
	reg1("Ceil", "Rundet auf die nächste ganze Zahl auf.", math.Ceil)
	reg1("Floor", "Rundet auf die nächste ganze Zahl ab.", math.Floor)
	reg1("Trunc", "Schneidet die Nachkommastellen ab.", math.Trunc)
	reg1("Exp", "Berechnet die Exponentialfunktion (e^n).", math.Exp)
	reg1("Cbrt", "Berechnet die Kubikwurzel (3. Wurzel).", math.Cbrt)

	reg2("Min", "a, b", "Gibt die kleinere der beiden Zahlen zurück.", math.Min)
	reg2("Max", "a, b", "Gibt die größere der beiden Zahlen zurück.", math.Max)
	reg2("Pow", "basis, exp", "Berechnet die Potenz (Basis^Exponent).", math.Pow)
	reg2("Atan2", "y, x", "Berechnet den Arkustangens aus zwei Koordinaten.", math.Atan2)

	// math.Pi() -> 3.14159...
	Register(ns+"Pi", "math", "-", "Gibt die Konstante π zurück (ca. 3,14159).", func(args []Value) Value {
		return NumVal(math.Pi)
	})

	// math.Int(wert) -> Schneidet Nachkommastellen ab
	Register(ns+"Int", "math", "n", "Schneidet alle Nachkommastellen ab.", func(args []Value) Value {
		v, err := getMathArg(args, 0, "Int")
		if err.Kind == KindError {
			return err
		}
		// Wir nutzen math.Trunc, um Nachkommastellen einfach zu entfernen
		return NumVal(math.Trunc(v))
	})

	// math.Val(string) -> Wandelt Text in Zahl um (VB-Style: 0 bei Fehler)
	// Hinweis: Deine getMathArg ist strenger. Val ist oft als "Sicherheitsnetz" gedacht.
	Register(ns+"Val", "math", "s", "Wandelt Text in Zahl um (gibt 0 zurück, falls kein Text).", func(args []Value) Value {
		if len(args) < 1 {
			return NumVal(0)
		}

		// Wir versuchen die Wandlung ohne das Skript abzubrechen
		s := strings.TrimSpace(args[0].Str)
		s = strings.ReplaceAll(s, ",", ".")
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return NumVal(f)
		}
		return NumVal(0) // VB-Standard: Wenn keine Zahl erkennbar, dann 0
	})

	// math.E() -> 2.71828...
	Register(ns+"E", "math", "-", "Gibt die Eulersche Zahl zurück (ca. 2,71828).", func(args []Value) Value {
		return NumVal(math.E)
	})

	Register(ns+"Mod", "math", "a, b", "Berechnet den Rest der Ganzzahl-Division (Modulo).", func(args []Value) Value {
		a, e1 := getMathArg(args, 0, "Mod")
		if e1.Kind == KindError {
			return e1
		}
		b, e2 := getMathArg(args, 1, "Mod")
		if e2.Kind == KindError {
			return e2
		}

		if b == 0 {
			return ErrorVal("Modulo durch 0")
		}
		return NumVal(float64(int(a) % int(b)))
	})

	Register(ns+"LogBase", "math", "n, base", "Berechnet den Logarithmus von n zur beliebigen Basis.", func(args []Value) Value {
		v, e1 := getMathArg(args, 0, "LogBase")
		base, e2 := getMathArg(args, 1, "LogBase")
		if e1.Kind == KindError {
			return e1
		}
		if e2.Kind == KindError {
			return e2
		}
		if v <= 0 || base <= 0 || base == 1 {
			return ErrorVal("LogBase: Ungültige Eingabewerte")
		}
		return NumVal(math.Log(v) / math.Log(base))
	})

	// --- Finanzmathematik (Excel/VBA Style) ---

	Register(ns+"Sqrt", "math", "n", "Berechnet die Quadratwurzel.", func(args []Value) Value {
		v, err := getMathArg(args, 0, "Sqrt")
		if err.Kind == KindError {
			return err
		}
		if v < 0 {
			return ErrorVal("Sqrt: Wert darf nicht negativ sein")
		}
		return NumVal(math.Sqrt(v))
	})

	// Root(wert, n) -> Die n-te Wurzel
	Register(ns+"Root", "math", "v, n", "Berechnet die n-te Wurzel von v.", func(args []Value) Value {
		v, e1 := getMathArg(args, 0, "Root")
		if e1.Kind == KindError {
			return e1
		}
		n, e2 := getMathArg(args, 1, "Root")
		if e2.Kind == KindError {
			return e2
		}

		if n == 0 {
			return ErrorVal("Root: Die 0-te Wurzel ist nicht definiert")
		}
		// Mathematisch: n-te Wurzel von v = v^(1/n)
		return NumVal(math.Pow(v, 1.0/n))
	})

	Register(ns+"Sign", "math", "n", "Gibt das Vorzeichen zurück (-1, 0 oder 1).", func(args []Value) Value {
		v, err := getMathArg(args, 0, "Sign")
		if err.Kind == KindError {
			return err
		}
		if v < 0 {
			return NumVal(-1)
		}
		if v > 0 {
			return NumVal(1)
		}
		return NumVal(0)
	})

	Register(ns+"Round", "math", "v, [d]", "Rundet auf d Nachkommastellen (kaufmännisch).", func(args []Value) Value {
		val, err := getMathArg(args, 0, "Round")
		if err.Kind == KindError {
			return err
		}
		decimals := 0.0
		if len(args) > 1 {
			d, errD := getMathArg(args, 1, "Round")
			if errD.Kind == KindError {
				return errD
			}
			decimals = d
		}
		pow := math.Pow(10, math.Trunc(decimals))
		return NumVal(math.Round(val*pow) / pow)
	})

	Register(ns+"RoundBank", "math", "v, [d]", "Rundet auf die nächste gerade Zahl (Banker's Rounding).", func(args []Value) Value {
		val, err := getMathArg(args, 0, "RoundBank")
		if err.Kind == KindError {
			return err
		}

		decimals := 0.0
		if len(args) > 1 {
			d, errD := getMathArg(args, 1, "RoundBank")
			if errD.Kind == KindError {
				return errD
			}
			decimals = d
		}

		pow := math.Pow(10, math.Trunc(decimals))
		return NumVal(math.RoundToEven(val*pow) / pow)
	})

	Register(ns+"Clamp", "math", "v, min, max", "Hält den Wert v innerhalb der Grenzen min und max.", func(args []Value) Value {
		v, e1 := getMathArg(args, 0, "Clamp")
		if e1.Kind == KindError {
			return e1
		}
		low, e2 := getMathArg(args, 1, "Clamp")
		if e2.Kind == KindError {
			return e2
		}
		high, e3 := getMathArg(args, 2, "Clamp")
		if e3.Kind == KindError {
			return e3
		}
		return NumVal(math.Max(low, math.Min(v, high)))
	})

	Register(ns+"Log", "math", "n", "Berechnet den natürlichen Logarithmus.", func(args []Value) Value {
		v, err := getMathArg(args, 0, "Log")
		if err.Kind == KindError {
			return err
		}
		if v <= 0 {
			return ErrorVal("Log nur für Werte > 0")
		}
		return NumVal(math.Log(v))
	})

	Register(ns+"DegToRad", "math", "deg", "Rechnet Grad in Bogenmaß (Radiant) um.", func(args []Value) Value {
		v, err := getMathArg(args, 0, "DegToRad")
		if err.Kind == KindError {
			return err
		}
		return NumVal(v * math.Pi / 180)
	})

	Register(ns+"RadToDeg", "math", "rad", "Rechnet Bogenmaß (Radiant) in Grad um.", func(args []Value) Value {
		v, err := getMathArg(args, 0, "RadToDeg")
		if err.Kind == KindError {
			return err
		}
		return NumVal(v * 180 / math.Pi)
	})

	Register(ns+"Percent", "math", "p, v", "Berechnet p Prozent von v.", func(args []Value) Value {
		p, e1 := getMathArg(args, 0, "Percent")
		if e1.Kind == KindError {
			return e1
		}
		v, e2 := getMathArg(args, 1, "Percent")
		if e2.Kind == KindError {
			return e2
		}
		return NumVal((p / 100) * v)
	})

	Register(ns+"PercentOf", "math", "part, total", "Berechnet, wie viel Prozent part von total sind.", func(args []Value) Value {
		part, e1 := getMathArg(args, 0, "PercentOf")
		if e1.Kind == KindError {
			return e1
		}
		total, e2 := getMathArg(args, 1, "PercentOf")
		if e2.Kind == KindError {
			return e2
		}
		if total == 0 {
			return ErrorVal("Division durch 0 in PercentOf")
		}
		return NumVal((part / total) * 100)
	})
}
