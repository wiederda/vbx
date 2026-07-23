// ------------------------
// helpers.go
// ------------------------

package main

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// ------------------------
// Value-Konvertierungen
// ------------------------

// MustValue gibt das Value an Index idx zurück oder einen leeren Value, falls nicht vorhanden
func MustValue(args []Value, idx int) Value {
	if idx < len(args) {
		return args[idx]
	}
	return Value{}
}

// getArg gibt Argument an Position i zurück oder KindInvalid
func getArg(args []Value, i int) Value {
	if i >= 0 && i < len(args) {
		return args[i]
	}
	return Value{Kind: KindInvalid}
}

func StrVal(s string) Value {
	return Value{Kind: KindStr, Str: s}
}

func NumVal(n float64) Value {
	return Value{Kind: KindNum, Num: n}
}

func IntVal(i int64) Value {
	return Value{
		Kind: KindNum,
		Num:  float64(i),
	}
}

func StrErr(msg string) Value {
	return Value{Kind: KindStr, Str: "error: " + msg}
}

func StrOk() Value {
	return Value{Kind: KindStr, Str: "ok"}
}

func BoolVal(b bool) Value {
	return Value{Kind: KindBool, Bool: b}
}

func ErrVal(err error) Value {
	if err == nil {
		return StrVal("")
	}
	return StrVal("error: " + err.Error())
}

// ToInt konvertiert Value zu int
func ToInt(v Value) (int, bool) {
	switch v.Kind {
	case KindNum:
		return int(v.Num), true

	case KindStr:
		s := strings.TrimSpace(v.Str)
		if s == "" {
			return 0, false
		}
		i, err := strconv.Atoi(s)
		if err == nil {
			return i, true
		}
	}
	return 0, false
}

// ToFloat konvertiert Value zu float64
func ToFloat(v Value) (float64, bool) {

	switch v.Kind {

	case KindNum:
		return v.Num, true

	case KindStr:
		s := strings.TrimSpace(v.Str)
		if s == "" {
			return 0, false
		}

		// deutsches Dezimal-Komma erlauben
		if strings.Count(s, ",") == 1 && !strings.Contains(s, ".") {
			s = strings.Replace(s, ",", ".", 1)
		}

		f, err := strconv.ParseFloat(s, 64)
		if err == nil {
			return f, true
		}
	}

	return 0, false
}

// toNumVal wandelt jeden Value in eine float64 Zahl um
func toNumVal(v Value) float64 {
	if f, ok := ToFloat(v); ok {
		return f
	}
	return 0
}

// isTruthy prüft, ob ein Value "wahr" ist (VB-Style)
func isTruthy(v Value) bool {
	switch v.Kind {
	case KindBool:
		return v.Bool
	case KindNum:
		return v.Num != 0
	case KindStr:
		return v.Str != ""
	case KindArr:
		return len(v.Arr) != 0 || len(v.Arr2D) != 0
	case KindObj:
		if v.Obj == nil {
			return false
		}
		return len(v.Obj.Fields) != 0
	case KindNull:
		return false
	default:
		return false
	}
}

// toArrVal wandelt Value in ein String-Array um (falls KindArr)
func toArrVal(v Value) []string {
	if v.Kind == KindArr {
		res := make([]string, len(v.Arr))
		for i, e := range v.Arr {
			res[i] = ToString(e)
		}
		return res
	}
	return []string{}
}

// toArr2DVal wandelt Value in ein 2D-Array von Strings um (falls KindArr2D)
func toArr2DVal(v Value) [][]string {
	if v.Kind == KindArr2D {
		res := make([][]string, len(v.Arr2D))
		for i, row := range v.Arr2D {
			res[i] = make([]string, len(row))
			for j, e := range row {
				res[i][j] = ToString(e)
			}
		}
		return res
	}
	return [][]string{}
}

