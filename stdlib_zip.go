// ------------------------
// stdlib_zip.go
// ------------------------

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	zip "github.com/alexmullins/zip"
)

// =============================================================================
// HELPER-FUNKTIONEN
// =============================================================================

// absPathStrict ist wie filepath.Abs, gibt aber einen expliziten Fehler zurück
// statt ihn stillschweigend zu verwerfen.
func absPathStrict(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", fmt.Errorf("pfad '%s' konnte nicht aufgelöst werden: %w", p, err)
	}
	return abs, nil
}

// commonBasePath ermittelt den kleinsten gemeinsamen Basisordner einer Pfadliste.
func commonBasePath(paths []string) string {
	if len(paths) == 0 {
		return ""
	}

	cleaned := make([]string, len(paths))
	for i, p := range paths {
		c := filepath.Clean(p)
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			c = filepath.Dir(c)
		}
		cleaned[i] = c
	}

	if len(cleaned) == 1 {
		return cleaned[0]
	}

	base := strings.Split(cleaned[0], string(os.PathSeparator))

	for _, p := range cleaned[1:] {
		curr := strings.Split(p, string(os.PathSeparator))
		limit := len(base)
		if len(curr) < limit {
			limit = len(curr)
		}
		cutAt := limit
		for i := 0; i < limit; i++ {
			if base[i] != curr[i] {
				cutAt = i
				break
			}
		}
		base = base[:cutAt]
	}

	result := strings.Join(base, string(os.PathSeparator))
	if result == "" {
		return string(os.PathSeparator)
	}
	return result
}

