// ------------------------
// stdlib_keystore.go
// ------------------------

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// keytool-Datumsformat (englisch, UTC erzwungen): "Mon Jan 02 15:04:05 UTC 2006"
const ksDateLayout = "Mon Jan 02 15:04:05 MST 2006"

// ---------- keytool-Bridge ----------

type ksEntry struct {
	Alias   string
	Type    string
	Owner   string
	Until   time.Time
	HasDate bool // SecretKeyEntry hat z.B. kein Gültigkeitsdatum
}

func (e ksEntry) Expired(now time.Time) bool {
	return e.HasDate && e.Until.Before(now)
}

// keytoolPath: expliziter Parameter hat Vorrang, dann VBX_KEYTOOL,
// dann JAVA_HOME/bin, zuletzt PATH.
func keytoolPath(explicitPath string) string {
	if p := strings.TrimSpace(explicitPath); p != "" {
		return p
	}

	if p := strings.TrimSpace(os.Getenv("VBX_KEYTOOL")); p != "" {
		return p
	}

	exe := "keytool"
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}

	if jh := strings.TrimSpace(os.Getenv("JAVA_HOME")); jh != "" {
		candidate := filepath.Join(jh, "bin", exe)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	if p, err := exec.LookPath(exe); err == nil {
		return p
	}

	return ""
}

// runKeytool ruft keytool auf. Das Passwort wird per Umgebungsvariable
// übergeben (-storepass:env), damit es nicht in der Prozessliste steht.
// Sprache und Zeitzone werden erzwungen, damit die Ausgabe parsebar bleibt.
// Im Fehlerfall steht die keytool-Meldung in err (der Output ist dann leer).
func runKeytool(keytool, password string, args ...string) (string, error) {
	kt := keytoolPath(keytool)
	if kt == "" {
		return "", errors.New("keytool nicht gefunden (Parameter, VBX_KEYTOOL, JAVA_HOME oder PATH prüfen)")
	}

	full := append([]string{
		"-J-Duser.language=en",
		"-J-Duser.country=US",
		"-J-Duser.timezone=UTC",
	}, args...)

	full = append(full, "-storepass:env", "VBX_KS_PASS")

	cmd := exec.Command(kt, full...)
	cmd.Env = append(os.Environ(), "VBX_KS_PASS="+password)

	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("keytool: %s", msg)
	}

	return string(out), nil
}

// ksAliasMissing erkennt gezielt die keytool-Meldung "Alias <name> does not exist".
// Bewusst eng gefasst: "Keystore file does not exist" oder ein fehlendes
// keytool dürfen NICHT als "Alias nicht vorhanden" gewertet werden.
func ksAliasMissing(msg, alias string) bool {
	return strings.Contains(
		strings.ToLower(msg),
		"alias <"+strings.ToLower(alias)+"> does not exist",
	)
}

// ksAbsPath löst den Keystore-Pfad auf.
// TODO: auf absPathVal umstellen
func ksAbsPath(p string) (string, error) {
	return filepath.Abs(p)
}

// ksParse liest die Ausgabe von "keytool -list -v".
// Bei Zertifikatsketten zählt jeweils das erste Zertifikat (Leaf).
func ksParse(out string) []ksEntry {
	var entries []ksEntry
	var cur *ksEntry
	flush := func() {
		if cur != nil {
			entries = append(entries, *cur)
			cur = nil
		}
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		switch {
		case strings.HasPrefix(line, "Alias name:"):
			flush()
			cur = &ksEntry{Alias: strings.TrimSpace(strings.TrimPrefix(line, "Alias name:"))}
		case cur == nil:
			// Kopfzeilen vor dem ersten Eintrag ignorieren
		case strings.HasPrefix(line, "Entry type:"):
			cur.Type = strings.TrimSpace(strings.TrimPrefix(line, "Entry type:"))
		case strings.HasPrefix(line, "Owner:") && cur.Owner == "":
			cur.Owner = strings.TrimSpace(strings.TrimPrefix(line, "Owner:"))
		case strings.HasPrefix(line, "Valid from:") && !cur.HasDate:
			if i := strings.Index(line, "until:"); i >= 0 {
				raw := strings.TrimSpace(line[i+len("until:"):])
				if t, err := time.Parse(ksDateLayout, raw); err == nil {
					cur.Until, cur.HasDate = t, true
				}
			}
		}
	}
	flush()
	return entries
}

func ksListEntries(path, password, ksType, keytool string) ([]ksEntry, error) {
	args := []string{"-list", "-v", "-keystore", path}
	if ksType != "" {
		args = append(args, "-storetype", ksType)
	}

	out, err := runKeytool(keytool, password, args...)
	if err != nil {
		return nil, err
	}
	return ksParse(out), nil
}

// ---------- Registrierung ----------

