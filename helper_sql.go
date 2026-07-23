// ----------------------------------------
// helper_sql.go | mssql, postgres, sqlite
// ----------------------------------------

package main

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"
)

// readSQLFile prüft die Endung und liest den Inhalt der SQL-Datei ein
func readSQLFile(path string) (string, error) {
	// 1. Endung prüfen
	if !strings.HasSuffix(strings.ToLower(path), ".sql") {
		return "", fmt.Errorf("nur .sql Dateien sind erlaubt")
	}

	// 2. Existenz prüfen
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", fmt.Errorf("datei nicht gefunden: %s", path)
	}

	// 3. Lesen
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("fehler beim Lesen: %v", err)
	}

	return string(content), nil
}

// OptimizeSmart bereitet SQL-Strings für die Batch-Ausführung vor.
// Er entfernt Kommentare, normalisiert Trenner und schützt Strings.
func OptimizeSQL(raw string, dialect string) string {
	dialect = strings.ToLower(dialect)

	// 1. Kommentare entfernen (Mehrzeilig /* */ und Einzeilig --)
	reMulti := regexp.MustCompile(`(?s)/\*.*?\*/`)
	clean := reMulti.ReplaceAllString(raw, "")

	reSingle := regexp.MustCompile(`(?m)--.*$`)
	clean = reSingle.ReplaceAllString(clean, "")

	// 2. In Zeilen zerlegen für Keyword-Analyse
	lines := strings.Split(clean, "\n")
	var batches []string
	var currentBatch []string

	// Start-Keywords, die einen neuen logischen Block einleiten
	keywords := []string{
		"SELECT", "INSERT", "UPDATE", "DELETE",
		"CREATE", "DROP", "ALTER", "IF", "BEGIN",
		"WITH", "EXEC", "TRUNCATE", "MERGE", "GRANT",
		"COMMIT", "ROLLBACK", "TRANSACTION", // NEU hinzugefügt
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		upperLine := strings.ToUpper(trimmed)
		isNewCommand := false

		// Prüfen, ob die Zeile mit einem Kommando startet
		for _, k := range keywords {
			if strings.HasPrefix(upperLine, k) {
				isNewCommand = true
				break
			}
		}

		// Wenn ein neuer Befehl startet, alten Batch verarbeiten
		if isNewCommand && len(currentBatch) > 0 {
			batches = append(batches, finalizeBatch(currentBatch))
			currentBatch = []string{}
		}

		currentBatch = append(currentBatch, trimmed)
	}

	// Letzten Block hinzufügen
	if len(currentBatch) > 0 {
		batches = append(batches, finalizeBatch(currentBatch))
	}

	// 3. Zusammenbau mit Dialekt-Trennern
	if dialect == "mssql" || dialect == "sqlserver" {
		// MSSQL braucht das GO als harten Batch-Separator für den Exec-Handler
		return strings.Join(batches, "\nGO\n") + "\nGO"
	}

	// Standard (Postgres/SQLite) nutzt nur Semikolon
	return strings.Join(batches, "\n")
}

// finalizeBatch säubert ein einzelnes SQL-Statement
func finalizeBatch(lines []string) string {
	full := strings.Join(lines, " ")

	// REGEX: Mehrfache Semikolons am Ende (inkl. Whitespaces dazwischen)
	// auf genau EIN Semikolon reduzieren.
	reMultiSemi := regexp.MustCompile(`\s*;[\s;]*$`)

	// Falls am Ende Semikolons sind -> ersetzen durch eines
	if reMultiSemi.MatchString(full) {
		full = reMultiSemi.ReplaceAllString(full, ";")
	} else {
		// Falls gar kein Semikolon da ist (und es kein Spezialfall ist) -> eins dran
		trimmed := strings.TrimSpace(full)
		if trimmed != "" && !strings.HasSuffix(trimmed, ";") {
			full = trimmed + ";"
		}
	}

	return full
}

func getConn(alias string) (*sql.DB, string, error) {
	alias = strings.ToLower(alias)
	db, ok := connections[alias]
	if !ok {
		return nil, "", fmt.Errorf("verbindung '%s' nicht gefunden", alias)
	}
	return db, drivers[alias], nil
}

func getTableExistsSQL(driver, tableName string) (string, []interface{}) {
	switch driver {
	case "mssql", "sqlserver":
		return "SELECT COUNT(*) FROM sys.tables WHERE name = @p1", []interface{}{tableName}
	case "postgres", "pg":
		return "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = $1", []interface{}{tableName}
	case "sqlite", "sqlite3":
		return "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name = ?", []interface{}{tableName}
	default:
		return "", nil
	}
}

func getListTablesSQL(driver string) string {
	switch driver {
	case "mssql", "sqlserver":
		return "SELECT name FROM sys.tables ORDER BY name"
	case "postgres", "pg":
		return "SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' ORDER BY table_name"
	case "sqlite", "sqlite3":
		return "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name"
	default:
		return ""
	}
}

func isSafeDBName(name string) bool {
	// Dein Regex: /^[a-zA-Z0-9_]+$/
	return validNameMSSQL.MatchString(name)
}

func getListColumnsSQL(driver, table string) (string, []interface{}) {
	switch driver {
	case "mssql", "sqlserver":
		return "SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = @p1 ORDER BY ORDINAL_POSITION", []interface{}{table}
	case "postgres", "pg":
		return "SELECT column_name FROM information_schema.columns WHERE table_name = $1 ORDER BY ordinal_position", []interface{}{table}
	case "sqlite", "sqlite3":
		return fmt.Sprintf("PRAGMA table_info([%s])", table), nil // SQLite nutzt PRAGMA statt SELECT
	}
	return "", nil
}

