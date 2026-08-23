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
	"path/filepath"

	circlkem "github.com/cloudflare/circl/kem/mlkem/mlkem768"
	"github.com/cloudflare/circl/sign/mldsa/mldsa65"
	"golang.org/x/crypto/argon2"
)

// =============================================================================
// HELPER-FUNKTIONEN
// =============================================================================

// keypairResult gibt ein 4-Element-Array [bool, pubB64, privB64, errMsg] zurück.
// Wird von GenerateKeyPair, GenerateSigKeyPair und Encapsulate genutzt.
func keypairResult(pub, priv string) Value {
	res := make([]Value, 4)
	res[0] = BoolVal(true)
	res[1] = StrVal(pub)
	res[2] = StrVal(priv)
	res[3] = StrVal("")
	return Value{Kind: KindArr, Arr: res}
}

func keypairErr(msg string) Value {
	res := make([]Value, 4)
	res[0] = BoolVal(false)
	res[1] = StrVal("")
	res[2] = StrVal("")
	res[3] = StrVal(msg)
	return Value{Kind: KindArr, Arr: res}
}

// decodeB64 dekodiert einen Base64-String und gibt einen sprechenden Fehler zurück.
func decodeB64(s, label string) ([]byte, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("%s: ungültiges Base64", label)
	}
	return b, nil
}

// encryptAndSaveKey verschlüsselt einen privaten Schlüssel (als Base64-String)
// mit AES-256-GCM + Argon2id und schreibt ihn atomar in absPath.
// Ist password == "", wird ein sicheres Zufallspasswort generiert und zurückgegeben.
// Das Passwort wird NIE automatisch auf der Festplatte gespeichert —
// das ist Aufgabe des Aufrufers.
func encryptAndSaveKey(privKeyB64, absPath, password string) (string, error) {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789!@#$%^&*"

	if password == "" {
		passBytes := make([]byte, 25)
		for i := range passBytes {
			n, err := crand.Int(crand.Reader, big.NewInt(int64(len(charset))))
			if err != nil {
				return "", fmt.Errorf("passwort-generierung fehlgeschlagen: %w", err)
			}
			passBytes[i] = charset[n.Int64()]
		}
		password = string(passBytes)
	}

	salt := make([]byte, 16)
	if _, err := io.ReadFull(crand.Reader, salt); err != nil {
		return "", fmt.Errorf("salt-generierung fehlgeschlagen: %w", err)
	}

	key := argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, 32)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("cipher-initialisierung fehlgeschlagen: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("GCM-initialisierung fehlgeschlagen: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(crand.Reader, nonce); err != nil {
		return "", fmt.Errorf("nonce-generierung fehlgeschlagen: %w", err)
	}

	aad := []byte("PQC-KEYSTORE-v1")
	ciphertext := gcm.Seal(nil, nonce, []byte(privKeyB64), aad)

	// Format: magic(4) | salt(16) | nonce(n) | ciphertext
	out := make([]byte, 0, 4+len(salt)+len(nonce)+len(ciphertext))
	out = append(out, []byte("PQC1")...)
	out = append(out, salt...)
	out = append(out, nonce...)
	out = append(out, ciphertext...)

	// Atomar schreiben: tmp → rename
	tmpFile, err := os.CreateTemp(
		filepath.Dir(absPath),
		filepath.Base(absPath)+".tmp-*",
	)
	if err != nil {
		return "", fmt.Errorf("temp-datei konnte nicht erstellt werden: %w", err)
	}
	//defer tmpFile.Close()

	if _, err := tmpFile.Write(out); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("schreiben fehlgeschlagen: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("sync fehlgeschlagen: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("close fehlgeschlagen: %w", err)
	}

	if err := os.Rename(tmpFile.Name(), absPath); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("umbenennen fehlgeschlagen: %w", err)
	}

	return password, nil
}

// decryptFile liest eine mit encryptAndSaveKey erzeugte Datei und entschlüsselt sie.
func decryptFile(absPath, password string) (string, error) {
	blob, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("datei nicht lesbar: %w", err)
	}

	if len(blob) < 4 || string(blob[:4]) != "PQC1" {
		return "", fmt.Errorf("unbekanntes dateiformat (kein PQC1-Header)")
	}
	blob = blob[4:]

	if len(blob) < 16 {
		return "", fmt.Errorf("datei korrupt: zu kurz für salt")
	}
	salt := blob[:16]
	blob = blob[16:]

	key := argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, 32)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("cipher-initialisierung fehlgeschlagen: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("GCM-initialisierung fehlgeschlagen: %w", err)
	}

	ns := gcm.NonceSize()
	if len(blob) < ns {
		return "", fmt.Errorf("datei korrupt: zu kurz für nonce")
	}

	plaintext, err := gcm.Open(nil, blob[:ns], blob[ns:], []byte("PQC-KEYSTORE-v1"))
	if err != nil {
		return "", fmt.Errorf("passwort falsch oder datei manipuliert")
	}

	return string(plaintext), nil
}

