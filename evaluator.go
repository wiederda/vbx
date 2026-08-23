package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// ---------------- Evaluator / Value ----------------

type Kind int

const (
	KindInvalid Kind = iota
	KindUnknown
	KindUndefined
	KindNum
	KindStr
	KindArr
	KindArr2D
	KindObj
	KindNull
	KindBool
	KindError
	KindMap
	KindNil
	KindNone
)

type Signal int

const (
	SignalNone Signal = iota
	SignalExitLoop
	SignalExitSub
	SignalExitFunction
	SignalReturn
	SignalError
)

type ExecResult struct {
	Signal Signal
	Value  Value
}

type Object struct {
	TypeName string
	Fields   map[string]Value
}

type Value struct {
	Kind  Kind
	Num   float64
	Str   string
	Bool  bool
	Arr   []Value
	Arr2D [][]Value
	Obj   *Object
	Map   map[string]Value // <-- neu
}

type CaseBranch struct {
	Conditions []Expr // Für CASE 1, 2, 3
	Body       []Stmt
}

type SelectNode struct {
	Expression Expr
	Cases      []CaseBranch
	Default    []Stmt
}

var NilValue = Value{Kind: KindNil}

// GetRef liefert einen Pointer auf den Wert in der Map
type Environment struct {
	vars        map[string]*Value
	parent      *Environment
	currentLine int
	currentFile string
}

func calculateBinaryOp(op TokenType, l Value, r Value) Value {
	switch op {

	case PLUS:
		return NumVal(l.Num + r.Num)

	case MINUS:
		return NumVal(l.Num - r.Num)

	case MUL:
		return NumVal(l.Num * r.Num)

	case DIV:
		if r.Num == 0 {
			return ErrorVal("division by zero")
		}
		return NumVal(l.Num / r.Num)
	}

	return ErrorVal("unsupported operator")
}

func mapAssignOp(op TokenType) TokenType {
	switch op {
	case PLUS_ASSIGN:
		return PLUS
	case MINUS_ASSIGN:
		return MINUS
	case MUL_ASSIGN:
		return MUL
	case DIV_ASSIGN:
		return DIV
	default:
		panic("invalid compound operator")
	}
}

// Normales Setzen einer Variable
func (e *Environment) Set(name string, val Value) {
	// 1. Schau, ob die Variable HIER existiert -> Dann Update
	if ptr, ok := e.vars[name]; ok {
		*ptr = val
		return
	}

	// 2. Schau, ob sie beim VATER existiert (Rekursion nach oben) -> Dann dort Update
	if e.parent != nil {
		e.parent.Set(name, val)
		return
	}

	// 3. Wenn sie nirgendwo existiert -> Erstelle sie HIER (lokal)
	copyVal := val
	e.vars[name] = &copyVal
}

func (e *Environment) Get(name string) (Value, bool) {
	// 1. Schau lokal nach
	if ptr, ok := e.vars[name]; ok {
		return *ptr, true
	}
	// 2. Wenn nicht lokal, frag den Parent (Global)
	if e.parent != nil {
		return e.parent.Get(name)
	}
	// 3. Wenn nirgendwo gefunden
	return NilValue, false
}

func (e *Environment) GetRef(name string) *Value {
	if ptr, ok := e.vars[name]; ok {
		return ptr
	}
	// Nur für interne Schleifenvariablen (FOR i = ...) gedacht!
	// Nicht für Nutzervariablen verwenden.
	newVal := &Value{Kind: KindNum, Num: 0}
	e.vars[name] = newVal
	return newVal
}

func (e *Environment) PutInternal(name string, val Value) {
	// Einfach reinwerfen, ohne "exists"-Check und ohne Printf
	e.vars[name] = &val
}

// Define erstellt IMMER eine Variable im aktuellen Scope (für DIM)
func (e *Environment) Define(name string, val Value, inLoop bool) {
	// ANSI Farbcodes
	const (
		colorYellow = "\033[33m"
		colorCyan   = "\033[36m"
		colorReset  = "\033[0m"
	)

	_, existsLocally := e.vars[name]

	if existsLocally {
		if inLoop {
			// Freifahrtschein: Wert in der Schleife einfach aktualisieren
			e.vars[name] = &val
			return
		}
		// Klassischer Doppel-Dim außerhalb einer Schleife -> GELB
		fmt.Printf("%s![HINWEIS]: Variable '%s' bereits deklariert.%s\n", colorYellow, name, colorReset)
	} else {
		// Optional: Falls die Variable in einem äußeren Scope existiert (Shadowing)
		if e.parent != nil {
			if _, existsAbove := e.parent.Get(name); existsAbove {
				// Einmalige Info in Cyan beim ersten Schleifendurchlauf
				fmt.Printf("%s![INFO]: Variable '%s' wird im lokalen Scope verwendet (Shadowing).%s\n", colorCyan, name, colorReset)
			}
		}
	}

	// Wert setzen/speichern
	e.vars[name] = &val
}

// Hilfsfunktion zur Index-Prüfung (vermeidet Code-Doppelung)
// Hilfsfunktion zur Index-Prüfung (angepasst an deinen Typ *Environment)
func checkArrayIndex(expr Expr, env *Environment, limit int, name string) (int, Value) {
	val := evalExpr(expr, env) // evalExpr erwartet sicher auch *Environment
	if val.Kind == KindError {
		return 0, val
	}

	// Konvertiert den Index zu einer Zahl
	idx := int(toNumVal(val))

	if idx < 0 || idx >= limit {
		return 0, ErrorVal(fmt.Sprintf("Index out of bounds: %s (Index %d bei Größe %d)", name, idx, limit))
	}
	return idx, NilValue
}

