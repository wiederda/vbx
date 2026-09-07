package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ---------------- Main ----------------

// RunFile lädt, validiert, entschlüsselt und führt ein Skript aus.
func RunFile(filename string) error {
	_, _, err := runFileInternal(filename, false)
	return err
}

// CanExecuteFile lädt, validiert, entschlüsselt und parst ein Skript - GENAU wie
// RunFile, aber ohne es auszuführen (evalStatements wird übersprungen). Nutzt
// dieselbe Pipeline wie RunFile, damit ein Skript pro Aufruf nur einmal geparst
// wird, egal ob man es nur prüfen oder direkt ausführen will.
func CanExecuteFile(filename string) (int, error) {
	_, warnCount, err := runFileInternal(filename, true)
	return warnCount, err
}

// runFileInternal ist die gemeinsame Pipeline für RunFile und CanExecuteFile.
// Bei validateOnly=true wird nach dem Parsen (inkl. registerFuncsAndSubs)
// abgebrochen, bevor evalStatements läuft - es gibt also keine Ausführung und
// keine Seiteneffekte. Ein Fehler (Rückgabewert err) bedeutet in beiden Modi:
// "Skript kann/darf so nicht ausgeführt werden".
func runFileInternal(filename string, validateOnly bool) (finalVal Value, warnCount int, err error) {
	defer func() {
		if r := recover(); r != nil {
			if validateOnly {
				err = fmt.Errorf("%v", r)
			} else {
				err = fmt.Errorf("[SYNTAX ERROR] %v", r)
			}
		}
	}()

	// 1. Pfad absichern
	absName, errVal := absPathVal(filename)
	if errVal != nil {
		return Value{}, 0, fmt.Errorf("[SECURITY ERROR]: %s", errVal.Str)
	}
	filename = absName
	currentScriptName = filepath.Base(filename)
	scriptPath = filename

	// STUFE 1: Endungsprüfung
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".vb" && ext != ".vbc" {
		return Value{}, 0, fmt.Errorf("zugriff verweigert: '%s' ist kein gültiges Format", filename)
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return Value{}, 0, fmt.Errorf("lesefehler: %v", err)
	}

	// STUFE 2: Entschlüsselung bei .vbc
	if ext == ".vbc" {
		magic := "VBC!"
		if len(data) < len(magic) || string(data[:len(magic)]) != magic {
			return Value{}, 0, fmt.Errorf("validierungsfehler: Header fehlt in %s", filename)
		}
		data = crypt(data[len(magic):])
	} else if isBinary(data) {
		return Value{}, 0, fmt.Errorf("formatfehler: '%s' enthält ungültige Binärdaten", filename)
	}

	return runParsedInternal(data, filename, validateOnly)
}

// runContentInternal prüft VBX-Quelltext, der noch NICHT auf der Platte
// liegt (z.B. gerade von GitHub/net.Download geholt), ohne ihn zu
// speichern. Teilt sich die komplette Parse-/Precheck-/Ausführungs-
// Pipeline mit runFileInternal - nur Datei-Lesen und .vbc-Entschlüsselung
// entfallen, da hier immer Klartext vorliegt.
func runContentInternal(content string, label string, validateOnly bool) (finalVal Value, warnCount int, err error) {
	defer func() {
		if r := recover(); r != nil {
			if validateOnly {
				err = fmt.Errorf("%v", r)
			} else {
				err = fmt.Errorf("[SYNTAX ERROR] %v", r)
			}
		}
	}()

	data := []byte(content)

	if isBinary(data) {
		return Value{}, 0, fmt.Errorf("formatfehler: '%s' enthält ungültige Binärdaten", label)
	}

	return runParsedInternal(data, label, validateOnly)
}

