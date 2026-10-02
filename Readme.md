# 📖 VBX – Referenz

VBX ist eine modulare, in Go geschriebene Runtime für eine VB-Skriptsprache. Sie vereint die vertraute Einfachheit von Visual Basic mit der Performance und Portabilität moderner System-Tools. Der Kern bleibt schlank und stabil – Komplexität wird in Module ausgelagert.

---

# Architektur

VBX besteht aus drei Hauptkomponenten:

* **Core Runtime** – Sprachlogik, Kontrollstrukturen und Standardfunktionen
* **Modul-System** – Erweiterungen für Netzwerk, Kryptografie, Datenbanken und mehr
* **Shell-Interface** – Direkte Ausführung von Funktionen über die Kommandozeile

Über **575 Funktionen** stehen zur Verfügung, ohne den Kern unnötig aufzublähen.

---

# Aufruf

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

# Besonderheiten von VBX

## Verschlüsselte Skripte

VBX kann Skripte in verschlüsselter Form bereitstellen.

```vbx
vbx Build server.vb
```

Erzeugt:

```text
server.vbc
```

Eigenschaften:

* verarbeitet alle Includes automatisch
* verschlüsselt den vollständigen Quelltext
* erzeugt einen Magic Header
* Quelltext ist nicht direkt lesbar
* Zielsystem benötigt nur die Runtime

---

## Hintergrundprozesse

VBX kann Skripte direkt als Hintergrundprozess starten.

```vbx
vbx Worker server.vb
```

Eigenschaften:

* startet einen eigenen Prozess
* liefert die Prozess-ID zurück
* automatische Log-Datei
* unterstützt `.vb` und `.vbc`
* ideal für Dienste und Daemons

---

## Ausführungsschutz

VBX prüft jede Datei vor der Ausführung – nicht nur anhand der Dateiendung.

Eigenschaften:

* erkennt Binärdaten, ungültiges Encoding und leere Dateien
* erkennt typische Fehl-Downloads (z. B. eine HTML-Fehlerseite oder eine GitHub/GitLab-API-JSON-Antwort statt der eigentlichen Rohdatei)
* parst das komplette Skript, bevor auch nur eine Zeile ausgeführt wird
* Syntaxfehler werden sauber abgefangen, statt den Prozess abstürzen oder hängen zu lassen
* besonders relevant für automatisierte Update-/Deploy-Skripte

---

## Shell-Interface

Funktionen können direkt über die Kommandozeile ausgeführt werden.

```bash
vbx -shell -h
```

---

# Modul-System

Standardmodule sind immer verfügbar. Komplexere Funktionen müssen explizit aktiviert werden.

Im Skript:

```vbx
#use json,net,crypt
```

Per CLI:

```bash
vbx -modules=json,net script.vb
```

Modul-Hilfe:

```bash
vbx -modules=json -h
```

---

# Direktiven

Direktiven müssen am Anfang des Skripts stehen.

```vbx
#use zip
#requires 1.0.23
```

**#use** – Lädt optionale Module.

**#requires** – Definiert eine Mindestversion der Runtime. Ist die installierte Version älter, wird die Ausführung abgebrochen.

---

# Include

Bindet weitere Dateien ein.

```vbx
include "tools.vb"
include "lib/helpers.vb"
```

* relative Pfade werden unterstützt
* Dateien werden nur einmal geladen
* rekursive Includes werden erkannt
* Include-Schleifen führen zu einem Fehler

---

# Namespaces

## Permanente Namespaces

Immer verfügbar:
[app.*](md/app.md), [array.*](md/array.md), [date.*](md/date.md), [file.*](md/file.md), [folder.*](md/folder.md), [global.*](md/global.md), [math.*](md/math.md)

## Optionale Module

[ad.*](md/ad.md), [cert.*](md/cert.md), [computer.*](md/computer.md), [convert.*](md/convert.md), [crypt.*](md/crypt.md), [data.*](md/data.md), [db.*](md/db.md), [debug.*](md/debug.md), [docker.*](md/docker.md), [env.*](md/env.md), [export.*](md/export.md), [fin.*](md/fin.md), [geo.*](md/geo.md), [git.*](md/git.md), [hash.*](md/hash.md), [ini.*](md/ini.md), [json.*](md/json.md), [map.*](md/map.md), [net.*](md/net.md), [pgp.*](md/pgp.md), [picture.*](md/picture.md), [pqc.*](md/pqc.md), [proc.*](md/proc.md), [rand.*](md/rand.md), [reg.*](md/reg.md), [service.*](md/service.md), [sftp.*](md/sftp.md), [smtp.*](md/smtp.md), [ssh.*](md/ssh.md), [steg.*](md/steg.md), [string.*](md/string.md), [tar.*](md/tar.md), [template.*](md/template.md), [uptime-kuma.*](md/uptime-kuma.md), [win.*](md/win.md), [xml.*](md/xml.md), [yaml.*](md/yaml.md), [zip.*](md/zip.md)

