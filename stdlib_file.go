// ------------------------
// stdlib_file.go
// ------------------------

package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// InitFileFunctions registriert File-Funktionen inkl. erweiterter Features
func InitFileFunctions() {
	if builtins == nil {
		builtins = make(map[string]BuiltinInfo)
	}

	ns := "file."

	// ------------------------
	// Write
	// ------------------------
	Register(ns+"WriteAllText", "file", "path, content", "Schreibt den gesamten Text in eine Datei (überschreibt Vorhandenes).", func(args []Value) Value {
		// 1. Parameter-Check: Wir brauchen Pfad und Inhalt
		if len(args) < 2 {
			return ErrorVal("WriteAllText erwartet zwei Parameter: Pfad und Inhalt")
		}

		// 2. Deine Sicherheitsfunktion nutzen!
		// Sie liefert den absoluten Pfad und einen Zeiger auf einen Error-Value
		path, errVal := absPathVal(args[0].Str)
		if errVal != nil {
			return *errVal // Fehler direkt an VB zurückgeben (z.B. Sicherheitsfehler)
		}

		// 3. Datei schreiben
		// Wir nutzen den von absPathVal gelieferten 'path'
		err := os.WriteFile(path, []byte(args[1].Str), 0644)
		if err != nil {
			return ErrorVal("Schreibfehler in WriteAllText: " + err.Error())
		}

		return NullVal()
	})

	// ------------------------
	// Create
	// ------------------------
	Register(ns+"Create", "file", "path", "Erstellt eine leere Datei, falls diese noch nicht existiert.", func(args []Value) Value {
		if len(args) < 1 {
			return ErrorVal("file.Create benötigt Pfad")
		}

		path, errVal := absPathVal(args[0].Str)
		if errVal != nil {
			return *errVal
		}

		// Verzeichnis muss existieren
		if _, err := os.Stat(filepath.Dir(path)); err != nil {
			return ErrorVal("Verzeichnis existiert nicht: " + filepath.Dir(path))
		}

		// Nur erstellen wenn sie noch nicht existiert
		if _, err := os.Stat(path); os.IsNotExist(err) {
			f, err := os.Create(path)
			if err != nil {
				return ErrorVal("Datei konnte nicht erstellt werden: " + err.Error())
			}
			f.Close()
		}

		return NullVal()
	})

	// ------------------------
	// StreamWrite
	// ------------------------
	Register(ns+"StreamWrite", "file", "path, content", "Hängt Text gepuffert an eine Datei an (erstellt diese, falls nicht vorhanden).", func(args []Value) Value {
		if len(args) < 2 {
			return ErrorVal("file.StreamWrite(path, content) benötigt 2 Argumente")
		}

		path, errVal := absPathVal(args[0].Str)
		if errVal != nil {
			return *errVal
		}

		// Verzeichnis muss existieren
		if _, err := os.Stat(filepath.Dir(path)); err != nil {
			return ErrorVal("Verzeichnis existiert nicht: " + filepath.Dir(path))
		}

		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return ErrorVal("Stream konnte nicht geöffnet werden: " + err.Error())
		}
		defer f.Close()

		writer := bufio.NewWriter(f)
		if _, err := writer.WriteString(args[1].Str); err != nil {
			return ErrorVal("Fehler beim Schreiben in den Stream: " + err.Error())
		}

		if err := writer.Flush(); err != nil {
			return ErrorVal("Fehler beim Flushen des Buffers: " + err.Error())
		}

		return NullVal()
	})

	// ---------------- Exists ----------------
	Register(ns+"Exists", "path", "bool",
		"Prüft, ob eine Datei existiert.",
		func(args []Value) Value {

			if len(args) < 1 || args[0].Str == "" {
				return BoolVal(false)
			}

			path, errVal := absPathVal(args[0].Str)
			if errVal != nil {
				return BoolVal(false)
			}

			info, err := os.Stat(path)
			if err != nil {
				return BoolVal(false)
			}

			return BoolVal(!info.IsDir())
		})

	// --- Namespace: file. ---
	Register(ns+"CreateSymlink", "file", "target, linkPath, bool", "Erstellt einen Symlink auf eine DATEI.", func(args []Value) Value {
		if len(args) < 2 {
			return ErrorVal("file.CreateSymlink(target, linkPath) benötigt 2 Pfade")
		}
		return createSymlinkInternal(args[0].Str, args[1].Str, false)
	})

	// ---------------- Base64Encode ----------------
	Register(ns+"Base64Encode", "file", "inFile, outFile", "Liest eine Datei ein und speichert sie Base64-kodiert ab.", func(args []Value) Value {
		if len(args) < 2 {
			return ErrorVal("file.Base64Encode(in, out)")
		}
		inFile, e1 := absPathVal(args[0].Str)
		outFile, e2 := absPathVal(args[1].Str)
		if e1 != nil {
			return *e1
		}
		if e2 != nil {
			return *e2
		}

		in, err := os.Open(inFile)
		if err != nil {
			return ErrorVal(err.Error())
		}
		defer in.Close()
		out, err := os.Create(outFile)
		if err != nil {
			return ErrorVal(err.Error())
		}
		defer out.Close()

		enc := base64.NewEncoder(base64.StdEncoding, out)
		io.Copy(enc, in)
		enc.Close()
		return NullVal()
	})

	// ---------------- Base64Decode ----------------
	Register(ns+"Base64Decode", "file", "inFile, outFile", "Dekodiert eine Base64-Datei und speichert das Ergebnis als Binärdatei.", func(args []Value) Value {
		if len(args) < 2 {
			return ErrorVal("file.Base64Decode(infile, outfile) benötigt 2 Argumente")
		}

		inFile, e1 := absPathVal(args[0].Str)
		if e1 != nil {
			return *e1
		}

		outFile, e2 := absPathVal(args[1].Str)
		if e2 != nil {
			return *e2
		}

		// Zielverzeichnis muss existieren
		if _, err := os.Stat(filepath.Dir(outFile)); err != nil {
			return ErrorVal("Zielverzeichnis existiert nicht: " + filepath.Dir(outFile))
		}

		in, err := os.Open(inFile)
		if err != nil {
			return ErrorVal("Fehler beim Öffnen der Quelldatei: " + err.Error())
		}
		defer in.Close()

		out, err := os.Create(outFile)
		if err != nil {
			return ErrorVal("Zieldatei konnte nicht erstellt werden: " + err.Error())
		}
		defer out.Close()

		dec := base64.NewDecoder(base64.StdEncoding, in)
		if _, err := io.Copy(out, dec); err != nil {
			return ErrorVal("Dekodierungsfehler: " + err.Error())
		}

		return NullVal()
	})

	// ---------------- Delete ----------------
	Register(ns+"Delete", "file",
		"path",
		"Löscht eine Datei.",
		func(args []Value) Value {

			// -------------------------
			// Parametercheck
			// -------------------------
			if len(args) < 1 || args[0].Str == "" {
				return fileResult(false, "file.Delete: Pfad fehlt")
			}

			// -------------------------
			// Pfad absichern
			// -------------------------
			path, errVal := absPathVal(args[0].Str)
			if errVal != nil {
				return fileResult(false, "file.Delete: ungültiger Pfad")
			}

			// -------------------------
			// Datei löschen
			// -------------------------
			err := os.Remove(path)
			if err != nil {

				// "nicht existiert" ist kein echter Fehler
				if os.IsNotExist(err) {
					return fileResult(true, "")
				}

				return fileResult(false,
					"file.Delete: "+err.Error()+
						" path="+path,
				)
			}

			// -------------------------
			// Erfolg
			// -------------------------
			return fileResult(true, "")
		})

	// ---------------- Copy ----------------
	Register(ns+"Copy", "file",
		"src, dst",
		"Kopiert eine Datei (kein Überschreiben ohne Fehler).",
		func(args []Value) Value {

			if len(args) < 2 {
				return fileResult(false, "file.Copy: benötigt src und dst")
			}

			src, e1 := absPathVal(args[0].Str)
			if e1 != nil {
				return fileResult(false, "file.Copy: ungültiger Quellpfad")
			}

			dst, e2 := absPathVal(args[1].Str)
			if e2 != nil {
				return fileResult(false, "file.Copy: ungültiger Zielpfad")
			}

			if _, err := os.Stat(src); err != nil {
				return fileResult(false, "file.Copy: Quelle fehlt: "+src)
			}

			if _, err := os.Stat(filepath.Dir(dst)); err != nil {
				return fileResult(false, "file.Copy: Zielverzeichnis fehlt: "+filepath.Dir(dst))
			}

			if _, err := os.Stat(dst); err == nil {
				return fileResult(false, "file.Copy: Ziel existiert bereits: "+dst)
			}

			if err := copyFile(src, dst); err != nil {
				return fileResult(false, "file.Copy: "+err.Error())
			}

			return fileResult(true)
		})

	// ---------------- Move ----------------
	Register(ns+"Move", "file",
		"src, dst",
		"Verschiebt oder benennt eine Datei um (Rename + Cross-Drive Fallback).",
		func(args []Value) Value {

			// -------------------------
			// Parametercheck
			// -------------------------
			if len(args) < 2 {
				return fileResult(false, "file.Move: benötigt src und dst")
			}

			// -------------------------
			// Pfade absichern
			// -------------------------
			src, e1 := absPathVal(args[0].Str)
			if e1 != nil {
				return fileResult(false, "file.Move: ungültiger Quellpfad")
			}

			dst, e2 := absPathVal(args[1].Str)
			if e2 != nil {
				return fileResult(false, "file.Move: ungültiger Zielpfad")
			}

			// -------------------------
			// Quelle prüfen
			// -------------------------
			if _, err := os.Stat(src); err != nil {
				return fileResult(false, "file.Move: Quelle existiert nicht: "+src)
			}

			// -------------------------
			// Zielverzeichnis prüfen
			// -------------------------
			if _, err := os.Stat(filepath.Dir(dst)); err != nil {
				return fileResult(false, "file.Move: Zielverzeichnis fehlt: "+filepath.Dir(dst))
			}

			// -------------------------
			// Ziel existiert bereits
			// -------------------------
			if _, err := os.Stat(dst); err == nil {
				return fileResult(false, "file.Move: Zieldatei existiert bereits: "+dst)
			}

			// -------------------------
			// 1. Versuch: Rename (schnell, atomar)
			// -------------------------
			if err := os.Rename(src, dst); err == nil {
				return fileResult(true)
			}

			// -------------------------
			// 2. Fallback: Copy + Delete (Cross-Drive)
			// -------------------------
			if err := copyAndDelete(src, dst); err != nil {
				return fileResult(false,
					"file.Move: Cross-Drive-Fallback fehlgeschlagen: "+
						err.Error()+
						" src="+src+
						" dst="+dst,
				)
			}

			return fileResult(true)
		})

	// ------------------------
	// file.Compare (Optimiert)
	// ------------------------
	Register(ns+"Compare", "file", "pfad1, pfad2", "Vergleicht zwei Dateien auf Gleichheit.", func(args []Value) Value {
		if len(args) < 2 {
			return ErrorVal("file.Compare(path1, path2) benötigt 2 Pfade")
		}

		// Sicherheit: Falls Zahlen oder Null als Pfad übergeben werden
		p1Raw := ToString(args[0])
		p2Raw := ToString(args[1])

		p1, e1 := absPathVal(p1Raw)
		p2, e2 := absPathVal(p2Raw)
		if e1 != nil {
			return *e1
		}
		if e2 != nil {
			return *e2
		}

		// Performance-Check: Wenn es derselbe Pfad ist, sind sie identisch
		if p1 == p2 {
			return NumVal(0)
		}

		f1, err1 := os.Open(p1)
		if err1 != nil {
			return ErrorVal("Fehler beim Öffnen von Pfad 1: " + err1.Error())
		}
		defer f1.Close()

		f2, err2 := os.Open(p2)
		if err2 != nil {
			return ErrorVal("Fehler beim Öffnen von Pfad 2: " + err2.Error())
		}
		defer f2.Close()

		// Schneller Vorab-Check: Wenn die Dateigrößen unterschiedlich sind,
		// müssen wir gar nicht erst die Bytes lesen.
		stat1, _ := f1.Stat()
		stat2, _ := f2.Stat()
		if stat1.Size() != stat2.Size() {
			return NumVal(1) // Unterschiedlich
		}

		// Byte-für-Byte Vergleich mit Buffer (4KB ist meist optimal)
		buf1 := make([]byte, 4096)
		buf2 := make([]byte, 4096)

		for {
			n1, r1 := f1.Read(buf1)
			n2, r2 := f2.Read(buf2)

			// Vergleiche die Anzahl gelesener Bytes und den Inhalt
			// bytes.Equal ist deutlich schneller als ein String-Cast
			if n1 != n2 || !bytes.Equal(buf1[:n1], buf2[:n2]) {
				return NumVal(1) // Unterschiedlich
			}

			// Ende beider Dateien erreicht
			if r1 == io.EOF && r2 == io.EOF {
				break
			}

			// Falls ein anderer Fehler beim Lesen auftritt
			if r1 != nil && r1 != io.EOF {
				return ErrorVal("Lesefehler Datei 1: " + r1.Error())
			}
			if r2 != nil && r2 != io.EOF {
				return ErrorVal("Lesefehler Datei 2: " + r2.Error())
			}
		}

		return NumVal(0) // Identisch
	})

	// ---------------- Size ----------------
	Register(ns+"Size", "files", "pfad", "Gibt die Größe einer Datei in Bytes zurück.", func(args []Value) Value {
		if len(args) < 1 {
			return ErrorVal("file.Size benötigt Pfad")
		}
		path, errVal := absPathVal(args[0].Str)
		if errVal != nil {
			return *errVal
		}

		info, err := os.Stat(path)
		if err != nil {
			return ErrorVal("Größe konnte nicht ermittelt werden: " + err.Error())
		}

		size := float64(info.Size())
		if len(args) >= 2 && args[1].Str == "human" {
			return StrVal(formatSize(size)) // Hilfsfunktion nutzen
		}
		return NumVal(size)
	})

	// ---------------- ModTime ----------------
	Register(ns+"ModTime", "file", "path", "Gibt den Zeitpunkt der letzten Änderung im ISO-Format (RFC3339) zurück.", func(args []Value) Value {
		if len(args) < 1 {
			return ErrorVal("file.ModTime benötigt Pfad")
		}
		path, errVal := absPathVal(args[0].Str)
		if errVal != nil {
			return *errVal
		}
		info, err := os.Stat(path)
		if err != nil {
			return ErrorVal("Fehler beim Lesen der ModTime von '" + path + "': " + err.Error())
		}
		return StrVal(info.ModTime().Format(time.RFC3339))
	})

	// ---------------- CreateTime ----------------
	Register(ns+"CreateTime", "file", "path", "Gibt den Zeitpunkt der Erstellung im ISO-Format (RFC3339) zurück.", func(args []Value) Value {
		if len(args) < 1 {
			return ErrorVal("file.CreateTime benötigt Pfad")
		}
		path, errVal := absPathVal(args[0].Str)
		if errVal != nil {
			return *errVal
		}
		c, _, _, err := FileTimes(path)
		if err != nil {
			return ErrorVal("Fehler beim Lesen der CreateTime von '" + path + "': " + err.Error())
		}
		return StrVal(c.Format(time.RFC3339))
	})

	// ---------------- AccessTime ----------------
	Register(ns+"AccessTime", "file", "path", "Gibt den Zeitpunkt des letzten Zugriffs im ISO-Format (RFC3339) zurück.", func(args []Value) Value {
		if len(args) < 1 {
			return ErrorVal("file.AccessTime benötigt Pfad")
		}
		path, errVal := absPathVal(args[0].Str)
		if errVal != nil {
			return *errVal
		}
		_, a, _, err := FileTimes(path)
		if err != nil {
			return ErrorVal("Fehler beim Lesen der AccessTime von '" + path + "': " + err.Error())
		}
		return StrVal(a.Format(time.RFC3339))
	})

	// ---------------- Replace / ReplaceAll ----------------
	Register(ns+"Replace", "file", "text, alt, neu", "Ersetzt das erste Vorkommen von alt durch neu.", func(args []Value) Value {
		if len(args) < 3 {
			return ErrorVal("file.Replace(path, pattern, newContent) benötigt 3 Argumente")
		}

		// 1. Pfad-Sicherheitscheck (Gibt (string, *Value) zurück)
		path, errVal := absPathVal(args[0].Str)
		if errVal != nil {
			return *errVal
		}

		pattern := args[1].Str
		newContent := args[2].Str

		// 2. Datei einlesen
		data, err := os.ReadFile(path)
		if err != nil {
			return ErrorVal("Fehler beim Lesen: " + err.Error())
		}

		// 3. Zeilenweise verarbeiten
		// Wir normalisieren die Zeilenumbrüche für den Split
		content := strings.ReplaceAll(string(data), "\r\n", "\n")
		lines := strings.Split(content, "\n")

		found := false
		for i, line := range lines {
			if strings.Contains(line, pattern) {
				lines[i] = newContent
				found = true
				break // Nur das erste Vorkommen ersetzen
			}
		}

		// 4. Nur schreiben, wenn wirklich etwas gefunden wurde (optimiert die SSD)
		if found {
			err = os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
			if err != nil {
				return ErrorVal("Fehler beim Schreiben: " + err.Error())
			}
		}

		return NullVal()
	})

	Register(ns+"Search", "file", "pfad, muster", "Sucht nach einem Textmuster innerhalb einer Datei.", func(args []Value) Value {
		if len(args) < 2 {
			return ErrorVal("file.Search(path, pattern) benötigt Pfad und Suchmuster")
		}

		path, errVal := absPathVal(args[0].Str)
		if errVal != nil {
			return *errVal
		}
		pattern := args[1].Str

		// Nutzt unseren Shared-Open-Helper (Windows-sicher!)
		file, err := openFileShared(path)
		if err != nil {
			return ErrorVal("Datei konnte nicht geöffnet werden: " + err.Error())
		}
		defer file.Close()

		var results []string
		scanner := bufio.NewScanner(file)

		// Puffer auf 1MB erhöhen für extrem lange Log-Zeilen (z.B. Stacktraces)
		const maxLogLineSize = 1024 * 1024
		buf := make([]byte, 64*1024)
		scanner.Buffer(buf, maxLogLineSize)

		for scanner.Scan() {
			line := scanner.Text()
			// Tipp: In BASIC ist es oft hilfreich, Case-Insensitive zu suchen
			if strings.Contains(strings.ToLower(line), strings.ToLower(pattern)) {
				results = append(results, line)
			}
		}

		if err := scanner.Err(); err != nil {
			return ErrorVal("Fehler beim Scannen (Zeile zu lang?): " + err.Error())
		}

		return Value{Kind: KindArr, Arr: stringSliceToValueSlice(results)}
	})

	Register(ns+"SearchDateRange", "file", "pfad, von, bis", "Findet Dateien in einem Zeitraum (YYYY-MM-DD).", func(args []Value) Value {
		if len(args) < 3 {
			return ErrorVal("SearchDateRange(path, start, end, [layout], [max]) fehlt Argument")
		}

		path, errVal := absPathVal(args[0].Str)
		if errVal != nil {
			return *errVal
		}
		startStr := args[1].Str
		endStr := args[2].Str

		// Benutzerfreundliches Format (Default: YYYY-MM-DD HH:mm:SS)
		userLayout := "YYYY-MM-DD HH:mm:SS"
		if len(args) >= 4 && args[3].Str != "" {
			userLayout = args[3].Str
		}

		// INTERNE KONVERTIERUNG zu Go-Standard (2006-01-02...)
		layout := mapUserLayoutToGo(userLayout)

		maxResults := 5000
		if len(args) >= 5 {
			maxResults = int(toNumVal(args[4]))
		}

		// Ab hier bleibt die Logik gleich...
		startTime, errS := time.Parse(layout, startStr)
		endTime, errE := time.Parse(layout, endStr)
		if errS != nil || errE != nil {
			return ErrorVal("Zeit-Parsing Fehler: Passt das Datum zum Format " + userLayout + "?")
		}

		// 3. Shared Open (Windows-sicher)
		file, err := openFileShared(path)
		if err != nil {
			return ErrorVal("Datei-Fehler: " + err.Error())
		}
		defer file.Close()

		var results []string
		scanner := bufio.NewScanner(file)
		buf := make([]byte, 64*1024)
		scanner.Buffer(buf, 1024*1024) // 1MB Zeilen-Limit

		isInRange := false
		layoutLen := len(layout)

		for scanner.Scan() {
			// Abbruch, wenn Limit erreicht
			if len(results) >= maxResults {
				break
			}

			line := scanner.Text()

			// Datumserkennung am Zeilenanfang
			if len(line) >= layoutLen {
				lineDateStr := line[:layoutLen]
				if lineTime, err := time.Parse(layout, lineDateStr); err == nil {
					// Neues Datum gefunden -> Bereich neu prüfen
					isInRange = (lineTime.After(startTime) || lineTime.Equal(startTime)) &&
						(lineTime.Before(endTime) || lineTime.Equal(endTime))
				}
			}

			// Zeile hinzufügen, wenn wir im Bereich sind (auch Folgezeilen ohne Datum)
			if isInRange {
				results = append(results, line)
			}
		}

		if err := scanner.Err(); err != nil {
			return ErrorVal("Scan-Fehler: " + err.Error())
		}

		return Value{Kind: KindArr, Arr: stringSliceToValueSlice(results)}
	})

	Register(ns+"ReplaceAll", "file", "text, alt, neu", "Ersetzt alle Vorkommen von alt durch neu.", func(args []Value) Value {
		if len(args) < 3 {
			return ErrorVal("file.ReplaceAll(path, old, new) benötigt 3 Argumente")
		}

		// 1. Pfad validieren (Gibt path und *Value zurück)
		path, errVal := absPathVal(args[0].Str)
		if errVal != nil {
			return *errVal // Sicherheits-Check
		}

		oldStr := args[1].Str
		newStr := args[2].Str

		// 2. Datei komplett einlesen
		data, err := os.ReadFile(path)
		if err != nil {
			return ErrorVal("Fehler beim Lesen der Datei: " + err.Error())
		}

		// 3. Ersetzung im Speicher durchführen
		content := strings.ReplaceAll(string(data), oldStr, newStr)

		// 4. Datei überschreiben
		// Wir nutzen 0644 (Standard-Rechte), falls die Datei neu erstellt wird
		err = os.WriteFile(path, []byte(content), 0644)
		if err != nil {
			return ErrorVal("Fehler beim Speichern der Änderungen: " + err.Error())
		}

		return NullVal() // Erfolg
	})

	// ---------------- Hash ----------------
	Register(ns+"Hash", "file", "path [, algo]", "Berechnet den Hash einer Datei. Unterstützt 'md5' (Standard), 'sha256' und 'sha512'.", func(args []Value) Value {
		if len(args) < 1 {
			return ErrorVal("file.Hash(path [, algo]) benötigt mindestens einen Pfad")
		}
		path, errVal := absPathVal(args[0].Str)
		if errVal != nil {
			return *errVal
		}

		algo := "md5"
		if len(args) >= 2 {
			algo = strings.ToLower(args[1].Str)
		}

		f, err := os.Open(path)
		if err != nil {
			return ErrorVal("Hash-Fehler: " + err.Error())
		}
		defer f.Close()

		// Interface-Tricks für die verschiedenen Algos
		var h io.Writer
		var hashObj interface{ Sum([]byte) []byte }

		switch algo {
		case "sha256":
			s256 := sha256.New()
			h, hashObj = s256, s256
		case "sha512":
			s512 := sha512.New()
			h, hashObj = s512, s512
		default:
			m := md5.New()
			h, hashObj = m, m
		}

		// Streamt die Datei direkt in den Hash-Generator
		if _, err := io.Copy(h, f); err != nil {
			return ErrorVal("Lesefehler beim Hashing: " + err.Error())
		}

		return StrVal(hex.EncodeToString(hashObj.Sum(nil)))
	})

	// ---------------- Rename ----------------
	Register(ns+"Rename", "file", "alt, neu", "Benennt eine Datei um.", func(args []Value) Value {
		if len(args) < 2 {
			return ErrorVal("file.Rename(oldPath, newPath) benötigt 2 Argumente")
		}

		// 1. Quellpfad validieren
		oldPath, e1 := absPathVal(args[0].Str)
		if e1 != nil {
			return *e1 // Abbruch, falls Quelle illegal (z.B. C:\ auf Linux)
		}

		// 2. Zielpfad validieren
		newPath, e2 := absPathVal(args[1].Str)
		if e2 != nil {
			return *e2 // Abbruch, falls Ziel illegal
		}

		// 3. Die eigentliche Operation
		err := os.Rename(oldPath, newPath)
		if err != nil {
			return ErrorVal("Fehler beim Umbenennen/Verschieben: " + err.Error())
		}

		return NullVal()
	})

	// ------------------------
	// AppendAllText
	// ------------------------
	Register(ns+"AppendAllText", "file", "path, text", "Hängt Text an eine Datei an (erstellt diese, falls nicht vorhanden).", func(args []Value) Value {
		if len(args) < 2 {
			return ErrorVal("file.AppendAllText(path, text) benötigt Pfad und Text")
		}

		path, errVal := absPathVal(args[0].Str)
		if errVal != nil {
			return *errVal
		}

		// Verzeichnis muss existieren
		if _, err := os.Stat(filepath.Dir(path)); err != nil {
			return ErrorVal("Verzeichnis existiert nicht: " + filepath.Dir(path))
		}

		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return ErrorVal("Datei konnte nicht geöffnet werden: " + err.Error())
		}
		defer f.Close()

		if _, err := f.WriteString(args[1].Str); err != nil {
			return ErrorVal("Schreibfehler beim Anhängen: " + err.Error())
		}

		return NullVal()
	})

	// ---------------- ReadAllLines ----------------
	Register(ns+"ReadAllLines", "file", "pfad", "Liest eine Textdatei zeilenweise in ein Array.", func(args []Value) Value {
		if len(args) < 1 {
			return ErrorVal("file.ReadAllLines benötigt einen Pfad")
		}

		path, errVal := absPathVal(args[0].Str)
		if errVal != nil {
			return *errVal
		}

		// --- NEU: Shared Access nutzen ---
		file, err := openFileShared(path)
		if err != nil {
			return ErrorVal("Fehler beim Öffnen der Zeilen: " + err.Error())
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			return ErrorVal("Fehler beim Lesen der Zeilen: " + err.Error())
		}

		// Normalisierung der Zeilenenden
		content := strings.ReplaceAll(string(data), "\r\n", "\n")
		content = strings.TrimRight(content, "\n")

		lines := strings.Split(content, "\n")

		return Value{
			Kind: KindArr,
			Arr:  stringSliceToValueSlice(lines),
		}
	})

	// ---------------- ReadAllText ----------------
	Register(ns+"ReadAllText", "file", "pfad", "Liest den gesamten Inhalt einer Datei als String.", func(args []Value) Value {
		if len(args) < 1 {
			return ErrorVal("ReadAllText erwartet einen Dateipfad")
		}

		path, errVal := absPathVal(args[0].Str)
		if errVal != nil {
			return *errVal
		}

		// --- NEU: Shared Access nutzen ---
		file, err := openFileShared(path)
		if err != nil {
			return ErrorVal("Lesefehler in ReadAllText: " + err.Error())
		}
		defer file.Close()

		// Den gesamten Inhalt aus dem Handle lesen
		content, err := io.ReadAll(file)
		if err != nil {
			return ErrorVal("Fehler beim Stream-Lesen: " + err.Error())
		}

		return Value{Kind: KindStr, Str: string(content)}
	})

	// ---------------- Ext ----------------
	Register(ns+"Ext", "file", "pfad", "Gibt die Dateiendung zurück.", func(args []Value) Value {
		if len(args) < 1 {
			return StrVal("")
		}

		// 1. Pfad validieren (Wichtig: absPathVal gibt 2 Werte zurück!)
		path, errVal := absPathVal(args[0].Str)
		if errVal != nil {
			return *errVal // Sicherheits-Stopp
		}

		// 2. Erweiterung extrahieren
		ext := filepath.Ext(path)

		// 3. Den Punkt entfernen (aus ".txt" wird "txt")
		return StrVal(strings.TrimPrefix(ext, "."))
	})

	// ---------------- Name ----------------
	Register(ns+"Name", "file", "pfad", "Gibt den Dateinamen ohne Verzeichnispfad zurück.", func(args []Value) Value {
		if len(args) < 1 {
			return StrVal("")
		}

		// 1. Pfad validieren und entpacken
		path, errVal := absPathVal(args[0].Str)
		if errVal != nil {
			return *errVal // Sicherheits-Stopp bei illegalen Pfaden
		}

		// 2. Basisname holen (z.B. "test.txt")
		base := filepath.Base(path)

		// 3. Endung holen (z.B. ".txt")
		ext := filepath.Ext(base)

		// 4. Endung abschneiden und zurückgeben
		return StrVal(strings.TrimSuffix(base, ext))
	})

	// ---------------- Dir ----------------
	Register(ns+"Dir", "file", "pfad", "Gibt das Verzeichnis (Parent) eines Pfades zurück.", func(args []Value) Value {
		if len(args) < 1 {
			return StrVal("")
		}

		// 1. Pfad validieren (Gibt (string, *Value) zurück)
		path, errVal := absPathVal(args[0].Str)
		if errVal != nil {
			return *errVal // Sicherheits-Stopp
		}

		// 2. Das übergeordnete Verzeichnis ermitteln
		// filepath.Dir gibt unter Windows "\" oder unter Linux "/" zurück,
		// wenn kein Pfad mehr darüber liegt.
		dir := filepath.Dir(path)

		return StrVal(dir)
	})

	// ------------------------
	// Pfad zusammensetzen
	// ------------------------
	Register(ns+"Join", "file", "pfad, teil", "Verbindet zwei Pfadsegmente sicher miteinander.", func(args []Value) Value {
		if len(args) < 2 {
			return Value{Kind: KindStr, Str: ""}
		}

		parts := make([]string, len(args))
		for i, v := range args {
			// Wandelt alles zuverlässig in String um
			parts[i] = toStringSafe(v, "")
		}

		return Value{Kind: KindStr, Str: filepath.Join(parts...)}
	})

	Register(ns+"ReadBytes", "file", "path",
		"Liest eine Datei als Byte-Array ein und gibt ein Array von Zahlen (0–255) zurück.",
		func(args []Value) Value {
			if len(args) < 1 {
				return ErrorVal("file.ReadBytes benötigt Pfad")
			}

			path, errVal := absPathVal(args[0].Str)
			if errVal != nil {
				return *errVal
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return ErrorVal("Fehler beim Lesen: " + err.Error())
			}

			arr := make([]Value, len(data))
			for i, b := range data {
				arr[i] = NumVal(float64(b))
			}
			return Value{Kind: KindArr, Arr: arr}
		})

	Register(ns+"WriteBytes", "file", "path, byteArray",
		"Schreibt ein Byte-Array (Zahlen 0–255) in eine Datei.",
		func(args []Value) Value {
			if len(args) < 2 {
				return ErrorVal("file.WriteBytes(path, byteArray) benötigt 2 Argumente")
			}
			if args[1].Kind != KindArr {
				return ErrorVal("file.WriteBytes: zweites Argument muss ein Array sein")
			}

			path, errVal := absPathVal(args[0].Str)
			if errVal != nil {
				return *errVal
			}

			// Zielverzeichnis muss existieren
			if _, err := os.Stat(filepath.Dir(path)); err != nil {
				return ErrorVal("Zielverzeichnis existiert nicht: " + filepath.Dir(path))
			}

			buf := make([]byte, len(args[1].Arr))
			for i, v := range args[1].Arr {
				buf[i] = byte(v.Num)
			}

			if err := os.WriteFile(path, buf, 0644); err != nil {
				return ErrorVal("Fehler beim Schreiben: " + err.Error())
			}

			return NullVal()
		})

	Register(ns+"Tail", "file", "path, [lines], [refresh]", "Zeigt die letzten Zeilen einer Datei an. Bei 'refresh' (z.B. '1s') wird die Datei live überwacht.", func(args []Value) Value {
		if len(args) < 1 {
			return ErrorVal("Tail benötigt mindestens den Dateipfad")
		}

		// 1. Pfad auflösen
		path, eVal := absPathVal(args[0].Str)
		if eVal != nil {
			return *eVal
		}

		// 2. Anzahl der Zeilen (Default: 10)
		requestedLines := 10
		if len(args) >= 2 {
			requestedLines = int(toNumVal(args[1]))
		}
		if requestedLines <= 0 {
			return StrVal("")
		}
		if requestedLines > 5000 {
			requestedLines = 5000
		}

		// 3. Zeit-Parsing (Flexibel: "1s", "500ms" oder reine Zahl als ms)
		var d time.Duration
		isFollowMode := false

		if len(args) >= 3 {
			isFollowMode = true
			durationStr := args[2].Str

			// Versuchen als Duration zu parsen (z.B. "1s")
			parsed, err := time.ParseDuration(durationStr)
			if err != nil {
				// Fallback: Wenn es eine reine Zahl ist (z.B. "500"), als ms interpretieren
				ms, numErr := strconv.Atoi(durationStr)
				if numErr == nil {
					d = time.Duration(ms) * time.Millisecond
				} else {
					// Letzter Versuch: Falls toNumVal eine Zahl liefert
					d = time.Duration(toNumVal(args[2])) * time.Millisecond
				}
			} else {
				d = parsed
			}
		}

		// --- Phase 1: Die letzten X Zeilen lesen ---
		file, err := openFileShared(path)
		if err != nil {
			return ErrorVal("Datei konnte nicht geöffnet werden: " + err.Error())
		}
		defer file.Close()

		stat, _ := file.Stat()
		filesize := stat.Size()
		var result []byte

		if filesize > 0 {
			var lineCount int
			var offset int64
			chunkSize := int64(4096)

			for offset < filesize && lineCount < requestedLines {
				if filesize-offset < chunkSize {
					chunkSize = filesize - offset
				}
				offset += chunkSize
				file.Seek(filesize-offset, 0)

				buf := make([]byte, chunkSize)
				file.Read(buf)

				for i := len(buf) - 1; i >= 0; i-- {
					if buf[i] == '\n' {
						lineCount++
						if lineCount > requestedLines {
							buf = buf[i+1:]
							break
						}
					}
				}
				result = append(buf, result...)
			}
		}

		// Wenn kein Follow-Modus (Intervall <= 0 oder nicht angegeben): String zurückgeben
		if !isFollowMode || d <= 0 {
			return StrVal(string(result))
		}

		// --- Phase 2: AutoRefresh-Modus (Blockierend) ---
		fmt.Print(string(result))
		lastSize := filesize

		for {
			time.Sleep(d) // Nutzt die flexibel geparste Duration

			newStat, err := os.Stat(path)
			if err != nil {
				break
			}

			currentSize := newStat.Size()
			if currentSize > lastSize {
				file.Seek(lastSize, 0)
				newBytes := make([]byte, currentSize-lastSize)
				_, err = file.Read(newBytes)
				if err == nil {
					fmt.Print(string(newBytes))
				}
				lastSize = currentSize
			} else if currentSize < lastSize {
				fmt.Println("\n--- Datei wurde rotiert/gekürzt ---")
				lastSize = currentSize
				file.Seek(0, 0)
			}
		}
		return NullVal()
	})

	// file.Watch(path [, timeout_ms])
	Register(ns+"Watch", "file", "path, [timeoutMs]", "Wartet, bis die Datei geändert wird. Gibt True bei Änderung, False bei Timeout zurück.", func(args []Value) Value {
		if len(args) < 1 {
			return ErrorVal("file.Watch(path) benötigt einen Pfad")
		}

		path, eVal := absPathVal(args[0].Str)
		if eVal != nil {
			return *eVal
		}

		// Optionaler Timeout, damit das Skript nicht ewig hängt
		timeout := 0
		if len(args) >= 2 {
			timeout = int(toNumVal(args[1]))
		}

		info, err := os.Stat(path)
		if err != nil {
			return ErrorVal("Datei existiert nicht: " + err.Error())
		}

		lastMod := info.ModTime()
		start := time.Now()

		// Wir "pollen" hier in einem sehr feinen Intervall (100ms).
		// Das ist CPU-schonend, reagiert aber quasi sofort.
		for {
			// Prüfen, ob Timeout abgelaufen ist
			if timeout > 0 && time.Since(start).Milliseconds() > int64(timeout) {
				return BoolVal(false) // Zeit abgelaufen, keine Änderung
			}

			currentInfo, err := os.Stat(path)
			if err != nil {
				return ErrorVal("Datei während der Überwachung verloren: " + err.Error())
			}

			// Hat sich das Änderungsdatum oder die Größe geändert?
			if currentInfo.ModTime().After(lastMod) || currentInfo.Size() != info.Size() {
				return BoolVal(true) // Treffer!
			}

			// Kurze Pause, um die CPU nicht auf 100% zu jagen
			time.Sleep(100 * time.Millisecond)
		}
	})

	// file.WatchLog(path, pattern [, style_value])
	Register(ns+"WatchLog", "file", "path, pattern, [style]", "Überwacht ein Log live auf ein Suchmuster und hebt Zeilen farbig hervor.", func(args []Value) Value {

		if len(args) < 2 {
			return ErrorVal("watchlog(path, pattern, [style]) benötigt mindestens Pfad und Suchmuster")
		}

		path := ToString(args[0])
		pattern := ToString(args[1])

		ansiSequence := ""

		// --- STYLE HANDLING (UNIFIED) ---
		if len(args) >= 3 {
			switch args[2].Kind {

			case KindNum:
				// altes System (4102 etc.)
				ansiSequence = getAnsiCode(args[2].Num)

			case KindStr:
				// neues System (direkte ANSI-Codes)
				ansiSequence = args[2].Str
			}
		}

		return watchLogInternal(path, pattern, ansiSequence)
	})

	Register(ns+"UpdateValue", "file", "path, search, newValue",
		"Sucht die erste Zeile, die mit 'search' beginnt oder identisch ist, und ersetzt sie.",
		func(args []Value) Value {
			if len(args) < 3 {
				return ErrorVal("Parameter: Pfad, Suche/Eindeutiger String, Neuer Wert")
			}

			path := ToString(args[0])
			search := ToString(args[1])
			newValue := ToString(args[2])

			// 1. Datei einlesen
			input, err := os.ReadFile(path)
			if err != nil {
				return ErrorVal("Lesefehler: " + err.Error())
			}

			content := string(input)
			// Zeilenumbruch-Stil erkennen (für späteres Speichern)
			isCRLF := strings.Contains(content, "\r\n")

			// Normalisieren und Splitten
			lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")

			foundIndex := -1
			for i, line := range lines {
				// Wir prüfen auf den Anfang der Zeile (Prefix-Match)
				// Das erlaubt "Port=" ebenso wie "Vollständige Zeile"
				if strings.HasPrefix(strings.TrimSpace(line), search) {
					foundIndex = i
					break // Wir nehmen das erste Vorkommen, das eindeutig genug ist
				}
			}

			if foundIndex == -1 {
				return BoolVal(false) // Nichts gefunden -> Skript kann reagieren
			}

			// Zeile ersetzen
			lines[foundIndex] = newValue

			// 3. Zusammenbauen im Originalformat
			joinChar := "\n"
			if isCRLF {
				joinChar = "\r\n"
			}

			output := strings.Join(lines, joinChar)
			err = os.WriteFile(path, []byte(output), 0644)
			if err != nil {
				return ErrorVal("Schreibfehler: " + err.Error())
			}

			return BoolVal(true)
		})

	Register(ns+"ReadValue", "file", "path, prefix",
		"Sucht eine Zeile mit 'prefix', schneidet diesen ab und gibt den getrimmten Rest (Value) zurück.",
		func(args []Value) Value {
			if len(args) < 2 {
				return ErrorVal("Parameter: Pfad, Präfix")
			}

			path := ToString(args[0])
			prefix := ToString(args[1])

			input, err := os.ReadFile(path)
			if err != nil {
				return StrVal("")
			}

			// Normalisierung auf LF und Splitten
			lines := strings.Split(strings.ReplaceAll(string(input), "\r\n", "\n"), "\n")

			for _, line := range lines {
				trimmedLine := strings.TrimSpace(line)

				// Wir prüfen, ob die Zeile mit dem Präfix beginnt
				if strings.HasPrefix(trimmedLine, prefix) {
					// 1. Präfix abschneiden
					rawVal := strings.TrimPrefix(trimmedLine, prefix)
					// 2. Eventuelle Leerzeichen um den Wert herum entfernen
					finalVal := strings.TrimSpace(rawVal)

					return StrVal(finalVal)
				}
			}

			return StrVal("") // Nichts gefunden
		})

	Register(ns+"UniqueLines", "file", "path, caseSensitive", "Entfernt doppelte Zeilen aus einer Datei.", func(args []Value) Value {
		if len(args) < 2 {
			return ErrorVal("file.UniqueLines(path, caseSensitive) benötigt 2 Argumente")
		}

		path, errPtr := processTargetPath(args[0].Str)
		if errPtr != nil {
			return *errPtr
		}

		caseSensitive := args[1].Bool

		// 1. Datei einlesen
		content, err := os.ReadFile(path)
		if err != nil {
			return ErrorVal("Datei konnte nicht gelesen werden: " + err.Error())
		}

		// 2. Zeilenweise verarbeiten
		lines := strings.Split(string(content), "\n")
		seen := make(map[string]struct{})
		var result []string

		for _, line := range lines {
			trimmed := strings.TrimSpace(line)

			if trimmed == "" {
				continue
			}

			compareLine := trimmed
			if !caseSensitive {
				compareLine = strings.ToLower(trimmed)
			}

			if _, exists := seen[compareLine]; !exists {
				seen[compareLine] = struct{}{}
				result = append(result, trimmed)
			}
		}

		// 3. Ergebnis zurückschreiben (oder als String zurückgeben)
		newContent := strings.Join(result, "\n")
		err = os.WriteFile(path, []byte(newContent), 0644)
		if err != nil {
			return ErrorVal("Fehler beim Schreiben: " + err.Error())
		}

		return BoolVal(true)
	})

	Register(ns+"GetDuplicates", "file", "path, caseSensitive", "Findet doppelte Zeilen und gibt sie als Array zurück.", func(args []Value) Value {
		if len(args) < 2 {
			return ErrorVal("file.GetDuplicates(path, caseSensitive) benötigt 2 Argumente")
		}

		path, errPtr := processTargetPath(args[0].Str)
		if errPtr != nil {
			return *errPtr
		}

		caseSensitive := args[1].Bool

		// 1. Datei einlesen
		content, err := os.ReadFile(path)
		if err != nil {
			return ErrorVal("Datei konnte nicht gelesen werden: " + err.Error())
		}

		// 2. Zeilenweise zählen
		lines := strings.Split(string(content), "\n")
		counts := make(map[string]int)
		var duplicates []Value // Wir geben ein Array von VBMini-Values zurück

		// Erst alle vorkommen zählen
		for _, line := range lines {
			trimmed := strings.TrimRight(line, "\r")
			if trimmed == "" {
				continue
			}

			compareLine := trimmed
			if !caseSensitive {
				compareLine = strings.ToLower(trimmed)
			}
			counts[compareLine]++
		}

		// 3. Nur die Einträge sammeln, die öfter als 1x vorkommen
		for line, count := range counts {
			if count > 1 {
				duplicates = append(duplicates, StrVal(line))
			}
		}

		return ArrVal(duplicates)
	})

	Register(ns+"SecureDelete", "file", "path", "Hochsicherheits-Löschung: In-Place AES-Verschlüsselung + Zufalls-Rename + Truncate.", func(args []Value) Value {
		if len(args) < 1 {
			return ErrorVal("file.SecureDelete(path)")
		}

		path, errVal := absPathVal(args[0].Str)
		if errVal != nil {
			return *errVal
		}

		if err := secureDeleteFile(path); err != nil {
			return ErrorVal(err.Error())
		}

		return BoolVal(true)
	})
}

