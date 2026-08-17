# 📁 folder.* – Verzeichnisfunktionen

Dient zum Erstellen, Kopieren, Verschieben, Durchsuchen und Verwalten von Verzeichnissen.
Plattformübergreifend (Windows, Linux, macOS). Schreiboperationen nutzen `absPathVal` zur Pfadabsicherung.

---

## folder.Create(path)
- **Konkret:**
  Erstellt ein Verzeichnis rekursiv inklusive aller fehlenden Elternverzeichnisse.
  Existiert der Ordner bereits, wird kein Fehler zurückgegeben.
- **Parameter:**
  - `path`: Zu erstellender Verzeichnispfad.
- **Rückgabe:**
  `BoolVal`
  `true` wenn der Ordner existiert oder erfolgreich erstellt wurde.

---

## folder.Delete(path [, force])
- **Konkret:**
  Löscht einen Ordner und seinen gesamten Inhalt.
  Mit `force` wird versucht, Schreibschutz zu entfernen und so viele Dateien wie möglich zu löschen.
  Gibt bei normalem Erfolg ein leeres Array zurück.
  Im Force-Modus wird ein Array mit Pfaden zurückgegeben, die nicht gelöscht werden konnten.
- **Hinweis:**
  Windows-Pfade über MAX_PATH werden intern über `LongPath` abgesichert.
- **Parameter:**
  - `path`: Zu löschendes Verzeichnis.
  - `force`: Optional. `BoolVal` – bei `true` wird Schreibschutz ignoriert.
- **Rückgabe:**
  `ArrVal`
  Leer bei Erfolg. Enthält nicht gelöschte Pfade als `StrVal` bei teilweisem Fehler.

---

## folder.Exists(path)
- **Konkret:**
  Prüft, ob ein Pfad existiert und ein Verzeichnis (kein File) ist.
- **Parameter:**
  - `path`: Zu prüfender Pfad.
- **Rückgabe:**
  `BoolVal`

---

## folder.Size(path [, human])
- **Konkret:**
  Gibt die Gesamtgröße aller Dateien im Verzeichnis (rekursiv) zurück.
- **Parameter:**
  - `path`: Verzeichnispfad.
  - `human`: Optional. `BoolVal` – bei `true` lesbare Größenangabe (z. B. `"4.2 MB"`).
- **Rückgabe:**
  `NumVal` (Bytes) oder `StrVal` bei `human`.

---

## folder.Count(path [, ignore])
- Konkret: Gibt eine Zusammenfassung mit Anzahl Ordner, Dateien und Gesamtgröße zurück (rekursiv).
- Parameter:
   - `path`: Verzeichnispfad.
   - `ignore`: Optional. Liste von Datei-/Ordnernamen (Basename, nicht Pfad), die von der Zählung ausgeschlossen werden.
     Vergleich ist exakt und case-sensitive. Kann als Komma-String (`"node_modules, .git"`)
     oder als Array (`{"node_modules", ".git"}`) übergeben werden.
     Ein ignorierter Ordner wird komplett übersprungen (Ordner selbst + gesamter Inhalt zählen nicht mit).
     Eine ignorierte Datei wird nur selbst übersprungen, der restliche Ordnerinhalt zählt normal.
- Hinweis: Nicht lesbare Unterordner (z. B. fehlende Berechtigung) werden derzeit still übersprungen
  und mindern die Zählung ohne Fehlermeldung.
- Rückgabe: `StrVal` Beispiel: `"Ordner: 3, Dateien: 12 | Größe: 1.4 MB"`

---

## folder.ModTime(path)
- **Konkret:**
  Gibt den Zeitpunkt der letzten Änderung eines Verzeichnisses zurück.
- **Parameter:**
  - `path`: Verzeichnispfad.
- **Rückgabe:**
  `StrVal`
  Format: ISO 8601 / RFC3339.

---

## folder.CreateTime(path)
- **Konkret:**
  Gibt den Erstellungszeitpunkt eines Verzeichnisses zurück.
- **Parameter:**
  - `path`: Verzeichnispfad.
- **Rückgabe:**
  `StrVal`
  Format: ISO 8601 / RFC3339.

