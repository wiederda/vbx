#use kuma

' ============================
' Uptime-Kuma Beispiele für VBmini
' ============================

Dim server, user, password, secret
server = "http://"
user = "test"
password = "test"
secret = ""   ' TOTP-Secret eintragen, falls 2FA aktiv ist, sonst leer lassen

Dim monitorIDs
monitorIDs = {5, 7}   ' anpassen an eure echten Monitor-IDs

Print "=== kuma.SetMaintenance ==="
Dim maintenanceID
maintenanceID = kuma.SetMaintenance(server, user, password, monitorIDs, 20, secret)
Print "Ergebnis: " & maintenanceID

If Left(maintenanceID, 6) = "error:" Then
    Print "FEHLER bei SetMaintenance - Abbruch."
    Exit
End If

Print ""
Print "=== kuma.PauseMonitor (Monitor 5) ==="
Dim resultPause
resultPause = kuma.PauseMonitor(server, user, password, 5, secret)
Print "Ergebnis: " & resultPause

Print ""
Print "=== kuma.ResumeMonitor (Monitor 5) ==="
Dim resultResume
resultResume = kuma.ResumeMonitor(server, user, password, 5, secret)
Print "Ergebnis: " & resultResume

Print ""
Print "=== Simuliere 'Update war schneller fertig als geplant' ==="
Print "=== kuma.StopMaintenance ==="
Dim resultStop
resultStop = kuma.StopMaintenance(server, user, password, maintenanceID, secret)
Print "Ergebnis: " & resultStop

Print ""
Print "=== kuma.DeleteMaintenance (Aufräumen) ==="
Dim resultDelete
resultDelete = kuma.DeleteMaintenance(server, user, password, maintenanceID, secret)
Print "Ergebnis: " & resultDelete

Print ""
Print "=== Test abgeschlossen ==="