func stringSliceToValueSlice(s []string) []Value {
	arr := make([]Value, len(s))
	for i, str := range s {
		arr[i] = Value{Kind: KindStr, Str: str}
	}
	return arr
}

// Hilfswriter für Base64 mit Umbrüchen
type base64LineWriter struct {
	w     io.Writer
	limit int
	count int
}

func newBase64LineWriter(w io.Writer, limit int) *base64LineWriter {
	return &base64LineWriter{w: w, limit: limit}
}

func (lw *base64LineWriter) Write(p []byte) (int, error) {
	total := 0
	for len(p) > 0 {
		remain := lw.limit - lw.count
		if remain <= 0 {
			if _, err := lw.w.Write([]byte{'\n'}); err != nil {
				return total, err
			}
			lw.count = 0
			remain = lw.limit
		}
		n := len(p)
		if n > remain {
			n = remain
		}
		if _, err := lw.w.Write(p[:n]); err != nil {
			return total, err
		}
		lw.count += n
		total += n
		p = p[n:]
	}
	return total, nil
}

func (lw *base64LineWriter) Close() error {
	if lw.count > 0 {
		if _, err := lw.w.Write([]byte{'\n'}); err != nil {
			return err
		}
	}
	return nil
}

// Hilfsfunktion: Datei kopieren
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	// Kein MkdirAll — Verzeichnis muss bereits existieren
	out, err := os.Create(dst)
	if err != nil {
		return err
	}

	_, err = io.Copy(out, in)
	if err != nil {
		out.Close()
		return err
	}

	return out.Close()
}

