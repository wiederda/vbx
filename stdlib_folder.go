// ------------------------
// stdlib_folder.go
// ------------------------

package main

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gitlab.xfreibeuterx.ipv64.net/wiederda/pathutils"
)

func InitFolderFunctions() {
	if builtins == nil {
		builtins = make(map[string]BuiltinInfo)
	}

	ns := "folder."

	// folder.Create
	Register(ns+"Create", "folder", "path", "Erstellt ein Verzeichnis rekursiv. Gibt True zurück, wenn der Ordner existiert oder erfolgreich erstellt wurde.", func(args []Value) Value {
		if len(args) < 1 {
			return BoolVal(false)
		}
		path, errVal := absPathVal(args[0].Str)
		if errVal != nil {
			return BoolVal(false)
		}
		if err := os.MkdirAll(path, 0755); err != nil {
			return BoolVal(false)
		}
		return BoolVal(true)
	})

	// folder.Size(path [, human])
	Register(ns+"Size", "folder", "path [, human]", "Gibt die Gesamtgröße eines Verzeichnisses zurück.", func(args []Value) Value {
		path, _ := absPathVal(getArg(args, 0).Str)

		_, _, size := FolderStats(path, nil)

		if len(args) >= 2 && ToBool(args[1]) {
			return StrVal(formatHuman(size))
		}

		return NumVal(float64(size))
	})

	// folder.Delete
	Register(ns+"Delete", "folder", "path [, force]", "Löscht einen Ordner komplett. Mit 'force' wird versucht, Schreibschutz zu ignorieren.", func(args []Value) Value {
		if len(args) < 1 || args[0].Str == "" {
			return ErrorVal("folder.Delete benötigt einen Pfad")
		}

		path, errVal := absPathVal(args[0].Str)
		if errVal != nil {
			return *errVal
		}

		path = pathutils.LongPath(path)

		force := false
		if len(args) >= 2 && isTruthy(args[1]) {
			force = true
		}

		if force {
			os.Chmod(path, 0666)
		}

		err := os.RemoveAll(path)
		if err == nil || os.IsNotExist(err) {
			return Value{Kind: KindArr, Arr: []Value{}}
		}

		if force {
			var allPaths []string
			var remaining []Value

			filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
				if err == nil {
					allPaths = append(allPaths, pathutils.LongPath(p))
				}
				return nil
			})

			for i := len(allPaths) - 1; i >= 0; i-- {
				p := allPaths[i]
				os.Chmod(p, 0666)
				err := os.Remove(p)
				if err != nil && !os.IsNotExist(err) {
					info, statErr := os.Stat(p)
					if statErr == nil && !info.IsDir() {
						remaining = append(remaining, Value{Kind: KindStr, Str: p})
					}
				}
			}
			return Value{Kind: KindArr, Arr: remaining}
		}

		return Value{Kind: KindArr, Arr: []Value{{Kind: KindStr, Str: path}}}
	})

	// folder.ModTime
	Register(ns+"ModTime", "folder", "path", "Gibt das Datum der letzten Änderung im ISO-Format (RFC3339) zurück.", func(args []Value) Value {
		if len(args) < 1 {
			return ErrorVal("usage: folder.ModTime(path)")
		}
		path, errVal := absPathVal(args[0].Str)
		if errVal != nil {
			return *errVal
		}
		info, err := os.Stat(path)
		if err != nil {
			return ErrorVal("Änderungsdatum konnte nicht gelesen werden: " + err.Error())
		}
		return StrVal(info.ModTime().Format(time.RFC3339))
	})

	// folder.CreateTime
	Register(ns+"CreateTime", "folder", "path", "Gibt das Erstellungsdatum des Ordners zurück.", func(args []Value) Value {
		if len(args) < 1 {
			return ErrorVal("usage: folder.CreateTime(path)")
		}
		path, errVal := absPathVal(args[0].Str)
		if errVal != nil {
			return *errVal
		}
		ctime, _, _, err := FolderTimes(path)
		if err != nil {
			return ErrorVal("Erstellungszeit konnte nicht gelesen werden: " + err.Error())
		}
		return StrVal(ctime.Format(time.RFC3339))
	})

	// folder.AccessTime
	Register(ns+"AccessTime", "folder", "path", "Gibt den Zeitpunkt des letzten Zugriffs zurück.", func(args []Value) Value {
		if len(args) < 1 {
			return ErrorVal("usage: folder.AccessTime(path)")
		}
		path, errVal := absPathVal(args[0].Str)
		if errVal != nil {
			return *errVal
		}
		_, atime, _, err := FolderTimes(path)
		if err != nil {
			return ErrorVal("Zeitstempel (Access) konnte nicht gelesen werden: " + err.Error())
		}
		return StrVal(atime.Format(time.RFC3339))
	})

	// folder.Copy
	Register(ns+"Copy", "folder", "src, dst, [progress], [network]", "Kopiert einen Ordner rekursiv. progress=True zeigt Fortschritt. network=True optimiert für Netzwerk-Transfers.", func(args []Value) Value {
		if len(args) < 2 {
			return ErrorVal("usage: folder.Copy(src, dst, [progress], [network])")
		}

		src, e1 := absPathVal(args[0].Str)
		dst, e2 := absPathVal(args[1].Str)
		if e1 != nil {
			return *e1
		}
		if e2 != nil {
			return *e2
		}

		showProgress := false
		if len(args) >= 3 {
			showProgress = (args[2].Str == "true" || (args[2].Kind == KindBool && args[2].Bool))
		}

		networkMode := false
		if len(args) >= 4 {
			networkMode = (args[3].Str == "true" || (args[3].Kind == KindBool && args[3].Bool))
		}

		type job struct {
			src, dst string
			mode     os.FileMode
		}

		numWorkers := runtime.NumCPU()
		if networkMode {
			numWorkers = numWorkers * 4
			if numWorkers > 16 {
				numWorkers = 16
			}
		} else {
			if numWorkers > 4 {
				numWorkers = 4
			}
		}

		var totalFiles int64
		var copiedFiles int64

		if showProgress {
			filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
				if err == nil && !d.IsDir() {
					atomic.AddInt64(&totalFiles, 1)
				}
				return nil
			})
			mode := "lokal"
			if networkMode {
				mode = "Netzwerk"
			}
			fmt.Printf("Kopiere %d Dateien nach %s [%d Worker, Modus: %s]\n",
				totalFiles, dst, numWorkers, mode)
		}

		jobChan := make(chan job, numWorkers*4)
		var wg sync.WaitGroup

		var progressDone chan struct{}
		if showProgress {
			progressDone = make(chan struct{})
			go func() {
				ticker := time.NewTicker(100 * time.Millisecond)
				defer ticker.Stop()
				for {
					select {
					case <-ticker.C:
						done := atomic.LoadInt64(&copiedFiles)
						pct := 0
						if totalFiles > 0 {
							pct = int(float64(done) / float64(totalFiles) * 100)
						}
						filled := pct / 2
						fmt.Printf("\r[%s%s] %d/%d (%d%%)",
							strings.Repeat("█", filled),
							strings.Repeat("░", 50-filled),
							done, totalFiles, pct)
					case <-progressDone:
						return
					}
				}
			}()
		}

		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := range jobChan {
					copyFileInternalBuffered(j.src, j.dst, j.mode)
					if showProgress {
						atomic.AddInt64(&copiedFiles, 1)
					}
				}
			}()
		}

		filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			rel, _ := filepath.Rel(src, p)
			target := filepath.Join(dst, rel)

			if d.IsDir() {
				info, _ := d.Info()
				os.MkdirAll(target, info.Mode())
			} else {
				info, _ := d.Info()
				jobChan <- job{src: p, dst: target, mode: info.Mode()}
			}
			return nil
		})

		close(jobChan)
		wg.Wait()

		if showProgress {
			close(progressDone)
			fmt.Printf("\r[%s] %d/%d (100%%)\nFertig.\n",
				strings.Repeat("█", 50),
				totalFiles, totalFiles)
		}

		return NullVal()
	})

	// folder.Move
	Register(ns+"Move", "folder", "src, dst", "Verschiebt einen Ordner (Rename mit Fallback auf Copy/Delete).",
		func(args []Value) Value {
			if len(args) < 2 {
				return ErrorVal("usage: folder.Move(src, dst)")
			}
			src, _ := absPathVal(args[0].Str)
			dst, _ := absPathVal(args[1].Str)

			if err := os.Rename(src, dst); err == nil {
				return NullVal()
			}

			if copyInfo, ok := builtins[ns+"Copy"]; ok {
				copyRes := copyInfo.Fn([]Value{StrVal(src), StrVal(dst)})
				if copyRes.Kind == KindError {
					return copyRes
				}
			} else {
				return ErrorVal("interner Fehler: folder.Copy nicht gefunden")
			}

			os.RemoveAll(src)
			return NullVal()
		},
	)

	// folder.Rename
	Register(ns+"Rename", "folder", "oldPath, newPath", "Benennt einen Ordner um (auf demselben Volume).", func(args []Value) Value {
		if len(args) < 2 {
			return ErrorVal("usage: folder.Rename(oldPath, newPath)")
		}
		oldPath, errVal1 := absPathVal(args[0].Str)
		if errVal1 != nil {
			return *errVal1
		}
		newPath, errVal2 := absPathVal(args[1].Str)
		if errVal2 != nil {
			return *errVal2
		}
		err := os.Rename(oldPath, newPath)
		if err != nil {
			return ErrorVal("Umbenennen fehlgeschlagen: " + err.Error())
		}
		return NullVal()
	})

	// folder.PathCombine
	Register(ns+"PathCombine", "folder", "parts...", "Verbindet mehrere Pfadsegmente sicher zu einem Gesamtpfad.", func(args []Value) Value {
		if len(args) < 2 {
			return ErrorVal("folder.PathCombine benötigt mindestens zwei Pfadteile")
		}
		parts := make([]string, 0, len(args))
		for _, v := range args {
			parts = append(parts, v.Str)
		}
		return Value{Kind: KindStr, Str: filepath.Join(parts...)}
	})

	// folder.GetDirectories
	Register(ns+"GetDirectories", "folder", "[path], [pattern]", "Gibt ein Array mit den Namen aller direkten Unterverzeichnisse zurück.", func(args []Value) Value {
		rawDir := "."
		pattern := "*"

		if len(args) >= 1 && args[0].Str != "" {
			rawDir = args[0].Str
		}
		if len(args) >= 2 && args[1].Str != "" {
			pattern = args[1].Str
		}

		dir, errVal := absPathVal(rawDir)
		if errVal != nil {
			return *errVal
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			return ErrorVal("Verzeichnis konnte nicht gelesen werden: " + err.Error())
		}

		var results []Value
		for _, e := range entries {
			if e.IsDir() {
				match, _ := filepath.Match(pattern, e.Name())
				if match {
					results = append(results, StrVal(e.Name()))
				}
			}
		}

		return Value{Kind: KindArr, Arr: results}
	})

	// folder.GetFiles
	Register(ns+"GetFiles", "folder", "path [, pattern, recursive, fullPath]", "Gibt ein Array mit Dateien im Verzeichnis zurück.", func(args []Value) Value {
		opts := parseFolderArgs(args)

		var out []Value

		err := WalkFolder(opts, func(it FolderItem) error {
			if it.Info != nil && !it.Info.IsDir() {
				name := it.Name
				if opts.FullPath {
					name = it.Path
				}
				out = append(out, StrVal(name))
			}
			return nil
		})

		if err != nil {
			return ErrorVal(err.Error())
		}

		return Value{Kind: KindArr, Arr: out}
	})

	// folder.GetSubFolders
	Register(ns+"GetSubFolders", "folder", "path [, pattern, recursive, fullPath]", "Gibt ein Array mit Unterverzeichnissen zurück.", func(args []Value) Value {
		opts := parseFolderArgs(args)

		var out []Value

		err := WalkFolder(opts, func(it FolderItem) error {
			if it.Info != nil && it.Info.IsDir() && it.Path != opts.Path {
				name := it.Name
				if opts.FullPath {
					name = it.Path
				}
				out = append(out, StrVal(name))
			}
			return nil
		})

		if err != nil {
			return ErrorVal(err.Error())
		}

		return Value{Kind: KindArr, Arr: out}
	})

	// folder.Dir
	Register(ns+"Dir", "folder", "path [, pattern, recursive]", "Gibt ein Array mit allen Einträgen (Dateien + Ordner) als relative Pfade zurück.", func(args []Value) Value {
		opts := parseFolderArgs(args)

		var out []Value

		err := WalkFolder(opts, func(it FolderItem) error {
			rel, _ := filepath.Rel(opts.Path, it.Path)
			if rel != "." {
				out = append(out, StrVal(rel))
			}
			return nil
		})

		if err != nil {
			return ErrorVal(err.Error())
		}

		return Value{Kind: KindArr, Arr: out}
	})

	// folder.CreateSymlink
	Register(ns+"CreateSymlink", "folder", "target, linkPath", "Erstellt einen Symlink auf einen Ordner.", func(args []Value) Value {
		if len(args) < 2 {
			return ErrorVal("folder.CreateSymlink(target, linkPath) benötigt 2 Pfade")
		}
		return createSymlinkInternal(args[0].Str, args[1].Str, true)
	})

	// folder.EmptyFolder
	Register(ns+"EmptyFolder", "folder", "path [, force]", "Löscht den Inhalt eines Ordners, behält den Ordner selbst.", FolderEmpty)

	// folder.isFolderEmpty
	Register(ns+"isFolderEmpty", "folder", "path", "Prüft, ob ein Ordner leer ist.", FolderIsEmpty)

	// folder.Exists
	Register(ns+"Exists", "path", "bool", "Prüft, ob der Pfad ein existierender Ordner ist.", func(args []Value) Value {
		if len(args) < 1 || args[0].Str == "" {
			return BoolVal(false)
		}
		path, errVal := absPathVal(args[0].Str)
		if errVal != nil {
			return BoolVal(false)
		}
		info, err := os.Stat(path)
		return BoolVal(err == nil && info.IsDir())
	})

	// folder.FindDuplicates
	Register(ns+"FindDuplicates", "folder", "path [, pattern]", "Findet Dateien mit identischer Größe und gruppiert sie.", func(args []Value) Value {
		if len(args) < 1 {
			return ErrorVal("usage: folder.FindDuplicates(path [, pattern])")
		}

		var roots []string
		switch args[0].Kind {
		case KindArr:
			for _, v := range args[0].Arr {
				r, errVal := absPathVal(v.Str)
				if errVal != nil {
					return *errVal
				}
				roots = append(roots, r)
			}
		default:
			r, errVal := absPathVal(args[0].Str)
			if errVal != nil {
				return *errVal
			}
			roots = append(roots, r)
		}

		pattern := "*"
		if len(args) >= 2 {
			pattern = args[1].Str
		}

		sizeGroups := make(map[int64][]Value)

		for _, root := range roots {
			_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return nil
				}
				ok, _ := filepath.Match(pattern, d.Name())
				if !ok {
					return nil
				}
				info, err := d.Info()
				if err != nil {
					return nil
				}
				size := info.Size()
				if size == 0 {
					return nil
				}
				sizeGroups[size] = append(sizeGroups[size], Value{
					Kind: KindMap,
					Map: map[string]Value{
						"path": StrVal(p),
						"size": NumVal(float64(size)),
					},
				})
				return nil
			})
		}

		var result []Value
		for _, group := range sizeGroups {
			if len(group) > 1 {
				result = append(result, Value{Kind: KindArr, Arr: group})
			}
		}

		return Value{Kind: KindArr, Arr: result}
	})

	// folder.FindInFiles
	Register(ns+"FindInFiles", "folder", "path, ext, search [, pattern, flags]", "Durchsucht Dateien zeilenweise nach Text. Flags: i=case-insensitive.", func(args []Value) Value {
		if len(args) < 3 {
			return ErrorVal("usage: folder.FindInFiles(path, ext, search [, pattern, flags])")
		}

		root, errVal := absPathVal(args[0].Str)
		if errVal != nil {
			return *errVal
		}

		ext := args[1].Str
		search := args[2].Str

		pattern := "*"
		if len(args) >= 4 {
			pattern = args[3].Str
		}

		flags := ""
		if len(args) >= 5 {
			flags = args[4].Str
		}

		ignoreCase := strings.Contains(flags, "i")

		results := make(chan Value, 1024)
		var wg sync.WaitGroup
		sem := make(chan struct{}, 8)

		_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if ext != "" && filepath.Ext(d.Name()) != ext {
				return nil
			}
			ok, _ := filepath.Match(pattern, d.Name())
			if !ok {
				return nil
			}

			wg.Add(1)
			sem <- struct{}{}

			go func(filePath string) {
				defer wg.Done()
				defer func() { <-sem }()

				file, err := os.Open(filePath)
				if err != nil {
					return
				}
				defer file.Close()

				scanner := bufio.NewScanner(file)
				lineNumber := 0

				for scanner.Scan() {
					lineNumber++
					line := scanner.Text()
					match := false
					if ignoreCase {
						match = strings.Contains(strings.ToLower(line), strings.ToLower(search))
					} else {
						match = strings.Contains(line, search)
					}
					if match {
						results <- Value{
							Kind: KindArr,
							Arr: []Value{
								NumVal(float64(lineNumber)),
								StrVal(line),
								StrVal(filePath),
							},
						}
					}
				}
			}(p)

			return nil
		})

		go func() {
			wg.Wait()
			close(results)
		}()

		var out []Value
		for r := range results {
			out = append(out, r)
		}

		return Value{Kind: KindArr, Arr: out}
	})

	// folder.CheckHash
	Register(ns+"CheckHash", "folder", "groupsArray", "Validiert Dubletten mittels Quick-Hash + SHA256.", func(args []Value) Value {
		if len(args) < 1 || args[0].Kind != KindArr {
			return ErrorVal("usage: folder.CheckHash(groupsArray)")
		}

		var allDuplicates [][]Value

		for _, groupVal := range args[0].Arr {
			if groupVal.Kind != KindArr {
				continue
			}

			qhMap := make(map[string][]Value)
			for _, fileVal := range groupVal.Arr {
				path := fileVal.Map["path"].Str
				qh, err := quickHash(path)
				if err != nil {
					continue
				}
				qhMap[qh] = append(qhMap[qh], fileVal)
			}

			for _, candidates := range qhMap {
				if len(candidates) < 2 {
					continue
				}
				hashes := make(map[string][]Value)
				for _, fileVal := range candidates {
					path := fileVal.Map["path"].Str
					h, err := fileHash(path)
					if err != nil {
						continue
					}
					hashes[h] = append(hashes[h], fileVal)
				}
				for _, files := range hashes {
					if len(files) > 1 {
						allDuplicates = append(allDuplicates, files)
					}
				}
			}
		}

		return Value{Kind: KindArr2D, Arr2D: allDuplicates}
	})

	// folder.Count
	Register(ns+"Count", "folder", "path [, ignore]", "Gibt Anzahl Ordner, Dateien und Gesamtgröße zurück.", func(args []Value) Value {
		path, _ := absPathVal(getArg(args, 0).Str)

		ignore := map[string]bool{}
		if len(args) >= 2 {
			for _, s := range strings.Split(args[1].Str, ",") {
				ignore[strings.TrimSpace(s)] = true
			}
		}

		files, dirs, size := FolderStats(path, ignore)

		return StrVal(fmt.Sprintf(
			"Ordner: %d, Dateien: %d | Größe: %s",
			dirs, files, formatHuman(size),
		))
	})

	// folder.SecureDelete
	Register(ns+"SecureDelete", "folder", "path", "Löscht ein Verzeichnis rekursiv durch Crypto-Shredding.", func(args []Value) Value {
		if len(args) < 1 {
			return ErrorVal("folder.SecureDelete(path)")
		}
		root, errVal := absPathVal(args[0].Str)
		if errVal != nil {
			return *errVal
		}

		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return fmt.Errorf("Zugriffsfehler bei '%s': %w", path, err)
			}
			if !d.IsDir() {
				if err := secureDeleteFile(path); err != nil {
					return fmt.Errorf("Shredding fehlgeschlagen für '%s': %w", path, err)
				}
			}
			return nil
		})

		if err != nil {
			return ErrorVal(err.Error())
		}

		if err := os.RemoveAll(root); err != nil {
			return ErrorVal("Ordnerstruktur konnte nicht gelöscht werden: " + err.Error())
		}

		return BoolVal(true)
	})
}