// SetGlobal wandert zum obersten Parent und speichert dort (für PUBLIC)
func (e *Environment) SetGlobal(name string, val Value) {
	curr := e
	for curr.parent != nil {
		curr = curr.parent
	}
	curr.vars[name] = &val
}

// Update sucht von innen nach außen und überschreibt (für x = 10)
func (e *Environment) Update(name string, val Value) error {
	if ptr, ok := e.vars[name]; ok {
		*ptr = val
		return nil
	}
	if e.parent != nil {
		return e.parent.Update(name, val)
	}
	return fmt.Errorf("Variable '%s' nicht gefunden. Nutze DIM oder PUBLIC.", name)
}

func NewEnvironment(parent *Environment) *Environment {
	return &Environment{
		vars:   make(map[string]*Value),
		parent: parent, // Wichtig für die Scope-Kette!
	}
}

func (n *SelectNode) stmtNode() {}

// ExitNode repräsentiert Anweisungen wie "Exit For", "Exit While" oder "Exit Do".
type ExitNode struct {
	// Wir speichern den Typ, falls wir später prüfen wollen,
	// ob ein "Exit For" fälschlicherweise in einer "While"-Schleife steht.
	ExitType string
}

func evalFunctionCall(name string, args []Expr, env *Environment) Value {
	// 1. Array-Zugriff prüfen
	// ACHTUNG: Get liefert jetzt (Value, bool)
	v, found := env.Get(name)
	if found && (v.Kind == KindArr || v.Kind == KindArr2D || v.Kind == KindMap || v.Kind == KindObj) {

		if v.Kind == KindObj && len(args) == 1 {

			keyVal := evalExpr(args[0], env)
			key := ToString(keyVal)

			m := v.Obj.Fields

			if val, ok := m[key]; ok {
				return val
			}

			return NilValue
		}

		if v.Kind == KindMap && len(args) == 1 {
			keyVal := evalExpr(args[0], env)
			if keyVal.Kind == KindError {
				return keyVal
			}

			key := ToString(keyVal)

			if val, ok := v.Map[key]; ok {
				return val
			}

			return NilValue
		}

		// --- FALL A: 1D-Zugriff oder Zeilen-Zugriff ---
		if len(args) == 1 {
			idxVal := evalExpr(args[0], env)
			if idxVal.Kind == KindError {
				return idxVal
			}
			idx := int(toNumVal(idxVal))

			if v.Kind == KindArr {
				if idx < 0 || idx >= len(v.Arr) {
					return ErrorVal("Index out of bounds")
				}
				return v.Arr[idx]
			}
			if v.Kind == KindArr2D {
				if idx < 0 || idx >= len(v.Arr2D) {
					return ErrorVal("Index out of bounds")
				}
				// Gibt die ganze Zeile als 1D-Array zurück
				return Value{Kind: KindArr, Arr: v.Arr2D[idx]}
			}
		}

		// --- FALL B: 2D-Zugriff (m(0,0)) ---
		if len(args) == 2 && v.Kind == KindArr2D {
			idx1Val := evalExpr(args[0], env)
			idx2Val := evalExpr(args[1], env)
			if idx1Val.Kind == KindError {
				return idx1Val
			}
			if idx2Val.Kind == KindError {
				return idx2Val
			}

			idx1 := int(toNumVal(idx1Val))
			idx2 := int(toNumVal(idx2Val))

			if idx1 < 0 || idx1 >= len(v.Arr2D) || idx2 < 0 || idx2 >= len(v.Arr2D[0]) {
				return ErrorVal("Matrix-Index out of bounds")
			}
			return v.Arr2D[idx1][idx2]
		}
	}

	// 2. Argumente auswerten
	evaluated := make([]Value, len(args))
	for i, arg := range args {
		val := evalExpr(arg, env)
		if val.Kind == KindError {
			return val
		}
		evaluated[i] = val
	}

	// 3. Builtins prüfen
	// HIER passiert die Magie: Wir fangen den Namen ab, BEVOR
	// wir in der builtins-Map nachsehen.
	if name == "Dump" {
		dumpEnvironment(env) // Hier ist env bekannt (aus dem Funktionskopf)
		return NullVal()
	}

	// 3. Builtins
	if info, ok := builtins[name]; ok {
		// Nicht f(evaluated), sondern info.Fn(evaluated)
		return info.Fn(evaluated)
	}

	// --- 4. User-defined FUNCTION ---
	// --- 4. User-defined FUNCTION ---
	if fn, ok := funcs[name]; ok {
		required := 0
		for _, p := range fn.Params {
			if !p.IsOptional {
				required++
			}
		}
		if len(evaluated) < required || len(evaluated) > len(fn.Params) {
			return ErrorVal(fmt.Sprintf("Funktion '%s' erwartet %d bis %d Argumente, erhalten: %d",
				name, required, len(fn.Params), len(evaluated)))
		}

		local := NewEnvironment(env)
		local.PutInternal("_fnReturn", NumVal(0))
		local.PutInternal("_currentFuncName", Value{Kind: KindStr, Str: name})

		for i, p := range fn.Params {
			if i < len(evaluated) {
				local.PutInternal(p.Name, evaluated[i])
			} else {
				// Fehlendes optionales Argument -> Default im Aufruf-Scope auswerten
				defVal := evalExpr(p.Default, local)
				if defVal.Kind == KindError {
					return defVal
				}
				local.PutInternal(p.Name, defVal)
			}
		}

		retVal, sig := evalStatements(fn.Body, local)
		if sig == SignalError {
			return ErrorVal(fmt.Sprintf("Fehler in Funktion '%s': %s", name, retVal.Str))
		}

		ret, _ := local.Get("_fnReturn")
		return ret
	}

	// --- 5. User-defined SUB ---
	// --- 5. User-defined SUB ---
	if s, ok := subs[name]; ok {
		required := 0
		for _, p := range s.Params {
			if !p.IsOptional {
				required++
			}
		}
		if len(evaluated) < required || len(evaluated) > len(s.Params) {
			return ErrorVal(fmt.Sprintf("Sub '%s' erwartet %d bis %d Argumente, erhalten: %d",
				name, required, len(s.Params), len(evaluated)))
		}

		local := NewEnvironment(env)
		for i, p := range s.Params {
			if i < len(evaluated) {
				local.PutInternal(p.Name, evaluated[i])
			} else {
				defVal := evalExpr(p.Default, local)
				if defVal.Kind == KindError {
					return defVal
				}
				local.PutInternal(p.Name, defVal)
			}
		}

		retVal, sig := evalStatements(s.Body, local)
		if sig == SignalError {
			return ErrorVal(fmt.Sprintf("Fehler in Sub '%s': %s", name, retVal.Str))
		}
		return NumVal(0)
	}

	return ErrorVal(fmt.Sprintf("Funktion oder Sub '%s' nicht gefunden", name))
}