**Plugins:** crypt.*, data.*, docker.*, fin.*, ini.*, media.*, pgp.*, pqc.*, qr.*, rand.*, steg.*, tar.*, vault.*, xml.*, yaml.*, zip.*

Plugins werden im Verzeichnis `plugins` neben der VBX-Runtime gesucht. Alternativ kann das Plugin-Verzeichnis über `VBX_PLUGIN_PATH` festgelegt werden.

---

## Plugin-Cache

Die kompilierten WASM-Plugins werden von VBX in einem Compilation-Cache gespeichert.

Standardmäßig verwendet VBX dafür:

```text
<vbx-user-cache>/vbx/plugin-cache
C:\Users\<username>\AppData\Local\vbx\plugin-cache
```

Alternativ kann das Plugin-Cache über `VBX_PLUGIN_CACHE` festgelegt werden.

---

# Kommentare

```vbx
' Einzeilig

/' Mehrere
   Zeilen '/
```

---

# Datentypen

VBX verwendet dynamische Typisierung. Der Typ wird automatisch aus dem Wert bestimmt.

| Typ     | Beschreibung                                           |
| ------- | ------------------------------------------------------ |
| Number  | Ganzzahlen und Fließkommazahlen                        |
| String  | Zeichenketten                                          |
| Boolean | Wahr/Falsch                                            |
| Array   | Eindimensionales Array                                 |
| Array2D | Zweidimensionales Array                                |
| Map     | Schlüssel/Wert-Struktur                                |
| Null    | Leerer Wert, Literal `Nothing`                         |

## Nothing

`Nothing` ist das Literal für den leeren Wert. Geprüft wird mit `IsNothing(wert)` oder `IsNull(wert)` (beide verhalten sich gleich), oder per Vergleich.

```vbx
Dim n = Nothing
Print IsNothing(n)    ' true
Print n = Nothing     ' true
```

---

# Operatoren

### Arithmetisch

| Operator | Bedeutung      |
|----------|----------------|
| `+`      | Addition       |
| `-`      | Subtraktion    |
| `*`      | Multiplikation |
| `/`      | Division       |

### Vergleich

| Operator | Bedeutung      |
|----------|----------------|
| `=`      | Gleich         |
| `<>`     | Ungleich       |
| `<`      | Kleiner        |
| `>`      | Größer         |
| `<=`     | Kleiner/Gleich |
| `>=`     | Größer/Gleich  |

### Logisch

| Operator | Bedeutung |
|----------|-----------|
| `And`    | UND       |
| `Or`     | ODER      |
| `Not`    | NICHT     |

### String

| Operator | Bedeutung   |
|----------|-------------|
| `&`      | Verkettung  |

### Erweiterte Zuweisung

| Operator | Bedeutung                    |
|----------|-------------------------------|
| `+=`     | Addieren und zuweisen        |
| `-=`     | Subtrahieren und zuweisen    |
| `*=`     | Multiplizieren und zuweisen  |
| `/=`     | Dividieren und zuweisen      |

## Verhalten der Operatoren

**`+`** richtet sich nach den Operanden:

| Operanden | Ergebnis |
|---|---|
| Zahl + Zahl | Addition |
| Text + Text | Verkettung |
| Zahl + Text, Text als Zahl lesbar | Addition |
| Zahl + Text, Text nicht als Zahl lesbar | Verkettung |

```vbx
Print 2 + 3            ' 5
Print "12" + "30"      ' 1230
Print "12" + 30        ' 42
Print "Name: " + 5     ' Name: 5
```

**`-` `*` `/`** wandeln als Zahl lesbare Texte um (`7 - "2"` ergibt `5`), sonst gibt es einen Fehler. Division durch 0 ist ein Fehler.

