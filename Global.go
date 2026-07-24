package main

import (
	"bufio"
	"bytes"
	"crypto/ed25519"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
	"golang.org/x/crypto/ssh"
	"golang.org/x/net/idna"
	"golang.org/x/term"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type RegInfo struct {
	Name        string
	Kind        string
	Description string
}

var idnaProfile = idna.New(
	idna.MapForLookup(),          // UTS#46 Mapping aktivieren
	idna.ValidateLabels(true),    // Labels validieren
	idna.StrictDomainName(false), // Kompatibel mit älteren IDNA2003 Domains
)

func InitGlobal() {
	if builtins == nil {
		builtins = make(map[string]BuiltinInfo)
	}

	Register("Worker", "global", "path", "Startet ein VB-Skript im Hintergrund.", func(args []Value) Value {
		if len(args) < 1 {
			return ErrorVal("Pfad fehlt")
		}

		// 1. Pfad absolut machen & Sicherheitscheck (Windows-Pfade auf Linux etc.)
		absScriptPath, errVal := absPathVal(args[0].Str)
		if errVal != nil {
			return *errVal
		}

		// 2. EXPLIZITE ABSICHERUNG der Endung
		lowPath := strings.ToLower(absScriptPath)
		if !strings.HasSuffix(lowPath, ".vb") && !strings.HasSuffix(lowPath, ".vbc") {
			return ErrorVal("Sicherheit: Worker darf nur .vb oder .vbc Dateien ausführen!")
		}

		// 3. Log-Datei vorbereiten
		logFile, err := os.OpenFile(absScriptPath+".log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return ErrorVal(err.Error())
		}

		// 4. Prozess starten (ruft sich selbst auf)
		self, _ := os.Executable()
		cmd := exec.Command(self, absScriptPath)
		cmd.Stdout = logFile
		cmd.Stderr = logFile
		prepareCmdForBackground(cmd)

		if err := cmd.Start(); err != nil {
			logFile.Close()
			return ErrorVal(err.Error())
		}

		pidStr := strconv.Itoa(cmd.Process.Pid)
		logFile.WriteString("\n--- Worker PID " + pidStr + " gestartet am " + time.Now().Format("15:04:05") + " ---\n")
		logFile.Close()

		return StrVal(pidStr)
	})

	Register("Compare", "global", "v1, v2", "Vergleicht Versionsnummern im Format Major.Minor.Patch. Rückgabe: -1, 0, 1", func(args []Value) Value {
		if len(args) < 2 {
			return ErrorVal("version.Compare: zwei Versionen benötigt")
		}

		v1 := ToString(args[0])
		v2 := ToString(args[1])

		parseVersion := func(v string) []int {
			parts := strings.Split(v, ".")

			result := []int{0, 0, 0}

			for i := 0; i < len(parts) && i < 3; i++ {
				n, err := strconv.Atoi(parts[i])
				if err == nil {
					result[i] = n
				}
			}

			return result
		}

		a := parseVersion(v1)
		b := parseVersion(v2)

		for i := 0; i < 3; i++ {
			if a[i] > b[i] {
				return NumVal(1)
			}

			if a[i] < b[i] {
				return NumVal(-1)
			}
		}

		return NumVal(0)
	})

	Register("GenerateSSHKey", "crypt", "[outFile, algo, bits, pass]",
		"Erstellt ein SSH-Paar (RSA/Ed25519). Schreibt physisch in .ssh oder Zielpfad.",
		func(args []Value) Value {
			// 1. DEFAULTS & DATEN-VORBEREITUNG
			home, _ := os.UserHomeDir()
			algo := "rsa"
			bits := 4096 // Absolute Untergrenze für RSA

			// 2. INPUT-CHECK & ALGO-WAHL
			if len(args) >= 2 && args[1].Str != "" {
				algo = strings.ToLower(args[1].Str)
			}

			// Bits korrigieren / setzen
			if algo == "ed25519" {
				bits = 256
			} else {
				algo = "rsa" // Fallback für Unbekanntes
				if len(args) >= 3 {
					uBits, _ := strconv.Atoi(args[2].Str)
					if uBits > bits {
						bits = uBits
					}
				}
			}

			// 3. PFAD-HANDLING (Erzwingt Verzeichnis-Struktur)
			var targetDir string
			if len(args) >= 1 && strings.TrimSpace(args[0].Str) != "" {
				// Wir nehmen den Input IMMER als Verzeichnis-Pfad
				targetDir, _ = absPathVal(args[0].Str)
			} else {
				// Default: ~/.ssh
				targetDir = filepath.Join(home, ".ssh")
			}

			// JETZT: Ordner erstellen, bevor wir überhaupt an Dateinamen denken!
			// Wenn targetDir "/root/ssh" ist, wird hier ein ORDNER namens "ssh" erstellt.
			if err := os.MkdirAll(targetDir, 0700); err != nil {
				return ErrorVal("Berechtigungsfehler: Konnte Ordner nicht erstellen: " + err.Error())
			}

			// Erst wenn der Ordner sicher existiert, bauen wir den Dateipfad zusammen
			basePath := filepath.Join(targetDir, "id_"+algo+"_vbmini")

			// 4. SICHERHEITS-CHECK: EXISTENZ DER DATEI IM ORDNER
			privPath := basePath
			pubPath := basePath + ".pub"

			if _, err := os.Stat(privPath); err == nil {
				return ErrorVal("Abbruch: Datei existiert bereits im Zielordner: " + privPath)
			}
			if _, err := os.Stat(pubPath); err == nil {
				return ErrorVal("Abbruch: Public Key existiert bereits (" + pubPath + ").")
			}

			// --- AB HIER: PHYSIKALISCHE GENERIERUNG ---
			var privPem []byte
			var pubBytes []byte

			if algo == "ed25519" {
				// ED25519 LOGIK
				pub, priv, err := ed25519.GenerateKey(rand.Reader)
				if err != nil {
					return ErrorVal("Ed25519 Fehler: " + err.Error())
				}

				// MarshalPrivateKey liefert []byte (die rohen Daten)
				pemBlock, err := ssh.MarshalPrivateKey(priv, "")
				if err != nil {
					return ErrorVal("Marshal Fehler: " + err.Error())
				}

				privPem = pem.EncodeToMemory(pemBlock)

				// Public Key Format
				pubKey, _ := ssh.NewPublicKey(pub)
				pubBytes = ssh.MarshalAuthorizedKey(pubKey)

			} else {
				// RSA LOGIK
				privateKey, err := rsa.GenerateKey(rand.Reader, bits)
				if err != nil {
					return ErrorVal("RSA Fehler: " + err.Error())
				}

				// MarshalPKCS1PrivateKey liefert []byte
				rawPrivBytes := x509.MarshalPKCS1PrivateKey(privateKey)

				privPem = pem.EncodeToMemory(&pem.Block{
					Type:  "RSA PRIVATE KEY",
					Bytes: rawPrivBytes, // Korrekt: []byte in []byte Feld
				})

				// Public Key Format
				pubKey, _ := ssh.NewPublicKey(&privateKey.PublicKey)
				pubBytes = ssh.MarshalAuthorizedKey(pubKey)
			}

			// 5. SCHREIBEN DER DATEIEN
			// Private Key (0600)
			if err := os.WriteFile(privPath, privPem, 0600); err != nil {
				return ErrorVal("Schreibfehler Private Key: " + err.Error())
			}

			// Public Key (0644)
			if err := os.WriteFile(pubPath, pubBytes, 0644); err != nil {
				return ErrorVal("Schreibfehler Public Key: " + err.Error())
			}

			fmt.Printf("✔ SSH-Key (%s, %d Bit) erfolgreich erstellt.\n", algo, bits)
			//fmt.Printf("  Pfad: %s\n", outFile)

			return StrVal(basePath)
		})

	// 497: ToClipboard
	Register("ToClipboard", "system", "text", "Kopiert Text in die Zwischenablage.", func(args []Value) Value {
		// Nutzt deine toStringSafe Logik
		text := ""
		if len(args) > 0 {
			text = toStringSafe(args[0], "")
		}

		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("clip")
		} else {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		}
		cmd.Stdin = strings.NewReader(text)
		err := cmd.Run()
		return Value{Kind: KindBool, Bool: err == nil}
	})

	// ------------------------
	// Typumwandlung
	// ------------------------
	// ToInt: Wandelt String/Value robust in Ganzzahl um
	Register("ToInt", "global", "value", "Wandelt einen Wert in eine Ganzzahl um (schneidet Nachkommastellen ab).", func(args []Value) Value {
		if len(args) == 0 {
			return NumVal(0)
		}
		clean := sanitizeNumberStr(args[0])
		f, err := strconv.ParseFloat(clean, 64)
		if err != nil {
			return NumVal(0)
		}
		return NumVal(math.Trunc(f))
	})

	// ToFloat: Wandelt String/Value robust in Kommazahl um
	Register("ToFloat", "global", "value", "Wandelt einen Wert in eine Fließkommazahl um.", func(args []Value) Value {
		if len(args) == 0 {
			return NumVal(0)
		}
		clean := sanitizeNumberStr(args[0])
		f, err := strconv.ParseFloat(clean, 64)
		if err != nil {
			return NumVal(0)
		}
		return NumVal(f)
	})

	// ToBool: Wahrheitsgehalt prüfen
	Register("ToBool", "global", "value", "Wandelt einen Wert in einen Wahrheitswert (True/False) um..", func(args []Value) Value {
		v := MustValue(args, 0)
		return BoolVal(isTruthy(v))
	})

	// ToString: In Text umwandeln
	Register("ToString", "global", "value", "Wandelt einen Wert in einen String um.", func(args []Value) Value {
		return StrVal(ToString(MustValue(args, 0)))
	})

	// Erzeugt ein Zeichen aus einem Code (65 -> A)
	Register("Chr", "global", "code", "Gibt das Zeichen zum Unicode-Wert zurück.", func(args []Value) Value {
		n := MustInt(MustValue(args, 0), 0)
		return StrVal(string(rune(n)))
	})

	// Gibt den Code eines Zeichens zurück (A -> 65)
	Register("Asc", "global", "char", "Gibt den Unicode-Wert des ersten Zeichens zurück.", func(args []Value) Value {
		s := ToString(MustValue(args, 0))
		if len(s) == 0 {
			return NumVal(0)
		}
		return NumVal(float64([]rune(s)[0]))
	})

	// Len: Runen-basiert (Zählt Zeichen, nicht Bytes!)
	Register("Length", "global", "s", "Gibt die Anzahl der Zeichen zurück.", func(args []Value) Value {
		s, err := getStrArg(args, 0, "Len")
		if err.Kind == KindError {
			return err
		}
		return NumVal(float64(len([]rune(s))))
	})

	Register("TrimStart", "global", "s, cutset", "Trimmt nur den Start.", func(args []Value) Value {
		return StrVal(strings.TrimLeft(ToString(MustValue(args, 0)), ToString(MustValue(args, 1))))
	})

	Register("TrimEnd", "global", "s, cutset", "Trimmt nur das Ende.", func(args []Value) Value {
		return StrVal(strings.TrimRight(ToString(MustValue(args, 0)), ToString(MustValue(args, 1))))
	})

	Register("ToLower", "global", "s", "Wandelt den String in Kleinschreibung um.", func(args []Value) Value {
		s, err := getStrArg(args, 0, "ToLower")
		if err.Kind == KindError {
			return err
		}
		return StrVal(strings.ToLower(s))
	})

	Register("ToUpper", "global", "s", "Wandelt den String in Großschreibung um.", func(args []Value) Value {
		s, err := getStrArg(args, 0, "ToUpper")
		if err.Kind == KindError {
			return err
		}
		return StrVal(strings.ToUpper(s))
	})

	Register("Left", "global", "s, n", "Extrahiert n Zeichen von links", func(args []Value) Value {
		s, e1 := getStrArg(args, 0, "Left")
		if e1.Kind == KindError {
			return e1
		}
		n, e2 := getMathArg(args, 1, "Left")
		if e2.Kind == KindError {
			return e2
		}

		runes := []rune(s)
		length := int(n)
		if length <= 0 {
			return StrVal("")
		}
		if length > len(runes) {
			length = len(runes)
		}
		return StrVal(string(runes[:length]))
	})

	Register("Right", "global", "s, n", "Extrahiert n Zeichen von rechts.", func(args []Value) Value {
		s, e1 := getStrArg(args, 0, "Right")
		if e1.Kind == KindError {
			return e1
		}
		n, e2 := getMathArg(args, 1, "Right")
		if e2.Kind == KindError {
			return e2
		}

		runes := []rune(s)
		length := int(n)
		if length <= 0 {
			return StrVal("")
		}
		if length > len(runes) {
			length = len(runes)
		}
		return StrVal(string(runes[len(runes)-length:]))
	})

	Register("MD5", "global", "s", "Erzeugt einen MD5-Hash des übergebenen Textes.", func(args []Value) Value {
		s, err := getStrArg(args, 0, "MD5")
		if err.Kind == KindError {
			return err
		}
		hash := md5.Sum([]byte(s))
		return StrVal(hex.EncodeToString(hash[:]))
	})

	Register("SHA1", "global", "s", "Erzeugt einen SHA1-Hash (für Legacy-Kompatibilität).", func(args []Value) Value {
		hash := sha1.Sum([]byte(ToString(MustValue(args, 0))))
		return StrVal(hex.EncodeToString(hash[:]))
	})

	Register("SHA256", "global", "s", "Erzeugt einen sicheren SHA256-Hash des Textes.", func(args []Value) Value {
		hash := sha256.Sum256([]byte(ToString(MustValue(args, 0))))
		return StrVal(hex.EncodeToString(hash[:]))
	})

	Register("SHA512", "global", "s", "Erzeugt einen hochsicheren SHA512-Hash.", func(args []Value) Value {
		if len(args) < 1 {
			return StrVal("")
		}
		sum := sha512.Sum512([]byte(ToString(args[0])))
		return StrVal(fmt.Sprintf("%x", sum))
	})

	Register("Contains", "global", "s, sub", "Prüft, ob ein Teilstring im Text enthalten ist (Gibt True/False zurück).", func(args []Value) Value {
		s, e1 := getStrArg(args, 0, "Contains")
		if e1.Kind == KindError {
			return e1
		}
		sub, e2 := getStrArg(args, 1, "Contains")
		if e2.Kind == KindError {
			return e2
		}
		return BoolVal(strings.Contains(strings.ToLower(s), strings.ToLower(sub)))
	})

	Register("IndexOf", "global", "s, sub", "Gibt den ersten Index eines Teilstrings zurück (-1 wenn nicht gefunden).", func(args []Value) Value {
		s := ToString(MustValue(args, 0))
		sub := ToString(MustValue(args, 1))
		return NumVal(float64(strings.Index(s, sub)))
	})

	Register("LastIndexOf", "global", "s, sub", "Gibt den letzten Index eines Teilstrings zurück (-1 wenn nicht gefunden).", func(args []Value) Value {
		return NumVal(float64(strings.LastIndex(ToString(MustValue(args, 0)), ToString(MustValue(args, 1)))))
	})

	Register("Replace", "global", "s, find, repl", "Ersetzt alle Vorkommen eines Teilstrings im Text.", func(args []Value) Value {
		// Wir erwarten genau 3 Argumente
		if len(args) < 3 {
			return ErrorVal("Replace(text, find, replace) benötigt 3 Argumente")
		}

		s := ToString(args[0])
		find := ToString(args[1])
		repl := ToString(args[2])

		// strings.ReplaceAll ist in Go extrem performant
		return StrVal(strings.ReplaceAll(s, find, repl))
	})

	Register("ReplaceVars", "global", "text, key/value pairs", "Ersetzt mehrere Platzhalter in einem Text.", func(args []Value) Value {

		if len(args) < 1 {
			return StrVal("")
		}

		text := args[0].Str

		for i := 1; i+1 < len(args); i += 2 {
			key := args[i].Str
			value := args[i+1].Str

			text = strings.ReplaceAll(
				text,
				"{"+key+"}",
				value,
			)
		}

		return StrVal(text)
	})

	Register("Split", "global", "s, [sep]", "Zerlegt einen String in ein Array (Standard-Separator: Leerzeichen oder Komma)", func(args []Value) Value {
		if len(args) < 2 {
			return Value{Kind: KindArr, Arr: []Value{}}
		}

		text := ToString(args[0])
		sep := ToString(args[1])

		// Prüfe den 3. Parameter auf Boolean "unique"
		unique := false
		if len(args) > 2 {
			switch args[2].Kind {
			case KindBool:
				unique = args[2].Bool
			case KindNum:
				unique = args[2].Num == 1
			case KindStr:
				ls := strings.ToLower(args[2].Str)
				unique = (ls == "true" || ls == "1")
			}
		}

		var parts []string
		if sep == " " {
			// Fields entfernt automatisch leere Strings
			parts = strings.Fields(text)
		} else {
			parts = strings.Split(text, sep)
		}

		var result []Value
		seen := map[string]struct{}{}

		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}

			if unique {
				if _, exists := seen[p]; exists {
					continue
				}
				seen[p] = struct{}{}
			}

			result = append(result, StrVal(p))
		}

		return Value{Kind: KindArr, Arr: result}
	})

	Register("Trim", "global", "s, [cutset]", "Entfernt Leerzeichen oder Zeichen.", func(args []Value) Value {
		s := ToString(MustValue(args, 0))
		if len(args) >= 2 {
			return StrVal(strings.Trim(s, ToString(args[1])))
		}
		return StrVal(strings.TrimSpace(s))
	})

	Register("EncodeBase64", "global", "s", "Codiert einen Text sicher als Base64-String.", func(args []Value) Value {
		v := MustValue(args, 0)

		if v.Kind != KindStr {
			return StrVal("error: expected string")
		}

		encoded := base64.StdEncoding.EncodeToString([]byte(v.Str))
		return StrVal(encoded)
	})

	Register("DecodeBase64", "global", "s", "Wandelt einen Base64-String zurück in Klartext.", func(args []Value) Value {
		v := MustValue(args, 0)

		if v.Kind != KindStr {
			return StrVal("error: expected string")
		}

		data, err := base64.StdEncoding.DecodeString(v.Str)
		if err != nil {
			return StrVal("error: " + err.Error())
		}

		return StrVal(string(data))
	})

	Register("Substring", "global", "s, start, [len]", "Gibt einen Teilstring zurück (0-basiert).", func(args []Value) Value {
		if len(args) < 2 {
			return ErrorVal("Substring erwartet: String, StartIndex [, Length]")
		}

		s := ToString(args[0])
		runes := []rune(s)
		L := len(runes)

		// 0-basiert wie in VB.NET
		start := int(toNumVal(args[1]))

		if start < 0 {
			start = 0
		}
		if start >= L {
			return StrVal("")
		}

		length := L - start
		if len(args) >= 3 {
			reqLen := int(toNumVal(args[2]))
			if reqLen < 0 {
				length = 0
			} else if reqLen < length {
				length = reqLen
			}
		}

		return StrVal(string(runes[start : start+length]))
	})

	// rand.Int(max) -> 0 bis max-1
	Register("Int", "global", "[max]", "Gibt eine Ganzzahl von 0 bis max-1 zurück.", func(args []Value) Value {
		if len(args) == 0 {
			return NumVal(float64(randGen.Int()))
		}
		// Wir nutzen getMathArg für Sicherheit und Komma-Support
		max, err := getMathArg(args, 0, "rand.Int")
		if err.Kind == KindError {
			return err
		}

		if max <= 0 {
			return NumVal(0)
		}
		return NumVal(float64(randGen.Intn(int(max))))
	})

	// Macht aus "Hallo Welt!" -> "Hallo+Welt%21"
	Register("URLEncode", "global", "val", "Codiert einen String für die sichere Verwendung in URLs.", func(args []Value) Value {
		return StrVal(url.QueryEscape(ToString(MustValue(args, 0))))
	})

	// Macht aus "Hallo+Welt%21" -> "Hallo Welt!"
	Register("URLDecode", "global", "urlStr", "Decodiert einen URL-konformen String zurück in Text.", func(args []Value) Value {
		s := ToString(MustValue(args, 0))
		res, err := url.QueryUnescape(s)
		if err != nil {
			return ErrorVal(err.Error())
		}
		return StrVal(res)
	})

	Register("PunyEncode", "global", "domain",
		"Kodiert eine Domain in Punycode (IDNA2008/UTS#46).",
		func(args []Value) Value {

			s := ToString(MustValue(args, 0))

			encoded, err := idnaProfile.ToASCII(s)

			arr := make([]Value, 3)

			if err != nil {
				arr[0] = BoolVal(false)
				arr[1] = StrVal("")
				arr[2] = StrVal(err.Error())
			} else {
				arr[0] = BoolVal(true)
				arr[1] = StrVal(encoded)
				arr[2] = StrVal("")
			}

			return Value{Kind: KindArr, Arr: arr}
		})

	Register("PunyDecode", "global", "domain",
		"Dekodiert eine Punycode-Domain zurück in Unicode (IDNA2008/UTS#46).",
		func(args []Value) Value {

			s := ToString(MustValue(args, 0))

			decoded, err := idnaProfile.ToUnicode(s)

			arr := make([]Value, 3)

			if err != nil {
				arr[0] = BoolVal(false)
				arr[1] = StrVal("")
				arr[2] = StrVal(err.Error())
			} else {
				arr[0] = BoolVal(true)
				arr[1] = StrVal(decoded)
				arr[2] = StrVal("")
			}

			return Value{Kind: KindArr, Arr: arr}
		})

	// app.input([prompt]): Liest eine Zeile und erkennt automatisch Zahlen
	Register("Input", "global", "[prompt]", "Liest eine Benutzereingabe von der Konsole (Zahlen werden automatisch erkannt).", func(args []Value) Value {
		if len(args) > 0 {
			fmt.Print(ToString(args[0]))
		}

		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			text := scanner.Text()
			trimmed := strings.TrimSpace(text)

			// Versuch der automatischen Umwandlung in eine Zahl
			cleanNum := strings.Replace(trimmed, ",", ".", 1)
			if f, err := strconv.ParseFloat(cleanNum, 64); err == nil {
				return NumVal(f) // Gibt Typ KindNum zurück!
			}

			return Value{Kind: KindStr, Str: text}
		}
		return Value{Kind: KindStr, Str: ""}
	})

	// -------- date.Sleep(value, [unit]) --------
	Register("Sleep", "global", "wert, [einheit], [showOutput]",
		"Pausiert die Ausführung. ENTER überspringt. showOutput (bool) steuert die Anzeige.",
		func(args []Value) Value {
			if len(args) < 1 {
				return ErrorVal("Sleep: Zeitwert erwartet")
			}

			val := toNumVal(args[0])
			unitRaw := "ms"
			showOutput := true // Standardmäßig interaktiv

			if len(args) >= 2 {
				unitRaw = strings.ToLower(ToString(args[1]))
			}
			if len(args) >= 3 {
				showOutput = ToBool(args[2]) // Neuer Flag-Parameter
			}

			// 1. Zeitberechnung (totalSeconds & d)
			var totalSeconds int
			var d time.Duration
			switch unitRaw {
			case "ms", "msec":
				d = time.Duration(val) * time.Millisecond
				totalSeconds = int(val / 1000)
			case "s", "sec", "seconds":
				d = time.Duration(val) * time.Second
				totalSeconds = int(val)
			case "m", "min", "minutes":
				d = time.Duration(val) * time.Minute
				totalSeconds = int(val * 60)
			case "h", "hr", "hour", "hours":
				d = time.Duration(val) * time.Hour
				totalSeconds = int(val * 3600)
			default:
				return ErrorVal("Sleep: unbekannte Einheit")
			}

			// 2. Ausführung
			if showOutput && totalSeconds >= 1 {
				done := make(chan bool, 1)

				// Goroutine für den Tastenanschlag
				go func() {
					var b [1]byte
					os.Stdin.Read(b[:])
					done <- true
				}()

				fmt.Println("\n(ENTER zum Überspringen)")

				for i := totalSeconds; i > 0; i-- {
					select {
					case <-done:
						fmt.Print("\r\033[K[Übersprungen]          \n")
						return NumVal(val)
					default:
						if totalSeconds >= 60 {
							h, m, s := i/3600, (i%3600)/60, i%60
							if h > 0 {
								fmt.Printf("\rWartezeit: %02d:%02d:%02d...   ", h, m, s)
							} else {
								fmt.Printf("\rWartezeit: %02d:%02d...   ", m, s)
							}
						} else {
							fmt.Printf("\rWartezeit: %d s...   ", i)
						}
						time.Sleep(1 * time.Second)
					}
				}
				fmt.Print("\r\033[K")
			} else {
				// Silent Mode oder Millisekunden: Einfach warten
				time.Sleep(d)
			}

			return NumVal(val)
		})

	Register("Uptime", "global", "", "Gibt die System-Laufzeit in Sekunden zurück.", func(args []Value) Value {
		return Value{Kind: KindNum, Num: getUptimeSeconds()}
	})

	// In InitComputerFunctions registrieren:
	Register("UptimeString", "global", "", "Gibt die System-Uptime als lesbaren Text zurück.", func(args []Value) Value {
		return StrVal(formatUptime(getUptimeSeconds()))
	})

	// ---------------- Unblock ----------------
	Register("Unblock", "global", "number", "Hebt die NTFS-Sicherheitsblockierung (Zone.Identifier) von Dateien auf.", func(args []Value) Value {
		if len(args) < 1 {
			return ErrorVal("file.Unblock(path) benötigt einen Pfad")
		}

		path, errVal := absPathVal(args[0].Str)
		if errVal != nil {
			return *errVal
		}

		// Universal-Check: ADS gibt es nur auf Windows (NTFS)
		if runtime.GOOS != "windows" {
			// Wir geben 0 zurück, da nichts zu tun war, aber werfen keinen Fehler
			return NumVal(0)
		}

		count := 0
		// WalkDir für bessere Performance beim Scannen
		err := filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}

			// Der Pfad zum Alternate Data Stream
			adsPath := p + ":Zone.Identifier"

			// Prüfen, ob der Stream existiert
			if _, err := os.Stat(adsPath); err == nil {
				// Stream löschen = Datei ist "unblocked"
				if errRem := os.Remove(adsPath); errRem == nil {
					count++
				}
			}
			return nil
		})

		if err != nil {
			return ErrorVal("Unblock fehlgeschlagen: " + err.Error())
		}

		// Wir geben die Anzahl der entsperrten Dateien zurück
		return NumVal(float64(count))
	})

	Register("ConvNewLine", "global", "content, [target]",
		"Normalisiert Zeilenumbrüche. Standard: Aktuelles OS (auto). Optional: 'crlf', 'lf', 'cr'.",
		func(args []Value) Value {
			if len(args) < 1 {
				return ErrorVal("Fehler: Text-Inhalt fehlt.")
			}

			input := ToString(args[0])
			target := "auto"

			// Falls ein zweiter Parameter da ist, überschreibt er das Auto-OS
			if len(args) >= 2 {
				target = strings.ToLower(ToString(args[1]))
			}

			// 1. Konsistenz-Check: Alle Umbrüche intern auf LF bringen
			// (Entfernt Chaos aus Dateien, die Mix-Formate enthalten)
			temp := strings.ReplaceAll(input, "\r\n", "\n")
			temp = strings.ReplaceAll(temp, "\r", "\n")

			// 2. Automatik-Modus: Welches OS nutzen wir gerade?
			if target == "auto" || target == "" {
				if runtime.GOOS == "windows" {
					target = "crlf"
				} else {
					target = "lf"
				}
			}

			// 3. Finale Umwandlung im Speicher
			switch target {
			case "crlf":
				return StrVal(strings.ReplaceAll(temp, "\n", "\r\n"))
			case "lf":
				return StrVal(temp)
			case "cr":
				return StrVal(strings.ReplaceAll(temp, "\n", "\r"))
			default:
				return ErrorVal("Unbekanntes Zielformat: " + target)
			}
		})

	Register("GC", "global", "-", "Erzwingt den Garbage Collector, um ungenutzten Speicher sofort freizugeben.", func(args []Value) Value {
		runtime.GC()
		return NullVal()
	})

	Register("Format", "global", "expression, style", "Universal-Formatierung für Zahlen, Datum und Texte.", func(args []Value) Value {
		if len(args) < 1 {
			return StrVal("")
		}
		val := args[0]

		// Falls kein Style angegeben wurde, Standard-String-Konvertierung
		style := ""
		if len(args) >= 2 {
			style = args[1].Str
		} else {
			return StrVal(ToString(val))
		}

		// --- FALL 1: DATUM ---
		if val.Kind == KindStr {
			if t, ok := parseDateSafe(val.Str); ok {
				return StrVal(formatVBNew(t, style))
			}
		}

		// --- FALL 2: ZAHLEN ---
		if val.Kind == KindNum {
			lowStyle := strings.ToLower(style)

			// Benannte Formate (VB-Klassiker)
			switch lowStyle {
			case "currency":
				p := message.NewPrinter(language.German)
				return StrVal(p.Sprintf("%.2f €", val.Num))
			case "percent":
				return StrVal(fmt.Sprintf("%.2f%%", val.Num*100))
			case "hex":
				return StrVal(fmt.Sprintf("%X", int64(val.Num)))
			}

			// Custom-Zahlenformate (0, #, . ,)
			if strings.ContainsAny(style, "0#.,") {
				dec := 0
				if idx := strings.IndexAny(style, ".,"); idx != -1 {
					dec = len(style) - idx - 1
				}
				p := message.NewPrinter(language.German)
				return StrVal(p.Sprintf("%.*f", dec, val.Num))
			}
		}

		// --- FALL 3: TECHNISCHER FALLBACK (Go-Style) ---
		if strings.Contains(style, "%") {
			return StrVal(fmt.Sprintf(style, val.Num))
		}

		return StrVal(ToString(val))
	})

	Register("PrintFormat", "global", "", "Gibt eine Übersicht der Format-Optionen auf der Konsole aus.", func(args []Value) Value {
		help := []string{
			"\033[1mvbmini Format-Zentrale\033[0m",
			"Syntax: Format(wert, style)",
			"",
			"\033[33m[ Zahlen ]\033[0m",
			"  '0.00'      -> Festkomma (z.B. 1250,50)",
			"  '#,##0.0'   -> Tausender-Punkte (z.B. 1.250,5)",
			"  'Currency'  -> Währung mit €",
			"  'Hex'       -> Hexadezimal-Wandlung",
			"",
			"\033[33m[ Datum & Zeit ]\033[0m",
			"  'YYYY-MM-DD' -> ISO Datum",
			"  'DD.MM.YYYY' -> Deutsches Datum",
			"  'ddd'        -> Wochentag (Mo, Di...)",
			"  'MMM'        -> Monatsname (Jan, Feb...)",
			"",
		}
		// Direkte Ausgabe auf die Konsole
		fmt.Println(strings.Join(help, "\n"))

		// Gibt "nichts" zurück, da es ein Befehl ist
		return NullVal()
	})

	// 4. Die Größen-Formatierung (Schon besprochen)
	Register("FormatSize", "global", "bytes", "Wandelt Bytes in GB/MB um.", func(args []Value) Value {
		return StrVal(humanSize(toNumVal(args[0])))
	})

	// --- NEU: Basis-Validierung & Defaulting (Data Hygiene) ---
	Register("Default", "global", "val, default", "Nutzt Default-Wert, falls val NULL/leer ist.", func(args []Value) Value {
		if len(args) < 2 {
			return args[0]
		}
		if args[0].Kind == KindNull || (args[0].Kind == KindStr && args[0].Str == "") {
			return args[1]
		}
		return args[0]
	})

	Register("Wait", "global", "pid", "Pausiert das Skript, bis der Prozess mit der angegebenen PID beendet wurde.", func(args []Value) Value {
		if len(args) < 1 {
			return NumVal(-1)
		}

		pid := int32(args[0].Num)
		p, err := process.NewProcess(pid)
		if err != nil {
			return NumVal(-1)
		}

		// Wartet, bis der Prozess nicht mehr existiert
		for {
			running, _ := p.IsRunning()
			if !running {
				break
			}
			time.Sleep(250 * time.Millisecond)
		}
		return NumVal(1)
	})

	// --- Alert (Einfache Ausgabe mit Bestätigung) ---
	Register("Alert", "global", "text", "Pausiert das Skript und zeigt die Nachricht exakt so an, wie übergeben.", func(args []Value) Value {
		text := ""
		if len(args) >= 1 {
			text = ToString(args[0])
		}

		// Wir geben den Text direkt aus, damit deine Farben (vbRed etc.) wirken
		fmt.Print(text + "\n")

		// Dezenter Hinweis auf die nötige Aktion
		fmt.Print("... [Enter]")

		bufio.NewReader(os.Stdin).ReadString('\n')
		return BoolVal(true)
	})

	// --- Confirm (Ja/Nein Abfrage) ---
	// Confirm: Nur der Text, danach J/N Abfrage
	Register("Confirm", "global", "text", "Zeigt Text an und wartet auf J/N Bestätigung.", func(args []Value) Value {
		text := ""
		if len(args) >= 1 {
			text = ToString(args[0])
		}

		fmt.Print(text) // Keine automatischen Präfixe mehr

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.ToLower(strings.TrimSpace(input))

		// Erkennt J, Y oder Yes als True
		return BoolVal(input == "j" || input == "ja" || input == "y" || input == "yes")
	})

	// math.IsPositive(n) / math.IsNegative(n)
	Register("IsPositive", "global", "n", "Prüft, ob n > 0.", func(args []Value) Value {
		v, err := getMathArg(args, 0, "IsPositive")
		if err.Kind == KindError {
			return BoolVal(false)
		}
		return BoolVal(v > 0)
	})

	// Prüft, ob Value eine Zahl ist (Optimiert)
	Register("IsNumeric", "global", "v", "Prüft, ob der Wert eine Zahl ist oder in eine solche umgewandelt werden kann.", func(args []Value) Value {
		if len(args) != 1 {
			return ErrorVal("isNumeric erwartet genau 1 Argument")
		}
		v := args[0]

		switch v.Kind {
		case KindNum:
			return BoolVal(true)
		case KindStr:
			clean := strings.TrimSpace(sanitizeNumberStr(v))
			if clean == "" {
				return BoolVal(false) // Leere Strings sind keine Zahlen
			}
			// Versuch der Umwandlung
			_, err := strconv.ParseFloat(clean, 64)
			return BoolVal(err == nil)
		default:
			return BoolVal(false)
		}
	})

	Register("IsArray", "global", "val", "Prüft ob der Wert ein Array ist (1D oder 2D).", func(args []Value) Value {
		if len(args) < 1 {
			return BoolVal(false)
		}
		return BoolVal(args[0].Kind == KindArr || args[0].Kind == KindArr2D)
	})

	Register("IsNull", "global", "val", "Prüft ob der Wert Null oder nicht initialisiert ist.", func(args []Value) Value {
		if len(args) < 1 {
			return BoolVal(true)
		}
		k := args[0].Kind
		return BoolVal(k == KindNull || k == KindNil || k == KindNone || k == KindUndefined)
	})

	// Global: IsInteger(v) -> True, wenn es eine ganze Zahl ohne Nachkommastellen ist
	Register("IsInteger", "global", "v", "Prüft, ob der Wert eine Ganzzahl ist.", func(args []Value) Value {
		if len(args) < 1 {
			return BoolVal(false)
		}

		v := args[0]
		var num float64

		switch v.Kind {
		case KindNum:
			num = v.Num
		case KindStr:
			// VB-Style: Strings trimmen und Komma ersetzen
			s := strings.ReplaceAll(strings.TrimSpace(v.Str), ",", ".")
			f, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return BoolVal(false)
			}
			num = f
		case KindBool:
			// In VB ist ein Boolean technisch gesehen oft ein Integer (-1 oder 1)
			// Wenn du streng sein willst, gib hier false zurück.
			return BoolVal(true)
		default:
			// Maps, Arrays etc. sind niemals "Integer"
			return BoolVal(false)
		}

		// Prüfung auf Ganzzahligkeit
		return BoolVal(num == math.Trunc(num))
	})

	// --- In InitGlobal() ---

	Register("IsDate", "global", "val", "Prüft ob ein Wert als Datum parsebar ist.", func(args []Value) Value {
		if len(args) < 1 {
			return BoolVal(false)
		}
		if args[0].Kind != KindStr {
			return BoolVal(false)
		}
		_, ok := parseDateSafe(args[0].Str)
		return BoolVal(ok)
	})

	Register("IsString", "global", "val", "Prüft ob der Wert ein String ist.", func(args []Value) Value {
		if len(args) < 1 {
			return BoolVal(false)
		}
		return BoolVal(args[0].Kind == KindStr)
	})

	Register("IsMap", "global", "val", "Prüft ob der Wert eine Map ist.", func(args []Value) Value {
		if len(args) < 1 {
			return BoolVal(false)
		}
		return BoolVal(args[0].Kind == KindMap)
	})

	Register("IsPrime", "global", "n", "Prüft, ob n eine Primzahl ist (nur durch 1 und sich selbst teilbar).", func(args []Value) Value {
		v, err := getMathArg(args, 0, "IsPrime")
		if err.Kind == KindError {
			return BoolVal(false) // Oder err, falls du streng sein willst
		}

		// Primzahlen müssen Ganzzahlen > 1 sein
		if v < 2 || v != math.Trunc(v) {
			return BoolVal(false)
		}

		n := int64(v)
		if n == 2 || n == 3 {
			return BoolVal(true)
		}
		if n%2 == 0 || n%3 == 0 {
			return BoolVal(false)
		}

		// Optimierte Prüfung bis zur Quadratwurzel von n
		// Wir springen in 6er Schritten (alle Primzahlen > 3 sind 6k ± 1)
		limit := int64(math.Sqrt(float64(n)))
		for i := int64(5); i <= limit; i += 6 {
			if n%i == 0 || n%(i+2) == 0 {
				return BoolVal(false)
			}
		}

		return BoolVal(true)
	})

	// --- Die angepasste Build Funktion ---
	Register("Build", "global", "quelle", "Verschlüsselt ein VB-Skript mit Magic Header.", func(args []Value) Value {
		if len(args) < 1 {
			return ErrorVal("Pfad fehlt")
		}
		source := args[0].Str

		// 1. Quelldatei lesen (Klartext .vb)
		visited := make(map[string]bool)
		data, err := resolveIncludes(source, visited)
		if err != nil {
			return ErrorVal("Include-Error: " + err.Error())
		}

		// --- NEU: Magic Header vorbereiten ---
		header := []byte("VBC!")

		// 2. Daten verschlüsseln
		encryptedBody := crypt(data)

		// 3. Header + Verschlüsselte Daten kombinieren
		// Wir erstellen einen neuen Buffer, der mit "VBC!" beginnt
		finalData := make([]byte, len(header)+len(encryptedBody))
		copy(finalData, header)
		copy(finalData[len(header):], encryptedBody)

		// 4. Zielpfad ermitteln und speichern
		dest := getAvailablePath(source, ".vbc")
		err = os.WriteFile(dest, finalData, 0644)
		if err != nil {
			return ErrorVal("Write-Error: " + err.Error())
		}

		return StrVal("Gespeichert als: " + dest)
	})

	Register("SystemMemory", "global", "-", "Gibt [Total, Available, UsedPercent] des Systems zurück.", func(args []Value) Value {
		v, err := mem.VirtualMemory()
		if err != nil {
			return ErrorVal(err.Error())
		}

		// FIX: Removed 'Base64URLDecode' here
		return Value{
			Kind: KindArr,
			Arr: []Value{
				NumVal(float64(v.Total)),     // [0] Gesamt-RAM in Bytes
				NumVal(float64(v.Available)), // [1] Verfügbarer RAM
				NumVal(v.UsedPercent),        // [2] Auslastung in Prozent
			},
		}
	})

	Register("IsCompiled", "global", "path", "Prüft, ob eine Datei ein verschlüsseltes .vbc Skript mit gültigem Header ist.", func(args []Value) Value {
		if len(args) < 1 {
			return BoolVal(false)
		}

		absName, errVal := absPathVal(args[0].Str)
		if errVal != nil {
			return BoolVal(false)
		}

		data, err := os.ReadFile(absName)
		if err != nil {
			return BoolVal(false)
		}

		magic := "VBC!"
		isVBC := strings.ToLower(filepath.Ext(absName)) == ".vbc"
		hasHeader := len(data) >= len(magic) && string(data[:len(magic)]) == magic

		return BoolVal(isVBC && hasHeader)
	})

	Register("UserName", "global", "", "Aktueller Benutzername.", func(args []Value) Value { return StrVal(UserName()) })
	Register("ComputerName", "global", "", "Hostname des Rechners.", func(args []Value) Value { return StrVal(ComputerName()) })
	// UserDomain: Gibt die Windows-Domäne oder den Rechnernamen zurück
	Register("UserDomain", "global", "", "Gibt die Domäne zurück (Windows) oder den Hostnamen (Linux/macOS).", func(args []Value) Value {
		// 1. Versuch: Windows-Umgebungsvariable
		domain := os.Getenv("USERDOMAIN")

		// 2. Fallback für Linux/macOS: Wenn USERDOMAIN leer ist, nehmen wir den Hostnamen
		if domain == "" {
			hostname, err := os.Hostname()
			if err == nil {
				domain = hostname
			}
		}

		return StrVal(domain)
	})

	// vbArch: Gibt die Prozessor-Architektur zurück (amd64, arm64, etc.)
	Register("Arch", "global", "", "Gibt die Prozessor-Architektur zurück (z.B. amd64).", func(args []Value) Value {
		return Value{Kind: KindStr, Str: runtime.GOARCH}
	})

	Register("OS", "global", "", "Gibt die Prozessor-Architektur zurück (z.B. amd64).", func(args []Value) Value {
		return Value{Kind: KindStr, Str: runtime.GOOS}
	})

	// console.Password([prompt]) -> liest Eingabe ohne Echo
	Register("Password", "global", "[prompt]", "Liest eine Eingabe von der Konsole ohne sie anzuzeigen (für Passwörter).", func(args []Value) Value {
		if len(args) >= 1 && args[0].Str != "" {
			fmt.Print(ToString(args[0]))
		}

		fd := int(os.Stdin.Fd())

		// Fallback: Wenn stdin kein echtes Terminal ist (z.B. Pipe/Redirect),
		// kann der Echo nicht unterdrückt werden -> normale Zeile lesen.
		if !term.IsTerminal(fd) {
			reader := bufio.NewReader(os.Stdin)
			line, err := reader.ReadString('\n')
			if err != nil {
				return ErrorVal("console.Password: " + err.Error())
			}
			return StrVal(strings.TrimRight(line, "\r\n"))
		}

		pass, err := term.ReadPassword(fd)
		// Nach ReadPassword steht der Cursor noch in der Zeile -> Zeilenumbruch nachholen
		fmt.Println()

		if err != nil {
			return ErrorVal("console.Password: " + err.Error())
		}

		return StrVal(string(pass))
	})
}

