#use rand

' ============================
' Rand Beispiele für VBX
' ============================

Print "=== Zufallstest ==="

' ---------------- Float 0..1 ----------------
Print "--- Rand.Float() ---"
For i = 1 To 5
    Print "Zufallszahl: " & rand.Float()
Next

' ---------------- Int 0..max-1 ----------------
Print "--- Rand.Int(10) ---"
For i = 1 To 5
    Print "0..9: " & rand.Int(10)
Next

' ---------------- Range min..max ----------------
Print "--- Rand.Range(1,6) ---"
For i = 1 To 5
    Print "Würfel: " & rand.Range(1, 6)
Next

' ---------------- Seed-Test ----------------
Print "--- Seed 12345 ---"
rand.Seed(12345)

For i = 1 To 5
    Print "Float: " & rand.Float()
Next

For i = 1 To 5
    Print "Würfel: " & rand.Range(1, 6)
Next

Print "--- Seed 12345 erneut (muss identisch sein) ---"
rand.Seed(12345)

For i = 1 To 5
    Print "Float: " & rand.Float()
Next

For i = 1 To 5
    Print "Würfel: " & rand.Range(1, 6)
Next
