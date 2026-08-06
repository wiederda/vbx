# 🌐 Global – Kern- & Hilfsfunktionen

Enthält grundlegende Typ-Konvertierungen, String-Operationen, Hashing, Kryptografie, Prozesssteuerung und Systemabfragen.
Keine Namespace-Präfix – alle Funktionen sind direkt aufrufbar.

---

## ToInt(value)
- **Konkret:**
  Wandelt einen Wert in eine Ganzzahl um. Nachkommastellen werden abgeschnitten.
  Komma als Dezimaltrennzeichen wird akzeptiert.
- **Parameter:**
  - `value`: Zahl oder String.
- **Rückgabe:**
  `NumVal`

---

## ToFloat(value)
- **Konkret:**
  Wandelt einen Wert in eine Fließkommazahl um.
  Komma als Dezimaltrennzeichen wird akzeptiert.
- **Parameter:**
  - `value`: Zahl oder String.
- **Rückgabe:**
  `NumVal`

---

## ToBool(value)
- **Konkret:**
  Prüft den Wahrheitsgehalt eines Wertes.
- **Parameter:**
  - `value`: Beliebiger Wert.
- **Rückgabe:**
  `BoolVal`

---

## ToString(value)
- **Konkret:**
  Wandelt einen beliebigen Wert in seine String-Repräsentation um.
- **Parameter:**
  - `value`: Beliebiger Wert.
- **Rückgabe:**
  `StrVal`

---

## Chr(code)
- **Konkret:**
  Gibt das Zeichen zum angegebenen Unicode-Codepoint zurück.
- **Parameter:**
  - `code`: Unicode-Wert als Zahl (z. B. `65` → `"A"`).
- **Rückgabe:**
  `StrVal`

---

## Asc(char)
- **Konkret:**
  Gibt den Unicode-Wert des ersten Zeichens eines Strings zurück.
- **Parameter:**
  - `char`: String (es wird nur das erste Zeichen ausgewertet).
- **Rückgabe:**
  `NumVal`

---

## Length(s)
- **Konkret:**
  Gibt die Anzahl der Zeichen (Runen) zurück.
  Zählt Unicode-Zeichen korrekt, nicht Bytes.
- **Parameter:**
  - `s`: String.
- **Rückgabe:**
  `NumVal`

---

## TrimStart(s, cutset)
- **Konkret:**
  Entfernt alle Zeichen aus `cutset` am Anfang des Strings.
- **Parameter:**
  - `s`: String.
  - `cutset`: Zu entfernende Zeichen.
- **Rückgabe:**
  `StrVal`

---

## TrimEnd(s, cutset)
- **Konkret:**
  Entfernt alle Zeichen aus `cutset` am Ende des Strings.
- **Parameter:**
  - `s`: String.
  - `cutset`: Zu entfernende Zeichen.
- **Rückgabe:**
  `StrVal`

---

## Trim(s, [cutset])
- **Konkret:**
  Entfernt Leerzeichen (oder angegebene Zeichen) an beiden Enden.
- **Parameter:**
  - `s`: String.
  - `cutset`: Optional. Zu entfernende Zeichen. Standard: Whitespace.
- **Rückgabe:**
  `StrVal`

---

## ToLower(s)
- **Konkret:**
  Wandelt einen String vollständig in Kleinschreibung um.
- **Parameter:**
  - `s`: String.
- **Rückgabe:**
  `StrVal`

---

## ToUpper(s)
- **Konkret:**
  Wandelt einen String vollständig in Großschreibung um.
- **Parameter:**
  - `s`: String.
- **Rückgabe:**
  `StrVal`

---

## Left(s, n)
- **Konkret:**
  Extrahiert die ersten `n` Zeichen eines Strings (Runen-basiert).
- **Parameter:**
  - `s`: String.
  - `n`: Anzahl der Zeichen.
- **Rückgabe:**
  `StrVal`

---

## Right(s, n)
- **Konkret:**
  Extrahiert die letzten `n` Zeichen eines Strings (Runen-basiert).
- **Parameter:**
  - `s`: String.
  - `n`: Anzahl der Zeichen.
- **Rückgabe:**
  `StrVal`

---

## Substring(s, start, [len])
- **Konkret:**
  Gibt einen Teilstring zurück (0-basierter Index, Runen-basiert).
- **Parameter:**
  - `s`: String.
  - `start`: Startposition (0-basiert).
  - `len`: Optional. Anzahl der Zeichen. Standard: bis Stringende.
- **Rückgabe:**
  `StrVal`

---

