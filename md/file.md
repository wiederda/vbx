# 📁 file.* – Datei- & Pfadfunktionen

Dient zum Lesen, Schreiben, Suchen, Überwachen und Verwalten von Dateien und Pfaden.
Plattformübergreifend (Windows, Linux, macOS). Schreiboperationen nutzen `absPathVal` zur Pfadabsicherung.

---

## file.WriteAllText(path, content)
- **Konkret:**
  Schreibt den gesamten Text in eine Datei. Vorhandene Inhalte werden überschrieben.
- **Parameter:**
  - `path`: Zieldatei.
  - `content`: Zu schreibender Text.
- **Rückgabe:**
  `NullVal`

---

## file.Create(path)
- **Konkret:**
  Erstellt eine leere Datei, falls diese noch nicht existiert.
  Das Zielverzeichnis muss bereits vorhanden sein.
- **Parameter:**
  - `path`: Zielpfad der Datei.
- **Rückgabe:**
  `NullVal`

---

## file.StreamWrite(path, content)
- **Konkret:**
  Hängt Text gepuffert an eine Datei an.
  Erstellt die Datei, falls sie noch nicht existiert.
  Das Zielverzeichnis muss vorhanden sein.
- **Parameter:**
  - `path`: Zieldatei.
  - `content`: Anzuhängender Text.
- **Rückgabe:**
  `NullVal`

---

## file.AppendAllText(path, text)
- **Konkret:**
  Hängt Text ungepuffert an eine Datei an.
  Erstellt die Datei, falls sie noch nicht existiert.
  Das Zielverzeichnis muss vorhanden sein.
- **Parameter:**
  - `path`: Zieldatei.
  - `text`: Anzuhängender Text.
- **Rückgabe:**
  `NullVal`

---

## file.AppendLine(path, line)
- **Konkret:**
  Hängt eine Zeile an eine Datei an. Fügt automatisch einen Zeilenumbruch (`\n`) ein.
  Erstellt die Datei, falls sie noch nicht existiert. Das Zielverzeichnis muss vorhanden sein.
- **Parameter:**
  - `path`: Zieldatei.
  - `line`: Anzuhängende Zeile (ohne eigenen Zeilenumbruch).
- **Rückgabe:**
  `BoolVal` (`true`) bei Erfolg, `ErrorVal` bei Fehler.

---

## file.Exists(path)
- **Konkret:**
  Prüft, ob eine Datei unter dem angegebenen Pfad existiert.
  Verzeichnisse zählen nicht als Datei.
- **Parameter:**
  - `path`: Zu prüfender Pfad.
- **Rückgabe:**
  `BoolVal`

---

## file.HasContent(path)
- **Konkret:**
  Prüft, ob eine Datei existiert und mehr als 0 Byte Inhalt hat.
- **Parameter:**
  - `path`: Zu prüfender Pfad.
- **Rückgabe:**
  `BoolVal`
  `true` = Datei existiert und ist größer als 0 Byte. `false` = Datei existiert nicht oder ist leer.

--- 

## file.Delete(path)
- **Konkret:**
  Löscht eine Datei. Existiert die Datei nicht, wird kein Fehler zurückgegeben.
- **Parameter:**
  - `path`: Zu löschende Datei.
- **Rückgabe:**
  `ArrVal`
  Format: `[OK, Msg]`. Bei Erfolg ist `Msg` immer `Null` (auch wenn intern eine leere Meldung übergeben wird).

---

## file.Copy(src, dst)
- **Konkret:**
  Kopiert eine Datei. Ziel darf nicht bereits existieren.
  Das Zielverzeichnis muss vorhanden sein.
- **Parameter:**
  - `src`: Quelldatei.
  - `dst`: Zieldatei.
- **Rückgabe:**
  `ArrVal`
  Format: `[OK, Msg]`

---

## file.Move(src, dst)
- **Konkret:**
  Verschiebt oder benennt eine Datei um.
  Versucht zuerst ein atomares Rename; bei Cross-Drive-Operationen wird auf Copy+Delete zurückgegriffen.
  Ziel darf nicht bereits existieren.
- **Parameter:**
  - `src`: Quelldatei.
  - `dst`: Zieldatei.
- **Rückgabe:**
  `ArrVal`
  Format: `[OK, Msg]`

---

## file.Rename(oldPath, newPath)
- **Konkret:**
  Benennt eine Datei um (direktes `os.Rename`).
  Kann auch zum Verschieben innerhalb desselben Dateisystems genutzt werden.
