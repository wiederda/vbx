' ==============================================================
' TEST-SUITE: DATE-MODULE (GO-BACKEND 2026)
' ==============================================================
Dim nowVal, d1, d2, currentYear, ts, silvesterFull, silvesterShort

Print "=== date.Now & Parts ==="
nowVal = date.Now() ' Speichert den Standard-String (DD.MM.YYYY HH:mm)
Print "Full datetime:  " & date.Now("DD.MM.YYYY HH:mm")
Print "Year:           " & date.Now("year")
Print "Month:          " & date.Now("month")
Print "ISO Week:       " & date.Now("week")
Print "Weekday Name:   " & date.Now("weekdayname")
Print "Day of Year:    " & date.Now("yearday")
Print ""

Print "=== date.DateDiff (Differenzen) ==="
d1 = "01.01.2026"
d2 = "10.01.2026"

Print "=== date.Format ==="
' Nutzt ein vorhandenes Datum und formatiert es neu
Print "Custom Format:  " & Format(nowVal, "YYYYMMDD")
Print "Time only:      " & Format(nowVal, "HH:mm:ss")
Print ""

Print "=== date.Add (Manipulation) ==="
Print "Now +1 day:     " & date.Add(nowVal, 1, "days")
Print "Now -1 week:    " & date.Add(nowVal, -7, "days", "DD.MM.YYYY")
Print "Now +2 hours:   " & date.Add(nowVal, 2, "hours", "HH:mm")
Print "Now +1 month:   " & date.Add(nowVal, 1, "months", "month") 
Print ""

Print "=== date.TimeAdd (Reine Uhrzeit-Arithmetik) ==="
' Arbeitet nur auf HH:mm Basis, ohne Datumskontext
Print "21:00 + 4h:     " & date.TimeAdd("21:00", 4, "hours")   ' Sollte 01:00 ergeben
Print "08:00 - 90min:  " & date.TimeAdd("08:00", -90, "minutes")
Print ""


Print "Diff in Days:   " & date.DateDiff("d", d1, d2)
Print "Diff in Weeks:  " & date.DateDiff("w", d1, d2)
Print ""

Print "=== Schaltjahr-Logik ==="
currentYear = date.Now("year")
Print "Is " & currentYear & " Leap?  " & date.IsLeapYear(currentYear)
Print "Next Leap Year:    " & date.NextLeapYear(currentYear)
Print ""

Print "=== Jahresende & Formatierung ==="
' Logik in Go, Formatierung in VB
silvesterFull = date.EndOfYear(nowVal)
Print silvesterFull

silvesterShort = Format(silvesterFull, "DD.MM.YYYY")

Print "Silvester (Full): " & silvesterFull
Print "Silvester (Kurz): " & silvesterShort
Print ""

Print "=== Perioden (Start/End) ==="
Print "Start of Month: " & date.StartOfMonth(nowVal)
Print "End of Month:   " & date.EndOfMonth(nowVal, "DD.MM.YYYY")
Print "Start of Year:  " & date.StartOfYear(nowVal, "year")
Print ""

Print "=== date.Parse (Intelligentes Parsing) ==="
' Syntax: date.Parse(InputString, [OutputFormat/Part])
Print "Parse to Year:  " & date.Parse("15.08.2026", "year")
Print "Parse & Format: " & date.Parse("2026-12-31", "DD.MM.YYYY")
Print ""

Print "=== UTC / Local Konvertierung ==="
' date.UTC() liefert aktuelle Zeit in UTC
' date.ToUTC(str) konvertiert einen spezifischen String
Print "Local Now:      " & date.Local()
Print "UTC Now:        " & date.UTC()
Print "Convert Local to UTC: " & date.ToUTC(date.Local())
Print ""

Print "=== DONE ==="
