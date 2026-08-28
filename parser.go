package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ---------------- Parser ----------------
type Parser struct {
	tokens    []Token
	pos       int
	Errors    []string
	loopDepth int
	loopStack []TokenType
	env       *Environment
	procStack []TokenType // <-- neu hinzufügen
}

type MapIndexNode struct {
	Base Expr
	Key  Expr
}

// Konsumiert eine Kette von [Key][Key]... rein zur Fehlerprüfung.
// Rückgabe: true, wenn mindestens eine Klammer gefunden wurde.
func (p *Parser) rejectBracketAssignTarget() bool {
	found := false
	for p.peek().Type == LBRACKET {
		found = true
		p.next() // '[' konsumieren
		p.parseExpr()
		if p.next().Type != RBRACKET {
			p.error("Erwartet ']' nach Map-Zugriff")
		}
	}
	if found && p.peek().Type == EQ {
		p.error("Map-Zugriff mit '[...]' ist nur lesend erlaubt. Zuweisungen bitte über map.Set(...) vornehmen.")
	}
	return found
}

func (m *MapIndexNode) expressionNode() {}

type ForEachNode struct {
	KeyVar   string // erste Variable (Key)
	ValVar   string // zweite Variable (Value), optional bei Arrays
	Iterable Expr   // Ausdruck, der Array/Map liefert
	Body     []Stmt // Anweisungen in der Schleife
}

type ParamDef struct {
	Name       string
	IsOptional bool
	Default    Expr // nil, wenn kein Default angegeben
}

type PublicNode struct {
	Name  string
	Value Expr
}

// Für: Public a(10, 10)
type PublicArrayNode struct {
	Name  string
	Size1 Expr
	Size2 Expr // nil für 1D
}

type RangeNode struct {
	Low  Expr
	High Expr
}

type ArrayLiteralNode struct {
	Elements []Expr
}

func (n *ArrayLiteralNode) Pos() { /* Falls du Positionstracking hast */ }

func (r *RangeNode) expressionNode() {} // Interface-Erfüllung

type IsNode struct {
	Operator TokenType // z.B. GREATER, LESS
	Value    Expr
}

type CompoundAssignNode struct {
	Left  Expr
	Op    TokenType
	Right Expr
}

func (i *IsNode) expressionNode() {}

func (p *Parser) peek() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: EOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) next() Token {
	tok := p.peek()
	p.pos++
	return tok
}

func (p *Parser) skipStuff() {
	for p.peek().Type == NEWLINE || p.peek().Type == COMMENT {
		p.next()
	}
}

func (p *Parser) peekNext() Token {
	if p.pos+1 >= len(p.tokens) {
		return Token{Type: EOF}
	}
	return p.tokens[p.pos+1]
}

func (p *Parser) error(format string, args ...interface{}) {
	// 1. Die Nachricht formatieren
	msg := fmt.Sprintf(format, args...)

	// 2. Optional: In die Liste eintragen (falls du sie später gesammelt brauchst)
	p.Errors = append(p.Errors, msg)

	// 3. DAS IST DER ENTSCHEIDENDE PUNKT:
	// Wir werfen die echte Nachricht in den Panic-Topf!
	panic(msg)
}

func (p *Parser) parseArrayLiteral() Expr {
	p.next() // { konsumieren (LBRACE)
	var elements []Expr

	p.skipStuff() // Newlines nach der Klammer erlauben

	for p.peek().Type != RBRACE && p.peek().Type != EOF {
		// Rekursiv Ausdrücke parsen (könnte eine Zahl oder wieder ein LBRACE sein)
		elements = append(elements, p.parseExpr())

		p.skipStuff()
		if p.peek().Type == COMMA {
			p.next() // Komma fressen
			p.skipStuff()
		} else {
			break // Kein Komma mehr -> Ende der Liste erwartet
		}
	}

	if p.peek().Type != RBRACE {
		p.error("Erwartet '}' am Ende des Array-Literals")
	}
	p.next() // } konsumieren (RBRACE)

	return &ArrayLiteralNode{Elements: elements}
}

