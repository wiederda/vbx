#use proc
' Prozess starten
Dim pid
pid = proc.Start("/usr/bin/sleep", "30")

Print "Gestartete PID: " & pid

' Existiert der Prozess?
If proc.PidExists(pid) = true then
    Print "Prozess läuft."
End If

' CPU & Memory abfragen
Dim cpu
Dim mem
cpu = proc.CPU(pid)
mem = proc.Memory(pid)

Print "CPU: " & cpu + "%"
Print "Memory (Bytes): " & mem

' Parent anzeigen
Dim ppid
ppid = proc.ParentPid(pid)
Print "Parent PID: " & ppid

' Prozess beenden
proc.Kill(pid)

If proc.PidExists(pid) = false then
    Print "Prozess beendet."
End If

' Alle sleep Prozesse finden
Dim arr
arr = proc.ExistsByName("sleep")

Print "Gefundene Prozesse: " & array.Length(arr)

For i = 0 to array.Count(arr) - 1
    Print "PID: " & array.GetIndex(arr, i)
Next

' Alle sleep Prozesse beenden
Dim count
count = proc.KillByName("sleep")

Print "Beendete Prozesse: " & count

Dim target
target = "/usr/bin/node"

If proc.CountByPath(target) = 0 Then
    Print "Service läuft nicht. Starte neu..."
    proc.Start(target, "server.js")
Else
    Print "Service läuft bereits."
End If

arr = proc.ExistsByName("myapp")

If array.Count(arr) > 0 then
    pid = array.GetIndex(arr, 0)
    Print "Beende Prozessbaum von PID: " & pid
    proc.KillTree(pid)
    Else
    Print "Prozess (PID) nicht gefunden"
End If


