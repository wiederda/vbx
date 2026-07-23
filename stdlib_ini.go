// ------------------------
// stdlib_ini.go
// ------------------------

package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
)

// iniStore kapselt den gesamten INI-Zustand.
// Keine globalen Variablen mehr — mehrere Instanzen sind möglich
// und alle Operationen sind threadsicher.
type iniStore struct {
	mu   sync.RWMutex
	data map[string]map[string]string
	file string
}

// Paketweite Instanz (ersetzt die früheren globalen iniData / iniFile).
var defaultIni = &iniStore{}

// save schreibt den aktuellen Zustand atomar in die Datei.
// "Atomar" bedeutet: erst in eine .tmp-Datei schreiben, dann os.Rename —
// so ist die Originaldatei nie halb geschrieben, falls der Prozess abstürzt.
// Aufruf nur mit bereits gehaltenem Lock (mu.Lock).
func (s *iniStore) save() error {
	if s.file == "" {
		return fmt.Errorf("keine INI-Datei geladen (ini.Load zuerst aufrufen)")
	}

	var sb strings.Builder

	// Sektionen sortiert schreiben → deterministisches Diff-Verhalten
	sections := make([]string, 0, len(s.data))
	for sec := range s.data {
		sections = append(sections, sec)
	}
	sort.Strings(sections)

	for _, sec := range sections {
		sb.WriteString("[" + sec + "]\n")

		// Keys innerhalb einer Sektion ebenfalls sortieren
		keys := make([]string, 0, len(s.data[sec]))
		for k := range s.data[sec] {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			sb.WriteString(fmt.Sprintf("%s=%s\n", k, s.data[sec][k]))
		}
		sb.WriteString("\n")
	}

	// Atomar schreiben: tmp → Rename
	tmp := s.file + ".tmp"
	if err := os.WriteFile(tmp, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("temporäre Datei konnte nicht geschrieben werden: %w", err)
	}
	if err := os.Rename(tmp, s.file); err != nil {
		// Aufräumen, falls Rename scheitert
		_ = os.Remove(tmp)
		return fmt.Errorf("atomares Ersetzen fehlgeschlagen: %w", err)
	}
	return nil
}

// ---------------- Init INI Functions ----------------

