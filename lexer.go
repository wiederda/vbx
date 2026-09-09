package main

import (
	"fmt"
	"strings"
	"unicode"
)

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
	// Grobe Heuristik: ~1 Token pro 3 Zeichen, spart Reallocations bei größeren Skripten.
	tokens := make([]Token, 0, len(input)/3+16)

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

		// Doppel-Operatoren (+=, -=, *=, /=, <>, <=, >=)
		// Nur für die relevanten Startzeichen prüfen, statt für jedes Zeichen
		// im Skript eine []rune->string-Allokation + Map-Lookup zu machen.
		if i+1 < len(runes) {
			switch ch {
			case '+', '-', '*', '/', '<', '>':
				next := runes[i+1]
				switch {
				case ch == '+' && next == '=':
					emit(PLUS_ASSIGN, "+=")
					i += 2
					continue
				case ch == '-' && next == '=':
					emit(MINUS_ASSIGN, "-=")
					i += 2
					continue
				case ch == '*' && next == '=':
					emit(MUL_ASSIGN, "*=")
					i += 2
					continue
				case ch == '/' && next == '=':
					emit(DIV_ASSIGN, "/=")
					i += 2
					continue
				case ch == '<' && next == '>':
					emit(NEQ, "<>")
					i += 2
					continue
				case ch == '<' && next == '=':
					emit(LE, "<=")
					i += 2
					continue
				case ch == '>' && next == '=':
					emit(GE, ">=")
					i += 2
					continue
				}
			}
		}

		// NEU: $"..." interpolierter String
		if ch == '$' && i+1 < len(runes) && runes[i+1] == '"' {
			i += 2 // $" konsumieren
			tokenizeInterpolatedString(runes, &i, line, emit, emitError)
			continue
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

			// Fast-Path: strings.ToLower alloziert auch dann eine neue Kopie,
			// wenn das Wort schon komplett klein geschrieben ist (Normalfall bei
			// Keywords wie "if", "then", "end"). Erst prüfen, ob überhaupt ein
			// Großbuchstabe drin ist, bevor wir die Kopie erzwingen.
			lw := word
			for _, r := range word {
				if unicode.IsUpper(r) {
					lw = strings.ToLower(word)
					break
				}
			}

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

// tokenizeInterpolatedString liest den Rest eines $"..."-Strings ab der Position
// direkt nach dem öffnenden ", zerlegt ihn in Text- und {Ausdruck}-Segmente und
// emittiert sie als Kette von STRING/AMP/LPAREN/.../RPAREN-Tokens - der Parser
// sieht am Ende nichts anderes als eine normale &-Verkettung.
func tokenizeInterpolatedString(runes []rune, i *int, line int, emit func(TokenType, string), emitError func(string, ...any)) {
	var segments []string     // gesammelte Text-Segmente
	var exprSegments []string // gesammelte {Ausdruck}-Segmente, parallel dazu
	var sb strings.Builder
	segType := make([]bool, 0) // false=Text, true=Ausdruck, in Reihenfolge

	flushText := func() {
		segments = append(segments, sb.String())
		exprSegments = append(exprSegments, "")
		segType = append(segType, false)
		sb.Reset()
	}

	for *i < len(runes) {
		ch := runes[*i]

		if ch == '\n' || ch == '\r' {
			emitError("Unterminierter interpolierter String")
			return
		}

		if ch == '"' {
			// Escaped ""?
			if *i+1 < len(runes) && runes[*i+1] == '"' {
				sb.WriteRune('"')
				*i += 2
				continue
			}
			break // Ende des Strings
		}

		if ch == '{' {
			// Escaped {{?
			if *i+1 < len(runes) && runes[*i+1] == '{' {
				sb.WriteRune('{')
				*i += 2
				continue
			}
			flushText()
			*i++ // '{' konsumieren

			// Ausdruck bis zur passenden '}' einsammeln (Klammertiefe zählen,
			// damit z.B. {foo(a, b)} nicht an der ersten inneren ')' abbricht)
			var exprSb strings.Builder
			depth := 0
			for *i < len(runes) {
				c := runes[*i]
				if c == '}' && depth == 0 {
					break
				}
				if c == '{' {
					depth++
				}
				if c == '}' {
					depth--
				}
				exprSb.WriteRune(c)
				*i++
			}
			if *i >= len(runes) || runes[*i] != '}' {
				emitError("Erwartet '}' nach interpoliertem Ausdruck")
				return
			}
			*i++ // '}' konsumieren

			segments = append(segments, "")
			exprSegments = append(exprSegments, exprSb.String())
			segType = append(segType, true)
			continue
		}

		if ch == '}' {
			// Escaped }}?
			if *i+1 < len(runes) && runes[*i+1] == '}' {
				sb.WriteRune('}')
				*i += 2
				continue
			}
			emitError("Unerwartete '}' im interpolierten String")
			return
		}

		sb.WriteRune(ch)
		*i++
	}

	if *i >= len(runes) || runes[*i] != '"' {
		emitError("Unterminierter interpolierter String")
		return
	}
	*i++ // schließendes '"' konsumieren
	flushText()

	// Jetzt als (Text & (Ausdruck) & Text & (Ausdruck) & ...) emittieren
	emit(LPAREN, "(")
	first := true
	for idx, isExpr := range segType {
		if !first {
			emit(AMP, "&")
		}
		first = false
		if isExpr {
			emit(LPAREN, "(")
			subTokens := tokenize(exprSegments[idx])
			for _, t := range subTokens {
				if t.Type == EOF {
					continue
				}
				emit(t.Type, t.Value)
			}
			emit(RPAREN, ")")
		} else {
			emit(STRING, segments[idx])
		}
	}
	if first {
		// Leerer String $""
		emit(STRING, "")
	}
	emit(RPAREN, ")")
}