func formatVBNew(t time.Time, layout string) string {
	res := layout

	// Lange Namen zuerst (Lokalisierung)
	if strings.Contains(res, "dddd") {
		daysLong := []string{"Sonntag", "Montag", "Dienstag", "Mittwoch", "Donnerstag", "Freitag", "Samstag"}
		res = strings.ReplaceAll(res, "dddd", daysLong[t.Weekday()])
	}
	if strings.Contains(res, "MMMM") {
		monthsLong := []string{"", "Januar", "Februar", "März", "April", "Mai", "Juni", "Juli", "August", "September", "Oktober", "November", "Dezember"}
		res = strings.ReplaceAll(res, "MMMM", monthsLong[t.Month()])
	}

	// Standard-Mapping (Lange Kürzel zuerst)
	res = strings.ReplaceAll(res, "YYYY", "2006")
	res = strings.ReplaceAll(res, "YY", "06")
	res = strings.ReplaceAll(res, "MM", "01") // Monat mit 0
	res = strings.ReplaceAll(res, "DD", "02") // Tag mit 0
	res = strings.ReplaceAll(res, "HH", "15")
	res = strings.ReplaceAll(res, "mm", "04")
	res = strings.ReplaceAll(res, "ss", "05")

	// Kurze Kürzel (Einstellige Tage/Monate ohne 0)
	res = strings.ReplaceAll(res, "D", "2")
	res = strings.ReplaceAll(res, "M", "1")

	// Kürzel für Wochentag/Monat (3-stellig)
	if strings.Contains(res, "ddd") {
		daysShort := []string{"So", "Mo", "Di", "Mi", "Do", "Fr", "Sa"}
		res = strings.ReplaceAll(res, "ddd", daysShort[t.Weekday()])
	}
	if strings.Contains(res, "MMM") {
		monthsShort := []string{"", "Jan", "Feb", "Mär", "Apr", "Mai", "Jun", "Jul", "Aug", "Sep", "Okt", "Nov", "Dez"}
		res = strings.ReplaceAll(res, "MMM", monthsShort[t.Month()])
	}

	return t.Format(res)
}

