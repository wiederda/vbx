package main

import (
	"fmt"
	"strings"
)

// ------------------------------------------------------------
// Bekannte Namen sammeln (live aus dem geladenen Zustand,
// keine externe Datei nötig)
// ------------------------------------------------------------

var coreKeywords = map[string]bool{
	"iserror":   true,
	"errortext": true,
	"dump":      true,
	"cls":       true,
}

func knownNameSet() map[string]bool {
	known := make(map[string]bool, len(builtins)+len(funcs)+len(subs)+len(coreKeywords))

	for name := range builtins {
		known[strings.ToLower(name)] = true
	}
	for name := range funcs {
		known[strings.ToLower(name)] = true
	}
	for name := range subs {
		known[strings.ToLower(name)] = true
	}
	for name := range coreKeywords {
		known[name] = true
	}

	return known
}

// ------------------------------------------------------------
// Levenshtein für "Meintest du...?"
// ------------------------------------------------------------

func levenshtein(a, b string) int {
	la, lb := len(a), len(b)

	d := make([][]int, la+1)
	for i := range d {
		d[i] = make([]int, lb+1)
		d[i][0] = i
	}

	for j := 0; j <= lb; j++ {
		d[0][j] = j
	}

	for i := 1; i <= la; i++ {
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}

			d[i][j] = min3(
				d[i-1][j]+1,
				d[i][j-1]+1,
				d[i-1][j-1]+cost,
			)
		}
	}

	return d[la][lb]
}

func min3(a, b, c int) int {
	m := a

	if b < m {
		m = b
	}

	if c < m {
		m = c
	}

	return m
}

func closestSuggestion(name string, known map[string]bool) string {
	best := ""
	bestDist := 4 // ab Distanz 4 kein sinnvoller Vorschlag mehr

	for k := range known {
		d := levenshtein(name, k)

		if d < bestDist {
			bestDist = d
			best = k
		}
	}

	return best
}

// ------------------------------------------------------------
// Warnung
// ------------------------------------------------------------

type CallWarning struct {
	Name          string // Original-Schreibweise aus dem Skript
	Context       string // "Top-Level", "Sub X" oder "Function Y"
	Suggestion    string
	MissingParens bool
}

// ------------------------------------------------------------
// Deklarierte Variablennamen einsammeln
// ------------------------------------------------------------
//
// Wichtig: myArr(i) wird vom Parser IMMER als CallExprNode/CallNode
// geparst (siehe parseFactor und parseStmt), nicht als indizierter
// VarNode - ob es sich zur Laufzeit um Array-/Map-/Objekt-Zugriff
// oder einen echten Funktionsaufruf handelt, entscheidet sich erst
// in evalFunctionCall per env.Get(name). Dasselbe gilt für
// punktbehaftete Namen wie meinObjekt.Feld(i) - der Parser baut den
// vollen "fullName"-String identisch zu einem Modul-Aufruf wie
// ini.Sections(), bevor er auf LPAREN prüft.
//
// Damit der Precheck hier keine falschen Warnungen wirft (z.B.
// "Dim sections = ini.Sections()" gefolgt von "sections(i)"), müssen
// alle deklarierten Variablennamen bekannt sein und von der Prüfung
// ausgenommen werden.
//
// Scope-Modell (vereinfacht, aber passend zum Interpreter):
// Top-Level-Deklarationen sind überall sichtbar (auch in Sub/Function,
// da Environment.Get die Parent-Kette hochläuft). Innerhalb einer
// Sub/Function deklarierte Namen (inkl. Parameter) sind nur dort
// sichtbar. If/For/While/Do/Select teilen sich das Environment ihres
// umgebenden Blocks (kein eigener Scope) - Deklarationen darin zählen
// also zum jeweils umgebenden Top-Level- bzw. Sub/Function-Scope.

func collectDeclaredNames(stmts []Stmt, target map[string]bool) {
	for _, s := range stmts {
		collectDeclaredNamesStmt(s, target)
	}
}