## Contains(s, sub)
- **Konkret:**
  Prüft, ob ein Teilstring im Text enthalten ist.
  Suche ist case-insensitiv.
- **Parameter:**
  - `s`: String.
  - `sub`: Gesuchter Teilstring.
- **Rückgabe:**
  `BoolVal`

---

## IndexOf(s, sub)
- **Konkret:**
  Gibt den Index des ersten Vorkommens von `sub` zurück.
- **Parameter:**
  - `s`: String.
  - `sub`: Gesuchter Teilstring.
- **Rückgabe:**
  `NumVal`
  `-1` wenn nicht gefunden.

---

## LastIndexOf(s, sub)
- **Konkret:**
  Gibt den Index des letzten Vorkommens von `sub` zurück.
- **Parameter:**
  - `s`: String.
  - `sub`: Gesuchter Teilstring.
- **Rückgabe:**
  `NumVal`
  `-1` wenn nicht gefunden.

---

## Replace(s, find, repl)
- **Konkret:**
  Ersetzt alle Vorkommen von `find` durch `repl` im String.
- **Parameter:**
  - `s`: Quellstring.
  - `find`: Zu ersetzender Text.
  - `repl`: Ersatztext.
- **Rückgabe:**
  `StrVal`

---

## ReplaceVars(text, key1, val1, key2, val2, ...)
- **Konkret:**
  Ersetzt mehrere `{Platzhalter}` in einem Text in einem Aufruf.
  Platzhalter werden im Format `{key}` erwartet.
- **Parameter:**
  - `text`: Quelltext mit Platzhaltern.
  - `key, val`: Beliebig viele Schlüssel/Wert-Paare. Ungerade Anzahl ignoriert letzten Schlüssel.
- **Rückgabe:**
  `StrVal`

---

## Split(s, sep, [unique])
- **Konkret:**
  Zerlegt einen String anhand eines Trennzeichens in ein Array.
  Leere Segmente werden übersprungen. Teile werden getrimmt.
  Bei `sep = " "` wird `strings.Fields` genutzt (mehrere Leerzeichen werden zusammengefasst).
- **Parameter:**
  - `s`: Quellstring.
  - `sep`: Trennzeichen.
  - `unique`: Optional. `BoolVal` – bei `true` werden Duplikate entfernt.
- **Rückgabe:**
  `ArrVal`

---

## EncodeBase64(s)
- **Konkret:**
  Kodiert einen String als Base64.
- **Parameter:**
  - `s`: Quellstring.
- **Rückgabe:**
  `StrVal`

---

## DecodeBase64(s)
- **Konkret:**
  Dekodiert einen Base64-String zurück in Klartext.
- **Parameter:**
  - `s`: Base64-kodierter String.
- **Rückgabe:**
  `StrVal`
  Bei Fehler: `StrVal` mit Fehlerpräfix `"error: ..."`.

---

## URLEncode(val)
- **Konkret:**
  Kodiert einen String für die sichere Verwendung in URLs (Query-Escape).
- **Parameter:**
  - `val`: Zu kodierender String.
- **Rückgabe:**
  `StrVal`
  Beispiel: `"Hallo Welt!"` → `"Hallo+Welt%21"`.

---

## URLDecode(urlStr)
- **Konkret:**
  Dekodiert einen URL-kodierten String zurück in Klartext.
- **Parameter:**
  - `urlStr`: URL-kodierter String.
- **Rückgabe:**
  `StrVal`

---