func (n *ExitNode) stmtNode() {}

var subs = make(map[string]*SubNode)
var funcs = make(map[string]*FuncNode)

//var builtins = make(map[string]func([]Value) Value)

type BuiltinInfo struct {
	Fn           func([]Value) Value
	Beschreibung string
	Params       string
	Module       string
	ParamCount   int
}

var builtins = make(map[string]BuiltinInfo)

// 1. Die Listen-Funktion (Plural)
func evalStatements(stmts []Stmt, env *Environment) (Value, Signal) {
	var lastVal Value
	var sig Signal
	for _, s := range stmts {
		// Wir rufen für jedes Element die Single-Variante auf
		lastVal, sig = evalSingleStatement(s, env)
		if sig != SignalNone {
			return lastVal, sig
		}
	}
	return lastVal, SignalNone
}

// 2. Die Einzel-Funktion (Singular)
func evalSingleStatement(s Stmt, env *Environment) (Value, Signal) {
	// KEINE Schleife mehr hier!
	switch n := s.(type) {

	case *AssignNode:
		val := evalExpr(n.Value, env)
		if val.Kind == KindError {
			return val, SignalError
		}

		// 1. VB-Spezifisch: Rückgabewert setzen
		currFunc, _ := env.Get("_currentFuncName")
		if currFunc.Kind == KindStr && n.Name == currFunc.Str {
			env.Update("_fnReturn", val)
			return NullVal(), SignalNone
		}

		// 2. Deklaration vs. Zuweisung
		if n.IsDeclaration {
			// HIER: Das dritte Argument hinzufügen!
			// n.InLoop muss von deinem Parser/Walker gesetzt werden
			env.Define(n.Name, val, n.InLoop)
		} else {
			// Normale Zuweisung (x = 10): Suche erst lokal, dann global
			err := env.Update(n.Name, val)
			if err != nil {
				// Falls Update fehlschlägt, weil die Variable nirgends existiert:
				scopeInfo := "Public (global)"
				if env.parent != nil {
					scopeInfo = "lokal (Dim) oder Public (global)"
				}
				return ErrorVal(fmt.Sprintf("Variable '%s' muss erst mit Dim oder Public deklariert werden (nicht gefunden in %s)", n.Name, scopeInfo)), SignalError
			}
		}

		return NullVal(), SignalNone

	case *CompoundAssignNode:
		v, ok := n.Left.(*VarNode)
		if !ok {
			return ErrorVal("only variables supported"), SignalNone
		}

		current, ok := env.Get(v.Name)
		if !ok {
			return ErrorVal("undefined variable: " + v.Name), SignalNone
		}

		rhs := evalExpr(n.Right, env)
		if rhs.Kind == KindError {
			return rhs, SignalNone
		}

		result := calculateBinaryOp(mapAssignOp(n.Op), current, rhs)

		env.Set(v.Name, result)
		return result, SignalNone

	case *PrintNode:
		// 1. Gesamten Ausdruck auswerten (inkl. Verkettungen von Text & Farbe)
		val := evalExpr(n.Value, env)
		if val.Kind == KindError {
			return val, SignalError
		}

		outputStr := ToString(val)

		// --- LOGGING ---
		if logFile != nil {
			timestamp := time.Now().Format("15:04:05")
			fmt.Fprintf(logFile, "[%s] [PRINT] %s\n", timestamp, outputStr)
			return NullVal(), SignalNone
		}

		// --- AUSGABE ---
		// Wir hängen IMMER einen Reset an, um das Terminal sauber zu halten.
		// Das \033[0m wirkt sowohl nach n.Color als auch nach Inline-Farben.
		if n.Color != nil {
			colorVal := evalExpr(n.Color, env)
			if colorVal.Kind == KindStr {
				fmt.Printf("%s%s\033[0m\n", colorVal.Str, outputStr)
				return NullVal(), SignalNone
			}
			if colorVal.Kind == KindNum {
				c := int(colorVal.Num)
				fmt.Printf("\033[%dm%s\033[0m\n", 30+(c%10), outputStr)
				return NullVal(), SignalNone
			}
		}

		// Standardfall: Text (evtl. mit Inline-ANSI) + Reset
		fmt.Print(outputStr + "\033[0m\n")
		return NullVal(), SignalNone

	case *WhileNode:
		for {
			cond := evalExpr(n.Condition, env)
			if cond.Kind == KindError {
				return cond, SignalError
			}
			if !isTruthy(cond) {
				break
			}
			rv, sig := evalStatements(n.Body, env)
			if sig != SignalNone {
				if sig == SignalExitLoop {
					break
				}
				return rv, sig
			}
		}

	case *DoLoopNode:
		for {
			// Kopf-Bedingung
			if !n.CheckAtEnd && n.Condition != nil {
				cond := evalExpr(n.Condition, env)
				if cond.Kind == KindError {
					return cond, SignalError
				}
				shouldBreak := n.IsUntil == isTruthy(cond)
				if shouldBreak {
					break
				}
			}

			rv, sig := evalStatements(n.Body, env)
			if sig != SignalNone {
				if sig == SignalExitLoop {
					break
				}
				return rv, sig
			}

			// Fuß-Bedingung
			if n.CheckAtEnd && n.Condition != nil {
				cond := evalExpr(n.Condition, env)
				if cond.Kind == KindError {
					return cond, SignalError
				}
				shouldBreak := n.IsUntil == isTruthy(cond)
				if shouldBreak {
					break
				}
			}
		}

	case *ForEachNode:
		iterVal := evalExpr(n.Iterable, env)
		if iterVal.Kind == KindError {
			return iterVal, SignalError
		}
		keyPtr := env.GetRef(n.KeyVar)
		var valPtr *Value
		if n.ValVar != "" {
			valPtr = env.GetRef(n.ValVar)
		}

		if iterVal.Kind == KindArr {
			for i, val := range iterVal.Arr {
				if valPtr != nil {
					*keyPtr = NumVal(float64(i))
					*valPtr = val
				} else {
					*keyPtr = val
				}
				rv, sig := evalStatements(n.Body, env)
				if sig != SignalNone {
					if sig == SignalExitLoop {
						break
					}
					return rv, sig
				}
			}
		} else if iterVal.Kind == KindArr2D {
			for i, row := range iterVal.Arr2D {
				// Wir wandeln die [][]Value Zeile in ein normales KindArr um
				rowVal := Value{Kind: KindArr, Arr: row}

				if valPtr != nil {
					// Falls zwei Variablen genutzt werden: FOR EACH i, row IN csvData
					*keyPtr = NumVal(float64(i))
					*valPtr = rowVal
				} else {
					// Falls nur eine Variable genutzt wird: FOR EACH row IN csvData
					*keyPtr = rowVal
				}

				rv, sig := evalStatements(n.Body, env)
				if sig != SignalNone {
					if sig == SignalExitLoop {
						break
					}
					return rv, sig
				}
			}
		} else if iterVal.Kind == KindMap {
			for k, val := range iterVal.Map {
				if valPtr != nil {
					*keyPtr = StrVal(k)
					*valPtr = val
				} else {
					*keyPtr = val
				}
				rv, sig := evalStatements(n.Body, env)
				if sig != SignalNone {
					if sig == SignalExitLoop {
						break
					}
					return rv, sig
				}
			}
		} else {
			return ErrorVal("FOR EACH: Erwartet Array oder Map"), SignalError
		}

	case *ForNode:
		startVal := evalExpr(n.Start, env)
		if startVal.Kind == KindError {
			return startVal, SignalError
		}
		startNum := toNumVal(startVal)
		endVal := evalExpr(n.End, env)
		if endVal.Kind == KindError {
			return endVal, SignalError
		}
		endNum := toNumVal(endVal)
		step := n.Step
		if step == 0 {
			return ErrorVal("FOR mit STEP 0 ist nicht erlaubt"), SignalError
		}
		varPtr := env.GetRef(n.VarName)
		v := startNum
		for {
			if (step > 0 && v > endNum) || (step < 0 && v < endNum) {
				break
			}
			varPtr.Kind = KindNum
			varPtr.Num = v
			rv, sig := evalStatements(n.Body, env)
			if sig != SignalNone {
				if sig == SignalExitLoop {
					break
				}
				return rv, sig
			}
			v += step
		}

	case *SelectNode:
		valToTest := evalExpr(n.Expression, env)
		if valToTest.Kind == KindError {
			return valToTest, SignalError
		}

		matched := false
		for _, branch := range n.Cases {
			for _, condExpr := range branch.Conditions {
				isMatch := false

				// --- NEU: Hier unterscheiden wir zwischen Normal, Range und Is ---
				switch c := condExpr.(type) {
				case *RangeNode:
					// Nutze deine toFloat-Logik für " 25 "
					target := toFloatSafe([]Value{valToTest}, 0)

					lowVal := evalExpr(c.Low, env)
					low := toFloatSafe([]Value{lowVal}, 0)
					highVal := evalExpr(c.High, env)
					high := toFloatSafe([]Value{highVal}, 0)
					if target >= low && target <= high {
						isMatch = true
					}

				case *IsNode:
					// Nutze evalBinary für Vergleiche wie > 65
					res := evalBinary(valToTest, c.Operator, evalExpr(c.Value, env))
					if res.Kind == KindBool && res.Bool {
						isMatch = true
					}

				default:
					// Dein bisheriger Standard-Vergleich
					condVal := evalExpr(condExpr, env)
					if valuesAreEqual(valToTest, condVal) {
						isMatch = true
					}
				}
				// --- ENDE NEU ---

				if isMatch {
					matched = true
					rv, sig := evalStatements(branch.Body, env)
					if sig != SignalNone {
						return rv, sig
					}
					break
				}
			}
			if matched {
				break
			}
		}

		if !matched && len(n.Default) > 0 {
			rv, sig := evalStatements(n.Default, env)
			if sig != SignalNone {
				return rv, sig
			}
		}

	case *PublicNode:
		val := evalExpr(n.Value, env)
		if val.Kind == KindError {
			return val, SignalError
		}
		// SetGlobal wandert die Kette hoch bis zum Root-Environment
		env.SetGlobal(n.Name, val)
		return NullVal(), SignalNone

	case *PublicArrayNode:
		// 1. Dimension
		s1Val := evalExpr(n.Size1, env)
		if s1Val.Kind == KindError {
			return s1Val, SignalError
		}
		size1 := int(toNumVal(s1Val)) + 1

		var finalArray Value

		if n.Size2 != nil {
			// 2D Fall
			s2Val := evalExpr(n.Size2, env)
			if s2Val.Kind == KindError {
				return s2Val, SignalError
			}
			size2 := int(toNumVal(s2Val)) + 1

			matrix := make([][]Value, size1)
			for i := range matrix {
				row := make([]Value, size2)
				for j := range row {
					row[j] = NumVal(0)
				}
				matrix[i] = row
			}
			finalArray = Value{Kind: KindArr2D, Arr2D: matrix}
		} else {
			// 1D Fall
			arr := make([]Value, size1)
			for i := range arr {
				arr[i] = NumVal(0)
			}
			finalArray = Value{Kind: KindArr, Arr: arr}
		}

		// WICHTIG: SetGlobal statt Define/Set
		env.SetGlobal(n.Name, finalArray)
		return NullVal(), SignalNone

	case *DimArrayNode:
		// 1. Dimension auswerten
		size1 := 0
		if n.Size1 != nil {
			s1Val := evalExpr(n.Size1, env)
			if s1Val.Kind == KindError {
				return s1Val, SignalError
			}
			size1 = int(toNumVal(s1Val)) + 1 // +1 für BASIC-Konvention
		}

		// Prüfen: Ist es ein 2D-Array?
		if n.Size2 != nil {
			// 2. Dimension auswerten
			s2Val := evalExpr(n.Size2, env)
			if s2Val.Kind == KindError {
				return s2Val, SignalError
			}
			size2 := int(toNumVal(s2Val)) + 1

			// Matrix (KindArr2D) erstellen
			matrix := make([][]Value, size1)
			for i := range matrix {
				row := make([]Value, size2)
				for j := range row {
					row[j] = NumVal(0) // Mit 0 initialisieren
				}
				matrix[i] = row
			}

			// Lokal im Environment speichern
			env.Define(n.Name, Value{Kind: KindArr2D, Arr2D: matrix}, n.InLoop)

		} else {
			// Normales 1D-Array (KindArr) erstellen
			arr := make([]Value, size1)
			for i := range arr {
				arr[i] = NumVal(0)
			}

			// Lokal im Environment speichern
			env.Define(n.Name, Value{Kind: KindArr, Arr: arr}, n.InLoop)
		}

		return NullVal(), SignalNone

	case *ArrayAssignNode:
		v, found := env.Get(n.Name)
		if !found {
			return ErrorVal("Variable nicht deklariert: " + n.Name), SignalError
		}

		val := evalExpr(n.Value, env)
		if val.Kind == KindError {
			return val, SignalError
		}

		idx1Val := evalExpr(n.Index, env)
		idx1 := int(toNumVal(idx1Val))

		switch v.Kind {
		case KindArr2D:
			// --- 2D: STRENG (Kein Auto-Resize) ---
			if n.Index2 == nil {
				return ErrorVal("Zweiter Index fehlt"), SignalError
			}
			idx2Val := evalExpr(n.Index2, env)
			idx2 := int(toNumVal(idx2Val))

			// Harte Prüfung gegen die aktuellen Dimensionen
			if idx1 < 0 || idx1 >= len(v.Arr2D) || idx2 < 0 || (len(v.Arr2D) > 0 && idx2 >= len(v.Arr2D[0])) {
				return ErrorVal(fmt.Sprintf("Matrix-Index (%d,%d) außerhalb der Grenzen", idx1, idx2)), SignalError
			}
			v.Arr2D[idx1][idx2] = val

		case KindArr:
			// --- 1D: FLEXIBEL (Auto-Resize) ---
			if idx1 < 0 {
				return ErrorVal("Index negativ"), SignalError
			}

			if idx1 >= len(v.Arr) {
				newArr := make([]Value, idx1+1)
				copy(newArr, v.Arr)
				for i := len(v.Arr); i < len(newArr); i++ {
					newArr[i] = NumVal(0)
				}
				v.Arr = newArr
			}
			v.Arr[idx1] = val

		default:
			// Falls die Variable existiert, aber kein Array-Typ ist
			return ErrorVal(fmt.Sprintf("Variable '%s' ist kein Array", n.Name)), SignalError
		}

		env.Update(n.Name, v)
		return NullVal(), SignalNone

	case *ReturnNode:
		val := evalExpr(n.Value, env)
		if val.Kind == KindError {
			return val, SignalError
		}
		env.Set("_fnReturn", val)
		return val, SignalReturn

	case *IfNode:
		executed := false
		for _, br := range n.Branches {
			cond := evalExpr(br.Cond, env)
			if cond.Kind == KindError {
				return cond, SignalError
			}
			if isTruthy(cond) {
				rv, sig := evalStatements(br.Body, env)
				if sig != SignalNone {
					return rv, sig
				}
				executed = true
				break
			}
		}
		if !executed && len(n.Else) > 0 {
			rv, sig := evalStatements(n.Else, env)
			if sig != SignalNone {
				return rv, sig
			}
		}

	case *ExitNode:
		switch n.ExitType {
		case "Function":
			return Value{}, SignalExitFunction
		case "Sub":
			return Value{}, SignalExitSub
		default:
			return Value{}, SignalExitLoop
		}

	case *MultiStmtNode:
		// HIER SPAREN WIR DEN SPEICHER:
		// n.Stmts ist schon ein Slice, wir geben es direkt weiter.
		return evalStatements(n.Stmts, env)

	case *SubNode:
		subs[n.Name] = n

	case *FuncNode:
		funcs[n.Name] = n

	case *CallNode:
		// 1. Sonderfall: Cls (Bildschirm löschen)
		if strings.EqualFold(n.Name, "Cls") {
			var cmd *exec.Cmd
			if runtime.GOOS == "windows" {
				// Windows nutzt den internen CMD-Befehl 'cls'
				cmd = exec.Command("cmd", "/c", "cls")
			} else {
				// Ubuntu/Linux und macOS nutzen den 'clear' Befehl
				cmd = exec.Command("clear")
			}

			// Den Output des System-Befehls direkt an unser Terminal leiten
			cmd.Stdout = os.Stdout
			err := cmd.Run()

			if err != nil {
				// Falls 'clear' oder 'cls' fehlschlagen, nutzen wir die ANSI-Sequenz als Backup
				fmt.Print("\033[H\033[2J\033[3J")
			}
			return NullVal(), SignalNone
		}

		// 2. Reguläre Funktionsaufrufe (wie FromCSV, KindOf, etc.)
		val := evalFunctionCall(n.Name, n.Args, env)

		// Fehlerprüfung: Wenn die Funktion einen Error zurückgibt
		if val.Kind == KindError {
			return val, SignalError
		}

		// Rückgabe des Funktionswerts (oder NullVal, falls nichts zurückkommt)
		return val, SignalNone
	}
	return Value{}, SignalNone
}

