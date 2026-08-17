
' ============================
' File Beispiele für VBmini
' ============================

Print "=== FILE MODULE TEST ==="
' Datei anlegen

file.Create("c:\programme\text.txt")

file.Create("text.txt")

' Schreiben & Lesen
file.Write("text.txt", "Hallo Welt")

Dim txt
txt = file.Read("text.txt")
Print txt
Print string.ToUpper(txt)
Print string.ToUpper("HeLLo")

' Dateigröße
Print "Size:" & file.Size("text.txt")
Print "sizeHuman:" & file.Size("text.txt", "human")

' Zeitstempel
Print "ModTime=" & file.ModTime("text.txt")
Print "CreateTime=" & file.CreateTime("text.txt")
Print "AccessTime=" & file.AccessTime("text.txt")

' Prüfen ob Datei existiert
if file.Exists("text.txt") = 1 Then
    Print "Datei existiert"
EndIf

' Unblock Windows-Datei
file.Unblock("text.txt")

' Kopieren & Verschieben
file.Copy("text.txt", "text1.txt")
file.Copy("text.txt", "backup.txt", true)
file.Move("backup.txt", "text2.txt")
file.Rename("text2.txt", "backup1.txt")
'file.MoveAll("temp", "backup")

' Vergleich
if file.Compare("text.txt", "text1.txt") = 0 Then
    Print "Dateien identisch"
EndIf

' Suchen & Ersetzen
'file.Replace("d.txt", "alt", "neu")
'file.ReplaceAll("d.txt", "alt", "neu")
'lines = file.Search("d.txt", "Suchwort")

' Dateiname & Pfad
Print file.Ext("text.txt")      
Print file.Name("text.txt")    
Print file.Dir("text.txt")      

' Base64 Encode/Decode
'file.Base64Encode("image.jpg", "image.b64")
'file.Base64Encode("image.jpg", "image_unix.b64", "unix")
'file.Base64Decode("image.b64", "image2.jpg")

' Löschen
file.Delete("text.txt")
file.Delete("text1.txt")
file.Delete("backup1.txt", true)