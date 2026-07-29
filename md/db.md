# 🗄️ db.* – Datenbankfunktionen

Dient zur Verbindung, Abfrage und Verwaltung von Datenbanken.
Unterstützte Treiber: `sqlite` / `sqlite3`, `postgres` / `pg`, `mssql` / `sqlserver`.
Verbindungen werden über einen Alias verwaltet – `db.Open` muss zuerst aufgerufen werden.

---

## db.Open(driver, connStr, alias)
- **Konkret:**
  Öffnet eine Datenbankverbindung und speichert sie unter einem Alias.
  Führt nach dem Verbinden automatisch einen Ping durch.
  SQLite: relative Pfade werden relativ zum aktuellen Arbeitsverzeichnis aufgelöst.
- **Parameter:**
  - `driver`: Treiber. Unterstützte Werte: `"sqlite"`, `"sqlite3"`, `"postgres"`, `"pg"`, `"mssql"`, `"sqlserver"`.
  - `connStr`: Connection-String oder Dateipfad (SQLite).
  - `alias`: Frei wählbarer Name für diese Verbindung.
- **Rückgabe:**
  `BoolVal` (`true`) bei Erfolg, `ErrorVal` bei Fehler.

---

## db.Close(alias)
- **Konkret:**
  Schließt eine Datenbankverbindung und entfernt den Alias.
- **Parameter:**
  - `alias`: Verbindungsalias.
- **Rückgabe:**
  `BoolVal` (`true`) wenn gefunden und geschlossen, `false` wenn Alias unbekannt.

---

## db.DriverName(alias)
- **Konkret:**
  Gibt den Treibernamen der Verbindung zurück.
- **Parameter:**
  - `alias`: Verbindungsalias.
- **Rückgabe:**
  `StrVal` (z. B. `"mssql"`, `"sqlite"`, `"postgres"`). `"unknown"` bei unbekanntem Alias.

---

## db.Query(alias, sql)
- **Konkret:**
  Führt ein SELECT aus und gibt alle Ergebniszeilen zurück.
  Binärdaten (Blobs) werden automatisch als Base64-String kodiert.
  Steuerzeichen (außer Tab, CR, LF) werden aus Strings entfernt.
- **Parameter:**
  - `alias`: Verbindungsalias.
  - `sql`: SELECT-Statement.
- **Rückgabe:**
  `ArrVal`
  Array von Zeilen. Jede Zeile ist ein `ArrVal` mit den Spaltenwerten.

---

## db.QueryArray(alias, sql)
- **Konkret:**
  Führt ein SELECT aus und gibt das Ergebnis als 2D-Array zurück.
- **Parameter:**
  - `alias`: Verbindungsalias.
  - `sql`: SELECT-Statement.
- **Rückgabe:**
  `KindArr2D`

---

## db.Exec(alias, sqlString)
- **Konkret:**
  Führt ein oder mehrere SQL-Statements aus.
  MSSQL: trennt am `GO`-Keyword. Andere Treiber: trennen am Semikolon.
  Kapselt automatisch in eine Transaktion, wenn kein `COMMIT`/`ROLLBACK` im SQL vorhanden ist.
- **Parameter:**
  - `alias`: Verbindungsalias.
  - `sqlString`: SQL-Statement oder mehrteiliges Script.
- **Rückgabe:**
  `ArrVal`
  Array mit den Ergebnissen je Batch.

---

## db.ExecFile(alias, path [, dryRun])
- **Konkret:**
  Liest eine `.sql`-Datei und führt sie aus.
  Splitting und Transaktionshandling identisch zu `db.Exec`.
  Mit `dryRun = true` wird am Ende ein Rollback statt Commit ausgeführt.
- **Parameter:**
  - `alias`: Verbindungsalias.
  - `path`: Pfad zur SQL-Datei.
  - `dryRun`: Optional. `BoolVal` – bei `true` werden Änderungen nicht übernommen.
- **Rückgabe:**
  `ArrVal` mit Batch-Ergebnissen, `ErrorVal` bei Fehler.

---

## db.Insert(alias, table, columns, values)
- **Konkret:**
  Fügt einen einzelnen Datensatz über Spalten- und Werte-Arrays ein.
  Dialektspezifisches Quoting und Platzhalter werden automatisch gesetzt (MSSQL: `[..]`/`@p1,@p2`; Postgres: `"..."`/`$1,$2`; sonst `"..."`/`?`).
  `columns` und `values` müssen gleich lang sein.
