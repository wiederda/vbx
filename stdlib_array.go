// ------------------------
// stdlib_array.go
// ------------------------

package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/xuri/excelize/v2"
)

// InitArrayFunctions registriert array-Funktionen
func InitArrayFunctions() {
	if builtins == nil {
		builtins = make(map[string]BuiltinInfo)
	}

	ns := "array."

	// 1D- und 2D-Array erstellen
	Register(ns+"Create", "array", "[args...]", "Erstellt 1D- oder 2D-Arrays.", func(args []Value) Value {
		count := len(args)

		// --- 1. FALL: Dynamische 2D-Matrix nach Maß ---
		// Aufruf: array.Create(3, 3) -> Erstellt eine leere 3x3 Matrix (mit 0 gefüllt)
		if count == 2 && args[0].Kind == KindNum && args[1].Kind == KindNum {
			rows := int(args[0].Num)
			cols := int(args[1].Num)

			matrix := make([][]Value, rows)
			for i := range matrix {
				row := make([]Value, cols)
				for j := range row {
					row[j] = NumVal(0)
				}
				matrix[i] = row
			}
			return Value{Kind: KindArr2D, Arr2D: matrix}
		}

		// --- 2. FALL: Statische 2D-Matrix aus Daten ---
		// Aufruf: array.Create({1,2}, {3,4}) -> Konvertiert Literale in echtes KindArr2D
		if count > 0 && args[0].Kind == KindArr {
			res2D := make([][]Value, count)
			for i, v := range args {
				if v.Kind == KindArr {
					res2D[i] = v.Arr
				} else {
					res2D[i] = []Value{v} // Fallback für gemischte Daten
				}
			}
			return Value{Kind: KindArr2D, Arr2D: res2D}
		}

		// --- 3. FALL: 1D-Array mit fester Größe ---
		// Aufruf: array.Create(10) -> [0, 0, 0, 0, 0, 0, 0, 0, 0, 0]
		if count == 1 && args[0].Kind == KindNum {
			n := int(args[0].Num)
			arr := make([]Value, n)
			for i := range arr {
				arr[i] = NumVal(0)
			}
			return Value{Kind: KindArr, Arr: arr}
		}

		// --- 4. FALL: 1D-Array aus Elementen ---
		// Aufruf: array.Create(1, 2, "Test") -> [1, 2, "Test"]
		arr := make([]Value, count)
		copy(arr, args)
		return Value{Kind: KindArr, Arr: arr}
	})

	Register(ns+"Append", "array", "arr, value", "Alias für array.Add", func(args []Value) Value {
		return builtins["array.Add"].Fn(args)
	})

	// In deiner InitArrayFunctions() oder im entsprechenden Block:
	Register(ns+"Length", "array", "array, [dim]", "Alias für Count.", func(args []Value) Value {
		return getArrayCount(args)
	})

	// ------------------------
	// Count / LBound / UBound für Arrays
	// ------------------------
	Register(ns+"Count", "array", "array, [dim]", "Gibt die Anzahl der Elemente oder Dimensionen zurück.", func(args []Value) Value {
		return getArrayCount(args)
	})

	Register(ns+"LBound", "array", "arr, [dim]", "Gibt die Untergrenze der angegebenen Dimension zurück (Standard: 1).", func(args []Value) Value {
		if len(args) < 1 {
			return NumVal(-1)
		}

		dim := 1
		if len(args) >= 2 {
			dim = int(args[1].Num)
		}

		switch args[0].Kind {
		case KindArr:
			if dim != 1 || len(args[0].Arr) == 0 {
				return NumVal(-1)
			}
			return NumVal(0)
		case KindArr2D:
			if len(args[0].Arr2D) == 0 {
				return NumVal(-1)
			}
			// Wir prüfen, ob die Dimension 1 (Zeilen) oder 2 (Spalten) existiert
			if dim == 1 || (dim == 2 && len(args[0].Arr2D[0]) > 0) {
				return NumVal(0)
			}
			return NumVal(-1)
		default:
			return NumVal(-1)
		}
	})

	Register(ns+"UBound", "array", "arr, [dim]", "Gibt die Obergrenze der angegebenen Dimension zurück (Standard: 1).", func(args []Value) Value {
		if len(args) < 1 {
			return NumVal(-1)
		}

		dim := 1
		if len(args) >= 2 {
			dim = int(args[1].Num)
		}

		switch args[0].Kind {
		case KindArr:
			if dim != 1 || len(args[0].Arr) == 0 {
				return NumVal(-1)
			}
			return NumVal(float64(len(args[0].Arr) - 1))

		case KindArr2D:
			rows := len(args[0].Arr2D)
			if rows == 0 {
				return NumVal(-1)
			}

			// Hier der "Tagged Switch" auf dim
			switch dim {
			case 1:
				return NumVal(float64(rows - 1))
			case 2:
				// Wir nehmen die Länge der ersten Zeile als Spaltenanzahl
				return NumVal(float64(len(args[0].Arr2D[0]) - 1))
			default:
				return NumVal(-1)
			}

		default:
			return NumVal(-1)
		}
	})

	Register(ns+"First", "array", "array", "Gibt das erste Element eines Arrays zurück", func(args []Value) Value { return getBoundaryElement(args, true) })
	Register(ns+"Last", "array", "array", "Gibt das letzte Element eines Arrays zurück", func(args []Value) Value { return getBoundaryElement(args, false) })

	Register(ns+"Prepend", "array", "arr, value", "Fügt Element am Anfang ein", func(args []Value) Value {
		return builtins["array.Insert"].Fn([]Value{
			args[0],
			NumVal(0),
			args[1],
		})
	})

	Register(ns+"Split", "string", "str, sep", "Teilt String in Array", func(args []Value) Value {
		if len(args) < 2 {
			return ErrorVal("usage: array.Split(str, sep)")
		}

		str := ToString(args[0])
		sep := ToString(args[1])

		if sep == "" {
			return ErrorVal("separator cannot be empty")
		}

		parts := strings.Split(str, sep)

		var res []Value
		for _, p := range parts {
			res = append(res, StrVal(p))
		}

		return ArrVal(res)
	})

	Register(ns+"CleanArray", "array", "array", "Bereinigt Strings in Arrays (Trim + Entfernen von Steuerzeichen < 32, außer Tab)", func(args []Value) Value {
		if len(args) < 1 {
			return Value{}
		}

		clean := func(s string) string {
			s = strings.TrimSpace(s)
			return strings.Map(func(r rune) rune {
				if r >= 32 || r == 9 {
					return r
				}
				return -1
			}, s)
		}

		switch args[0].Kind {

		case KindArr:
			arr := make([]Value, len(args[0].Arr))
			for i, v := range args[0].Arr {
				if v.Kind == KindStr {
					arr[i] = StrVal(clean(v.Str))
				} else {
					arr[i] = v
				}
			}
			return Value{Kind: KindArr, Arr: arr}

		case KindArr2D:
			arr2D := make([][]Value, len(args[0].Arr2D))

			for i, row := range args[0].Arr2D {
				newRow := make([]Value, len(row))
				for j, v := range row {
					if v.Kind == KindStr {
						newRow[j] = StrVal(clean(v.Str))
					} else {
						newRow[j] = v
					}
				}
				arr2D[i] = newRow
			}

			return Value{Kind: KindArr2D, Arr2D: arr2D}
		}

		return Value{}
	})

	// Insert: Element an Index einfügen, Array ggf. erweitern
	Register(ns+"Insert", "array", "arr, index, value", "Fügt ein Element an der angegebenen Position ein (In-Place).", func(args []Value) Value {
		if len(args) < 3 || args[0].Kind != KindArr {
			return args[0]
		}

		// Index auslesen
		idx := int(toNumVal(args[1]))
		val := args[2]
		arr := args[0].Arr

		// VB.NET Style: Wenn der Index < 0 ist, setzen wir ihn auf 0
		if idx < 0 {
			idx = 0
		}

		// Wenn der Index größer als die aktuelle Länge ist,
		// verhält es sich wie ein normales Add am Ende.
		if idx >= len(arr) {
			args[0].Arr = append(arr, val)
			return args[0]
		}

		// In-Place Insert Logik:
		// 1. Wir vergrößern das Slice um ein Element (append eines Dummy-Werts)
		arr = append(arr, Value{})
		// 2. Wir schieben alle Elemente ab 'idx' um eins nach rechts
		copy(arr[idx+1:], arr[idx:])
		// 3. Wir setzen den neuen Wert an die freigewordene Stelle
		arr[idx] = val

		// Das aktualisierte Slice zurück ins Value-Objekt schreiben
		args[0].Arr = arr

		return args[0]
	})

	// In InitArrayFunctions()
	Register(ns+"Add", "array", "arr, val", "Fügt ein Element am Ende des Arrays ein (In-Place).", func(args []Value) Value {
		if len(args) < 2 || args[0].Kind != KindArr {
			return ErrorVal("array.Add erwartet ein 1D-Array und einen Wert")
		}

		// Direktes Anhängen an das Slice im Value-Objekt
		// Da args[0] die Referenz auf das Array-Value ist,
		// ist die Änderung sofort für die Variable im VB-Code gültig.
		args[0].Arr = append(args[0].Arr, args[1])

		return args[0]
	})

	Register(ns+"Join", "array", "arr, sep", "Verbindet Array-Elemente zu String", func(args []Value) Value {
		if len(args) < 2 {
			return ErrorVal("usage: array.Join(arr, sep)")
		}

		arr := args[0]
		sep := ToString(args[1])

		// Erwartung: ArrayVal intern
		if arr.Kind != KindArr {
			return ErrorVal("first argument must be array")
		}

		var parts []string
		for _, v := range arr.Arr {
			parts = append(parts, ToString(v))
		}

		return StrVal(strings.Join(parts, sep))
	})

	Register(ns+"Contains", "array", "arr, value", "Prüft, ob ein Wert im Array enthalten ist (Vergleich erfolgt als Text).", func(args []Value) Value {
		// 1. Check: Haben wir genug Argumente und ist das erste ein Array?
		if len(args) < 2 || args[0].Kind != KindArr {
			return BoolVal(false)
		}

		// 2. Suche: Wir wandeln das Suchkriterium einmal in einen String um
		searchStr := ToString(args[1])

		for _, v := range args[0].Arr {
			// Wir vergleichen alles als String, das macht den Interpreter "gnädiger"
			if ToString(v) == searchStr {
				return BoolVal(true)
			}
		}
		return BoolVal(false)
	})

	// Index eines Wertes im Array
	Register(ns+"IndexOf", "array", "arr, value", "Gibt die Position des ersten Treffers zurück (0-basiert), oder -1 falls nicht gefunden.", func(args []Value) Value {
		if len(args) < 2 || args[0].Arr == nil {
			return Value{Kind: KindNum, Num: -1}
		}

		// Kleiner Tipp: Nutze hier auch ToString(args[1]),
		// um konsistent zu Contains zu bleiben!
		searchStr := ToString(args[1])

		for i, v := range args[0].Arr {
			if ToString(v) == searchStr {
				return Value{Kind: KindNum, Num: float64(i)}
			}
		}
		return Value{Kind: KindNum, Num: -1}
	})

	Register(ns+"ToCSV", "array", "path, data, [sep, exclude, append]", "Speichert ein Array oder 2D-Array als CSV-Datei. Mit exclude können Zeilen ausgeschlossen und mit append Daten angehängt werden.", func(args []Value) Value {
		if len(args) < 2 {
			return Value{
				Kind: KindStr,
				Str:  "error: filename and data required",
			}
		}

		filename := args[0].Str
		data := args[1]

		// ------------------------------------------------------------
		// Separator
		// ------------------------------------------------------------

		separator := ';'

		if len(args) >= 3 && args[2].Kind == KindStr && args[2].Str != "" {
			separator = rune(args[2].Str[0])
		}

		// ------------------------------------------------------------
		// Ausschlusswerte
		// ------------------------------------------------------------

		excludeSet := make(map[string]struct{})

		if len(args) >= 4 && args[3].Kind == KindArr {
			for _, ex := range args[3].Arr {
				excludeSet[ToString(ex)] = struct{}{}
			}
		}

		rowExcluded := func(row []Value) bool {
			if len(excludeSet) == 0 {
				return false
			}

			for _, cell := range row {
				if _, exists := excludeSet[ToString(cell)]; exists {
					return true
				}
			}

			return false
		}

		// ------------------------------------------------------------
		// Daten prüfen
		// ------------------------------------------------------------

		if data.Kind != KindArr && data.Kind != KindArr2D {
			return Value{
				Kind: KindStr,
				Str:  "error: data must be Array or Array2D",
			}
		}

		// ------------------------------------------------------------
		// Datei öffnen
		// ------------------------------------------------------------

		var f *os.File
		var err error

		if len(args) >= 5 && args[4].Kind == KindBool && args[4].Bool {
			// Append
			f, err = os.OpenFile(
				filename,
				os.O_CREATE|os.O_WRONLY|os.O_APPEND,
				0644,
			)
		} else {
			// Datei neu erstellen / überschreiben
			f, err = os.Create(filename)
		}

		if err != nil {
			return Value{
				Kind: KindStr,
				Str:  "error: " + err.Error(),
			}
		}

		defer f.Close()

		// ------------------------------------------------------------
		// CSV Writer
		// ------------------------------------------------------------

		writer := csv.NewWriter(f)
		writer.Comma = separator

		// ------------------------------------------------------------
		// Eine Zeile schreiben
		// ------------------------------------------------------------

		writeRow := func(row []Value) error {
			if rowExcluded(row) {
				return nil
			}

			record := make([]string, len(row))

			for i, cell := range row {
				val := valToInterface(cell)

				if val == nil {
					record[i] = ""
				} else {
					record[i] = fmt.Sprintf("%v", val)
				}
			}

			return writer.Write(record)
		}

		// ------------------------------------------------------------
		// Daten schreiben
		// ------------------------------------------------------------

		if data.Kind == KindArr2D {

			for _, row := range data.Arr2D {
				if err := writeRow(row); err != nil {
					return Value{
						Kind: KindStr,
						Str:  "error: " + err.Error(),
					}
				}
			}

		} else {

			for _, rowVal := range data.Arr {

				if rowVal.Kind == KindArr {

					if err := writeRow(rowVal.Arr); err != nil {
						return Value{
							Kind: KindStr,
							Str:  "error: " + err.Error(),
						}
					}

				} else {

					// Flaches Array:
					// jedes Element wird zu einer eigenen Zeile
					// mit genau einer Spalte.

					if err := writeRow([]Value{rowVal}); err != nil {
						return Value{
							Kind: KindStr,
							Str:  "error: " + err.Error(),
						}
					}
				}
			}
		}

		// ------------------------------------------------------------
		// Flush prüfen
		// ------------------------------------------------------------

		writer.Flush()

		if err := writer.Error(); err != nil {
			return Value{
				Kind: KindStr,
				Str:  "error: " + err.Error(),
			}
		}

		return Value{
			Kind: KindStr,
			Str:  "ok",
		}
	})

	Register(ns+"FromCSV", "array", "path, [sep, exclude]", "Lädt eine CSV-Datei in ein 2D-Array. Leere Zeilen werden ignoriert. (Default-Separator: ;)", func(args []Value) Value {
		if len(args) < 1 {
			return Value{
				Kind:  KindArr2D,
				Arr2D: [][]Value{},
			}
		}

		path := args[0].Str

		// ------------------------------------------------------------
		// Separator
		// ------------------------------------------------------------

		separator := ';'

		if len(args) >= 2 &&
			args[1].Kind == KindStr &&
			args[1].Str != "" {

			separator = rune(args[1].Str[0])
		}

		// ------------------------------------------------------------
		// Ausschlusswerte
		// ------------------------------------------------------------

		excludeSet := make(map[string]struct{})

		if len(args) >= 3 && args[2].Kind == KindArr {

			for _, ex := range args[2].Arr {
				excludeSet[ToString(ex)] = struct{}{}
			}
		}

		rowExcluded := func(row []string) bool {

			if len(excludeSet) == 0 {
				return false
			}

			for _, cell := range row {

				value := strings.TrimSpace(cell)

				if _, exists := excludeSet[value]; exists {
					return true
				}
			}

			return false
		}

		// ------------------------------------------------------------
		// Leere Zeile
		//
		// Eine Zeile gilt als leer, wenn alle Zellen nach
		// TrimSpace keinen Inhalt enthalten.
		// ------------------------------------------------------------

		rowEmpty := func(row []string) bool {

			for _, cell := range row {

				if strings.TrimSpace(cell) != "" {
					return false
				}
			}

			return true
		}

		// ------------------------------------------------------------
		// Datei komplett roh einlesen
		// ------------------------------------------------------------

		rawContent, err := os.ReadFile(path)
		if err != nil {
			return Value{
				Kind:  KindArr2D,
				Arr2D: [][]Value{},
			}
		}

		rawContent = ensureUTF8(rawContent)

		// ------------------------------------------------------------
		// Sanitizing
		// ------------------------------------------------------------

		cleanContent := make([]byte, 0, len(rawContent))

		for i := 0; i < len(rawContent); i++ {

			b := rawContent[i]

			// Erlaube:
			// Druckbare ASCII/UTF8
			// Tab, LF, CR
			//
			// Entferne:
			// Steuerzeichen unter 32
			if b >= 32 ||
				b == '\n' ||
				b == '\r' ||
				b == '\t' {

				cleanContent = append(cleanContent, b)
			}
		}

		// ------------------------------------------------------------
		// CSV lesen
		// ------------------------------------------------------------

		reader := csv.NewReader(bytes.NewReader(cleanContent))

		reader.Comma = separator
		reader.LazyQuotes = true
		reader.FieldsPerRecord = -1

		records, err := reader.ReadAll()

		if err != nil {

			fmt.Println(
				"CSV-Parser-Error nach Reinigung:",
				err,
			)

			return Value{
				Kind:  KindArr2D,
				Arr2D: [][]Value{},
			}
		}

		// ------------------------------------------------------------
		// In 2D-Value-Array umwandeln
		// ------------------------------------------------------------

		res2D := make([][]Value, 0, len(records))

		for _, row := range records {

			// Ausschluss prüfen
			if rowExcluded(row) {
				continue
			}

			// Leere Zeilen ignorieren
			if rowEmpty(row) {
				continue
			}

			resRow := make([]Value, len(row))

			for j, cell := range row {

				resRow[j] = Value{
					Kind: KindStr,
					Str:  strings.TrimSpace(cell),
				}
			}

			res2D = append(res2D, resRow)
		}

		return Value{
			Kind:  KindArr2D,
			Arr2D: res2D,
		}
	})

	Register(ns+"ToXLSX", "array", "path, data, [sheetName, exclude, append, headers]", "Speichert ein Array oder 2D-Array als XLSX-Datei. Vorhandene Blätter werden standardmäßig ersetzt. Mit append=True werden Daten angehängt. Mit headers können optionale Spaltenüberschriften angegeben werden.", func(args []Value) Value {
		if len(args) < 2 {
			return Value{
				Kind: KindStr,
				Str:  "error: filename and data required",
			}
		}

		filename := args[0].Str
		data := args[1]

		// ------------------------------------------------------------
		// Parameter
		// ------------------------------------------------------------

		sheetName := "Sheet1"

		if len(args) >= 3 &&
			args[2].Kind == KindStr &&
			args[2].Str != "" {
			sheetName = args[2].Str
		}

		// ------------------------------------------------------------
		// Ausschlusswerte
		// ------------------------------------------------------------

		excludeSet := make(map[string]struct{})

		if len(args) >= 4 && args[3].Kind == KindArr {
			for _, ex := range args[3].Arr {
				excludeSet[ToString(ex)] = struct{}{}
			}
		}

		// ------------------------------------------------------------
		// Append
		// ------------------------------------------------------------

		appendMode := false

		if len(args) >= 5 && args[4].Kind == KindBool {
			appendMode = args[4].Bool
		}

		// ------------------------------------------------------------
		// Überschriften
		// ------------------------------------------------------------

		var headers []Value

		if len(args) >= 6 && args[5].Kind == KindArr {
			headers = args[5].Arr
		}

		// ------------------------------------------------------------
		// Daten prüfen
		// ------------------------------------------------------------

		if data.Kind != KindArr && data.Kind != KindArr2D {
			return Value{
				Kind: KindStr,
				Str:  "error: data must be Array or Array2D",
			}
		}

		// ------------------------------------------------------------
		// XLSX öffnen oder neu erstellen
		// ------------------------------------------------------------

		var f *excelize.File
		var err error

		if _, statErr := os.Stat(filename); statErr == nil {
			f, err = excelize.OpenFile(filename)
			if err != nil {
				return Value{
					Kind: KindStr,
					Str:  "error: " + err.Error(),
				}
			}
		} else {
			f = excelize.NewFile()
		}

		defer f.Close()

		// ------------------------------------------------------------
		// Blatt suchen
		// ------------------------------------------------------------

		sheetIndex, err := f.GetSheetIndex(sheetName)
		if err != nil {
			return Value{
				Kind: KindStr,
				Str:  "error: " + err.Error(),
			}
		}

		// ------------------------------------------------------------
		// Blatt existiert
		// ------------------------------------------------------------

		if sheetIndex >= 0 {

			if !appendMode {

				// Vorhandenes Blatt löschen
				if err := f.DeleteSheet(sheetName); err != nil {
					return Value{
						Kind: KindStr,
						Str:  "error: " + err.Error(),
					}
				}

				// Blatt neu anlegen
				sheetIndex, err = f.NewSheet(sheetName)
				if err != nil {
					return Value{
						Kind: KindStr,
						Str:  "error: " + err.Error(),
					}
				}
			}

		} else {

			// --------------------------------------------------------
			// Blatt existiert noch nicht
			// --------------------------------------------------------

			sheetIndex, err = f.NewSheet(sheetName)
			if err != nil {
				return Value{
					Kind: KindStr,
					Str:  "error: " + err.Error(),
				}
			}
		}

		_ = sheetIndex

		// ------------------------------------------------------------
		// Vorhandene Zeilen ermitteln
		// ------------------------------------------------------------

		rows, err := f.GetRows(sheetName)
		if err != nil {
			return Value{
				Kind: KindStr,
				Str:  "error: " + err.Error(),
			}
		}

		// ------------------------------------------------------------
		// Startzeile
		// ------------------------------------------------------------

		rowIdx := 1

		if appendMode && len(rows) > 0 {
			rowIdx = len(rows) + 1
		}

		// ------------------------------------------------------------
		// Zeile schreiben
		// ------------------------------------------------------------

		writeRow := func(rowIdx int, row []Value) error {
			for colIdx, cell := range row {

				cellRef, err := excelize.CoordinatesToCellName(
					colIdx+1,
					rowIdx,
				)

				if err != nil {
					return err
				}

				if err := f.SetCellValue(
					sheetName,
					cellRef,
					valToInterface(cell),
				); err != nil {
					return err
				}
			}

			return nil
		}

		// ------------------------------------------------------------
		// Überschriften schreiben
		// ------------------------------------------------------------

		if len(headers) > 0 {

			// Bei Append nur schreiben, wenn das Blatt leer ist.
			if !appendMode || len(rows) == 0 {

				headerStyle, err := f.NewStyle(&excelize.Style{
					Font: &excelize.Font{
						Bold: true,
					},
				})

				if err != nil {
					return Value{
						Kind: KindStr,
						Str:  "error: " + err.Error(),
					}
				}

				for colIdx, cell := range headers {

					cellRef, err := excelize.CoordinatesToCellName(
						colIdx+1,
						rowIdx,
					)

					if err != nil {
						return Value{
							Kind: KindStr,
							Str:  "error: " + err.Error(),
						}
					}

					if err := f.SetCellValue(
						sheetName,
						cellRef,
						valToInterface(cell),
					); err != nil {
						return Value{
							Kind: KindStr,
							Str:  "error: " + err.Error(),
						}
					}

					if err := f.SetCellStyle(
						sheetName,
						cellRef,
						cellRef,
						headerStyle,
					); err != nil {
						return Value{
							Kind: KindStr,
							Str:  "error: " + err.Error(),
						}
					}
				}

				rowIdx++
			}
		}

		// ------------------------------------------------------------
		// Zeile ausschließen
		// ------------------------------------------------------------

		rowExcluded := func(row []Value) bool {
			if len(excludeSet) == 0 {
				return false
			}

			for _, cell := range row {
				if _, exists := excludeSet[ToString(cell)]; exists {
					return true
				}
			}

			return false
		}

		// ------------------------------------------------------------
		// Daten schreiben
		// ------------------------------------------------------------

		if data.Kind == KindArr2D {

			for _, row := range data.Arr2D {

				if rowExcluded(row) {
					continue
				}

				if err := writeRow(rowIdx, row); err != nil {
					return Value{
						Kind: KindStr,
						Str:  "error: " + err.Error(),
					}
				}

				rowIdx++
			}

		} else {

			for _, rowVal := range data.Arr {

				var row []Value

				if rowVal.Kind == KindArr {
					row = rowVal.Arr
				} else {
					row = []Value{rowVal}
				}

				if rowExcluded(row) {
					continue
				}

				if err := writeRow(rowIdx, row); err != nil {
					return Value{
						Kind: KindStr,
						Str:  "error: " + err.Error(),
					}
				}

				rowIdx++
			}
		}

		// ------------------------------------------------------------
		// Leeres Sheet1 von excelize.NewFile() entfernen
		// ------------------------------------------------------------

		if sheetName != "Sheet1" {
			if idx, err := f.GetSheetIndex("Sheet1"); err == nil && idx >= 0 {
				_ = f.DeleteSheet("Sheet1")
			}
		}

		// ------------------------------------------------------------
		// Aktives Blatt setzen
		// ------------------------------------------------------------

		if idx, err := f.GetSheetIndex(sheetName); err == nil && idx >= 0 {
			f.SetActiveSheet(idx)
		}

		// ------------------------------------------------------------
		// Datei speichern
		// ------------------------------------------------------------

		if err := f.SaveAs(filename); err != nil {
			return Value{
				Kind: KindStr,
				Str:  "error: " + err.Error(),
			}
		}

		return Value{
			Kind: KindStr,
			Str:  "ok",
		}
	})

	Register(ns+"FromXLSX", "array", "path, [sheetName, exclude, column]", "Lädt eine XLSX-Datei in ein Array. Bei einer Spalte wird ein 1D-Array, bei mehreren Spalten ein 2D-Array zurückgegeben. Mit column kann gezielt eine Spalte eingelesen werden. Leere Zeilen werden ignoriert.", func(args []Value) Value {
		if len(args) < 1 {
			return Value{
				Kind: KindArr,
				Arr:  []Value{},
			}
		}

		path := args[0].Str

		// ------------------------------------------------------------
		// XLSX öffnen
		// ------------------------------------------------------------

		f, err := excelize.OpenFile(path)
		if err != nil {
			return Value{
				Kind: KindArr,
				Arr:  []Value{},
			}
		}
		defer f.Close()

		// ------------------------------------------------------------
		// Tabellenblatt
		// ------------------------------------------------------------

		sheetName := ""

		if len(args) >= 2 &&
			args[1].Kind == KindStr &&
			args[1].Str != "" {

			sheetName = args[1].Str

		} else {

			sheetName = f.GetSheetName(0)
		}

		// ------------------------------------------------------------
		// Ausschlusswerte
		// ------------------------------------------------------------

		excludeSet := make(map[string]struct{})

		if len(args) >= 3 && args[2].Kind == KindArr {
			for _, ex := range args[2].Arr {
				excludeSet[ToString(ex)] = struct{}{}
			}
		}

		rowExcluded := func(row []string) bool {
			if len(excludeSet) == 0 {
				return false
			}

			for _, cell := range row {
				value := strings.TrimSpace(cell)

				if _, exists := excludeSet[value]; exists {
					return true
				}
			}

			return false
		}

		// ------------------------------------------------------------
		// Leere Zeile
		//
		// Eine Zeile gilt als leer, wenn alle vorhandenen
		// Zellen nach TrimSpace leer sind.
		// ------------------------------------------------------------

		rowEmpty := func(row []string) bool {
			for _, cell := range row {
				if strings.TrimSpace(cell) != "" {
					return false
				}
			}

			return true
		}

		// ------------------------------------------------------------
		// Spalte
		//
		// -1 = alle Spalten
		//  0 = A
		//  1 = B
		//  2 = C
		// ------------------------------------------------------------

		column := -1

		if len(args) >= 4 && args[3].Kind == KindNum {
			column = int(args[3].Num)

			if column < 0 {
				column = -1
			}
		}

		// ------------------------------------------------------------
		// XLSX lesen
		// ------------------------------------------------------------

		rows, err := f.GetRows(sheetName)
		if err != nil {
			return Value{
				Kind: KindArr,
				Arr:  []Value{},
			}
		}

		// ============================================================
		// GEZIELTE SPALTE
		// ============================================================

		if column >= 0 {

			result := make([]Value, 0, len(rows))

			for _, row := range rows {

				// Ausschluss und komplett leere Zeilen ignorieren
				if rowExcluded(row) || rowEmpty(row) {
					continue
				}

				// Spalte existiert in dieser Zeile nicht
				if column >= len(row) {
					result = append(
						result,
						StrVal(""),
					)

				} else {

					result = append(
						result,
						StrVal(strings.TrimSpace(row[column])),
					)
				}
			}

			return Value{
				Kind: KindArr,
				Arr:  result,
			}
		}

		// ============================================================
		// ALLE SPALTEN
		// ============================================================

		var filteredRows [][]string

		maxColumns := 0

		for _, row := range rows {

			// Ausschluss und komplett leere Zeilen ignorieren
			if rowExcluded(row) || rowEmpty(row) {
				continue
			}

			filteredRows = append(filteredRows, row)

			if len(row) > maxColumns {
				maxColumns = len(row)
			}
		}

		// ------------------------------------------------------------
		// Keine Daten
		// ------------------------------------------------------------

		if len(filteredRows) == 0 {
			return Value{
				Kind: KindArr,
				Arr:  []Value{},
			}
		}

		// ============================================================
		// EINE SPALTE -> 1D-ARRAY
		// ============================================================

		if maxColumns == 1 {

			result := make([]Value, 0, len(filteredRows))

			for _, row := range filteredRows {

				if len(row) == 0 {
					result = append(
						result,
						StrVal(""),
					)
				} else {
					result = append(
						result,
						StrVal(strings.TrimSpace(row[0])),
					)
				}
			}

			return Value{
				Kind: KindArr,
				Arr:  result,
			}
		}

		// ============================================================
		// MEHRERE SPALTEN -> 2D-ARRAY
		// ============================================================

		res2D := make([][]Value, 0, len(filteredRows))

		for _, row := range filteredRows {

			resRow := make([]Value, len(row))

			for j, cell := range row {
				resRow[j] = StrVal(
					strings.TrimSpace(cell),
				)
			}

			res2D = append(res2D, resRow)
		}

		return Value{
			Kind:  KindArr2D,
			Arr2D: res2D,
		}
	})

	Register(ns+"XLSXSheets", "array", "path", "Gibt die Namen aller Tabellenblätter einer XLSX-Datei zurück.", func(args []Value) Value {
		if len(args) < 1 {
			return Value{Kind: KindArr, Arr: []Value{}}
		}

		f, err := excelize.OpenFile(args[0].Str)
		if err != nil {
			return Value{Kind: KindArr, Arr: []Value{}}
		}
		defer f.Close()

		names := f.GetSheetList()
		result := make([]Value, len(names))
		for i, n := range names {
			result[i] = StrVal(n)
		}
		return Value{Kind: KindArr, Arr: result}
	})

	Register(ns+"Clone", "array", "array", "Erstellt eine Kopie eines Arrays (shallow copy)", func(args []Value) Value {
		if len(args) < 1 || args[0].Kind != KindArr {
			return Value{}
		}

		return Value{
			Kind: KindArr,
			Arr:  cloneArray(args[0].Arr),
		}
	})

	Register(ns+"Reverse", "array", "array", "Gibt ein neues Array mit umgekehrter Reihenfolge zurück", func(args []Value) Value {
		if len(args) < 1 || args[0].Kind != KindArr {
			return Value{}
		}

		src := args[0].Arr
		n := len(src)

		res := make([]Value, n)
		for i := 0; i < n; i++ {
			res[i] = src[n-1-i]
		}

		return Value{Kind: KindArr, Arr: res}
	})

	Register(ns+"Sort", "array", "array", "Sortiert ein Array (numerisch oder lexikografisch) und gibt eine Kopie zurück", func(args []Value) Value {
		if len(args) < 1 || args[0].Kind != KindArr {
			return Value{}
		}

		res := cloneArray(args[0].Arr)

		sort.Slice(res, func(i, j int) bool {
			if res[i].Kind == KindNum && res[j].Kind == KindNum {
				return res[i].Num < res[j].Num
			}
			return ToString(res[i]) < ToString(res[j])
		})

		return Value{Kind: KindArr, Arr: res}
	})

	Register(ns+"NaturalSort", "array", "array", "Sortiert ein Array nach natürlicher Reihenfolge und gibt eine Kopie zurück.", func(args []Value) Value {
		if len(args) < 1 || args[0].Kind != KindArr {
			return Value{}
		}

		res := cloneArray(args[0].Arr)

		sort.SliceStable(res, func(i, j int) bool {

			// Reine Zahlen numerisch vergleichen
			if res[i].Kind == KindNum && res[j].Kind == KindNum {
				return res[i].Num < res[j].Num
			}

			// Natürliche Sortierung
			return naturalLess(
				ToString(res[i]),
				ToString(res[j]),
			)
		})

		return Value{
			Kind: KindArr,
			Arr:  res,
		}
	})

	Register(ns+"Unique", "array", "array", "Entfernt doppelte Werte (typsicherer Vergleich)", func(args []Value) Value {
		if len(args) < 1 || args[0].Kind != KindArr {
			return Value{}
		}

		seen := make(map[string]bool)
		var result []Value

		makeKey := func(v Value) string {
			switch v.Kind {
			case KindNum:
				return "n:" + strconv.FormatFloat(v.Num, 'g', -1, 64)
			case KindStr:
				return "s:" + v.Str
			default:
				return fmt.Sprintf("k:%d:%v", v.Kind, v)
			}
		}

		for _, v := range args[0].Arr {
			key := makeKey(v)
			if !seen[key] {
				seen[key] = true
				result = append(result, v)
			}
		}

		return Value{Kind: KindArr, Arr: result}
	})

	Register(ns+"IsEmpty", "array", "array", "Prüft, ob ein Array leer ist (true = leer, false = nicht leer)", func(args []Value) Value {
		if len(args) < 1 || args[0].Kind != KindArr {
			return Value{Kind: KindBool, Bool: true}
		}

		if len(args[0].Arr) == 0 {
			return Value{Kind: KindBool, Bool: true}
		}

		return Value{Kind: KindBool, Bool: false}
	})

	Register(ns+"Merge", "array", "array, array", "Verbindet zwei Arrays zu einem neuen Array", func(args []Value) Value {
		if len(args) < 2 || args[0].Kind != KindArr || args[1].Kind != KindArr {
			return Value{}
		}

		a := cloneArray(args[0].Arr)
		a = append(a, args[1].Arr...)

		return Value{Kind: KindArr, Arr: a}
	})

	Register(ns+"Remove", "array", "arr, value", "Sucht den Wert im Array und entfernt das erste Vorkommen (In-Place).", func(args []Value) Value {
		if len(args) < 2 || args[0].Kind != KindArr {
			return args[0]
		}

		arr := args[0].Arr
		searchStr := ToString(args[1]) // Wir vergleichen als String für maximale Flexibilität
		foundIdx := -1

		// 1. Suche nach dem Wert
		for i, v := range arr {
			if ToString(v) == searchStr {
				foundIdx = i
				break
			}
		}

		// 2. Wenn gefunden, In-Place löschen
		if foundIdx != -1 {
			args[0].Arr = append(arr[:foundIdx], arr[foundIdx+1:]...)
		}

		return args[0]
	})

	Register(ns+"Clear", "array", "array", "Leert ein Array (In-Place).", func(args []Value) Value {
		if len(args) < 1 || args[0].Kind != KindArr {
			return args[0]
		}

		args[0].Arr = args[0].Arr[:0]

		return args[0]
	})

	// array.Chunk(arr, size) -> [ [1,2], [3,4], [5] ]
	Register(ns+"Chunk", "array", "arr, size", "Unterteilt ein 1D-Array in mehrere Teil-Arrays (2D).", func(args []Value) Value {
		// Validierung
		if len(args) < 2 || args[0].Kind != KindArr {
			return Value{Kind: KindArr2D, Arr2D: [][]Value{}}
		}

		src := args[0].Arr
		size := int(toNumVal(args[1]))

		// Sicherheit: Größe muss positiv sein
		if size <= 0 {
			// Entweder leere Matrix oder das ganze Array als eine Zeile
			if len(src) == 0 {
				return Value{Kind: KindArr2D, Arr2D: [][]Value{}}
			}
			return Value{Kind: KindArr2D, Arr2D: [][]Value{cloneArray(src)}}
		}

		var res [][]Value
		for i := 0; i < len(src); i += size {
			end := i + size
			if end > len(src) {
				end = len(src)
			}

			// Wir erstellen einen neuen Chunk (Deep Copy)
			res = append(res, cloneArray(src[i:end]))
		}

		return Value{Kind: KindArr2D, Arr2D: res}
	})

	// Find: Filtert das Array nach einem Suchbegriff (mit Wildcards)
	Register(ns+"Find", "array", "arr, pattern, [case]", "Filtert ein Array mit Wildcards (*, ?).", func(args []Value) Value {
		if len(args) < 2 || args[0].Kind != KindArr {
			return Value{Kind: KindArr, Arr: []Value{}}
		}

		pattern := ToString(args[1])
		caseSensitive := false
		if len(args) >= 3 {
			caseSensitive = isTruthy(args[2])
		}

		var result []Value
		for _, v := range args[0].Arr {
			valStr := ToString(v)

			searchStr := valStr
			matchPattern := pattern

			if !caseSensitive {
				searchStr = strings.ToLower(valStr)
				matchPattern = strings.ToLower(pattern)
			}

			// Nutzt Shell-Style Wildcards wie "Log_*.txt"
			matched, _ := path.Match(matchPattern, searchStr)

			// Fallback: Wenn kein Wildcard da ist, nutze Contains
			if !strings.Contains(pattern, "*") && !strings.Contains(pattern, "?") {
				matched = strings.Contains(searchStr, matchPattern)
			}

			if matched {
				result = append(result, v)
			}
		}
		return Value{Kind: KindArr, Arr: result}
	})

	Register(ns+"KindOf", "array", "value", "Gibt den Datentyp des übergebenen Wertes als Text zurück (z.B. 'string', 'number', 'array').", func(args []Value) Value {
		if len(args) < 1 {
			return StrVal("none")
		}
		// Viel sauberer:
		return StrVal(GetKindName(args[0].Kind))
	})

	Register(ns+"RemoveAt", "array", "arr, index", "Entfernt das Element an der angegebenen Position (In-Place).", func(args []Value) Value {
		if len(args) < 2 || args[0].Kind != KindArr {
			return args[0] // Nichts zu tun oder falscher Typ
		}

		// Index auslesen (VB.NET Style: nur positive Zahlen)
		idx := int(toNumVal(args[1]))
		arr := args[0].Arr

		// Bounds-Check: Nur löschen, wenn der Index wirklich existiert
		if idx >= 0 && idx < len(arr) {
			// Go-Idiom zum Löschen eines Elements aus einem Slice:
			// Wir setzen das Original-Slice neu zusammen, ohne das Element bei 'idx'
			args[0].Arr = append(arr[:idx], arr[idx+1:]...)
		}

		// Wir geben das modifizierte Array zurück (erlaubt Chaining)
		return args[0]
	})
}

