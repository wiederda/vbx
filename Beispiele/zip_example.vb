#use zip

' ============================
' ZIP Beispiele für VBX
' ============================

Dim files, neu, path="/home/vbx/testfolder"

files=array.Create(path & "/file1.txt", path & "/file2.txt", path & "/subfolder/file3.txt")

If folder.Exists(path) = true then

For i=0 to array.UBound(files)
If file.Exists(files(i)) = false then
    file.Create(path & "/" & files(i))
End If
Next

If file.Exists(path & "/archive.zip") = true then     
file.Delete(path & "/archive.zip") 
End If

If file.Exists(path & "/archivewithpass.zip") = true then     
file.Delete(path & "/archivewithpass.zip") 
End If

zip.Create(path & "/archive.zip", files)
zip.Create(path & "/archivewithpass.zip", files, "myPass")



Print "Files=" & array.UBound(files)
Print "File 2=" & array.GetIndex(files, 2)

' --- 2. Flache ZIP erstellen ---
zip.CreateFlat("testfolder/flat.zip", files, "secret123")

' --- 3. Dateien in ZIP auflisten ---
Dim fileList
fileList = zip.List("testfolder/archive.zip")
Print "Dateien in archive.zip:"
Print fileList

' --- 4. Prüfen, ob Datei existiert ---
If zip.Exists("testfolder/archive.zip", "testfolder/file1.txt") Then
    Print "Datei file1.txt existiert im ZIP"
Else
    Print "Datei file1.txt nicht gefunden"
EndIf

' --- 5. ZIP entpacken ---
zip.Extract(path & "/archive.zip", path & "/extracted")
zip.Extract(path & "/archivewithpass.zip", path & "/extractedwithpass","myPass")
Print "ZIP erfolgreich entpackt."

' --- 6. Folder ZIP packen ---
zip.ZipCreateFromFolder("folder.zip", "testfolder", "secret123")
Print "ZIP mit Struktur erstellt"

zip.ZipCreateFlatFromFolder("folder_flat.zip", "testfolder", "secret123")
Print "ZIP ohne Struktur erstellt"
End If