#use xml
Dim res

if file.Exists("/home/vbmini/Beispiele/test.xml")=1 Then
    xml.Load("/home/vbmini/Beispiele/test.xml")
else
    file.Create("/home/vbmini/Beispiele/test.xml")
endif    

' Werte auslesen
Dim name
Dim age
Dim city
name = xml.Get("root.user[0].name")
age = xml.Get("root.user[0].age")
city = xml.Get("root.user[0].city")    
Print "Name: " & name & ", Age: " & age & ", City: " & city 

' Wert ändern
' User 1 (Nina) - Index [0]
xml.Set("root.user[0].age", "25")
xml.Set("root.user[0].name", "Nina")
xml.Set("root.user[0].city", "Wuppertal")

' User 2 (Sandra) - Index [1]
xml.Set("root.user[1].age", "32")
xml.Set("root.user[1].name", "Sandra")
xml.Set("root.user[1].city", "Hamburg")
' Neues Element hinzufügen

xml.Set("root.settings.theme", "dark")    
xml.Save("/home/vbmini/Beispiele/test.xml")