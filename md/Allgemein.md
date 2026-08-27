# 📖 VBX – Kurzreferenz

VBX ist eine modulare, in Go geschriebene Runtime für eine VB-Skriptsprache. Schlanker Kern, Komplexität wird in Module ausgelagert.

---

## Architektur

* **Core Runtime** – Sprachlogik, Kontrollstrukturen, Standardfunktionen
* **Modul-System** – Erweiterungen (Netzwerk, Kryptografie, Datenbanken, ...)
* **Shell-Interface** – Direkte Ausführung von Funktionen über die Kommandozeile

Über **650 Funktionen**, ohne den Kern aufzublähen.

---

## Aufruf

```bash
vbx <quelle.vb>
vbx <quelle.vbc>
vbx -modules=json,net <quelle.vb>

vbx -h
vbx -shell -h
vbx -const -h
vbx -modules=file -h
```

---

## Besonderheiten

**Verschlüsselte Skripte** – `vbx Build server.vb` erzeugt `server.vbc`: verschlüsselter Quelltext, Includes bereits aufgelöst, Zielsystem braucht nur die Runtime.

**Hintergrundprozesse** – `vbx Worker server.vb` startet einen eigenen Prozess (`.vb`/`.vbc`), liefert die Prozess-ID zurück, automatische Log-Datei.

**Ausführungsschutz** – prüft jede Datei vor Ausführung (Binärdaten, Encoding, Fehl-Downloads wie HTML-Fehlerseiten), parst das komplette Skript vorab.

**Shell-Interface** – `vbx -shell -h` zeigt direkt aufrufbare Funktionen.

---

## Modul-System

Standardmodule sind immer verfügbar. Weitere Module explizit aktivieren:

```vbx
#use json,net,crypt
```

oder per CLI: `vbx -modules=json,net script.vb`
Modul-Hilfe: `vbx -modules=json -h`

---

## Direktiven

Müssen am Anfang des Skripts stehen.

```vbx
1. #use zip
2. #requires 1.0.23
```

`#use` – lädt optionale Module. `#requires` – Mindestversion der Runtime.

---

## Include

```vbx
include "tools.vb"
include "lib/helpers.vb"
```

Relative Pfade, einmaliges Laden, rekursive Includes werden erkannt und Include-Schleifen führen zu einem Fehler.

---

## Kommentare

```vbx
' Einzeilig
/' Mehrzeilig '/
```

---

## Datentypen

| Typ | Beschreibung |
|---|---|
| Number | Ganzzahlen und Fließkommazahlen |
| String | Zeichenketten |
| Boolean | Wahr/Falsch |
| Array | Eindimensionales Array |
| Array2D | Zweidimensionales Array |
| Map | Schlüssel/Wert-Struktur |
| Null | Leerer Wert |

---

## Operatoren

| Kategorie | Operatoren |
|---|---|
| Arithmetisch | `+` `-` `*` `/` |
| Vergleich | `=` `<>` `<` `>` `<=` `>=` |
| Logisch | `And` `Or` `Not` |
| String | `&` (Verkettung) |
| Erweiterte Zuweisung | `+=` `-=` `*=` `/=` |

---

## Klammern

| Typ | Verwendung |
|---|---|
| `( )` | Funktionsaufruf, Array-Zugriff via Index |
| `[ ]` | Map-Zugriff via Key, **nur lesend**, verkettbar |
| `{ }` | Array-Literal |

`[ ]` ist ausschließlich für Maps und nur lesend – schreibend dafür `map.Set(...)` verwenden.

---

## Variablen

**Lokal (`Dim`)** – nur im aktuellen Scope sichtbar, Standardwert `0`, mehrere Deklarationen pro Zeile möglich. In Function/Sub können gleichnamige Variablen mit eigenem Wert existieren (Shadowing, VBX gibt dabei einen Hinweis aus).

**Global (`Public`)** – im gesamten Skript sichtbar, wird in Function/Sub genutzt, wenn keine lokale Variable gleichen Namens existiert.

---

## Optionale Parameter

`Sub`/`Function` unterstützen optionale Parameter per `Optional name = wert` (müssen nach allen Pflichtparametern stehen). Auch `= wert` ohne das Schlüsselwort macht einen Parameter optional. Der Default-Ausdruck wird bei jedem Aufruf neu ausgewertet und kann auf vorherige Parameter zugreifen. Bei zu wenigen/zu vielen Argumenten liefert VBX einen Fehler mit der erwarteten Argumentanzahl (`min`–`max`).

---

## Fehlerbehandlung

Funktionen, die scheitern können, geben einen Fehlerwert (`ErrorVal`) statt eines regulären Werts zurück. Wird dieser direkt in Bedingung/Berechnung/Verkettung verwendet, bricht das Skript sofort ab – daher zuerst einer Variable zuweisen und prüfen.

`IsError(wert)` – prüft, ob `wert` ein Fehlerwert ist.
`ErrorText(wert)` – gibt den Fehlertext als String zurück (leer, falls kein Fehler).

Beide sind Sprach-Kernfunktionen, immer verfügbar, unabhängig von `#use`.

---

## Maps

Entstehen z. B. über `json.FromJSON` oder `map.Create`. Lesender Zugriff über `[ ]` (auch verkettbar, z. B. `arr(i)["key"]`). Schreibend nur über `map.Set(map, key, wert)`.

---

## Print & Farben

```vbx
Print "Wert: " & x
Print vbRed() & "Fehler" & vbNormal()
```

---

## Konstanten

| Kategorie | Beispiele |
|---|---|
| Logik | `vbTrue()`, `vbFalse()`, `vbNullString()` |
| Formatierung | `vbCrLf()`, `vbNewLine()`, `vbTab()` |
| Farben | `vbBlack()`, `vbRed()`, `vbGreen()`, `vbYellow()`, `vbBlue()`, `vbWhite()`, `vbCyan()`, `vbMagenta()`, `vbLightGray()`, `vbGray()` |
| Hintergrundfarben | `vbBgBlack()`, `vbBgRed()`, `vbBgGreen()`, `vbBgYellow()`, `vbBgBlue()`, `vbBgWhite()`, `vbBgCyan()`, `vbBgMagenta()` |
| Stile | `vbBold()`, `vbUnderline()`, `vbNormal()` |

---

## Kontrollstrukturen

| Struktur | Syntax-Skelett | Abschluss |
|---|---|---|
| If | `If bed Then` … `[ElseIf bed Then …]` `[Else …]` | `End If` |
| Select Case | `Select Case ausdruck` `Case wert / wert1, wert2 / x To y / Is > x` `[Case Else]` | `End Select` |
| For | `For i = start To end [Step n]` | `Next [i]` |
| For Each | `For Each [k,] v In array / map` | `Next` |
| While | `While bed` | `End While` |
| Do Loop | `Do [While/Until bed]` … `Loop [While/Until bed]` | `Loop` |
| Exit | `Exit For` / `Exit While` / `Exit Do` / `Exit Sub` / `Exit Function` | – |
| Sub | `Sub Name(param1, param2 [, Optional param3 = wert])` | `End Sub` |
| Function | `Function Name(...)` … `Return wert` oder `Name = wert` | `End Function` |
| Cls | `Cls()` | – |
| Print | `Print wert` | – |