func getDatabaseNameSQL(driver string) string {
	switch driver {
	case "mssql", "sqlserver":
		return "SELECT DB_NAME()"
	case "postgres", "pg":
		return "SELECT current_database()"
	case "sqlite", "sqlite3":
		return "SELECT name FROM pragma_database_list WHERE seq = 0"
	}
	return ""
}

func executeSQL(dbOrTx interface{}, batch string) (Value, error) {
	batch = strings.TrimSpace(batch)
	if batch == "" {
		return Value{Kind: KindNull}, nil
	}

	// 1. Robuste Typerkennung für SELECT/Queries
	// Wir ignorieren führende Kommentare bei der Typerkennung
	trimmedUpper := strings.ToUpper(batch)
	re := regexp.MustCompile(`(?s)^(--.*?\n|/\*.*?\*/\s*)*`)
	cleanStart := re.ReplaceAllString(trimmedUpper, "")

	// Ein Batch gilt als Query, wenn er mit Daten-liefernden Befehlen beginnt
	isQuery := strings.HasPrefix(cleanStart, "SELECT") ||
		strings.HasPrefix(cleanStart, "WITH") ||
		strings.HasPrefix(cleanStart, "EXEC") ||
		strings.HasPrefix(cleanStart, "SHOW")

	var err error

	// 2. Ausführung je nach Typ
	if isQuery {
		var rows *sql.Rows
		switch conn := dbOrTx.(type) {
		case *sql.Tx:
			rows, err = conn.Query(batch)
		case *sql.DB:
			rows, err = conn.Query(batch)
		default:
			return Value{}, fmt.Errorf("ungültiger Datenbank-Kontext (Query)")
		}

		if err != nil {
			return Value{}, err
		}
		// Übergibt an deine sqlToArr2D (inkl. Header-Logik)
		return sqlToArr2D(rows), nil

	} else {
		var res sql.Result
		switch conn := dbOrTx.(type) {
		case *sql.Tx:
			res, err = conn.Exec(batch)
		case *sql.DB:
			res, err = conn.Exec(batch)
		default:
			return Value{}, fmt.Errorf("ungültiger Datenbank-Kontext (Exec)")
		}

		if err != nil {
			return Value{}, err
		}

		ra, _ := res.RowsAffected()
		// Rückgabe einer Erfolgsmeldung für das Result-Array
		return StrVal(fmt.Sprintf("OK: %d Zeilen betroffen", ra)), nil
	}
}

func sqlToArr2D(rows *sql.Rows) Value {
	defer rows.Close()
	cols, _ := rows.Columns()
	allRows := [][]Value{}

	// --- NEU: Header-Zeile als allererstes hinzufügen ---
	headerRow := make([]Value, len(cols))
	for i, colName := range cols {
		headerRow[i] = StrVal(colName) // Spaltennamen sind immer Strings
	}
	allRows = append(allRows, headerRow)
	// ----------------------------------------------------

	vals := make([]interface{}, len(cols))
	ptrs := make([]interface{}, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}

	cleanFunc := func(s string) string {
		s = strings.TrimSpace(s)
		return strings.Map(func(r rune) rune {
			// Behalte druckbare Zeichen (>=32) oder Tab (9)
			if r >= 32 || r == 9 || r == 10 || r == 13 {
				return r
			}
			return -1
		}, s)
	}

	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			continue
		}

		rowValues := make([]Value, len(cols))
		for i, v := range vals {
			if v == nil {
				rowValues[i] = Value{Kind: KindNull}
				continue
			}

			switch val := v.(type) {
			case string:
				rowValues[i] = StrVal(cleanFunc(val))
			case []byte:
				if isBinaryData(val) {
					rowValues[i] = StrVal(base64.StdEncoding.EncodeToString(val))
				} else {
					rowValues[i] = StrVal(cleanFunc(string(val)))
				}
			default:
				rowValues[i] = convertToValue(v)
			}
		}
		allRows = append(allRows, rowValues)
	}

	return Value{Kind: KindArr2D, Arr2D: allRows}
}

func isBinaryData(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	// Wir prüfen die ersten 512 Bytes (reicht für die Erkennung)
	checkLen := len(data)
	if checkLen > 512 {
		checkLen = 512
	}

	nullCount := 0
	for i := 0; i < checkLen; i++ {
		b := data[i]

		// 1. Steuerzeichen (außer Tab, LineFeed, CarriageReturn)
		// 0-8, 11-12, 14-31 sind verdächtig
		if b < 32 && b != 9 && b != 10 && b != 13 {
			// Wenn wir ein Null-Byte finden, ist es fast sicher Binär (z.B. EXE/JPG)
			if b == 0 {
				nullCount++
			}
			// Bei mehr als 2 "echten" Steuerzeichen oder einem Null-Byte -> Binär
			if nullCount > 0 || i > 10 {
				return true
			}
		}
	}

	// 2. UTF-8 Validierung: Wenn es kein gültiges UTF-8 ist,
	// muss es als Base64 kodiert werden, damit der JSON/VB-Transport nicht bricht.
	if !utf8.Valid(data) {
		return true
	}

	return false
}

// Hilfs-Wrapper, der den Pointer-Check sauber kapselt
func processTargetPath(p string) (string, *Value) {
	abs, errVal := absPathVal(p) // Nutzt deine Funktion, die (string, *Value) liefert
	if errVal != nil {
		return "", errVal // errVal ist hier bereits ein *Value, also ist nil-Check erlaubt
	}
	return abs, nil
}
