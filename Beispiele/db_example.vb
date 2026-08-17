#use db
' --- 1. VERBINDUNGEN ÖFFNEN ---
' MSSQL (Quelle) - Nutzt Integrated Security (AD-Anmeldung)
Print "Öffne MSSQL..."
db.Open("mssql", "server=DEIN_SERVER;database=PROD_DB;integrated security=true", "src")

' SQLite (Ziel) - Verschlüsseltes lokales Backup
Print "Erstelle verschlüsselte SQLite..."
db.Open("sqlite", "test_backup.db", "dest", "SicheresPasswort123")

' --- 2. METADATEN CHECK ---
Print "Treiber Quelle: " & db.DriverName("src")
Print "Datenbank Name: " & db.DatabaseName("src")

' Tabellen auflisten
Dim tables
tables = db.ListTables("src")
Print "Gefundene Tabellen in Quelle: " & UBound(tables) + 1

' Views und Procedures (falls vorhanden)
Dim views, procs
views = db.ListViews("src")
procs = db.ListProcedures("src")
Print "Views: " & UBound(views) + 1 & " | Prozeduren: " & UBound(procs) + 1

' --- 3. DATEN-TRANSFER (DUMMY-TEST) ---
' Wir nehmen die erste Tabelle und zählen die Zeilen
If UBound(tables) >= 0 Then
    Dim firstTable, rowCount
    firstTable = tables(0)
    rowCount = db.Count("src", firstTable)
    Print "Tabelle '" & firstTable & "' hat " & rowCount & " Zeilen."

    ' Struktur in SQLite nachbauen (Minimal-Beispiel)
    db.Exec("dest", "CREATE TABLE IF NOT EXISTS backup_log (id INTEGER PRIMARY KEY, msg TEXT, ts DATETIME)")
    db.Exec("dest", "INSERT INTO backup_log (msg, ts) VALUES (?, DATETIME('now'))", "Backup gestartet für " & firstTable)
End If

' --- 4. BLOB-TEST (BILDER/DOKUMENTE) ---
' Wir ziehen ein Logo/Bild aus MSSQL und schieben es in SQLite
Print "Teste Blob-Transfer..."
Dim b64Data
' Annahme: Tabelle 'Assets' mit Spalte 'FileData'
b64Data = db.GetBlob("src", "Assets", "FileData", "AssetID", "1")

If Left(b64Data, 5) <> "error" Then
    ' Tabelle in SQLite vorbereiten
    db.Exec("dest", "CREATE TABLE IF NOT EXISTS local_assets (id TEXT, data BLOB)")
    ' Blob speichern
    db.SetBlob "dest", "local_assets", "data", "id", "1", b64Data
    Print "Blob erfolgreich kopiert."
Else
    Print "Kein Blob gefunden oder Fehler: " & b64Data
End If

' --- 5. EXECFILE & TRANSAKTIONS-TEST ---
' Wir erstellen ein kleines SQL-Skript File temporär (oder nutzen ein vorhandenes)
' Dieses Skript nutzt 'GO' (da wir es für MSSQL/Postgres vorbereitet haben)
Print "Führe Batch-Skript aus..."
Dim res
' Ein Skript mit Mix aus INSERT und SELECT
res = db.ExecFile("dest", "C:\scripts\init_test.sql", False) 

' --- 6. PASSWORT ÄNDERN (SQLCipher Test) ---
Print "Ändere Verschlüsselung..."
Dim pwChange
pwChange = db.ChangePassword("dest", "SicheresPasswort123", "NeuesPasswort456")
Print "Rekey Status: " & pwChange

' --- 7. AUFRÄUMEN ---
db.Close "src"
db.Close "dest"
Print "Test abgeschlossen. Alle Verbindungen geschlossen."


/' Verbindung zum lokalen SQL-Server öffnen
If db.Open("mssql", "server=localhost;user id=sa;password=geheim;", "LokalSQL") Then
    Print "Verbindung für Wartung geöffnet."
End If

' Name der Datenbank und das Ziel auf dem Server
Dim dbName = "Kunden_DB"
Dim ziel = "C:\SQL_Backups\Kunden_Auto_" & date.Now("yyyyMMdd") & ".bak"

Print "Starte interne Sicherung von " & dbName & "..."

' Nutzt den Alias "LokalSQL"
If db.Backup("LokalSQL", dbName, ziel) Then
    Print "OK: Datei liegt auf dem Server unter " & ziel
Else
    Print "Fehler beim internen Backup!"
End If

' Pfad auf DEINEM Rechner (dank absPath sicher auf jedem OS)
Dim meinLokalPfad = "C:\Eigene_Projekte\Backups\remote_pg_dump.sql"
If sys.OS() <> "windows" Then 
    meinLokalPfad = "/home/user/backups/remote_pg_dump.sql"
End If

' Verbindungsinformationen für einen entfernten Postgres-Server
Dim remoteConn = "postgresql://backupuser:strenggeheim@192.168.178.50:5432"

Print "Hole Daten von Remote-Postgres..."

' db.Export braucht KEIN db.Open vorher!
If db.Export("pg", remoteConn, "produktions_db", meinLokalPfad) Then
    Print "Erfolg! Backup wurde über das Netz übertragen."
    Print "Datei-Speicherort: " & meinLokalPfad
Else
    Print "Export fehlgeschlagen. Prüfe Netzwerk/pg_dump Installation."
End If'/