func collectDeclaredNamesStmt(s Stmt, target map[string]bool) {
	switch n := s.(type) {

	case *AssignNode:
		if n.IsDeclaration {
			target[strings.ToLower(n.Name)] = true
		}

	case *PublicNode:
		target[strings.ToLower(n.Name)] = true
	}

	switch n := s.(type) {

	case *DimArrayNode:
		target[strings.ToLower(n.Name)] = true

	case *PublicArrayNode:
		target[strings.ToLower(n.Name)] = true

	case *ForEachNode:
		target[strings.ToLower(n.KeyVar)] = true

		if n.ValVar != "" {
			target[strings.ToLower(n.ValVar)] = true
		}

		collectDeclaredNames(n.Body, target)

	case *ForNode:
		target[strings.ToLower(n.VarName)] = true
		collectDeclaredNames(n.Body, target)

	case *WhileNode:
		collectDeclaredNames(n.Body, target)

	case *DoLoopNode:
		collectDeclaredNames(n.Body, target)

	case *IfNode:
		for _, br := range n.Branches {
			collectDeclaredNames(br.Body, target)
		}

		collectDeclaredNames(n.Else, target)

	case *SelectNode:
		for _, branch := range n.Cases {
			collectDeclaredNames(branch.Body, target)
		}

		collectDeclaredNames(n.Default, target)

	case *MultiStmtNode:
		collectDeclaredNames(n.Stmts, target)

		// *SubNode, *FuncNode bewusst NICHT rekursiv erfasst - deren
		// eigene Namen gehören zu ihrem eigenen Scope, nicht zum
		// umgebenden (siehe collectProcNames/findProcs).
	}
}

// collectProcNames sammelt für eine einzelne Sub/Function ihre
// Parameter- und lokal deklarierten Namen (ihr eigener Scope).
func collectProcNames(params []ParamDef, body []Stmt) map[string]bool {
	names := make(map[string]bool)

	for _, p := range params {
		names[strings.ToLower(p.Name)] = true
	}

	collectDeclaredNames(body, names)

	return names
}

// findProcs sammelt alle Sub/Function-Deklarationen aus dem
// Top-Level-AST (verschachtelte Deklarationen sind laut Parser
// ohnehin nicht erlaubt, siehe p.procStack-Prüfung).
func findProcs(stmts []Stmt) (subNodes []*SubNode, funcNodes []*FuncNode) {
	for _, s := range stmts {
		switch n := s.(type) {

		case *SubNode:
			subNodes = append(subNodes, n)

		case *FuncNode:
			funcNodes = append(funcNodes, n)

		case *MultiStmtNode:
			sn, fn := findProcs(n.Stmts)

			subNodes = append(subNodes, sn...)
			funcNodes = append(funcNodes, fn...)
		}
	}

	return
}

func mergeVarSets(a, b map[string]bool) map[string]bool {
	merged := make(map[string]bool, len(a)+len(b))

	for k := range a {
		merged[k] = true
	}

	for k := range b {
		merged[k] = true
	}

	return merged
}

// ------------------------------------------------------------
// AST-Walker
// ------------------------------------------------------------

type callChecker struct {
	known    map[string]bool
	warnings []CallWarning
}

type scope struct {
	context string
	vars    map[string]bool // topLevelVars ∪ (aktuelle Proc-Namen, falls in einer Sub/Function)
}

// CheckUnknownCalls läuft rekursiv über den kompletten AST und meldet
// jeden CallNode/CallExprNode, dessen Name weder in builtins, funcs,
// subs, den fest verdrahteten Kernfunktionen NOCH in einer im
// aktuellen Scope sichtbaren deklarierten Variable auftaucht.
//
// Zusätzlich wird bei VarNode geprüft, ob ein punktbehafteter Name
// wie "app.ExecutablePath" tatsächlich eine registrierte Modul-Funktion
// ist. In diesem Fall fehlt der Aufruf "()".
//
// Vergleich ist case-insensitive (bewusst konservativ, um keine
// False-Positives zu erzeugen, falls Aufrufe irgendwo case-insensitiv
// behandelt werden) - fängt also NICHT den Fall ab, dass eine
// tatsächlich case-sensitive Groß-/Kleinschreibung im Skript nicht
// zur Registrierung passt, nur echte Tippfehler in Namen.