// Hilfsfunktion, um das gerade konsumierte Token zu erhalten
func (p *Parser) previous() Token {
	if p.pos == 0 {
		return Token{} // Sollte theoretisch nie passieren
	}
	return p.tokens[p.pos-1]
}

// Falls du isAtEnd noch nicht hast:
func (p *Parser) isAtEnd() bool {
	// Entweder prüfen wir auf ein EOF-Token oder das Ende des Slice
	return p.pos >= len(p.tokens) || p.peek().Type == EOF
}

func (p *Parser) parseBlock(context string, stopTokens ...TokenType) []Stmt {
	var stmts []Stmt
	for p.peek().Type != EOF {
		p.skipStuff()
		t := p.peek().Type

		// Deine Stopp-Logik...
		if t == END {
			return stmts
		}
		for _, stop := range stopTokens {
			if t == stop {
				return stmts
			}
		}

		if p.peek().Type == EOF {
			break
		}

		s := p.parseStmt()
		if s != nil {
			stmts = append(stmts, s)
		}
	}

	if p.peek().Type == EOF {
		p.error("Struktur-Fehler: Datei-Ende erreicht, aber der '%s'-Block ist noch offen.", context)
	}

	return stmts
}

func (p *Parser) expect(expected TokenType) Token {
	tok := p.next()
	if tok.Type != expected {
		p.error("Syntaxfehler: Erwartet wurde das Symbol '%v', aber '%v' gefunden.", expected, tok.Type)
	}
	return tok
}

func (p *Parser) expectIdentifier() string {
	tok := p.next()
	if tok.Type != IDENT {
		p.error("Syntaxfehler: Hier wird ein Name (Identifier) erwartet, aber '%v' gefunden.", tok.Type)
	}
	return tok.Value
}

func (p *Parser) match(expected TokenType) bool {
	if p.peek().Type == expected {
		p.next()
		return true
	}
	return false
}

func (p *Parser) parseForEach() Stmt {
	p.next() // FOR
	p.next() // EACH

	keyVar := p.expectIdentifier()
	var valVar string
	if p.match(COMMA) {
		valVar = p.expectIdentifier()
	}

	p.expect(IN)
	iterable := p.parseExpr()

	p.skipStuff()

	// Kopf-Newline fressen
	for p.peek().Type == NEWLINE {
		p.next()
	}

	// --- WICHTIG: LOOP DEPTH ERHÖHEN ---
	p.loopDepth++

	// Body bis NEXT lesen
	body := p.parseBlock("Next", NEXT)

	// --- WICHTIG: LOOP DEPTH VERRINGERN ---
	p.loopDepth--

	if p.peek().Type == EOF {
		p.error("Syntaxfehler: Erwartet 'Next', nach For Each.")
	}

	// NEXT konsumieren
	p.next()

	// Optional: NEXT i
	if p.peek().Type == IDENT && p.peek().Value == keyVar {
		p.next()
	}

	return &ForEachNode{
		KeyVar:   keyVar,
		ValVar:   valVar,
		Iterable: iterable,
		Body:     body,
		// Falls ForEachNode auch ein InLoop-Feld braucht:
		// InLoop: p.loopDepth > 0,
	}
}

func (p *Parser) expectEnd(expected TokenType) {
	p.skipStuff() // statt eatNewlines()

	if p.peek().Type != END {
		p.error("Struktur-Fehler: Der '%v'-Block wurde nicht geschlossen. Erwartet: 'End %v', gefunden: '%s'",
			expected, expected, p.peek().Value)
		return
	}
	p.next()
	p.skipStuff() // statt eatNewlines()

	if p.peek().Type != expected {
		p.error("Struktur-Fehler: Du versuchst mit 'End %s' zu schließen, aber der '%v'-Block ist noch offen.",
			p.peek().Value, expected)
		return
	}
	p.next()
}

