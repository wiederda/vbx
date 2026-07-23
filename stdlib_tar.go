package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func InitTarFunctions() {
	if builtins == nil {
		builtins = make(map[string]BuiltinInfo)
	}

	ns := "tar."

	Register(ns+"Create", "tar", "tarPath, files...",
		"Erstellt ein TAR-Archiv. Gibt True bei Erfolg zurück, False wenn leer oder Fehler.", func(args []Value) Value {
			if len(args) < 2 {
				return ErrorVal("tar.Create benötigt Pfad und Quelldaten")
			}
			return tarCreate(args[0].Str, args[1:], false, false)
		})

	Register(ns+"CreateFlat", "tar", "tarPath, files...",
		"Erstellt ein TAR-Archiv ohne Ordnerstruktur (alle Dateien im Root). Gibt True bei Erfolg zurück.",
		func(args []Value) Value {
			if len(args) < 2 {
				return ErrorVal("tar.CreateFlat(tarPath, files...) benötigt mindestens 2 Argumente")
			}
			return tarCreate(args[0].Str, args[1:], true, false)
		})

	Register(ns+"Extract", "tar", "archive, dest",
		"Entpackt ein TAR-Archiv in das Zielverzeichnis 'dest'. Gibt true bei Erfolg zurück.", func(args []Value) Value {
			if len(args) < 2 {
				return ErrorVal("tar.Extract(archive, dest) benötigt 2 Argumente")
			}
			return extractArchive(args[0].Str, args[1].Str, false)
		})

	Register(ns+"List", "tar", "path",
		"Gibt ein Array mit Details (Name, Size, IsDir, ModTime) aller Einträge im TAR-Archiv zurück.",
		func(args []Value) Value {
			if len(args) < 1 {
				return ErrorVal("tar.List(path) benötigt einen Pfad")
			}
			return listArchive(args[0].Str, false)
		})

	Register(ns+"Exists", "tar", "tarPath, file",
		"Prüft blitzschnell, ob eine bestimmte Datei im TAR-Archiv existiert, ohne alles zu entpacken.", func(args []Value) Value {
			if len(args) < 2 {
				return NumVal(0)
			}
			return existsInTar(args[0].Str, args[1].Str, false)
		})

	Register(ns+"Add", "tar", "tarPath, files...",
		"Fügt bestehende TAR-Archive weitere Dateien hinzu (Append-Modus).", func(args []Value) Value {
			if len(args) < 2 {
				return ErrorVal("tar.Add(tarPath, files...) benötigt mindestens 2 Argumente")
			}
			return tarAddToArchive(args[0].Str, args[1:], false)
		})

	Register(ns+"ToGz", "tar", "tarPath [, deleteSource]",
		"Komprimiert ein vorhandenes TAR zu einer .tar.gz Datei. Optional kann das Original gelöscht werden.", func(args []Value) Value {
			if len(args) < 1 {
				return ErrorVal("tar.ToGz(tarPath [, deleteSource]) benötigt einen Pfad")
			}
			deleteSource := len(args) > 1 && args[1].Bool
			return convertTarToGz(args[0].Str, deleteSource)
		})

	Register(ns+"GzCreate", "tar", "output, files...",
		"Erstellt ein komprimiertes GZ-Archiv. Gibt True bei Erfolg zurück.", func(args []Value) Value {
			if len(args) < 2 {
				return ErrorVal("tar.GzCreate benötigt Pfad und Quelldaten")
			}
			return tarCreate(args[0].Str, args[1:], false, true)
		})

	Register(ns+"GzCreateFlat", "tar", "output, files...",
		"Erstellt ein komprimiertes GZ-Archiv ohne Ordnerstruktur (alle Dateien im Root). Gibt True bei Erfolg zurück.",
		func(args []Value) Value {
			if len(args) < 2 {
				return ErrorVal("tar.GzCreateFlat(output, files...) benötigt mindestens 2 Argumente")
			}
			return tarCreate(args[0].Str, args[1:], true, true)
		})

	Register(ns+"GzExtract", "tar", "archive, dest",
		"Entpackt ein .tar.gz Archiv vollständig in das Zielverzeichnis.", func(args []Value) Value {
			if len(args) < 2 {
				return ErrorVal("tar.GzExtract(archive, dest) benötigt 2 Argumente")
			}
			return extractArchive(args[0].Str, args[1].Str, true)
		})

	Register(ns+"GzList", "tar", "path",
		"Gibt ein Array mit Details (Name, Size, IsDir, ModTime) aller Dateien im .tar.gz Archiv zurück.",
		func(args []Value) Value {
			if len(args) < 1 {
				return ErrorVal("tar.GzList(path) benötigt einen Pfad")
			}
			return listArchive(args[0].Str, true)
		})

	Register(ns+"GzExists", "tar", "tarPath, file",
		"Sucht nach einer Datei innerhalb eines .tar.gz Archivs.", func(args []Value) Value {
			if len(args) < 2 {
				return NumVal(0)
			}
			return existsInTar(args[0].Str, args[1].Str, true)
		})

	Register(ns+"GzIsValid", "tar", "path",
		"Validiert, ob die Datei ein echtes, unbeschädigtes Gzip-TAR-Archiv ist (Header-Check).", func(args []Value) Value {
			if len(args) < 1 {
				return NumVal(0)
			}
			tarPath := strings.TrimSpace(ToString(args[0]))
			if tarPath == "" {
				return NumVal(0)
			}
			file, err := os.Open(tarPath)
			if err != nil {
				return NumVal(0)
			}
			defer file.Close()

			gr, err := gzip.NewReader(file)
			if err != nil {
				return NumVal(0)
			}
			defer gr.Close()

			tr := tar.NewReader(gr)
			_, err = tr.Next()
			if err != nil {
				return NumVal(0)
			}
			return NumVal(1)
		})
}

