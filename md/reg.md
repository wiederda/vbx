# 🗝️ reg.* – Windows-Registry-Funktionen

Dient zum Lesen, Schreiben, Prüfen und Löschen von Registry-Einträgen.
**Plattform: Ausschließlich Windows.**

---

## reg.Read(root, path, name)
- **Konkret:**
  Liest einen String-Wert aus der Registry.
- **Parameter:**
  - `root`: Stammschlüssel. `"HKCU"` (Current User) oder `"HKLM"` (Local Machine).
  - `path`: Pfad zum Registry-Key (z. B. `"Software\MyApp"`).
  - `name`: Name des Wertes.
- **Rückgabe:**
  `StrVal` bei Erfolg, `ErrorVal` bei Fehler.

---

## reg.Write(root, path, name, data)
- **Konkret:**
  Schreibt einen String-Wert in die Registry.
  Erstellt den Key automatisch, falls er nicht existiert.
- **Parameter:**
  - `root`: Stammschlüssel. `"HKCU"` oder `"HKLM"`.
  - `path`: Pfad zum Registry-Key.
  - `name`: Name des Wertes.
  - `data`: Zu schreibender String.
- **Rückgabe:**
  `StrVal` (`"OK"`) bei Erfolg, `ErrorVal` bei Fehler.

---

## reg.Exists(root, path [, name])
- **Konkret:**
  Prüft, ob ein Registry-Key oder ein spezifischer Wert existiert.
  Mit 2 Argumenten wird nur das Vorhandensein des Keys geprüft.
  Mit 3 Argumenten wird zusätzlich der Wert geprüft.
- **Parameter:**
  - `root`: Stammschlüssel. `"HKCU"` oder `"HKLM"`.
  - `path`: Pfad zum Registry-Key.
  - `name`: Optional. Name des Wertes.
- **Rückgabe:**
  `BoolVal`

---

## reg.Delete(root, path [, name])
- **Konkret:**
  Löscht einen einzelnen Wert oder einen gesamten Key rekursiv inklusive aller Unterkeys.
  Mit 2 Argumenten wird der gesamte Pfad rekursiv gelöscht.
  Mit 3 Argumenten wird nur der angegebene Wert gelöscht.
- **Hinweis:**
  Die rekursive Key-Löschung ist unwiderruflich. Alle Unterkeys werden ohne Rückfrage entfernt.
- **Parameter:**
  - `root`: Stammschlüssel. `"HKCU"` oder `"HKLM"`.
  - `path`: Pfad zum Registry-Key.
  - `name`: Optional. Name des zu löschenden Wertes.
- **Rückgabe:**
  `StrVal` (`"OK"`) bei Erfolg, `ErrorVal` bei Fehler.