func (p *Parser) parseFactor() Expr {
	tok := p.peek()

	if tok.Type == ERROR {
		p.error("%s", tok.Value)
		return nil
	}

	// NEU: NOT
	if tok.Type == NOT {
		p.next()
		right := p.parseFactor()
		return &UnaryOpNode{Op: NOT, Right: right}
	}

	// NEU: unäres Minus
	if tok.Type == MINUS {
		p.next()
		right := p.parseFactor()
		return &UnaryOpNode{Op: MINUS, Right: right}
	}

	switch tok.Type {
	case NUMBER:
		p.next()
		v, _ := strconv.ParseFloat(tok.Value, 64)
		return &NumberNode{Value: v}

	case LBRACE:
		return p.parseArrayLiteral()

	case BOOL:
		p.next()
		return Value{
			Kind: KindBool,
			Bool: strings.ToLower(tok.Value) == "true",
		}

	case STRING:
		p.next()
		return &StringNode{Value: tok.Value}

	case NEWLINE:
		p.next()
		return nil // Signalisiert dem Haupt-Loop: "Hier war nichts, mach weiter"

	case IDENT:
		p.next()
		fullName := tok.Value
		for p.peek().Type == DOT {
			p.next()
			nextTok := p.next()
			if nextTok.Type == EOF {
				p.error("Syntaxfehler: Nach dem Punkt (.) wurde ein Name erwartet, aber das Dateiende erreicht.")
			}
			fullName += "." + nextTok.Value
		}

		var expr Expr

		// Funktionsaufruf / Array-Zugriff via ()
		if p.peek().Type == LPAREN {
			p.next()
			var args []Expr
			if p.peek().Type != RPAREN {
				for {
					args = append(args, p.parseExpr())
					if p.peek().Type == COMMA {
						p.next()
						continue
					}
					break
				}
			}
			if p.next().Type != RPAREN {
				p.error("Erwartet ')' in Function-Call")
			}
			expr = &CallExprNode{Name: fullName, Args: args}
		} else {
			expr = &VarNode{Name: fullName}
		}

		// NEU: Map-Zugriff via [] – rein lesend, chainable
		for p.peek().Type == LBRACKET {
			p.next() // '[' konsumieren
			key := p.parseExpr()
			if p.next().Type != RBRACKET {
				p.error("Erwartet ']' nach Map-Zugriff")
			}
			expr = &MapIndexNode{Base: expr, Key: key}
		}

		return expr

	case LPAREN:
		p.next()
		expr := p.parseExpr()
		if p.next().Type != RPAREN {
			p.error("Erwartet ')'")
		}
		return expr

	default:
		p.error("Unerwartetes Zeichen im Ausdruck: '%s' (Typ: %v).", tok.Value, tok.Type)
		return nil
	}
}

func (p *Parser) parseTerm() Expr {
	left := p.parseFactor()
	for {
		switch p.peek().Type {
		case MUL, DIV:
			op := p.next()
			right := p.parseFactor()
			left = &BinOpNode{Left: left, Op: op.Type, Right: right}
		default:
			return left
		}
	}
}

func (p *Parser) parseConcat() Expr {
	left := p.parseAddSub() // Zuerst Mathe (+, -) prüfen
	for p.peek().Type == AMP {
		op := p.next()
		right := p.parseAddSub() // Auch rechts erst Mathe (+, -) prüfen
		left = &BinOpNode{Left: left, Op: op.Type, Right: right}
	}
	return left
}

func (p *Parser) parseAddSub() Expr {
	left := p.parseTerm()
	for {
		switch p.peek().Type {
		case PLUS, MINUS:
			op := p.next()
			right := p.parseTerm()
			left = &BinOpNode{Left: left, Op: op.Type, Right: right}
		default:
			return left
		}
	}
}

func (p *Parser) parseCompare() Expr {
	left := p.parseConcat() // <--- Jetzt wird die String-Ebene beachtet
	switch p.peek().Type {
	case LT, GT, LE, GE, EQ, NEQ:
		op := p.next()
		right := p.parseConcat() // <--- Und hier auch
		return &BinOpNode{Left: left, Op: op.Type, Right: right}
	}
	return left
}