## PunyEncode(domain)
- **Konkret:**
  Kodiert eine Unicode-Domain in Punycode (IDNA2008/UTS#46).
- **Parameter:**
  - `domain`: Domain als Unicode-String (z. B. `"münchen.de"`).
- **Rückgabe:**
  `ArrVal`
  Format: `[OK, EncodedDomain, Msg]`

---

## PunyDecode(domain)
- **Konkret:**
  Dekodiert eine Punycode-Domain zurück in Unicode.
- **Parameter:**
  - `domain`: Punycode-Domain (z. B. `"xn--mnchen-3ya.de"`).
- **Rückgabe:**
  `ArrVal`
  Format: `[OK, DecodedDomain, Msg]`

---

## MD5(s)
- **Konkret:**
  Erzeugt einen MD5-Hash des übergebenen Textes.
- **Parameter:**
  - `s`: Quellstring.
- **Rückgabe:**
  `StrVal`
  Hex-kodierter Hash.

---

## SHA1(s)
- **Konkret:**
  Erzeugt einen SHA1-Hash (für Legacy-Kompatibilität).
- **Parameter:**
  - `s`: Quellstring.
- **Rückgabe:**
  `StrVal`
  Hex-kodierter Hash.

---

## SHA256(s)
- **Konkret:**
  Erzeugt einen SHA256-Hash des Textes.
- **Parameter:**
  - `s`: Quellstring.
- **Rückgabe:**
  `StrVal`
  Hex-kodierter Hash.

---

## SHA512(s)
- **Konkret:**
  Erzeugt einen SHA512-Hash des Textes.
- **Parameter:**
  - `s`: Quellstring.
- **Rückgabe:**
  `StrVal`
  Hex-kodierter Hash.

---

## Int([max])
- **Konkret:**
  Gibt eine zufällige Ganzzahl zurück.
  Ohne Parameter: beliebige Ganzzahl. Mit `max`: Wert von `0` bis `max-1`.
- **Parameter:**
  - `max`: Optional. Obere Grenze (exklusiv).
- **Rückgabe:**
  `NumVal`

---

## Input([prompt])
- **Konkret:**
  Liest eine Benutzereingabe von der Konsole.
  Zahlen werden automatisch als `NumVal` zurückgegeben.
- **Parameter:**
  - `prompt`: Optional. Anzuzeigender Text vor der Eingabe.
- **Rückgabe:**
  `NumVal` oder `StrVal` (automatische Erkennung).

---

## Alert(text)
- **Konkret:**
  Gibt Text aus und wartet auf Bestätigung via ENTER.
- **Parameter:**
  - `text`: Anzuzeigender Text.
- **Rückgabe:**
  `BoolVal` (`true`)

---

## Confirm(text)
- **Konkret:**
  Zeigt Text an und wartet auf eine J/N-Eingabe.
  Akzeptiert: `j`, `ja`, `y`, `yes`.
- **Parameter:**
  - `text`: Anzuzeigender Text.
- **Rückgabe:**
  `BoolVal`

---

## Sleep(wert, [einheit], [showOutput])
- **Konkret:**
  Pausiert die Ausführung. Bei Sekunden/Minuten/Stunden wird ein Countdown angezeigt, der per ENTER übersprungen werden kann.
- **Parameter:**
  - `wert`: Zeitwert.
  - `einheit`: Optional. `"ms"`, `"s"`, `"m"`, `"h"` (Standard: `"ms"`).
  - `showOutput`: Optional. `BoolVal` – steuert Countdown-Anzeige (Standard: `true`).
- **Rückgabe:**
  `NumVal` (übergebener Zeitwert)

---

## Wait(pid)
- **Konkret:**
  Pausiert das Skript, bis der Prozess mit der angegebenen PID beendet ist.
  Pollt in 250-ms-Intervallen.
- **Parameter:**
  - `pid`: Prozess-ID als Zahl.
- **Rückgabe:**
  `NumVal`
  `1` bei Erfolg, `-1` wenn Prozess nicht gefunden.

---

## Format(expression, style)
- **Konkret:**
  Universelle Formatierung für Zahlen, Datumsangaben und Text.
- **Parameter:**
  - `expression`: Zahl oder Datumsstring.
  - `style`: Formatstring.
- **Stile (Zahlen):**
  - `"0.00"` – Festkomma
  - `"Currency"` – Währung mit €
  - `"Percent"` – Prozent
  - `"Hex"` – Hexadezimal
- **Stile (Datum):**
  - `"YYYY-MM-DD"` – ISO
  - `"DD.MM.YYYY"` – Deutsch
  - `"ddd"` – Wochentag (Mo, Di …)
  - `"dddd"` – Wochentag ausgeschrieben (Montag, Dienstag …)
  - `"MMM"` – Monatsname (Jan, Feb …)
  - `"MMMM"` – Monatsname ausgeschrieben (Januar, Februar …)
  - `"HH"` – Stunde (24h, zweistellig)
  - `"mm"` – Minute (zweistellig, **kleingeschrieben**)
  - `"ss"` / `"SS"` – Sekunde (zweistellig, beide Schreibweisen möglich)
  - **Wichtig:** `MM` (großgeschrieben) steht immer für den Monat, auch mehrfach im selben Formatstring. Für Zeitangaben `HH:mm:ss` verwenden, nicht `HH:MM:SS` – letzteres würde zweimal den Monat einsetzen statt Minute und Sekunde.
- **Beispiel:**
```vbx
  Print Format(file.AccessTime(path), "YYYY-MM-DD HH:mm:ss")
  ' 2026-08-04 12:42:25
```
- **Rückgabe:**
  `StrVal`

---

## FormatSize(bytes)
- **Konkret:**
  Wandelt Bytes in eine lesbare Größenangabe um (z. B. `"4.2 MB"`).
- **Parameter:**
  - `bytes`: Dateigröße als Zahl.
- **Rückgabe:**
  `StrVal`

---

## Default(val, default)
- **Konkret:**
  Gibt `default` zurück, wenn `val` `null` oder ein leerer String ist.
- **Parameter:**
  - `val`: Zu prüfender Wert.
  - `default`: Fallback-Wert.
- **Rückgabe:**
  Ursprünglicher oder Fallback-Wert.

---

## ConvNewLine(content, [target])
- **Konkret:**
  Normalisiert Zeilenumbrüche in einem String.
  Intern werden zunächst alle Varianten auf `\n` vereinheitlicht.
- **Parameter:**
  - `content`: String mit gemischten Zeilenumbrüchen.
  - `target`: Optional. `"lf"`, `"crlf"`, `"cr"`, oder `"auto"` (OS-abhängig, Standard).
- **Rückgabe:**
  `StrVal`

---

## IsNumeric(v)
- **Konkret:**
  Prüft, ob ein Wert eine Zahl ist oder als Zahl interpretiert werden kann.
- **Parameter:**
  - `v`: Zu prüfender Wert.
- **Rückgabe:**
  `BoolVal`

---

## IsInteger(v)
- **Konkret:**
  Prüft, ob ein Wert eine Ganzzahl ohne Nachkommastellen ist.
- **Parameter:**
  - `v`: Zu prüfender Wert.
- **Rückgabe:**
  `BoolVal`

---

## IsPositive(n)
- **Konkret:**
  Prüft, ob `n > 0`.
- **Parameter:**
  - `n`: Zahl.
- **Rückgabe:**
  `BoolVal`

---

## IsNothing(b)
- **Konkret:**
  Prüft, ob ein Wert nicht initialisiert ist (`Undefined`, `Null`, `Nil`, `None`).
- **Parameter:**
  - `b`: Zu prüfender Wert.
- **Rückgabe:**
  `BoolVal`

---

## IsPrime(n)
- **Konkret:**
  Prüft, ob `n` eine Primzahl ist.
  Optimierter Algorithmus mit 6k±1-Schritten bis zur Quadratwurzel.
- **Parameter:**
  - `n`: Ganzzahl.
- **Rückgabe:**
  `BoolVal`

---

## Uptime()
- **Konkret:**
  Gibt die System-Laufzeit in Sekunden zurück.
- **Rückgabe:**
  `NumVal`

---

## UptimeString()
- **Konkret:**
  Gibt die System-Uptime als lesbaren Text zurück (z. B. `"2h 15m"`).
- **Rückgabe:**
  `StrVal`

---

## SystemMemory()
- **Konkret:**
  Gibt Arbeitsspeicher-Informationen des Systems zurück.
- **Rückgabe:**
  `ArrVal`
  Format: `[Total, Available, UsedPercent]`
  Bytes für Total/Available, Prozent für UsedPercent.

---

## UserName()
- **Konkret:**
  Gibt den aktuellen Benutzernamen zurück.
- **Rückgabe:**
  `StrVal`

---

## ComputerName()
- **Konkret:**
  Gibt den Hostnamen des Rechners zurück.
- **Rückgabe:**
  `StrVal`

---

## UserDomain()
- **Konkret:**
  Gibt die Windows-Domäne zurück. Fallback auf Hostnamen (Linux/macOS).
- **Rückgabe:**
  `StrVal`

---

## Arch()
- **Konkret:**
  Gibt die Prozessor-Architektur zurück.
- **Rückgabe:**
  `StrVal`
  Beispiele: `"amd64"`, `"arm64"`.

---

## OS()
- **Konkret:**
  Gibt das Betriebssystem zurück.
- **Rückgabe:**
  `StrVal`
  Beispiele: `"windows"`, `"linux"`, `"darwin"`.

---

## ToClipboard(text)
- **Konkret:**
  Kopiert Text in die Zwischenablage.
  Windows: `clip`. Linux: `xclip`.
- **Parameter:**
  - `text`: Zu kopierender Text.
- **Rückgabe:**
  `BoolVal`
  `true` bei Erfolg.

---

## Unblock(path)
- **Konkret:**
  Hebt die NTFS-Sicherheitsblockierung (`Zone.Identifier` ADS) von Dateien auf.
  Auf Nicht-Windows-Systemen ist die Funktion ein No-op und gibt `0` zurück.
- **Parameter:**
  - `path`: Datei oder Verzeichnis.
- **Rückgabe:**
  `NumVal`
  Anzahl der entsperrten Dateien.

---

## GC()
- **Konkret:**
  Erzwingt den Go-Garbage-Collector, um ungenutzten Speicher sofort freizugeben.
- **Rückgabe:**
  `NullVal`

---

## Worker(path)
- **Konkret:**
  Startet ein `.vb`- oder `.vbc`-Skript als Hintergrundprozess.
  Stdout und Stderr werden in `<skriptpfad>.log` umgeleitet.
  Nur `.vb`- und `.vbc`-Dateien sind erlaubt.
- **Parameter:**
  - `path`: Pfad zum Skript.
- **Rückgabe:**
  `StrVal`
  PID des gestarteten Prozesses.

---

## Build(quelle)
- **Konkret:**
  Verschlüsselt ein `.vb`-Skript mit Magic-Header (`VBC!`) und speichert es als `.vbc`-Datei.
  Löst `#include`-Direktiven rekursiv auf. Erkennt zirkuläre Abhängigkeiten.
- **Parameter:**
  - `quelle`: Pfad zur `.vb`-Quelldatei.
- **Rückgabe:**
  `StrVal`
  Pfad der erzeugten `.vbc`-Datei.

---

## IsCompiled(path)
- **Konkret:**
  Prüft, ob eine Datei ein gültiges verschlüsseltes `.vbc`-Skript mit Magic-Header ist.
- **Parameter:**
  - `path`: Dateipfad.
- **Rückgabe:**
  `BoolVal`

---

## GenerateSSHKey([outDir, algo, bits, pass])
- **Konkret:**
  Erstellt ein SSH-Schlüsselpaar und schreibt es auf die Festplatte.
  Private Key mit Rechten `0600`, Public Key mit `0644`.
  Bricht ab, wenn die Zieldateien bereits existieren.
- **Parameter:**
  - `outDir`: Optional. Zielverzeichnis (Standard: `~/.ssh`).
  - `algo`: Optional. `"rsa"` (Standard) oder `"ed25519"`.
  - `bits`: Optional. Schlüssellänge für RSA (Standard: 4096, Minimum wird erzwungen).
  - `pass`: Optional. Passphrase (aktuell nicht verwendet).
- **Rückgabe:**
  `StrVal`
  Basispfad des erstellten Schlüsselpaares (ohne `.pub`).

---

## Password([prompt])
- **Konkret:**
  Liest eine Eingabe von der Konsole ohne sie anzuzeigen (Echo unterdrückt).
  Falls stdin kein echtes Terminal ist (z. B. Pipe), wird auf normales Zeilenlesen zurückgefallen.
- **Parameter:**
  - `prompt`: Optional. Text der vor der Eingabe angezeigt wird.
- **Rückgabe:**
  `StrVal`, `ErrorVal` bei Lesefehler.

---

## IsArray(val)
- **Konkret:**
  Prüft ob ein Wert ein Array ist (`KindArr` oder `KindArr2D`).
- **Parameter:**
  - `val`: Zu prüfender Wert.
- **Rückgabe:**
  `BoolVal`

---

## IsNull(val)
- **Konkret:**
  Prüft ob ein Wert `Null`, `Nothing` oder nicht initialisiert ist (`KindNull`, `KindNil`, `KindNone`).
- **Parameter:**
  - `val`: Zu prüfender Wert.
- **Rückgabe:**
  `BoolVal`

---

## PrintFormat()
- **Konkret:**
  Gibt eine Übersicht aller verfügbaren `Format()`-Optionen auf der Konsole aus.
- **Rückgabe:**
  `NullVal`

---

## IsDate(val)
- **Konkret:**
  Prüft ob ein String als Datum parsebar ist.
  Unterstützte Formate identisch zu `date.*`: ISO, Deutsch, US, RFC3339.
- **Parameter:**
  - `val`: Zu prüfender Wert (nur `StrVal` gibt `true`).
- **Rückgabe:**
  `BoolVal`

---

## IsString(val)
- **Konkret:**
  Prüft ob der Wert ein String ist (`KindStr`).
- **Parameter:**
  - `val`: Zu prüfender Wert.
- **Rückgabe:**
  `BoolVal`

---

## IsMap(val)
- **Konkret:**
  Prüft ob der Wert eine Map ist (`KindMap`).
- **Parameter:**
  - `val`: Zu prüfender Wert.
- **Rückgabe:**
  `BoolVal`