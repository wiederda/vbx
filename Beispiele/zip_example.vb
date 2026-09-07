#use zip

' ============================
' ZIP Beispiele für VBX
' ============================

Dim files, path
path = "/home/vbx/testfolder"

files = array.Create(path & "/file1.txt", path & "/file2.txt", path & "/subfolder/file3.txt")

If folder.Exists(path) = true Then

    Dim i
    For i = 0 To array.UBound(files)
        If file.Exists(files(i)) = false Then
            file.Create(files(i))
        End If
    Next

    If file.Exists(path & "/archive.zip") = true Then
        file.Delete(path & "/archive.zip")
    End If

    If file.Exists(path & "/archivewithpass.zip") = true Then
        file.Delete(path & "/archivewithpass.zip")
    End If

    ' --- 1. ZIP erstellen (mit und ohne Passwort) ---
    zip.Create(path & "/archive.zip", files)
    zip.Create(path & "/archivewithpass.zip", files, "myPass")

    Print "Files=" & array.UBound(files)
    Print "File 2=" & array.GetIndex(files, 2)

    ' --- 2. Flache ZIP erstellen (kein Unterordner-Struktur) ---
    zip.CreateFlat(path & "/flat.zip", files, "secret123")

    ' --- 3. Dateien in ZIP auflisten ---
    Dim fileList
    fileList = zip.List(path & "/archive.zip")

    Print "Dateien in archive.zip:"

    For j = 0 To array.Length(fileList) - 1
        Print fileList(j)["Name"] & " (" & fileList(j)["Size"] & " bytes)"
    Next

    ' --- 4. Prüfen, ob Datei existiert ---
    If zip.Exists(path & "/archive.zip", "file1.txt") Then
        Print "Datei file1.txt existiert im ZIP"
    Else
        Print "Datei file1.txt nicht gefunden"
    End If

    ' --- 5. ZIP entpacken ---
    zip.Extract(path & "/archive.zip", path & "/extracted")
    zip.Extract(path & "/archivewithpass.zip", path & "/extractedwithpass", "myPass")
    Print "ZIP erfolgreich entpackt."

End If