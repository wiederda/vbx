# 🗂️ app.* – Anwendungs- & Pfadfunktionen

Dient zur Abfrage von Pfaden der laufenden Anwendung und systemspezifischen Standardverzeichnissen.
Plattformübergreifend (Windows, Linux, macOS).

---

## app.StartupPath()
- **Konkret:**
  Gibt das Verzeichnis zurück, in dem die ausführbare Datei liegt.
- **Rückgabe:**
  `StrVal`

---

## app.ExecutablePath()
- **Konkret:**
  Gibt den vollständigen Pfad der ausführbaren Datei zurück (inkl. Dateiname).
- **Rückgabe:**
  `StrVal`

---

## app.CurrentDirectory()
- **Konkret:**
  Gibt das aktuelle Arbeitsverzeichnis zurück (Working Directory).
  Das ist das Verzeichnis von dem aus VBMini gestartet wurde, nicht das Verzeichnis der Executable.
- **Rückgabe:**
  `StrVal`

---

## app.TempPath()
- **Konkret:**
  Gibt den Pfad zum temporären Verzeichnis des Betriebssystems zurück.
- **Rückgabe:**
  `StrVal`
  Beispiele: `"/tmp"` (Linux/macOS), `"C:\Users\Name\AppData\Local\Temp"` (Windows).

---

## app.SpecialFolder(name)
- **Konkret:**
  Gibt einen systemspezifischen Standardpfad zurück.
  Auf Windows werden Umgebungsvariablen ausgewertet.
  Auf Linux/macOS werden Pfade relativ zum Home-Verzeichnis aufgebaut.
- **Parameter:**
  - `name`: Name des Ordners (case-insensitiv).
- **Unterstützte Werte:**

  | Name             | Windows                        | Linux / macOS              |
  |------------------|--------------------------------|----------------------------|
  | `"home"`         | –                              | `~`                        |
  | `"desktop"`      | –                              | `~/Desktop`                |
  | `"documents"`    | –                              | `~/Documents`              |
  | `"downloads"`    | –                              | `~/Downloads`              |
  | `"pictures"`     | –                              | `~/Pictures`               |
  | `"music"`        | –                              | `~/Music`                  |
  | `"videos"`       | –                              | `~/Videos`                 |
  | `"temp"`         | `%TEMP%`                       | `/tmp`                     |
  | `"appdata"`      | `%APPDATA%`                    | –                          |
  | `"localappdata"` | `%LOCALAPPDATA%`               | –                          |
  | `"programdata"`  | `%PROGRAMDATA%`                | –                          |

- **Rückgabe:**
  `StrVal`
  Leerer String wenn der Name nicht unterstützt wird oder der Pfad nicht ermittelt werden kann.