///////////////////////////////////////////////////////
// CORE LOGIC
///////////////////////////////////////////////////////

func tarCreate(output string, args []Value, flat bool, gzipEnabled bool) Value {
	var filePaths []string
	if len(args) == 1 && args[0].Kind == KindArr {
		for _, v := range args[0].Arr {
			filePaths = append(filePaths, v.Str)
		}
	} else {
		for _, v := range args {
			filePaths = append(filePaths, v.Str)
		}
	}

	if len(filePaths) == 0 {
		return BoolVal(false)
	}

	outFile, err := os.Create(output)
	if err != nil {
		return BoolVal(false)
	}

	// FIX: success-Flag steuert Cleanup im defer
	success := false
	fileCount := 0

	defer func() {
		outFile.Close()
		if !success || fileCount == 0 {
			os.Remove(output)
		}
	}()

	var tw *tar.Writer

	if gzipEnabled {
		gw, err := gzip.NewWriterLevel(outFile, gzip.DefaultCompression)
		if err != nil {
			return BoolVal(false)
		}
		// FIX: gzip.Writer explizit schließen und Fehler prüfen —
		// defer würde den Close-Fehler (fehlender Checksum-Flush) still verschlucken.
		defer func() {
			if closeErr := gw.Close(); closeErr != nil && success {
				// Archiv ist unvollständig — Cleanup erzwingen
				success = false
			}
		}()
		tw = tar.NewWriter(gw)
	} else {
		tw = tar.NewWriter(outFile)
	}

	// FIX: tar.Writer ebenfalls explizit schließen und Fehler prüfen
	defer func() {
		if closeErr := tw.Close(); closeErr != nil && success {
			success = false
		}
	}()

	base := ""
	if !flat {
		base = commonBasePath(filePaths)
	}

	for _, p := range filePaths {
		if p == "" {
			continue
		}
		if err := addPathToTar(tw, p, base, flat, &fileCount); err != nil {
			return BoolVal(false)
		}
	}

	success = (fileCount > 0)
	return BoolVal(success)
}

func existsInTar(tarPath, fileName string, gzipEnabled bool) Value {
	inFile, err := os.Open(tarPath)
	if err != nil {
		return NumVal(0) // FIX: konsistent NumVal statt Value{Num: 0}
	}
	defer inFile.Close()

	var tr *tar.Reader
	if gzipEnabled {
		gr, err := gzip.NewReader(inFile)
		if err != nil {
			return NumVal(0)
		}
		defer gr.Close()
		tr = tar.NewReader(gr)
	} else {
		tr = tar.NewReader(inFile)
	}

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return NumVal(0)
		}
		if hdr.Name == fileName {
			return NumVal(1)
		}
	}
	return NumVal(0)
}