func cloneArray(src []Value) []Value {
	dst := make([]Value, len(src))
	copy(dst, src)
	return dst
}

func ensureUTF8(b []byte) []byte {
	if utf8.Valid(b) {
		return b
	}
	// Wenn nicht valide, konvertieren wir von Windows-1252 (ANSI) nach UTF-8
	result := make([]rune, len(b))
	for i, ch := range b {
		result[i] = rune(ch)
	}
	return []byte(string(result))
}

func GetKindName(k Kind) string {
	switch k {
	case KindNum:
		return "Zahl (Double)"
	case KindStr:
		return "Text (String)"
	case KindBool:
		return "Boolean"
	case KindNull:
		return "Null/Nothing"
	case KindArr:
		return "Array (1D)"
	case KindArr2D:
		return "Array (2D)"
	case KindObj:
		return "Objekt"
	case KindError:
		return "Fehler-Objekt"
	default:
		return "unbekannter Typ"
	}
}

func getBoundaryElement(args []Value, isFirst bool) Value {
	if len(args) < 1 {
		return NullVal()
	}
	v := args[0]

	if v.Kind == KindArr {
		if len(v.Arr) == 0 {
			return NullVal()
		}
		if isFirst {
			return v.Arr[0]
		}
		return v.Arr[len(v.Arr)-1]
	}

	if v.Kind == KindArr2D {
		if len(v.Arr2D) == 0 {
			return NullVal()
		}
		var row []Value
		if isFirst {
			row = v.Arr2D[0]
		} else {
			row = v.Arr2D[len(v.Arr2D)-1]
		}
		return Value{Kind: KindArr, Arr: row}
	}

	return ErrorVal(fmt.Sprintf("Funktion kann nicht auf %s angewendet werden", GetKindName(v.Kind)))
}

