#use sftp

' ============================
' SFTP Beispiele für VBX
' ============================

Dim files, found, downloaded, bytesUp, bytesDown

Print "=== sftp.Connect (Passwort) ==="
' Mit knownHostsPath -> TOFU-Host-Key-Prüfung, ohne -> ungeprüft
sftp.Connect("server1.local", "vbx", "geheim123", "server1", "/home/vbx/.ssh/known_hosts", 22)
Print "Verbindung 'server1' aufgebaut (Passwort-Auth)"
Print ""

Print "=== sftp.ConnectWithKey ==="
sftp.ConnectWithKey("server2.local", "vbx", "/home/vbx/.ssh/id_ed25519", "server2", "/home/vbx/.ssh/known_hosts", 22)
Print "Verbindung 'server2' aufgebaut (Key-Auth)"
Print ""

Print "=== sftp.List ==="
files = sftp.List("server2", "/remote/backups", "modtime", true)
For i = 0 To array.UBound(files)
    Print files(i)("name") & " | " & files(i)("size") & " Bytes | " & files(i)("modTime")
Next
Print ""

Print "=== sftp.Upload ==="
bytesUp = sftp.Upload("server2", "/home/vbx/lokal/report.pdf", "/remote/backups/report.pdf")
Print "Hochgeladen: " & bytesUp & " Bytes"
Print ""

Print "=== sftp.Download ==="
bytesDown = sftp.Download("server2", "/remote/backups/report.pdf", "/home/vbx/downloads/report.pdf")
Print "Heruntergeladen: " & bytesDown & " Bytes"
Print ""

Print "=== sftp.FindByExt (einzelne Datei) ==="
found = sftp.FindByExt("server2", "/remote/backups", "sql")
If found = "" Then
    Print "Keine .sql-Datei gefunden"
Else
    Print "Gefunden: " & found
End If
Print ""

Print "=== sftp.FindByExt (alle Treffer) ==="
files = sftp.FindByExt("server2", "/remote/backups", "sql", true)
For i = 0 To array.UBound(files)
    Print "Treffer: " & files(i)
Next
Print ""

Print "=== sftp.DownloadByExt ==="
downloaded = sftp.DownloadByExt("server2", "/remote/backups", "sql", "/home/vbx/downloads", true)
Print downloaded & " .sql-Dateien heruntergeladen"
Print ""

Print "=== sftp.DownloadFolder ==="
' Legt lokal /home/vbx/downloads/backups an (letztes Segment von remotePath)
downloaded = sftp.DownloadFolder("server2", "/remote/backups", "/home/vbx/downloads", true, true)
Print downloaded & " Dateien aus /remote/backups heruntergeladen (rekursiv, mit Fortschrittsanzeige)"
Print ""

Print "=== sftp.Close ==="
sftp.Close("server1")
sftp.Close("server2")
Print "Verbindungen 'server1' und 'server2' geschlossen"
Print ""

Print "=== Test abgeschlossen ==="
