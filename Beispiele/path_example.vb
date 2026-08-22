
' ============================
' Path Beispiele für VBX
' ============================

Dim pfad

pfad = "/testfolder/subfolder/file3.txt"
Print "Pfad=" & pfad

Print "Ext=" & file.Ext(pfad)  ' <-- hier Str holen

Print "Name=" & file.Name(pfad)

Print "Dir=" & file.Dir(pfad)

Dim fullPath
fullPath = file.Join("C:\Ordner", "Unterordner", "Datei.txt")
Print "Join=" & fullPath

Dim text 
text = "Dateiname.txt"
Dim leftPart
Dim rightPart

leftPart = Left(text, 4)   ' => "Date"
rightPart = Right(text, 4)
Print "Left=" & leftPart
Print "Right=" & rightPart

Dim parts 
parts = array.Split("C:\Ordner\Datei.txt", "\")
Print "0=" & parts(0) 
Print "1=" & parts(1)  
Print "2=" & parts(2) 

Print "Array=" & array.Join(parts,"\")

Print "Ubound=" & array.UBound(parts)