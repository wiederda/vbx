# 🔤 string.* – String-Operationen

Dient zur Verarbeitung, Analyse, Formatierung und Konvertierung von Strings.
Alle Funktionen sind Runen-basiert und damit korrekt für Unicode/UTF-8.

---

## string.HexEncode(val)
- **Konkret:**
  Wandelt einen String in seine Hex-Darstellung um.
- **Parameter:**
  - `val`: Quellstring.
- **Rückgabe:**
  `StrVal`
  Beispiel: `"ABC"` → `"414243"`.

---

## string.HexDecode(hexStr)
- **Konkret:**
  Wandelt einen Hex-String zurück in Text.
- **Parameter:**
  - `hexStr`: Hex-kodierter String.
- **Rückgabe:**
  `StrVal` bei Erfolg, `ErrorVal` bei ungültigem Hex.

---

## string.CharAt(str, index)
- **Konkret:**
  Gibt das Zeichen an einer bestimmten Position zurück (1-basiert, Runen-basiert).
  Gibt leeren String zurück wenn der Index außerhalb liegt.
- **Parameter:**
  - `str`: Quellstring.
  - `index`: Position (1-basiert).
- **Rückgabe:**
  `StrVal`

---

## string.Join(arr [, sep])
- **Konkret:**
  Verbindet alle Elemente eines Arrays zu einem String.
- **Parameter:**
  - `arr`: `ArrVal` mit zu verbindenden Elementen.
  - `sep`: Optional. Trennzeichen (Standard: `"\n"`).
- **Rückgabe:**
  `StrVal`

---

## string.StartsWith(s, prefix)
- **Konkret:**
  Prüft, ob ein String mit dem angegebenen Präfix beginnt.
- **Parameter:**
  - `s`: String.
  - `prefix`: Gesuchtes Präfix.
- **Rückgabe:**
  `BoolVal`

---

## string.EndsWith(s, suffix)
- **Konkret:**
  Prüft, ob ein String mit dem angegebenen Suffix endet.
- **Parameter:**
  - `s`: String.
  - `suffix`: Gesuchtes Suffix.
- **Rückgabe:**
  `BoolVal`

---

## string.Repeat(s, n)
- **Konkret:**
  Wiederholt einen String `n`-mal.
- **Parameter:**
  - `s`: String.
  - `n`: Anzahl Wiederholungen.
- **Rückgabe:**
  `StrVal`

---

## string.Reverse(s)
- **Konkret:**
  Kehrt die Reihenfolge aller Zeichen um (Runen-basiert).
- **Parameter:**
  - `s`: String.
- **Rückgabe:**
  `StrVal`

---

## string.PadLeft(s, n [, c])
- **Konkret:**
  Füllt einen String links auf eine Mindestlänge auf.
  Ist der String bereits lang genug, wird er unverändert zurückgegeben.
- **Parameter:**
  - `s`: String.
  - `n`: Zielbreite.
  - `c`: Optional. Füllzeichen (Standard: Leerzeichen).
- **Rückgabe:**
  `StrVal`

---

## string.PadRight(s, n [, c])
- **Konkret:**
  Füllt einen String rechts auf eine Mindestlänge auf.
  Ist der String bereits lang genug, wird er unverändert zurückgegeben.
- **Parameter:**
  - `s`: String.
  - `n`: Zielbreite.
  - `c`: Optional. Füllzeichen (Standard: Leerzeichen).
- **Rückgabe:**
  `StrVal`

---

## string.WordCount(s)
- **Konkret:**
  Zählt die Anzahl der Wörter in einem Text.
  Mehrfache Leerzeichen werden korrekt behandelt.
- **Parameter:**
  - `s`: String.
- **Rückgabe:**
  `NumVal`

---

## string.CharCount(s)
- **Konkret:**
  Zählt alle Buchstaben und Ziffern im Text (keine Leer- oder Sonderzeichen).
- **Parameter:**
  - `s`: String.
- **Rückgabe:**
  `NumVal`

---

## string.Between(s, start, end)
- **Konkret:**
  Extrahiert den Text zwischen zwei Markern (erste Fundstelle).
  Gibt leeren String zurück wenn ein Marker nicht gefunden wird.