func evalBinary(left Value, op TokenType, right Value) Value {
	// Numerische Vergleiche: Wandelt beide Seiten in Floats um
	lNum := toFloatSafe([]Value{left}, 0)
	rNum := toFloatSafe([]Value{right}, 0)

	switch op {
	case GT: // OK
		return Value{Kind: KindBool, Bool: lNum > rNum}
	case LT: // Geändert von LESS zu LT
		return Value{Kind: KindBool, Bool: lNum < rNum}
	case GE: // OK
		return Value{Kind: KindBool, Bool: lNum >= rNum}
	case LE: // OK
		return Value{Kind: KindBool, Bool: lNum <= rNum}
	case EQ: // Geändert von EQUAL zu EQ
		return Value{Kind: KindBool, Bool: valuesAreEqual(left, right)}
	case NEQ: // Geändert von NOTEQUAL zu NEQ
		return Value{Kind: KindBool, Bool: !valuesAreEqual(left, right)}
	case AND: // OK
		return Value{Kind: KindBool, Bool: isTruthy(left) && isTruthy(right)}
	case OR: // OK
		return Value{Kind: KindBool, Bool: isTruthy(left) || isTruthy(right)}
	}

	return ErrorVal(fmt.Sprintf("Operator %v wird im Select Case noch nicht unterstützt", op))
}