**Vergleiche:** `<` `>` `<=` `>=` wandeln lesbare Texte ebenfalls in Zahlen um (`2 <= "2"` ist wahr). `=` und `<>` vergleichen, sobald ein Operand ein Text ist, als Text – die Groß-/Kleinschreibung zählt.

**`&`** verkettet immer als String.

**`And` / `Or`** werten den rechten Operanden nur aus, wenn das Ergebnis nicht schon feststeht (Short-Circuit):

```vbx
If x <> 0 And 10 / x > 1 Then   ' keine Division durch 0, wenn x = 0
    Print "ok"
End If
```

**Erweiterte Zuweisung:** `a += b` ist gleichbedeutend mit `a = a + b` (entsprechend `-=`, `*=`, `/=`). Mit Texten gelten dieselben Regeln wie bei den normalen Operatoren (`s += " Welt"` hängt Text an). Bei einem Fehler bleibt die Variable unverändert.

---

# Klammern

| Typ   | Verwendung                                      |
|-------|--------------------------------------------------|
| `( )` | Funktionsaufruf, Array-Zugriff via Index         |
| `[ ]` | Map-Zugriff via Key, nur lesend, verkettbar      |
| `{ }` | Array-Literal (statische Listenwerte)            |

```vbx
Left("Hallo", 3)       ' Funktionsaufruf
arr(0)                 ' Array-Zugriff
arr(0,1)               ' 2D Array-Zugriff
user["mail"]           ' Map-Zugriff (lesend)
grp(i)["path"]         ' Kombiniert: Array-Index, dann Map-Key
config["db"]["host"]   ' Verkettung mehrerer Map-Zugriffe
Dim farben = {"rot", "grün"}  ' Array-Literal
```

`[ ]` ist ausschließlich für Maps gedacht und nur lesend. Zuweisungen wie `user["mail"] = "x"` sind ein Syntaxfehler – dafür `map.Set(user, "mail", "x")` verwenden. Array-Zugriff (1D und 2D) läuft weiterhin über `( )`.

---

# Variablen

Variablen müssen vor der Verwendung deklariert werden.

## Lokale Variablen (Dim)

```vbx
Dim name = "Max"

Dim arr(10)
arr(0) = 1

Dim matrix(3, 3)
matrix(0,1) = 5

Dim farben = {"rot", "grün"}
```

* nur im aktuellen Scope sichtbar
* Standardwert ohne Zuweisung ist `0`
* mehrere Deklarationen pro Zeile möglich
* In Function/Sub können gleichnamige Variablen mit unterschiedlichen Werten existieren

## Globale Variablen (Public)

```vbx
Public counter = 0

Public arr(10)
arr(0) = 1

Public matrix(3, 3)
matrix(0,1) = 5
```

* im gesamten Skript sichtbar
* wird in Function/Sub genutzt wenn keine lokale Variable mit gleichem Namen existiert

## Konstanten (Const)

```vbx
Const maxTries = 3
```

* wie `Dim`, aber schreibgeschützt
* ein Initialwert ist zwingend erforderlich
* eine spätere Zuweisung (auch `+=` etc.) bricht das Skript mit einem Fehler ab
* keine Arrays
* in Function/Sub kann eine lokale `Const` denselben Namen wie eine äußere `Const`/Variable tragen (Shadowing, ohne Hinweis-Ausgabe)
* nicht zu verwechseln mit den vb-Konstanten (siehe Abschnitt „Konstanten“ unten) – `vbx -const -h` zeigt Letztere

## Sichtbarkeit in Function und Sub

Ein `Dim` auf oberster Ebene des Skripts ist in allen Function/Sub sichtbar (lesen und ändern), genau wie `Public`. Function/Sub sehen **nie** die lokalen Variablen ihres Aufrufers:

```vbx
Dim g = 7

Sub Zeige()
    Print g         ' 7 – Variable der obersten Ebene
End Sub

Sub Aussen()
    Dim h = 5
    Innen()         ' Fehler: h ist in Innen() nicht sichtbar
End Sub

Sub Innen()
    Print h
End Sub
```

Ein benötigter Wert wird als Parameter übergeben oder `Public` deklariert.

---

# Shadowing

```vbx
Dim x = 10

Sub Test()
    Dim x = 99
    Print x     ' 99
End Sub

Test()
Print x         ' 10
```

VBX gibt beim Shadowing einen Hinweis aus.