- **Parameter:**
  - `s`: Quellstring.
  - `start`: Linker Marker.
  - `end`: Rechter Marker.
- **Rückgabe:**
  `StrVal`

---

## string.Val(n)
- **Konkret:**
  Extrahiert den numerischen Wert am Anfang eines Strings (von links).
  Akzeptiert Vorzeichen und einen Dezimalpunkt.
  Bricht beim ersten nicht-numerischen Zeichen ab.
- **Parameter:**
  - `n`: String mit führendem Zahlenwert.
- **Rückgabe:**
  `NumVal`
  Beispiel: `"42px"` → `42`, `"abc"` → `0`.

---

## string.CleanLines(s)
- **Konkret:**
  Trimmt Leerzeichen und Tabs am Ende jeder Zeile.
  Leerzeilen bleiben erhalten.
  Führende und abschließende Leerzeilen werden entfernt.
- **Parameter:**
  - `s`: String.
- **Rückgabe:**
  `StrVal`

---

## string.CleanAllLines(s)
- **Konkret:**
  Entfernt alle Leerzeilen und trimmt verbleibende Zeilen vollständig.
  Ergebnis ist ein kompakter Block ohne Leerzeilen.
- **Parameter:**
  - `s`: String.
- **Rückgabe:**
  `StrVal`

---

## string.TrimLines(content)
- **Konkret:**
  Entfernt Leerzeichen, Tabs und `\r` am Ende jeder einzelnen Zeile.
  Leerzeilen bleiben erhalten.
- **Parameter:**
  - `content`: String.
- **Rückgabe:**
  `StrVal`

---

## string.RegExp(s, pattern)
- **Konkret:**
  Führt einen regulären Ausdruck aus und gibt alle Treffer inklusive Capture Groups zurück.
  Index 0 = vollständiger Treffer, Index 1+ = Capture Groups.
- **Parameter:**
  - `s`: Quellstring.
  - `pattern`: Go-Regex-Pattern.
- **Rückgabe:**
  `ArrVal`
  Leeres Array wenn kein Treffer. `ErrorVal` bei ungültigem Pattern.

---

## string.Extract(text, pattern)
- **Konkret:**
  Extrahiert den Wert der ersten Capture Group eines Regex-Patterns.
  Gibt leeren String zurück wenn kein Treffer oder keine Gruppe.
- **Parameter:**
  - `text`: Quellstring.
  - `pattern`: Go-Regex-Pattern mit mindestens einer Capture Group `()`.
- **Rückgabe:**
  `StrVal`

---

## string.ExtractMax(text, pattern)
- **Konkret:**
  Findet alle Treffer der ersten Capture Group und gibt den höchsten Wert zurück.
  Sortierung nach Länge, dann lexikografisch (geeignet für Versionsnummern).
- **Parameter:**
  - `text`: Quellstring.
  - `pattern`: Go-Regex-Pattern mit mindestens einer Capture Group.
- **Rückgabe:**
  `StrVal`
  Höchster gefundener Wert, oder leerer String wenn kein Treffer.

---

## string.Like(text, pattern)
- **Konkret:**
  Einfacher Mustervergleich im VB-Stil. Vollständiger Match (Anfang bis Ende).
- **Parameter:**
  - `text`: Quellstring.
  - `pattern`: Muster mit folgenden Wildcards:
    - `#` – eine Ziffer
    - `?` – ein beliebiges Zeichen
    - `*` – beliebig viele Zeichen
- **Rückgabe:**
  `BoolVal`

---

## string.StrComp(s1, s2 [, mode])
- **Konkret:**
  Vergleicht zwei Strings lexikografisch.
- **Parameter:**
  - `s1`: Erster String.
  - `s2`: Zweiter String.
  - `mode`: Optional. `0` = case-sensitive (Standard), `1` = case-insensitive.
- **Rückgabe:**
  `NumVal`
  `-1` wenn s1 < s2, `0` wenn gleich, `1` wenn s1 > s2.

---

## string.CompareText(s1, s2)
- **Konkret:**
  Vergleicht zwei Strings immer case-insensitiv.
  Kurzform von `string.StrComp(s1, s2, 1)`.