func resolveIncludes(source string, visited map[string]bool) ([]byte, error) {
	// Zirkuläre Abhängigkeiten verhindern
	absPath, err := filepath.Abs(source)
	if err != nil {
		return nil, err
	}
	if visited[absPath] {
		return nil, fmt.Errorf("zirkuläre Abhängigkeit: %s", source)
	}
	visited[absPath] = true

	data, err := os.ReadFile(source)
	if err != nil {
		return nil, err
	}

	baseDir := filepath.Dir(absPath)
	var result []byte

	// Zeilenweise verarbeiten
	for _, line := range bytes.Split(data, []byte("\n")) {
		trimmed := bytes.TrimSpace(line)

		// Prüfen ob es ein #include ist
		if bytes.HasPrefix(trimmed, []byte(`#include "`)) && bytes.HasSuffix(trimmed, []byte(`"`)) {
			// Dateiname extrahieren
			inner := trimmed[len(`#include "`) : len(trimmed)-1]
			includePath := filepath.Join(baseDir, string(inner))

			// Rekursiv auflösen
			included, err := resolveIncludes(includePath, visited)
			if err != nil {
				return nil, fmt.Errorf("in %s: %w", source, err)
			}
			result = append(result, included...)
			result = append(result, '\n')
		} else {
			result = append(result, line...)
			result = append(result, '\n')
		}
	}

	return result, nil
}