// ToString konvertiert Value zu string
func ToString(v Value) string {
	switch v.Kind {

	case KindStr:
		return v.Str

	case KindNum:
		return strconv.FormatFloat(v.Num, 'g', -1, 64)

	case KindBool:
		if v.Bool {
			return "true"
		}
		return "false"

	case KindNull:
		return ""

	case KindArr:
		parts := make([]string, len(v.Arr))
		for i, e := range v.Arr {
			parts[i] = ToString(e)
		}
		return strings.Join(parts, ",")

	case KindArr2D:
		rows := make([]string, len(v.Arr2D))
		for i, row := range v.Arr2D {
			cols := make([]string, len(row))
			for j, cell := range row {
				cols[j] = ToString(cell)
			}
			rows[i] = strings.Join(cols, ",")
		}
		return strings.Join(rows, "\n")

	case KindObj:
		if v.Obj != nil {
			return fmt.Sprintf("%v", v.Obj)
		}
		return ""

	case KindError:
		return "Fehler: " + v.Str

	default:
		return ""
	}
}

// ToBool konvertiert Value zu bool (0/false -> false, alles andere -> true)
func ToBool(v Value) bool {
	switch v.Kind {
	case KindBool:
		return v.Bool

	case KindNum:
		return v.Num != 0

	case KindStr:
		s := strings.ToLower(strings.TrimSpace(v.Str))
		// Explizite Prüfung auf positive Werte
		return s == "true" || s == "1" || s == "yes" || s == "on" || s == "wahr"

	case KindNull:
		return false

	case KindArr:
		return len(v.Arr) > 0

	case KindArr2D:
		return len(v.Arr2D) > 0

	case KindObj:
		return v.Obj != nil

	default:
		return false
	}
}

// ------------------------
// Hilfsfunktionen
// ------------------------

// MustInt gibt int zurück oder Default
func MustInt(v Value, def int) int {
	if i, ok := ToInt(v); ok {
		return i
	}
	return def
}

func ArrVal(a []Value) Value {
	return Value{Kind: KindArr, Arr: a}
}

func NullVal() Value {
	return Value{Kind: KindNull}
}

func CountVal(v Value) int {
	if v.Kind == KindArr {
		return len(v.Arr)
	}
	return 0
}

func LBoundVal(v Value) int {
	if v.Kind == KindArr && len(v.Arr) > 0 {
		return 0
	}
	return -1
}

func UBoundVal(v Value) int {
	if v.Kind == KindArr && len(v.Arr) > 0 {
		return len(v.Arr) - 1
	}
	return -1
}

func FolderEmpty(args []Value) Value {
	if len(args) < 1 || args[0].Str == "" {
		return ErrorVal("folder.EmptyFolder benötigt einen Pfad")
	}

	// 1. Parameter & Pfad-Check
	path := args[0].Str
	force := false
	if len(args) >= 2 {
		val := strings.ToLower(args[1].Str)
		if val == "true" || val == "1" || val == "force" || val == "-f" {
			force = true
		}
	}

	abs, errVal := absPathVal(path)
	if errVal != nil {
		return *errVal
	}

	entries, err := os.ReadDir(abs)
	if err != nil {
		return ErrorVal("Ordner konnte nicht gelesen werden: " + err.Error())
	}

	// Liste für fehlgeschlagene Löschvorgänge
	var failedItems []Value

	// 2. Lösch-Schleife (Wir machen IMMER weiter)
	for _, e := range entries {
		fullPath := filepath.Join(abs, e.Name())

		if force {
			os.Chmod(fullPath, 0666) // Schreibschutz aufheben
		}

		err := os.RemoveAll(fullPath)
		if err != nil {
			// Fehler sammeln statt abbrechen
			failedItems = append(failedItems, Value{Kind: KindStr, Str: e.Name()})
		}
	}

	// 3. Rückgabe als Array
	// Wenn alles geklappt hat, ist das Array leer (Länge 0)
	return Value{Kind: KindArr, Arr: failedItems}
}

