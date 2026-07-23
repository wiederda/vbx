package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

var logFile *os.File
var currentScriptName string = "unnamed_script"
var lastLogPath string
var lastSampleTime time.Time
var lastThreadTime float64
var lastThreadSample time.Time

var lastCPUTime float64
var lastCPUSample time.Time

// InitStringFunctions registriert String-Funktionen
func InitDebugFunctions(env *Environment) {
	if builtins == nil {
		builtins = make(map[string]BuiltinInfo)
	}

	ns := "debug."

	// Globaler Zeitstempel (oder im Interpreter-Objekt)
	var globalTimerStart = time.Now()

	// In deiner Built-in-Initialisierung
	Register(ns+"Assert", "debug", "condition, [msg]", "Prüft eine Bedingung und bricht bei 'False' mit Fehlermeldung ab.", func(args []Value) Value {
		if len(args) == 0 {
			return ErrorVal("ASSERT erwartet mindestens ein Argument")
		}

		// Nutze deine neue isTruthy Funktion
		if !isTruthy(args[0]) {
			msg := "Assertion failed"
			if len(args) > 1 {
				msg = ToString(args[1])
			}

			// Das ErrorVal sorgt in evalFunctionCall dafür,
			// dass der Interpreter den Fehler erkennt und stoppt.
			return ErrorVal("[ASSERT] " + msg)
		}

		// Wenn alles okay ist, geben wir True zurück (VB-Style)
		return Value{Kind: KindBool, Bool: true}
	})

	Register(ns+"AssertEquals", "debug", "expected, actual, [info]", "Vergleicht zwei Werte und bricht bei Ungleichheit mit Zeilennummer ab.", func(args []Value) Value {
		if len(args) < 2 {
			return ErrorVal("AssertEquals erwartet 2 Argumente")
		}

		expected := args[0]
		actual := args[1]

		if !valuesAreEqual(expected, actual) {
			// JETZT nutzen wir das übergebene env!
			// Da env ein Pointer ist, sieht diese Funktion immer
			// den aktuellsten Stand von currentLine.
			lineInfo := fmt.Sprintf("[ZEILE %d]", env.currentLine)

			msg := fmt.Sprintf("%s Assertion failed: Erwartet [%s], erhalten [%s]",
				lineInfo, ToString(expected), ToString(actual))

			if len(args) > 2 {
				msg += " - Info: " + ToString(args[2])
			}

			return ErrorVal(msg)
		}

		return Value{Kind: KindBool, Bool: true}
	})

	// Timer Start
	Register(ns+"TimerStart", "debug", "-", "Startet einen hochauflösenden Timer für Performance-Messungen.", func(args []Value) Value {
		globalTimerStart = time.Now()
		return Value{Kind: KindBool, Bool: true} // Einfach True zurückgeben
	})

	// Timer MS
	Register(ns+"TimerMs", "debug", "-", "Gibt die verstrichene Zeit seit TimerStart in Millisekunden (mit Nachkommastellen) zurück.", func(args []Value) Value {
		if globalTimerStart.IsZero() {
			return Value{Kind: KindNum, Num: 0}
		}
		// Wir nehmen Nanosekunden und teilen durch 1.000.000 für echte Komma-Millisekunden
		// ODER wir geben direkt Mikrosekunden zurück.
		ms := float64(time.Since(globalTimerStart).Nanoseconds()) / 1e6
		return Value{Kind: KindNum, Num: ms}
	})

	// 1. Systemweite Last (0-100% über alle Kerne)
	Register(ns+"CPUUsage", "debug", "", "Last im Gesamtsystem.", func(args []Value) Value {
		p, _ := process.NewProcess(int32(os.Getpid()))
		times, _ := p.Times()
		now := time.Now()
		total := times.User + times.System

		if lastCPUSample.IsZero() {
			lastCPUTime, lastCPUSample = total, now
			return NumVal(0)
		}

		seconds := now.Sub(lastCPUSample).Seconds()
		// Hier teilen wir durch die Anzahl der Kerne
		percent := ((total - lastCPUTime) / seconds) * 100 / float64(runtime.NumCPU())

		lastCPUTime, lastCPUSample = total, now // Update
		return NumVal(percent)
	})

	// 2. Thread-Last (0-100% bezogen auf EINEN Kern)
	Register(ns+"ThreadUsage", "debug", "", "Last eines Kerns.", func(args []Value) Value {
		p, _ := process.NewProcess(int32(os.Getpid()))
		times, _ := p.Times()
		now := time.Now()
		total := times.User + times.System

		if lastThreadSample.IsZero() {
			lastThreadTime, lastThreadSample = total, now
			return NumVal(0)
		}

		seconds := now.Sub(lastThreadSample).Seconds()
		percent := ((total - lastThreadTime) / seconds) * 100

		lastThreadTime, lastThreadSample = total, now // Update
		if percent > 100 {
			percent = 100
		} // Deckeln auf 100%
		return NumVal(percent)
	})

	// 3. Aktueller Kern-Index
	Register(ns+"CurrentCPU", "debug", "", "Gibt den Index des CPU-Kerns zurück, auf dem das Skript gerade läuft.", func(args []Value) Value {
		// runtime.NumCPU() gibt die Gesamtanzahl,
		// aber es gibt keine Standard-Go-Funktion für die aktuelle Core-ID.
		// Unter Windows/Linux/Unix nutzen wir einen Trick oder syscall.
		return NumVal(float64(getCurrentCPU()))
	})

	Register(ns+"MemUsage", "debug", "-", "Gibt den aktuell vom Skript belegten Arbeitsspeicher in Megabyte (MB) zurück.", func(args []Value) Value {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		return NumVal(float64(m.Alloc) / 1024 / 1024)
	})

	Register(ns+"CloseLog", "debug", "-", "Schließt die aktive Log-Datei und stellt die Ausgabe wieder auf die Konsole um.", func(args []Value) Value {
		if logFile != nil {
			fmt.Fprintln(logFile, "--- LOG ENDE: "+time.Now().Format("15:04:05")+" ---")
			logFile.Sync() // Erzwingt das Schreiben auf die Disk
			logFile.Close()
			logFile = nil // Wichtig: Setzt den Status zurück auf "Konsole"
			fmt.Printf("\n\033[90m[INFO] Log gespeichert unter: %s\033[0m\n", lastLogPath)
		}
		return Value{Kind: KindBool, Bool: true}
	})

	Register(ns+"OpenLog", "debug", "-", "Erstellt eine automatische Log-Datei im System-Log-Ordner und leitet 'Print' dorthin um.", func(args []Value) Value {
		logDir := getLogDir()

		// Zeitstempel im Format: 20260319_1933
		// Go nutzt das Referenzdatum: 2006 (Year) 01 (Month) 02 (Day) 15 (Hour) 04 (Minute)
		timestamp := time.Now().Format("20060102_1504")

		// Den Skriptnamen säubern (Dateiendung .vbs entfernen)
		cleanName := strings.TrimSuffix(currentScriptName, filepath.Ext(currentScriptName))

		// Finaler Dateiname: mein_skript_20260319_1933.log
		fileName := fmt.Sprintf("%s_%s.log", cleanName, timestamp)
		fullPath := filepath.Join(logDir, fileName)

		f, err := os.OpenFile(fullPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return ErrorVal("Log-Fehler: " + err.Error())
		}
		logFile = f

		lastLogPath = fullPath

		fmt.Fprintln(logFile, "--- SESSION START: "+time.Now().Format("15:04:05")+" ---")
		return StrVal(fullPath)
	})

	Register(ns+"CleanOldLogs", "debug", "[days]", "Löscht alte Log-Dateien (Standard: 7 Tage), um Speicherplatz zu sparen.", func(args []Value) Value {
		logDir := getLogDir()

		// Standardwert: 7 Tage
		days := 7
		if len(args) > 0 && args[0].Kind == KindNum {
			days = int(args[0].Num)
		}

		// Zeitgrenze berechnen
		cutoff := time.Now().AddDate(0, 0, -days)

		files, err := os.ReadDir(logDir)
		if err != nil {
			return ErrorVal("Konnte Log-Verzeichnis nicht lesen: " + err.Error())
		}

		deletedCount := 0
		for _, file := range files {
			if file.IsDir() {
				continue
			}

			info, err := file.Info()
			if err != nil {
				continue
			}

			// Wenn die Datei älter als der Cutoff ist und auf .log endet
			if info.ModTime().Before(cutoff) && filepath.Ext(file.Name()) == ".log" {
				err := os.Remove(filepath.Join(logDir, file.Name()))
				if err == nil {
					deletedCount++
				}
			}
		}

		// Wir geben die Anzahl der gelöschten Dateien zurück
		return Value{Kind: KindNum, Num: float64(deletedCount)}
	})

}

