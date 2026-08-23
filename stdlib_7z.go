// ------------------------
// stdlib_7z.go
// ------------------------

package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/bodgit/sevenzip"
)

// =============================================================================
// STATE: konfigurierbarer 7z-Binary-Pfad (für portable Installationen)
// =============================================================================

var (
	custom7zPathMu sync.RWMutex
	custom7zPath   string
)

// =============================================================================
// HELPER-FUNKTIONEN
// =============================================================================

// open7z öffnet ein 7z-Archiv, mit oder ohne Passwort.
func open7z(absPath, password string) (*sevenzip.ReadCloser, error) {
	if password != "" {
		return sevenzip.OpenReaderWithPassword(absPath, password)
	}
	return sevenzip.OpenReader(absPath)
}

// open7zFriendly öffnet ein 7z-Archiv wie open7z, übersetzt Fehler aber
// in eine für Skript-Autoren verständliche Meldung (statt der rohen
// internen sevenzip-Fehlerkette).
func open7zFriendly(absPath, password string) (*sevenzip.ReadCloser, error) {
	r, err := open7z(absPath, password)
	if err != nil {
		if password == "" {
			return nil, fmt.Errorf("archiv konnte nicht geöffnet werden (möglicherweise passwortgeschützt): %w", err)
		}
		return nil, fmt.Errorf("archiv konnte nicht geöffnet werden (falsches passwort oder beschädigtes archiv): %w", err)
	}
	return r, nil
}

// find7zBinary sucht das 7z-Kommandozeilentool in dieser Reihenfolge:
//  1. explizit über 7z.SetBinaryPath gesetzter Pfad
//  2. Umgebungsvariable VBX_7Z_PATH
//  3. System-PATH (7z, 7zz, 7za)
//  4. bekannte Standard-Installationsorte je Betriebssystem
func find7zBinary() (string, error) {
	custom7zPathMu.RLock()
	configured := custom7zPath
	custom7zPathMu.RUnlock()

	if configured != "" {
		if _, err := os.Stat(configured); err == nil {
			return configured, nil
		}
		return "", fmt.Errorf("konfigurierter 7z-Pfad '%s' existiert nicht", configured)
	}

	if envPath := os.Getenv("VBX_7Z_PATH"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath, nil
		}
	}

	for _, name := range []string{"7z", "7zz", "7za"} {
		if p, err := exec.LookPath(name); err == nil {
			return p, nil
		}
	}

	var candidates []string
	switch runtime.GOOS {
	case "windows":
		candidates = []string{
			`C:\Program Files\7-Zip\7z.exe`,
			`C:\Program Files (x86)\7-Zip\7z.exe`,
		}
	default:
		candidates = []string{
			"/usr/bin/7z", "/usr/local/bin/7z",
			"/usr/bin/7zz", "/usr/local/bin/7zz",
			"/opt/homebrew/bin/7zz",
		}
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}

	return "", fmt.Errorf("7z-Binary nicht gefunden (weder im PATH noch an Standardorten) — Pfad ggf. über 7z.SetBinaryPath() setzen")
}

