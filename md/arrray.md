# 📋 array.* – Array-Funktionen

Dient zur Erstellung, Manipulation und Auswertung von ein- und zweidimensionalen Arrays.
Funktioniert mit `KindArr` (1D) und `KindArr2D` (2D). Viele Operationen sind mutierend (In-Place).
Export/Import als Datei wird für CSV und XLSX unterstützt.

---

## array.Create([args...])
- **Konkret:**
  Erstellt Arrays in vier Varianten abhängig von den übergebenen Argumenten.
- **Varianten:**
  - `array.Create(10)` – 1D-Array mit 10 Elementen, alle `0`.
  - `array.Create(3, 3)` – Leere 3×3-Matrix (`KindArr2D`), alle `0`.
  - `array.Create({1,2}, {3,4})` – 2D-Matrix aus Array-Literalen.
  - `array.Create(1, 2, "Test")` – 1D-Array aus den übergebenen Werten.
- **Rückgabe:**
  `ArrVal` oder `KindArr2D`.

---

## array.Add(arr, val)
- **Konkret:**
  Fügt einen Wert am Ende eines 1D-Arrays ein (In-Place).
- **Parameter:**
  - `arr`: 1D-Array.
  - `val`: Einzufügender Wert.
- **Rückgabe:**
  `ArrVal` (das modifizierte Array).

---

## array.Append(arr, value)
- **Konkret:**
  Alias für `array.Add`.
- **Rückgabe:**
  `ArrVal`

---

## array.Prepend(arr, value)
- **Konkret:**
  Fügt einen Wert am Anfang eines Arrays ein (Index 0).
- **Parameter:**
  - `arr`: 1D-Array.
  - `value`: Einzufügender Wert.
- **Rückgabe:**
  `ArrVal`

---

## array.Insert(arr, index, value)
- **Konkret:**
  Fügt einen Wert an einer bestimmten Position ein (In-Place).
  Alle nachfolgenden Elemente werden verschoben.
  Bei negativem Index wird am Anfang eingefügt.
  Bei Index größer als die Länge wird am Ende angehängt.
- **Parameter:**
  - `arr`: 1D-Array.
  - `index`: Zielposition (0-basiert).
  - `value`: Einzufügender Wert.
- **Rückgabe:**
  `ArrVal`

---

## array.Remove(arr, value)
- **Konkret:**
  Entfernt das erste Vorkommen eines Wertes aus dem Array (In-Place).
  Vergleich erfolgt als String.
- **Parameter:**
  - `arr`: 1D-Array.
  - `value`: Zu entfernender Wert.
- **Rückgabe:**
  `ArrVal`

---

## array.RemoveAt(arr, index)
- **Konkret:**
  Entfernt das Element an einer bestimmten Position (In-Place).
  Ignoriert ungültige Indizes ohne Fehler.
- **Parameter:**
  - `arr`: 1D-Array.
  - `index`: Position (0-basiert).
- **Rückgabe:**
  `ArrVal`

---

## array.Clear(array)
- **Konkret:**
  Leert ein bestehendes Array (In-Place). Die Variable bleibt ein Array, enthält aber keine Elemente mehr.
- **Parameter:**
  - `array`: 1D-Array.
- **Rückgabe:**
  `ArrVal` (leer)

---

## array.Count(array [, dim])
- **Konkret:**
  Gibt die Anzahl der Elemente zurück.
  Bei 2D-Arrays: `dim=1` = Zeilen (Standard), `dim=2` = Spalten.
  Bei Strings: Zeichenanzahl.
- **Parameter:**
  - `array`: 1D- oder 2D-Array.
  - `dim`: Optional. Dimension für 2D-Arrays (Standard: `1`).
- **Rückgabe:**
  `NumVal`

---

## array.Length(array [, dim])
- **Konkret:**
  Alias für `array.Count`.
- **Rückgabe:**
  `NumVal`

---

## array.LBound(arr [, dim])
- **Konkret:**
  Gibt die Untergrenze der angegebenen Dimension zurück.
  In VBMini immer `0` (0-basierte Arrays).
  Gibt `-1` zurück wenn das Array leer ist oder die Dimension nicht existiert.