// =============================================================================
// HELPER
// =============================================================================

func secureDeleteFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("Datei nicht gefunden: %w", err)
	}

	key := make([]byte, 32)
	iv := make([]byte, 16)
	if _, err := rand.Read(key); err != nil {
		return fmt.Errorf("Schlüsselgenerierung fehlgeschlagen: %w", err)
	}
	if _, err := rand.Read(iv); err != nil {
		return fmt.Errorf("IV-Generierung fehlgeschlagen: %w", err)
	}

	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("Zugriff verweigert: %w", err)
	}
	defer f.Close()

	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("Cipher-Initialisierung fehlgeschlagen: %w", err)
	}
	stream := cipher.NewCTR(block, iv)

	buf := make([]byte, 64*1024)
	var offset int64
	for offset < info.Size() {
		n, err := f.ReadAt(buf, offset)
		if n > 0 {
			stream.XORKeyStream(buf[:n], buf[:n])
			if _, err := f.WriteAt(buf[:n], offset); err != nil {
				return fmt.Errorf("Schreibfehler beim Überschreiben: %w", err)
			}
		}
		if err != nil {
			break
		}
		offset += int64(n)
	}

	f.Sync()
	f.Truncate(0)
	f.Close()

	randomName := fmt.Sprintf("shred_%x%x", key[:4], iv[:4])
	newPath := filepath.Join(filepath.Dir(path), randomName)
	if err := os.Rename(path, newPath); err != nil {
		return fmt.Errorf("Umbenennen fehlgeschlagen: %w", err)
	}
	if err := os.Remove(newPath); err != nil {
		return fmt.Errorf("Löschen fehlgeschlagen: %w", err)
	}

	return nil
}