func InitIniFunctions() {
	if builtins == nil {
		builtins = make(map[string]BuiltinInfo)
	}

	ns := "ini."
	s := defaultIni // Kurzreferenz auf die Store-Instanz

	// ---------------- ini.Load ----------------
	Register(ns+"Load", "ini", "filename", "Lädt eine INI-Datei in den Speicher. Erstellt einen leeren Cache, falls die Datei nicht existiert.", func(args []Value) Value {
		if len(args) < 1 {
			return ErrorVal("ini.Load(filename) benötigt einen Pfad")
		}

		path, errVal := absPathVal(args[0].Str)
		if errVal != nil {
			return *errVal
		}

		s.mu.Lock()
		defer s.mu.Unlock()

		s.file = path
		s.data = make(map[string]map[string]string)

		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				// Neue Konfiguration — leerer Cache ist gültiger Zustand
				return NullVal()
			}
			return ErrorVal("INI-Lesefehler: " + err.Error())
		}

		var currentSection string
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)

			// Leerzeilen und Kommentare überspringen
			if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
				continue
			}

			// Sektion erkennen: [SectionName]
			if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
				// Sektionsnamen normalisieren: Leerzeichen trimmen.
				// Groß-/Kleinschreibung bleibt erhalten — bei Bedarf hier
				// strings.ToLower() ergänzen und im Header dokumentieren.
				currentSection = strings.TrimSpace(line[1 : len(line)-1])
				if s.data[currentSection] == nil {
					s.data[currentSection] = make(map[string]string)
				}
				continue
			}

			if currentSection == "" {
				continue // Einträge außerhalb einer Sektion ignorieren
			}

			// Key=Value — Inline-Kommentare ("; …") werden NICHT getrimmt.
			// Wer sie braucht, kann hier strings.SplitN(val, ";", 2)[0] ergänzen.
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				s.data[currentSection][key] = val
			}
		}

		return NullVal()
	})

	// ---------------- ini.Get ----------------
	Register(ns+"Get", "ini", "section, key [, default]", "Liest einen Wert aus einer Sektion. Gibt den Default-Wert (oder leer) zurück, falls nicht gefunden.", func(args []Value) Value {
		if len(args) < 2 {
			return StrVal("")
		}

		section := args[0].Str
		key := args[1].Str
		def := ""
		if len(args) >= 3 {
			def = args[2].Str
		}

		s.mu.RLock()
		defer s.mu.RUnlock()

		if sec, ok := s.data[section]; ok {
			if val, ok := sec[key]; ok {
				return StrVal(val)
			}
		}
		return StrVal(def)
	})

	// ---------------- ini.Set ----------------
	Register(ns+"Set", "ini", "section, key, value [, autosave]", "Setzt einen Wert. 'autosave' (Standard: true) schreibt Änderungen sofort in die Datei.", func(args []Value) Value {
		if len(args) < 3 {
			return ErrorVal("ini.Set(section, key, value, [autosave]) benötigt mindestens 3 Argumente")
		}

		section := args[0].Str
		key := args[1].Str

		var value string
		switch args[2].Kind {
		case KindStr:
			value = args[2].Str
		case KindNum:
			value = fmt.Sprintf("%v", args[2].Num)
		default:
			value = args[2].Str
		}

		saveImmediately := true
		if len(args) > 3 {
			saveImmediately = isTruthy(args[3])
		}

		s.mu.Lock()
		defer s.mu.Unlock()

		if s.data[section] == nil {
			s.data[section] = make(map[string]string)
		}
		s.data[section][key] = value

		if saveImmediately {
			if err := s.save(); err != nil {
				return ErrorVal("Fehler beim Speichern der INI: " + err.Error())
			}
		}
		return NullVal()
	})

	// ---------------- ini.Save ----------------
	// Expliziter Flush für den Fall, dass autosave=false verwendet wird.
	Register(ns+"Save", "ini", "", "Schreibt alle ausstehenden Änderungen manuell in die Datei (nützlich nach ini.Set mit autosave=false).", func(args []Value) Value {
		s.mu.Lock()
		defer s.mu.Unlock()

		if err := s.save(); err != nil {
			return ErrorVal("ini.Save fehlgeschlagen: " + err.Error())
		}
		return NullVal()
	})

	// ---------------- ini.Exists ----------------
	Register(ns+"Exists", "ini", "section, key", "Prüft, ob ein Key in einer Sektion existiert und nicht leer ist. Gibt true oder false zurück.", func(args []Value) Value {
		if len(args) < 2 {
			return Value{Kind: KindBool, Bool: false}
		}

		s.mu.RLock()
		defer s.mu.RUnlock()

		if val, ok := s.data[args[0].Str][args[1].Str]; ok && val != "" {
			return Value{Kind: KindBool, Bool: true}
		}
		return Value{Kind: KindBool, Bool: false}
	})

	// ---------------- ini.Sections ----------------
	// Korrigierte Argument-Reihenfolge bei Register(): params vor description.
	Register(ns+"Sections", "ini", "", "Gibt ein sortiertes Array mit allen Sektionsnamen der geladenen INI-Datei zurück.", func(args []Value) Value {
		s.mu.RLock()
		defer s.mu.RUnlock()

		names := make([]string, 0, len(s.data))
		for sec := range s.data {
			names = append(names, sec)
		}
		sort.Strings(names) // konsistente Reihenfolge

		arr := make([]Value, len(names))
		for i, n := range names {
			arr[i] = StrVal(n)
		}
		return Value{Kind: KindArr, Arr: arr}
	})

	// ---------------- ini.Keys ----------------
	Register(ns+"Keys", "ini", "section", "Gibt ein sortiertes Array mit allen Keys einer bestimmten Sektion zurück.", func(args []Value) Value {
		if len(args) < 1 {
			return Value{Kind: KindArr, Arr: []Value{}}
		}

		s.mu.RLock()
		defer s.mu.RUnlock()

		sec := s.data[args[0].Str]
		keys := make([]string, 0, len(sec))
		for k := range sec {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		arr := make([]Value, len(keys))
		for i, k := range keys {
			arr[i] = StrVal(k)
		}
		return Value{Kind: KindArr, Arr: arr}
	})

	// ---------------- ini.Delete ----------------
	Register(ns+"Delete", "ini", "section [, key]", "Löscht einen Key oder eine ganze Sektion. Änderungen werden sofort gespeichert.", func(args []Value) Value {
		if len(args) < 1 {
			return ErrorVal("ini.Delete(section, [key]) benötigt mindestens die Sektion")
		}

		section := args[0].Str

		s.mu.Lock()
		defer s.mu.Unlock()

		modified := false

		if len(args) == 1 {
			if _, ok := s.data[section]; ok {
				delete(s.data, section)
				modified = true
			}
		} else {
			key := args[1].Str
			if sec, ok := s.data[section]; ok {
				if _, exists := sec[key]; exists {
					delete(sec, key)
					modified = true
					if len(sec) == 0 {
						delete(s.data, section) // leere Sektionen aufräumen
					}
				}
			}
		}

		if modified {
			if err := s.save(); err != nil {
				return ErrorVal("Fehler beim Aktualisieren der INI: " + err.Error())
			}
		}

		return NullVal()
	})
}
