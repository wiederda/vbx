# ⚙️ ini.* – INI-Konfigurationsdatei-Funktionen

Dient zum Lesen, Schreiben und Verwalten von INI-Konfigurationsdateien.
Arbeitet mit einem internen Dokumenten-Zustand – `ini.Load` muss zuerst aufgerufen werden.
Schreiboperationen sind atomar (temp-Datei + Rename). Thread-sicher via `sync.RWMutex`.
Sektions- und Key-Namen sind case-sensitiv.

---

## ini.Load(filename)
- **Konkret:**
  Lädt eine INI-Datei in den Arbeitsspeicher.
  Existiert die Datei nicht, wird ein leerer Cache angelegt – das ist ein gültiger Zustand für neue Konfigurationen.
  Kommentarzeilen (`; …` und `# …`) sowie Einträge außerhalb einer Sektion werden ignoriert.
- **Parameter:**
  - `filename`: Pfad zur INI-Datei.
- **Rückgabe:**
  `NullVal` bei Erfolg, `ErrorVal` bei Lesefehler.

---

## ini.Get(section, key [, default])
- **Konkret:**
  Liest einen Wert aus einer Sektion.
  Gibt den Default-Wert zurück wenn der Key nicht gefunden wird.
- **Parameter:**
  - `section`: Sektionsname.
  - `key`: Schlüsselname.
  - `default`: Optional. Rückgabewert wenn nicht gefunden (Standard: `""`).
- **Rückgabe:**
  `StrVal`

---

## ini.Set(section, key, value [, autosave])
- **Konkret:**
  Setzt einen Wert in einer Sektion.
  Existiert die Sektion noch nicht, wird sie automatisch angelegt.
  Mit `autosave = true` (Standard) wird die Datei sofort atomar gespeichert.
  Mit `autosave = false` wird nur der Cache aktualisiert – `ini.Save` muss manuell aufgerufen werden.
- **Parameter:**
  - `section`: Sektionsname.
  - `key`: Schlüsselname.
  - `value`: Zu setzender Wert (String oder Zahl).
  - `autosave`: Optional. `BoolVal` (Standard: `true`).
- **Rückgabe:**
  `NullVal` bei Erfolg, `ErrorVal` bei Fehler.

---

## ini.Save()
- **Konkret:**
  Schreibt alle ausstehenden Änderungen manuell in die Datei.
  Nützlich nach `ini.Set` mit `autosave = false`.
  Sektionen und Keys werden alphabetisch sortiert geschrieben.
- **Rückgabe:**
  `NullVal` bei Erfolg, `ErrorVal` bei Fehler.

---

## ini.Exists(section, key)
- **Konkret:**
  Prüft ob ein Key in einer Sektion existiert und nicht leer ist.
- **Parameter:**
  - `section`: Sektionsname.
  - `key`: Schlüsselname.
- **Rückgabe:**
  `BoolVal`

---

## ini.Delete(section [, key])
- **Konkret:**
  Löscht einen einzelnen Key oder eine gesamte Sektion.
  Leere Sektionen werden automatisch entfernt.
  Änderungen werden sofort gespeichert.
- **Parameter:**
  - `section`: Sektionsname.
  - `key`: Optional. Schlüsselname. Ohne `key` wird die gesamte Sektion gelöscht.
- **Rückgabe:**
  `NullVal` bei Erfolg, `ErrorVal` bei Fehler.

---

## ini.Sections()
- **Konkret:**
  Gibt alle Sektionsnamen der geladenen INI-Datei zurück (alphabetisch sortiert).
- **Rückgabe:**
  `ArrVal`
  Array von `StrVal`-Einträgen.

---

## ini.Keys(section)
- **Konkret:**
  Gibt alle Keys einer Sektion zurück (alphabetisch sortiert).
- **Parameter:**
  - `section`: Sektionsname.
- **Rückgabe:**
  `ArrVal`
  Array von `StrVal`-Einträgen. Leer wenn Sektion nicht gefunden.