// extract7zTo entpackt einen geöffneten Reader nach absDest, mit Zip-Slip-Schutz.
func extract7zTo(r *sevenzip.ReadCloser, absDest string) error {
	buf := make([]byte, 1024*1024)
	destPrefix := filepath.Clean(absDest) + string(os.PathSeparator)

	for _, f := range r.File {
		targetPath := filepath.Join(absDest, f.Name)

		if !strings.HasPrefix(filepath.Clean(targetPath)+string(os.PathSeparator), destPrefix) {
			return fmt.Errorf("zip-slip erkannt: '%s' liegt außerhalb des zielordners", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, f.Mode()); err != nil {
				return fmt.Errorf("verzeichnis '%s' konnte nicht erstellt werden: %w", targetPath, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("elternverzeichnis für '%s' konnte nicht erstellt werden: %w", targetPath, err)
		}

		outFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return fmt.Errorf("datei '%s' konnte nicht erstellt werden: %w", targetPath, err)
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return fmt.Errorf("eintrag '%s' konnte nicht geöffnet werden: %w", f.Name, err)
		}

		_, copyErr := io.CopyBuffer(outFile, rc, buf)
		outFile.Close()
		rc.Close()

		if copyErr != nil {
			return fmt.Errorf("entpacken von '%s' fehlgeschlagen: %w", f.Name, copyErr)
		}
	}

	return nil
}

// =============================================================================
// REGISTRIERUNG
// =============================================================================

func InitSevenZipFunctions() {
	ns := "7z."

	// ---------------------------------------------------------------------------
	// 7z.SetBinaryPath(path)  →  bool
	// ---------------------------------------------------------------------------
	Register(ns+"SetBinaryPath", "7z", "path",
		"Legt den Pfad zu einer 7z.exe/7z-Binary explizit fest (z.B. für portable Installationen).",
		func(args []Value) Value {
			if len(args) < 1 {
				return ErrorVal("usage: 7z.SetBinaryPath(path)")
			}

			absP, err := absPathStrict(args[0].Str)
			if err != nil {
				return ErrorVal(err.Error())
			}

			if _, err := os.Stat(absP); err != nil {
				return ErrorVal(fmt.Sprintf("datei '%s' nicht gefunden", absP))
			}

			custom7zPathMu.Lock()
			custom7zPath = absP
			custom7zPathMu.Unlock()

			return BoolVal(true)
		})

	// ---------------------------------------------------------------------------
	// 7z.Extract(archivePath, dest [, password])  →  null | error
	// ---------------------------------------------------------------------------
	Register(ns+"Extract", "7z", "archivePath, dest [, password]",
		"Entpackt ein 7z-Archiv nativ (ohne externes 7z benötigt). Schützt gegen Zip-Slip durch Pfad-Validierung.",
		func(args []Value) Value {
			if len(args) < 2 {
				return ErrorVal("usage: 7z.Extract(archivePath, dest [, password])")
			}

			absArchive, err := absPathStrict(args[0].Str)
			if err != nil {
				return ErrorVal(err.Error())
			}

			absDest, err := absPathStrict(args[1].Str)
			if err != nil {
				return ErrorVal(err.Error())
			}

			password := ""
			if len(args) >= 3 {
				password = args[2].Str
			}

			r, err := open7zFriendly(absArchive, password)
			if err != nil {
				return ErrorVal(err.Error())
			}
			defer r.Close()

			if err := extract7zTo(r, absDest); err != nil {
				return ErrorVal(err.Error())
			}

			return NullVal()
		})

	// ---------------------------------------------------------------------------
	// 7z.List(archivePath [, password])  →  Array von Maps {Name, Size, IsDir, ModTime}
	// ---------------------------------------------------------------------------
	Register(ns+"List", "7z", "archivePath [, password]",
		"Gibt Details über den Inhalt eines 7z-Archivs zurück.",
		func(args []Value) Value {
			if len(args) < 1 {
				return ErrorVal("usage: 7z.List(archivePath [, password])")
			}

			absP, err := absPathStrict(args[0].Str)
			if err != nil {
				return ErrorVal(err.Error())
			}

			password := ""
			if len(args) >= 2 {
				password = args[1].Str
			}

			r, err := open7zFriendly(absP, password)
			if err != nil {
				return ErrorVal(err.Error())
			}
			defer r.Close()

			results := make([]Value, 0, len(r.File))
			for _, f := range r.File {
				entry := map[string]Value{
					"Name":    StrVal(f.Name),
					"Size":    NumVal(float64(f.FileInfo().Size())),
					"IsDir":   BoolVal(f.FileInfo().IsDir()),
					"ModTime": StrVal(f.FileInfo().ModTime().Format("2006-01-02 15:04:05")),
				}
				results = append(results, Value{Kind: KindMap, Map: entry})
			}

			return Value{Kind: KindArr, Arr: results}
		})

	// ---------------------------------------------------------------------------
	// 7z.ListNames(archivePath [, password])  →  Array von Strings
	// ---------------------------------------------------------------------------
	Register(ns+"ListNames", "7z", "archivePath [, password]",
		"Gibt ein Array mit allen Dateinamen im 7z-Archiv zurück.",
		func(args []Value) Value {
			if len(args) < 1 {
				return ErrorVal("usage: 7z.ListNames(archivePath [, password])")
			}

			absP, err := absPathStrict(args[0].Str)
			if err != nil {
				return ErrorVal(err.Error())
			}

			password := ""
			if len(args) >= 2 {
				password = args[1].Str
			}

			r, err := open7zFriendly(absP, password)
			if err != nil {
				return ErrorVal(err.Error())
			}
			defer r.Close()

			names := make([]Value, 0, len(r.File))
			for _, f := range r.File {
				names = append(names, StrVal(f.Name))
			}

			return Value{Kind: KindArr, Arr: names}
		})

	// ---------------------------------------------------------------------------
	// 7z.Create(archivePath, files... [, password])  →  bool
	// Nutzt das externe 7z-Binary (kein pure-Go-Encoder verfügbar).
	// ---------------------------------------------------------------------------
	Register(ns+"Create", "7z", "archivePath, files... [, password]",
		"Erstellt ein 7z-Archiv über eine 7z-Binary (System-PATH, Standardpfad oder via 7z.SetBinaryPath gesetzt).",
		func(args []Value) Value {
			if len(args) < 2 {
				return ErrorVal("usage: 7z.Create(archivePath, files... [, password])")
			}

			bin, err := find7zBinary()
			if err != nil {
				return ErrorVal(err.Error())
			}

			files, pass := extractFilesAndPass(args)
			if len(files) == 0 {
				return ErrorVal("keine quelldateien angegeben")
			}

			absArchive, err := absPathStrict(args[0].Str)
			if err != nil {
				return ErrorVal(err.Error())
			}

			cmdArgs := []string{"a"}
			if pass != "" {
				cmdArgs = append(cmdArgs, "-p"+pass, "-mhe=on")
			}
			cmdArgs = append(cmdArgs, absArchive)
			cmdArgs = append(cmdArgs, files...)

			cmd := exec.Command(bin, cmdArgs...)
			out, err := cmd.CombinedOutput()
			if err != nil {
				return ErrorVal(fmt.Sprintf("7z-fehler: %v (%s)", err, strings.TrimSpace(string(out))))
			}

			return BoolVal(true)
		})
}