- **Parameter:**
  - `arr`: 1D- oder 2D-Array.
  - `dim`: Optional. Dimension (Standard: `1`).
- **Rückgabe:**
  `NumVal`

---

## array.UBound(arr [, dim])
- **Konkret:**
  Gibt die Obergrenze der angegebenen Dimension zurück (letzter gültiger Index).
  Gibt `-1` zurück wenn das Array leer ist oder die Dimension nicht existiert.
- **Parameter:**
  - `arr`: 1D- oder 2D-Array.
  - `dim`: Optional. Dimension (Standard: `1`).
- **Rückgabe:**
  `NumVal`

---

## array.First(array)
- **Konkret:**
  Gibt das erste Element zurück.
  Bei 2D-Arrays: gibt die erste Zeile als `ArrVal` zurück.
- **Parameter:**
  - `array`: 1D- oder 2D-Array.
- **Rückgabe:**
  Erstes Element oder `NullVal` wenn leer.

---

## array.Last(array)
- **Konkret:**
  Gibt das letzte Element zurück.
  Bei 2D-Arrays: gibt die letzte Zeile als `ArrVal` zurück.
- **Parameter:**
  - `array`: 1D- oder 2D-Array.
- **Rückgabe:**
  Letztes Element oder `NullVal` wenn leer.

---

## array.Contains(arr, value)
- **Konkret:**
  Prüft ob ein Wert im Array enthalten ist.
  Vergleich erfolgt als String.
- **Parameter:**
  - `arr`: 1D-Array.
  - `value`: Gesuchter Wert.
- **Rückgabe:**
  `BoolVal`

---

## array.IndexOf(arr, value)
- **Konkret:**
  Gibt die Position des ersten Treffers zurück (0-basiert).
  Vergleich erfolgt als String.
- **Parameter:**
  - `arr`: 1D-Array.
  - `value`: Gesuchter Wert.
- **Rückgabe:**
  `NumVal`
  Position des Treffers, oder `-1` wenn nicht gefunden.

---

## array.Find(arr, pattern [, case])
- **Konkret:**
  Filtert ein Array mit Wildcard-Muster (`*`, `?`).
  Ohne Wildcards wird `Contains` genutzt.
  Suche ist standardmäßig case-insensitiv.
- **Parameter:**
  - `arr`: 1D-Array.
  - `pattern`: Suchmuster (z. B. `"Log_*.txt"`).
  - `case`: Optional. `BoolVal` – bei `true` case-sensitiv.
- **Rückgabe:**
  `ArrVal` aller Treffer.

---

## array.Sort(array)
- **Konkret:**
  Sortiert ein Array und gibt eine Kopie zurück (nicht In-Place).
  Zahlen werden numerisch sortiert, alles andere lexikografisch.
- **Parameter:**
  - `array`: 1D-Array.
- **Rückgabe:**
  `ArrVal` (sortierte Kopie)

---

## array.Reverse(array)
- **Konkret:**
  Gibt ein neues Array mit umgekehrter Reihenfolge zurück (nicht In-Place).
- **Parameter:**
  - `array`: 1D-Array.
- **Rückgabe:**
  `ArrVal`

---

## array.Unique(array)
- **Konkret:**
  Entfernt doppelte Werte und gibt eine Kopie zurück.
  Vergleich ist typsicher (Zahl `1` ≠ String `"1"`).
- **Parameter:**
  - `array`: 1D-Array.
- **Rückgabe:**
  `ArrVal`

---

## array.Merge(array, array)
- **Konkret:**
  Verbindet zwei 1D-Arrays zu einem neuen Array.
- **Parameter:**
  - Zwei `ArrVal`.
- **Rückgabe:**
  `ArrVal`

---

## array.Clone(array)
- **Konkret:**
  Erstellt eine flache Kopie eines Arrays (Shallow Copy).
- **Parameter:**
  - `array`: 1D-Array.
