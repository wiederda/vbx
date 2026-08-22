#use computer,net

' ============================
' Computer Beispiele für VBX
' ============================

Print "Benutzername: " & UserName()
Print "Computername: " & ComputerName()
Print "Betriebssystem: " & OS()
Print "Architektur: " & Arch()
Print "CPU-Kerne: " & computer.CPUCount()

Dim ips = net.LocalIPs()
Print "IPv4 Adressen:"
For i = 0 To array.UBound(ips)
    Print "  " & ips(i)
Next

Dim macs = net.MACs()
Print "MAC Adressen:"
For i = 0 To array.UBound(macs)
    Print "  " & macs(i)
Next

Print "Öffentliche IP: " & net.PublicIP()

Dim disks = computer.Disks()
Print "Laufwerke:"
For i = 0 To array.UBound(disks)
    Print "  " & disks(i)
Next

Dim space = computer.DiskSpace(disks(0))
Print "Speicher von " & disks(0) & ": Total=" & FormatSize(space(0)) & " Free=" & FormatSize(space(1)) & " Used=" & FormatSize(space(2))