func CheckUnknownCalls(stmts []Stmt) []CallWarning {
	c := &callChecker{
		known: knownNameSet(),
	}

	topLevelVars := make(map[string]bool)

	collectDeclaredNames(stmts, topLevelVars)

	c.walkStmts(
		stmts,
		scope{
			context: "Top-Level",
			vars:    topLevelVars,
		},
	)

	subNodes, funcNodes := findProcs(stmts)

	for _, sn := range subNodes {
		procVars := collectProcNames(sn.Params, sn.Body)
		merged := mergeVarSets(topLevelVars, procVars)

		c.walkStmts(
			sn.Body,
			scope{
				context: "Sub " + sn.Name,
				vars:    merged,
			},
		)
	}

	for _, fn := range funcNodes {
		procVars := collectProcNames(fn.Params, fn.Body)
		merged := mergeVarSets(topLevelVars, procVars)

		c.walkStmts(
			fn.Body,
			scope{
				context: "Function " + fn.Name,
				vars:    merged,
			},
		)
	}

	return c.warnings
}

// ------------------------------------------------------------
// Unbekannte Funktion / Aufruf prüfen
// ------------------------------------------------------------

func (c *callChecker) reportIfUnknown(name string, sc scope) {
	low := strings.ToLower(name)

	if c.known[low] {
		return
	}

	if sc.vars[low] {
		// z.B. "sections" - deklarierte Variable,
		// Array-/Map-Zugriff
		return
	}

	// Punktbehafteter Name (z.B. "meinObjekt.Feld"):
	// Falls der Teil vor dem ersten Punkt eine deklarierte Variable ist,
	// kann es sich um Objekt-Feld-Zugriff auf ein Array/eine Map handeln.
	//
	// Das Kind ist wie bei "sections(i)" erst zur Laufzeit bekannt,
	// daher ebenfalls keine Warnung.
	//
	// Trifft NICHT auf echte Modul-Aufrufe wie "ini.Sections" zu,
	// weil "ini" selbst nie eine deklarierte Variable ist
	// (Module werden nicht per Dim/Public angelegt).
	if idx := strings.IndexByte(low, '.'); idx > 0 {
		prefix := low[:idx]

		if sc.vars[prefix] {
			return
		}
	}

	c.warnings = append(c.warnings, CallWarning{
		Name:       name,
		Context:    sc.context,
		Suggestion: closestSuggestion(low, c.known),
	})
}

// ------------------------------------------------------------
// Fehlende Klammern bei Modul-Funktion prüfen
// ------------------------------------------------------------
//
// Beispiel:
//
//     app.ExecutablePath
//
// wird vom Parser als VarNode erkannt, weil () fehlen.
//
// Wenn "app.ExecutablePath" aber einer registrierten Funktion
// entspricht, muss der Aufruf:
//
//     app.ExecutablePath()
//
// lauten.
//
// Dabei werden beide möglichen Registrierungsformen unterstützt:
//
//     builtins["ExecutablePath"]       mit Module == "app"
//     builtins["app.ExecutablePath"]   mit Module == "app"
//
// ------------------------------------------------------------

func (c *callChecker) reportIfMissingParentheses(name string, sc scope) {
	parts := strings.SplitN(name, ".", 2)

	if len(parts) != 2 {
		return
	}

	module := parts[0]
	function := parts[1]

	if _, ok := findBuiltinByModuleAndName(module, function); !ok {
		return
	}

	c.warnings = append(c.warnings, CallWarning{
		Name:          name,
		Context:       sc.context,
		Suggestion:    name + "()",
		MissingParens: true,
	})
}

// ------------------------------------------------------------
// Statements
// ------------------------------------------------------------

func (c *callChecker) walkStmts(stmts []Stmt, sc scope) {
	for _, s := range stmts {
		c.walkStmt(s, sc)
	}
}