func InitKeystoreFunctions() {
	if builtins == nil {
		builtins = make(map[string]BuiltinInfo)
	}

	ns := "keystore."

	Register(ns+"List", "keystore",
		"path, password, [onlyExpired], [type], [keytool]",
		"Gibt die Aliase eines Java-Keystores als Array zurück. Mit onlyExpired=True nur abgelaufene Einträge. type optional (JKS, PKCS12, JCEKS), sonst automatisch erkannt. Technische Fehler werden als Runtime Error gemeldet.",
		func(args []Value) Value {
			if len(args) < 2 {
				return ErrorVal("keystore.List: path und password benötigt")
			}

			path, err := ksAbsPath(ToString(args[0]))
			if err != nil {
				return ErrorVal("keystore.List: ungültiger Pfad: " + err.Error())
			}

			password := ToString(args[1])
			onlyExpired := len(args) >= 3 && isTruthy(args[2])

			ksType := ""
			if len(args) >= 4 {
				ksType = strings.TrimSpace(ToString(args[3]))
			}

			keytool := ""
			if len(args) >= 5 {
				keytool = strings.TrimSpace(ToString(args[4]))
			}

			entries, err := ksListEntries(path, password, ksType, keytool)
			if err != nil {
				return ErrorVal("keystore.List: " + err.Error())
			}

			now := time.Now()
			aliases := []Value{}
			for _, e := range entries {
				if onlyExpired && !e.Expired(now) {
					continue
				}
				aliases = append(aliases, StrVal(e.Alias))
			}
			return ArrVal(aliases)
		})

	Register(ns+"Find", "keystore",
		"path, password, alias, [type], [keytool]",
		"Prüft, ob ein Alias in einem Java-Keystore existiert. Gibt True zurück, wenn der Alias vorhanden ist, andernfalls False. Technische Fehler werden als Runtime Error gemeldet.",
		func(args []Value) Value {
			if len(args) < 3 {
				return ErrorVal("keystore.Find: path, password und alias benötigt")
			}

			path, err := ksAbsPath(ToString(args[0]))
			if err != nil {
				return ErrorVal("keystore.Find: ungültiger Pfad: " + err.Error())
			}

			password := ToString(args[1])
			alias := strings.TrimSpace(ToString(args[2]))

			if alias == "" {
				return ErrorVal("keystore.Find: Alias darf nicht leer sein")
			}

			ksType := ""
			if len(args) >= 4 {
				ksType = strings.TrimSpace(ToString(args[3]))
			}

			keytool := ""
			if len(args) >= 5 {
				keytool = strings.TrimSpace(ToString(args[4]))
			}

			cmdArgs := []string{
				"-list",
				"-alias", alias,
				"-keystore", path,
			}

			if ksType != "" {
				cmdArgs = append(cmdArgs, "-storetype", ksType)
			}

			_, err = runKeytool(keytool, password, cmdArgs...)
			if err == nil {
				return BoolVal(true)
			}

			if ksAliasMissing(err.Error(), alias) {
				return BoolVal(false)
			}

			return ErrorVal("keystore.Find: " + err.Error())
		})

	Register(ns+"Info", "keystore",
		"path, password, [type], [keytool]",
		"Gibt alle Einträge eines Java-Keystores als 2D-Array zurück: [Alias, Typ, Owner, GültigBis (UTC, yyyy-MM-dd HH:mm:ss, leer falls ohne Datum), Abgelaufen (Bool)].",
		func(args []Value) Value {
			if len(args) < 2 {
				return ErrorVal("keystore.Info: path und password benötigt")
			}

			path, err := ksAbsPath(ToString(args[0]))
			if err != nil {
				return ErrorVal("keystore.Info: ungültiger Pfad: " + err.Error())
			}

			password := ToString(args[1])

			ksType := ""
			if len(args) >= 3 {
				ksType = strings.TrimSpace(ToString(args[2]))
			}

			keytool := ""
			if len(args) >= 4 {
				keytool = strings.TrimSpace(ToString(args[3]))
			}

			entries, err := ksListEntries(path, password, ksType, keytool)
			if err != nil {
				return ErrorVal("keystore.Info: " + err.Error())
			}

			now := time.Now()
			rows := []Value{}
			for _, e := range entries {
				until := ""
				if e.HasDate {
					until = e.Until.Format("2006-01-02 15:04:05")
				}
				rows = append(rows, ArrVal([]Value{
					StrVal(e.Alias),
					StrVal(e.Type),
					StrVal(e.Owner),
					StrVal(until),
					BoolVal(e.Expired(now)),
				}))
			}
			return ArrVal(rows)
		})

	Register(ns+"Delete", "keystore",
		"path, password, alias, [type], [keytool]",
		"Löscht einen Alias aus einem Java-Keystore. Rückgabe: Array {Boolean, Meldung}. True = erfolgreich gelöscht, False = Alias nicht gefunden. Technische Fehler werden als Runtime Error gemeldet.",
		func(args []Value) Value {
			if len(args) < 3 {
				return ErrorVal("keystore.Delete: path, password und alias benötigt")
			}

			path, err := ksAbsPath(ToString(args[0]))
			if err != nil {
				return ErrorVal("keystore.Delete: ungültiger Pfad: " + err.Error())
			}

			password := ToString(args[1])
			alias := strings.TrimSpace(ToString(args[2]))

			if alias == "" {
				return ErrorVal("keystore.Delete: Alias darf nicht leer sein")
			}

			ksType := ""
			if len(args) >= 4 {
				ksType = strings.TrimSpace(ToString(args[3]))
			}

			keytool := ""
			if len(args) >= 5 {
				keytool = strings.TrimSpace(ToString(args[4]))
			}

			cmdArgs := []string{
				"-delete",
				"-alias", alias,
				"-keystore", path,
			}

			if ksType != "" {
				cmdArgs = append(cmdArgs, "-storetype", ksType)
			}

			_, err = runKeytool(keytool, password, cmdArgs...)
			if err != nil {
				// Ein fehlender Alias ist kein technischer Fehler.
				if ksAliasMissing(err.Error(), alias) {
					return ArrVal([]Value{
						BoolVal(false),
						StrVal("Eintrag konnte nicht gefunden werden"),
					})
				}

				return ErrorVal("keystore.Delete: " + err.Error())
			}

			return ArrVal([]Value{
				BoolVal(true),
				StrVal("Erfolgreich gelöscht"),
			})
		})
}