// Verwendet Readdirnames(1), um bei der ersten gefundenen Datei sofort abzubrechen.
func FolderIsEmpty(args []Value) Value {
	if len(args) < 1 {
		return ErrorVal("FolderIsEmpty benötigt einen Pfad")
	}

	// 1. Pfad-Sicherheitscheck
	abs, errVal := absPathVal(args[0].Str)
	if errVal != nil {
		return *errVal
	}

	// 2. Verzeichnis zum Streamen öffnen
	f, err := os.Open(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrorVal("Verzeichnis existiert nicht: " + abs)
		}
		if os.IsPermission(err) {
			return ErrorVal("Zugriff verweigert: Keine Leserechte für '" + abs + "'")
		}
		return ErrorVal("Fehler beim Öffnen des Ordners: " + err.Error())
	}
	defer f.Close()

	// 3. Nur den ersten Eintrag anfordern
	// Readdirnames ist effizienter als ReadDir, da es keine FileInfo-Structs baut
	_, err = f.Readdirnames(1)

	// Wenn der Fehler EOF (End of File) ist, war kein einziger Name vorhanden
	if err == io.EOF {
		return BoolVal(true)
	}

	// Falls ein anderer Fehler auftritt
	if err != nil {
		return ErrorVal("Fehler beim Lesen: " + err.Error())
	}

	// Ein Name wurde gefunden -> Ordner ist nicht leer
	return BoolVal(false)
}

// Hilfsfunktion: längster gemeinsamer Präfix von zwei Pfaden
func commonPrefix(a, b string) string {
	partsA := strings.Split(filepath.Clean(a), string(os.PathSeparator))
	partsB := strings.Split(filepath.Clean(b), string(os.PathSeparator))
	var common []string
	for i := 0; i < len(partsA) && i < len(partsB); i++ {
		if partsA[i] != partsB[i] {
			break
		}
		common = append(common, partsA[i])
	}
	return filepath.Join(common...)
}

// ---------------- Helper ----------------
func toStringSafe(v Value, def string) string {
	s := ToString(v)
	if s == "" {
		return def
	}
	return s
}

// toIntSafe gibt int zurück oder def, falls ungültig
func toIntSafe(args []Value, index int, def int) int {
	if index < 0 || index >= len(args) {
		return def
	}
	v := args[index]

	switch v.Kind {
	case KindNum:
		return int(v.Num)
	case KindStr:
		s := strings.TrimSpace(v.Str)
		if i, err := strconv.Atoi(s); err == nil {
			return i
		}
	case KindBool:
		if v.Bool {
			return 1
		}
		return 0
	}
	return def
}

// toFloatSafe gibt den float64-Wert eines Arguments zurück oder 0, falls ungültig
func toFloatSafe(args []Value, index int) float64 {
	if index < 0 || index >= len(args) {
		return 0
	}
	v := args[index]

	switch v.Kind {
	case KindNum:
		return v.Num
	case KindStr:
		s := strings.TrimSpace(v.Str)
		s = strings.Replace(s, ",", ".", 1) // Komma zu Punkt
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f
		}
	}
	return 0
}

