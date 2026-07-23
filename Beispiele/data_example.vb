#use data
' Datenmengen
Dim bytes
Dim kb
Dim mb
Dim gb
Dim tb

bytes = 2048
kb = data.ByteToKb(bytes)      ' 1000er KB
Print bytes & " Byte = " & kb & " KB"

kb = 2048
mb = data.KbToMb(kb)
Print kb & " KB = " & mb & " MB"

mb = 1024
gb = data.MbToGb(mb)
Print mb & " MB = " & gb & " GB"

gb = 2
tb = data.GbToTb(gb)
Print gb & " GB = " & tb & " TB"

' Binär-Konvertierung
bytes = 4096
Print bytes & " Byte = " & data.ByteToKiB(bytes) & " KiB"
Print bytes & " Byte = " & data.ByteToKb(bytes) & " KB"

' Leistung
Dim watt, kw

watt = 1500
kw = data.WattToKilowatt(watt)
Print watt & " W = " & kw & " kW"

kw = 2
watt = data.KilowattToWatt(kw)
Print kw & " kW = " & watt & " W"

' Zeit
Dim minutes
Dim hours
Dim days
Dim seconds

minutes = 120
hours = data.MinutesToHours(minutes)
Print minutes & " Minuten = " & hours & " Stunden"

hours = 3
minutes = data.HoursToMinutes(hours)
Print hours & " Stunden = " & minutes & " Minuten"

days = 1.5
seconds = data.DaysToSeconds(days)
Print days & " Tage = " & seconds & " Sekunden"

seconds = 3600
days = data.SecondsToDays(seconds)
Print seconds & " Sekunden = " & days & " Tage"