func (p *Parser) parse() []Stmt {
	var stmts []Stmt
	for p.peek().Type != EOF {
		if p.peek().Type == NEWLINE || p.peek().Type == COMMENT {
			p.pos++
			continue
		}

		// NEU: ERROR-Token vom Lexer abfangen
		if p.peek().Type == ERROR {
			p.error("%s", p.peek().Value) // Fehlermeldung kommt direkt vom Lexer
			p.pos++
			continue
		}

		s := p.parseStmt()
		if s != nil {
			stmts = append(stmts, s)
		}
	}
	return stmts
}

func (e *ParseError) Error() string {
	return "Parse error: " + e.Message
}

func (p *Parser) parseLogical() Expr {
	left := p.parseCompare()
	for {
		tok := p.peek()
		if tok.Type == AND || tok.Type == OR {
			p.next()
			right := p.parseCompare()
			left = &BinOpNode{Left: left, Op: tok.Type, Right: right}
			continue
		}
		break
	}
	return left
}

func (p *Parser) parseExpr() Expr {
	return p.parseLogical()
}

// Parameterliste
func (p *Parser) parseParams() []ParamDef {
	var params []ParamDef
	if p.peek().Type != LPAREN {
		return params
	}
	p.next() // '(' konsumieren

	seenOptional := false

	for p.peek().Type != RPAREN {
		isOptional := false

		// "Optional" ist bei dir kein eigenes Token, sondern ein IDENT mit diesem Wert
		if p.peek().Type == IDENT && strings.EqualFold(p.peek().Value, "Optional") {
			p.next()
			isOptional = true
		}

		t := p.next()
		if t.Type != IDENT {
			p.error("Erwartet Parametername")
		}
		name := t.Value

		var def Expr
		if p.peek().Type == EQ {
			p.next() // '=' konsumieren
			def = p.parseExpr()
			isOptional = true // '= Wert' macht den Parameter implizit optional
		}

		if isOptional {
			seenOptional = true
			if def == nil {
				// "Optional x" ohne "= Wert" -> Fallback-Default
				def = &NumberNode{Value: 0}
			}
		} else if seenOptional {
			p.error("Pflichtparameter '%s' darf nicht nach einem optionalen Parameter stehen", name)
		}

		params = append(params, ParamDef{Name: name, IsOptional: isOptional, Default: def})

		if p.peek().Type == COMMA {
			p.next()
		}
	}
	p.next() // ')' konsumieren
	p.skipStuff()
	return params
}

