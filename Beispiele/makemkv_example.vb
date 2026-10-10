#use computer,folder,makemkv

' ------------------------------------------------------------
' Beispiel: Disc auslesen und passende Titel auf die Platte rippen
' ------------------------------------------------------------

Dim outputPath = "e:\"
Dim exe = "C:\Program Files (x86)\MakeMKV\makemkvcon.exe"

Dim minDuration = 20
Dim language = {"deu"}
Dim subtitle = {"deu"}

' ------------------------------------------------------------
' Disc ermitteln
' ------------------------------------------------------------

Dim disc = makemkv.GetDiscInfo(0, exe)

PRINT "Disc: " & disc(0)
PRINT "Laufwerk: " & disc(1)
PRINT "Pfad: " & disc(2)

' ------------------------------------------------------------
' Zielordner bestimmen
' ------------------------------------------------------------

' Ungültige Zeichen für Windows-Verzeichnisse ersetzen
Dim discName = ReplaceMany( _
disc(0), _
"\", "-", _
"/", "-", _
":", "-", _
"*", "-", _
"?", "-", _
"""", "-", _
"<", "-", _
">", "-", _
"|", "-" _
)

Dim discPath = outputPath & discName

If folder.Exists(discPath) = false Then
    folder.Create(discPath)
End If

PRINT "Zielordner: " & discPath

' ------------------------------------------------------------
' Titel ermitteln
' ------------------------------------------------------------

Dim titles = makemkv.GetTitles( _
disc(1), _
minDuration, _
language, _
subtitle, _
exe _
)

If array.Length(titles) = 0 Then
    PRINT "Keine passenden Titel gefunden."
    EXIT
End If

PRINT "Titelanzahl: " & array.Length(titles)

' ------------------------------------------------------------
' Benötigten Speicher berechnen
' ------------------------------------------------------------

Dim requiredBytes = 0

For Each title In titles
    requiredBytes = requiredBytes + title(2)
Next

PRINT "Benötigter Speicher: " & requiredBytes & " Bytes"

' ------------------------------------------------------------
' Freien Speicher prüfen
' ------------------------------------------------------------

Dim disk = computer.DiskSpace(discPath)
Dim freeBytes = disk(1)

PRINT "Freier Speicher: " & freeBytes & " Bytes"

If freeBytes < requiredBytes Then
    PRINT "FEHLER: Nicht genügend Speicherplatz."
    PRINT "Benötigt: " & requiredBytes & " Bytes"
    PRINT "Frei: " & freeBytes & " Bytes"
    EXIT
End If

PRINT "Speicherplatz ausreichend."

' ------------------------------------------------------------
' Ausgewählte Titel anzeigen
' ------------------------------------------------------------

For Each title In titles
    PRINT ""
    PRINT "Titelindex: " & title(0)
    PRINT "Dauer: " & title(1) & " Sekunden"
    PRINT "Größe: " & title(2) & " Bytes"
    PRINT "Name: " & title(3)
    PRINT "Playlist: " & title(4)
    PRINT "Datei: " & title(5)
    PRINT "Audio: " & title(6)
    PRINT "Untertitel: " & title(7)
    PRINT "Kapitel: " & title(8)
Next

' ------------------------------------------------------------
' Titel rippen
' ------------------------------------------------------------

Dim count = 0

For Each title In titles
    count = count + 1

    PRINT ""
    PRINT "Rippe " & count & "/" & array.Length(titles) & ": Titel " & title(0) & " (" & title(5) & ")"

    Dim file = makemkv.Rip(disc(1), title, discPath, exe)

    PRINT "Fertig: " & file
Next

PRINT ""
PRINT "Alle Titel gespeichert in: " & discPath