package main

import (
	"fmt"
	"strings"
	"unicode"
)

var doubleOperators = map[string]TokenType{
	"+=": PLUS_ASSIGN,
	"-=": MINUS_ASSIGN,
	"*=": MUL_ASSIGN,
	"/=": DIV_ASSIGN,

	"<>": NEQ,
	"<=": LE,
	">=": GE,
}

var keywords = map[string]TokenType{
	"dim":      DIM,
	"public":   PUBLIC,
	"print":    PRINT,
	"if":       IF,
	"then":     THEN,
	"else":     ELSE,
	"elseif":   ELSEIF,
	"for":      FOR,
	"next":     NEXT,
	"each":     EACH,
	"is":       IS,
	"to":       TO,
	"step":     STEP,
	"sub":      SUB,
	"function": FUNCTION,
	"return":   RETURN,
	"in":       IN,
	"include":  INCLUDE,

	"true":  BOOL,
	"false": BOOL,
	"and":   AND,
	"or":    OR,
	"not":   NOT,

	"while":    WHILE,
	"do":       DO,
	"loop":     LOOP,
	"until":    UNTIL,
	"exit":     EXIT,
	"continue": CONTINUE,

	"case":   CASE,
	"select": SELECT,
	"end":    END,
}

// ---------------- Lexer ----------------
func tokenize(input string) []Token {
	var tokens []Token

	runes := []rune(input)
	i := 0
	line := 1

	emit := func(t TokenType, value string) {
		tokens = append(tokens, Token{
			Type:  t,
			Value: value,
		})
	}

	emitError := func(format string, args ...any) {
		emit(ERROR, fmt.Sprintf("Zeile %d: %s",
			line,
			fmt.Sprintf(format, args...),
		))
	}

	for i < len(runes) {
		ch := runes[i]

		// 1. BLOCK-KOMMENTAR /' ... '/
		if ch == '/' && i+1 < len(runes) && runes[i+1] == '\'' {

			i += 2
			start := i

			for i+1 < len(runes) &&
				!(runes[i] == '\'' && runes[i+1] == '/') {
				i++
			}

			if i+1 >= len(runes) {
				emitError("Nicht geschlossener Block-Kommentar")
				break
			}

			commentText := string(runes[start:i])
			emit(COMMENT, commentText)

			i += 2
			continue
		}

		// Einzeiliger Kommentar
		if ch == '\'' {
			i++
			start := i

			for i < len(runes) && runes[i] != '\n' {
				i++
			}

			emit(COMMENT, string(runes[start:i]))
			continue
		}

		if i+1 < len(runes) {
			op := string([]rune{ch, runes[i+1]})

			if tok, ok := doubleOperators[op]; ok {
				emit(tok, op)
				i += 2
				continue
			}
		}

		switch {
		case ch == '\r':
			// Windows CRLF ignorieren
			i++

		case ch == '\n':
			emit(NEWLINE, "\n")
			line++
			i++

		case unicode.IsSpace(ch):
			i++

		// Operatoren
		case ch == '+':
			emit(PLUS, string(ch))
			i++
		case ch == '-':
			emit(MINUS, string(ch))
			i++
		case ch == '*':
			emit(MUL, string(ch))
			i++
		case ch == '/':
			emit(DIV, string(ch))
			i++
		case ch == '=':
			emit(EQ, string(ch))
			i++
		case ch == '(':
			emit(LPAREN, string(ch))
			i++
		case ch == ')':
			emit(RPAREN, string(ch))
			i++
		case ch == '{':
			emit(LBRACE, string(ch))
			i++
		case ch == '}':
			emit(RBRACE, string(ch))
			i++
		case ch == '[':
			emit(LBRACKET, string(ch))
			i++
		case ch == ']':
			emit(RBRACKET, string(ch))
			i++
		case ch == '&':
			emit(AMP, string(ch))
			i++
		case ch == ',':
			emit(COMMA, string(ch))
			i++
		case ch == '.':
			emit(DOT, string(ch))
			i++

			// Vergleichsoperatoren
		case ch == '<':
			emit(LT, "<")
			i++

		case ch == '>':
			emit(GT, ">")
			i++

		// Strings
		case ch == '"':
			j := i + 1
			var sb strings.Builder
			for j < len(runes) {
				if runes[j] == '\n' || runes[j] == '\r' {
					// Newline mitten im String = unterminiert, sofort abbrechen
					break
				}
				if runes[j] == '"' {
					if j+1 < len(runes) && runes[j+1] == '"' {
						sb.WriteRune('"')
						j += 2
						continue
					}
					break
				}
				sb.WriteRune(runes[j])
				j++
			}
			if j >= len(runes) || runes[j] != '"' {
				emitError("Unterminierter String")
				i = j
				continue
			}
			emit(STRING, sb.String())
			i = j + 1
			// Zahlen
			// Zahlen / Identifier mit führender Zahl
		case unicode.IsDigit(ch):
			j := i

			// Zuerst die Ziffern lesen
			for j < len(runes) && unicode.IsDigit(runes[j]) {
				j++
			}

			// Falls direkt ein Buchstabe oder '_' folgt:
			// -> Identifier statt Zahl
			//
			// Beispiele:
			//   7z
			//   7zip
			//   123abc
			//   7_z
			if j < len(runes) && (unicode.IsLetter(runes[j]) || runes[j] == '_') {
				for j < len(runes) &&
					(unicode.IsLetter(runes[j]) ||
						unicode.IsDigit(runes[j]) ||
						runes[j] == '_') {
					j++
				}

				word := string(runes[i:j])
				emit(IDENT, word)
				i = j
				continue
			}

			// Normale Zahl, eventuell mit Dezimalpunkt
			if j < len(runes) && runes[j] == '.' {
				j++

				for j < len(runes) && unicode.IsDigit(runes[j]) {
					j++
				}
			}

			emit(NUMBER, string(runes[i:j]))
			i = j
		// Identifier / Keywords
		case unicode.IsLetter(ch) || ch == '_':
			// 1. Check auf Zeilenfortsetzung
			if ch == '_' {
				j := i + 1
				// Whitespace nach _ ignorieren
				for j < len(runes) && (runes[j] == ' ' || runes[j] == '\t') {
					j++
				}
				// Kommentar nach _ ignorieren
				if j < len(runes) && runes[j] == '\'' {
					for j < len(runes) && runes[j] != '\n' && runes[j] != '\r' {
						j++
					}
				}
				// Wenn jetzt ein Newline kommt -> Zeilenfortsetzung!
				if j < len(runes) && (runes[j] == '\n' || runes[j] == '\r') {
					i = j
					if runes[i] == '\r' && i+1 < len(runes) && runes[i+1] == '\n' {
						i++
					}
					i++      // Überspringe das Newline-Zeichen
					continue // Nächstes Token in der neuen Zeile suchen
				}

				// Falls nach dem '_' direkt ein Buchstabe oder Zahl kommt,
				// ist es KEINE Fortsetzung, sondern ein Identifier (z.B. _temp)
				// Wir lassen ihn einfach in den normalen Identifier-Scanner laufen.
			}

			// 2. Normaler Identifier oder Keyword Scanner
			j := i
			for j < len(runes) && (unicode.IsLetter(runes[j]) || unicode.IsDigit(runes[j]) || runes[j] == '_') {
				j++
			}

			word := string(runes[i:j])
			lw := strings.ToLower(word)

			if tok, ok := keywords[lw]; ok {
				emit(tok, word)
			} else {
				emit(IDENT, word)
			}

			i = j
		}
	}

	emit(EOF, "")
	return tokens
}
