Print "=== FOLDER MODULE FULL TEST ==="

Dim base
Dim dir
Dim copyDir
Dim movedDir
Dim renamedDir

' Testordner definieren
base       = "testfolder_test/sub"
dir        = base & "/MyFolder"
copyDir    = base & "/CopyFolder"
movedDir   = base & "/MovedFolder"
renamedDir = base & "/RenamedFolder"

' ----------------------------
' Erstellen von Ordnern
' ----------------------------
folder.Create(base)
folder.Create(dir)

Print "Exists Base: " & folder.Exists(base)      '  True
Print "Exists Dir: " & folder.Exists(dir)        '  True

' ----------------------------
' Dateien erstellen
' ----------------------------
For i = 0 To 3
    file.Write(base & "/text" & i & ".txt", "Hallo Welt")
Next

file.Write(dir & "/text.txt", "Hallo Welt")

' ----------------------------
' PathCombine testen
' ----------------------------
Print "PathCombine: " & folder.PathCombine(base, "Sub", "File.txt")

' ----------------------------
' Dateien / Ordner auflisten
' ----------------------------
Print "Files im Base: " & array.Join(folder.GetFiles(base), ", ")
Print "Folders im Base: " & array.Join(folder.GetDirectories(base), ", ")
Print "Alle Inhalte rekursiv: " & array.Join(folder.Dir(base, "*", True), ", ")
Print "Count (Files + Folders): " & folder.Count(base)

' ----------------------------
' Ordnerzustand
' ----------------------------
Print "IsEmpty Base: " & folder.IsEmpty(base)

' ----------------------------
' Zeitangaben
' ----------------------------
Print "Modification Time: " & folder.ModTime(base)
Print "Creation Time: " & folder.CreateTime(base)
Print "Access Time: " & folder.AccessTime(base)

' ----------------------------
' Kopieren / Verschieben / Umbenennen
' ----------------------------
folder.Copy(dir, copyDir)
Print "Copy Exists: " & folder.Exists(copyDir)

folder.Move(copyDir, movedDir)
Print "Moved Exists: " & folder.Exists(movedDir)

folder.Rename(movedDir, renamedDir)
Print "Renamed Exists: " & folder.Exists(renamedDir)

' ----------------------------
' Duplikate suchen
' ----------------------------
' Optional: kann große Arrays erzeugen
'Print "Duplicate scan: " & array.Join(folder.FindDuplicates(base), ", ")

' ----------------------------
' Aufräumen
' ----------------------------
folder.Empty(base)
folder.Delete(renamedDir)
folder.Delete(base)

Print "=== TEST END ==="

Dim shares = Array("\\nas\Freigabe1", "\\nas\Freigabe2")
Dim groups = folder.FindDuplicates(shares, "*")

For Each grp In groups
    For i = 0 To array.Count(grp) - 1
        Print grp(i)["path"]
    Next
Next