- **Rückgabe:**
  `ArrVal`

---

## array.Chunk(arr, size)
- **Konkret:**
  Unterteilt ein 1D-Array in gleichgroße Teilarrays.
  Das letzte Chunk kann kleiner sein wenn die Gesamtlänge nicht aufgeht.
- **Parameter:**
  - `arr`: 1D-Array.
  - `size`: Maximale Größe je Teilarray.
- **Rückgabe:**
  `KindArr2D`

---

## array.Join(arr, sep)
- **Konkret:**
  Verbindet alle Elemente eines Arrays zu einem String.
- **Parameter:**
  - `arr`: 1D-Array.
  - `sep`: Trennzeichen.
- **Rückgabe:**
  `StrVal`

---

## array.Split(str, sep)
- **Konkret:**
  Teilt einen String anhand eines Trennzeichens in ein Array.
- **Parameter:**
  - `str`: Quellstring.
  - `sep`: Trennzeichen (darf nicht leer sein).
- **Rückgabe:**
  `ArrVal`

---

## array.CleanArray(array)
- **Konkret:**
  Trimmt alle String-Elemente und entfernt Steuerzeichen unter ASCII 32 (außer Tab).
  Funktioniert mit 1D- und 2D-Arrays.
- **Parameter:**
  - `array`: 1D- oder 2D-Array.
- **Rückgabe:**
  `ArrVal` oder `KindArr2D` (bereinigt)

---

## array.IsEmpty(array)
- **Konkret:**
  Prüft ob ein Array leer ist oder kein Array ist.
- **Parameter:**
  - `array`: Zu prüfender Wert.
- **Rückgabe:**
  `BoolVal`
  `true` wenn leer oder kein Array.

---

## array.KindOf(value)
- **Konkret:**
  Gibt den Datentyp eines Wertes als lesbaren String zurück.
- **Parameter:**
  - `value`: Beliebiger Wert.
- **Rückgabe:**
  `StrVal`
  Mögliche Werte: `"Zahl (Double)"`, `"Text (String)"`, `"Boolean"`, `"Null/Nothing"`, `"Array (1D)"`, `"Array (2D)"`, `"Objekt"`, `"Fehler-Objekt"`, `"unbekannter Typ"`.

---

## array.ToCSV(path, data [, sep, exclude, append])

- **Konkret:**
  Speichert ein 1D- oder 2D-Array als CSV-Datei.
  Bei einem 1D-Array aus verschachtelten Arrays wird jedes innere Array als eigene Zeile geschrieben.
  Bei einem flachen 1D-Array aus einfachen Werten (Strings, Zahlen) wird jedes Element als eigene Zeile mit genau einer Zelle geschrieben.

  Mit `exclude` können Zeilen ausgeschlossen werden. Eine Zeile wird nicht geschrieben, wenn mindestens eine ihrer Zellen einen Wert aus dem Ausschluss-Array {} enthält.

  Mit `append=True` werden die Daten an eine bereits vorhandene CSV-Datei angehängt. Ist die Datei noch nicht vorhanden, wird sie automatisch erstellt.

- **Parameter:**
  - `path`: Zieldatei.
  - `data`: `ArrVal` oder `KindArr2D`.
  - `sep`: Optional. Trennzeichen (Standard: `;`).
  - `exclude`: Optional. Array {} mit Werten, deren Vorkommen in einer beliebigen Zelle zum Überspringen der gesamten Zeile führt.
  - `append`: Optional. Boolean. Bei `True` werden die Daten an die bestehende Datei angehängt (Standard: `False`).

---

## array.FromCSV(path [, sep, exclude])

- **Konkret:**
  Lädt eine CSV-Datei in ein 2D-Array.

  Bereinigt automatisch ungültige Steuerzeichen und Null-Bytes.
  Konvertiert Windows-1252 nach UTF-8 falls nötig.
  Toleriert ungleichmäßige Spaltenanzahl und einsame Anführungszeichen.

  Mit `exclude` können komplette Zeilen beim Einlesen ausgeschlossen werden. Eine Zeile wird übersprungen, wenn mindestens eine ihrer Zellen einen Wert aus dem Ausschluss-Array {} enthält.

