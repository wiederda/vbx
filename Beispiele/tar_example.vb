#use tar

Dim files, path="/home/vbmini/testfolder"
files=array.Create(path & "/file1.txt", path & "/file2.txt", path & "/subfolder/file3.txt")

If folder.Exists(path) = true then

For i=0 to array.UBound(files)
If file.Exists(files(i)) = false then
    file.Create(path & "/" & files(i))
End If
Next

If file.Exists(path & "/archive.tar") = true then     
file.Delete(path & "/archive.tar") 
End If

If file.Exists(path & "/archive.tar") = true then     
file.Delete(path & "/archive.tar") 
End If

If file.Exists(path & "/archive.tar.gz") = true then     
file.Delete(path & "/archive.tar.gz") 
End If



Print "0=" & array.GetIndex(files, 0)

' --- 7. TAR erstellen ---
tar.Create(path & "/archive.tar", files)
tar.GzCreate(path & "/archive.tar.gz", files)

' --- 8. TAR entpacken ---
'tar.Extract("testfolder/archive.tar", "testfolder/extracted_tar")
'Print "TAR erfolgreich entpackt."

' --- 10. TAR.GZ entpacken ---
'tar.GzExtract("testfolder/archive.tar.gz", "testfolder/extracted_targz")
'Print "TAR.GZ erfolgreich entpackt."
End If


' Konfiguration
Dim logFile, archive
logFile = "C:\logs\access.log"
archive = "C:\Backups\daily_logs.tar"

' 1. Schritt: Log-Eintrag zum Tages-Archiv hinzufügen
' Da es ein normales TAR ist, geht das blitzschnell (Append)
Print "Füge " & logFile & " zum Archiv hinzu..."
tar.Add(archive, logFile)

' 2. Schritt: Prüfen, ob es Zeit für die Kompression ist
' (Beispiel: Wenn es nach 23 Uhr ist)
Dim stunde
stunde = date.Now("HH")

If val(stunde) >= 23 Then
    Print "Nachtmodus: Kompression wird gestartet..."
    
    ' Konvertiert daily_logs.tar -> daily_logs.tar.gz
    ' Das 'True' löscht das .tar nach Erfolg
    Dim finalFile
    finalFile = tar.ToGz(archive, True)
    
    Print "Archivierung abgeschlossen: " & finalFile
End If


