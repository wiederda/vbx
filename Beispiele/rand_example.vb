#use rand

Print "=== rand.* Plugin Test ==="
Print ""

Print "--- Float ---"

Print "rand.Float() = " & rand.Float()
Print "rand.Float() = " & rand.Float()
Print "rand.Float() = " & rand.Float()

Print ""

Print "--- Bool ---"

Print "rand.Bool() = " & rand.Bool()
Print "rand.Bool() = " & rand.Bool()
Print "rand.Bool() = " & rand.Bool()
Print "rand.Bool() = " & rand.Bool()
Print "rand.Bool() = " & rand.Bool()

Print ""

Print "--- Range ---"

Print "Range(1, 6):"

For i = 1 To 10
    Print "  " & rand.Range(1, 6)
Next

Print ""

Print "Range(10, 10) = " & rand.Range(10, 10) & "   erwartet: 10"

Print ""

Print "--- RangeFloat ---"

Print "RangeFloat(0, 1):"

For i = 1 To 5
    Print "  " & rand.RangeFloat(0, 1)
Next

Print ""

Print "RangeFloat(10, 20):"

For i = 1 To 5
    Print "  " & rand.RangeFloat(10, 20)
Next

Print ""

Print "RangeFloat(5, 5) = " & rand.RangeFloat(5, 5) & "   erwartet: 5"

Print ""

Print "--- Choice ---"

Dim values
values = {"Apfel", "Birne", "Banane", "Orange"}

For i = 1 To 10
    Print "Choice: " & rand.Choice(values)
Next

Print ""

Print "--- Seed-Test ---"

rand.Seed(12345)

Dim float1
Dim float2
Dim float3
Dim range1
Dim range2
Dim range3

float1 = rand.Float()
float2 = rand.Float()
float3 = rand.Float()

range1 = rand.Range(1, 6)
range2 = rand.Range(1, 6)
range3 = rand.Range(1, 6)

Print "Seed 12345:"
Print "  Float 1: " & float1
Print "  Float 2: " & float2
Print "  Float 3: " & float3
Print "  Range 1: " & range1
Print "  Range 2: " & range2
Print "  Range 3: " & range3

Print ""

rand.Seed(12345)

Dim float1b
Dim float2b
Dim float3b
Dim range1b
Dim range2b
Dim range3b

float1b = rand.Float()
float2b = rand.Float()
float3b = rand.Float()

range1b = rand.Range(1, 6)
range2b = rand.Range(1, 6)
range3b = rand.Range(1, 6)

Print "Seed 12345 erneut:"
Print "  Float 1: " & float1b
Print "  Float 2: " & float2b
Print "  Float 3: " & float3b
Print "  Range 1: " & range1b
Print "  Range 2: " & range2b
Print "  Range 3: " & range3b

Print ""

Print "--- Seed-Vergleich ---"

Print "Float 1 identisch: " & (float1 = float1b)
Print "Float 2 identisch: " & (float2 = float2b)
Print "Float 3 identisch: " & (float3 = float3b)

Print "Range 1 identisch: " & (range1 = range1b)
Print "Range 2 identisch: " & (range2 = range2b)
Print "Range 3 identisch: " & (range3 = range3b)

Print ""

Print "--- Seed ohne Argument ---"

rand.Seed()

Print "Float nach Seed(): " & rand.Float()

Print ""

Print "--- Fehlerfälle ---"

Dim result

result = rand.Choice()
Print "Choice(): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

result = rand.Choice("test")
Print "Choice(""test""): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

Dim emptyArray
emptyArray = {}

result = rand.Choice(emptyArray)
Print "Choice({}): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

result = rand.Range()
Print "Range(): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

result = rand.Range("test", 10)
Print "Range(""test"", 10): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

result = rand.Range(1, "test")
Print "Range(1, ""test""): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

result = rand.Range(10, 1)
Print "Range(10, 1): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

result = rand.RangeFloat()
Print "RangeFloat(): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

result = rand.RangeFloat("test", 10)
Print "RangeFloat(""test"", 10): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

result = rand.RangeFloat(1, "test")
Print "RangeFloat(1, ""test""): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

result = rand.RangeFloat(10, 1)
Print "RangeFloat(10, 1): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

result = rand.Seed("test")
Print "Seed(""test""): IsError = " & IsError(result)
Print "Fehler: " & ErrorText(result)

Print ""

Print "=== Test abgeschlossen ==="
