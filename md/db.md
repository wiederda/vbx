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