// Hilfsfunktion für Kontrast-Check
func getSafeColor(fg int, bg int) string {
	// Wenn Vordergrund Gelb (3) oder Weiß (7) ist und kein Hintergrund gesetzt wurde
	// erzwingen wir einen dunklen Hintergrund oder schalten auf eine dunklere Farbe um.
	if bg == 0 {
		if fg == 3 { // Gelb auf Standard (evtl. Weiß) -> Schwer lesbar
			return "1;33" // Wir machen es wenigstens FETT Gelb, das hilft oft
		}
		if fg == 11 { // Light Yellow
			return "33" // Downgrade auf normales Dunkelgelb
		}
	}
	return ""
}

func mapUserLayoutToGo(userLayout string) string {
	r := strings.NewReplacer(
		"YYYY", "2006",
		"MM", "01",
		"DD", "02",
		"HH", "15",
		"mm", "04", // Achtung: mm für Minuten, MM für Monat
		"SS", "05",
	)
	return r.Replace(userLayout)
}

func fileResult(ok bool, msg ...string) Value {
	if ok {
		return ArrVal([]Value{
			BoolVal(true),
			NullVal(),
		})
	}

	m := ""
	if len(msg) > 0 {
		m = msg[0]
	}

	return ArrVal([]Value{
		BoolVal(false),
		StrVal(m),
	})
}

func copyAndDelete(src, dst string) error {
	if err := copyFile(src, dst); err != nil {
		return fmt.Errorf("copy failed: %w", err)
	}

	if err := os.Remove(src); err != nil {
		return fmt.Errorf("delete after copy failed: %w", err)
	}

	return nil
}