- **Parameter:**
  - `oldPath`: Aktueller Pfad.
  - `newPath`: Neuer Pfad.
- **Rückgabe:**
  `NullVal`
- **Hinweis:** Anders als `file.Move` gibt es hier **keinen** Cross-Drive-Fallback und **keine** Prüfung, ob `newPath` bereits existiert – Verhalten bei existierendem Ziel ist plattformabhängig.

---

## file.Compare(path1, path2)
- **Konkret:**
  Vergleicht zwei Dateien byteweise auf Gleichheit.
  Schneller Vorab-Check via Dateigröße; gleiche Pfade gelten direkt als identisch.
- **Parameter:**
  - `path1`: Erste Datei.
  - `path2`: Zweite Datei.
- **Rückgabe:**
  `NumVal`
  `0` = identisch, `1` = unterschiedlich.

---

## file.Size(path, [unit])
- **Konkret:**
  Gibt die Größe einer Datei zurück.
- **Parameter:**
  - `path`: Zieldatei.
  - `unit`: Optional. Bei `"human"` wird die Größe als lesbarer String zurückgegeben (z. B. `"4.2 MB"`).
- **Rückgabe:**
  `NumVal` (Bytes) oder `StrVal` bei `"human"`.

---

## file.Hash(path, [algo])
- **Konkret:**
  Berechnet den kryptografischen Hash einer Datei.
  Streamt die Datei direkt in den Hash-Generator (speichereffizient).
- **Parameter:**
  - `path`: Zieldatei.
  - `algo`: Algorithmus. Unterstützt `"md5"` (Standard), `"sha1"`, `"sha224"`, `"sha256"`, `"sha384"`, `"sha512"`.
- **Rückgabe:**
  `StrVal`
  Hex-kodierter Hash-String.

---

## file.HashBatch(paths, [algo, workers])
- **Konkret:**
  Berechnet Hashes mehrerer Dateien parallel über einen Worker-Pool (Standard: 8 Worker).
  Nutzt intern dieselbe Hash-Logik wie `file.Hash`.
- **Parameter:**
  - `paths`: `ArrVal` mit Dateipfaden (`StrVal`).
  - `algo`: Optional. Algorithmus, identisch zu `file.Hash` (Standard: `"md5"`).
  - `workers`: Optional. Anzahl paralleler Worker (Standard: 8).
- **Rückgabe:**
  `MapVal` – Pfad (Schlüssel) auf Hash (`StrVal`) oder `ErrorVal` bei Lesefehler der jeweiligen Datei.
  
---

## file.HashVerify(hash, algo)
- **Konkret:**
  Prüft, ob ein String rein formal ein gültiger Hash-Wert des angegebenen Algorithmus ist (korrekte Länge, nur Hex-Zeichen).
  Greift nicht auf eine Datei zu – reine String-Prüfung. Nützlich um Ergebnisse aus `file.Hash`/`file.HashBatch` auf Plausibilität zu prüfen (z. B. um `ErrorVal`-Werte oder abgeschnittene/korrupte Hashes zu erkennen), bevor sie weiterverarbeitet oder in eine Datenbank geschrieben werden.
- **Parameter:**
  - `hash`: Zu prüfender String.
  - `algo`: Algorithmus. Unterstützt `"md5"`, `"sha1"`, `"sha224"`, `"sha256"`, `"sha384"`, `"sha512"`.
- **Rückgabe:**
  `BoolVal`. `ErrorVal` bei unbekanntem Algorithmus.
- **Beispiel:**
```vbx
  Dim h = file.Hash("test.mp4", "sha256")

  If file.HashVerify(h, "sha256") = False Then
      Print "Fehler beim Hashen: " & h
  Else
      Print "Hash gültig: " & h
  End If
```

---

## file.VerifyHash(path, expectedHash, [algo])
- **Konkret:**
  Berechnet den Hash einer Datei und vergleicht ihn mit einem erwarteten Wert (case-insensitiv).
  Nützlich zur Integritätsprüfung, z. B. nach einem Download oder Backup.
- **Parameter:**
  - `path`: Zieldatei.
  - `expectedHash`: Erwarteter Hash-Wert als String.
  - `algo`: Optional. Algorithmus (Standard: `"sha256"`).
- **Rückgabe:**
  `BoolVal` (`true` bei Übereinstimmung), `ErrorVal` bei Lesefehler.
- **Beispiel:**
```vbx
  If file.VerifyHash("download.zip", erwarteterHash, "sha256") Then
      Print "Datei OK"
  Else
      Print "Datei beschädigt oder manipuliert"
  End If
```

---

