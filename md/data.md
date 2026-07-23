# 📊 data.* – Konvertierungs- & Messfunktionen

Dient zur Umrechnung von Datenmengen, Leistungs- und Zeiteinheiten sowie zur einfachen Performance-Messung.

---

## Datenmengen (SI, Basis 1000)

## data.ByteToKb(val)
- **Konkret:**
  Konvertiert Byte in Kilobyte (Basis 1000).
- **Rückgabe:**
  `NumVal`

---

## data.KbToMb(val)
- **Konkret:**
  Konvertiert Kilobyte in Megabyte (Basis 1000).
- **Rückgabe:**
  `NumVal`

---

## data.MbToGb(val)
- **Konkret:**
  Konvertiert Megabyte in Gigabyte (Basis 1000).
- **Rückgabe:**
  `NumVal`

---

## data.GbToTb(val)
- **Konkret:**
  Konvertiert Gigabyte in Terabyte (Basis 1000).
- **Rückgabe:**
  `NumVal`

---

## Datenmengen (Binär, Basis 1024)

## data.ByteToKiB(val)
- **Konkret:**
  Konvertiert Byte in Kibibyte (Basis 1024).
- **Rückgabe:**
  `NumVal`

---

## data.MiBToByte(val)
- **Konkret:**
  Konvertiert Mebibyte in Byte (Basis 1024²).
- **Rückgabe:**
  `NumVal`

---

## Leistung

## data.WattToKilowatt(val)
- **Konkret:**
  Konvertiert Watt in Kilowatt.
- **Rückgabe:**
  `NumVal`

---

## data.KilowattToWatt(val)
- **Konkret:**
  Konvertiert Kilowatt in Watt.
- **Rückgabe:**
  `NumVal`

---

## Zeit

## data.MinutesToHours(val)
- **Konkret:**
  Konvertiert Minuten in Stunden.
- **Rückgabe:**
  `NumVal`

---

## data.HoursToMinutes(val)
- **Konkret:**
  Konvertiert Stunden in Minuten.
- **Rückgabe:**
  `NumVal`

---

## data.SecondsToDays(val)
- **Konkret:**
  Konvertiert Sekunden in Tage.
- **Rückgabe:**
  `NumVal`

---

## data.DaysToSeconds(val)
- **Konkret:**
  Konvertiert Tage in Sekunden.
- **Rückgabe:**
  `NumVal`

---

## data.FormatSeconds(seconds)
- **Konkret:**
  Formatiert eine Sekundenanzahl als lesbaren Zeitstring.
- **Parameter:**
  - `seconds`: Anzahl Sekunden.
- **Rückgabe:**
  `StrVal`
  Format: `HH:MM:SS`. Beispiel: `3661` → `"01:01:01"`.

---

## Performance-Messung

## data.TimerStart([name])
- **Konkret:**
  Startet eine benannte Zeitmessung. Mehrere Timer können gleichzeitig laufen.
  Ohne Namen wird `"default"` verwendet.
- **Parameter:**
  - `name`: Optional. Name des Timers.
- **Rückgabe:**
  `BoolVal` (`true`)

---

## data.TimerElapsed([name])
- **Konkret:**
  Gibt die verstrichene Zeit in Sekunden seit dem letzten `data.TimerStart` zurück.
  Gibt `0` zurück wenn der Timer noch nicht gestartet wurde.
- **Parameter:**
  - `name`: Optional. Name des Timers (Standard: `"default"`).
- **Rückgabe:**
  `NumVal` (Sekunden als Fließkommazahl)