#use json
' ----------------------------------------
' JSON Datei laden
' ----------------------------------------
Dim path = ""

Dim res
res = json.Load(path)
Print res

' ----------------------------------------
' Werte lesen
' ----------------------------------------

Dim name
Dim age

name = json.Get("user.name")
age  = json.Get("user.age")

Print "Name: " & name
Print "Alter: " & age


' ----------------------------------------
' Wert ändern (als ZAHL!)
' ----------------------------------------

json.Set("user.age", 36)


' ----------------------------------------
' Neues Feld hinzufügen
' ----------------------------------------

json.Set("user.city", "Berlin")

' ----------------------------------------
' Optionales Feld entfernen
' ----------------------------------------

If json.Exists("settings.theme") = 1 Then
    json.Delete("settings.theme")
    Print "Theme entfernt"
Else
    Print "Theme nicht gesetzt"
End If




' ----------------------------------------
' Speichern
' ----------------------------------------

res = json.Save()

If Left(res, 6) = "error:" Then
    Print "Fehler beim Speichern: " & res
Else
    Print "Speichern erfolgreich"
End If


' ----------------------------------------
' Existenz prüfen
' ----------------------------------------

Dim n
n = json.Exists("user.name")

If n = 1 Then
    Print "user.name existiert"
Else
    Print "user.name existiert NICHT"
End If



