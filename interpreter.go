package main

// ---------------- Token ----------------
type TokenType int

const (
	EOF TokenType = iota
	IDENT
	NUMBER
	STRING
	PLUS
	MINUS
	MUL
	DIV
	ASSIGN
	LPAREN
	RPAREN
	DIM
	PRINT
	IF
	THEN
	ELSE
	ELSEIF
	LT
	GT
	LE
	GE
	EQ
	NEQ
	FOR
	NEXT
	TO
	IS
	STEP
	AMP
	SUB
	COMMA
	FUNCTION
	RETURN
	DOT
	AND
	BOOL
	OR
	NOT
	WHILE
	DO
	LOOP
	EXIT
	SELECT
	CASE
	END
	EACH
	IN
	NEWLINE
	INCLUDE
	COMMENT
	LBRACKET // [
	RBRACKET // ]
	PUBLIC
	LBRACE // {
	RBRACE // }
	PLUS_ASSIGN
	MINUS_ASSIGN
	MUL_ASSIGN
	DIV_ASSIGN
	ERROR
	UNTIL
)

type Token struct {
	Type  TokenType
	Value string
}

type UnaryOpNode struct {
	Op    TokenType
	Right Expr
}

func (n *UnaryOpNode) Pos() int { return 0 } // Oder echte Position, falls du sie speicherst

type ParseError struct {
	Message string
}

type RuntimeError struct {
	Message string
}

func (e *RuntimeError) Error() string {
	return "Runtime error: " + e.Message
}

// ---------------- AST ----------------
type Expr interface{}
type NumberNode struct{ Value float64 }
type StringNode struct{ Value string }
type VarNode struct {
	Name   string
	Index1 Expr // nil, wenn keine Indizes
	Index2 Expr // nil für 1D-Array
}
type CallExprNode struct {
	Name string
	Args []Expr
}
type ScanResult struct {
	Files int
	Dirs  int
	Size  int64
}
type ArrayAssignNode struct {
	Name   string
	Index  Expr
	Index2 Expr // Zweite Dimension (Spalte) -> Kann nil sein bei 1D
	Value  Expr
}

type ArrayAccessNode struct {
	Name  string
	Index Expr
}

type BinOpNode struct {
	Left  Expr
	Op    TokenType
	Right Expr
}
type DimArrayNode struct {
	Name   string
	Size1  Expr
	Size2  Expr // nil für 1D-Array
	InLoop bool
}
type Stmt interface{}

type AssignNode struct {
	Name          string
	Value         Expr
	IsDeclaration bool
	InLoop        bool
}
type PrintNode struct {
	Value Expr // Der Text/Wert
	Color Expr // <-- Dieses Feld muss neu rein!
}

type IfNode struct {
	Branches []struct {
		Cond Expr
		Body []Stmt
	}
	Else []Stmt
}
type ForNode struct {
	VarName string
	Start   Expr
	End     Expr
	Step    float64
	Body    []Stmt
}
type SubNode struct {
	Name   string
	Params []ParamDef
	Body   []Stmt
}

type FuncNode struct {
	Name   string
	Params []ParamDef
	Body   []Stmt
}
type CallNode struct {
	Name string
	Args []Expr
}

// Für: While ... EndWhile
type WhileNode struct {
	Condition Expr
	Body      []Stmt
}

// Für: Do While ... Loop ODER Do ... Loop While
type DoLoopNode struct {
	Condition  Expr
	Body       []Stmt
	CheckAtEnd bool // False = Prüfung oben, True = Prüfung unten
	IsUntil    bool // NEU: Speichert, ob es 'Until' statt 'While' ist
	//IsPostCondition bool // Optional: Falls du Do...Loop While (fußgesteuert) nutzt
}

// Dummy-Methoden, damit sie das Stmt-Interface erfüllen
func (n *WhileNode) stmtNode()  {}
func (n *DoLoopNode) stmtNode() {}

type ReturnNode struct{ Value Expr }

func (t TokenType) String() string {
	switch t {
	case EOF:
		return "EOF"
	case IDENT:
		return "IDENT"
	case NUMBER:
		return "NUMBER"
	case STRING:
		return "STRING"
	case DIM:
		return "DIM"
	case PRINT:
		return "PRINT"
	//case USE:
	//	return "USE"
	case IF:
		return "IF"
	case THEN:
		return "THEN"
	case ELSE:
		return "ELSE"
	case ELSEIF:
		return "ELSEIF"
	case FOR:
		return "FOR"
	case NEXT:
		return "NEXT"
	case SUB:
		return "SUB"
	case FUNCTION:
		return "FUNCTION"
	case INCLUDE:
		return "INCLUDE"
	case WHILE:
		return "WHILE"
	case DO:
		return "DO"
	case TO:
		return "TO"
	case IS:
		return "IS"
	case LOOP:
		return "LOOP"
	case SELECT:
		return "SELECT"
	case CASE:
		return "CASE"
	case END:
		return "END"
	case NEWLINE:
		return "NEWLINE"
	case PUBLIC:
		return "PUBLIC"
	// Mathematische Operatoren
	case ERROR:
		return "ERROR"
	case PLUS:
		return "+"
	case MINUS:
		return "-"
	case MUL:
		return "*"
	case DIV:
		return "/"
	case ASSIGN:
		return "="
	case PLUS_ASSIGN:
		return "+="
	case MINUS_ASSIGN:
		return "-="
	case MUL_ASSIGN:
		return "*="
	case DIV_ASSIGN:
		return "/="
	case EQ:
		return "="
	case NEQ:
		return "<>"
	case LT:
		return "<"
	case GT:
		return ">"
	case LE:
		return "<="
	case GE:
		return ">="
	case AMP:
		return "&"
	case COMMA:
		return ","
	default:
		return "TOKEN" // Fallback
	}
}
