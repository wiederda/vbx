package main

import (
	"fmt"
	"math/rand"
	"time"
)

var randGen = rand.New(rand.NewSource(time.Now().UnixNano()))

func InitRandFunctions() {
	if builtins == nil {
		builtins = make(map[string]BuiltinInfo)
	}

	ns := "rand."

	// Hilfs-Check für die Argument-Anzahl (ähnlich wie bei math)
	checkArgs := func(args []Value, min int, funcName string) Value {
		if len(args) < min {
			return ErrorVal(fmt.Sprintf("%s erwartet mindestens %d Argument(e)", funcName, min))
		}
		return Value{}
	}

	// rand.Float() -> 0.0 bis 1.0
	Register(ns+"Float", "rand", "-", "Gibt eine Zufallszahl zwischen 0.0 und 1.0 zurück.", func(args []Value) Value {
		return NumVal(randGen.Float64())
	})

	// rand.Choice(array) -> Pickt ein zufälliges Element
	Register(ns+"Choice", "rand", "array", "Wählt ein zufälliges Element aus einem Array aus.", func(args []Value) Value {
		if len(args) < 1 || args[0].Kind != KindArr {
			return ErrorVal("rand.Choice erwartet ein Array")
		}

		arr := args[0].Arr
		if len(arr) == 0 {
			return ErrorVal("Array ist leer")
		}

		idx := randGen.Intn(len(arr))
		return arr[idx]
	})

	// rand.Bool() -> true oder false
	Register(ns+"Bool", "rand", "-", "Gibt zufällig true oder false zurück.", func(args []Value) Value {
		return BoolVal(randGen.Intn(2) == 0)
	})

	// rand.Range(min, max) -> Ganzzahl von min bis max (inklusiv)
	Register(ns+"Range", "rand", "min, max", "Gibt eine Ganzzahl zwischen min und max (inklusive) zurück.", func(args []Value) Value {
		if err := checkArgs(args, 2, "rand.Range"); err.Kind == KindError {
			return err
		}

		min, err1 := getMathArg(args, 0, "rand.Range")
		if err1.Kind == KindError {
			return err1
		}
		max, err2 := getMathArg(args, 1, "rand.Range")
		if err2.Kind == KindError {
			return err2
		}

		iMin, iMax := int(min), int(max)
		if iMax <= iMin {
			return NumVal(float64(iMin))
		}

		return NumVal(float64(randGen.Intn(iMax-iMin+1) + iMin))
	})

	// rand.RangeFloat(min, max)
	Register(ns+"RangeFloat", "rand", "min, max", "Gibt eine Fließkommazahl im angegebenen Bereich zurück.", func(args []Value) Value {
		if err := checkArgs(args, 2, "rand.RangeFloat"); err.Kind == KindError {
			return err
		}

		min, err1 := getMathArg(args, 0, "rand.RangeFloat")
		if err1.Kind == KindError {
			return err1
		}
		max, err2 := getMathArg(args, 1, "rand.RangeFloat")
		if err2.Kind == KindError {
			return err2
		}

		if max <= min {
			return NumVal(min)
		}
		return NumVal(min + randGen.Float64()*(max-min))
	})

	// rand.Seed(n)
	Register(ns+"Seed", "rand", "[n]", "Initialisiert den Zufallsgenerator mit einem Startwert.", func(args []Value) Value {
		if len(args) == 0 {
			randGen = rand.New(rand.NewSource(time.Now().UnixNano()))
			return NumVal(0)
		}
		s, err := getMathArg(args, 0, "rand.Seed")
		if err.Kind == KindError {
			return err
		}

		randGen = rand.New(rand.NewSource(int64(s)))
		return NumVal(s)
	})
}
