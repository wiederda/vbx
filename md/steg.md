# 🖼️ steg.* – Steganografie-Funktionen

Dient zum verdeckten Einbetten und Extrahieren von Daten in Bilddateien via LSB-Steganografie (Least Significant Bit).
Unterstützte Formate: BMP, PNG.
Eingebettete Daten werden durch einen Seed-basierten Stride und eine AES-CTR-Permutation verteilt – ohne Seed sind die Daten nicht extrahierbar.

---

## steg.Inject(inPath, outPath, dataB64, seed)
- **Konkret:**
  Bettet Base64-kodierte Daten in ein Bild ein.
  Schreibt atomar (temp-Datei + Rename).
  BMP: LSB der Pixel-Bytes. PNG: LSB des Rot-Kanals.
  Die Datenverteilung ist durch Seed und Stride deterministisch und nicht sequenziell.
- **Parameter:**
  - `inPath`: Quelldatei (BMP oder PNG).
  - `outPath`: Zieldatei.
  - `dataB64`: Einzubettende Daten als Base64-String.
  - `seed`: Seed-String (steuert Verteilung und Stride).
- **Rückgabe:**
  `ArrVal`
  Format: `[OK, Seed, Msg]`

---

## steg.Extract(path, seed)
- **Konkret:**
  Extrahiert eingebettete Daten aus einem Bild.
  Seed muss identisch zum Seed beim Einbetten sein.
  Gibt einen Fehler zurück wenn kein gültiger STEG-Header gefunden wird.
- **Parameter:**
  - `path`: Quelldatei (BMP oder PNG).
  - `seed`: Seed-String.
- **Rückgabe:**
  `ArrVal`
  Format: `[OK, DataBase64, Msg]`

---

## steg.GenerateSeed(pass, salt)
- **Konkret:**
  Erzeugt einen deterministischen Seed aus Passwort und Salt via HMAC-SHA256.
  Gleiche Eingaben erzeugen immer denselben Seed.
- **Parameter:**
  - `pass`: Passwort.
  - `salt`: Salt-Wert.
- **Rückgabe:**
  `StrVal`
  Hex-kodierter 64-Zeichen-Seed.

---

## steg.GetCapacity(path, dataLen, seed)
- **Konkret:**
  Prüft ob ein Bild groß genug für einen Payload ist.
  Berechnet die nutzbare Kapazität abzüglich 8 Bytes Header-Overhead.
  Gibt zusätzlich einen Netto-Wert mit 35 % Sicherheitspuffer zurück.
- **Hinweis:**
  `dataLen` ist die rohe Byte-Länge der Nutzdaten, nicht die Base64-Länge.
- **Parameter:**
  - `path`: Zu prüfendes Bild (BMP oder PNG).
  - `dataLen`: Geplante Datengröße in Bytes.
  - `seed`: Seed-String (beeinflusst den Stride).
- **Rückgabe:**
  `ArrVal`
  Format: `[OK, NettoBytes, Msg, BruttoBytes]`
  `NettoBytes` = verfügbar mit 35 % Puffer. `BruttoBytes` = maximale Rohkapazität.