# 🔐 crypt.* – Kryptografie- & Zufallsfunktionen

Dient zur kryptografisch sicheren Erzeugung von Zufallswerten, Passwörtern und GUIDs sowie zur AES-256-GCM-Verschlüsselung von Text.
Alle Zufallsoperationen nutzen `crypto/rand` (kein `math/rand`).

---

## crypt.GUID()
- **Konkret:**
  Generiert eine eindeutige GUID (Version 4, RFC 4122).
  Setzt die Versions- und Variant-Bits korrekt.
- **Rückgabe:**
  `ArrVal`
  Format: `[OK, GUID, Msg]`
  Beispiel: `"550e8400-e29b-41d4-a716-446655440000"`.

---

## crypt.RNGCryptoProvider([length])
- **Konkret:**
  Erzeugt kryptografisch sichere Zufallsbytes und gibt sie als Hex-String zurück.
- **Parameter:**
  - `length`: Optional. Anzahl Bytes (Standard: 16).
- **Rückgabe:**
  `ArrVal`
  Format: `[OK, HexString, Msg]`

---

## crypt.RandomString(length)
- **Konkret:**
  Erzeugt eine kryptografisch zufällige alphanumerische Zeichenfolge (`a-z`, `A-Z`, `0-9`).
  Minimum: 10 Zeichen (kleinere Werte werden auf 10 aufgerundet).
- **Parameter:**
  - `length`: Gewünschte Länge (Minimum: 10).
- **Rückgabe:**
  `StrVal`

---

## crypt.RandomPassword(len [, num, low, up, spec, head])
- **Konkret:**
  Generiert ein kryptografisch zufälliges Passwort mit konfigurierbarer Zeichensatz-Zusammensetzung.
  Stellt sicher, dass jede aktivierte Zeichenklasse mindestens einmal enthalten ist.
  Minimum: 10 Zeichen. Fallback: Kleinbuchstaben, falls kein Charset aktiviert.
- **Parameter:**
  - `len`: Passwortlänge (Minimum: 10).
  - `num`: Optional. `BoolVal` – Ziffern einschließen.
  - `low`: Optional. `BoolVal` – Kleinbuchstaben einschließen.
  - `up`: Optional. `BoolVal` – Großbuchstaben einschließen.
  - `spec`: Optional. `BoolVal` – Sonderzeichen einschließen.
  - `head`: Optional. `BoolVal` – erstes Zeichen erzwungener Buchstabe.
- **Rückgabe:**
  `StrVal`

---

## crypt.AESEncrypt(text, pass)
- **Konkret:**
  Verschlüsselt einen String mit AES-256-GCM.
  Der Schlüssel wird aus dem Passwort via SHA-256 abgeleitet.
  Nonce wird kryptografisch zufällig erzeugt und dem Ciphertext vorangestellt.
  Ausgabe ist Base64-kodiert.
- **Parameter:**
  - `text`: Zu verschlüsselnder Klartext.
  - `pass`: Passwort (wird intern zu 256-Bit-Key gehasht).
- **Rückgabe:**
  `ArrVal`
  Format: `[OK, CipherBase64, Msg]`

---

## crypt.AESDecrypt(cipher, pass)
- **Konkret:**
  Entschlüsselt AES-256-GCM-verschlüsselte Daten (erzeugt von `crypt.AESEncrypt`).
  Extrahiert Nonce automatisch aus den Rohdaten.
  Schlägt bei falschem Passwort oder korrupten Daten fehl.
- **Parameter:**
  - `cipher`: Base64-kodierter Ciphertext.
  - `pass`: Passwort (muss identisch mit dem bei der Verschlüsselung sein).
- **Rückgabe:**
  `ArrVal`
  Format: `[OK, Plaintext, Msg]`