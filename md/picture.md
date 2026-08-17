# 🖼️ picture.* – Bildfunktionen

Dient zur Konvertierung, Skalierung und Aufnahme von Bildern.
Unterstützte Formate: `jpg`, `png`, `webp`, `ico`. `webp` wird auch als **Input** erkannt (z. B. `picture.Convert("bild.webp", "bild.png", "png")`).

---

## picture.Convert(inFile, outFile, format [, w, h, q])
- **Konkret:**
  Konvertiert ein Bild in ein anderes Format. Optional mit Größenänderung und Qualitätsstufe.
  Bei `w` und `h` zusammen: Bild wird auf exakte Größe zugeschnitten (center crop).
  Bei nur `w` oder nur `h`: Proportionale Skalierung.
  `inFile` kann `jpg`, `png` oder `webp` sein – das Format wird automatisch anhand der Datei erkannt, unabhängig von der Dateiendung.
- **ICO-Sonderverhalten:**
  Statt `w`, `h`, `q` wird eine kommagetrennte Größenliste angegeben (z. B. `"16,32,48"`).
  Pro Größe wird eine separate Datei erzeugt: `<base>_16.ico`, `<base>_32.ico` usw.
  Gültige ICO-Größen: 1–256.
- **WebP-Sonderverhalten (`format = "webp"`):**
  `q` steuert hier den Encoder-Modus statt einer reinen JPEG-Qualität:
  - `q` nicht angegeben, `0` oder `≥ 100` → **verlustfrei** (VP8L), jedes Pixel bleibt exakt erhalten.
  - `q` zwischen `1` und `99` → **verlustbehaftet** (VP8), analog zur JPEG-Qualität.
-- **Parameter:**
  - `inFile`: Quelldatei.
  - `outFile`: Zieldatei. Die Dateiendung wird **nicht** geprüft oder erzwungen – maßgeblich für das tatsächliche Bildformat ist ausschließlich `format`. Bei `outFile = "bild.  png"` mit `format = "jpg"` entsteht z. B. eine Datei mit `.png`-Endung, die aber JPEG-Bytes enthält.
  - `format`: Zielformat. Unterstützte Werte: `"jpg"`, `"jpeg"`, `"png"`, `"webp"`, `"ico"`.
  - `w`: Optional. Zielbreite in Pixeln.
  - `h`: Optional. Zielhöhe in Pixeln.
  - `q`: Optional. Qualität (1–100, Standard: 100). Bei `jpg`/`jpeg` klassische JPEG-Qualität, bei `webp` siehe oben (verlustfrei vs. verlustbehaftet). Für `png` ohne Wirkung.
- **Rückgabe:**
  `NullVal` bei Erfolg, `ErrorVal` bei Fehler.

---

## picture.ConvertAll(inDir, outDir, format [, filter, w, h, q])
- **Konkret:**
  Konvertiert alle Bilddateien eines Ordners.
  Erstellt das Zielverzeichnis automatisch, falls es nicht existiert.
  Verzeichnisse im Quellordner werden übersprungen.
  Intern wird `picture.Convert` für jede Datei aufgerufen – Eingabeformate `jpg`, `png`, `webp` werden dabei automatisch erkannt.
- **Parameter:**
  - `inDir`: Quellverzeichnis.
  - `outDir`: Zielverzeichnis.
  - `format`: Zielformat (`"jpg"`, `"png"`, `"webp"`, `"ico"`).
  - `filter`: Optional. Dateifilter (wird an `picture.Convert` weitergegeben).
  - `w`: Optional. Zielbreite.
  - `h`: Optional. Zielhöhe.
  - `q`: Optional. Qualität (siehe `picture.Convert` – Bedeutung hängt vom Zielformat ab).
- **Rückgabe:**
  `StrVal`
  Beispiel: `"Erfolg: 12, Fehler: 1"`.

---

## picture.Snapshot(outFile [, monitorIndex])
- **Konkret:**
  Erstellt einen Screenshot eines Monitors und speichert ihn als Datei.
  Das Dateiformat wird automatisch anhand der Dateiendung erkannt (`.png`, `.jpg`).
- **Parameter:**
  - `outFile`: Zielpfad inkl. Dateiname und Endung.
  - `monitorIndex`: Optional. Index des Monitors (0-basiert, Standard: 0).
- **Rückgabe:**
  `NullVal` bei Erfolg, `ErrorVal` wenn Monitor nicht gefunden oder Speichern fehlschlägt.