///////////////////////////////////////////////////////
// HELPERS
///////////////////////////////////////////////////////

func addPathToTar(tw *tar.Writer, path, base string, flat bool, count *int) error {
	return filepath.WalkDir(path, func(currPath string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Defensiv: einzelne Lesefehler überspringen
		}
		if d.IsDir() && flat {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil // Datei zwischen WalkDir und Info() gelöscht
		}

		var name string
		if flat {
			name = d.Name()
		} else {
			rel, err := filepath.Rel(base, currPath)
			if err != nil {
				name = d.Name()
			} else {
				name = rel
			}
		}

		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return nil
		}
		hdr.Name = filepath.ToSlash(name)
		if d.IsDir() {
			hdr.Name += "/"
		}

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		if !d.IsDir() {
			file, err := os.Open(currPath)
			if err != nil {
				return nil
			}
			defer file.Close()

			if _, err = io.Copy(tw, file); err == nil {
				(*count)++
			}
		}
		return nil
	})
}

// isSafePath prüft, ob target sicher innerhalb von base liegt.
// FIX: Robustere Path-Traversal-Absicherung — verhindert "../"-Escapes
// auch auf Windows und bei Pfaden ohne abschließenden Separator.
func isSafePath(base, target string) bool {
	// Beide Pfade normalisieren
	cleanBase := filepath.Clean(base)
	cleanTarget := filepath.Clean(target)

	// Sicherstellen dass cleanBase mit Separator endet, damit
	// "/safe/dirmalicious" nicht als Kind von "/safe/dir" gilt
	if !strings.HasSuffix(cleanBase, string(os.PathSeparator)) {
		cleanBase += string(os.PathSeparator)
	}

	return strings.HasPrefix(cleanTarget+string(os.PathSeparator), cleanBase)
}

func extractArchive(tarPath, dest string, gzipEnabled bool) Value {
	cleanDest, errVal := absPathVal(dest)
	if errVal != nil {
		return *errVal
	}
	os.MkdirAll(cleanDest, 0755)

	srcPath, errVal := absPathVal(tarPath)
	if errVal != nil {
		return *errVal
	}

	file, err := os.Open(srcPath)
	if err != nil {
		return ErrorVal(err.Error())
	}
	defer file.Close()

	var tr *tar.Reader
	if gzipEnabled {
		gr, err := gzip.NewReader(file)
		if err != nil {
			return ErrorVal(err.Error())
		}
		defer gr.Close()
		tr = tar.NewReader(gr)
	} else {
		tr = tar.NewReader(file)
	}

	buf := make([]byte, 1024*1024)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return ErrorVal(err.Error())
		}

		target := filepath.Join(cleanDest, hdr.Name)

		// FIX: isSafePath statt strings.HasPrefix direkt —
		// verhindert "/safe/dirmalicious"-Fehlklassifikation
		if !isSafePath(cleanDest, target) {
			// Eintrag still überspringen, kein Abbruch
			continue
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(target, 0755)
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(target), 0755)
			outFile, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, hdr.FileInfo().Mode())
			if err != nil {
				return ErrorVal(err.Error())
			}
			_, err = io.CopyBuffer(outFile, tr, buf)
			outFile.Close()
			if err != nil {
				return ErrorVal(err.Error())
			}
		}
	}
	return NullVal()
}

func listArchive(tarPath string, gzipEnabled bool) Value {
	srcPath, _ := filepath.Abs(tarPath)

	file, err := os.Open(srcPath)
	if err != nil {
		return Value{Kind: KindArr, Arr: []Value{}}
	}
	defer file.Close()

	var tr *tar.Reader
	if gzipEnabled {
		gr, err := gzip.NewReader(file)
		if err != nil {
			return Value{Kind: KindArr, Arr: []Value{}}
		}
		defer gr.Close()
		tr = tar.NewReader(gr)
	} else {
		tr = tar.NewReader(file)
	}

	var fileList []Value
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			// FIX: Fehler im Return-Wert signalisieren statt still abzubrechen.
			// Caller bekommt die bisher gelesenen Einträge + ein Sentinel-Objekt.
			fileList = append(fileList, Value{Kind: KindMap, Map: map[string]Value{
				"Error": StrVal(fmt.Sprintf("Archiv korrupt ab Eintrag %d: %v", len(fileList), err)),
			}})
			break
		}

		fObj := map[string]Value{
			"Name":    StrVal(hdr.Name),
			"Size":    NumVal(float64(hdr.Size)),
			"IsDir":   BoolVal(hdr.Typeflag == tar.TypeDir),
			"ModTime": StrVal(hdr.ModTime.Format("2006-01-02 15:04:05")),
		}
		fileList = append(fileList, Value{Kind: KindMap, Map: fObj})
	}

	return Value{Kind: KindArr, Arr: fileList}
}

