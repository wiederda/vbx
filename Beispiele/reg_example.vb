#use reg

' ============================
' Reg Beispiele für VBmini
' ============================

Print "=== Registry Write ==="
reg.Write("HKCU", "Software\\VBMiniTest", "Version", "1.0")
reg.Write("HKCU", "Software\\VBMiniTest", "Path", "C:\\Test")
Print "Werte geschrieben"
Print ""

Print "=== reg.Exists (Key) ==="
If reg.Exists("HKCU", "Software\\VBMiniTest") Then
    Print "Key existiert"
Else
    Print "Key existiert NICHT"
End If
Print ""

Print "=== reg.Exists (Values) ==="
Print "Version exists: " & reg.Exists("HKCU", "Software\\VBMiniTest", "Version")
Print "Path exists: " & reg.Exists("HKCU", "Software\\VBMiniTest", "Path")
Print "Foo exists: " & reg.Exists("HKCU", "Software\\VBMiniTest", "Foo")
Print ""

Print "=== reg.Read ==="
Print "Version: " & reg.Read("HKCU", "Software\\VBMiniTest", "Version")
Print "Path: " & reg.Read("HKCU", "Software\\VBMiniTest", "Path")
Print ""

Print "=== reg.Delete (Value) ==="
reg.Delete("HKCU", "Software\\VBMiniTest", "Path")
Print "Path exists after delete: " & reg.Exists("HKCU", "Software\\VBMiniTest", "Path")
Print ""

Print "=== reg.Delete (Key) ==="
reg.Delete("HKCU", "Software\\VBMiniTest")
Print "Key exists after delete: " & reg.Exists("HKCU", "Software\\VBMiniTest")
