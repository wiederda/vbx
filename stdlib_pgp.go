package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
)

// =============================================================================
// HELPER-FUNKTIONEN
// =============================================================================

// decryptEntity entschlüsselt den Hauptschlüssel UND alle Subkeys einer Entity.
// Ist der Key unverschlüsselt und pass == "", wird kein Fehler zurückgegeben.
func decryptEntity(entity *openpgp.Entity, pass string) error {
	if entity.PrivateKey == nil {
		return fmt.Errorf("entity hat keinen privaten Schlüssel")
	}

	if entity.PrivateKey.Encrypted {
		if pass == "" {
			return fmt.Errorf("schlüssel ist verschlüsselt, aber kein Passwort angegeben")
		}
		if err := entity.PrivateKey.Decrypt([]byte(pass)); err != nil {
			return fmt.Errorf("falsches Passwort für Hauptschlüssel")
		}
	}

	for i, sub := range entity.Subkeys {
		if sub.PrivateKey != nil && sub.PrivateKey.Encrypted {
			if err := entity.Subkeys[i].PrivateKey.Decrypt([]byte(pass)); err != nil {
				// Subkey-Fehler sind nicht immer fatal (z.B. unterschiedliche Passwörter
				// in alten Key-Formaten), aber wir loggen sie zur Sicherheit.
				fmt.Printf("Warnung: Subkey %d konnte nicht entschlüsselt werden: %v\n", i, err)
			}
		}
	}

	return nil
}

// encryptEntity verschlüsselt Hauptschlüssel und alle Subkeys mit einem neuen Passwort.
// Ist pass == "", bleibt die Entity unverschlüsselt (erlaubt, wird gewarnt).
func encryptEntity(entity *openpgp.Entity, pass string) error {
	if pass == "" {
		return nil
	}

	if err := entity.PrivateKey.Encrypt([]byte(pass)); err != nil {
		return fmt.Errorf("fehler beim Verschlüsseln des Hauptschlüssels: %w", err)
	}

	for i, sub := range entity.Subkeys {
		if sub.PrivateKey != nil {
			if err := entity.Subkeys[i].PrivateKey.Encrypt([]byte(pass)); err != nil {
				return fmt.Errorf("fehler beim Verschlüsseln von Subkey %d: %w", i, err)
			}
		}
	}

	return nil
}

// parseArmoredKey liest eine EntityList aus einem Armor-String oder einem Dateipfad.
func parseArmoredKey(input string) (openpgp.EntityList, error) {
	var r io.Reader

	// Armor-String direkt verwenden
	if strings.Contains(input, "-----BEGIN PGP") {
		r = strings.NewReader(input)
	} else {
		// Pfad-Behandlung
		absP, errVal := absPathVal(input)
		if errVal != nil {
			return nil, fmt.Errorf("ungültiger Pfad: %s", input)
		}
		data, err := os.ReadFile(absP)
		if err != nil {
			return nil, fmt.Errorf("datei nicht lesbar: %w", err)
		}
		r = bytes.NewReader(data)
	}

	entities, err := openpgp.ReadArmoredKeyRing(r)
	if err != nil {
		return nil, fmt.Errorf("ungültiges PGP-Key-Format: %w", err)
	}
	if len(entities) == 0 {
		return nil, fmt.Errorf("keine Schlüssel in der Eingabe gefunden")
	}

	return entities, nil
}

// serializePrivateKey serialisiert einen Entity als Armor-String (im Speicher).
// Der Aufrufer ist verantwortlich, den String nach Verwendung zu verwerfen.
func serializePrivateKey(entity *openpgp.Entity) (string, error) {
	buf := new(bytes.Buffer)
	w, err := armor.Encode(buf, openpgp.PrivateKeyType, nil)
	if err != nil {
		return "", fmt.Errorf("armor-encoder konnte nicht erstellt werden: %w", err)
	}

	if err := entity.SerializePrivate(w, nil); err != nil {
		w.Close()
		return "", fmt.Errorf("serialisierung fehlgeschlagen: %w", err)
	}
	w.Close()

	return buf.String(), nil
}