// addFileToZip fügt eine einzelne Datei zum ZIP-Writer hinzu.
// Bei gesetztem Passwort wird Deflate-Komprimierung explizit aktiviert.
// Korrigiert den shadowed-err-Bug der Originalversion.
func addFileToZip(zw *zip.Writer, absPath, zipEntryPath, password string, buf []byte) error {
	zipEntryPath = filepath.ToSlash(zipEntryPath)

	file, err := os.Open(absPath)
	if err != nil {
		return fmt.Errorf("datei '%s' nicht lesbar: %w", absPath, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat '%s' fehlgeschlagen: %w", absPath, err)
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return fmt.Errorf("header für '%s' fehlgeschlagen: %w", absPath, err)
	}

	header.Name = zipEntryPath
	header.Method = zip.Deflate // Immer Deflate — auch bei Verschlüsselung

	var fw io.Writer
	if password != "" {
		fw, err = zw.Encrypt(zipEntryPath, password)
	} else {
		fw, err = zw.CreateHeader(header)
	}

	if err != nil {
		return fmt.Errorf("zip-entry für '%s' konnte nicht erstellt werden: %w", zipEntryPath, err)
	}

	if _, err := io.CopyBuffer(fw, file, buf); err != nil {
		return fmt.Errorf("kopieren von '%s' fehlgeschlagen: %w", absPath, err)
	}

	return nil
}

// zipCreate erstellt ein ZIP-Archiv aus einer Liste von Pfaden.
// flat=true legt alle Dateien ohne Verzeichnisstruktur ab.
// Schreibt atomar: erst in eine .tmp-Datei, dann umbenennen.
func zipCreate(zipPath string, paths []string, password string, flat bool) (int, error) {
	if len(paths) == 0 {
		return 0, fmt.Errorf("keine quelldateien angegeben")
	}

	absZipPath, err := absPathStrict(zipPath)
	if err != nil {
		return 0, err
	}

	tmpPath := absZipPath + ".tmp"

	zf, err := os.Create(tmpPath)
	if err != nil {
		return 0, fmt.Errorf("archiv konnte nicht erstellt werden: %w", err)
	}

	zipWriter := zip.NewWriter(zf)
	buf := make([]byte, 1024*1024)
	fileCount := 0

	baseFolder := ""
	if !flat {
		absPaths := make([]string, 0, len(paths))
		for _, p := range paths {
			if abs, err := absPathStrict(p); err == nil {
				absPaths = append(absPaths, abs)
			}
		}
		baseFolder = commonBasePath(absPaths)
	}

	for _, p := range paths {
		absP, err := absPathStrict(p)
		if err != nil {
			fmt.Printf("[ZIP-Warnung]: Überspringe '%s': %v\n", p, err)
			continue
		}

		info, err := os.Stat(absP)
		if err != nil {
			fmt.Printf("[ZIP-Warnung]: Überspringe '%s': %v\n", p, err)
			continue
		}

		if info.IsDir() {
			filepath.WalkDir(absP, func(path string, d os.DirEntry, walkErr error) error {
				if walkErr != nil {
					fmt.Printf("[ZIP-Warnung]: Überspringe '%s': %v\n", path, walkErr)
					return nil
				}
				if d.IsDir() {
					return nil
				}

				entryPath := filepath.Base(path)
				if !flat {
					if rel, err := filepath.Rel(baseFolder, path); err == nil {
						entryPath = rel
					}
				}

				if err := addFileToZip(zipWriter, path, entryPath, password, buf); err != nil {
					fmt.Printf("[ZIP-Warnung]: Überspringe '%s': %v\n", path, err)
				} else {
					fileCount++
				}
				return nil
			})
		} else {
			entryPath := filepath.Base(absP)
			if !flat {
				if rel, err := filepath.Rel(baseFolder, absP); err == nil {
					entryPath = rel
				}
			}

			if err := addFileToZip(zipWriter, absP, entryPath, password, buf); err != nil {
				fmt.Printf("[ZIP-Warnung]: Überspringe '%s': %v\n", absP, err)
			} else {
				fileCount++
			}
		}
	}

	zipWriter.Close()
	zf.Close()

	if fileCount == 0 {
		os.Remove(tmpPath)
		return 0, fmt.Errorf("archiv leer: keine validen dateien gefunden")
	}

	// Atomar umbenennen
	if err := os.Rename(tmpPath, absZipPath); err != nil {
		os.Remove(tmpPath)
		return 0, fmt.Errorf("umbenennen fehlgeschlagen: %w", err)
	}

	return fileCount, nil
}

// zipExtract entpackt ein ZIP-Archiv in ein Zielverzeichnis.
// Schützt gegen Zip-Slip durch Präfix-Validierung.
func zipExtract(zipPath, dest, password string) error {
	absZip, err := absPathStrict(zipPath)
	if err != nil {
		return err
	}

	absDest, err := absPathStrict(dest)
	if err != nil {
		return err
	}

	r, err := zip.OpenReader(absZip)
	if err != nil {
		return fmt.Errorf("archiv '%s' konnte nicht geöffnet werden: %w", absZip, err)
	}
	defer r.Close()

	buf := make([]byte, 1024*1024)
	destPrefix := filepath.Clean(absDest) + string(os.PathSeparator)

	for _, f := range r.File {
		if password != "" {
			f.SetPassword(password)
		}

		targetPath := filepath.Join(absDest, f.Name)

		// Zip-Slip-Schutz
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

// extractFilesAndPass trennt Dateipfade und optionales Passwort aus den Argumenten.
// Unterstützt zwei Aufrufkonventionen:
//   - (zipPath, []array, [pass])
//   - (zipPath, file1, file2, ..., [pass])
func extractFilesAndPass(args []Value) ([]string, string) {
	var files []string
	pass := ""

	// Konvention A: zweites Argument ist ein Array
	if len(args) >= 2 && args[1].Kind == KindArr {
		for _, v := range args[1].Arr {
			files = append(files, v.Str)
		}
		if len(args) >= 3 {
			pass = args[2].Str
		}
		return files, pass
	}

	// Konvention B: variadische Dateipfade, letztes Argument optional Passwort
	lastIdx := len(args) - 1
	if lastIdx > 1 && !looksLikeFilePath(args[lastIdx].Str) {
		pass = args[lastIdx].Str
		lastIdx--
	}

	for i := 1; i <= lastIdx; i++ {
		files = append(files, args[i].Str)
	}
	return files, pass
}

// looksLikeFilePath erkennt ob ein String ein Dateipfad ist.
func looksLikeFilePath(s string) bool {
	return strings.ContainsAny(s, "/\\") ||
		strings.HasPrefix(s, ".") ||
		filepath.IsAbs(s)
}

// =============================================================================
// REGISTRIERUNG
// =============================================================================

func InitZipFunctions() {
	ns := "zip."

	// ---------------------------------------------------------------------------
	// zip.Create(zipPath, files... [, password])  →  bool
	// ---------------------------------------------------------------------------
	Register(ns+"Create", "zip", "zipPath, files... [, password]",
		"Erstellt ein ZIP-Archiv mit erhaltener Verzeichnisstruktur.",
		func(args []Value) Value {
			if len(args) < 2 {
				return ErrorVal("usage: zip.Create(zipPath, files... [, password])")
			}

			files, pass := extractFilesAndPass(args)
			count, err := zipCreate(args[0].Str, files, pass, false)
			if err != nil {
				return ErrorVal(err.Error())
			}

			return BoolVal(count > 0)
		})

	// ---------------------------------------------------------------------------
	// zip.CreateFlat(zipPath, files... [, password])  →  bool
	// ---------------------------------------------------------------------------
	Register(ns+"CreateFlat", "zip", "zipPath, files... [, password]",
		"Erstellt ein ZIP-Archiv ohne Unterordner (alle Dateien auf oberster Ebene).",
		func(args []Value) Value {
			if len(args) < 2 {
				return ErrorVal("usage: zip.CreateFlat(zipPath, files... [, password])")
			}

			files, pass := extractFilesAndPass(args)
			count, err := zipCreate(args[0].Str, files, pass, true)
			if err != nil {
				return ErrorVal(err.Error())
			}

			return BoolVal(count > 0)
		})

	// ---------------------------------------------------------------------------
	// zip.Extract(zipPath, dest [, password])  →  null | error
	// ---------------------------------------------------------------------------
	Register(ns+"Extract", "zip", "zipPath, dest [, password]",
		"Entpackt ein Archiv. Schützt gegen Zip-Slip durch Pfad-Validierung.",
		func(args []Value) Value {
			if len(args) < 2 {
				return ErrorVal("usage: zip.Extract(zipPath, dest [, password])")
			}

			pass := ""
			if len(args) >= 3 {
				pass = args[2].Str
			}

			if err := zipExtract(args[0].Str, args[1].Str, pass); err != nil {
				return ErrorVal(err.Error())
			}

			return NullVal()
		})

	// ---------------------------------------------------------------------------
	// zip.List(zipPath)  →  Array von Maps {Name, Size, IsDir, ModTime}
	// ---------------------------------------------------------------------------
	Register(ns+"List", "zip", "zipPath",
		"Gibt Details über den Inhalt eines ZIP-Archivs zurück.",
		func(args []Value) Value {
			if len(args) < 1 {
				return ErrorVal("usage: zip.List(zipPath)")
			}

			absP, err := absPathStrict(args[0].Str)
			if err != nil {
				return ErrorVal(err.Error())
			}

			r, err := zip.OpenReader(absP)
			if err != nil {
				return ErrorVal("archiv konnte nicht geöffnet werden: " + err.Error())
			}
			defer r.Close()

			results := make([]Value, 0, len(r.File))
			for _, f := range r.File {
				entry := map[string]Value{
					"Name":    StrVal(f.Name),
					"Size":    NumVal(float64(f.UncompressedSize64)),
					"IsDir":   BoolVal(f.FileInfo().IsDir()),
					"ModTime": StrVal(f.FileInfo().ModTime().Format("2006-01-02 15:04:05")),
				}
				results = append(results, Value{Kind: KindMap, Map: entry})
			}

			return Value{Kind: KindArr, Arr: results}
		})

	// ---------------------------------------------------------------------------
	// zip.ListNames(zipPath)  →  Array von Strings
	// ---------------------------------------------------------------------------
	Register(ns+"ListNames", "zip", "zipPath",
		"Gibt ein Array mit allen Dateinamen im ZIP-Archiv zurück.",
		func(args []Value) Value {
			if len(args) < 1 {
				return ErrorVal("usage: zip.ListNames(zipPath)")
			}

			absP, err := absPathStrict(args[0].Str)
			if err != nil {
				return ErrorVal(err.Error())
			}

			r, err := zip.OpenReader(absP)
			if err != nil {
				return ErrorVal("archiv konnte nicht geöffnet werden: " + err.Error())
			}
			defer r.Close()

			names := make([]Value, 0, len(r.File))
			for _, f := range r.File {
				names = append(names, StrVal(f.Name))
			}

			return Value{Kind: KindArr, Arr: names}
		})
}
