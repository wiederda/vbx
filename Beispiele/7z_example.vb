#use 7z

' ============================
' 7z Beispiele für VBX
' ============================

Dim files, path="/home/vbx/testfolder"

files=array.Create(path & "/file1.txt", path & "/file2.txt", path & "/subfolder/file3.txt")

If folder.Exists(path) = true then

For i=0 to array.UBound(files)
If file.Exists(files(i)) = false then
    file.Create(path & "/" & files(i))
End If
Next

If file.Exists(path & "/archive.7z") = true then     
file.Delete(path & "/archive.7z") 
End If

If file.Exists(path & "/archivewithpass.7z") = true then     
file.Delete(path & "/archivewithpass.7z") 
End If

' Optional: portable 7z-Installation angeben, falls nicht im PATH
' 7z.SetBinaryPath("D:\Tools\7-Zip\7z.exe")

7z.Create(path & "/archive.7z", files)
7z.Create(path & "/archivewithpass.7z", files, "myPass")

Print "Files=" & array.UBound(files)
Print "File 2=" & array.GetIndex(files, 2)

' --- Dateien im Archiv auflisten ---
Dim fileList
fileList = 7z.List(path & "/archive.7z")
Print "Dateien in archive.7z:"
Print fileList

Dim fileNames
fileNames = 7z.ListNames(path & "/archive.7z")
Print "Dateinamen in archive.7z:"
Print fileNames

' --- Archiv entpacken ---
7z.Extract(path & "/archive.7z", path & "/extracted")
7z.Extract(path & "/archivewithpass.7z", path & "/extractedwithpass", "myPass")
Print "7z erfolgreich entpackt."

End If