// ---------------- Parser: Statements ----------------
func (p *Parser) parseStmt() Stmt {
	p.skipStuff()
	for p.peek().Type == NEWLINE {
		p.next()
	}
	if p.peek().Type == EOF {
		return nil
	}

	// NEU: ERROR-Token abfangen
	if p.peek().Type == ERROR {
		p.error("%s", p.peek().Value)
		return nil
	}

	switch p.peek().Type {

	case SELECT:
		p.next() // SELECT konsumieren
		// Fehler abfangen, wenn CASE fehlt
		if p.peek().Type != CASE {
			p.error("Nach 'Select' muss 'Case' folgen (z.B. Select Case x)")
		}
		p.next() // CASE konsumieren

		node := &SelectNode{Expression: p.parseExpr()}
		p.skipStuff()

		for p.peek().Type == CASE {
			p.next() // CASE konsumieren
			if p.peek().Type == ELSE {
				p.next()                                // ELSE
				node.Default = p.parseBlock("End", END) // Bis zum END des Selects
			} else {
				branch := CaseBranch{}
				// Innerhalb der Case-Schleife:
				for {
					var cond Expr

					if p.peek().Type == IS {
						p.next()       // 'Is' konsumieren
						op := p.next() // Den Operator (> , <, =, etc.)
						val := p.parseExpr()
						cond = &IsNode{Operator: op.Type, Value: val}
					} else {
						// Normalen Ausdruck parsen (z.B. die "1" in "1 To 10")
						left := p.parseExpr()

						if p.peek().Type == TO {
							p.next() // 'To' konsumieren
							right := p.parseExpr()
							cond = &RangeNode{Low: left, High: right}
						} else {
							cond = left
						}
					}

					branch.Conditions = append(branch.Conditions, cond)

					if p.peek().Type != COMMA {
						break
					}
					p.next() // Komma konsumieren
				}
				// Wichtig: Ein Case-Block endet beim nächsten CASE oder beim END
				branch.Body = p.parseBlock("Case", CASE, END)
				node.Cases = append(node.Cases, branch)
			}
			p.skipStuff()
		}
		p.expectEnd(SELECT)
		return node

	case DIM, PUBLIC:
		isPublic := p.peek().Type == PUBLIC
		p.next()
		var stmts []Stmt
		inLoopContext := p.loopDepth > 0

		for {
			nameTok := p.next()

			if nameTok.Type != IDENT {
				p.error("Erwartet Variablennamen nach DIM/PUBLIC")
			}

			name := nameTok.Value

			if p.peek().Type == LPAREN {
				p.next() // (
				var size1 Expr
				var size2 Expr

				if p.peek().Type != RPAREN {
					size1 = p.parseExpr()

					// --- NEU: Prüfen, ob ein Komma für die 2. Dimension folgt ---
					if p.peek().Type == COMMA {
						p.next() // Komma überspringen
						size2 = p.parseExpr()
					}
				}

				// Hier muss jetzt zwingend die schließende Klammer kommen
				if p.next().Type != RPAREN {
					p.error("Erwartet ')' nach Array-Definition")
				}

				if isPublic {
					// Hinweis: Stelle sicher, dass PublicArrayNode auch Size2 hat!
					stmts = append(stmts, &PublicArrayNode{Name: name, Size1: size1, Size2: size2})
				} else {
					stmts = append(stmts, &DimArrayNode{
						Name:   name,
						Size1:  size1,
						Size2:  size2, // Jetzt wird Size2 mitgegeben
						InLoop: inLoopContext,
					})
				}
			} else {
				// ... (Rest deiner Logik für normale Variablen: init = 0, etc.)
				var init Expr = &NumberNode{Value: 0}
				if p.peek().Type == EQ {
					p.next() // =
					if p.peek().Type == LBRACE {
						init = p.parseArrayLiteral()
					} else {
						init = p.parseExpr()
					}
				}

				if isPublic {
					stmts = append(stmts, &PublicNode{Name: name, Value: init})
				} else {
					stmts = append(stmts, &AssignNode{
						Name:          name,
						Value:         init,
						IsDeclaration: true,
						InLoop:        inLoopContext,
					})
				}
			}

			if p.peek().Type != COMMA {
				break
			}
			p.next()
		}

		if len(stmts) == 1 {
			return stmts[0]
		}
		return &MultiStmtNode{Stmts: stmts}

	case PRINT:
		p.next() // 'Print' überspringen

		// 1. Das erste Argument parsen (z.B. 'a')
		valExpr := p.parseExpr()
		node := &PrintNode{Value: valExpr}

		// 2. Schauen, ob ein Komma folgt
		// Wir nutzen p.peek(), um zu sehen was als nächstes kommt
		if p.peek().Type == COMMA {
			p.next()                   // Das Komma verbrauchen (überspringen)
			node.Color = p.parseExpr() // Das zweite Argument (Farbe) parsen
		}

		return node

	case IF:
		p.next() // IF
		cond := p.parseExpr()
		if p.next().Type != THEN {
			p.error("Erwartet THEN nach IF")
		}
		p.skipStuff()

		branches := []struct {
			Cond Expr
			Body []Stmt
		}{{Cond: cond, Body: p.parseBlock("If", ELSEIF, ELSE, END)}}

		// --- HIER IST DIE RETTUNG ---
		p.skipStuff() // Putzt Newlines/Kommentare weg, damit peek() das ELSEIF sieht!

		for p.peek().Type == ELSEIF {
			p.next() // ELSEIF
			eCond := p.parseExpr()
			if p.next().Type != THEN {
				p.error("Erwartet THEN nach ELSEIF")
			}
			p.skipStuff()

			branches = append(branches, struct {
				Cond Expr
				Body []Stmt
			}{Cond: eCond, Body: p.parseBlock("If", ELSEIF, ELSE, END)})

			p.skipStuff() // Und hier nochmal putzen für den nächsten Durchlauf oder das ELSE
		}

		var elseBody []Stmt
		if p.peek().Type == ELSE {
			p.next() // ELSE
			p.skipStuff()
			elseBody = p.parseBlock("If", ELSEIF, ELSE, END)
			p.skipStuff() // Putzen vor dem finalen End If
		}

		p.expectEnd(IF)
		return &IfNode{Branches: branches, Else: elseBody}

	case FOR:
		// Sicherer Lookahead für FOR EACH
		if p.peekNext().Type == EACH {
			return p.parseForEach()
		}

		// --- Dein originales FOR mit Loop-Tracking ---
		p.next() // FOR
		varName := p.next().Value
		p.expect(EQ)
		start := p.parseExpr()
		p.expect(TO)
		end := p.parseExpr()

		p.skipStuff()

		step := 1.0
		if p.peek().Type == STEP {
			p.next() // STEP
			multiplier := 1.0
			if p.peek().Type == MINUS {
				p.next() // '-'
				multiplier = -1.0
			}
			if nv, ok := p.parseExpr().(*NumberNode); ok {
				step = nv.Value * multiplier
			}
		}

		// --- WICHTIG: Loop-Tracking starten ---
		p.loopDepth++
		p.loopStack = append(p.loopStack, FOR)

		body := p.parseBlock("For", NEXT)

		p.loopStack = p.loopStack[:len(p.loopStack)-1]
		p.loopDepth--

		if p.peek().Type == EOF {
			p.error("Syntaxfehler: Erwartet 'Next', nach For")
		}

		p.next() // NEXT
		if p.peek().Type == IDENT && p.peek().Value == varName {
			p.next()
		}

		return &ForNode{
			VarName: varName,
			Start:   start,
			End:     end,
			Step:    step,
			Body:    body,
		}

	case SUB:
		p.next() // SUB überspringen
		nameTok := p.next()
		if nameTok.Type != IDENT {
			p.error("Erwartet Sub-Name nach SUB")
		}

		if len(p.procStack) > 0 {
			p.error("Eine 'Sub'-Deklaration ist innerhalb einer anderen Sub/Function nicht erlaubt (umgebend: '%v').", p.procStack[len(p.procStack)-1])
		}

		// Parameterliste parsen (z.B. "(a, b)")
		params := p.parseParams()

		// Der "Staubsauger": Liest alle Statements bis zum Wort 'END'
		p.procStack = append(p.procStack, SUB)
		body := p.parseBlock("Sub", END)
		p.procStack = p.procStack[:len(p.procStack)-1]

		// Prüft, ob danach wirklich 'Sub' folgt (für 'End Sub')
		p.expectEnd(SUB)

		return &SubNode{
			Name:   nameTok.Value,
			Params: params,
			Body:   body,
		}

	case FUNCTION:
		p.next() // FUNCTION überspringen
		nameTok := p.next()
		if nameTok.Type != IDENT {
			p.error("Erwartet Funktionsname nach FUNCTION")
		}

		if len(p.procStack) > 0 {
			p.error("Eine 'Function'-Deklaration ist innerhalb einer anderen Sub/Function nicht erlaubt (umgebend: '%v').", p.procStack[len(p.procStack)-1])
		}

		params := p.parseParams()

		// Optional: VB.NET 'As Type' Teil überspringen, falls vorhanden
		if p.peek().Value == "as" {
			p.next() // 'as'
			p.next() // 'Integer', 'String', etc.
		}

		// Nutzt die neue parseBlock-Logik
		// Wir lesen alles bis zum Token 'END'
		p.procStack = append(p.procStack, FUNCTION)
		body := p.parseBlock("Function", END)
		p.procStack = p.procStack[:len(p.procStack)-1]

		// Validiert das Zwei-Wort-Ende: 'End Function'
		p.expectEnd(FUNCTION)

		return &FuncNode{
			Name:   nameTok.Value,
			Params: params,
			Body:   body,
		}

	case RETURN:
		p.next()
		val := p.parseExpr()
		return &ReturnNode{Value: val}

	case WHILE:
		p.next() // WHILE
		cond := p.parseExpr()

		p.loopDepth++
		p.loopStack = append(p.loopStack, WHILE)

		body := p.parseBlock("While", END)

		p.loopStack = p.loopStack[:len(p.loopStack)-1]
		p.loopDepth--

		p.expectEnd(WHILE)
		return &WhileNode{Condition: cond, Body: body}

	case DO:
		p.next()
		p.skipStuff()
		var cond Expr
		isUntil := false
		checkAtEnd := false

		// Kopf-Bedingung
		if p.peek().Type == WHILE {
			p.next()
			cond = p.parseExpr()
		} else if p.peek().Type == UNTIL { // Token-Typ statt String
			p.next()
			cond = p.parseExpr()
			isUntil = true
		}

		p.loopDepth++
		p.loopStack = append(p.loopStack, DO)

		body := p.parseBlock("Do", LOOP)

		p.loopStack = p.loopStack[:len(p.loopStack)-1]
		p.loopDepth--

		p.next() // LOOP

		// Fuß-Bedingung
		if cond == nil && (p.peek().Type == WHILE || p.peek().Type == UNTIL) { // Token-Typ
			checkAtEnd = true
			if p.peek().Type == UNTIL {
				isUntil = true
			}
			p.next()
			cond = p.parseExpr()
		}

		return &DoLoopNode{
			Condition:  cond,
			Body:       body,
			IsUntil:    isUntil,
			CheckAtEnd: checkAtEnd,
		}

	case EXIT:
		p.next() // 'Exit' überspringen

		next := p.peek()
		exitType := ""

		switch next.Type {
		case FOR, WHILE, DO, SUB, FUNCTION:
			exitType = next.Value
			p.next()
		default:
			// optional: leer lassen oder Fehler, je nach gewünschtem Verhalten
			// z.B. Exit alleine = Exit Loop
			exitType = ""
		}

		return &ExitNode{ExitType: exitType}

	case CONTINUE:
		p.next() // CONTINUE

		if p.loopDepth == 0 {
			p.error("'Continue' darf nur innerhalb einer Schleife verwendet werden.")
		}

		continueType := ""

		switch p.peek().Type {
		case FOR, WHILE, DO:
			continueType = p.peek().Value
			current := p.loopStack[len(p.loopStack)-1]
			if p.peek().Type != current {
				p.error("'Continue %s' passt nicht zur umgebenden Schleife (aktuell: '%v').", continueType, current)
			}
			p.next()
		}

		return &ContinueNode{
			ContinueType: continueType,
		}

	case INCLUDE:
		p.next() // include konsumieren

		// Pfad parsen
		pathExpr := p.parseExpr()

		pathNode, ok := pathExpr.(*StringNode)
		if !ok {
			p.error("Nach 'include' wird ein Dateipfad als String erwartet.")
		}

		path := pathNode.Value

		// Prüfen, ob Include-Datei existiert
		if _, err := os.Stat(path); os.IsNotExist(err) {
			// Optionales Include: Datei nicht vorhanden -> ignorieren
			return &MultiStmtNode{
				Stmts: []Stmt{},
			}
		}

		// Datei laden
		content, err := os.ReadFile(path)
		if err != nil {
			p.error("Konnte Datei '%s' nicht öffnen: %v", path, err)
		}

		// #use aus Include-Datei entfernen
		contentLines := strings.Split(string(content), "\n")

		contentLines, modules := ExtractUse(contentLines)

		// Module in bestehender Umgebung laden
		if len(modules) > 0 {
			LoadModules(p.env, modules)
		}

		// Bereinigten Code erneut lexen
		includedTokens := tokenize(strings.Join(contentLines, "\n"))

		// Gleicher Parser-Kontext
		subParser := &Parser{
			tokens:    includedTokens,
			pos:       0,
			loopDepth: p.loopDepth,
			env:       p.env,
		}

		includedStmts := subParser.parse()

		// Fehler aus Include übernehmen
		if len(subParser.Errors) > 0 {
			p.Errors = append(p.Errors, subParser.Errors...)
			p.error("Fehler in inkludierter Datei '%s'", path)
		}

		return &MultiStmtNode{
			Stmts: includedStmts,
		}

	case IDENT:
		// 1. Namen zusammenbauen (deine bewährte Logik)
		nameTok := p.next()
		fullName := nameTok.Value
		for p.peek().Type == DOT {
			p.next()
			nextTok := p.next()
			if nextTok.Type == EOF {
				p.error("Syntaxfehler: Nach dem Punkt (.) wurde ein Name erwartet, aber das Dateiende erreicht.")
			}
			if nextTok.Type == ERROR {
				p.error("%s", nextTok.Value)
			}
			fullName += "." + nextTok.Value
		}
		// NEU: entry["path"] = ... explizit als ungültiges Zuweisungsziel abfangen
		if p.peek().Type == LBRACKET {
			p.rejectBracketAssignTarget()
			// Kein gültiges Statement, wenn wir hier ankommen (z.B. entry["path"] allein als Zeile)
			p.error("Ein Ausdruck mit '[...]' kann nicht als eigenständige Anweisung stehen.")
		}
		// 2. Funktionsaufruf oder Array-Zugriff?
		if p.peek().Type == LPAREN {
			p.next() // '('
			var args []Expr
			if p.peek().Type != RPAREN {
				for {
					args = append(args, p.parseExpr())
					if p.peek().Type == COMMA {
						p.next()
						continue
					}
					break
				}
			}
			if p.next().Type != RPAREN {
				p.error("Erwartet ')' in Call/Index")
			}

			// NEU: grp(i)["path"] = ... ebenfalls abfangen, bevor die normale Array-Zuweisung geprüft wird
			if p.peek().Type == LBRACKET {
				p.rejectBracketAssignTarget()
				p.error("Ein Ausdruck mit '[...]' kann nicht als eigenständige Anweisung stehen.")
			}

			// Fall A: Zuweisung an Array-Index -> myArr(5) = "Wert"
			if p.peek().Type == EQ {
				p.next() // '='
				if len(args) == 0 {
					p.error("Array-Index erwartet")
				}

				// 1. Das Node-Objekt vorbereiten
				node := &ArrayAssignNode{
					Name:  fullName,
					Index: args[0],
					Value: p.parseExpr(), // Hier wird der Wert NACH dem '=' geparst
				}

				// 2. WICHTIG: Das zweite Argument für 2D-Arrays mitschicken
				if len(args) > 1 {
					node.Index2 = args[1]
				}

				// 3. Das vorbereitete Objekt zurückgeben (NICHT neu erstellen!)
				return node
			}

			// Fall B: Nackter Funktionsaufruf -> file.Write("...")
			// Wir geben einen CallNode zurück, der als Statement fungiert
			return &CallNode{Name: fullName, Args: args}
		}

		// 3. Einfache Variablen-Zuweisung -> x = 10
		if p.peek().Type == EQ {
			p.next() // '='
			val := p.parseExpr()
			p.skipStuff()
			return &AssignNode{Name: fullName, Value: val}
		}

		// 3b. Kombinierte Zuweisung -> x += 10
		if p.peek().Type == PLUS_ASSIGN ||
			p.peek().Type == MINUS_ASSIGN ||
			p.peek().Type == MUL_ASSIGN ||
			p.peek().Type == DIV_ASSIGN {

			op := p.next().Type
			val := p.parseExpr()
			p.skipStuff()

			return &CompoundAssignNode{
				Left:  &VarNode{Name: fullName},
				Op:    op,
				Right: val,
			}
		}

		// 4. Fehlerfall: Alles, was kein Befehl ist
		tok := p.peek()
		prevTok := p.tokens[p.pos-1] // Das ist das 'ddd'

		if tok.Type == EOF || tok.Type == NEWLINE {
			p.error("Die Anweisung mit '%s' ist unvollständig. Hinweis: Ein Name darf nicht alleine stehen.", prevTok.Value)
			return nil
		}

		p.error("Unerwartetes Token '%s' nach '%s'.", tok.Value, prevTok.Value)
		return nil

	default:
		t := p.peek()
		p.error("Ich kann mit dem Zeichen '%s' (Typ: %v) an dieser Stelle nichts anfangen.", t.Value, t.Type)
		return nil
	}
}