func safeParseFloat(v Value) (float64, bool) {
	s := strings.TrimSpace(ToString(v))
	// Der entscheidende Teil: Komma zu Punkt!
	s = strings.ReplaceAll(s, ",", ".")

	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

// Hilfsfunktion: Holt ein String-Argument sicher ab
func getStrArg(args []Value, index int, funcName string) (string, Value) {
	if index < 0 || index >= len(args) {
		return "", ErrorVal(fmt.Sprintf("%s: Fehlendes Argument an Position %d", funcName, index+1))
	}
	// Wir nutzen das globale ToString, damit auch Zahlen/Bools automatisch zu Strings werden
	return ToString(args[index]), Value{}
}

// toBoolSafe gibt den bool-Wert eines Value zurück oder def, falls ungültig
func toBoolSafe(v Value, def bool) bool {
	switch v.Kind {
	case KindBool:
		return v.Bool
	case KindNum:
		return v.Num != 0
	case KindStr:
		s := strings.TrimSpace(v.Str)
		return s != "" && s != "0"
	default:
		if v.Obj != nil {
			return true
		}
		return def
	}
}

func IsNumeric(v Value) bool {
	// 1. Es ist bereits vom Typ Nummer
	if v.Kind == KindNum {
		return true
	}
	// 2. Es ist ein String, aber können wir ihn umwandeln?
	if v.Kind == KindStr {
		s := strings.TrimSpace(v.Str)
		s = strings.Replace(s, ",", ".", 1) // Komma-Korrektur
		_, err := strconv.ParseFloat(s, 64)
		return err == nil && s != ""
	}
	return false
}

func convertToValue(val interface{}) Value {
	// 1. NULL Handling
	if val == nil {
		return Value{Kind: KindStr, Str: ""}
	}

	// Falls der Treiber ein Pointer-Interface zurückgibt (wie oft in MSSQL/PG)
	// lösen wir den Pointer hier auf.
	if ptr, ok := val.(*interface{}); ok {
		return convertToValue(*ptr)
	}

	switch v := val.(type) {
	// 2. Zahlen (Alle Go-Zahlen-Typen auf float64 mappen)
	case int, int32, int64:
		// Wir nutzen eine kleine Helfer-Logik für den Cast
		switch n := any(v).(type) {
		case int:
			return NumVal(float64(n))
		case int32:
			return NumVal(float64(n))
		case int64:
			return NumVal(float64(n))
		}
	case float32:
		return NumVal(float64(v))
	case float64:
		return NumVal(v)

	// 3. Booleans
	case bool:
		return BoolVal(v)

	// 4. Zeitstempel (Der Synergie-Effekt mit deiner date-Klasse)
	case time.Time:
		// Hier nutzen wir dein defaultVBFormat ("02.01.2006 15:04:05")
		return StrVal(formatVB(v, ""))

	// 5. Strings und Byte-Slices
	case string:
		// Falls der String ein Datum ist (SQLite), jagen wir ihn durch den Cache
		if t, ok := parseDateSafe(v); ok {
			return StrVal(formatVB(t, ""))
		}
		return StrVal(v)

	case []byte:
		s := string(v)
		// Auch bei Bytes schauen wir, ob es ein Datum sein könnte (typisch für SQLite)
		if t, ok := parseDateSafe(s); ok {
			return StrVal(formatVB(t, ""))
		}
		return StrVal(s)

	// 6. Fallback für Exoten (UUIDs, JSONB etc.)
	default:
		return StrVal(fmt.Sprintf("%v", v))
	}

	return Value{Kind: KindStr, Str: ""}
}

func toRawGoValue(v Value) interface{} {
	switch v.Kind {
	case KindBool:
		return v.Bool // Gibt echtes true/false an den SQL-Treiber
	case KindNum:
		return v.Num
	case KindStr:
		return v.Str
	case KindArr:
		return fmt.Sprintf("%v", v.Arr)
	default:
		return nil
	}
}

func publicKey(priv crypto.PrivateKey) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

type MultiStmtNode struct {
	Stmts []Stmt
}

func ErrorVal(msg string) Value {
	return Value{
		Kind: KindError,
		Str:  msg,
	}
}

func extractRecipients(v Value) []string {
	var result []string

	if v.Kind == KindStr {
		result = append(result, v.Str)
	}

	if v.Kind == KindArr {
		for _, item := range v.Arr {
			if item.Kind == KindStr {
				result = append(result, item.Str)
			}
		}
	}

	return result
}

func humanSize(bytes float64) string {
	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	size := bytes
	i := 0
	for size >= 1024 && i < len(units)-1 {
		size /= 1024
		i++
	}
	// 1 Nachkommastelle
	return fmt.Sprintf("%.1f %s", size, units[i])
}

func compareValues(op TokenType, l, r Value) Value {
	if l.Kind == KindError {
		return l
	}
	if r.Kind == KindError {
		return r
	}

	switch op {

	case EQ:
		if l.Kind == KindStr || r.Kind == KindStr {
			return BoolVal(ToString(l) == ToString(r))
		}
		if l.Kind != KindNum || r.Kind != KindNum {
			return ErrorVal("type mismatch in comparison")
		}
		return BoolVal(l.Num == r.Num)

	case NEQ:
		if l.Kind == KindStr || r.Kind == KindStr {
			return BoolVal(ToString(l) != ToString(r))
		}
		if l.Kind != KindNum || r.Kind != KindNum {
			return ErrorVal("type mismatch in comparison")
		}
		return BoolVal(l.Num != r.Num)

	case LT:
		if l.Kind != KindNum || r.Kind != KindNum {
			return ErrorVal("type mismatch in comparison")
		}
		return BoolVal(l.Num < r.Num)

	case GT:
		if l.Kind != KindNum || r.Kind != KindNum {
			return ErrorVal("type mismatch in comparison")
		}
		return BoolVal(l.Num > r.Num)

	case LE:
		if l.Kind != KindNum || r.Kind != KindNum {
			return ErrorVal("type mismatch in comparison")
		}
		return BoolVal(l.Num <= r.Num)

	case GE:
		if l.Kind != KindNum || r.Kind != KindNum {
			return ErrorVal("type mismatch in comparison")
		}
		return BoolVal(l.Num >= r.Num)
	}

	return ErrorVal("invalid comparison operator")
}

func requireNumber(v Value, context string) (float64, Value) {
	switch v.Kind {
	case KindNum:
		return v.Num, Value{}
	case KindStr:
		// Deine getMathArg-Logik hier rein:
		s := strings.TrimSpace(v.Str)
		s = strings.ReplaceAll(s, ",", ".")
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f, Value{}
		}
	}

	// Einheitliche Fehlermeldung mit GetKindName
	msg := fmt.Sprintf("Fehler bei '%s': '%s' ist keine gültige Zahl.", context, GetKindName(v.Kind))
	return 0, ErrorVal(msg)
}

