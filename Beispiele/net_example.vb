#use net
Print "--- Quick Service Check ---"
Dim host = "google.de"
Dim ips = net.ResolveAll(host)

If ips(0) = 1 Then

    For i = 1 To array.UBound(ips) Step 2
    Dim version = ips(i)
    Dim adresse = ips(i + 1)
    Print "Gefunden (IPv" & version & "): " & adresse
    Next
       
    ' Kurzer Check auf Port 443 (HTTPS)
    If net.Connect(ips(2), 443) = 1 Then
        Print "Status: HTTPS ist ONLINE"
    Else
        Print "Status: HTTPS ist OFFLINE"
    End If
Else
    Print "Fehler: Host nicht gefunden."
End If

If net.Ping(host) = 1 Then
    Print "Host erreichbar"
Else
    Print "Host nicht erreichbar"
End If