## file.GitBlobHash(path)
- **Konkret:**
  Berechnet den Git-Blob-SHA1 einer Datei – kompatibel zu `git hash-object`.
  Es wird der git-übliche `"blob <Länge>\0"`-Header vor den Dateiinhalt gehasht.
- **Parameter:**
  - `path`: Zieldatei.
- **Rückgabe:**
  `StrVal`
  Hex-kodierter SHA1-String, `ErrorVal` bei Lesefehler.

---

## file.ModTime(path)
- **Konkret:**
  Gibt den Zeitpunkt der letzten Änderung einer Datei zurück.
- **Parameter:**
  - `path`: Zieldatei.
- **Rückgabe:**
  `StrVal`
  Format: ISO 8601 / RFC3339 (z. B. `"2024-03-15T14:32:00+01:00"`).

---

## file.CreateTime(path)
- **Konkret:**
  Gibt den Erstellungszeitpunkt einer Datei zurück.
- **Parameter:**
  - `path`: Zieldatei.
- **Rückgabe:**
  `StrVal`
  Format: ISO 8601 / RFC3339.

---

## file.AccessTime(path)
- **Konkret:**
  Gibt den Zeitpunkt des letzten Zugriffs auf eine Datei zurück.
- **Parameter:**
  - `path`: Zieldatei.
- **Rückgabe:**
  `StrVal`
  Format: ISO 8601 / RFC3339.

---

## file.ReadAllText(path)
- **Konkret:**
  Liest den gesamten Inhalt einer Datei als einen String.
  Nutzt Shared-Access (Windows-kompatibel).
- **Parameter:**
  - `path`: Quelldatei.
- **Rückgabe:**
  `StrVal`

---

## file.ReadAllLines(path)
- **Konkret:**
  Liest eine Textdatei zeilenweise ein.
  Normalisiert Zeilenumbrüche (`\r\n` → `\n`), entfernt abschließende Leerzeilen.
  Nutzt Shared-Access (Windows-kompatibel).
- **Parameter:**
  - `path`: Quelldatei.
- **Rückgabe:**
  `ArrVal`
  Array von `StrVal`-Einträgen, ein Eintrag pro Zeile.

---

## file.LineCount(path)
- **Konkret:**
  Gibt die Anzahl der Zeilen einer Textdatei zurück.
  Nutzt Shared-Access (Windows-kompatibel), unterstützt Zeilen bis 1 MB Länge.
- **Parameter:**
  - `path`: Quelldatei.
- **Rückgabe:**
  `NumVal`, `ErrorVal` bei Lesefehler.

---

## file.Head(path, n)
- **Konkret:**
  Gibt die ersten `n` Zeilen einer Datei zurück.
  Nutzt Shared-Access (Windows-kompatibel), unterstützt Zeilen bis 1 MB Länge.
- **Parameter:**
  - `path`: Quelldatei.
  - `n`: Anzahl Zeilen (Max: 5000, analog zu `file.Tail`).
- **Rückgabe:**
  `ArrVal`, `ErrorVal` bei Lesefehler.

---

## file.ReadBytes(path)
- **Konkret:**
  Liest eine Datei als Byte-Array ein.
- **Parameter:**
  - `path`: Quelldatei.
- **Rückgabe:**
  `ArrVal`
  Array von `NumVal`-Einträgen (Werte 0–255).

---

## file.WriteBytes(path, byteArray)
- **Konkret:**
  Schreibt ein Byte-Array in eine Datei.
  Das Zielverzeichnis muss vorhanden sein.
- **Parameter:**
  - `path`: Zieldatei.
  - `byteArray`: `ArrVal` mit `NumVal`-Einträgen (0–255).
- **Rückgabe:**
  `NullVal`

---

## file.Replace(path, pattern, newContent)
- **Konkret:**
  Sucht zeilenweise nach dem ersten Vorkommen von `pattern` und ersetzt die gesamte Zeile durch `newContent`.
  Schreibt nur bei tatsächlichem Fund.
- **Parameter:**
  - `path`: Zieldatei.
  - `pattern`: Suchmuster (Substring).
  - `newContent`: Ersetzungszeile.
- **Rückgabe:**
  `NullVal`

---

## file.ReplaceAll(path, old, new)
- **Konkret:**
  Ersetzt alle Vorkommen von `old` durch `new` im gesamten Dateiinhalt (nicht zeilenweise).
- **Parameter:**
  - `path`: Zieldatei.
  - `old`: Zu ersetzender Text.
  - `new`: Ersatztext.
- **Rückgabe:**
  `NullVal`

