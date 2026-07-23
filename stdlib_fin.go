package main

import "math"

func InitFinFunctions() {
	if builtins == nil {
		builtins = make(map[string]BuiltinInfo)
	}

	ns := "fin."

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
		Register(ns+name, "fin", params, desc, func(args []Value) Value {
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

	Register(ns+"Fv", "fin", "rate, nper, pmt, [pv]", "Berechnet den Endwert einer Investition.", func(args []Value) Value {
		rate, _ := getMathArg(args, 0, "Fv")
		nper, _ := getMathArg(args, 1, "Fv")
		pmt, _ := getMathArg(args, 2, "Fv")
		pv := 0.0
		if len(args) > 3 {
			p, _ := getMathArg(args, 3, "Fv")
			pv = p
		}
		if rate == 0 {
			return NumVal(-(pv + pmt*nper))
		}
		term := math.Pow(1+rate, nper)
		return NumVal(-(pv*term + pmt*(term-1)/rate))
	})

	Register(ns+"Pmt", "fin", "rate, nper, pv", "Berechnet die periodische Rate eines Kredits.", func(args []Value) Value {
		rate, e1 := getMathArg(args, 0, "Pmt")
		if e1.Kind == KindError {
			return e1
		}
		nper, e2 := getMathArg(args, 1, "Pmt")
		if e2.Kind == KindError {
			return e2
		}
		pv, e3 := getMathArg(args, 2, "Pmt")
		if e3.Kind == KindError {
			return e3
		}

		if rate == 0 {
			return NumVal(-pv / nper)
		}
		pv = -pv
		return NumVal((rate * pv) / (1 - math.Pow(1+rate, -nper)))
	})

	Register(ns+"Npv", "fin", "rate, values", "Berechnet den Kapitalwert (Net Present Value).", func(args []Value) Value {
		rate, e1 := getMathArg(args, 0, "Npv")
		if e1.Kind == KindError {
			return e1
		}
		if len(args) < 2 || args[1].Kind != KindArr {
			return ErrorVal("Npv: Zweites Argument muss ein Array sein")
		}
		cashflows := args[1].Arr
		npv := 0.0
		for i, val := range cashflows {
			if val.Kind == KindNum {
				npv += val.Num / math.Pow(1+rate, float64(i+1))
			}
		}
		return NumVal(npv)
	})

	Register(ns+"Irr", "fin", "values, [guess]", "Berechnet den Internen Zinsfuß.", func(args []Value) Value {
		if len(args) < 1 || args[0].Kind != KindArr {
			return ErrorVal("Irr: Array erforderlich")
		}
		cashflows := args[0].Arr
		guess := 0.1
		if len(args) > 1 {
			if g, err := getMathArg(args, 1, "Irr"); err.Kind != KindError {
				guess = g
			}
		}
		rate := guess
		for i := 0; i < 100; i++ {
			npv, dnpv := 0.0, 0.0
			for t, val := range cashflows {
				if val.Kind == KindNum {
					t_f := float64(t)
					npv += val.Num / math.Pow(1+rate, t_f)
					if t > 0 {
						dnpv -= t_f * val.Num / math.Pow(1+rate, t_f+1)
					}
				}
			}
			if math.Abs(npv) < 1e-7 {
				return NumVal(rate)
			}
			if dnpv == 0 {
				break
			}
			newRate := rate - npv/dnpv
			if math.Abs(newRate-rate) < 1e-7 {
				return NumVal(newRate)
			}
			rate = newRate
		}
		return ErrorVal("Irr: Konvergiert nicht")
	})

	// --- Wissenschaftliche Register ---

	Register(ns+"Fact", "fin", "n", "Berechnet die Fakultät einer Ganzzahl (n!).", func(args []Value) Value {
		v, err := getMathArg(args, 0, "Fact")
		if err.Kind == KindError {
			return err
		}
		n := int64(v)
		if n < 0 {
			return ErrorVal("Fakultät nicht für negative Zahlen")
		}
		res := 1.0
		for i := int64(2); i <= n; i++ {
			res *= float64(i)
		}
		return NumVal(res)
	})

	reg1("Gamma", "Gibt den Wert der Gamma-Funktion zurück.", math.Gamma)
	reg1("Log10", "Berechnet den Zehnerlogarithmus.", math.Log10)
	reg1("Log2", "Berechnet den Logarithmus zur Basis 2.", math.Log2)
	reg2("Hypot", "x, y", "Berechnet die Länge der Hypotenuse.", math.Hypot)
	reg2("Remainder", "x, y", "Berechnet den Rest nach IEEE 754.", math.Remainder)

}