- **Parameter:**
  - `alias`: Verbindungsalias.
  - `table`: Zieltabelle.
  - `columns`: Array der Spaltennamen.
  - `values`: Array der Werte, gleiche Reihenfolge wie `columns`.
- **Rückgabe:**
  `BoolVal` (`true`) bei Erfolg, `ErrorVal` bei fehlenden/unpassenden Argumenten oder SQL-Fehler.
- **Beispiel:**
  ```vb
  db.Insert("main", "Kunden", {"Name", "Alter"}, {"Max", "30"})
  ```

---

## db.CopyTable(sourceAlias, targetAlias, sourceTable, targetTable)
- **Konkret:**
  Kopiert alle Zeilen einer Tabelle von einer Verbindung in eine andere.
  Liest per `SELECT *` und schreibt zeilenweise per vorbereitetem INSERT innerhalb einer Transaktion auf der Zielverbindung.
  Platzhalter sind fest `?` – bei Zielverbindung mit anderer Platzhaltersyntax (z. B. Postgres) aktuell nicht dialektspezifisch angepasst.
- **Parameter:**
  - `sourceAlias`: Alias der Quellverbindung.
  - `targetAlias`: Alias der Zielverbindung.
  - `sourceTable`: Quelltabelle.
  - `targetTable`: Zieltabelle.
- **Rückgabe:**
  `NumVal` (Anzahl kopierter Zeilen), `ErrorVal` bei fehlenden Argumenten, unbekanntem Alias oder Fehler.

---

## db.SyncTable(sourceAlias, targetAlias, table, idColumn)
- **Konkret:**
  Gleicht eine Tabelle zwischen zwei Verbindungen ab, statt wie `db.CopyTable` den kompletten Inhalt zu übertragen.
  Für jede Quellzeile wird per ID-Spalte ein UPDATE auf die Zielverbindung ausgeführt; betrifft das UPDATE keine Zeile, wird stattdessen ein INSERT ausgeführt.
  Am Ende werden Zieldatensätze gelöscht, deren ID in der Quelle nicht mehr vorkommt. Ist die Quelltabelle leer, wird die Zieltabelle komplett geleert.
  Tabellen- und Spaltenprüfung vorab (Tabelle muss in Quelle und Ziel existieren, alle Quellspalten müssen im Ziel vorhanden sein – Spaltenabgleich ist case-insensitiv).
  Läuft komplett in einer Transaktion auf der Zielverbindung (Rollback bei jedem Fehler); UPDATE/INSERT nutzen vorbereitete Statements.
- **Parameter:**
  - `sourceAlias`: Alias der Quellverbindung.
  - `targetAlias`: Alias der Zielverbindung.
  - `table`: Tabellenname (muss in Quelle und Ziel unter gleichem Namen existieren).
  - `idColumn`: Spalte, die als eindeutiger Schlüssel für den Abgleich dient.
- **Rückgabe:**
  `ArrVal` mit drei `NumVal`-Einträgen in fester Reihenfolge: `{inserts, updates, deletes}`.
  `ErrorVal` bei fehlenden Argumenten, ungültigem Tabellen-/Spaltennamen, fehlender Tabelle/Spalte oder SQL-Fehler.
- **Hinweise:**
  Bei sehr großen Quelltabellen kann der abschließende `DELETE ... WHERE idColumn NOT IN (...)`-Schritt an Parameter-Limits des Zieltreibers stoßen (MSSQL: ca. 2100 Parameter pro Statement) – für solche Fälle ggf. IDs in Batches aufteilen lassen.

---

## db.SyncTables(sourceAlias, targetAlias, idColumn, table1, table2, ...)
- **Konkret:**
  Synchronisiert mehrere Tabellen in einem Aufruf, jede über `syncTable()` (gleiche Logik wie `db.SyncTable`).
  Alle übergebenen Tabellen teilen sich dieselbe ID-Spalte (`idColumn`). Für Tabellen mit abweichendem ID-Spaltennamen `db.SyncTable` einzeln aufrufen.
  Bricht beim ersten Fehler sofort ab (keine Teilabarbeitung der restlichen Tabellen) und meldet, welche Tabelle betroffen war.