// runParsedInternal ist der gemeinsame Kern: Sanity-Check (looksLikeScript
// - fängt genau den Fall ab, den du beschreibst: eine HTML-Fehlerseite
// statt echtem Skript-Rohtext nach einem fehlgeschlagenen Download),
// #use/#requires, Tokenizen, Parsen, Precheck (CheckUnknownCalls) und
// - falls validateOnly=false - Ausführung.
func runParsedInternal(data []byte, label string, validateOnly bool) (finalVal Value, warnCount int, err error) {

	if reason, ok := looksLikeScript(data); !ok {
		return Value{}, 0, fmt.Errorf("formatfehler: '%s' - %s", label, reason)
	}

	lines := strings.Split(string(data), "\n")

	env := NewEnvironment(nil)

	lines, scriptModules := ExtractUse(lines)

	if len(scriptModules) > 0 {
		LoadModules(env, scriptModules)
	}

	if len(lines) >= 2 {
		line2 := strings.TrimSpace(lines[1])
		if strings.HasPrefix(strings.ToLower(line2), "#requires") {
			reqVer := strings.TrimSpace(line2[9:])

			if reqVer != "" {
				if isVersionGreater(reqVer, Version) {
					return Value{}, 0, fmt.Errorf("inkompatibel: Skript (v%s) erfordert einen neueren vbmini (v%s)", reqVer, Version)
				}
			}
			lines[1] = ""
		}
	}

	code := strings.Join(lines, "\n")
	tokens := tokenize(code)
	parser := &Parser{tokens: tokens, env: env}
	stmts := parser.parse()

	if len(stmts) == 0 {
		return Value{}, 0, fmt.Errorf("formatfehler: '%s' enthält keine erkennbaren VBX-Anweisungen", label)
	}

	registerFuncsAndSubs(stmts)

	warnings := CheckUnknownCalls(stmts)
	warnCount = len(warnings)

	if validateOnly {
		for _, w := range warnings {
			msg := fmt.Sprintf("[WARNUNG] (%s) Unbekannter Aufruf: '%s'", w.Context, w.Name)
			if w.Suggestion != "" {
				msg += fmt.Sprintf(" - meintest du '%s'?", w.Suggestion)
			}
			fmt.Println(msg)
		}

		return Value{}, warnCount, nil
	}

	finalVal, sig := evalStatements(stmts, env)

	switch sig {
	case SignalError:
		return Value{}, warnCount, fmt.Errorf("[RUNTIME ERROR] %s", finalVal.Str)
	case SignalExitSub, SignalExitFunction, SignalExitLoop, SignalNone:
		return finalVal, warnCount, nil
	default:
		return Value{}, warnCount, fmt.Errorf("[UNKNOWN SIGNAL] %v", sig)
	}
}

// CanExecuteString prüft VBX-Quelltext direkt aus dem Speicher, OHNE ihn
// jemals auf die Platte zu schreiben oder auszuführen - erzwingt intern
// validateOnly=true, es gibt also keinen Weg, damit heruntergeladenen
// Code direkt scharf laufen zu lassen.
func CanExecuteString(content, label string) (int, error) {
	_, warnCount, err := runContentInternal(content, label, true)
	return warnCount, err
}

// Hilfsfunktion für den Versionsvergleich
func isVersionSmaller(testVer, baseVer string) bool {
	// Teilt beide Strings in Teile (z.B. "1", "1", "44")
	tParts := strings.Split(testVer, ".")
	bParts := strings.Split(baseVer, ".")

	for i := 0; i < len(tParts) && i < len(bParts); i++ {
		var tNum, bNum int
		fmt.Sscanf(tParts[i], "%d", &tNum)
		fmt.Sscanf(bParts[i], "%d", &bNum)

		if tNum < bNum {
			return true
		} // Test-Version ist kleiner
		if tNum > bNum {
			return false
		} // Test-Version ist größer
	}
	// Falls 1.1 kleiner ist als 1.1.1
	return len(tParts) < len(bParts)
}

func isTooFarAhead(reqVer, sysVer string, tolerance int) bool {
	var rMaj, rMin, rPat int
	var sMaj, sMin, sPat int
	fmt.Sscanf(reqVer, "%d.%d.%d", &rMaj, &rMin, &rPat)
	fmt.Sscanf(sysVer, "%d.%d.%d", &sMaj, &sMin, &sPat)

	// Major/Minor müssen passen
	if rMaj != sMaj || rMin != sMin {
		return true
	}

	// Wenn das Skript mehr als 2 Patches weiter ist als das System
	if (rPat - sPat) > tolerance {
		return true
	}
	return false
}

func isVersionGreater(testVer, baseVer string) bool {
	tParts := strings.Split(testVer, ".")
	bParts := strings.Split(baseVer, ".")

	for i := 0; i < len(tParts) && i < len(bParts); i++ {
		var tNum, bNum int
		fmt.Sscanf(tParts[i], "%d", &tNum)
		fmt.Sscanf(bParts[i], "%d", &bNum)

		if tNum > bNum {
			return true
		}
		if tNum < bNum {
			return false
		}
	}
	// Falls das Skript 1.1.1 fordert, aber System nur 1.1 ist
	return len(tParts) > len(bParts)
}

// ExtractUse entfernt #use-Zeilen aus dem Code
// und gibt die gefundenen Module zurück.
func ExtractUse(lines []string) ([]string, []string) {

	var modules []string
	var cleanLines []string

	for _, line := range lines {

		trim := strings.TrimSpace(line)

		if strings.HasPrefix(strings.ToLower(trim), "#use") {

			raw := strings.TrimSpace(trim[4:])

			if raw != "" {
				for _, m := range strings.Split(raw, ",") {

					m = strings.TrimSpace(strings.ToLower(m))

					if m != "" {
						modules = append(modules, m)
					}
				}
			}

			// #use nicht an Lexer/Parser weitergeben
			continue
		}

		cleanLines = append(cleanLines, line)
	}

	return cleanLines, modules
}