- **Parameter:**
  - `s1`: Erster String.
  - `s2`: Zweiter String.
- **Rückgabe:**
  `NumVal`
  `-1`, `0` oder `1`.

---

## string.Choose(n, v...)
- **Konkret:**
  Gibt das Element an Position `n` aus einer Liste zurück (1-basiert).
  Gibt `Undefined` zurück wenn der Index außerhalb liegt.
- **Parameter:**
  - `n`: Index (1-basiert).
  - `v...`: Beliebig viele Auswahloptionen.
- **Rückgabe:**
  Gewählter Wert oder `KindUndefined`.

---

## string.Switch(v...)
- **Konkret:**
  Wertet Bedingung-Wert-Paare aus und gibt den Wert des ersten wahren Paares zurück.
  Erwartet eine gerade Anzahl an Argumenten.
  Gibt `Undefined` zurück wenn keine Bedingung zutrifft.
- **Parameter:**
  - `v...`: Abwechselnd Bedingung und Wert (z. B. `cond1, val1, cond2, val2`).
- **Rückgabe:**
  Erster Wert dessen Bedingung `true` ist, oder `KindUndefined`.

---

## string.StrConv(input)
- **Konkret:**
  Konvertiert einen String sicher nach UTF-8.
  Ungültige Byte-Sequenzen werden durch `U+FFFD` (Replacement Character) ersetzt.
- **Parameter:**
  - `input`: String mit potenziell ungültigen Sequenzen.
- **Rückgabe:**
  `StrVal` (valides UTF-8).

---

## string.TxtToSqlInsert(data, table, dialect [, extras, columns, batchSize])
- **Konkret:**
  Generiert SQL-INSERT-Statements aus einem Array.
  Umschließt die Ausgabe mit Transaktionsblöcken je nach Dialekt.
  Werte werden automatisch escaped (`'` → `''`).
- **Parameter:**
  - `data`: `ArrVal` mit einzufügenden Werten (ein Wert pro Zeile).
  - `table`: Zielterbellenname.
  - `dialect`: SQL-Dialekt. Unterstützt: `"sqlite"`, `"postgres"`, `"mssql"`.
  - `extras`: Optional. `ArrVal` – zusätzliche Spalten-Werte pro Zeile.
  - `columns`: Optional. `ArrVal` – Spaltennamen für den INSERT-Header.
  - `batchSize`: Optional. `NumVal` – Anzahl Zeilen pro INSERT-Statement (0 = kein Batching).
- **Rückgabe:**
  `StrVal`
  Fertiges SQL-Script.

---

## string.Space(n)
- **Konkret:**
  Gibt einen String mit `n` Leerzeichen zurück.
- **Parameter:**
  - `n`: Anzahl Leerzeichen. Negative Werte ergeben einen leeren String.
- **Rückgabe:**
  `StrVal`

---

## string.StrDup(n, char)
- **Konkret:**
  Gibt ein einzelnes Zeichen `n`-mal wiederholt zurück.
  Wird ein mehrzeichen String übergeben, wird nur das erste Zeichen verwendet.
- **Parameter:**
  - `n`: Anzahl Wiederholungen. Negative Werte ergeben einen leeren String.
  - `char`: Zu wiederholendes Zeichen.
- **Rückgabe:**
  `StrVal`
  Beispiel: `string.StrDup(5, "*")` → `"*****"`.

---

## string.Base32Encode(val)
- **Konkret:**
  Wandelt Text in Base32 um (Standard-Alphabet, RFC 4648, mit Padding).
- **Parameter:**
  - `val`: Zu kodierender Text.
- **Rückgabe:**
  `StrVal`
  Beispiel: `"Hello"` → `"JBSWY3DP"`.

---

## string.Base32Decode(base32Str)
- **Konkret:**
  Wandelt einen Base32-String zurück in Text.
- **Parameter:**
  - `base32Str`: Base32-kodierter String (Standard-Alphabet mit Padding).
- **Rückgabe:**
  `StrVal`, `ErrorVal` bei ungültigem Base32-String.