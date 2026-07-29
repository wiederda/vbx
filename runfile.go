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
	_, err := runFileInternal(filename, false)
	return err
}

// CanExecuteFile lädt, validiert, entschlüsselt und parst ein Skript - GENAU wie
// RunFile, aber ohne es auszuführen (evalStatements wird übersprungen). Nutzt
// dieselbe Pipeline wie RunFile, damit ein Skript pro Aufruf nur einmal geparst
// wird, egal ob man es nur prüfen oder direkt ausführen will.
func CanExecuteFile(filename string) error {
	_, err := runFileInternal(filename, true)
	return err
}

// runFileInternal ist die gemeinsame Pipeline für RunFile und CanExecuteFile.
// Bei validateOnly=true wird nach dem Parsen (inkl. registerFuncsAndSubs)
// abgebrochen, bevor evalStatements läuft - es gibt also keine Ausführung und
// keine Seiteneffekte. Ein Fehler (Rückgabewert err) bedeutet in beiden Modi:
// "Skript kann/darf so nicht ausgeführt werden".
func runFileInternal(filename string, validateOnly bool) (finalVal Value, err error) {
	defer func() {
		if r := recover(); r != nil {
			// Wir wandeln die Panic in einen Rückgabewert 'err' um
			// Das verhindert den hässlichen Stacktrace UND verhindert,
			// dass ein kaputtes Skript den Prozess hängen/abstürzen lässt.
			err = fmt.Errorf("[SYNTAX ERROR] %v", r)
		}
	}()

	// 1. Pfad absichern
	absName, errVal := absPathVal(filename)
	if errVal != nil {
		return Value{}, fmt.Errorf("[SECURITY ERROR]: %s", errVal.Str)
	}
	filename = absName
	currentScriptName = filepath.Base(filename)

	// STUFE 1: Endungsprüfung
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".vb" && ext != ".vbc" {
		return Value{}, fmt.Errorf("zugriff verweigert: '%s' ist kein gültiges Format", filename)
	}

	// Datei einlesen
	data, err := os.ReadFile(filename)
	if err != nil {
		return Value{}, fmt.Errorf("lesefehler: %v", err)
	}

	// STUFE 2: Entschlüsselung bei .vbc
	if ext == ".vbc" {
		magic := "VBC!"
		if len(data) < len(magic) || string(data[:len(magic)]) != magic {
			return Value{}, fmt.Errorf("validierungsfehler: Header fehlt in %s", filename)
		}
		// Entschlüsseln (Header überspringen)
		data = crypt(data[len(magic):])
	} else if isBinary(data) {
		return Value{}, fmt.Errorf("formatfehler: '%s' enthält ungültige Binärdaten", filename)
	}

	// STUFE 3: Zusätzliche Sanity-Checks für offensichtlich falsche Inhalte
	// (z.B. HTML-Fehlerseite statt Git-Rohdatei nach einem net.Download).
	// Läuft in BEIDEN Modi, damit RunFile ein kaputtes Skript auch ohne
	// vorherigen CanExecuteFile-Aufruf gar nicht erst versucht auszuführen.
	if reason, ok := looksLikeScript(data); !ok {
		return Value{}, fmt.Errorf("formatfehler: '%s' - %s", filename, reason)
	}

	lines := strings.Split(string(data), "\n")

	env := NewEnvironment(nil)

	lines, scriptModules := ExtractUse(lines)

	if len(scriptModules) > 0 {
		LoadModules(env, scriptModules)
	}

	// --- ZEILE 2: #requires ---
	if len(lines) >= 2 {
		line2 := strings.TrimSpace(lines[1])
		if strings.HasPrefix(strings.ToLower(line2), "#requires") {
			reqVer := strings.TrimSpace(line2[9:])

			if reqVer != "" {
				// Regel 2 & 3: Prüfe, ob das Skript NEUER ist als das System
				if isVersionGreater(reqVer, Version) {
					// Regel 2: Skript 1.1.63 auf 1.1.62 -> STOPP
					return Value{}, fmt.Errorf("inkompatibel: Skript (v%s) erfordert einen neueren vbmini (v%s)", reqVer, Version)
				}
				// Regel 3: Skript 1.1.63 auf 1.1.64 -> isVersionGreater ist false -> LÄUFT
			}
			lines[1] = ""
		}
		// Regel 1: Kein #requires -> reqVer ist "" oder Präfix fehlt -> LÄUFT
	}

	// 4. Parser (ein einziges Mal, für beide Modi gemeinsam)
	code := strings.Join(lines, "\n")
	tokens := tokenize(code)
	parser := &Parser{
		tokens: tokens,
		env:    env,
	}
	stmts := parser.parse()

	// Sanity-Check NACH dem Parsen: reiner Text ohne echte VBX-Befehle wirft bei
	// einem tolerant geschriebenen Parser oft keine Panic, sondern liefert einfach
	// 0 Statements. Das ist so gut wie nie ein echtes Skript (z.B. eine Klartext-
	// Fehlermeldung statt der erwarteten Rohdatei nach einem net.Download).
	if len(stmts) == 0 {
		return Value{}, fmt.Errorf("formatfehler: '%s' enthält keine erkennbaren VBX-Anweisungen", filename)
	}

	// Funktionen/Subs registrieren
	registerFuncsAndSubs(stmts)

	// Bei reiner Validierung hier abbrechen - Parsen war erfolgreich,
	// keine Ausführung, keine Seiteneffekte.
	if validateOnly {
		return Value{}, nil
	}

	// Ausführen
	finalVal, sig := evalStatements(stmts, env)

	// 5. Signale auswerten
	switch sig {
	case SignalError:
		return Value{}, fmt.Errorf("[RUNTIME ERROR] %s", finalVal.Str)
	case SignalExitSub, SignalExitFunction, SignalExitLoop, SignalNone:
		return finalVal, nil
	default:
		return Value{}, fmt.Errorf("[UNKNOWN SIGNAL] %v", sig)
	}
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
