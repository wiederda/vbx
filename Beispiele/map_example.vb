#use map

' ============================
' Map Beispiele für VBX
' ============================

Dim user = map.Create()

' Werte setzen
map.Set(user, "name", "Alice")
map.Set(user, "age", 30)
map.Set(user, "city", "Lübeck")

' Werte lesen
Print "Name: " & map.Get(user, "name")
Print "Alter: " & map.Get(user, "age")

' Existenz prüfen
Print "Hat 'city': " & map.Has(user, "city")
Print "Hat 'email': " & map.Has(user, "email")

' Alias-Test
Dim copy = user
map.Set(copy, "role", "Admin")

Print "Rolle: " & map.Get(user, "role")

' Alle Schlüssel ausgeben
Print "--- Keys ---"

Dim keys = map.Keys(user)

For Each key In keys
    Print key & " = " & map.Get(user, key)
Next

' Eintrag entfernen
map.Remove(user, "age")

Print "--- Nach Remove ---"
Print "Hat 'age': " & map.Has(user, "age")
Print "Alter: " & map.Get(user, "age")