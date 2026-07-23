# 🖼️ picture.* – Bildfunktionen

Dient zur Konvertierung, Skalierung und Aufnahme von Bildern.
Unterstützte Formate: `jpg`, `png`, `ico`.

---

## picture.Convert(inFile, outFile, format [, w, h, q])
- **Konkret:**
  Konvertiert ein Bild in ein anderes Format. Optional mit Größenänderung und Qualitätsstufe.
  Bei `w` und `h` zusammen: Bild wird auf exakte Größe zugeschnitten (center crop).
  Bei nur `w` oder nur `h`: Proportionale Skalierung.
- **ICO-Sonderverhalten:**
  Statt `w`, `h`, `q` wird eine kommagetrennte Größenliste angegeben (z. B. `"16,32,48"`).
  Pro Größe wird eine separate Datei erzeugt: `<base>_16.ico`, `<base>_32.ico` usw.
  Gültige ICO-Größen: 1–256.
- **Parameter:**
  - `inFile`: Quelldatei.
  - `outFile`: Zieldatei.
  - `format`: Zielformat. Unterstützte Werte: `"jpg"`, `"jpeg"`, `"png"`, `"ico"`.
  - `w`: Optional. Zielbreite in Pixeln.
  - `h`: Optional. Zielhöhe in Pixeln.
  - `q`: Optional. JPEG-Qualität (1–100, Standard: 100). Nur für `jpg`/`jpeg`.
- **Rückgabe:**
  `NullVal` bei Erfolg, `ErrorVal` bei Fehler.

---

## picture.ConvertAll(inDir, outDir, format [, filter, w, h, q])
- **Konkret:**
  Konvertiert alle Bilddateien eines Ordners.
  Erstellt das Zielverzeichnis automatisch, falls es nicht existiert.
  Verzeichnisse im Quellordner werden übersprungen.
  Intern wird `picture.Convert` für jede Datei aufgerufen.
- **Parameter:**
  - `inDir`: Quellverzeichnis.
  - `outDir`: Zielverzeichnis.
  - `format`: Zielformat (`"jpg"`, `"png"`, `"ico"`).
  - `filter`: Optional. Dateifilter (wird an `picture.Convert` weitergegeben).
  - `w`: Optional. Zielbreite.
  - `h`: Optional. Zielhöhe.
  - `q`: Optional. JPEG-Qualität.
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