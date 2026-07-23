#use proc,date

Print "--- vbmini System-Monitor (" & vbPlatform() & ") ---"
Print "Drücke STRG+C zum Beenden."
Print "------------------------------------------"

Do
    Dim pids = proc.ListPids()
    Dim count = array.UBound(pids)
    
    ' Arrays für unsere Top 5
    Dim topNames(5)
    Dim topMems(5)
    Dim topPids(5)

    ' Alle Prozesse durchlaufen
    For i = 0 To count
        Dim p = pids(i)
        Dim m = proc.Memory(p) ' RSS (Physisch) ist Standard

        ' Einfacher Sortier-Algorithmus (Insert Sort Style)
        For j = 0 To 4
            If m > topMems(j) Then
                ' Platz machen und nach unten schieben
                For k = 4 To j + 1 Step -1
                    topMems(k) = topMems(k-1)
                    topNames(k) = topNames(k-1)
                    topPids(k) = topPids(k-1)
                Next
                
                ' Neuen Wert eintragen
                topMems(j) = m
                topPids(j) = p
                Dim info = proc.Info(p)
                topNames(j) = info(0)
                Exit For
            End If
        Next
    Next

    ' Ausgabe der Top 5
    Print "Zeit: " & date.Now("HH:mm:ss") & " | Prozesse gesamt: " & (count + 1)
    For i = 0 To 4
        If topPids(i) > 0 Then
            Dim mb = ToInt(topMems(i) / 1024 / 1024)
            Print "#" & (i+1) & ": " & topNames(i) & " (" & topPids(i) & ") -> " & mb & " MB"
        End If
    Next
    Print "------------------------------------------"

    ' WARNUNG / SCHUTZ: Ohne Sleep würde die CPU glühen!
    ' Wir warten 2 Sekunden bis zum nächsten Scan.
    Sleep(2000) 
Loop