var internalKey = []byte{0xAF, 0x42, 0x13, 0x37, 0xDE, 0xAD, 0xBE, 0xEF}

func crypt(data []byte) []byte {
	result := make([]byte, len(data))
	for i := 0; i < len(data); i++ {
		// XOR Verknüpfung: Wenn man es zweimal anwendet,
		// kommt wieder das Original heraus.
		result[i] = data[i] ^ internalKey[i%len(internalKey)]
	}
	return result
}

// Hilfsfunktion: Findet einen freien Dateinamen (z.B. datei_1.vbc)
func getAvailablePath(basePath string, ext string) string {
	folder := filepath.Dir(basePath)
	// Dateiname ohne Endung extrahieren
	name := strings.TrimSuffix(filepath.Base(basePath), filepath.Ext(basePath))

	finalPath := filepath.Join(folder, name+ext)

	// Wenn die Datei existiert, hängen wir eine Nummer an
	counter := 1
	for {
		if _, err := os.Stat(finalPath); os.IsNotExist(err) {
			break // Pfad ist frei!
		}
		finalPath = filepath.Join(folder, fmt.Sprintf("%s_%d%s", name, counter, ext))
		counter++
	}
	return finalPath
}

func dumpEnvironment(e *Environment) {
	current := e
	depth := 0

	fmt.Println("\n" + strings.Repeat("=", 45))
	fmt.Println("       ENVIRONMENT STACK DUMP")
	fmt.Println(strings.Repeat("=", 45))

	for current != nil {
		scopeName := "GLOBAL / PUBLIC"
		if current.parent != nil {
			scopeName = fmt.Sprintf("LOCAL SCOPE (Ebene %d)", depth)
		}

		fmt.Printf("\n[%s]\n", scopeName)

		if len(current.vars) == 0 {
			fmt.Println("  (leer)")
		} else {
			// 1. Namen in einen Slice sammeln
			var keys []string
			for name := range current.vars {
				// Interne Variablen (Start mit _) ignorieren
				if !strings.HasPrefix(name, "_") {
					keys = append(keys, name)
				}
			}

			// 2. Namen alphabetisch sortieren
			sort.Strings(keys)

			// 3. In der sortierten Reihenfolge ausgeben
			for _, name := range keys {
				ptr := current.vars[name]
				v := *ptr
				valStr := formatValue(v)
				fmt.Printf("  %-15s : %-10s = %s\n", name, v.Kind.String(), valStr)
			}
		}

		current = current.parent
		depth++
	}
	fmt.Println(strings.Repeat("=", 45) + "\n")
}