// serializePublicKey serialisiert nur den Public-Key-Teil einer Entity.
func serializePublicKey(entity *openpgp.Entity) (string, error) {
	var buf strings.Builder
	w, err := armor.Encode(&buf, openpgp.PublicKeyType, nil)
	if err != nil {
		return "", fmt.Errorf("armor-encoder konnte nicht erstellt werden: %w", err)
	}

	if err := entity.Serialize(w); err != nil {
		w.Close()
		return "", fmt.Errorf("serialisierung fehlgeschlagen: %w", err)
	}
	w.Close()

	return buf.String(), nil
}

// writeKeyFile schreibt einen Armor-String atomar in eine Datei.
// Die Datei wird mit 0600 gesichert (kein world-readable).
// Atomic: zuerst in eine Temp-Datei, dann umbenennen.
func writeKeyFile(path, armorContent, armorType string, entity *openpgp.Entity) error {
	// Temp-Datei im gleichen Verzeichnis (für atomares Rename)
	tmpPath := path + ".tmp"

	f, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("datei konnte nicht erstellt werden: %w", err)
	}

	// Encoder erstellen
	w, err := armor.Encode(f, armorType, nil)
	if err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("armor-encoder fehlgeschlagen: %w", err)
	}

	// Inhalt schreiben
	var writeErr error
	if armorType == openpgp.PrivateKeyType {
		writeErr = entity.SerializePrivate(w, nil)
	} else {
		writeErr = entity.Serialize(w)
	}

	w.Close()
	f.Close()

	if writeErr != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("serialisierung fehlgeschlagen: %w", writeErr)
	}

	// Atomar umbenennen (kein Race-Window mit falschen Permissions)
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("umbenennen fehlgeschlagen: %w", err)
	}

	return nil
}

// errResult ist ein Hilfstyp für die 3-Element-Arrays [bool, string, string].
func errResult(msg string) Value {
	res := make([]Value, 3)
	res[0] = BoolVal(false)
	res[1] = StrVal("")
	res[2] = StrVal(msg)
	return Value{Kind: KindArr, Arr: res}
}

func okResult(payload string) Value {
	res := make([]Value, 3)
	res[0] = BoolVal(true)
	res[1] = StrVal(payload)
	res[2] = StrVal("")
	return Value{Kind: KindArr, Arr: res}
}

// =============================================================================
// REGISTRIERUNG
// =============================================================================

