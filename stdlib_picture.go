package main

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	ico "github.com/Kodeworks/golang-image-ico"
	"github.com/disintegration/imaging"
	"github.com/kbinani/screenshot"
)

func InitPictureFunctions() {
	if builtins == nil {
		builtins = make(map[string]BuiltinInfo)
	}

	ns := "picture." // Wichtig: Mit := deklarieren

	// ---------------- picture.Convert ----------------
	Register(ns+"Convert", "picture", "in, out, fmt [, w, h, q]",
		"Konvertiert ein Bild. Formate: jpg, png. Bei ico: w/h/q durch Größen-Liste (16,32) ersetzen.",
		func(args []Value) Value {
			if len(args) < 3 {
				return ErrorVal("usage: picture.Convert(infile, outfile, format [, width, height, quality | icoSizes])")
			}

			inFile, errIn := absPathVal(args[0].Str)
			if errIn != nil {
				return *errIn
			}

			outFile, errOut := absPathVal(args[1].Str)
			if errOut != nil {
				return *errOut
			}

			format := strings.ToLower(args[2].Str)
			src, err := imaging.Open(inFile)
			if err != nil {
				return ErrorVal("Bild-Ladefehler: " + err.Error())
			}

			// --- ICO Sonderlogik ---
			if format == "ico" {
				if len(args) < 4 || args[3].Str == "" {
					return ErrorVal("Fehlende ICO-Größen (z.B. '16,32,48')")
				}
				return handleIcoExport(src, outFile, args[3].Str)
			}

			// --- Normale Formate (JPG, PNG) ---
			width, _ := strconv.Atoi(args[3].Str)
			height, _ := strconv.Atoi(args[4].Str)
			quality := 100
			if len(args) >= 6 {
				if q, err := strconv.Atoi(args[5].Str); err == nil {
					quality = q
				}
			}

			if width > 0 || height > 0 {
				if width > 0 && height > 0 {
					src = imaging.Fill(src, width, height, imaging.Center, imaging.Lanczos)
				} else {
					src = imaging.Resize(src, width, height, imaging.Lanczos)
				}
			}

			var saveErr error
			switch format {
			case "jpg", "jpeg":
				saveErr = imaging.Save(src, outFile, imaging.JPEGQuality(quality))
			case "png":
				saveErr = imaging.Save(src, outFile)
			default:
				return ErrorVal("Nicht unterstütztes Format: " + format)
			}

			if saveErr != nil {
				return ErrorVal("Speicherfehler: " + saveErr.Error())
			}
			return NullVal()
		})

	// ---------------- picture.ConvertAll ----------------
	Register(ns+"ConvertAll", "picture", "inDir, outDir, fmt [, filter, w, h, q]",
		"Konvertiert alle Bilder eines Ordners.",
		func(args []Value) Value {
			if len(args) < 3 {
				return ErrorVal("usage: picture.ConvertAll(...)")
			}

			inFolder, errIn := absPathVal(args[0].Str)
			if errIn != nil {
				return *errIn
			}
			outFolder, errOut := absPathVal(args[1].Str)
			if errOut != nil {
				return *errOut
			}

			files, err := os.ReadDir(inFolder)
			if err != nil {
				return ErrorVal(err.Error())
			}
			os.MkdirAll(outFolder, 0755)

			converted, failed := 0, 0
			// WICHTIG: Wir holen uns die Funktion von Convert direkt
			convertFunc := builtins[ns+"Convert"].Fn

			for _, f := range files {
				if f.IsDir() {
					continue
				}

				inFile := filepath.Join(inFolder, f.Name())
				baseName := strings.TrimSuffix(f.Name(), filepath.Ext(f.Name()))
				outFile := filepath.Join(outFolder, baseName+"."+args[2].Str)

				// Interner Aufruf von picture.Convert
				res := convertFunc([]Value{
					StrVal(inFile), StrVal(outFile), args[2], // in, out, format
					args[3], args[4], args[5], args[6], // Filter/Size, W, H, Q
				})

				if res.Kind == KindError {
					failed++
				} else {
					converted++
				}
			}
			return StrVal(fmt.Sprintf("Erfolg: %d, Fehler: %d", converted, failed))
		})

	// ---------------- picture.Snapshot ----------------
	Register(ns+"Snapshot", "picture", "outFile [, monitorIndex]",
		"Erstellt einen Screenshot eines Monitors.",
		func(args []Value) Value {
			if len(args) < 1 {
				return ErrorVal("usage: picture.Snapshot(outFile, [idx])")
			}

			// 1. Pfad auflösen
			outFile, errVal := absPathVal(args[0].Str)
			if errVal != nil {
				return *errVal
			}

			// 2. Monitor-Index ermitteln
			idx := 0
			if len(args) >= 2 {
				idx = int(args[1].Num)
			}

			if idx >= screenshot.NumActiveDisplays() {
				return ErrorVal("Monitor nicht gefunden")
			}

			// 3. Screenshot aufnehmen
			bounds := screenshot.GetDisplayBounds(idx)
			img, err := screenshot.CaptureRect(bounds)
			if err != nil {
				// Hier war der Typen-Fehler: Go error -> Value
				return ErrorVal("Screenshot fehlgeschlagen: " + err.Error())
			}

			// 4. Das Bild auch SPEICHERN
			// Wir nutzen imaging.Save, das erkennt das Format (.png, .jpg) automatisch
			if err := imaging.Save(img, outFile); err != nil {
				return ErrorVal("Fehler beim Speichern des Screenshots: " + err.Error())
			}

			return NullVal()
		})
}

// Hilfsfunktion für ICO (macht den Hauptcode sauberer)
func handleIcoExport(src image.Image, outFile, sizesStr string) Value {
	base := strings.TrimSuffix(outFile, filepath.Ext(outFile))
	failed := 0
	for _, s := range strings.Split(sizesStr, ",") {
		size, _ := strconv.Atoi(strings.TrimSpace(s))
		if size <= 0 || size > 256 {
			continue
		}

		resized := imaging.Resize(src, size, size, imaging.Lanczos)
		f, err := os.Create(fmt.Sprintf("%s_%d.ico", base, size))
		if err == nil {
			ico.Encode(f, resized)
			f.Close()
		} else {
			failed++
		}
	}
	if failed > 0 {
		return ErrorVal("Einige ICOs fehlgeschlagen")
	}
	return NullVal()
}
