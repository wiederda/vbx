package main

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gitlab.xfreibeuterx.ipv64.net/wiederda/pathutils"
)

// ------------------------
// fileHash: SHA256 einer Datei
// ------------------------
func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Nutze einen Buffer (z.B. 32KB), um Windows-I/O zu beschleunigen
	reader := bufio.NewReaderSize(f, 32*1024)
	h := sha256.New()
	if _, err := io.Copy(h, reader); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func formatHuman(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}

	// Einheiten-Array
	units := []string{"KB", "MB", "GB", "TB", "PB"}

	// Berechnung des Exponenten
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	// %.2f sorgt für die 2 Nachkommastellen
	return fmt.Sprintf("%.2f %s", float64(b)/float64(div), units[exp])
}

func createSymlinkInternal(target, linkPath string, expectDir bool) Value {
	t, _ := filepath.Abs(target)
	l, _ := filepath.Abs(linkPath)

	info, err := os.Stat(t)
	if err != nil {
		return BoolVal(false) // Ziel existiert nicht
	}

	// Validierung: Passt das Ziel zum Namespace?
	if expectDir && !info.IsDir() {
		return ErrorVal("folder.CreateSymlink: Ziel ist kein Ordner")
	}
	if !expectDir && info.IsDir() {
		return ErrorVal("file.CreateSymlink: Ziel ist keine Datei")
	}

	// Symlink erstellen (Go regelt das Windows-Flag intern via os.Symlink)
	if err := os.Symlink(t, l); err != nil {
		return BoolVal(false)
	}
	return BoolVal(true)
}

func quickHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	buf := make([]byte, 4096) // 4KB
	n, _ := f.Read(buf)

	h := sha256.Sum256(buf[:n])
	return string(h[:]), nil
}

// absPath wandelt einen Pfad in einen absoluten Pfad um
/*func absPath(p string) (string, *Value) {
	clean := filepath.Clean(p)

	if runtime.GOOS != "windows" && strings.Contains(clean, ":") {
		err := ErrorVal("Sicherheitsfehler: Windows-Pfad '" + clean + "' ist auf " + runtime.GOOS + " nicht erlaubt.")
		return "", &err
	}

	abs, err := filepath.Abs(clean)
	if err != nil {
		errVal := ErrorVal("Systemfehler: Pfad konnte nicht aufgelöst werden: " + err.Error())
		return clean, &errVal
	}

	return abs, nil
}*/

func absPathVal(raw string) (string, *Value) {
	path, err := pathutils.AbsPath(raw)
	if err != nil {
		v := ErrorVal(err.Error())
		return "", &v
	}
	return path, nil
}