- **Parameter:**
  - `sourceAlias`: Alias der Quellverbindung.
  - `targetAlias`: Alias der Zielverbindung.
  - `idColumn`: Gemeinsame ID-Spalte für alle folgenden Tabellen.
  - `table1, table2, ...`: Beliebig viele Tabellennamen (mindestens einer).
- **Rückgabe:**
  `ArrVal`. Ein Eintrag pro Tabelle als `{Tabellenname, inserts, updates, deletes}`, plus ein abschließender Summeneintrag `{"Gesamt", totalInserts, totalUpdates, totalDeletes}`.
  `ErrorVal` (mit vorangestelltem Tabellennamen) bei fehlenden Argumenten oder falls eine der Tabellen fehlschlägt.

---

## db.ClearTable(alias, table)
- **Konkret:**
  Löscht alle Datensätze einer Tabelle.
  MSSQL und SQLite: `DELETE FROM table`.
  Postgres: `TRUNCATE TABLE table RESTART IDENTITY CASCADE` (setzt Sequenzen zurück, löscht kaskadierend abhängige Zeilen).
- **Parameter:**
  - `alias`: Verbindungsalias.
  - `table`: Zu leerende Tabelle.
- **Rückgabe:**
  `BoolVal` (`true`) bei Erfolg, `ErrorVal` bei unbekanntem Alias, nicht unterstütztem Treiber oder SQL-Fehler.

---

## db.Count(alias, table [, filter, val])
- **Konkret:**
  Zählt Zeilen in einer Tabelle.
  Mit Filter: zählt nur Zeilen die die Bedingung erfüllen.
  MSSQL: `?`-Platzhalter werden automatisch in `@p1` umgewandelt.
- **Parameter:**
  - `alias`: Verbindungsalias.
  - `table`: Tabellenname.
  - `filter`: Optional. WHERE-Bedingung (z. B. `"Status = ?"`).
  - `val`: Optional. Wert für den Platzhalter.
- **Rückgabe:**
  `NumVal`

---

## db.GetLastID(alias [, sql])
- **Konkret:**
  Gibt die zuletzt erzeugte Auto-Increment-ID zurück.
  Ohne SQL: automatisch je nach Treiber (`SCOPE_IDENTITY()` / `last_insert_rowid()`).
  Postgres: erfordert eigenes SQL mit `RETURNING id`.
- **Parameter:**
  - `alias`: Verbindungsalias.
  - `sql`: Optional. Eigenes SQL zum Abrufen der ID.
- **Rückgabe:**
  `NumVal`, `ErrorVal` bei Postgres ohne SQL.

---

## db.TableExists(alias, tableName)
- **Konkret:**
  Prüft ob eine Tabelle in der Datenbank existiert.
- **Parameter:**
  - `alias`: Verbindungsalias.
  - `tableName`: Tabellenname.
- **Rückgabe:**
  `BoolVal`

---

## db.ListTables(alias)
- **Konkret:**
  Gibt alle Tabellennamen der Datenbank zurück.
- **Parameter:**
  - `alias`: Verbindungsalias.
- **Rückgabe:**
  `ArrVal`
  Array von `StrVal`-Einträgen.

---

## db.ListColumns(alias, table)
- **Konkret:**
  Gibt alle Spaltennamen einer Tabelle zurück.
- **Parameter:**
  - `alias`: Verbindungsalias.
  - `table`: Tabellenname.
- **Rückgabe:**
  `ArrVal`
  Array von `StrVal`-Einträgen.

---

## db.ListViews(alias)
- **Konkret:**
  Gibt alle Views der Datenbank zurück.
  Format: `Schema.Name` (z. B. `"dbo.MeineView"`).
- **Parameter:**
  - `alias`: Verbindungsalias.
- **Rückgabe:**
  `ArrVal`
  Array von `StrVal`-Einträgen.

---

## db.ListProcedures(alias)
- **Konkret:**
  Gibt alle Stored Procedures zurück (MSSQL und Postgres).
  SQLite wird nicht unterstützt (leeres Array).
  Format: `Schema.Name`.
- **Parameter:**
  - `alias`: Verbindungsalias.
- **Rückgabe:**
  `ArrVal`
  Array von `StrVal`-Einträgen.