// Hilfsfunktion für schöne Array-Anzeige im Dump
func formatValue(v Value) string {
	// ANSI Farbcodes
	const (
		reset  = "\033[0m"
		green  = "\033[32m"
		yellow = "\033[33m"
		blue   = "\033[34m"
		cyan   = "\033[36m"
		red    = "\033[31m"
		gray   = "\033[90m"
	)

	switch v.Kind {
	case KindNum:
		return fmt.Sprintf(yellow+"%g"+reset, v.Num)
	case KindStr:
		return fmt.Sprintf(green+"\"%s\""+reset, v.Str)
	case KindBool:
		return fmt.Sprintf(cyan+"%t"+reset, v.Bool)

	case KindArr: // 1D-Array Vorschau
		length := len(v.Arr)
		if length == 0 {
			return gray + "Array(leer)" + reset
		}

		limit := 3
		if length < limit {
			limit = length
		}

		var items []string
		for i := 0; i < limit; i++ {
			items = append(items, formatValue(v.Arr[i]))
		}

		preview := strings.Join(items, ", ")
		if length > limit {
			preview += gray + ", ..." + reset
		}
		return fmt.Sprintf(blue+"Array(len:%d)"+reset+" [%s]", length, preview)

	case KindArr2D: // 2D-Array (Matrix) Vorschau
		rows := len(v.Arr2D)
		if rows == 0 {
			return gray + "Matrix(leer)" + reset
		}
		cols := len(v.Arr2D[0])

		var firstRow []string
		colLimit := 3
		if cols < colLimit {
			colLimit = cols
		}

		for j := 0; j < colLimit; j++ {
			firstRow = append(firstRow, formatValue(v.Arr2D[0][j]))
		}

		rowPreview := strings.Join(firstRow, gray+" | "+reset)
		if cols > colLimit {
			rowPreview += gray + " | ..." + reset
		}

		return fmt.Sprintf(blue+"Matrix(%dx%d)"+reset+" [Row 0: %s]", rows, cols, rowPreview)

	default:
		return red + "nil" + reset
	}
}

func getLogDir() string {
	var baseDir string
	var err error

	switch runtime.GOOS {
	case "windows":
		// C:\Users\Name\AppData\Roaming
		baseDir, err = os.UserConfigDir()
	case "darwin":
		// /Users/Name/Library/Logs
		home, _ := os.UserHomeDir()
		baseDir = filepath.Join(home, "Library", "Logs")
	case "linux":
		// /home/name/.cache oder /var/log (wir nehmen User-Cache/Logs)
		// XDG_CACHE_HOME ist der Standard für User-Logs unter Linux
		baseDir = os.Getenv("XDG_CACHE_HOME")
		if baseDir == "" {
			home, _ := os.UserHomeDir()
			baseDir = filepath.Join(home, ".cache")
		}
	default:
		// Fallback für andere (BSD etc.)
		baseDir, _ = os.UserHomeDir()
	}

	if err != nil || baseDir == "" {
		return "." // Letzter Ausweg: Aktueller Ordner
	}

	// Projekt-Unterordner erstellen (WICHTIG: Ersetze "MeinVBLang" durch dein Projekt-Namen)
	finalPath := filepath.Join(baseDir, "vbmini", "Logs")

	// Ordner-Struktur anlegen (0755 = Standard-Rechte)
	os.MkdirAll(finalPath, 0755)

	return finalPath
}

// In einer separaten Datei oder am Ende deiner main.go
func getCurrentCPU() int {
	// Dieser Wert ist unter Go oft schwer zu greifen, da der Scheduler
	// Goroutinen extrem schnell verschiebt.
	// Wir nutzen eine Annäherung oder rufen das System-API auf.
	return runtime.GOMAXPROCS(0) // Platzhalter: In der Realität nutzt man oft CGO oder syscalls wie sched_getcpu
}
