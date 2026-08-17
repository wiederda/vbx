
' ============================
' App Beispiele für VBmini
' ============================

Print "=== APP MODULE TEST ==="
Print ""
Print ""
Print "=== app.StartupPath ==="
Print app.StartupPath()
Print ""

Print "=== app.ExecutablePath ==="
Print app.ExecutablePath()
Print ""

Print "=== app.CurrentDirectory ==="
Print app.CurrentDirectory()
Print ""

Print "=== app.TempPath ==="
Print app.TempPath()
Print ""

Print "=== app.SpecialFolder ==="
Print "Desktop: " & app.SpecialFolder("Desktop")
Print "Documents: " & app.SpecialFolder("Documents")
Print "Pictures: " & app.SpecialFolder("Pictures")
Print "Music: " & app.SpecialFolder("Music")
Print "Videos: " & app.SpecialFolder("Videos")
Print "Downloads: " & app.SpecialFolder("Downloads")
Print "Temp: " & app.SpecialFolder("Temp")

' Windows-spezifisch, auf Linux/macOS evtl. leer:
Print "AppData: " & app.SpecialFolder("AppData")
Print "LocalAppData: " & app.SpecialFolder("LocalAppData")
Print "ProgramData: " & app.SpecialFolder("ProgramData")