---

# Arrays

`Dim a(n)` legt ein Array mit n+1 Elementen an (Index 0 bis n), `Dim m(r, c)` ein 2D-Array. Alle Elemente starten mit `0`.

```vbx
Dim a(2)           ' Elemente 0, 1, 2
a(1) = "x"
Print a(1)         ' x

Dim m(1, 2)
m(1, 0) = 5
Print m(1, 0)      ' 5
Print m(1)         ' ganze Zeile als 1D-Array
```

* Lesen außerhalb der Grenzen und negative Indizes sind ein Fehler
* Schreiben hinter das Ende eines 1D-Arrays vergrößert es automatisch, bei 2D-Arrays nicht

**Zuweisung teilt das Array.** `b = a` kopiert nicht: beide Variablen verweisen auf dasselbe Array (wie in VB.NET), eine Änderung über `b` ist auch über `a` sichtbar. Das gilt auch, wenn ein Array an eine Function/Sub übergeben wird. Wächst eines der beiden Arrays, ist es danach vom anderen getrennt.

```vbx
Dim a(2)
a(0) = 1
Dim b = a
b(0) = 99
Print a(0)         ' 99
```

---

# Optionale Parameter

Sub und Function unterstützen optionale Parameter mit Default-Wert über `Optional`.

```vbx
Sub Greet(name, Optional greeting = "Hallo")
    Print greeting & ", " & name & "!"
End Sub

Greet("Max")              ' Hallo, Max!
Greet("Max", "Servus")    ' Servus, Max!
```

* `Optional` ist kein reserviertes Wort, es wird nur in Parameterlisten erkannt
* Optionale Parameter müssen **nach** allen Pflichtparametern stehen
* `= wert` allein (ohne das Schlüsselwort `Optional`) macht einen Parameter ebenfalls optional
* Der Default-Ausdruck wird bei jedem Aufruf neu ausgewertet und kann auf vorherige Parameter derselben Signatur zugreifen:

```vbx
Sub ShowDouble(x, Optional y = x * 2)
    Print x & " / " & y
End Sub

ShowDouble(5)        ' 5 / 10
ShowDouble(5, 100)   ' 5 / 100
```

* Mehrere optionale Parameter sind erlaubt:

```vbx
Function Add(a, b, Optional c = 0, Optional d = 0)
    Add = a + b + c + d
End Function

Print Add(1, 2)          ' 3
Print Add(1, 2, 3)        ' 6
Print Add(1, 2, 3, 4)     ' 10
```

* Wird ein Aufruf mit zu wenigen Pflichtargumenten oder zu vielen Argumenten gemacht, liefert VBX einen Fehler mit der erwarteten Argumentanzahl (`min` bis `max`)

---

# Fehlerbehandlung

Funktionen, die scheitern können, geben einen Fehlerwert (`ErrorVal`) zurück statt eines regulären Wertes. Sobald dieser Wert verwendet wird – in einem Ausdruck, einer Bedingung, einer Zuweisung, bei `Dim` oder als eigenständiger Aufruf –, bricht das Skript mit einer Fehlermeldung ab. Ein Fehlerwert lässt sich also nicht in einer Variablen aufbewahren.

Fehler abfangen geht mit `Try/Catch` (siehe unten). Zum schnellen Prüfen eines einzelnen Aufrufs gibt es `IsError`:

```vbx
If IsError(Mid(text, 0)) Then
    Print "Ungültiger Startwert"
End If
```

`IsError(wert)` – Prüft, ob `wert` ein Fehlerwert ist. Gibt `true` oder `false` zurück. Funktioniert direkt auf einem Aufruf und auf der Catch-Variablen.
`ErrorText(wert)` – Gibt den Fehlertext eines Fehlerwerts als String zurück. Ist `wert` kein Fehler, wird ein leerer String zurückgegeben.
Beide sind Sprach-Kernfunktionen (keine Modul-Funktionen) und immer verfügbar, unabhängig von `#use`.

## Try / Catch / Finally

Fängt einen Skriptabbruch innerhalb eines Blocks ab, statt das ganze Skript zu beenden.

```vbx
#use json

Try
    Dim count = json.PropertyCount(obj)
    Print "Anzahl: " & count
Catch err
    Print "Fehler: " & ErrorText(err)
Finally
    Print "Wird immer ausgeführt"
End Try
```

