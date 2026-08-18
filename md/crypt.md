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

  ---

  ## crypt.Wipe(byteArray)
- **Konkret:**
  Überschreibt ein Byte-Array (z. B. von `reg.ReadProtectedValueBytes`) in-place mit Nullen.
- **Parameter:**
  - `byteArray`: Ein Array vom Typ `KindArr` mit Byte-Werten (0–255).
- **Rückgabe:**
  `BoolVal` (`true`) bei Erfolg, `ErrorVal` bei falschem Argumenttyp.
- **Hinweis:**
  Arrays sind in VBX mutable (im Gegensatz zu Strings). `crypt.Wipe` nutzt das aus und überschreibt den tatsächlichen Speicherinhalt, nicht nur eine Kopie – verifiziert per Test: Nach dem Aufruf sind alle Elemente der Original-Variable `0`.
  Sollte aufgerufen werden, sobald ein per `reg.ReadProtectedValueBytes` gelesener Wert nicht mehr benötigt wird.

---

## crypt.BytesToString(byteArray)
- **Konkret:**
  Wandelt ein Byte-Array (0–255-Werte) in einen UTF-8-String um.
- **Parameter:**
  - `byteArray`: Ein Array vom Typ `KindArr` mit Byte-Werten (0–255).
- **Rückgabe:**
  `StrVal` bei Erfolg, `ErrorVal` bei falschem Argumenttyp oder Nicht-Zahlen-Werten im Array.
- **Hinweis:**
  Der erzeugte String ist ab diesem Moment eine eigene Speicherkopie und **nicht** mehr über `crypt.Wipe` löschbar (Wipe wirkt nur auf das Byte-Array). Für kurzlebige Verwendung gedacht – direkt vor dem eigentlichen Gebrauch aufrufen und danach mit `crypt.WipeString` löschen.

---

## crypt.WipeString(value)
- **Konkret:**
  Überschreibt den tatsächlichen Speicherinhalt eines Strings mit Nullbytes (`0x00`). Im Unterschied zu `x = ""` wird hier nicht nur die Variable auf einen neuen String umgebogen, sondern der ursprüngliche Speicherbereich selbst gelöscht.
- **Parameter:**
  - `value`: Ein String (`KindStr`).
- **Rückgabe:**
  `BoolVal` (`true`) bei Erfolg, `ErrorVal` bei falschem Argumenttyp.
- **Hinweis:**
  Nutzt intern `unsafe`, um Go's String-Immutability bewusst zu umgehen. Nach dem Aufruf hat die Variable weiterhin die ursprüngliche Länge, besteht aber nur noch aus Nullbytes (verifiziert: `Len()` bleibt bei Umlauten sogar größer als die sichtbare Zeichenzahl, weil einzelne Mehrbyte-UTF-8-Sequenzen durch mehrere Einzel-Nullbytes ersetzt werden).

  **Wichtige Einschränkungen:**
  - Löscht nur den Speicherbereich, auf den *dieser eine* String zeigt. Wurde der Wert vorher verkettet (`&`), umgewandelt oder anderweitig zu einem neuen String verarbeitet, hat diese Kopie ein eigenes Backing-Array und bleibt von `crypt.WipeString` unberührt.
  - Sollte immer der *letzte* Umgang mit dem Wert sein – jede Nutzung danach liest nur noch Nullbytes.
  - Nur für dynamisch zur Laufzeit erzeugte Strings (z. B. via `crypt.BytesToString`) sicher. **Niemals** auf String-Literale aus dem Quellcode anwenden (z. B. `Dim x = "MeinPasswort"` gefolgt von `crypt.WipeString(x)`) – Go kann identische String-Literale intern deduplizieren/teilen, wodurch ein Wipe unabsichtlich andere, unabhängige Stellen im Programm beschädigen könnte.

---

## Empfohlener Lifecycle für sensible Werte

Um einen per DPAPI gespeicherten Wert möglichst spurenarm zu verwenden:

```vb
#use reg,crypt
Dim SecretBytes = reg.ReadProtectedValueBytes(TestRoot, TestPath, TestName)
Dim SecretStr = crypt.BytesToString(SecretBytes)

' ... SecretStr sofort und einmalig verwenden (z.B. für einen API-Call) ...
' KEINE Verkettung oder Kopie von SecretStr vor dem Wipe!

crypt.Wipe(SecretBytes)       ' Byte-Array nullen
crypt.WipeString(SecretStr)   ' String-Backing-Array nullen
```

**Was damit erreicht wird:**
Beide Speicherkopien (Rohbytes und der daraus erzeugte String) werden aktiv überschrieben, statt unkontrolliert im Speicher bis zur GC-Freigabe liegen zu bleiben.

**Was damit *nicht* erreicht wird:**
Vollständiger Schutz gegen einen Memory-Dump *während* der kurzen Nutzung von `SecretStr` ist technisch nicht möglich, solange der Wert überhaupt als normaler String verwendet wird (z. B. für einen HTTP-Header). Der Ansatz minimiert das Zeitfenster, in dem der Klartext im Speicher steht, ersetzt aber keine dedizierte Secure-Memory-Lösung. Für die meisten Zwecke (Schutz gegen Registry-Auslesen durch andere Nutzer/Prozesse, Schutz gegen nachträgliche Speicheranalyse nach Programmende) ist das aber ausreichend.