// =============================================================================
// WALK / STATS
// =============================================================================

type FolderOptions struct {
	Path      string
	Pattern   string
	Recursive bool
	FullPath  bool
	Ignore    map[string]bool
	Flags     string
}

type FolderItem struct {
	Path string
	Name string
	Info fs.FileInfo
	Err  error
}

func WalkFolder(opts FolderOptions, fn func(FolderItem) error) error {

	root := opts.Path

	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {

		if err != nil {
			return nil
		}

		// -------------------------
		// IGNORE FILTER
		// -------------------------
		if opts.Ignore != nil {
			if opts.Ignore[d.Name()] {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// -------------------------
		// NON-RECURSIVE MODE
		// -------------------------
		if !opts.Recursive && p != root {
			if d.IsDir() {
				return filepath.SkipDir // Unterverzeichnisse skippen
			}
			// Datei im direkten Root → durchlassen
		}

		// -------------------------
		// PATTERN FILTER
		// Root selbst nie filtern (d.Name() wäre der Ordnername, nicht eine Datei)
		// -------------------------
		if p != root && opts.Pattern != "" && opts.Pattern != "*" {
			ok, _ := filepath.Match(opts.Pattern, d.Name())
			if !ok {
				if d.IsDir() {
					return nil // Ordner überspringen aber nicht skippen (Geschwister bleiben erreichbar)
				}
				return nil
			}
		}

		// -------------------------
		// FILE INFO
		// -------------------------
		var info fs.FileInfo
		if d.Type().IsRegular() || d.IsDir() {
			info, _ = d.Info()
		}

		return fn(FolderItem{
			Path: p,
			Name: d.Name(),
			Info: info,
		})
	})
}

func parseFolderArgs(args []Value) FolderOptions {
	opts := FolderOptions{
		Path:      ".",
		Pattern:   "*",
		Recursive: false,
		FullPath:  true,
		Ignore:    nil,
	}

	if len(args) >= 1 && args[0].Str != "" {
		opts.Path = args[0].Str
	}
	if len(args) >= 2 && args[1].Str != "" {
		opts.Pattern = args[1].Str
	}
	if len(args) >= 3 {
		opts.Recursive = isTruthy(args[2])
	}
	if len(args) >= 4 {
		opts.FullPath = isTruthy(args[3])
	}
	return opts
}

func FolderStats(path string, ignore map[string]bool) (files, dirs int64, size int64) {

	opts := FolderOptions{
		Path:      path,
		Recursive: true, // FolderStats muss immer rekursiv zählen
		Ignore:    ignore,
	}

	_ = WalkFolder(opts, func(it FolderItem) error {
		if it.Info == nil {
			return nil
		}
		if it.Info.IsDir() {
			dirs++
		} else {
			files++
			size += it.Info.Size()
		}
		return nil
	})

	return
}

func WalkWithWorkers(root string, workers int, fn func(string, fs.DirEntry) error) error {
	jobs := make(chan string, workers*4)
	var wg sync.WaitGroup

	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		jobs <- p
		return nil
	})

	if err != nil {
		return err
	}

	close(jobs)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range jobs {
				_ = fn(p, nil)
			}
		}()
	}

	wg.Wait()
	return nil
}
