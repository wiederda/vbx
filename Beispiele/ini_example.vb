#use ini
' ------------------------------
' INI Testskript
' ------------------------------

Dim res,sections,dbKeys,cacheKeys

Print "=== file.Create ==="
file.Create("/home/vbmini/test.ini")
Print "test.ini angelegt"
Print ""

Print "=== ini.Load ==="
res = ini.Load("/home/vbmini/test.ini")
If res <> "" Then
    Print "Fehler beim Laden: " & res
Else
    Print "INI geladen"
End If
Print ""

Print "=== ini.Set (auto-save, default) ==="
ini.Set("Database", "Host", "localhost")
ini.Set("Database", "Port", "5432")
ini.Set("Database", "User", "admin")
ini.Set("App", "Theme", "dark")
Print "Werte gesetzt (auto-save aktiv)"
Print ""

Print "=== ini.Set (kein sofortiges Speichern) ==="
ini.Set("Cache", "Enabled", "true", false)
ini.Set("Cache", "TTL", "300", false)
Print "Cache-Werte gesetzt (noch nicht gespeichert)"
Print ""

Print "=== ini.Get ==="
Print "Database.Host = " & ini.Get("Database", "Host")
Print "Database.Port = " & ini.Get("Database", "Port")
Print "Database.User = " & ini.Get("Database", "User")
Print "App.Theme     = " & ini.Get("App", "Theme")
Print "Nicht existierend (Default): " & ini.Get("App", "DoesNotExist", "default")
Print ""

Print "=== ini.Exists ==="
Print "Database.Host exists: " & ini.Exists("Database", "Host")
Print "App.DoesNotExist exists: " & ini.Exists("App", "DoesNotExist")
Print ""

Print "=== ini.Sections ==="
sections = ini.Sections()
For i = 0 To array.UBound(sections)
    Print "Section: " & sections(i)
Next
Print ""

Print "=== ini.Keys(Database) ==="
dbKeys = ini.Keys("Database")
For i = 0 To array.UBound(dbKeys)
    Print "Database Key: " & dbKeys(i)
Next
Print ""

Print "=== ini.Keys(Cache) ==="
cacheKeys = ini.Keys("Cache")
For i = 0 To array.UBound(cacheKeys)
    Print "Cache Key: " & cacheKeys(i)
Next
Print ""

Print "=== ini.Delete ==="
ini.Delete("Database", "Host")
Print "Database.User gelöscht"
Print ""

Print "=== ini.Delete ==="
ini.Delete("Database")
Print "Database gelöscht"
Print ""

Print "=== Test abgeschlossen ==="