// func evalExpr(e Expr, env map[string]Value) Value {
func evalExpr(e Expr, env *Environment) Value {
	switch n := e.(type) {
	case Value:
		return n

	case *UnaryOpNode:
		rightVal := evalExpr(n.Right, env)
		switch n.Op {
		case NOT:
			return Value{Kind: KindBool, Bool: !isTruthy(rightVal)}
		case MINUS:
			if rightVal.Kind == KindNum {
				return NumVal(-rightVal.Num)
			}
			return ErrorVal("Minus erfordert eine Zahl")
		}

	case *ArrayLiteralNode:
		var elements []Value
		for _, expr := range n.Elements {
			// Rekursiver Aufruf: Wertet jedes Element aus
			val := evalExpr(expr, env)

			// Das "Sicherheitsende": Fehler werden sofort gemeldet
			if val.Kind == KindError {
				return val // Reicht das ErrorVal direkt an den Parser/Runner weiter
			}
			elements = append(elements, val)
		}
		// Finales Paket: Ein fertiges VB-Array
		return Value{Kind: KindArr, Arr: elements}

	case *NumberNode:
		return NumVal(n.Value)
	case *StringNode:
		return Value{Kind: KindStr, Str: n.Value}

	// Beide Knotentypen nutzen jetzt die identische Logik
	case *CallNode:
		return evalFunctionCall(n.Name, n.Args, env)
	case *CallExprNode:
		return evalFunctionCall(n.Name, n.Args, env)

	case *MultiStmtNode:
		var lastVal Value
		// Wir brauchen eine Variable für das Signal, um den Rückgabewert
		// von evalStatements "aufzufangen", aber wir geben es nicht weiter.
		var sig Signal

		for _, stmt := range n.Stmts {
			// Hier empfangen wir beide Werte
			lastVal, sig = evalStatements([]Stmt{stmt}, env)

			// Wenn innerhalb eines Ausdrucks ein Fehler auftritt,
			// geben wir den Fehler-Wert zurück.
			if sig == SignalError {
				return lastVal
			}
			// Andere Signale wie SignalReturn ignorieren wir hier (oder behandeln sie als Ende)
			if sig != SignalNone {
				break
			}
		}
		// WICHTIG: Hier nur EINEN Wert zurückgeben!
		return lastVal

	case *MapIndexNode:
		baseVal := evalExpr(n.Base, env)
		if baseVal.Kind == KindError {
			return baseVal
		}

		if baseVal.Kind != KindMap {
			return ErrorVal("Map-Zugriff mit '[...]' ist nur auf Maps möglich")
		}

		keyVal := evalExpr(n.Key, env)
		if keyVal.Kind == KindError {
			return keyVal
		}
		key := ToString(keyVal)

		if val, ok := baseVal.Map[key]; ok {
			return val
		}
		return NilValue

	case *VarNode:
		// 1. Punkt-Notation (Objekte)
		if strings.Contains(n.Name, ".") {
			parts := strings.SplitN(n.Name, ".", 2)
			objVal, found := env.Get(parts[0])

			if !found {
				return ErrorVal(fmt.Sprintf("Objekt '%s' ist nicht definiert", parts[0]))
			}

			if objVal.Kind != KindObj {
				return ErrorVal(parts[0] + " ist kein Object")
			}

			if objVal.Obj == nil {
				return NilValue
			}

			m := objVal.Obj.Fields
			if val, ok := m[parts[1]]; ok {
				return val
			}
			return NilValue
		}

		// 2. Basis-Variable holen (Scope-Chain!)
		v, found := env.Get(n.Name)
		if !found {
			scopeInfo := "Public (global)"
			if env.parent != nil {
				scopeInfo = "lokal (Dim) oder Public (global)"
			}
			return ErrorVal(fmt.Sprintf("Variable '%s' ist weder %s definiert", n.Name, scopeInfo))
		}
		if v.Kind == KindNil {
			return ErrorVal("undefined variable: " + n.Name)
		}

		// 3. Index-Zugriff
		if n.Index1 != nil {

			// FALL A: 2D-Array
			if n.Index2 != nil {
				if v.Kind != KindArr2D {
					return ErrorVal(n.Name + " ist kein 2D-Array")
				}

				idx1, err1 := checkArrayIndex(n.Index1, env, len(v.Arr2D), n.Name+" (Zeile)")
				if err1.Kind == KindError {
					return err1
				}

				if len(v.Arr2D[idx1]) == 0 {
					return ErrorVal("2D-Array Zeile ist leer")
				}

				idx2, err2 := checkArrayIndex(n.Index2, env, len(v.Arr2D[idx1]), n.Name+" (Spalte)")
				if err2.Kind == KindError {
					return err2
				}

				return v.Arr2D[idx1][idx2]
			}

			// FALL B: 1D-Array
			if v.Kind == KindArr {
				idx, err := checkArrayIndex(n.Index1, env, len(v.Arr), n.Name)
				if err.Kind == KindError {
					return err
				}
				return v.Arr[idx]
			}

			// FALL C: Map-Zugriff via String-Key (z.B. m["mail"])
			if v.Kind == KindMap {
				keyVal := evalExpr(n.Index1, env)
				if keyVal.Kind == KindError {
					return keyVal
				}
				key := ToString(keyVal)
				if val, ok := v.Map[key]; ok {
					return val
				}
				return NilValue
			}

			return ErrorVal(n.Name + " ist kein Array oder Map")
		}

		// 4. Einfache Variable (kein Index)
		return v

	//case *UseNode:
	//	LoadModules(env, n.Modules)
	//	return Value{}

	case *BinOpNode:
		l := evalExpr(n.Left, env)
		if l.Kind == KindError {
			return l
		}
		r := evalExpr(n.Right, env)
		if r.Kind == KindError {
			return r
		}

		switch n.Op {
		case PLUS:
			// 1. Versuch: Beides als Zahlen behandeln (inkl. Auto-Konvertierung von Strings)
			ln, errL := requireNumber(l, "+")
			rn, errR := requireNumber(r, "+")

			if errL.Kind != KindError && errR.Kind != KindError {
				// Erfolg! Beides sind Zahlen (oder Zahlen-Strings wie "10")
				return NumVal(ln + rn)
			}

			// 2. Fallback: Wenn es keine reinen Zahlen sind, behandeln wir es als Text
			// Das ist das "Haus" + 10 -> "Haus10" Szenario
			return StrVal(ToString(l) + ToString(r))

		case MINUS:
			ln, err := requireNumber(l, "-")
			if err.Kind == KindError {
				return err
			}
			rn, err := requireNumber(r, "-")
			if err.Kind == KindError {
				return err
			}
			return NumVal(ln - rn)

		case MUL:
			ln, err := requireNumber(l, "*")
			if err.Kind == KindError {
				return err
			}
			rn, err := requireNumber(r, "*")
			if err.Kind == KindError {
				return err
			}
			return NumVal(ln * rn)

		case DIV:
			ln, err := requireNumber(l, "/")
			if err.Kind == KindError {
				return err
			}
			rn, err := requireNumber(r, "/")
			if err.Kind == KindError {
				return err
			}
			if rn == 0 {
				return ErrorVal("Division durch Null")
			}
			return NumVal(ln / rn)

		case AMP:
			return StrVal(ToString(l) + ToString(r))

		case AND:
			if !isTruthy(l) {
				return BoolVal(false)
			}
			return BoolVal(isTruthy(r))

		case OR:
			if isTruthy(l) {
				return BoolVal(true)
			}
			return BoolVal(isTruthy(r))

		case EQ:
			// 1. Booleans
			if l.Kind == KindBool && r.Kind == KindBool {
				return BoolVal(l.Bool == r.Bool)
			}
			// 2. Strings (einer von beiden ist String)
			if l.Kind == KindStr || r.Kind == KindStr {
				return BoolVal(ToString(l) == ToString(r))
			}
			// 3. Zahlen
			if l.Kind == KindNum && r.Kind == KindNum {
				return BoolVal(l.Num == r.Num)
			}
			// 4. Fallback für gemischte Typen (z.B. Bool vs Num)
			return BoolVal(valuesAreEqual(l, r))

		case NEQ:
			if l.Kind == KindBool && r.Kind == KindBool {
				return BoolVal(l.Bool != r.Bool)
			}
			if l.Kind == KindStr || r.Kind == KindStr {
				return BoolVal(ToString(l) != ToString(r))
			}
			if l.Kind == KindNum && r.Kind == KindNum {
				return BoolVal(l.Num != r.Num)
			}
			return BoolVal(!valuesAreEqual(l, r))

		case LT, GT, LE, GE:
			// 1. Den Namen des Operators holen
			opStr := n.Op.String()

			// 2. Den Namen beim Aufruf von requireNumber BENUTZEN
			ln, err := requireNumber(l, opStr) // Hier wird opStr jetzt benutzt!
			if err.Kind == KindError {
				return err
			}

			rn, err := requireNumber(r, opStr) // Und hier auch!
			if err.Kind == KindError {
				return err
			}
			switch n.Op {
			case LT:
				return BoolVal(ln < rn)
			case GT:
				return BoolVal(ln > rn)
			case LE:
				return BoolVal(ln <= rn)
			case GE:
				return BoolVal(ln >= rn)
			}
		}

	default:
		// Falls ein Operator durch den Parser gerutscht ist,
		// den der Evaluator noch nicht kennt.
		return ErrorVal(fmt.Sprintf("Unbekannter Ausdruckstyp: %T", e))

	} // Ende switch n.Op

	// Falls wir hier landen, ist etwas im Programmfluss
	// fundamental schiefgelaufen (sollte theoretisch nie passieren)
	return ErrorVal("Kritischer Fehler in der Ausdrucks-Auswertung")
}