* `Catch` ist optional, `Finally` ist optional – mindestens einer der beiden Zweige muss vorhanden sein
* `Finally` läuft in jedem Fall (Erfolg, abgefangener Fehler, oder ein Signal wie `Return`/`Exit For` aus dem Try-Block)
* `Try/Catch` fängt jeden Fehler im Block ab, auch bei `Dim x = RiskyCall()`, bei Zuweisungen und bei `+=`
* die Catch-Variable enthält den Fehlerwert, `ErrorText(err)` den Text
* bei verschachtelten Aufrufen enthält der Text die ganze Kette (`Fehler in Sub 'A': Fehler in Sub 'B': …`)

**Try und Schleifen:** Ein `Try` außen um eine Schleife bricht bei einem Fehler die **gesamte Schleife** ab (restliche Durchläufe werden nicht mehr erreicht), lässt das Skript danach aber normal weiterlaufen. Um nur den fehlerhaften Durchlauf zu überspringen und mit dem Rest der Schleife fortzufahren, muss `Try` **innerhalb** des Schleifenkörpers stehen:

```vbx
For Each x In files
    Try
        RiskyCall(x)
    Catch err
        Print "Fehler bei " & x & ": " & ErrorText(err)
    End Try
Next
```

---

# Maps

Maps speichern Schlüssel/Wert-Paare. Sie entstehen über Funktionen wie `json.FromJSON` oder `map.Create`.

```vbx
#use map

Dim user = map.Create()
map.Set(user, "name", "Max")
map.Set(user, "mail", "max@test.de")

Print user["name"]

For Each key, val In user
    Print key & " = " & val
Next
```

`[ ]` ist ausschließlich lesend und beliebig verkettbar (`x["a"]["b"]`, auch in Kombination mit Array-Index: `arr(i)["key"]`). Schreibender Zugriff via `[ ]` (`user["key"] = wert`) ist nicht unterstützt – dafür `map.Set` verwenden.

---

# Print

```vbx
Print "Hallo Welt"
Print x
Print "Wert: " & x
```

## Farben

```vbx
Print vbRed() & "Fehler" & vbNormal()
Print vbGreen() & "OK" & vbNormal()
Print vbBold() & vbYellow() & "Wichtig" & vbNormal()
```

---

# Konstanten

## Logik

| Konstante        | Wert    | Beschreibung  |
|------------------|---------|----------------|
| `vbTrue()`       | `true`  | Wahr          |
| `vbFalse()`      | `false` | Falsch        |
| `vbNullString()` | `""`    | Leerer String |

## Formatierung

| Konstante     | Wert   | Beschreibung              |
|---------------|--------|-----------------------------|
| `vbCrLf()`    | `\r\n` | Windows-Zeilenumbruch     |
| `vbNewLine()` | `\n`   | System-Zeilenumbruch      |
| `vbTab()`     | `\t`   | Tabulator                 |

## Farben

| Konstante       | Beschreibung |
|-----------------|--------------|
| `vbBlack()`     | Schwarz      |
| `vbRed()`       | Rot          |
| `vbGreen()`     | Grün         |
| `vbYellow()`    | Gelb         |
| `vbBlue()`      | Blau         |
| `vbWhite()`     | Weiß         |
| `vbCyan()`      | Cyan         |
| `vbMagenta()`   | Magenta      |
| `vbLightGray()` | Hellgrau     |
| `vbGray()`      | Grau         |

## Hintergrundfarben

| Konstante        | Beschreibung        |
|------------------|-----------------------|
| `vbBgBlack()`    | Hintergrund Schwarz |
| `vbBgRed()`      | Hintergrund Rot     |
| `vbBgGreen()`    | Hintergrund Grün    |
| `vbBgYellow()`   | Hintergrund Gelb    |
| `vbBgBlue()`     | Hintergrund Blau    |
| `vbBgWhite()`    | Hintergrund Weiß    |
| `vbBgCyan()`     | Hintergrund Cyan    |
| `vbBgMagenta()`  | Hintergrund Magenta |

## Stile

| Konstante       | Beschreibung                       |
|-----------------|---------------------------------------|
| `vbBold()`      | Fett                               |
| `vbUnderline()` | Unterstrichen                      |
| `vbNormal()`    | Alle Stile und Farben zurücksetzen |

---

# Reservierte Schlüsselwörter