---

## folder.AccessTime(path)
- **Konkret:**
  Gibt den Zeitpunkt des letzten Zugriffs auf ein Verzeichnis zurück.
- **Parameter:**
  - `path`: Verzeichnispfad.
- **Rückgabe:**
  `StrVal`
  Format: ISO 8601 / RFC3339.

---

## folder.Copy(src, dst [, progress, network])
- **Konkret:**
  Kopiert einen Ordner rekursiv mit parallelen Worker-Goroutinen.
  Im lokalen Modus werden max. 4 Worker genutzt, im Netzwerkmodus bis zu 16.
  Bei aktiviertem `progress` wird ein Echtzeit-Fortschrittsbalken auf stdout ausgegeben.
- **Parameter:**
  - `src`: Quellverzeichnis.
  - `dst`: Zielverzeichnis.
  - `progress`: Optional. `BoolVal` – Fortschrittsanzeige.
  - `network`: Optional. `BoolVal` – optimierter Modus für Netzwerk-Transfers.
- **Rückgabe:**
  `NullVal`

---

## folder.Move(src, dst)
- **Konkret:**
  Verschiebt einen Ordner. Versucht zuerst ein atomares Rename.
  Schlägt das fehl (z. B. laufwerksübergreifend), wird auf Copy + Delete zurückgegriffen.
- **Parameter:**
  - `src`: Quellverzeichnis.
  - `dst`: Zielverzeichnis.
- **Rückgabe:**
  `NullVal` bei Erfolg, `ErrorVal` bei Fehler.

---

## folder.Rename(oldPath, newPath)
- **Konkret:**
  Benennt einen Ordner um oder verschiebt ihn auf demselben Volume (direktes `os.Rename`).
  Für laufwerksübergreifende Operationen `folder.Move` verwenden.
- **Parameter:**
  - `oldPath`: Aktueller Pfad.
  - `newPath`: Neuer Pfad.
- **Rückgabe:**
  `NullVal` bei Erfolg, `ErrorVal` bei Fehler.

---

## folder.PathCombine(parts...)
- **Konkret:**
  Verbindet mehrere Pfadsegmente plattformsicher zu einem vollständigen Pfad.
  Verwendet betriebssystemspezifische Trennzeichen.
- **Parameter:**
  - Mindestens zwei Pfadsegmente als `StrVal`.
- **Rückgabe:**
  `StrVal`

---

## folder.GetFiles(path [, pattern, recursive, fullPath])
- **Konkret:**
  Gibt ein Array mit Dateinamen (oder vollständigen Pfaden) im Verzeichnis zurück.
- **Parameter:**
  - `path`: Verzeichnispfad (Standard: `"."`).
  - `pattern`: Optional. Glob-Muster (Standard: `"*"`).
  - `recursive`: Optional. `BoolVal` – Unterverzeichnisse einschließen.
  - `fullPath`: Optional. `BoolVal` – vollständige Pfade statt nur Dateinamen.
- **Rückgabe:**
  `ArrVal`
  Array von `StrVal`-Einträgen.

---

## folder.GetSubFolders(path [, pattern, recursive, fullPath])
- **Konkret:**
  Gibt ein Array mit Unterverzeichnisnamen (oder vollständigen Pfaden) zurück.
  Identische Parameter wie `folder.GetFiles`.
- **Parameter:**
  - `path`: Verzeichnispfad (Standard: `"."`).
  - `pattern`: Optional. Glob-Muster.
  - `recursive`: Optional. `BoolVal`.
  - `fullPath`: Optional. `BoolVal`.
- **Rückgabe:**
  `ArrVal`

---

## folder.GetDirectories(path [, pattern])
- **Konkret:**
  Gibt ein Array mit den Namen direkt enthaltener Unterverzeichnisse zurück, die dem Muster entsprechen.
  Nicht rekursiv.
- **Parameter:**
  - `path`: Verzeichnispfad (Standard: `"."`).
  - `pattern`: Optional. Glob-Muster (Standard: `"*"`).
- **Rückgabe:**
  `ArrVal`
  Array von `StrVal`-Einträgen.

---