// Hilfsfunktion für die Logik (DRY - Don't Repeat Yourself)
func getArrayCount(args []Value) Value {
	if len(args) < 1 {
		return NumVal(0)
	}
	v := args[0]

	// Fall 1: 1D-Array
	if v.Kind == KindArr {
		return NumVal(float64(len(v.Arr)))
	}

	// Fall 2: 2D-Matrix
	if v.Kind == KindArr2D {
		// Wenn ein zweiter Parameter angegeben ist (z.B. für Spalten)
		// array.Count(m, 1) -> Zeilen (Standard)
		// array.Count(m, 2) -> Spalten
		dimension := 1
		if len(args) >= 2 {
			dimension = int(toNumVal(args[1]))
		}

		if dimension == 2 {
			if len(v.Arr2D) > 0 {
				return NumVal(float64(len(v.Arr2D[0])))
			}
			return NumVal(0)
		}
		// Standard: Zeilenanzahl
		return NumVal(float64(len(v.Arr2D)))
	}

	// Fallback für Strings (Länge des Texts)
	if v.Kind == KindStr {
		return NumVal(float64(len(v.Str)))
	}

	return NumVal(0)
}

// Prüft, ob der Index innerhalb des Arrays liegt (VB.NET Style: nur 0 bis Length-1)
func isValidIndex(idx int, length int) bool {
	return idx >= 0 && idx < length
}