---

## db.ListDatabases(alias)
- **Konkret:**
  Listet alle Datenbanken auf dem Server auf.
  MSSQL: ohne Systemdatenbanken (`master`, `tempdb`, `model`, `msdb`).
  Postgres: ohne Templates und `postgres`.
  SQLite: gibt immer `["main"]` zurück.
- **Parameter:**
  - `alias`: Verbindungsalias.
- **Rückgabe:**
  `ArrVal`

---

## db.DatabaseName(alias)
- **Konkret:**
  Gibt den Namen der aktuell verbundenen Datenbank zurück.
- **Parameter:**
  - `alias`: Verbindungsalias.
- **Rückgabe:**
  `StrVal`

---

## db.SetBlob(alias, table, col, whereCol, whereVal, b64Data)
- **Konkret:**
  Speichert Binärdaten (als Base64-String) in einer Tabellenspalte via UPDATE.
  Dialektspezifische Platzhalter werden automatisch gesetzt.
- **Parameter:**
  - `alias`: Verbindungsalias.
  - `table`: Tabellenname.
  - `col`: Zielspalte.
  - `whereCol`: WHERE-Spalte.
  - `whereVal`: WHERE-Wert.
  - `b64Data`: Binärdaten als Base64-String.
- **Rückgabe:**
  `BoolVal` (`true`) bei Erfolg, `ErrorVal` bei Fehler.

---

## db.GetBlob(alias, table, col, whereCol, whereVal)
- **Konkret:**
  Liest Binärdaten aus einer Tabellenspalte und gibt sie als Base64-String zurück.
- **Parameter:**
  - `alias`: Verbindungsalias.
  - `table`: Tabellenname.
  - `col`: Quellspalte.
  - `whereCol`: WHERE-Spalte.
  - `whereVal`: WHERE-Wert.
- **Rückgabe:**
  `StrVal` (Base64-kodiert). Leerer String wenn kein Datensatz gefunden.

---

## db.Backup(alias, dbName, targetPath)
- **Konkret:**
  Erstellt ein Datenbank-Backup via SQL-Befehl.
  MSSQL: `BACKUP DATABASE ... TO DISK`.
  SQLite: `VACUUM INTO`.
  Postgres: nicht unterstützt (für Postgres `db.Export` verwenden).
- **Parameter:**
  - `alias`: Verbindungsalias.
  - `dbName`: Name der Datenbank.
  - `targetPath`: Zieldatei.
- **Rückgabe:**
  `BoolVal` (`true`) bei Erfolg, `ErrorVal` bei Fehler.

---

## db.Export(driver, connStr, dbName, targetPath)
- **Konkret:**
  Exportiert eine Datenbank via externe System-Tools.
  Postgres: nutzt `pg_dump`. MSSQL: nutzt `sqlcmd`.
  Die Tools müssen im PATH verfügbar sein.
- **Parameter:**
  - `driver`: Treiber (`"postgres"` oder `"mssql"`).
  - `connStr`: Connection-String.
  - `dbName`: Name der Datenbank.
  - `targetPath`: Zieldatei.
- **Rückgabe:**
  `BoolVal` (`true`) bei Erfolg, `ErrorVal` bei Fehler oder fehlendem Tool.

---

## db.Optimize(sql, dialect)
- **Konkret:**
  Bereinigt und optimiert einen SQL-String nach Dialekt-spezifischen Regeln.
- **Parameter:**
  - `sql`: SQL-Statement.
  - `dialect`: Dialekt (`"mssql"`, `"sqlite"`, `"postgres"`).
- **Rückgabe:**
  `StrVal` (optimierter SQL-String).

---

## db.OptimizeFile(path, dialect [, outPath])
- **Konkret:**
  Liest eine `.sql`-Datei, optimiert sie und schreibt das Ergebnis zurück.
  Ohne `outPath` wird die Quelldatei überschrieben.
  Nur `.sql`-Dateien werden akzeptiert.
- **Parameter:**
  - `path`: Pfad zur SQL-Datei.
  - `dialect`: Dialekt.
  - `outPath`: Optional. Zielpfad (muss ebenfalls `.sql` enden).
- **Rückgabe:**
  `StrVal` (Bestätigungsmeldung), `ErrorVal` bei Fehler.