func (c *callChecker) walkStmt(s Stmt, sc scope) {
	if s == nil {
		return
	}

	switch n := s.(type) {

	case *AssignNode:
		c.walkExpr(n.Value, sc)

	case *CompoundAssignNode:
		c.walkExpr(n.Left, sc)
		c.walkExpr(n.Right, sc)

	case *PrintNode:
		c.walkExpr(n.Value, sc)
		c.walkExpr(n.Color, sc)

	case *WhileNode:
		c.walkExpr(n.Condition, sc)
		c.walkStmts(n.Body, sc)

	case *DoLoopNode:
		c.walkExpr(n.Condition, sc)
		c.walkStmts(n.Body, sc)

	case *ForEachNode:
		c.walkExpr(n.Iterable, sc)
		c.walkStmts(n.Body, sc)

	case *ForNode:
		c.walkExpr(n.Start, sc)
		c.walkExpr(n.End, sc)
		c.walkStmts(n.Body, sc)

	case *SelectNode:
		c.walkExpr(n.Expression, sc)

		for _, branch := range n.Cases {
			for _, cond := range branch.Conditions {
				c.walkExpr(cond, sc)
			}

			c.walkStmts(branch.Body, sc)
		}

		c.walkStmts(n.Default, sc)

	case *PublicNode:
		c.walkExpr(n.Value, sc)

	case *PublicArrayNode:
		c.walkExpr(n.Size1, sc)
		c.walkExpr(n.Size2, sc)

	case *DimArrayNode:
		c.walkExpr(n.Size1, sc)
		c.walkExpr(n.Size2, sc)

	case *ArrayAssignNode:
		c.walkExpr(n.Index, sc)
		c.walkExpr(n.Index2, sc)
		c.walkExpr(n.Value, sc)

	case *ReturnNode:
		c.walkExpr(n.Value, sc)

	case *IfNode:
		for _, br := range n.Branches {
			c.walkExpr(br.Cond, sc)
			c.walkStmts(br.Body, sc)
		}

		c.walkStmts(n.Else, sc)

	case *ExitNode:
		// keine Kinder

	case *ContinueNode:
		// keine Kinder

	case *MultiStmtNode:
		c.walkStmts(n.Stmts, sc)

	case *SubNode:
		// wird separat mit eigenem Scope behandelt
		// (CheckUnknownCalls) - hier nicht nochmal absteigen,
		// sonst falscher Kontext/Scope.

	case *FuncNode:
		// wie SubNode - separat behandelt.

	case *CallNode:
		c.reportIfUnknown(n.Name, sc)

		for _, a := range n.Args {
			c.walkExpr(a, sc)
		}

	default:
		// Bewusst NICHT stillschweigend ignorieren: falls künftig ein
		// neuer Stmt-Typ dazukommt und hier vergessen wird, soll das
		// auffallen statt lautlos aus der Prüfung rauszufallen.
		fmt.Printf(
			"[PRECHECK] Unbekannter Stmt-Typ im Walker: %T (bitte in walkStmt ergänzen)\n",
			s,
		)
	}
}

// ------------------------------------------------------------
// Expressions
// ------------------------------------------------------------

func (c *callChecker) walkExpr(e Expr, sc scope) {
	if e == nil {
		return
	}

	switch n := e.(type) {

	case Value:
		// Literal (z.B. BOOL true/false aus parseFactor)
		// - keine Kinder

	case *NumberNode:
		// Literal

	case *StringNode:
		// Literal

	case *UnaryOpNode:
		c.walkExpr(n.Right, sc)

	case *ArrayLiteralNode:
		for _, el := range n.Elements {
			c.walkExpr(el, sc)
		}

	case *CallExprNode:
		c.reportIfUnknown(n.Name, sc)

		for _, a := range n.Args {
			c.walkExpr(a, sc)
		}

	case *CallNode:
		// evalExpr behandelt CallNode ebenfalls als gültigen Expr-Fall,
		// daher hier gespiegelt.
		c.reportIfUnknown(n.Name, sc)

		for _, a := range n.Args {
			c.walkExpr(a, sc)
		}

	case *MapIndexNode:
		c.walkExpr(n.Base, sc)
		c.walkExpr(n.Key, sc)

	case *VarNode:
		// Ein VarNode kann z.B. sein:
		//
		//     app.ExecutablePath
		//
		// Wenn dies tatsächlich eine registrierte Modul-Funktion ist,
		// fehlen die ().
		c.reportIfMissingParentheses(n.Name, sc)

		// Eventuelle Index-Ausdrücke weiterhin prüfen.
		c.walkExpr(n.Index1, sc)
		c.walkExpr(n.Index2, sc)

	case *BinOpNode:
		c.walkExpr(n.Left, sc)
		c.walkExpr(n.Right, sc)

	case *RangeNode:
		c.walkExpr(n.Low, sc)
		c.walkExpr(n.High, sc)

	case *IsNode:
		c.walkExpr(n.Value, sc)

	case *MultiStmtNode:
		// laut evalExpr auch als Expr genutzt
		// (verkettete Statements innerhalb eines Ausdrucks)
		c.walkStmts(n.Stmts, sc)

	default:
		fmt.Printf(
			"[PRECHECK] Unbekannter Expr-Typ im Walker: %T (bitte in walkExpr ergänzen)\n",
			e,
		)
	}
}
