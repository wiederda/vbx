Print "=== MATH MODULE TEST ==="

' ---------------- Basic Math ----------------
Print "5+3: " & 5 + 3                            ' 8
Print "10-4: " & 10 - 4                          ' 6
Print "6*7: " & 6 * 7                            ' 42
Print "20/4: " & 20 / 4                          ' 5
Print "Pow(2,3): " & math.Pow(2,3)               ' 8
Print "Sqrt(16): " & math.Sqrt(16)               ' 4
Print "Abs(-5): " & math.Abs(-5)                 ' 5
Print "Sign(-5): " & math.Sign(-5)               ' -1
Print "Exp(1): " & math.Exp(1)                   ' 2.718281828459045

' ---------------- Trig ----------------
Print "Sin(0.5): " & math.Sin(0.5)               ' 0.479425538604203
Print "Cos(0.5): " & math.Cos(0.5)               ' 0.8775825618903728
Print "Tan(0.5): " & math.Tan(0.5)               ' 0.5463024898437905
Print "Asin(0.5): " & math.Asin(0.5)             ' 0.5235987755982989
Print "Acos(0.5): " & math.Acos(0.5)             ' 1.0471975511965979
Print "Atan(1): " & math.Atan(1)                 ' 0.7853981633974483
Print "Atan2(1,1): " & math.Atan2(1,1)           ' 0.7853981633974483
Print "DegToRad(180): " & math.DegToRad(180)     ' 3.141592653589793
Print "RadToDeg(3.14159): " & math.RadToDeg(3.14159) ' 179.9998479605043

' ---------------- Round / Floor / Ceil ----------------
Print "Round(3.14159): " & math.Round(3.14159)           ' 3
Print "Round(3.14159,3): " & math.Round(3.14159,3)      ' 3.142
Print "Ceil(3.2): " & math.Ceil(3.2)                     ' 4
Print "Floor(3.8): " & math.Floor(3.8)                   ' 3
Print "Trunc(3.8): " & math.Trunc(3.8)                   ' 3

' ---------------- Min / Max / Clamp ----------------
Print "Min(5,10): " & math.Min(5,10)                     ' 5
Print "Max(5,10): " & math.Max(5,10)                     ' 10
Print "Clamp(15,0,10): " & math.Clamp(15,0,10)          ' 10

' ---------------- Percentage ----------------
Print "PercentOf(50,200): " & math.PercentOf(50,200)        ' 25
Print "Percent(50,36) : " & math.Percent(20,36)          ' 7.2  

Dim r = 6371
' Ergebnis ausgeben
Print "Der Erdumfang beträgt ca. " & math.Round(2 * math.Pi() * r,2) & " km"



