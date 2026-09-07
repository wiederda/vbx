#use fin

' ============================================================
' Testskript: fin.* Plugin
' Testet alle 10 Funktionen mit Normalfällen + Fehlerfällen
' ============================================================

Print "=== fin.* Plugin Test ==="
Print ""

' ------------------------------------------------------------
' Finanzfunktionen
' ------------------------------------------------------------

Print "--- Finanzfunktionen ---"

Print "Fv 0, 10, 100, 0       = " & fin.Fv(0, 10, 100, 0) & "   erwartet: -1000"
Print "Fv 0.05, 10, 100, 0   = " & fin.Fv(0.05, 10, 100, 0) & "   erwartet: ca. -1257.79"

Print "Pmt 0, 10, 1000        = " & fin.Pmt(0, 10, 1000) & "   erwartet: -100"
Print "Pmt 0.05, 10, 1000    = " & fin.Pmt(0.05, 10, 1000) & "   erwartet: ca. -129.50"

Print ""

' ------------------------------------------------------------
' Kapitalwert
' ------------------------------------------------------------

Print "--- Kapitalwert ---"

Print "Npv 0, {100,200,300}     = " & fin.Npv(0, {100, 200, 300}) & "   erwartet: 600"
Print "Npv 0.1, {100,200,300}   = " & fin.Npv(0.1, {100, 200, 300}) & "   erwartet: ca. 481.59"

Print ""

' ------------------------------------------------------------
' Interner Zinsfuß
' ------------------------------------------------------------

Print "--- Interner Zinsfuß ---"

Print "Irr {-100,110}          = " & fin.Irr({-100, 110}) & "   erwartet: 0.1"
Print "Irr {-100,60,60}        = " & fin.Irr({-100, 60, 60}) & "   erwartet: ca. 0.13066"
Print "Irr {-100,110}, 0.2     = " & fin.Irr({-100, 110}, 0.2) & "   erwartet: 0.1"

Print ""

' ------------------------------------------------------------
' Mathematik
' ------------------------------------------------------------

Print "--- Mathematik ---"

Print "Fact 0                  = " & fin.Fact(0) & "   erwartet: 1"
Print "Fact 5                  = " & fin.Fact(5) & "   erwartet: 120"
Print "Fact 10                 = " & fin.Fact(10) & "   erwartet: 3628800"

Print "Gamma 0.5               = " & fin.Gamma(0.5) & "   erwartet: ca. 1.77245"
Print "Gamma 5                 = " & fin.Gamma(5) & "   erwartet: 24"

Print "Log10 1000              = " & fin.Log10(1000) & "   erwartet: 3"
Print "Log10 100                = " & fin.Log10(100) & "   erwartet: 2"

Print "Log2 1024               = " & fin.Log2(1024) & "   erwartet: 10"
Print "Log2 256                = " & fin.Log2(256) & "   erwartet: 8"

Print "Hypot 3, 4              = " & fin.Hypot(3, 4) & "   erwartet: 5"
Print "Hypot 5, 12             = " & fin.Hypot(5, 12) & "   erwartet: 13"

Print "Remainder 10, 3         = " & fin.Remainder(10, 3) & "   erwartet: 1"
Print "Remainder 10, 4         = " & fin.Remainder(10, 4) & "   erwartet: 2"
Print "Remainder 7.5, 2       = " & fin.Remainder(7.5, 2) & "   erwartet: -0.5"

Print ""

' ------------------------------------------------------------
' Nachkommastellen / Sonderfälle
' ------------------------------------------------------------

Print "--- Nachkommastellen ---"

Print "Fv 0.1, 5, 50, 100      = " & fin.Fv(0.1, 5, 50, 100)
Print "Pmt 0.01, 12, 1000     = " & fin.Pmt(0.01, 12, 1000)
Print "Gamma 2.5               = " & fin.Gamma(2.5)
Print "Hypot 1.5, 2            = " & fin.Hypot(1.5, 2)

Print ""

' ------------------------------------------------------------
' Fehlerfälle
' ------------------------------------------------------------

Print "--- Fehlerfälle ---"

Dim result

result = fin.Fv("test", 10, 100)
Print "Fv(""test"", 10, 100): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

result = fin.Pmt("test", 10, 1000)
Print "Pmt(""test"", 10, 1000): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

result = fin.Npv("test", {100, 200, 300})
Print "Npv(""test"", {100, 200, 300}): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

result = fin.Irr({"test", 110})
Print "Irr({""test"", 110}): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

result = fin.Fact("test")
Print "Fact(""test""): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

result = fin.Fact(-1)
Print "Fact(-1): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

result = fin.Gamma("test")
Print "Gamma(""test""): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

result = fin.Log10("test")
Print "Log10(""test""): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

result = fin.Log2("test")
Print "Log2(""test""): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

result = fin.Hypot("test", 4)
Print "Hypot(""test"", 4): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

result = fin.Remainder("test", 3)
Print "Remainder(""test"", 3): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

Print ""