// hashFileSHA256 berechnet den SHA-256-Hash einer Datei als []byte.
func hashFileSHA256(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("datei nicht lesbar: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, fmt.Errorf("hash-berechnung fehlgeschlagen: %w", err)
	}

	return h.Sum(nil), nil
}

// printPasswordWarning gibt die Passwort-Hinweisbox auf der Konsole aus.
func printPasswordWarning(keyPath string) {
	fmt.Println("╔══════════════════════════════════════════════════╗")
	fmt.Println("║         WICHTIG: MASTER-PASSWORT                ║")
	fmt.Println("╠══════════════════════════════════════════════════╣")
	fmt.Println("║  Das Passwort wurde gespeichert unter:          ║")
	fmt.Printf("║  %-49s║\n", keyPath)
	fmt.Println("╠══════════════════════════════════════════════════╣")
	fmt.Println("║  1. Öffne die Datei und notiere das Passwort.   ║")
	fmt.Println("║  2. Bewahre es sicher auf (z.B. Passwortmanager)║")
	fmt.Println("║  3. Lösche die Datei anschließend.              ║")
	fmt.Println("║                                                  ║")
	fmt.Println("║  Ohne dieses Passwort ist der Key VERLOREN.     ║")
	fmt.Println("╚══════════════════════════════════════════════════╝")
}

// printUnencryptedWarning warnt vor einem ungeschützten privaten Schlüssel.
func printUnencryptedWarning(filename string) {
	fmt.Println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
	fmt.Println("⚠️  SICHERHEITSWARNUNG: PRIVATE KEY UNVERSCHLÜSSELT!  ⚠️")
	fmt.Printf(" Die Datei %s ist NICHT geschützt. Bitte verschlüsseln!\n", filename)
	fmt.Println(" Jeder der Zugriff auf diese Datei hat, kann alles lesen!")
	fmt.Println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
}

// setupIdentityFiles kapselt die gemeinsame Logik von SetupIdentity und
// SetupSignIdentity: Ordner anlegen, Key speichern, Passwort-Datei schreiben.
func setupIdentityFiles(absFolder, privB64, pubB64, keyName string, shouldEncrypt bool) error {
	if err := os.MkdirAll(absFolder, 0700); err != nil {
		return fmt.Errorf("ordner konnte nicht erstellt werden: %w", err)
	}

	// Public Key immer speichern
	pubPath := filepath.Join(absFolder, keyName+".pub")
	if err := os.WriteFile(pubPath, []byte(pubB64), 0644); err != nil {
		return fmt.Errorf("public key konnte nicht gespeichert werden: %w", err)
	}

	privPath := filepath.Join(absFolder, keyName)

	if shouldEncrypt {
		keyFilePath := filepath.Join(absFolder, keyName+".key")

		pass, err := encryptAndSaveKey(privB64, privPath, "")
		if err != nil {
			os.Remove(pubPath) // Rollback
			return err
		}

		// Passwort in .key-Datei ablegen (Nutzer wird aufgefordert, sie zu löschen)
		if err := os.WriteFile(keyFilePath, []byte(pass), 0600); err != nil {
			os.Remove(pubPath)
			os.Remove(privPath)
			return fmt.Errorf("passwort-datei konnte nicht geschrieben werden: %w", err)
		}

		printPasswordWarning(keyFilePath)
	} else {
		// Privaten Key unverschlüsselt speichern
		if err := os.WriteFile(privPath, []byte(privB64), 0600); err != nil {
			os.Remove(pubPath) // Rollback
			return fmt.Errorf("private key konnte nicht gespeichert werden: %w", err)
		}

		printUnencryptedWarning(keyName)
	}

	return nil
}

// =============================================================================
// REGISTRIERUNG
// =============================================================================

