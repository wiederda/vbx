package main

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"

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

func charClass(r rune) int {
	if unicode.IsDigit(r) {
		return 1
	}
	if unicode.IsLetter(r) {
		return 2
	}
	return 0 // Symbole, Satzzeichen, Unterstrich, Leerzeichen, etc.
}

func normalizeForSort(r rune) (rune, int) {
	switch unicode.ToLower(r) {
	case 'ä':
		return 'a', 1
	case 'ö':
		return 'o', 1
	case 'ü':
		return 'u', 1
	default:
		return unicode.ToLower(r), 0
	}
}

func naturalLess(a, b string) bool {
	ra := []rune(a)
	rb := []rune(b)

	i, j := 0, 0

	for i < len(ra) && j < len(rb) {

		// Zahlen als Zahlen vergleichen
		if unicode.IsDigit(ra[i]) && unicode.IsDigit(rb[j]) {

			startI := i
			startJ := j

			for i < len(ra) && unicode.IsDigit(ra[i]) {
				i++
			}

			for j < len(rb) && unicode.IsDigit(rb[j]) {
				j++
			}

			numA := string(ra[startI:i])
			numB := string(rb[startJ:j])

			numATrim := strings.TrimLeft(numA, "0")
			numBTrim := strings.TrimLeft(numB, "0")

			if numATrim == "" {
				numATrim = "0"
			}

			if numBTrim == "" {
				numBTrim = "0"
			}

			if len(numATrim) != len(numBTrim) {
				return len(numATrim) < len(numBTrim)
			}

			if numATrim != numBTrim {
				return numATrim < numBTrim
			}

			if len(numA) != len(numB) {
				return len(numA) < len(numB)
			}

			continue
		}

		// Zeichenklasse NICHT verändern.
		// Damit bleibt:
		// Sonderzeichen -> Zahlen -> Buchstaben
		classA := charClass(ra[i])
		classB := charClass(rb[j])

		if classA != classB {
			return classA < classB
		}

		// Buchstaben normalisieren.
		ca, umlautA := normalizeForSort(ra[i])
		cb, umlautB := normalizeForSort(rb[j])

		if ca != cb {
			return ca < cb
		}

		// Gleicher Grundbuchstabe:
		// normaler Buchstabe vor Umlaut.
		//
		// a < ä
		// o < ö
		// u < ü
		if umlautA != umlautB {
			return umlautA < umlautB
		}

		i++
		j++
	}

	return len(ra) < len(rb)
}
