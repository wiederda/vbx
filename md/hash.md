# #️⃣ hash.* – Hash-Funktionen

Dient zur Erzeugung und Verifizierung von kryptografischen Hashes.
Die einfachen Hash-Funktionen (`MD5`, `SHA*`) sind auch als globale Funktionen ohne Namespace verfügbar.

---

## hash.MD5(s)
- **Konkret:**
  Erzeugt einen MD5-Hash als Hex-String.
  Nicht für Sicherheitszwecke geeignet – nur für Legacy-Kompatibilität und Prüfsummen.
- **Parameter:**
  - `s`: Quellstring.
- **Rückgabe:**
  `StrVal` (32-stelliger Hex-String)

---

## hash.SHA1(s)
- **Konkret:**
  Erzeugt einen SHA1-Hash als Hex-String.
  Nur für Legacy-Kompatibilität – für neue Anwendungen SHA256 bevorzugen.
- **Parameter:**
  - `s`: Quellstring.
- **Rückgabe:**
  `StrVal` (40-stelliger Hex-String)

---

## hash.SHA256(s)
- **Konkret:**
  Erzeugt einen SHA256-Hash als Hex-String.
- **Parameter:**
  - `s`: Quellstring.
- **Rückgabe:**
  `StrVal` (64-stelliger Hex-String)

---

## hash.SHA512(s)
- **Konkret:**
  Erzeugt einen SHA512-Hash als Hex-String.
- **Parameter:**
  - `s`: Quellstring.
- **Rückgabe:**
  `StrVal` (128-stelliger Hex-String)

---

## hash.HMAC(s, key [, algo])
- **Konkret:**
  Erzeugt einen HMAC (Hash-based Message Authentication Code).
  Geeignet für API-Authentifizierung, Webhook-Validierung und Signaturen.
- **Parameter:**
  - `s`: Zu signierender String.
  - `key`: Geheimer Schlüssel.
  - `algo`: Optional. Hash-Algorithmus (Standard: `"sha256"`). Unterstützt: `"sha256"`, `"sha512"`, `"sha1"`, `"md5"`.
- **Rückgabe:**
  `StrVal` (Hex-String), `ErrorVal` bei unbekanntem Algorithmus.

---

## hash.Bcrypt(pass [, cost])
- **Konkret:**
  Erzeugt einen sicheren Bcrypt-Hash eines Passworts.
  Bcrypt ist absichtlich langsam und damit sicher für Passwort-Hashing.
  Der gleiche Hash sieht bei jedem Aufruf anders aus (integriertes Salt).
- **Parameter:**
  - `pass`: Zu hashendes Passwort.
  - `cost`: Optional. Kosten-Faktor (Standard: `10`, gültig: `4`–`31`). Höher = langsamer = sicherer.
- **Rückgabe:**
  `StrVal` (Bcrypt-Hash), `ErrorVal` bei Fehler.

---

## hash.BcryptVerify(pass, hash)
- **Konkret:**
  Prüft ob ein Klartext-Passwort zu einem Bcrypt-Hash passt.
- **Parameter:**
  - `pass`: Klartext-Passwort.
  - `hash`: Bcrypt-Hash (aus `hash.Bcrypt`).
- **Rückgabe:**
  `BoolVal` (`true` wenn Passwort korrekt)

---

## hash.File(path [, algo])
- **Konkret:**
  Berechnet den Hash einer Datei.
  Liest die gesamte Datei in den Speicher – für sehr große Dateien ggf. langsam.
- **Parameter:**
  - `path`: Pfad zur Datei.
  - `algo`: Optional. Hash-Algorithmus (Standard: `"sha256"`). Unterstützt: `"sha256"`, `"sha512"`, `"sha1"`, `"md5"`.
- **Rückgabe:**
  `StrVal` (Hex-String), `ErrorVal` bei Lesefehler.