func InitPQCFunctions() {
	ns := "pqc."

	// ---------------------------------------------------------------------------
	// pqc.GenerateKeyPair()  →  [bool, pubB64, privB64, err]
	// ---------------------------------------------------------------------------
	Register(ns+"GenerateKeyPair", "pqc", "-",
		"Erzeugt ein ML-KEM-768 Schlüsselpaar. Rückgabe: [pubB64, privB64].",
		func(args []Value) Value {
			pub, priv, err := circlkem.GenerateKeyPair(nil)
			if err != nil {
				return ErrorVal("Schlüsselgenerierung fehlgeschlagen: " + err.Error())
			}
			pubBytes, err := pub.MarshalBinary()
			if err != nil {
				return ErrorVal("Public Key serialisierung fehlgeschlagen: " + err.Error())
			}
			privBytes, err := priv.MarshalBinary()
			if err != nil {
				return ErrorVal("Private Key serialisierung fehlgeschlagen: " + err.Error())
			}
			return Value{Kind: KindArr, Arr: []Value{
				StrVal(base64.StdEncoding.EncodeToString(pubBytes)),
				StrVal(base64.StdEncoding.EncodeToString(privBytes)),
			}}
		})

	// ---------------------------------------------------------------------------
	// pqc.GenerateSigKeyPair()  →  [bool, pubB64, privB64, err]
	// ---------------------------------------------------------------------------
	Register(ns+"GenerateSigKeyPair", "pqc", "-",
		"Erzeugt ein ML-DSA-65 Signatur-Schlüsselpaar. Rückgabe: [pubB64, privB64].",
		func(args []Value) Value {
			scheme := mldsa65.Scheme()
			pk, sk, err := scheme.GenerateKey()
			if err != nil {
				return ErrorVal("Schlüsselgenerierung fehlgeschlagen: " + err.Error())
			}
			pubBytes, err := pk.MarshalBinary()
			if err != nil {
				return ErrorVal("Public Key serialisierung fehlgeschlagen: " + err.Error())
			}
			privBytes, err := sk.MarshalBinary()
			if err != nil {
				return ErrorVal("Private Key serialisierung fehlgeschlagen: " + err.Error())
			}
			return Value{Kind: KindArr, Arr: []Value{
				StrVal(base64.StdEncoding.EncodeToString(pubBytes)),
				StrVal(base64.StdEncoding.EncodeToString(privBytes)),
			}}
		})

	// ---------------------------------------------------------------------------
	// pqc.Encapsulate(pubKeyB64)  →  [bool, ciphertextB64, sharedSecretB64, err]
	// ---------------------------------------------------------------------------
	Register(ns+"Encapsulate", "pqc", "pubKeyB64",
		"Erzeugt ein Shared Secret für einen ML-KEM Public Key. Rückgabe: [ciphertextB64, sharedSecretB64].",
		func(args []Value) Value {
			if len(args) < 1 {
				return ErrorVal("Parameter fehlt: pqc.Encapsulate(pubKeyB64)")
			}
			pubBytes, err := decodeB64(args[0].Str, "Public Key")
			if err != nil {
				return ErrorVal(err.Error())
			}
			scheme := circlkem.Scheme()
			pubKey, err := scheme.UnmarshalBinaryPublicKey(pubBytes)
			if err != nil {
				return ErrorVal("Public Key ungültig: " + err.Error())
			}
			ciphertext, sharedSecret, err := scheme.Encapsulate(pubKey)
			if err != nil {
				return ErrorVal("Encapsulate fehlgeschlagen: " + err.Error())
			}
			return Value{Kind: KindArr, Arr: []Value{
				StrVal(base64.StdEncoding.EncodeToString(ciphertext)),
				StrVal(base64.StdEncoding.EncodeToString(sharedSecret)),
			}}
		})

	// ---------------------------------------------------------------------------
	// pqc.Decapsulate(ciphertextB64, privKeyB64)  →  [bool, sharedSecretB64, err]
	// ---------------------------------------------------------------------------
	Register(ns+"Decapsulate", "pqc", "ciphertextB64, privKeyB64",
		"Entkapselt ein Shared Secret mit einem ML-KEM Private Key.",
		func(args []Value) Value {
			if len(args) < 2 {
				return ErrorVal("Parameter fehlen: pqc.Decapsulate(ciphertextB64, privKeyB64)")
			}
			ctBytes, err := decodeB64(args[0].Str, "Ciphertext")
			if err != nil {
				return ErrorVal(err.Error())
			}
			skBytes, err := decodeB64(args[1].Str, "Private Key")
			if err != nil {
				return ErrorVal(err.Error())
			}
			scheme := circlkem.Scheme()
			sk, err := scheme.UnmarshalBinaryPrivateKey(skBytes)
			if err != nil {
				return ErrorVal("Private Key ungültig: " + err.Error())
			}
			ss, err := scheme.Decapsulate(sk, ctBytes)
			if err != nil {
				return ErrorVal("Decapsulate fehlgeschlagen: " + err.Error())
			}
			return StrVal(base64.StdEncoding.EncodeToString(ss))
		})

	// ---------------------------------------------------------------------------
	// pqc.Sign(msg, privKeyB64)  →  [bool, signatureB64, err]
	// ---------------------------------------------------------------------------
	Register(ns+"Sign", "pqc", "msg, privKeyB64",
		"Signiert eine Nachricht mit ML-DSA-65.",
		func(args []Value) Value {
			if len(args) < 2 {
				return ErrorVal("Parameter fehlen: pqc.Sign(msg, privKeyB64)")
			}
			skBytes, err := decodeB64(args[1].Str, "Private Key")
			if err != nil {
				return ErrorVal(err.Error())
			}
			scheme := mldsa65.Scheme()
			sk, err := scheme.UnmarshalBinaryPrivateKey(skBytes)
			if err != nil {
				return ErrorVal("Private Key ungültig: " + err.Error())
			}
			sig := scheme.Sign(sk, []byte(args[0].Str), nil)
			return StrVal(base64.StdEncoding.EncodeToString(sig))
		})

	// ---------------------------------------------------------------------------
	// pqc.Verify(msg, sigB64, pubKeyB64)  →  [bool, err]
	// ---------------------------------------------------------------------------
	Register(ns+"Verify", "pqc", "msg, sigB64, pubKeyB64",
		"Prüft eine ML-DSA-65 Signatur gegen eine Nachricht.",
		func(args []Value) Value {
			if len(args) < 3 {
				return ErrorVal("Parameter fehlen: pqc.Verify(msg, sigB64, pubKeyB64)")
			}
			sig, err := decodeB64(args[1].Str, "Signatur")
			if err != nil {
				return ErrorVal(err.Error())
			}
			pubBytes, err := decodeB64(args[2].Str, "Public Key")
			if err != nil {
				return ErrorVal(err.Error())
			}
			scheme := mldsa65.Scheme()
			pk, err := scheme.UnmarshalBinaryPublicKey(pubBytes)
			if err != nil {
				return ErrorVal("Public Key ungültig: " + err.Error())
			}
			if !scheme.Verify(pk, []byte(args[0].Str), sig, nil) {
				return ErrorVal("Signatur ungültig")
			}
			return BoolVal(true)
		})

	// ---------------------------------------------------------------------------
	// pqc.SignFile(filePath, privKeyB64)  →  [bool, signatureB64, err]
	// ---------------------------------------------------------------------------
	Register(ns+"SignFile", "pqc", "filePath, privKeyB64",
		"Signiert eine Datei (über ihren SHA-256-Hash) mit ML-DSA-65.",
		func(args []Value) Value {
			if len(args) < 2 {
				return ErrorVal("Parameter fehlen: pqc.SignFile(filePath, privKeyB64)")
			}
			absPath, errVal := absPathVal(args[0].Str)
			if errVal != nil {
				return ErrorVal(fmt.Sprintf("ungültiger Pfad '%s'", args[0].Str))
			}
			fileHash, err := hashFileSHA256(absPath)
			if err != nil {
				return ErrorVal(err.Error())
			}
			skBytes, err := decodeB64(args[1].Str, "Private Key")
			if err != nil {
				return ErrorVal(err.Error())
			}
			scheme := mldsa65.Scheme()
			sk, err := scheme.UnmarshalBinaryPrivateKey(skBytes)
			if err != nil {
				return ErrorVal("Private Key ungültig: " + err.Error())
			}
			sig := scheme.Sign(sk, fileHash, nil)
			return StrVal(base64.StdEncoding.EncodeToString(sig))
		})

	// ---------------------------------------------------------------------------
	// pqc.VerifyFile(filePath, sigB64, pubKeyB64)  →  [bool, status, err]
	// ---------------------------------------------------------------------------
	Register(ns+"VerifyFile", "pqc", "filePath, sigB64, pubKeyB64",
		"Verifiziert eine Datei gegen eine ML-DSA-65 Signatur.",
		func(args []Value) Value {
			if len(args) < 3 {
				return ErrorVal("Parameter fehlen: pqc.VerifyFile(filePath, sigB64, pubKeyB64)")
			}
			absPath, errVal := absPathVal(args[0].Str)
			if errVal != nil {
				return ErrorVal(fmt.Sprintf("ungültiger Pfad '%s'", args[0].Str))
			}
			fileHash, err := hashFileSHA256(absPath)
			if err != nil {
				return ErrorVal(err.Error())
			}
			sigBytes, err := decodeB64(args[1].Str, "Signatur")
			if err != nil {
				return ErrorVal(err.Error())
			}
			pkBytes, err := decodeB64(args[2].Str, "Public Key")
			if err != nil {
				return ErrorVal(err.Error())
			}
			scheme := mldsa65.Scheme()
			pk, err := scheme.UnmarshalBinaryPublicKey(pkBytes)
			if err != nil {
				return ErrorVal("Public Key ungültig: " + err.Error())
			}
			if !scheme.Verify(pk, fileHash, sigBytes, nil) {
				return ErrorVal("Signatur stimmt nicht mit Datei überein")
			}
			return BoolVal(true)
		})

	// ---------------------------------------------------------------------------
	// pqc.SetupIdentity(folder, [encrypt])  →  str "OK"
	// ---------------------------------------------------------------------------
	Register(ns+"SetupIdentity", "pqc", "folder, [encrypt]",
		"Erzeugt ein ML-KEM-768 Schlüsselpaar und speichert es im Ordner.",
		func(args []Value) Value {
			if len(args) < 1 {
				return ErrorVal("Parameter fehlen: pqc.SetupIdentity(folder, [encrypt])")
			}

			absFolder, errVal := absPathVal(args[0].Str)
			if errVal != nil {
				return ErrorVal(fmt.Sprintf("ungültiger Pfad '%s'", args[0].Str))
			}

			shouldEncrypt := true
			if len(args) >= 2 {
				shouldEncrypt = ToBool(args[1])
			}

			pub, priv, err := circlkem.GenerateKeyPair(nil)
			if err != nil {
				return ErrorVal("Schlüsselgenerierung fehlgeschlagen: " + err.Error())
			}

			pubBytes, err := pub.MarshalBinary()
			if err != nil {
				return ErrorVal("Public Key serialisierung fehlgeschlagen: " + err.Error())
			}

			privBytes, err := priv.MarshalBinary()
			if err != nil {
				return ErrorVal("Private Key serialisierung fehlgeschlagen: " + err.Error())
			}

			pubB64 := base64.StdEncoding.EncodeToString(pubBytes)
			privB64 := base64.StdEncoding.EncodeToString(privBytes)

			if err := setupIdentityFiles(absFolder, privB64, pubB64, "id_pqc", shouldEncrypt); err != nil {
				return ErrorVal(err.Error())
			}

			return StrVal("OK")
		})

	// ---------------------------------------------------------------------------
	// pqc.SetupSignIdentity(folder, [encrypt])  →  str "OK"
	// ---------------------------------------------------------------------------
	Register(ns+"SetupSignIdentity", "pqc", "folder, [encrypt]",
		"Erzeugt ein ML-DSA-65 Signatur-Schlüsselpaar und speichert es im Ordner.",
		func(args []Value) Value {
			if len(args) < 1 {
				return ErrorVal("Parameter fehlen: pqc.SetupSignIdentity(folder, [encrypt])")
			}

			absFolder, errVal := absPathVal(args[0].Str)
			if errVal != nil {
				return ErrorVal(fmt.Sprintf("ungültiger Pfad '%s'", args[0].Str))
			}

			shouldEncrypt := true
			if len(args) >= 2 {
				shouldEncrypt = ToBool(args[1])
			}

			scheme := mldsa65.Scheme()
			pk, sk, err := scheme.GenerateKey()
			if err != nil {
				return ErrorVal("Schlüsselgenerierung fehlgeschlagen: " + err.Error())
			}

			pubBytes, err := pk.MarshalBinary()
			if err != nil {
				return ErrorVal("Public Key serialisierung fehlgeschlagen: " + err.Error())
			}

			privBytes, err := sk.MarshalBinary()
			if err != nil {
				return ErrorVal("Private Key serialisierung fehlgeschlagen: " + err.Error())
			}

			pubB64 := base64.StdEncoding.EncodeToString(pubBytes)
			privB64 := base64.StdEncoding.EncodeToString(privBytes)

			if err := setupIdentityFiles(absFolder, privB64, pubB64, "id_sig", shouldEncrypt); err != nil {
				return ErrorVal(err.Error())
			}

			return StrVal("OK")
		})

	// ---------------------------------------------------------------------------
	// pqc.ExportKey(privKeyB64, path, [password])  →  str password
	// ---------------------------------------------------------------------------
	Register(ns+"ExportKey", "pqc", "privKeyB64, path, [password]",
		"Exportiert einen ML-KEM Private Key verschlüsselt in eine Datei.",
		func(args []Value) Value {
			if len(args) < 2 {
				return ErrorVal("Parameter fehlen: pqc.ExportKey(privKeyB64, path, [password])")
			}

			absP, errVal := absPathVal(args[1].Str)
			if errVal != nil {
				return ErrorVal(fmt.Sprintf("ungültiger Pfad '%s'", args[1].Str))
			}

			password := ""
			if len(args) >= 3 {
				password = args[2].Str
			}

			if password == "" {
				// Kein Passwort → Klartext-Export
				if err := os.WriteFile(absP, []byte(args[0].Str), 0600); err != nil {
					return ErrorVal("Schreibfehler: " + err.Error())
				}
				return StrVal("")
			}

			usedPassword, err := encryptAndSaveKey(args[0].Str, absP, password)
			if err != nil {
				return ErrorVal(err.Error())
			}

			return StrVal(usedPassword)
		})

	// ---------------------------------------------------------------------------
	// pqc.ImportKey(path, [password])  →  str privKeyB64
	// ---------------------------------------------------------------------------
	Register(ns+"ImportKey", "pqc", "path, [password]",
		"Lädt einen ML-KEM Private Key aus einer Datei (optional entschlüsselt).",
		func(args []Value) Value {
			if len(args) < 1 {
				return ErrorVal("Parameter fehlt: pqc.ImportKey(path, [password])")
			}

			absP, errVal := absPathVal(args[0].Str)
			if errVal != nil {
				return ErrorVal(fmt.Sprintf("ungültiger Pfad '%s'", args[0].Str))
			}

			if len(args) >= 2 && args[1].Str != "" {
				decrypted, err := decryptFile(absP, args[1].Str)
				if err != nil {
					return ErrorVal(err.Error())
				}
				return StrVal(decrypted)
			}

			b, err := os.ReadFile(absP)
			if err != nil {
				return ErrorVal("Datei nicht lesbar: " + err.Error())
			}

			return StrVal(string(b))
		})

	// ---------------------------------------------------------------------------
	// pqc.ExportSignKey(privKeyB64, path, [password])  →  str password
	// ---------------------------------------------------------------------------
	Register(ns+"ExportSignKey", "pqc", "privKeyB64, path, [password]",
		"Exportiert einen ML-DSA Private Key verschlüsselt in eine Datei.",
		func(args []Value) Value {
			if len(args) < 2 {
				return ErrorVal("Parameter fehlen: pqc.ExportSignKey(privKeyB64, path, [password])")
			}

			absP, errVal := absPathVal(args[1].Str)
			if errVal != nil {
				return ErrorVal(fmt.Sprintf("ungültiger Pfad '%s'", args[1].Str))
			}

			password := ""
			if len(args) >= 3 {
				password = args[2].Str
			}

			usedPassword, err := encryptAndSaveKey(args[0].Str, absP, password)
			if err != nil {
				return ErrorVal(err.Error())
			}

			return StrVal(usedPassword)
		})

	// ---------------------------------------------------------------------------
	// pqc.ImportSignKey(path, [password])  →  str privKeyB64
	// ---------------------------------------------------------------------------
	Register(ns+"ImportSignKey", "pqc", "path, [password]",
		"Lädt einen ML-DSA Private Key aus einer Datei (optional entschlüsselt).",
		func(args []Value) Value {
			if len(args) < 1 {
				return ErrorVal("Parameter fehlt: pqc.ImportSignKey(path, [password])")
			}

			absP, errVal := absPathVal(args[0].Str)
			if errVal != nil {
				return ErrorVal(fmt.Sprintf("ungültiger Pfad '%s'", args[0].Str))
			}

			if len(args) >= 2 && args[1].Str != "" {
				decrypted, err := decryptFile(absP, args[1].Str)
				if err != nil {
					return ErrorVal(err.Error())
				}
				return StrVal(decrypted)
			}

			b, err := os.ReadFile(absP)
			if err != nil {
				return ErrorVal("Datei nicht lesbar: " + err.Error())
			}

			return StrVal(string(b))
		})
}