## folder.CreateSymlink(target, linkPath)
- **Konkret:**
  Erstellt einen symbolischen Link auf ein Verzeichnis.
- **Parameter:**
  - `target`: Zielverzeichnis des Symlinks.
  - `linkPath`: Pfad des zu erstellenden Links.
- **Rückgabe:**
  `BoolVal`

---

## folder.EmptyFolder(path [, force])
- **Konkret:**
  Löscht den gesamten Inhalt eines Ordners, behält aber den Ordner selbst.
- **Parameter:**
  - `path`: Zu leerendes Verzeichnis.
  - `force`: Optional. Schreibschutz ignorieren.
- **Rückgabe:**
  `ArrVal`
  Array der Pfade, die nicht gelöscht werden konnten. Leer bei vollständigem Erfolg.

---

## folder.isFolderEmpty(path)
- **Konkret:**
  Prüft, ob ein Ordner keine Dateien oder Unterordner enthält.
- **Parameter:**
  - `path`: Verzeichnispfad.
- **Rückgabe:**
  `BoolVal`

---

## folder.FindDuplicates(path [, pattern])
- **Konkret:**
  Sucht nach Dateien mit identischer Größe und gruppiert sie als Duplikat-Kandidaten.
  Leere Dateien werden ignoriert.
  Für eine exakte Prüfung anschließend `folder.CheckHash` verwenden.
- **Parameter:**
  - `path`: Wurzelverzeichnis. Kann auch ein `ArrVal` mehrerer Wurzelverzeichnisse sein – nützlich um z. B. mehrere NAS-Freigaben in einem Durchlauf zu vergleichen.
  - `pattern`: Optional. Glob-Muster (Standard: `"*"`).
- **Rückgabe:**
  `ArrVal`
  Array von Gruppen. Jede Gruppe ist ein `ArrVal` mit Maps:

  | Schlüssel | Typ      | Inhalt          |
  |-----------|----------|-----------------|
  | `path`    | `StrVal` | Absoluter Pfad  |
  | `size`    | `NumVal` | Größe in Bytes  |

  Bei mehreren Wurzelverzeichnissen werden gleich große Dateien gruppenübergreifend zusammengeführt, unabhängig davon unter welcher Wurzel sie liegen.

---

## folder.CheckHash(groupsArray)
- **Konkret:**
  Validiert Duplikat-Kandidaten (z. B. aus `folder.FindDuplicates`) in zwei Stufen:
  zuerst Quick-Hash, dann vollständiger SHA256-Vergleich.
  Gibt nur tatsächlich identische Dateien zurück.
- **Parameter:**
  - `groupsArray`: `ArrVal` von Gruppen im Format von `folder.FindDuplicates`.
- **Rückgabe:**
  `ArrVal` (2D)
  Gruppen bestätigter Duplikate.

---

## folder.FindInFiles(path, ext, search [, pattern, flags])
- **Konkret:**
  Durchsucht Dateien zeilenweise nach einem Suchbegriff.
  Verarbeitet Dateien parallel (max. 8 gleichzeitig).
- **Parameter:**
  - `path`: Wurzelverzeichnis.
  - `ext`: Dateiendung (z. B. `".log"`). Leerstring = alle Dateien.
  - `search`: Suchbegriff.
  - `pattern`: Optional. Glob-Muster (Standard: `"*"`).
  - `flags`: Optional. `"i"` = case-insensitive.
- **Rückgabe:**
  `ArrVal`
  Array von Treffern. Jeder Treffer ist ein `ArrVal`:
  `[Zeilennummer, Zeileninhalt, Dateipfad]`

---

## folder.SecureDelete(path)
- **Konkret:**
  Löscht ein Verzeichnis rekursiv durch kryptografisches Shredding jeder einzelnen Datei.
  Jede Datei wird mit AES-CTR verschlüsselt überschrieben, truncated, zufällig umbenannt und dann gelöscht.
  Anschließend wird die Verzeichnisstruktur entfernt.
- **Parameter:**
  - `path`: Zu shreddendes Verzeichnis.
- **Rückgabe:**
  `BoolVal`
  `true` bei vollständigem Erfolg, `ErrorVal` bei Fehler.