---

## file.Search(path, pattern)
- **Konkret:**
  Durchsucht eine Datei zeilenweise nach einem Muster.
  Suche ist case-insensitiv.
  Unterstützt Zeilen bis 1 MB Länge (z. B. große Log-Dateien).
  Nutzt Shared-Access (Windows-kompatibel).
- **Parameter:**
  - `path`: Quelldatei.
  - `pattern`: Suchmuster.
- **Rückgabe:**
  `ArrVal`
  Array aller Zeilen, die das Muster enthalten.

---

## file.SearchDateRange(path, start, end, [layout], [max])
- **Konkret:**
  Liest Zeilen aus einer Datei, deren Zeitstempel (am Zeilenanfang) in einem angegebenen Zeitraum liegen.
  Folgezeilen ohne eigenen Zeitstempel werden dem letzten erkannten Datum zugeordnet.
- **Parameter:**
  - `path`: Quelldatei.
  - `start`: Startdatum als String.
  - `end`: Enddatum als String.
  - `layout`: Datumsformat (Standard: `"YYYY-MM-DD HH:mm:SS"`).
  - `max`: Maximale Trefferanzahl (Standard: 5000).
- **Rückgabe:**
  `ArrVal`
  Array aller Zeilen im Zeitraum.

---

## file.ReadValue(path, prefix)
- **Konkret:**
  Sucht die erste Zeile, die mit `prefix` beginnt, und gibt den Wert dahinter zurück (getrimmt).
  Geeignet für einfache Key-Value-Konfigurationsdateien.
- **Parameter:**
  - `path`: Quelldatei.
  - `prefix`: Präfix (z. B. `"Port="`).
- **Rückgabe:**
  `StrVal` – Wert nach dem Präfix, oder leerer String wenn nicht gefunden. `ErrorVal` bei echtem Lesefehler (z. B. Datei fehlt).
- **Fix:** Der Pfad läuft jetzt über `absPathVal` (vorher ungeprüft) – gleiche Absicherung wie bei den übrigen `file.*`-Funktionen. Außerdem war "nicht gefunden" vorher nicht von "Datei nicht lesbar" unterscheidbar (beides `""`); jetzt liefert ein Lesefehler `ErrorVal`.

---

## file.UpdateValue(path, search, newValue)
- **Konkret:**
  Sucht die erste Zeile, die mit `search` beginnt (Prefix-Match, getrimmt), und ersetzt sie vollständig durch `newValue`.
  Erhält den originalen Zeilenumbruch-Stil (`\r\n` oder `\n`).
- **Parameter:**
  - `path`: Zieldatei.
  - `search`: Zeilenpräfix zum Auffinden der Zeile.
  - `newValue`: Neue vollständige Zeile.
- **Rückgabe:**
  `BoolVal`
  `true` = Zeile gefunden und ersetzt, `false` = nicht gefunden.
- **Fix:** Der Pfad läuft jetzt über `absPathVal` (vorher ungeprüft).

---

## file.UniqueLines(path, caseSensitive)
- **Konkret:**
  Entfernt doppelte Zeilen aus einer Datei und schreibt das Ergebnis zurück.
  Leere Zeilen werden ignoriert.
- **Parameter:**
  - `path`: Zieldatei.
  - `caseSensitive`: `BoolVal` – ob Groß-/Kleinschreibung berücksichtigt wird.
- **Rückgabe:**
  `BoolVal`
  `true` bei Erfolg.

---

## file.GetDuplicates(path, caseSensitive)
- **Konkret:**
  Findet alle Zeilen, die mehr als einmal in der Datei vorkommen.
  Leere Zeilen werden ignoriert.
- **Parameter:**
  - `path`: Quelldatei.
  - `caseSensitive`: `BoolVal` – ob Groß-/Kleinschreibung berücksichtigt wird.
- **Rückgabe:**
  `ArrVal`
  Array der duplizierten Zeilen (als `StrVal`), **alphabetisch sortiert**.

---

## file.Tail(path, [lines], [refresh])
- **Konkret:**
  Gibt die letzten `n` Zeilen einer Datei zurück.
  Bei Angabe eines Refresh-Intervalls wechselt die Funktion in einen blockierenden Live-Modus und gibt neue Zeilen kontinuierlich auf stdout aus.
- **Parameter:**
  - `path`: Quelldatei.
  - `lines`: Anzahl der Zeilen (Standard: 10, Max: 5000).
  - `refresh`: Aktualisierungsintervall (z. B. `"1s"`, `"500ms"` oder Zahl in ms).