Folgende Wörter sind reserviert und können nicht als Namen für Variablen, Funktionen, Subs oder Parameter verwendet werden (Groß-/Kleinschreibung spielt keine Rolle):

| Kategorie | Schlüsselwörter |
|---|---|
| Deklaration | `Dim`, `Public`, `Const` |
| Bedingungen | `If`, `Then`, `Else`, `ElseIf` |
| Schleifen | `For`, `Next`, `Each`, `To`, `Step`, `While`, `Do`, `Loop`, `Until`, `In` |
| Schleifensteuerung | `Exit`, `Continue` |
| Prozeduren | `Sub`, `Function`, `Return` |
| Fallunterscheidung | `Select`, `Case`, `Is` |
| Fehlerbehandlung | `Try`, `Catch`, `Finally` |
| Logik | `And`, `Or`, `Not`, `True`, `False`, `Nothing` |
| Sonstiges | `Print`, `End`, `Include` |

---

# Kontrollstrukturen – Übersicht

| Struktur    | Syntax-Skelett | Abschluss |
|-------------|-----------------|-----------|
| If          | `If bed Then` … `[ElseIf bed Then …]` `[Else …]` | `End If` |
| Select Case | `Select Case ausdruck` `Case wert / wert1, wert2 / x To y / Is > x` `[Case Else]` | `End Select` |
| For         | `For i = start To end [Step n]` | `Next [i]` |
| For Each    | `For Each [k,] v In array / 2D-array / map` | `Next [v]` |
| While       | `While bed` | `End While`             |
| Do Loop     | `Do [While/Until bed]` … `Loop [While/Until bed]` | `Loop` |
| Continue    | `Continue For` / `Continue While` / `Continue Do` | – |
| Try/Catch   | `Try` … `[Catch var …]` `[Finally …]` (mind. Catch oder Finally nötig) | `End Try` |
| Exit        | `Exit For` / `Exit While` / `Exit Do` / `Exit Sub` / `Exit Function` | – |
| Sub         | `Sub Name(param1, param2 [, Optional param3 = wert])` | `End Sub` |
| Function    | `Function Name(param1, param2 [, Optional param3 = wert])` … `Return wert` oder `Name = wert` | `End Function` |
| Cls         | `Cls()` | – |
| Print       | `Print wert` | – |

**For Each:** Mit einer Variable (`For Each v In …`) liefert ein 1D-Array die Elemente, ein 2D-Array die Zeilen (jeweils als 1D-Array), eine Map die Werte. Mit zwei Variablen (`For Each k, v In …`) liefern Arrays Index und Element, Maps Schlüssel und Wert. Maps werden nach Schlüssel sortiert durchlaufen.

**Select Case:** Es wird der erste passende `Case` ausgeführt. Überlappende Werte in mehreren `Case`-Zweigen sind erlaubt, der erste Treffer gewinnt.

**Continue:** `Continue For`, `Continue While` und `Continue Do` überspringen nur den Rest des aktuellen Schleifendurchlaufs und setzen die Schleife mit dem nächsten Durchlauf fort. Bei `Do … Loop While/Until` springt `Continue Do` zur Bedingung am Ende, die dann normal ausgewertet wird. Im Unterschied dazu beenden `Exit For`, `Exit While` und `Exit Do` die jeweilige Schleife vollständig.

---

# Beispiele

## While

```vbx
Dim i = 0

While i < 5
    i += 1
    Print "Durchlauf: " & i
End While
```

## Do Loop

```vbx
' Kopfgesteuert – läuft solange Bedingung true
Dim i = 0

Do While i < 3
    i += 1
    Print "While: " & i
Loop

' Fußgesteuert – läuft mindestens einmal
Dim j = 0

Do
    j += 1
    Print "Loop: " & j
Loop While j < 3

' Until – läuft bis Bedingung true wird
Dim k = 0

Do
    k += 1
    Print "Until: " & k
Loop Until k = 3
```

## Continue

```vbx
' Die 3 überspringen
For i = 1 To 5
    If i = 3 Then
        Continue For
    End If
    Print i            ' 1, 2, 4, 5
Next
```

## Select Case

```vbx
Dim x = 2

Select Case x
    Case 1, 2
        Print "A"      ' erster Treffer gewinnt
    Case 2, 3
        Print "B"
    Case Else
        Print "sonst"
End Select
```

Weitere Beispiele befinden sich in den Beispiel-VB-Dateien.