// Hilfsfunktion: Alle FuncNode/SubNode rekursiv registrieren
func registerFuncsAndSubs(stmts []Stmt) {
	for _, s := range stmts {
		switch n := s.(type) {
		case *FuncNode:
			funcs[n.Name] = n
		case *SubNode:
			subs[n.Name] = n
		case *MultiStmtNode:
			registerFuncsAndSubs(n.Stmts)
		}
	}
}

func valuesAreEqual(l, r Value) bool {
	// 1. Wenn Typen gleich sind: Direkter Vergleich
	if l.Kind == r.Kind {
		switch l.Kind {
		case KindNum:
			return l.Num == r.Num
		case KindStr:
			return strings.EqualFold(l.Str, r.Str) // Case-Insensitive wie in VB!
		case KindBool:
			return l.Bool == r.Bool
		case KindNull, KindNil:
			return true
		}
	}

	// 2. Cross-Type Vergleich: Zahl vs String
	// Erlaubt: Case 10 ... wenn x = "10"
	if (l.Kind == KindNum && r.Kind == KindStr) || (l.Kind == KindStr && r.Kind == KindNum) {
		return toNumVal(l) == toNumVal(r)
	}

	// 3. Boolean Checks (VB: -1 ist True, 0 ist False)
	if l.Kind == KindBool || r.Kind == KindBool {
		return isTruthy(l) == isTruthy(r)
	}

	return false
}

// Hilfsfunktion
func isPrimitive(v Value) bool {
	return v.Kind == KindNum || v.Kind == KindStr || v.Kind == KindBool
}
