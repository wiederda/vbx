// ------------------------
// stdlib_crypt.go
// ------------------------

package main

import (
	"crypto/aes"
	"crypto/cipher"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"math/big"
	"os"
	"unicode"
	"unsafe"
)

// ---------------- Crypt State ----------------
func ok(val Value) Value {
	return Value{Kind: KindArr, Arr: []Value{BoolVal(true), val, StrVal("")}}
}

func fail(msg string) Value {
	return Value{Kind: KindArr, Arr: []Value{BoolVal(false), StrVal(""), StrVal(msg)}}
}

func cryptoRandInt(max int) int {
	if max <= 0 {
		return 0
	}
	n, err := crand.Int(crand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0
	}
	return int(n.Int64())
}

func shuffleBytes(b []byte) {
	for i := len(b) - 1; i > 0; i-- {
		j := cryptoRandInt(i + 1)
		b[i], b[j] = b[j], b[i]
	}
}

func shuffleInts(s []int) {
	for i := len(s) - 1; i > 0; i-- {
		j := cryptoRandInt(i + 1)
		s[i], s[j] = s[j], s[i]
	}
}

// ---------------- Init ----------------
func InitCryptFunctions() {
	if builtins == nil {
		builtins = make(map[string]BuiltinInfo)
	}

	ns := "crypt."

	// GUID: Jetzt mit korrekter Hex-Formatierung
	Register(ns+"GUID", "crypt", "-", "Generiert eine eindeutige GUID.", func(args []Value) Value {
		b := make([]byte, 16)

		if _, err := crand.Read(b); err != nil {
			return fail("RNG Fehler: " + err.Error())
		}

		b[6] = (b[6] & 0x0f) | 0x40
		b[8] = (b[8] & 0x3f) | 0x80

		guid := fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
			b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])

		return ok(StrVal(guid))
	})

	// crypt.Wipe
	Register(ns+"Wipe", "crypt", "byteArray",
		"Überschreibt ein Byte-Array (z.B. von reg.ReadProtectedValueBytes) in-place mit Nullen. Sollte aufgerufen werden, sobald ein entschlüsselter Wert nicht mehr benötigt wird.", func(args []Value) Value {
			if len(args) != 1 {
				return ErrorVal("crypt.Wipe erwartet genau 1 Argument")
			}
			if args[0].Kind != KindArr {
				return ErrorVal("crypt.Wipe erwartet ein Byte-Array (KindArr)")
			}

			// WICHTIG: args[0].Arr ist ein Slice-Header, der denselben
			// Backing-Array wie die Original-Variable im Environment teilt
			// (solange dort kein append() eine Neu-Allokation ausgelöst hat).
			// Elementweises Überschreiben wirkt daher auch auf die Original-Variable.
			for i := range args[0].Arr {
				args[0].Arr[i] = NumVal(0)
			}

			return BoolVal(true)
		})

	// crypt.WipeString
	// ACHTUNG: Nutzt "unsafe", um Go's String-Immutability bewusst zu brechen.
	// Damit wird der tatsächliche Speicherinhalt eines Strings mit Nullbytes
	// überschrieben - nicht nur die Variable auf einen neuen (leeren) String
	// umgebogen, wie es "x = """ tun würde.
	Register(ns+"WipeString", "crypt", "value",
		"Überschreibt den Speicherinhalt eines Strings mit Nullbytes (0x00). Im Gegensatz zu 'x = \"\"' wird hier der tatsächliche RAM-Inhalt gelöscht, nicht nur die Variable umgebogen. Nach dem Aufruf enthält die Variable eine Zeichenkette gleicher Länge, aber nur aus Nullbytes.", func(args []Value) Value {
			if len(args) != 1 {
				return ErrorVal("crypt.WipeString erwartet genau 1 Argument")
			}
			if args[0].Kind != KindStr {
				return ErrorVal("crypt.WipeString erwartet einen String")
			}

			s := args[0].Str
			if len(s) == 0 {
				return BoolVal(true) // nichts zu tun
			}

			// Mutable View auf den Backing-Speicher des Strings holen.
			// Ab Go 1.20: unsafe.StringData + unsafe.Slice
			data := unsafe.Slice(unsafe.StringData(s), len(s))
			for i := range data {
				data[i] = 0
			}

			return BoolVal(true)
		})

	// crypt.BytesToString
	Register(ns+"BytesToString", "crypt", "byteArray",
		"Wandelt ein Byte-Array (0-255 Werte) in einen UTF-8-String um. Für kurzzeitige Verwendung gedacht - danach crypt.Wipe auf das Byte-Array anwenden.", func(args []Value) Value {
			if len(args) != 1 {
				return ErrorVal("crypt.BytesToString erwartet genau 1 Argument")
			}
			if args[0].Kind != KindArr {
				return ErrorVal("crypt.BytesToString erwartet ein Byte-Array (KindArr)")
			}

			buf := make([]byte, len(args[0].Arr))
			for i, v := range args[0].Arr {
				if v.Kind != KindNum {
					return ErrorVal("crypt.BytesToString: Array enthält Nicht-Zahlen-Werte")
				}
				buf[i] = byte(int(v.Num))
			}

			return StrVal(string(buf))
		})

	Register(ns+"RNGCryptoProvider", "crypt", "length", "Erzeugt kryptografisch sichere Zufallsbytes und gibt sie als Hex-String zurück.", func(args []Value) Value {
		length := 16
		if len(args) >= 1 {
			length = MustInt(args[0], 16)
		}

		b := make([]byte, length)
		if _, err := crand.Read(b); err != nil {
			return fail("RNG Fehler: " + err.Error())
		}

		return ok(StrVal(fmt.Sprintf("%x", b)))
	})

	Register(ns+"RandomString", "crypt", "length",
		"Erzeugt eine zufällige alphanumerische Zeichenfolge (min. 10 Zeichen).",
		func(args []Value) Value {

			length := 10
			if len(args) >= 1 {
				length = MustInt(args[0], 10)
				if length < 10 {
					length = 10
				}
			}

			const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
			buf := make([]byte, length)

			max := big.NewInt(int64(len(chars))) // nur einmal erzeugen

			for i := 0; i < length; i++ {
				nBig, err := crand.Int(crand.Reader, max)
				if err != nil {
					return ErrorVal("RNG Fehler: " + err.Error())
				}
				buf[i] = chars[nBig.Int64()]
			}

			return StrVal(string(buf))
		})

	Register(ns+"RandomPassword", "crypt", "len, [num], [low], [up], [spec], [head]", "Generiert ein komplexes Zufallspasswort.", func(args []Value) Value {
		length := 12
		numbers := false
		lower := false
		upper := false
		specials := false
		firstLetter := true // Default jetzt true: Passwort startet standardmäßig mit einem Buchstaben

		// --- 1. Argumente ---
		if len(args) >= 1 {
			length = MustInt(args[0], 12)
		}
		if len(args) >= 2 {
			numbers = toBoolSafe(args[1], false)
		}
		if len(args) >= 3 {
			lower = toBoolSafe(args[2], false)
		}
		if len(args) >= 4 {
			upper = toBoolSafe(args[3], false)
		}
		if len(args) >= 5 {
			specials = toBoolSafe(args[4], false)
		}
		if len(args) >= 6 {
			firstLetter = toBoolSafe(args[5], true) // Default für toBoolSafe ebenfalls true
		}

		if length < 10 {
			length = 10
		}

		// Fallback, falls alles false
		if !(numbers || lower || upper || specials) {
			lower = true
		}

		// --- 2. Charsets ---
		lowerChars := "abcdefghijklmnopqrstuvwxyz"
		upperChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		numberChars := "0123456789"
		specialChars := "+-/*#,;.:-_^!()[]{}=?<>@"
		letters := lowerChars + upperChars

		// Falls firstLetter=true, muss mindestens lower oder upper aktiv sein,
		// sonst gibt es keine Buchstaben für pass[0].
		if firstLetter && !lower && !upper {
			lower = true
		}

		charset := ""
		mustInclude := []byte{}

		if numbers {
			charset += numberChars
			mustInclude = append(mustInclude, numberChars[cryptoRandInt(len(numberChars))])
		}
		if lower {
			charset += lowerChars
			mustInclude = append(mustInclude, lowerChars[cryptoRandInt(len(lowerChars))])
		}
		if upper {
			charset += upperChars
			mustInclude = append(mustInclude, upperChars[cryptoRandInt(len(upperChars))])
		}
		if specials {
			charset += specialChars
			mustInclude = append(mustInclude, specialChars[cryptoRandInt(len(specialChars))])
		}

		// --- 3. Passwort initial zufällig füllen ---
		pass := make([]byte, length)
		for i := range pass {
			pass[i] = charset[cryptoRandInt(len(charset))]
		}

		// --- 4. Pflichtzeichen an eindeutigen Positionen einfügen ---
		indices := make([]int, length)
		for i := 0; i < length; i++ {
			indices[i] = i
		}
		shuffleInts(indices)

		for i, c := range mustInclude {
			pass[indices[i]] = c
		}

		// --- 5. Optional: erstes Zeichen Buchstabe ---
		if firstLetter {
			pass[0] = letters[cryptoRandInt(len(letters))]
		}

		// --- 6. Endgültiges Shuffle für gleichmäßige Verteilung ---
		shuffleBytes(pass)

		return StrVal(string(pass))
	})

	// ---------- AES-GCM ----------

	// AESEncrypt: Gibt jetzt [OK, CipherBase64, ErrorMsg] zurück
	Register(ns+"AESEncrypt", "crypt", "text, pass",
		"Verschlüsselt Text mit AES-256-GCM. Rückgabe: [OK, CipherBase64, Msg]",
		func(args []Value) Value {

			if len(args) < 2 {
				return Value{Kind: KindArr, Arr: []Value{
					BoolVal(false),
					StrVal(""),
					StrVal("AESEncrypt benötigt Text und Passwort"),
				}}
			}

			text := args[0].Str
			pass := args[1].Str

			// --- Key aus Passwort ---
			key := sha256.Sum256([]byte(pass))

			// --- Cipher erstellen ---
			block, err := aes.NewCipher(key[:])
			if err != nil {
				return Value{Kind: KindArr, Arr: []Value{
					BoolVal(false),
					StrVal(""),
					StrVal("Cipher Fehler: " + err.Error()),
				}}
			}

			// --- GCM ---
			gcm, err := cipher.NewGCM(block)
			if err != nil {
				return Value{Kind: KindArr, Arr: []Value{
					BoolVal(false),
					StrVal(""),
					StrVal("GCM Fehler: " + err.Error()),
				}}
			}

			// --- Nonce ---
			nonce := make([]byte, gcm.NonceSize())
			if _, err := io.ReadFull(crand.Reader, nonce); err != nil {
				return Value{Kind: KindArr, Arr: []Value{
					BoolVal(false),
					StrVal(""),
					StrVal("Nonce Fehler: " + err.Error()),
				}}
			}

			// --- Verschlüsselung ---
			// Format: nonce + ciphertext
			data := gcm.Seal(nonce, nonce, []byte(text), nil)

			// --- Base64 ---
			encoded := base64.StdEncoding.EncodeToString(data)

			return Value{Kind: KindArr, Arr: []Value{
				BoolVal(true),
				StrVal(encoded),
				StrVal(""),
			}}
		})

	// AESDecrypt: Gibt jetzt [OK, Plaintext, ErrorMsg] zurück
	Register(ns+"AESDecrypt", "crypt", "cipher, pass",
		"Entschlüsselt AES-GCM Daten. Rückgabe: [OK, Plaintext, Msg]",
		func(args []Value) Value {

			if len(args) < 2 {
				return Value{Kind: KindArr, Arr: []Value{
					BoolVal(false),
					StrVal(""),
					StrVal("AESDecrypt benötigt Cipher und Passwort"),
				}}
			}

			cipherText := args[0].Str
			pass := args[1].Str

			// --- Base64 Decode ---
			raw, err := base64.StdEncoding.DecodeString(cipherText)
			if err != nil {
				return Value{Kind: KindArr, Arr: []Value{
					BoolVal(false),
					StrVal(""),
					StrVal("Ungültiges Base64"),
				}}
			}

			// --- Key ---
			key := sha256.Sum256([]byte(pass))

			// --- Cipher ---
			block, err := aes.NewCipher(key[:])
			if err != nil {
				return Value{Kind: KindArr, Arr: []Value{
					BoolVal(false),
					StrVal(""),
					StrVal("Cipher Fehler: " + err.Error()),
				}}
			}

			// --- GCM ---
			gcm, err := cipher.NewGCM(block)
			if err != nil {
				return Value{Kind: KindArr, Arr: []Value{
					BoolVal(false),
					StrVal(""),
					StrVal("GCM Fehler: " + err.Error()),
				}}
			}

			// --- Nonce extrahieren ---
			nsz := gcm.NonceSize()
			if len(raw) < nsz {
				return Value{Kind: KindArr, Arr: []Value{
					BoolVal(false),
					StrVal(""),
					StrVal("Daten zu kurz"),
				}}
			}

			nonce := raw[:nsz]
			cipherData := raw[nsz:]

			// --- Entschlüsselung ---
			plain, err := gcm.Open(nil, nonce, cipherData, nil)
			if err != nil {
				return Value{Kind: KindArr, Arr: []Value{
					BoolVal(false),
					StrVal(""),
					StrVal("Falsches Passwort oder Daten korrupt"),
				}}
			}

			return Value{Kind: KindArr, Arr: []Value{
				BoolVal(true),
				StrVal(string(plain)),
				StrVal(""),
			}}
		})

	Register(ns+"AESEncryptFile", "crypt", "sourcePath, destPath, pass [, deleteSource]",
		"Verschlüsselt eine Datei mit AES-256-GCM und speichert sie unter destPath. Mit deleteSource=true wird die Original-Datei nach erfolgreicher Verschlüsselung gelöscht (Standard: false). Rückgabe: [OK, Msg]",
		func(args []Value) Value {

			if len(args) < 3 {
				return Value{Kind: KindArr, Arr: []Value{
					BoolVal(false),
					StrVal("AESEncryptFile benötigt sourcePath, destPath und Passwort"),
				}}
			}

			srcPath, errVal := absPathVal(args[0].Str)
			if errVal != nil {
				return Value{Kind: KindArr, Arr: []Value{
					BoolVal(false),
					StrVal("Pfad-Fehler (sourcePath): " + errVal.Str),
				}}
			}
			destPath, errVal := absPathVal(args[1].Str)
			if errVal != nil {
				return Value{Kind: KindArr, Arr: []Value{
					BoolVal(false),
					StrVal("Pfad-Fehler (destPath): " + errVal.Str),
				}}
			}
			pass := args[2].Str

			deleteSource := false
			if len(args) > 3 && args[3].Kind == KindBool {
				deleteSource = args[3].Bool
			}

			data, err := os.ReadFile(srcPath)
			if err != nil {
				return Value{Kind: KindArr, Arr: []Value{
					BoolVal(false),
					StrVal("Datei-Lesefehler: " + err.Error()),
				}}
			}

			// --- Key aus Passwort ---
			key := sha256.Sum256([]byte(pass))

			// --- Cipher erstellen ---
			block, err := aes.NewCipher(key[:])
			if err != nil {
				return Value{Kind: KindArr, Arr: []Value{
					BoolVal(false),
					StrVal("Cipher Fehler: " + err.Error()),
				}}
			}

			// --- GCM ---
			gcm, err := cipher.NewGCM(block)
			if err != nil {
				return Value{Kind: KindArr, Arr: []Value{
					BoolVal(false),
					StrVal("GCM Fehler: " + err.Error()),
				}}
			}

			// --- Nonce ---
			nonce := make([]byte, gcm.NonceSize())
			if _, err := io.ReadFull(crand.Reader, nonce); err != nil {
				return Value{Kind: KindArr, Arr: []Value{
					BoolVal(false),
					StrVal("Nonce Fehler: " + err.Error()),
				}}
			}

			// --- Verschlüsselung ---
			// Format: nonce + ciphertext
			encrypted := gcm.Seal(nonce, nonce, data, nil)

			// --- Atomar schreiben: erst .tmp, dann umbenennen ---
			tmpPath := destPath + ".tmp"
			if err := os.WriteFile(tmpPath, encrypted, 0600); err != nil {
				return Value{Kind: KindArr, Arr: []Value{
					BoolVal(false),
					StrVal("Schreibfehler: " + err.Error()),
				}}
			}
			if err := os.Rename(tmpPath, destPath); err != nil {
				return Value{Kind: KindArr, Arr: []Value{
					BoolVal(false),
					StrVal("Rename-Fehler: " + err.Error()),
				}}
			}

			// --- Optional: Original löschen (NUR nach erfolgreichem Schreiben) ---
			if deleteSource {
				if err := os.Remove(srcPath); err != nil {
					// Verschlüsselung war erfolgreich, nur das Löschen schlug fehl.
					// Das melden wir als Teilerfolg, nicht als kompletten Fehler.
					return Value{Kind: KindArr, Arr: []Value{
						BoolVal(true),
						StrVal("OK, aber Original konnte nicht gelöscht werden: " + err.Error()),
					}}
				}
			}

			return Value{Kind: KindArr, Arr: []Value{
				BoolVal(true),
				StrVal("OK"),
			}}
		})

	Register(ns+"AESDecryptFileToString", "crypt", "sourcePath, pass",
		"Entschlüsselt eine mit crypt.AESEncryptFile verschlüsselte Datei direkt im Speicher und gibt den Inhalt zurück. Die Datei bleibt dabei verschlüsselt auf der Platte - es wird zu keinem Zeitpunkt eine Klartext-Version geschrieben. Rückgabe: [OK, Content, Msg]",
		func(args []Value) Value {

			if len(args) < 2 {
				return Value{Kind: KindArr, Arr: []Value{
					BoolVal(false),
					StrVal(""),
					StrVal("AESDecryptFileToString benötigt sourcePath und Passwort"),
				}}
			}

			srcPath, errVal := absPathVal(args[0].Str)
			if errVal != nil {
				return Value{Kind: KindArr, Arr: []Value{
					BoolVal(false),
					StrVal(""),
					StrVal("Pfad-Fehler: " + errVal.Str),
				}}
			}
			pass := args[1].Str

			data, err := os.ReadFile(srcPath)
			if err != nil {
				return Value{Kind: KindArr, Arr: []Value{
					BoolVal(false),
					StrVal(""),
					StrVal("Datei-Lesefehler: " + err.Error()),
				}}
			}

			key := sha256.Sum256([]byte(pass))

			block, err := aes.NewCipher(key[:])
			if err != nil {
				return Value{Kind: KindArr, Arr: []Value{
					BoolVal(false),
					StrVal(""),
					StrVal("Cipher Fehler: " + err.Error()),
				}}
			}

			gcm, err := cipher.NewGCM(block)
			if err != nil {
				return Value{Kind: KindArr, Arr: []Value{
					BoolVal(false),
					StrVal(""),
					StrVal("GCM Fehler: " + err.Error()),
				}}
			}

			nonceSize := gcm.NonceSize()
			if len(data) < nonceSize {
				return Value{Kind: KindArr, Arr: []Value{
					BoolVal(false),
					StrVal(""),
					StrVal("Datei ist zu kurz oder beschädigt (kein gültiger Nonce-Header)"),
				}}
			}

			nonce, ciphertext := data[:nonceSize], data[nonceSize:]
			plain, err := gcm.Open(nil, nonce, ciphertext, nil)
			if err != nil {
				return Value{Kind: KindArr, Arr: []Value{
					BoolVal(false),
					StrVal(""),
					StrVal("Entschlüsselung fehlgeschlagen (falsches Passwort oder Datei beschädigt): " + err.Error()),
				}}
			}

			content := string(plain)

			// plain-Buffer nullen, sobald die (unvermeidliche) String-Kopie
			// erzeugt ist. Der String selbst bleibt danach nur über
			// crypt.WipeString löschbar (siehe crypt.md).
			for i := range plain {
				plain[i] = 0
			}

			return Value{Kind: KindArr, Arr: []Value{
				BoolVal(true),
				StrVal(content),
				StrVal("OK"),
			}}
		})

	Register(ns+"AESDecryptFile", "crypt", "sourcePath, destPath, pass [, deleteSource]",
		"Entschlüsselt eine mit crypt.AESEncryptFile verschlüsselte Datei und speichert sie unter destPath. Mit deleteSource=true wird die verschlüsselte Quelldatei nach erfolgreicher Entschlüsselung gelöscht (Standard: false). Rückgabe: [OK, Msg]",
		func(args []Value) Value {

			if len(args) < 3 {
				return Value{Kind: KindArr, Arr: []Value{
					BoolVal(false),
					StrVal("AESDecryptFile benötigt sourcePath, destPath und Passwort"),
				}}
			}

			srcPath, errVal := absPathVal(args[0].Str)
			if errVal != nil {
				return Value{Kind: KindArr, Arr: []Value{
					BoolVal(false),
					StrVal("Pfad-Fehler (sourcePath): " + errVal.Str),
				}}
			}
			destPath, errVal := absPathVal(args[1].Str)
			if errVal != nil {
				return Value{Kind: KindArr, Arr: []Value{
					BoolVal(false),
					StrVal("Pfad-Fehler (destPath): " + errVal.Str),
				}}
			}
			pass := args[2].Str

			deleteSource := false
			if len(args) > 3 && args[3].Kind == KindBool {
				deleteSource = args[3].Bool
			}

			data, err := os.ReadFile(srcPath)
			if err != nil {
				return Value{Kind: KindArr, Arr: []Value{
					BoolVal(false),
					StrVal("Datei-Lesefehler: " + err.Error()),
				}}
			}

			key := sha256.Sum256([]byte(pass))

			block, err := aes.NewCipher(key[:])
			if err != nil {
				return Value{Kind: KindArr, Arr: []Value{
					BoolVal(false),
					StrVal("Cipher Fehler: " + err.Error()),
				}}
			}

			gcm, err := cipher.NewGCM(block)
			if err != nil {
				return Value{Kind: KindArr, Arr: []Value{
					BoolVal(false),
					StrVal("GCM Fehler: " + err.Error()),
				}}
			}

			nonceSize := gcm.NonceSize()
			if len(data) < nonceSize {
				return Value{Kind: KindArr, Arr: []Value{
					BoolVal(false),
					StrVal("Datei ist zu kurz oder beschädigt (kein gültiger Nonce-Header)"),
				}}
			}

			nonce, ciphertext := data[:nonceSize], data[nonceSize:]
			plain, err := gcm.Open(nil, nonce, ciphertext, nil)
			if err != nil {
				return Value{Kind: KindArr, Arr: []Value{
					BoolVal(false),
					StrVal("Entschlüsselung fehlgeschlagen (falsches Passwort oder Datei beschädigt): " + err.Error()),
				}}
			}
			defer func() {
				for i := range plain {
					plain[i] = 0
				}
			}()

			tmpPath := destPath + ".tmp"
			if err := os.WriteFile(tmpPath, plain, 0600); err != nil {
				return Value{Kind: KindArr, Arr: []Value{
					BoolVal(false),
					StrVal("Schreibfehler: " + err.Error()),
				}}
			}
			if err := os.Rename(tmpPath, destPath); err != nil {
				return Value{Kind: KindArr, Arr: []Value{
					BoolVal(false),
					StrVal("Rename-Fehler: " + err.Error()),
				}}
			}

			// --- Optional: verschlüsselte Quelldatei löschen ---
			if deleteSource {
				if err := os.Remove(srcPath); err != nil {
					return Value{Kind: KindArr, Arr: []Value{
						BoolVal(true),
						StrVal("OK, aber verschlüsseltes Original konnte nicht gelöscht werden: " + err.Error()),
					}}
				}
			}

			return Value{Kind: KindArr, Arr: []Value{
				BoolVal(true),
				StrVal("OK"),
			}}
		})

	// crypt.CheckPasswordPolicy
	Register(ns+"CheckPassword", "crypt",
		"password [, minLength] [, maxLength] [, requireUpper] [, requireLower] [, requireDigit] [, requireSpecial]",
		"Prüft ein vom Nutzer gewähltes Passwort gegen konfigurierbare Regeln (Mindestlänge, Groß-/Kleinschreibung, Ziffern, Sonderzeichen). Gibt eine Map mit Detail-Ergebnissen zurück, statt das Passwort blind zu übernehmen.",
		func(args []Value) Value {
			if len(args) < 1 {
				return ErrorVal("crypt.CheckPasswordPolicy erwartet mindestens 1 Argument (password)")
			}
			if args[0].Kind != KindStr {
				return ErrorVal("crypt.CheckPasswordPolicy: 'password' muss ein String sein")
			}
			pw := args[0].Str

			// Defaults, falls nicht angegeben
			minLength := 8
			maxLength := 128
			requireUpper := true
			requireLower := true
			requireDigit := true
			requireSpecial := true

			if len(args) > 1 && args[1].Kind == KindNum {
				minLength = int(args[1].Num)
			}
			if len(args) > 2 && args[2].Kind == KindNum {
				maxLength = int(args[2].Num)
			}
			if len(args) > 3 && args[3].Kind == KindBool {
				requireUpper = args[3].Bool
			}
			if len(args) > 4 && args[4].Kind == KindBool {
				requireLower = args[4].Bool
			}
			if len(args) > 5 && args[5].Kind == KindBool {
				requireDigit = args[5].Bool
			}
			if len(args) > 6 && args[6].Kind == KindBool {
				requireSpecial = args[6].Bool
			}

			runeCount := len([]rune(pw))
			hasUpper := false
			hasLower := false
			hasDigit := false
			hasSpecial := false

			for _, r := range pw {
				switch {
				case unicode.IsUpper(r):
					hasUpper = true
				case unicode.IsLower(r):
					hasLower = true
				case unicode.IsDigit(r):
					hasDigit = true
				case unicode.IsPunct(r) || unicode.IsSymbol(r):
					hasSpecial = true
				}
			}

			var errs []Value

			if runeCount < minLength {
				errs = append(errs, StrVal(fmt.Sprintf("Passwort ist zu kurz (mindestens %d Zeichen erforderlich, hat %d)", minLength, runeCount)))
			}
			if runeCount > maxLength {
				errs = append(errs, StrVal(fmt.Sprintf("Passwort ist zu lang (maximal %d Zeichen erlaubt, hat %d)", maxLength, runeCount)))
			}
			if requireUpper && !hasUpper {
				errs = append(errs, StrVal("Mindestens ein Großbuchstabe erforderlich"))
			}
			if requireLower && !hasLower {
				errs = append(errs, StrVal("Mindestens ein Kleinbuchstabe erforderlich"))
			}
			if requireDigit && !hasDigit {
				errs = append(errs, StrVal("Mindestens eine Ziffer erforderlich"))
			}
			if requireSpecial && !hasSpecial {
				errs = append(errs, StrVal("Mindestens ein Sonderzeichen erforderlich"))
			}

			result := map[string]Value{
				"Valid":      BoolVal(len(errs) == 0),
				"Length":     NumVal(float64(runeCount)),
				"HasUpper":   BoolVal(hasUpper),
				"HasLower":   BoolVal(hasLower),
				"HasDigit":   BoolVal(hasDigit),
				"HasSpecial": BoolVal(hasSpecial),
				"Errors":     Value{Kind: KindArr, Arr: errs},
			}

			return Value{Kind: KindMap, Map: result}
		})
}