- **Rückgabe:**
  `StrVal` (ohne Refresh) oder `NullVal` (mit Refresh, blockierend – kehrt nur zurück, wenn die Datei währenddessen verschwindet).

---

## file.Watch(path, [timeoutMs])
- **Konkret:**
  Wartet blockierend, bis sich eine Datei ändert (ModTime oder Größe).
  Pollt in 100-ms-Intervallen.
- **Parameter:**
  - `path`: Zu überwachende Datei.
  - `timeoutMs`: Maximale Wartezeit in Millisekunden (optional, 0 = unbegrenzt).
- **Rückgabe:**
  `BoolVal`
  `true` = Datei hat sich geändert, `false` = Timeout abgelaufen.

---

## file.WatchLog(path, pattern, [style])
- **Konkret:**
  Überwacht eine Logdatei live und hebt Zeilen, die `pattern` enthalten, farbig hervor.
  Blockierend.
- **Parameter:**
  - `path`: Zu überwachende Logdatei.
  - `pattern`: Suchmuster.
  - `style`: Optional. ANSI-Escape-Code (`StrVal`) oder numerischer Stilcode (`NumVal`).
- **Rückgabe:**
  `NullVal` (blockierend).

---

## file.SecureDelete(path)
- **Konkret:**
  Löscht eine Datei sicher durch In-Place-Verschlüsselung (AES), zufälliges Umbenennen und anschließendes Truncate.
- **Parameter:**
  - `path`: Zu löschende Datei.
- **Rückgabe:**
  `BoolVal`
  `true` bei Erfolg.

---

## file.Base64Encode(inFile, outFile)
- **Konkret:**
  Liest eine Datei ein und speichert sie Base64-kodiert in einer neuen Datei.
  Das Zielverzeichnis muss vorhanden sein.
- **Parameter:**
  - `inFile`: Quelldatei.
  - `outFile`: Zieldatei.
- **Rückgabe:**
  `NullVal`
- **Fix:** Prüft jetzt wie `file.Base64Decode`, ob das Zielverzeichnis existiert, bevor geschrieben wird (vorher fehlte dieser Check, was zu einem rohen OS-Fehler statt einer sprechenden Meldung führte).

---

## file.Base64Decode(inFile, outFile)
- **Konkret:**
  Dekodiert eine Base64-kodierte Datei und speichert das Ergebnis als Binärdatei.
  Das Zielverzeichnis muss vorhanden sein.
- **Parameter:**
  - `inFile`: Base64-Quelldatei.
  - `outFile`: Zieldatei.
- **Rückgabe:**
  `NullVal`

---

## file.CreateSymlink(target, linkPath, [replaceExisting])
- **Konkret:**
  Erstellt einen symbolischen Link auf eine Datei.
- **Parameter:**
  - `target`: Ziel des Symlinks.
  - `linkPath`: Pfad des zu erstellenden Links.
  - `replaceExisting`: Optional, `BoolVal` (Standard `false`). Steuert, ob ein bereits vorhandener Link/Datei an `linkPath` ersetzt wird.
- **Rückgabe:**
  `BoolVal`
- **Fix:** Der dritte Parameter wurde bisher entgegengenommen, aber ignoriert (`createSymlinkInternal` erhielt intern immer `false`). Er wird jetzt tatsächlich ausgewertet und durchgereicht.

---

## file.Ext(path)
- **Konkret:**
  Gibt die Dateiendung zurück (ohne führenden Punkt).
- **Parameter:**
  - `path`: Dateipfad.
- **Rückgabe:**
  `StrVal`
  Beispiel: `"txt"`, `"go"`, `""` bei keiner Endung.

---

## file.Name(path)
- **Konkret:**
  Gibt den Dateinamen ohne Verzeichnispfad und ohne Endung zurück.
- **Parameter:**
  - `path`: Dateipfad.
- **Rückgabe:**
  `StrVal`
  Beispiel: `"readme"` für `/home/user/readme.md`.

---

## file.Dir(path)
- **Konkret:**
  Gibt das übergeordnete Verzeichnis eines Pfades zurück.
- **Parameter:**
  - `path`: Dateipfad.
- **Rückgabe:**
  `StrVal`
  Beispiel: `"/home/user"` für `"/home/user/readme.md"`.

---

## file.Join(path, part, ...)
- **Konkret:**
  Verbindet mehrere Pfadsegmente plattformsicher zu einem vollständigen Pfad.
- **Parameter:**
  - Beliebig viele Segmente als `StrVal`.
- **Rückgabe:**
  `StrVal`