func tarAddToArchive(tarPath string, args []Value, gzipEnabled bool) Value {
	if gzipEnabled {
		// Gzip-Append technisch nicht unterstützt
		return ErrorVal("tar.GzAdd wird nicht unterstützt — bitte GzCreate verwenden")
	}

	if _, err := os.Stat(tarPath); os.IsNotExist(err) {
		return tarCreate(tarPath, args, false, false)
	}

	f, err := os.OpenFile(tarPath, os.O_RDWR, 0644)
	if err != nil {
		return BoolVal(false)
	}
	defer f.Close()

	offset, err := findTarAppendOffset(f)
	if err != nil {
		return BoolVal(false)
	}
	if _, err = f.Seek(offset, io.SeekStart); err != nil {
		return BoolVal(false)
	}

	tw := tar.NewWriter(f)
	defer tw.Close()

	var paths []string
	for _, a := range args {
		if a.Str != "" {
			paths = append(paths, a.Str)
		}
	}

	base := commonBasePath(paths)
	fileCount := 0

	for _, p := range paths {
		if err := addPathToTar(tw, p, base, false, &fileCount); err != nil {
			// Warnung loggen, aber weitermachen
			fmt.Printf("[TAR-Append-Warning]: %v\n", err)
		}
	}

	return BoolVal(fileCount > 0)
}

func convertTarToGz(tarPath string, deleteSource bool) Value {
	srcFile, err := os.Open(tarPath)
	if err != nil {
		return ErrorVal("Quelle konnte nicht geöffnet werden: " + err.Error())
	}
	defer srcFile.Close()

	fi, err := srcFile.Stat()
	if err != nil {
		return ErrorVal("Stat fehlgeschlagen: " + err.Error())
	}

	gzPath := tarPath + ".gz"
	dstFile, err := os.Create(gzPath)
	if err != nil {
		return ErrorVal("Ziel konnte nicht erstellt werden: " + err.Error())
	}
	defer dstFile.Close()

	gw, err := gzip.NewWriterLevel(dstFile, gzip.BestCompression)
	if err != nil {
		return ErrorVal("Gzip-Writer Fehler: " + err.Error())
	}
	gw.Name = filepath.Base(tarPath)
	gw.ModTime = fi.ModTime()

	if _, err = io.Copy(gw, srcFile); err != nil {
		gw.Close()
		os.Remove(gzPath)
		return ErrorVal("Kompression fehlgeschlagen: " + err.Error())
	}

	if err := gw.Close(); err != nil {
		os.Remove(gzPath)
		return ErrorVal("Gzip-Abschluss fehlgeschlagen: " + err.Error())
	}

	if deleteSource {
		os.Remove(tarPath)
	}

	return StrVal(gzPath)
}

// FIX: Infinite-Loop-Schutz — bei korruptem Archiv (non-EOF-Fehler)
// wurde vorher ewig geloopt. Jetzt wird der Fehler zurückgegeben.
func findTarAppendOffset(f *os.File) (int64, error) {
	tr := tar.NewReader(f)

	for {
		_, err := tr.Next()
		if err == io.EOF {
			offset, err := f.Seek(0, io.SeekCurrent)
			if err != nil {
				return 0, err
			}
			if offset < 1024 {
				return 0, fmt.Errorf("TAR zu klein oder leer")
			}
			return offset - 1024, nil
		}
		if err != nil {
			// FIX: war vorher nicht behandelt → Endlosschleife bei korruptem Archiv
			return 0, fmt.Errorf("korruptes TAR-Archiv: %w", err)
		}
	}
}
