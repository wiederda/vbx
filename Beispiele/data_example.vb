#use data

Print "=== data.* Plugin Test ==="
Print ""

Print "--- Datenumrechnungen (SI, Faktor 1000) ---"

Print "ByteToKb 1000          = " & data.ByteToKb(1000) & "   erwartet: 1"
Print "KbToMb 1000            = " & data.KbToMb(1000) & "   erwartet: 1"
Print "MbToGb 1000            = " & data.MbToGb(1000) & "   erwartet: 1"
Print "GbToTb 1000            = " & data.GbToTb(1000) & "   erwartet: 1"

Print ""

Print "--- Datenumrechnungen (Binär, Faktor 1024) ---"

Print "ByteToKiB 1024         = " & data.ByteToKiB(1024) & "   erwartet: 1"
Print "KiBToMiB 1024          = " & data.KiBToMiB(1024) & "   erwartet: 1"
Print "MiBToGiB 1024          = " & data.MiBToGiB(1024) & "   erwartet: 1"
Print "GiBToTiB 1024          = " & data.GiBToTiB(1024) & "   erwartet: 1"

Print ""

Print "--- Leistung ---"

Print "WattToKilowatt 1000    = " & data.WattToKilowatt(1000) & "   erwartet: 1"
Print "KilowattToWatt 1       = " & data.KilowattToWatt(1) & "   erwartet: 1000"

Print ""

Print "--- Zeitumrechnungen ---"

Print "MinutesToHours 60      = " & data.MinutesToHours(60) & "   erwartet: 1"
Print "HoursToMinutes 1       = " & data.HoursToMinutes(1) & "   erwartet: 60"
Print "SecondsToDays 86400    = " & data.SecondsToDays(86400) & "   erwartet: 1"
Print "DaysToSeconds 1        = " & data.DaysToSeconds(1) & "   erwartet: 86400"

Print ""

Print "--- FormatSeconds ---"

Print "FormatSeconds 0        = " & data.FormatSeconds(0) & "   erwartet: 00:00:00"
Print "FormatSeconds 1        = " & data.FormatSeconds(1) & "   erwartet: 00:00:01"
Print "FormatSeconds 59       = " & data.FormatSeconds(59) & "   erwartet: 00:00:59"
Print "FormatSeconds 60       = " & data.FormatSeconds(60) & "   erwartet: 00:01:00"
Print "FormatSeconds 3661     = " & data.FormatSeconds(3661) & "   erwartet: 01:01:01"
Print "FormatSeconds 86399    = " & data.FormatSeconds(86399) & "   erwartet: 23:59:59"

Print ""

Print "--- Nachkommastellen ---"

Print "ByteToKb 1500          = " & data.ByteToKb(1500) & "   erwartet: 1.5"
Print "KbToMb 2500            = " & data.KbToMb(2500) & "   erwartet: 2.5"
Print "ByteToKiB 1536         = " & data.ByteToKiB(1536) & "   erwartet: 1.5"
Print "WattToKilowatt 1250    = " & data.WattToKilowatt(1250) & "   erwartet: 1.25"
Print "MinutesToHours 90      = " & data.MinutesToHours(90) & "   erwartet: 1.5"
Print "SecondsToDays 43200    = " & data.SecondsToDays(43200) & "   erwartet: 0.5"

Print ""

Print "--- Größere Werte ---"

Print "ByteToKb 1000000       = " & data.ByteToKb(1000000) & "   erwartet: 1000"
Print "ByteToKiB 1048576      = " & data.ByteToKiB(1048576) & "   erwartet: 1024"
Print "GbToTb 5000            = " & data.GbToTb(5000) & "   erwartet: 5"
Print "GiBToTiB 2048          = " & data.GiBToTiB(2048) & "   erwartet: 2"

Print ""

Print "--- Fehlerfälle ---"

Dim result

result = data.ByteToKb("test")
Print "ByteToKb(""test""): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

result = data.KbToMb("test")
Print "KbToMb(""test""): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

result = data.MbToGb("test")
Print "MbToGb(""test""): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

result = data.GbToTb("test")
Print "GbToTb(""test""): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

result = data.ByteToKiB("test")
Print "ByteToKiB(""test""): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

result = data.KiBToMiB("test")
Print "KiBToMiB(""test""): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

result = data.MiBToGiB("test")
Print "MiBToGiB(""test""): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

result = data.GiBToTiB("test")
Print "GiBToTiB(""test""): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

result = data.WattToKilowatt("test")
Print "WattToKilowatt(""test""): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

result = data.KilowattToWatt("test")
Print "KilowattToWatt(""test""): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

result = data.MinutesToHours("test")
Print "MinutesToHours(""test""): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

result = data.HoursToMinutes("test")
Print "HoursToMinutes(""test""): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

result = data.SecondsToDays("test")
Print "SecondsToDays(""test""): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

result = data.DaysToSeconds("test")
Print "DaysToSeconds(""test""): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

result = data.FormatSeconds("test")
Print "FormatSeconds(""test""): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

Print ""

Print "=== Test abgeschlossen ==="