func (k Kind) String() string {
	switch k {
	case KindNum:
		return "Number"
	case KindStr:
		return "String"
	case KindBool:
		return "Boolean"
	case KindArr:
		return "1D Array"
	case KindArr2D:
		return "2D Array"
	case KindObj:
		return "Object"
	case KindNull:
		return "Null"
	case KindError:
		return "Error"
	default:
		return "Invalid"
	}
}

func isSafeIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '.') {
			return false
		}
	}
	return true
}

func expectArgCount(args []Value, expected int, name string) Value {
	if len(args) != expected {
		return ErrorVal(name + " erwartet " + strconv.Itoa(expected) + " Argument(e)")
	}
	return Value{}
}

func getFloat(args []Value, index int, name string) (float64, Value, bool) {
	if index < 0 || index >= len(args) {
		return 0, ErrorVal(name + ": Argument fehlt"), false
	}

	f, ok := ToFloat(args[index])
	if !ok {
		return 0, ErrorVal(name + ": Argument nicht numerisch"), false
	}

	return f, Value{}, true
}

// isBinary prüft die ersten 512 Bytes einer Datei.
// Wenn ein Null-Byte (0x00) gefunden wird, ist es mit 99,9% Sicherheit
// eine Binärdatei (Bild, EXE, ZIP), kein Quellcode.
func isBinary(data []byte) bool {
	limit := len(data)
	if limit > 512 {
		limit = 512
	}

	for i := 0; i < limit; i++ {
		if data[i] == 0 {
			return true
		}
	}
	return false
}