- **Parameter:**
  - `path`: Quelldatei.
  - `sep`: Optional. Trennzeichen (Standard: `;`).
  - `exclude`: Optional. Array mit Werten, deren Vorkommen in einer beliebigen Zelle zum Überspringen der gesamten Zeile führt.

 ---

 ## array.ToXLSX(path, data [, sheetName, exclude, append])

- **Konkret:**
  Speichert ein 1D- oder 2D-Array als XLSX-Datei (Excel-Format), ohne dass Excel oder eine andere Office-Anwendung installiert sein muss.
  Bei einem 1D-Array aus verschachtelten Arrays wird jedes innere Array als eigene Zeile geschrieben.
  Bei einem flachen 1D-Array aus einfachen Werten (Strings, Zahlen) wird jedes Element als eigene Zeile mit genau einer Zelle geschrieben.
  Die Funktion unterstützt mehrere Tabellenblätter innerhalb derselben XLSX-Datei.
  Wird eine bereits vorhandene XLSX-Datei angegeben, kann ein weiteres Tabellenblatt hinzugefügt werden.
  Standardmäßig wird ein bereits vorhandenes Tabellenblatt mit demselben Namen gelöscht und anschließend neu erstellt.

  Mit `append=True` bleibt das vorhandene Tabellenblatt erhalten und die neuen Daten werden ab der ersten freien Zeile angehängt. Existiert das Tabellenblatt noch nicht, wird es neu erstellt.

  Mit `exclude` können Zeilen ausgeschlossen werden. Eine Zeile wird nicht geschrieben, wenn mindestens eine ihrer Zellen einen Wert aus dem Ausschluss-Array {} enthält.

  Nutzt intern die Go-Bibliothek `excelize` zur direkten Erzeugung und Bearbeitung des XLSX-Dateiformats (ZIP+XML).

- **Parameter:**
  - `path`: Zieldatei.
  - `data`: `ArrVal` oder `KindArr2D`.
  - `sheetName`: Optional. Name des Tabellenblatts (Standard: `"Sheet1"`).
  - `exclude`: Optional. Array {} mit Werten, deren Vorkommen in einer beliebigen Zelle zum Überspringen der gesamten Zeile führt.
  - `append`: Optional. Boolean. Bei `True` werden die Daten an das vorhandene Tabellenblatt angehängt (Standard: `False`).

---

## array.FromXLSX(path [, sheetName, exclude, column])

- **Konkret:**
  Lädt eine XLSX-Datei in ein Array.

  Ohne Angabe von `sheetName` wird das erste Tabellenblatt der Datei gelesen.
  Enthält das Tabellenblatt nur eine Spalte, wird ein 1D-Array (`KindArr`) zurückgegeben.
  Enthält das Tabellenblatt mehrere Spalten, wird ein 2D-Array (`KindArr2D`) zurückgegeben.

  Mit `exclude` können komplette Zeilen beim Einlesen ausgeschlossen werden. Eine Zeile wird übersprungen, wenn mindestens eine ihrer Zellen einen Wert aus dem Ausschluss-Array {} enthält.

  Mit `column` kann gezielt eine einzelne Spalte aus einer mehrspaltigen Tabelle eingelesen werden. Die Spaltennummer ist 0-basiert, wobei `0` der ersten Spalte (A), `1` der zweiten Spalte (B) usw. entspricht. Bei Verwendung von `column` wird ein 1D-Array (`KindArr`) zurückgegeben.

- **Parameter:**

  - `path`: Quelldatei.
  - `sheetName`: Optional. Name des zu lesenden Tabellenblatts (Standard: erstes Blatt).
  - `exclude`: Optional. Array mit Werten, deren Vorkommen in einer beliebigen Zelle zum Überspringen der gesamten Zeile führt.
  - `column`: Optional. 0-basierte Spaltennummer. Wird nur eine bestimmte Spalte einer mehrspaltigen Tabelle benötigt.