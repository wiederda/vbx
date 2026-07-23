// ------------------------
// stdlib_string.go
// ------------------------

package main

import (
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// InitStringFunctions registriert String-Funktionen
func InitStringFunctions() {
	if builtins == nil {
		builtins = make(map[string]BuiltinInfo)
	}

	ns := "string."

	// Wandelt den gesamten String in Hex-Darstellung (ASCII/UTF8 -> Hex)
	Register(ns+"HexEncode", "string", "val", "Wandelt Text in Hex-Strings um (ABC -> 414243).", func(args []Value) Value {
		return StrVal(hex.EncodeToString([]byte(ToString(MustValue(args, 0)))))
	})

	// Wandelt Hex-Strings zurück in Text (414243 -> ABC)
	Register(ns+"HexDecode", "string", "hexStr", "Wandelt Hex-Strings zurück in Text.", func(args []Value) Value {
		s := ToString(MustValue(args, 0))
		data, err := hex.DecodeString(s)
		if err != nil {
			return ErrorVal(err.Error())
		}
		return StrVal(string(data))
	})

	// CleanLines: Trimm jedes Zeilenende, behält aber bewusste Leerzeilen
	Register(ns+"CleanLines", "string", "s", "Trimmt jede Zeile einzeln, behält aber Leerzeilen bei.", func(args []Value) Value {
		if len(args) < 1 {
			return StrVal("")
		}

		input := strings.ReplaceAll(ToString(args[0]), "\r\n", "\n")
		lines := strings.Split(input, "\n")

		for i, line := range lines {
			lines[i] = strings.TrimRight(line, " \t")
		}

		return StrVal(strings.Trim(strings.Join(lines, "\n"), "\n"))
	})

	// CleanAllLines: Entfernt ALLES, was leer ist (kompakter Block)
	Register(ns+"CleanAllLines", "string", "s", "Entfernt alle Leerzeilen und trimmt verbleibende Zeilen.", func(args []Value) Value {
		if len(args) < 1 {
			return StrVal("")
		}
		input := strings.ReplaceAll(ToString(args[0]), "\r\n", "\n")
		lines := strings.Split(input, "\n")

		var result []string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if len(trimmed) > 0 {
				result = append(result, trimmed)
			}
		}
		return StrVal(strings.Join(result, "\n"))
	})

	Register(ns+"CharAt", "string", "str, index", "Gibt ein einzelnes Zeichen zurück", func(args []Value) Value {
		if len(args) < 2 {
			return ErrorVal("usage: string.CharAt(str, index)")
		}

		str := ToString(args[0])
		idx := int(toNumVal(args[1]))

		runes := []rune(str)

		// 1-basiert
		if idx < 1 || idx > len(runes) {
			return StrVal("")
		}

		return StrVal(string(runes[idx-1]))
	})

	// string.Join(array, separator) -> Verbindet Array-Elemente zu einem String
	Register(ns+"Join", "string", "arr, [sep]", "Verbindet Array-Elemente.", func(args []Value) Value {
		if len(args) < 1 || args[0].Kind != KindArr {
			return StrVal("")
		}

		sep := "\n" // Default
		if len(args) >= 2 {
			sep = ToString(args[1])
		}

		var elements []string
		for _, v := range args[0].Arr {
			elements = append(elements, ToString(v))
		}

		return StrVal(strings.Join(elements, sep))
	})

	Register(ns+"Choose", "string", "n,v...", "Wählt ein Element basierend auf einem Index (1-basiert)", func(args []Value) Value {
		// Mindestens Index und eine Option müssen vorhanden sein
		if len(args) < 2 {
			return Value{Kind: KindUndefined}
		}

		// VB.NET konvertiert den Index intern zu Integer (Abrunden)
		// Wir nehmen den Float-Wert und machen einen int daraus
		index := int(args[0].Num)

		// WICHTIG: 1-basierter Index Check
		// args[0] = Index
		// args[1] = Option 1
		// args[2] = Option 2 ...
		if index < 1 || index >= len(args) {
			// VB.NET gibt bei falschem Index 'Nothing' zurück
			return Value{Kind: KindUndefined}
		}

		// Rückgabe der gewählten Option
		return args[index]
	})

	Register(ns+"Switch", "string", "v...", "Gibt den Wert der ersten wahren Bedingung zurück", func(args []Value) Value {
		// Switch braucht immer Paare (Bedingung + Ergebnis).
		// Eine ungerade Anzahl an Argumenten oder weniger als 2 ist in VB ein Fehler/Undefined.
		if len(args) < 2 || len(args)%2 != 0 {
			return Value{Kind: KindUndefined}
		}

		// Wir laufen in 2er-Schritten durch das Array
		for i := 0; i < len(args); i += 2 {
			condition := args[i]
			value := args[i+1]

			// Hier nutzt du deine interne Logik, um zu prüfen: Ist das 'True'?
			if ToBool(condition) {
				return value
			}
		}

		// Wenn keine Bedingung zutrifft, gibt VB.NET Nothing (Undefined) zurück
		return Value{Kind: KindUndefined}
	})

	Register(ns+"RegExp", "string", "s, pattern", "Extrahiert Treffer basierend auf einem Regex-Muster (inkl. Groups).", func(args []Value) Value {
		if len(args) < 2 {
			return Value{Kind: KindArr, Arr: []Value{}}
		}

		text := ToString(args[0])
		pattern := ToString(args[1])

		re, err := regexp.Compile(pattern)
		if err != nil {
			return ErrorVal("RegExp: invalid pattern: " + err.Error())
		}

		// FindStringSubmatch ist oft nützlicher als FindAllString,
		// weil es uns erlaubt, Teile in Klammern () direkt zu isolieren.
		matches := re.FindStringSubmatch(text)

		var result []Value
		for _, m := range matches {
			result = append(result, StrVal(m))
		}

		// result[0] ist der ganze Treffer, result[1] die erste Klammer
		return Value{Kind: KindArr, Arr: result}
	})

	Register(ns+"Extract", "string", "text, pattern", "Extrahiert die erste RegEx-Gruppe", func(args []Value) Value {
		if len(args) < 2 {
			return StrVal("")
		}
		text := ToString(args[0])
		pattern := ToString(args[1])

		re, err := regexp.Compile(pattern)
		if err != nil {
			return StrVal("")
		}

		matches := re.FindStringSubmatch(text)
		// matches[0] ist der gesamte Treffer, matches[1] die erste Gruppe ()
		if len(matches) > 1 {
			return StrVal(matches[1])
		}
		return StrVal("")
	})

	Register(ns+"ExtractMax", "string", "text, pattern", "Findet alle Treffer und gibt die höchste Version zurück", func(args []Value) Value {
		if len(args) < 2 {
			return StrVal("")
		}
		text := ToString(args[0])
		pattern := ToString(args[1])

		re, err := regexp.Compile(pattern)
		if err != nil {
			return StrVal("")
		}

		matches := re.FindAllStringSubmatch(text, -1)
		if len(matches) == 0 {
			return StrVal("")
		}

		// Wir sammeln alle gefundenen Versionen
		var versions []string
		for _, m := range matches {
			if len(m) > 1 {
				versions = append(versions, m[1])
			}
		}

		// Ein einfacher "SemVer"-Sort (Alpha-Numerisch reicht hier meist)
		sort.Slice(versions, func(i, j int) bool {
			// Ein sehr simpler "Version-Sort" Hack:
			// Wir vergleichen die Länge und dann den Inhalt.
			if len(versions[i]) != len(versions[j]) {
				return len(versions[i]) < len(versions[j])
			}
			return versions[i] < versions[j]
		})

		return StrVal(versions[len(versions)-1])
	})

	Register(ns+"WordCount", "string", "s", "Zählt die Anzahl der Wörter in einem Text.", func(args []Value) Value {
		s, err := getStrArg(args, 0, "WordCount")
		if err.Kind == KindError {
			return err
		}
		// strings.Fields kümmert sich um doppelte Leerzeichen etc.
		words := strings.Fields(s)
		return NumVal(float64(len(words)))
	})

	Register(ns+"CharCount", "string", "s", "Zählt Buchstaben und Ziffern im Text.", func(args []Value) Value {
		s, err := getStrArg(args, 0, "CharCount")
		if err.Kind == KindError {
			return err
		}

		count := 0
		for _, r := range s {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				count++
			}
		}
		return NumVal(float64(count))
	})

	Register(ns+"StartsWith", "string", "s, prefix", "Prüft auf Start-Präfix.", func(args []Value) Value {
		if len(args) < 2 {
			return Value{Kind: KindBool, Bool: false}
		}
		s := ToString(args[0])
		prefix := ToString(args[1])
		return Value{Kind: KindBool, Bool: strings.HasPrefix(s, prefix)}
	})

	Register(ns+"EndsWith", "string", "s, suffix", "Prüft auf End-Suffix.", func(args []Value) Value {
		if len(args) < 2 {
			return BoolVal(false)
		}
		// Der volle Text (z.B. der Pfad)
		text := ToString(args[0])
		// Das Ende, das wir suchen (z.B. "stdlib_convert.go")
		suffix := ToString(args[1])

		return BoolVal(strings.HasSuffix(text, suffix))
	})

	Register(ns+"Repeat", "string", "s, n", "Wiederholt den Text n-mal.", func(args []Value) Value {
		return StrVal(strings.Repeat(ToString(MustValue(args, 0)), MustInt(MustValue(args, 1), 0)))
	})

	Register(ns+"Reverse", "string", "s", "Kehrt die Reihenfolge der Zeichen um.", func(args []Value) Value {
		runes := []rune(ToString(MustValue(args, 0)))
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		return StrVal(string(runes))
	})

	Register(ns+"PadLeft", "string", "s, n, [c]", "Füllt den String links auf eine Mindestlänge auf.", func(args []Value) Value {
		s := ToString(MustValue(args, 0))
		n := MustInt(MustValue(args, 1), len(s))

		pad := " "
		if len(args) >= 3 {
			pad = ToString(MustValue(args, 2))
		}

		if len(s) >= n {
			return StrVal(s)
		}

		var b strings.Builder
		padRunes := []rune(pad)
		target := n - len([]rune(s))

		for i := 0; i < target; i++ {
			b.WriteString(string(padRunes[i%len(padRunes)]))
		}

		b.WriteString(s)
		return StrVal(b.String())
	})

	Register(ns+"PadRight", "string", "s, n, [c]", "Füllt den String rechts auf eine Mindestlänge auf.", func(args []Value) Value {
		s := ToString(MustValue(args, 0))
		n := MustInt(MustValue(args, 1), len(s))

		pad := " "
		if len(args) >= 3 {
			pad = ToString(MustValue(args, 2))
		}

		if len(s) >= n {
			return StrVal(s)
		}

		var b strings.Builder
		b.WriteString(s)

		padRunes := []rune(pad)
		target := n - len([]rune(s))

		for i := 0; i < target; i++ {
			b.WriteString(string(padRunes[i%len(padRunes)]))
		}

		return StrVal(b.String())
	})

	Register(ns+"Val", "string", "n", "Extrahiert den numerischen Teil eines Strings von links beginnend", func(args []Value) Value {
		if len(args) == 0 {
			return NumVal(0)
		}

		raw := strings.TrimLeft(toStringSafe(args[0], "0"), " ")

		var b strings.Builder
		hasDecimal := false
		started := false

		for _, r := range raw {
			switch {
			case r >= '0' && r <= '9':
				b.WriteRune(r)
				started = true

			case r == '.' && !hasDecimal:
				b.WriteRune(r)
				hasDecimal = true
				started = true

			case (r == '-' || r == '+') && !started:
				b.WriteRune(r)
				started = true

			default:
				if started {
					goto DONE
				}
			}
		}

	DONE:
		s := b.String()

		if s == "" || s == "-" || s == "+" || s == "." {
			return NumVal(0)
		}

		val, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return NumVal(0)
		}

		return NumVal(val)
	})

	Register(ns+"Like", "string", "text, pattern",
		"Einfacher Mustervergleich: # (Ziffer), ? (ein Zeichen), * (viele Zeichen).",
		func(args []Value) Value {
			if len(args) < 2 {
				return BoolVal(false)
			}
			text := ToString(args[0])
			pattern := vbLikeToRegex(ToString(args[1]))

			matched, _ := regexp.MatchString(pattern, text)
			return BoolVal(matched)
		})

	Register(ns+"TrimLines", "string", "content",
		"Entfernt Leerzeichen am Ende jeder einzelnen Zeile.",
		func(args []Value) Value {
			lines := strings.Split(ToString(args[0]), "\n")
			for i, line := range lines {
				lines[i] = strings.TrimRight(line, " \t\r")
			}
			return StrVal(strings.Join(lines, "\n"))
		})

	Register(ns+"StrConv", "string", "input",
		"Konvertiert einen String sicher in UTF-8, ideal für plattformübergreifende Daten.",
		func(args []Value) Value {
			if len(args) < 1 {
				return ErrorVal("Kein Inhalt")
			}
			// Go-Strings sind nativ UTF-8, aber wir können hier
			// Validierung oder Konvertierung von ISO-8859-1 einbauen.
			return StrVal(ToUtf8(ToString(args[0])))
		})

	// compare: 0 = binary (case-sensitive, default), 1 = text (case-insensitive)
	Register(ns+"StrComp", "string", "s1, s2, [mode]", "Vergleicht zwei Strings (-1, 0, 1). Mode 1 = Ignoriere Case.", func(args []Value) Value {
		s1 := ToString(MustValue(args, 0))
		s2 := ToString(MustValue(args, 1))

		// Default: case-sensitive
		caseInsensitive := false
		if len(args) >= 3 && MustInt(MustValue(args, 2), 0) != 0 {
			caseInsensitive = true
		}

		if caseInsensitive {
			s1 = strings.ToLower(s1)
			s2 = strings.ToLower(s2)
		}

		switch {
		case s1 < s2:
			return NumVal(-1)
		case s1 > s2:
			return NumVal(1)
		default:
			return NumVal(0)
		}
	})

	// CompareText(string1, string2) - immer case-insensitive
	Register(ns+"CompareText", "string", "string1, string2", "Vergleicht zwei Texte ohne Berücksichtigung der Groß-/Kleinschreibung.",
		func(args []Value) Value {
			if len(args) < 2 {
				return ErrorVal("usage: string.CompareText(s1, s2)")
			}

			s1 := strings.ToLower(ToString(args[0]))
			s2 := strings.ToLower(ToString(args[1]))

			switch {
			case s1 < s2:
				return NumVal(-1)
			case s1 > s2:
				return NumVal(1)
			default:
				return NumVal(0)
			}
		},
	)

	Register(ns+"TxtToSqlInsert", "string", "data, table, dialect...", "Generiert SQL-Inserts aus einem Array (mit Transaktionen).", func(args []Value) Value {
		if len(args) < 3 {
			return ErrorVal("TxtToSqlInsert(data, table, dialect, [extras], [columns], [batchSize])")
		}

		dataVal := args[0]
		table := ToString(args[1])
		dialect := strings.ToLower(ToString(args[2]))

		// Hilfsfunktion zum Escapen (lokal definiert für Konsistenz)
		escape := func(v Value) string {
			if v.Kind == KindNum {
				return strconv.FormatFloat(v.Num, 'f', -1, 64)
			}
			s := ToString(v)
			return "'" + strings.ReplaceAll(s, "'", "''") + "'"
		}

		// Extras vorbereiten (direkt escapen)
		var extrasEscaped []string
		if len(args) >= 4 && args[3].Kind == KindArr {
			for _, v := range args[3].Arr {
				extrasEscaped = append(extrasEscaped, escape(v))
			}
		}

		// Spaltennamen
		colPart := ""
		if len(args) >= 5 && args[4].Kind == KindArr {
			var columns []string
			for _, c := range args[4].Arr {
				columns = append(columns, ToString(c))
			}
			colPart = " (" + strings.Join(columns, ", ") + ")"
		}

		batchSize := 0
		if len(args) >= 6 {
			batchSize = MustInt(args[5], 0)
		}

		if dataVal.Kind != KindArr || len(dataVal.Arr) == 0 {
			return StrVal("")
		}

		var sb strings.Builder

		// 1. Transaction Start
		switch dialect {
		case "sqlite", "postgres":
			sb.WriteString("BEGIN;\n")
		case "mssql":
			sb.WriteString("BEGIN TRANSACTION\n")
		}

		// 2. Data Loop
		for i, row := range dataVal.Arr {
			// Batch Header (INSERT INTO...)
			if i == 0 || (batchSize > 0 && i%batchSize == 0) {
				sb.WriteString("INSERT INTO " + table + colPart + " VALUES\n")
			}

			// Zeile bauen: Erst den Hauptwert (row), dann die Extras
			var rowParts []string
			rowParts = append(rowParts, escape(row))
			rowParts = append(rowParts, extrasEscaped...)

			sb.WriteString("(" + strings.Join(rowParts, ", ") + ")")

			// Trenner oder Abschluss
			isLastRow := i == len(dataVal.Arr)-1
			isEndOfBatch := batchSize > 0 && (i+1)%batchSize == 0

			if isLastRow || isEndOfBatch {
				sb.WriteString(";\n")
				if dialect == "mssql" {
					sb.WriteString("GO\n")
				}
			} else {
				sb.WriteString(",\n")
			}
		}

		// 3. Transaction Ende
		switch dialect {
		case "sqlite", "postgres":
			sb.WriteString("COMMIT;\n")
		case "mssql":
			sb.WriteString("COMMIT\nGO\n")
		}

		return StrVal(sb.String())
	})

	// string.Between(source, start, end)
	Register(ns+"Between", "string", "s, start, end", "Extrahiert den Text zwischen zwei Markern.", func(args []Value) Value {
		if len(args) < 3 {
			return ErrorVal("string.Between(source, start, end) benötigt 3 Argumente")
		}

		s := ToString(args[0])
		start := ToString(args[1])
		end := ToString(args[2])

		startIndex := strings.Index(s, start)
		if startIndex == -1 {
			return StrVal("")
		}
		startIndex += len(start)

		rest := s[startIndex:]
		endIndex := strings.Index(rest, end)
		if endIndex == -1 {
			return StrVal("")
		}

		return StrVal(rest[:endIndex])
	})

	// encoding.Base32Encode(s) -> "JBSWY3DP"
	Register(ns+"Base32Encode", "string", "val", "Wandelt Text in Base32 um (Standard-Alphabet, mit Padding).", func(args []Value) Value {
		if len(args) < 1 {
			return StrVal("")
		}
		data := []byte(ToString(args[0]))
		return StrVal(base32.StdEncoding.EncodeToString(data))
	})

	// encoding.Base32Decode("JBSWY3DP") -> "Hello"
	Register(ns+"Base32Decode", "string", "base32Str", "Wandelt einen Base32-String zurück in Text.", func(args []Value) Value {
		if len(args) < 1 {
			return StrVal("")
		}
		s := ToString(args[0])

		data, err := base32.StdEncoding.DecodeString(s)
		if err != nil {
			return ErrorVal("encoding.Base32Decode: " + err.Error())
		}
		return StrVal(string(data))
	})

	// --- In InitStringFunctions() ---

	Register(ns+"Space", "string", "n", "Gibt n Leerzeichen zurück.", func(args []Value) Value {
		if len(args) < 1 {
			return StrVal("")
		}
		n := int(toNumVal(args[0]))
		if n < 0 {
			n = 0
		}
		return StrVal(strings.Repeat(" ", n))
	})

	Register(ns+"StrDup", "string", "n, char", "Gibt ein Zeichen n-mal wiederholt zurück.", func(args []Value) Value {
		if len(args) < 2 {
			return StrVal("")
		}
		n := int(toNumVal(args[0]))
		char := ToString(args[1])
		if n < 0 {
			n = 0
		}
		if char == "" {
			return StrVal("")
		}
		// Nur das erste Zeichen nehmen – identisch zu VB.NET
		firstChar := string([]rune(char)[0])
		return StrVal(strings.Repeat(firstChar, n))
	})
}

func vbLikeToRegex(pattern string) string {
	// Konvertiert das einfache VB-Muster in eine interne Regex-Struktur
	p := regexp.QuoteMeta(pattern)
	p = strings.ReplaceAll(p, "\\#", "[0-9]")
	p = strings.ReplaceAll(p, "\\?", ".")
	p = strings.ReplaceAll(p, "\\*", ".*")
	return "^" + p + "$"
}

func ToUtf8(val interface{}) string {
	s := fmt.Sprint(val)
	if utf8.ValidString(s) {
		return s
	}

	// Wir bauen den String neu auf und ersetzen ungültige Sequenzen
	runes := make([]rune, 0, len(s))
	for i, r := range s {
		if r == utf8.RuneError {
			_, size := utf8.DecodeRuneInString(s[i:])
			if size == 1 {
				// HIER war der Fehler: Wir nutzen das offizielle Replacement-Character
				runes = append(runes, '\uFFFD')
				continue
			}
		}
		runes = append(runes, r)
	}
	return string(runes)
}