func InitPGPFunctions() {
	ns := "pgp."

	// ---------------------------------------------------------------------------
	// pgp.SetupIdentity(folder, name, email, [password])
	// ---------------------------------------------------------------------------
	Register(ns+"SetupIdentity", "pgp", "folder, name, email, [password]",
		"Erstellt ein neues PGP-Schlüsselpaar und speichert es im angegebenen Ordner.",
		func(args []Value) Value {
			if len(args) < 3 {
				return ErrorVal("Parameter fehlen: pgp.SetupIdentity(folder, name, email, [password])")
			}

			folder := args[0].Str
			name := args[1].Str
			email := args[2].Str
			pass := ""
			if len(args) >= 4 {
				pass = args[3].Str
			}

			absFolder, errVal := absPathVal(folder)
			if errVal != nil {
				return ErrorVal(fmt.Sprintf("ungültiger Pfad '%s'", folder))
			}

			if email == "" {
				email = name + "@vbx.local"
			}

			entity, err := openpgp.NewEntity(name, "vbx-generated", email, nil)
			if err != nil {
				return ErrorVal("Schlüsselgenerierung fehlgeschlagen: " + err.Error())
			}

			if err := encryptEntity(entity, pass); err != nil {
				return ErrorVal(err.Error())
			}
			if pass != "" {
				fmt.Println("PGP-Schlüssel wurde mit Passwort geschützt.")
			} else {
				fmt.Println("⚠️  WARNUNG: PGP-Privatschlüssel wurde OHNE Passwort erstellt!")
			}

			if err := os.MkdirAll(absFolder, 0700); err != nil {
				return ErrorVal("Ordner konnte nicht erstellt werden: " + err.Error())
			}

			// Public Key speichern
			pubPath := filepath.Join(absFolder, "id_pgp.pub")
			if err := writeKeyFile(pubPath, "", openpgp.PublicKeyType, entity); err != nil {
				return ErrorVal("Public Key: " + err.Error())
			}

			// Private Key speichern
			privPath := filepath.Join(absFolder, "id_pgp")
			if err := writeKeyFile(privPath, "", openpgp.PrivateKeyType, entity); err != nil {
				os.Remove(pubPath) // Rollback
				return ErrorVal("Private Key: " + err.Error())
			}

			return StrVal("OK")
		})

	// ---------------------------------------------------------------------------
	// pgp.Encrypt(msg, pubKeyPath)
	// ---------------------------------------------------------------------------
	Register(ns+"Encrypt", "pgp", "msg, pubKeyPath",
		"Verschlüsselt eine Nachricht mit einem PGP-Public-Key.",
		func(args []Value) Value {
			if len(args) < 2 {
				return ErrorVal("Parameter fehlen: pgp.Encrypt(msg, pubKeyPath)")
			}

			msg := args[0].Str
			entityList, err := parseArmoredKey(args[1].Str)
			if err != nil {
				return ErrorVal("Public Key: " + err.Error())
			}

			buf := new(bytes.Buffer)
			w, err := armor.Encode(buf, "PGP MESSAGE", nil)
			if err != nil {
				return ErrorVal("Armor-Encoder fehlgeschlagen: " + err.Error())
			}

			pw, err := openpgp.Encrypt(w, entityList, nil, nil, nil)
			if err != nil {
				w.Close()
				return ErrorVal("Verschlüsselung fehlgeschlagen: " + err.Error())
			}

			if _, err := pw.Write([]byte(msg)); err != nil {
				pw.Close()
				w.Close()
				return ErrorVal("Schreiben fehlgeschlagen: " + err.Error())
			}

			pw.Close()
			w.Close()

			return StrVal(buf.String())
		})

	// ---------------------------------------------------------------------------
	// pgp.Decrypt(cipher, keyFolder, [password])
	// ---------------------------------------------------------------------------
	Register(ns+"Decrypt", "pgp", "cipher, keyFolder, [password]",
		"Entschlüsselt eine PGP-Nachricht mit dem privaten Schlüssel aus einem Ordner.",
		func(args []Value) Value {
			if len(args) < 2 {
				return ErrorVal("Parameter fehlen: pgp.Decrypt(cipher, keyFolder, [password])")
			}

			cipherText := args[0].Str
			folder := args[1].Str
			pass := ""
			if len(args) >= 3 {
				pass = args[2].Str
			}

			absFolder, errVal := absPathVal(folder)
			if errVal != nil {
				return ErrorVal(fmt.Sprintf("ungültiger Pfad '%s'", folder))
			}

			keyPath := filepath.Join(absFolder, "id_pgp")
			entityList, err := parseArmoredKey(keyPath)
			if err != nil {
				return ErrorVal("Privater Schlüssel: " + err.Error())
			}

			// Alle Entities entschlüsseln (inkl. Subkeys — vorher fehlend)
			for _, entity := range entityList {
				if err := decryptEntity(entity, pass); err != nil {
					return ErrorVal(err.Error())
				}
			}

			dec, err := armor.Decode(strings.NewReader(cipherText))
			if err != nil {
				return ErrorVal("Ungültiges Cipher-Format: " + err.Error())
			}

			md, err := openpgp.ReadMessage(dec.Body, entityList, nil, nil)
			if err != nil {
				return ErrorVal("Entschlüsselung fehlgeschlagen: " + err.Error())
			}

			content, err := io.ReadAll(md.UnverifiedBody)
			if err != nil {
				return ErrorVal("Lesen des entschlüsselten Inhalts fehlgeschlagen: " + err.Error())
			}

			return StrVal(string(content))
		})

	// ---------------------------------------------------------------------------
	// pgp.Sign(msg, privKey, [password])
	// ---------------------------------------------------------------------------
	Register(ns+"Sign", "pgp", "msg, privKey, [password]",
		"Erzeugt eine abgetrennte PGP-Signatur über eine Textnachricht.",
		func(args []Value) Value {
			if len(args) < 2 {
				return errResult("Parameter fehlen: pgp.Sign(msg, privKey, [password])")
			}

			cleanMsg := strings.TrimSpace(args[0].Str)
			pass := ""
			if len(args) >= 3 {
				pass = args[2].Str
			}

			entityList, err := parseArmoredKey(args[1].Str)
			if err != nil {
				return errResult("Privater Schlüssel: " + err.Error())
			}

			if err := decryptEntity(entityList[0], pass); err != nil {
				return errResult(err.Error())
			}

			buf := new(bytes.Buffer)
			w, err := armor.Encode(buf, openpgp.SignatureType, nil)
			if err != nil {
				return errResult("Armor-Encoder fehlgeschlagen: " + err.Error())
			}

			if err := openpgp.DetachSign(w, entityList[0], strings.NewReader(cleanMsg), nil); err != nil {
				w.Close()
				return errResult("Signierfehler: " + err.Error())
			}
			w.Close()

			return okResult(buf.String())
		})

	// ---------------------------------------------------------------------------
	// pgp.SignFile(filePath, privKeyArmor, [password])
	// ---------------------------------------------------------------------------
	Register(ns+"SignFile", "pgp", "filePath, privKeyArmor, [password]",
		"Erzeugt eine abgetrennte PGP-Signatur über eine Datei.",
		func(args []Value) Value {
			if len(args) < 2 {
				return errResult("Parameter fehlen: pgp.SignFile(filePath, privKeyArmor, [password])")
			}

			absPath, errVal := absPathVal(args[0].Str)
			if errVal != nil {
				return errResult(fmt.Sprintf("ungültiger Pfad '%s'", args[0].Str))
			}

			pass := ""
			if len(args) >= 3 {
				pass = args[2].Str
			}

			entityList, err := parseArmoredKey(args[1].Str)
			if err != nil {
				return errResult("Privater Schlüssel: " + err.Error())
			}

			if err := decryptEntity(entityList[0], pass); err != nil {
				return errResult(err.Error())
			}

			f, err := os.Open(absPath)
			if err != nil {
				return errResult("Datei nicht gefunden: " + err.Error())
			}
			defer f.Close()

			buf := new(bytes.Buffer)
			w, err := armor.Encode(buf, openpgp.SignatureType, nil)
			if err != nil {
				return errResult("Armor-Encoder fehlgeschlagen: " + err.Error())
			}

			if err := openpgp.DetachSign(w, entityList[0], f, nil); err != nil {
				w.Close()
				return errResult("Signierfehler: " + err.Error())
			}
			w.Close()

			return okResult(buf.String())
		})

	// ---------------------------------------------------------------------------
	// pgp.Verify(msg, sigArmor, pubKey)
	// ---------------------------------------------------------------------------
	Register(ns+"Verify", "pgp", "msg, sigArmor, pubKey",
		"Verifiziert einen Text gegen eine abgetrennte PGP-Signatur.",
		func(args []Value) Value {
			if len(args) < 3 {
				return errResult("Parameter fehlen: pgp.Verify(msg, sigArmor, pubKey)")
			}

			cleanMsg := strings.TrimSpace(args[0].Str)
			sigArmor := strings.TrimSpace(args[1].Str)

			entityList, err := parseArmoredKey(args[2].Str)
			if err != nil {
				return errResult("Public Key: " + err.Error())
			}

			block, err := armor.Decode(strings.NewReader(sigArmor))
			if err != nil {
				return errResult("Ungültiges Signatur-Format: " + err.Error())
			}

			_, err = openpgp.CheckDetachedSignature(entityList, strings.NewReader(cleanMsg), block.Body, nil)
			if err != nil {
				return errResult(classifyVerifyError(err))
			}

			return okResult("Gültig")
		})

	// ---------------------------------------------------------------------------
	// pgp.VerifyFile(filePath, sigArmor, pubKeyArmor)
	// ---------------------------------------------------------------------------
	Register(ns+"VerifyFile", "pgp", "filePath, sigArmor, pubKeyArmor",
		"Verifiziert eine Datei gegen eine abgetrennte PGP-Signatur.",
		func(args []Value) Value {
			if len(args) < 3 {
				return errResult("Parameter fehlen: pgp.VerifyFile(filePath, sigArmor, pubKeyArmor)")
			}

			absPath, errVal := absPathVal(args[0].Str)
			if errVal != nil {
				return errResult(fmt.Sprintf("ungültiger Pfad '%s'", args[0].Str))
			}

			entityList, err := parseArmoredKey(args[2].Str)
			if err != nil {
				return errResult("Public Key: " + err.Error())
			}

			block, err := armor.Decode(strings.NewReader(args[1].Str))
			if err != nil {
				return errResult("Ungültiges Signatur-Format: " + err.Error())
			}

			f, err := os.Open(absPath)
			if err != nil {
				return errResult("Datei nicht gefunden: " + err.Error())
			}
			defer f.Close()

			signer, err := openpgp.CheckDetachedSignature(entityList, f, block.Body, nil)
			if err != nil {
				return errResult(classifyVerifyError(err))
			}

			signerName := "Unbekannt"
			for _, id := range signer.Identities {
				signerName = id.Name
				break
			}

			return okResult("Gültig: " + signerName)
		})

	// ---------------------------------------------------------------------------
	// pgp.ChangePassword(keyFolder, [oldPass], [newPass])
	// ---------------------------------------------------------------------------
	Register(ns+"ChangePassword", "pgp", "keyFolder, [oldPass], [newPass]",
		"Ändert das Passwort eines PGP-Schlüssels. Leer lassen zum Entfernen.",
		func(args []Value) Value {
			if len(args) < 1 {
				return ErrorVal("Parameter fehlen: pgp.ChangePassword(keyFolder, [oldPass], [newPass])")
			}

			folder := args[0].Str
			oldPass := ""
			if len(args) >= 2 {
				oldPass = args[1].Str
			}
			newPass := ""
			if len(args) >= 3 {
				newPass = args[2].Str
			}

			absFolder, errVal := absPathVal(folder)
			if errVal != nil {
				return ErrorVal(fmt.Sprintf("ungültiger Pfad '%s'", folder))
			}

			keyPath := filepath.Join(absFolder, "id_pgp")
			entityList, err := parseArmoredKey(keyPath)
			if err != nil {
				return ErrorVal("Schlüssel nicht lesbar: " + err.Error())
			}

			entity := entityList[0]

			if err := decryptEntity(entity, oldPass); err != nil {
				return ErrorVal(err.Error())
			}

			if err := encryptEntity(entity, newPass); err != nil {
				return ErrorVal(err.Error())
			}

			if err := writeKeyFile(keyPath, "", openpgp.PrivateKeyType, entity); err != nil {
				return ErrorVal("Speichern fehlgeschlagen: " + err.Error())
			}

			return StrVal("OK")
		})

	// ---------------------------------------------------------------------------
	// pgp.ExportKey(privKeyArmor, path, [password])
	// ---------------------------------------------------------------------------
	Register(ns+"ExportKey", "pgp", "privKeyArmor, path, [password]",
		"Exportiert einen PGP-Schlüssel in eine Datei, optional mit neuem Passwort.",
		func(args []Value) Value {
			if len(args) < 2 {
				return ErrorVal("Parameter fehlen: pgp.ExportKey(privKeyArmor, path, [password])")
			}

			pass := ""
			if len(args) >= 3 {
				pass = args[2].Str
			}

			absP, errVal := absPathVal(args[1].Str)
			if errVal != nil {
				return ErrorVal(fmt.Sprintf("ungültiger Pfad '%s'", args[1].Str))
			}

			entityList, err := parseArmoredKey(args[0].Str)
			if err != nil {
				return ErrorVal("Ungültiger PGP-Schlüssel: " + err.Error())
			}

			entity := entityList[0]

			if err := encryptEntity(entity, pass); err != nil {
				return ErrorVal(err.Error())
			}

			if err := writeKeyFile(absP, "", openpgp.PrivateKeyType, entity); err != nil {
				return ErrorVal("Export fehlgeschlagen: " + err.Error())
			}

			return StrVal("OK")
		})

	// ---------------------------------------------------------------------------
	// pgp.ImportKey(path, [password])
	// ---------------------------------------------------------------------------
	Register(ns+"ImportKey", "pgp", "path, [password]",
		"Lädt einen PGP-Schlüssel aus einer Datei und gibt ihn entschlüsselt zurück.",
		func(args []Value) Value {
			if len(args) < 1 {
				return ErrorVal("Parameter fehlen: pgp.ImportKey(path, [password])")
			}

			pass := ""
			if len(args) >= 2 {
				pass = args[1].Str
			}

			entityList, err := parseArmoredKey(args[0].Str)
			if err != nil {
				return ErrorVal("Schlüssel nicht lesbar: " + err.Error())
			}

			entity := entityList[0]

			// Nur entschlüsseln wenn nötig
			if entity.PrivateKey != nil && entity.PrivateKey.Encrypted {
				if err := decryptEntity(entity, pass); err != nil {
					return ErrorVal(err.Error())
				}
			}

			// Hinweis: Im Arbeitsspeicher liegt der Key nun im Klartext.
			// Der Aufrufer sollte den zurückgegebenen String so kurz wie möglich halten.
			result, err := serializePrivateKey(entity)
			if err != nil {
				return ErrorVal("Serialisierung fehlgeschlagen: " + err.Error())
			}

			return StrVal(result)
		})

	// ---------------------------------------------------------------------------
	// pgp.GetPublicKey(pathOrKey)
	// ---------------------------------------------------------------------------
	Register(ns+"GetPublicKey", "pgp", "pathOrKey",
		"Extrahiert den öffentlichen Schlüssel aus einem Pfad oder einem Armor-String.",
		func(args []Value) Value {
			if len(args) < 1 {
				return ErrorVal("Parameter fehlt: pgp.GetPublicKey(pathOrKey)")
			}

			entityList, err := parseArmoredKey(args[0].Str)
			if err != nil {
				return ErrorVal("Schlüssel nicht lesbar: " + err.Error())
			}

			result, err := serializePublicKey(entityList[0])
			if err != nil {
				return ErrorVal("Extrahierung fehlgeschlagen: " + err.Error())
			}

			return StrVal(result)
		})
}

// =============================================================================
// HILFSFUNKTION: Fehlerklassifizierung für Verify
// =============================================================================

// classifyVerifyError wandelt einen kryptografischen Fehler in eine
// benutzerfreundliche, aber dennoch präzise Fehlermeldung um.
func classifyVerifyError(err error) string {
	s := err.Error()
	switch {
	case strings.Contains(s, "signature invalid"):
		return "Inhalt wurde verändert oder Signatur passt nicht zum Text"
	case strings.Contains(s, "expired"):
		return "Signatur oder Schlüssel ist abgelaufen"
	case strings.Contains(s, "no matching keys"):
		return "Signatur wurde mit einem anderen Schlüssel erstellt"
	case strings.Contains(s, "unsupported"):
		return "Nicht unterstützter kryptografischer Algorithmus"
	default:
		return "Verifikation fehlgeschlagen: " + s
	}
}
