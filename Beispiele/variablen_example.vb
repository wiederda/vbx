Print "=== VARIABLEN, ARRAYS & FUNKTIONEN TEST ==="

' Variablen mit Initialisierung
Dim a = 5, b = 10, c, d = true

if d = true Then
    Print "d = true"
End If    

Print "a=" & a
' 1D-Array mit Initialisierung
dim arr=array.Create("3","5","6","7","8")

' Sub und Function Definitionen
Sub SagHallo(name)
    Print "Hallo, " & name
End Sub

Function Summe(x, y)
    'Summe = math.Add(x,y)
    Return math.Add(x,y)
End Function

' Sub-Aufruf
SagHallo("Testuser")

' Function-Aufruf
Dim result
result = Summe(a, b)
Print "Summe von a und b: " & result

' If/Else Test
If a < b Then
    Print "a ist kleiner als b"
Else
    Print "a ist nicht kleiner als b"
End If

' For-Schleife
For i = 1 To 3 Step 1
    Print "Zähler: " & i
Next

' 1D-Array Zugriff
For i = 0 To array.UBound(arr)
    Print "arr(" & i & ") = " & array.GetIndex(arr,i)
Next

Dim counter = 1
While counter <= 5
    Print "Durchlauf Nummer: " & counter
    counter = counter + 1
End While
Print "Schleife beendet!"

Dim i = 1
Do While i <= 3
    Print "Do-Kopf: " & i
    i = i + 1
Loop
Print "Test 1 fertig"

Sub TestExit()
    Dim i
    For i = 1 To 5
        Print "For i=" & i
        If i = 3 Then 
        Exit For
        End If    
    Next
    Print "Nach For"

    Dim w = 1
    While w <= 5
        Print "While w=" & w
        If w = 2 Then 
        Exit While
        End If    
        w = w + 1
End While     
    Print "Nach While"

 Dim d = 1
    Do While d <= 5
        Print "Do d=" & d
        If d = 4 Then 
        Exit Do
        End If    
        d = d + 1
    Loop
    Print "Nach Do"
End Sub



Function TestExitFunc()
    Dim x = 1
    While x <= 5
        If x = 3 Then
            Return x
        Exit Function
        End If    
        x = x + 1
    End While
    Return x
End Function

Dim x = 1

SELECT CASE x
    CASE 1, 2, 3
        Print "CASE 1, 2, 3"
    CASE 4
        Print "Body"
    CASE ELSE
        Print "CASE ELSE"
END SELECT

' vbCrLf-Test
Print "Mit vbCrLf:" & vbCrLf() & "Neue Zeile"

Print "--- Starte Exit-Tests ---"
TestExit()
Print "Ergebnis TestExitFunc: " & TestExitFunc()    