// formatSize wandelt Bytes in eine lesbare Form (KB, MB, GB...) um
func formatSize(size float64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%.0f B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", size/float64(div), "KMGTPE"[exp])
}

func formatBytes(size float64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	i := 0
	for size >= 1024 && i < len(units)-1 {
		size /= 1024
		i++
	}
	return fmt.Sprintf("%.2f %s", size, units[i])
}

// sanitizeNumberStr bereinigt Strings für die Konvertierung (Trim + Komma-Ersetzung)
func sanitizeNumberStr(v Value) string {
	s := strings.TrimSpace(ToString(v))
	return strings.ReplaceAll(s, ",", ".")
}

func ToNativePath(input string) string {
	// 1. Alle Backslashes in normale Slashes umwandeln (interner Standard)
	standardized := strings.ReplaceAll(input, "\\", "/")

	// 2. Doppelte Slashes am Anfang schützen (für Netzwerkpfade //server)
	isNetwork := strings.HasPrefix(standardized, "//")

	if runtime.GOOS == "windows" {
		// Zurück zu Backslashes für Windows
		final := strings.ReplaceAll(standardized, "/", "\\")
		// Falls es ein Netzwerkpfad war, sicherstellen, dass er mit \\ beginnt
		if isNetwork && !strings.HasPrefix(final, "\\\\") {
			final = "\\" + final
		}
		return final
	}

	// Für Linux/Unix/Mac: Bleibt bei Slashes
	if isNetwork && !strings.HasPrefix(standardized, "//") {
		standardized = "/" + standardized
	}
	return standardized
}

func parallelFolderScan(root string, ignoreMap map[string]bool) ScanResult {
	entries, err := os.ReadDir(root)
	if err != nil {
		return ScanResult{}
	}

	// Dynamische Worker-Anzahl (Max 4)
	numWorkers := runtime.NumCPU()
	if numWorkers > 4 {
		numWorkers = 4
	}
	if numWorkers < 1 {
		numWorkers = 1
	}

	jobs := make(chan string, len(entries))
	results := make(chan ScanResult, numWorkers)

	// Der eigentliche Worker
	worker := func() {
		var r ScanResult
		for p := range jobs {
			_ = filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if d.IsDir() {
					if ignoreMap[d.Name()] {
						return filepath.SkipDir
					}
					r.Dirs++
				} else {
					r.Files++
					if info, err := d.Info(); err == nil {
						r.Size += info.Size()
					}
				}
				return nil
			})
		}
		results <- r
	}

	// Start der Worker
	for i := 0; i < numWorkers; i++ {
		go worker()
	}

	// Verteilung der Top-Level Einträge
	var total ScanResult
	pending := 0
	for _, entry := range entries {
		if ignoreMap[entry.Name()] {
			continue
		}
		fullPath := filepath.Join(root, entry.Name())
		if entry.IsDir() {
			jobs <- fullPath
			pending++
		} else {
			total.Files++
			if info, err := entry.Info(); err == nil {
				total.Size += info.Size()
			}
		}
	}
	close(jobs)

	// Ergebnisse einsammeln
	for i := 0; i < numWorkers; i++ {
		res := <-results
		total.Files += res.Files
		total.Dirs += res.Dirs
		total.Size += res.Size
	}
	return total
}

// copyFileInternalBuffered kopiert eine Datei mit konfigurierbarem Buffer.
// Kleine Dateien: 32KB (RAM-schonend), große Dateien: 4MB (Netzwerk-optimiert).
func copyFileInternalBuffered(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	// Dateigröße prüfen für optimale Pufferwahl
	stat, err := in.Stat()
	if err != nil {
		return err
	}

	bufSize := 32 * 1024 // 32KB Standard (lokal/SSD)
	if stat.Size() > 1024*1024 {
		bufSize = 4 * 1024 * 1024 // 4MB für große Dateien (Netzwerk)
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	buf := make([]byte, bufSize)
	_, err = io.CopyBuffer(out, in, buf)
	return err
}
