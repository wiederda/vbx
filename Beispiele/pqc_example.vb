#use pqc, crypt

' ============================
' PQC Beispiele für VBmini
' ============================

Cls()
Dim meinGeheimnis = "Verschlüsselt mit Post-Quanten-Logik 2026"

' =========================================================================
' SCHRITT 1: DIE SCHLÜSSELBOX (Vorbereitung beim Empfänger)
' =========================================================================
' GenerateKeyPair() erzeugt ein ML-KEM-768 Schlüsselpaar.
' Teil (1) ist der Public Key: Den darf jeder sehen (wie eine offene Tresortür).
' Teil (2) ist der Private Key: Den MUSST du sicher verstecken (dein Generalschlüssel).
Dim keys = pqc.GenerateKeyPair()
Dim empfaengerPubKey  = keys(1)
Dim empfaengerPrivKey = keys(2)


' =========================================================================
' SCHRITT 2: ENCAPSULATE (Absender erstellt den geheimen Schlüssel)
' =========================================================================
' Wir nehmen den Public Key des Empfängers und "werfen" einen geheimen 
' AES-Schlüssel hinein. 
' Das Ergebnis (encap) enthält zwei wichtige Dinge:
' (1) Ciphertext: Der "verschlossene Tresor" (muss zum Empfänger geschickt werden).
' (2) SharedSecret: Das "rohe Gold" (unser AES-Passwort für die Verschlüsselung).
Dim encap = pqc.Encapsulate(empfaengerPubKey)
Dim pqcPaket = encap(1)     ' Das Paket, das den AES-Key schützt
Dim aesPasswort = encap(2)  ' Das 32-Byte Passwort für AES-256


' =========================================================================
' SCHRITT 3: AES-VERSCHLÜSSELUNG (Die eigentliche Daten-Arbeit)
' =========================================================================
' PQC ist zu langsam für große Texte, daher nutzen wir das Passwort aus 
' Schritt 2 für das schnelle AES-GCM Verfahren.
Dim verschluesselt = crypt.AESEncrypt(meinGeheimnis, aesPasswort)
Dim cipherText = verschluesselt(1) ' Der Datensalat

Print "Status: Nachricht ist jetzt quantensicher verpackt."
Print "---------------------------------------------------"


' =========================================================================
' SCHRITT 4: DECAPSULATE (Empfänger öffnet den Tresor)
' =========================================================================
' Der Empfänger nutzt seinen Private Key, um das pqcPaket zu öffnen.
' Nur er bekommt dadurch das EXAKT GLEICHE aesPasswort wie der Absender.
Dim decap = pqc.Decapsulate(pqcPaket, empfaengerPrivKey)
Dim empfaengerKey = decap(1)


' =========================================================================
' SCHRITT 5: FINALE (Nachricht wieder lesbar machen)
' =========================================================================
' Jetzt kann der Empfänger mit dem zurückgewonnenen Key den AES-Salat lösen.
Dim ergebnis = crypt.AESDecrypt(cipherText, empfaengerKey)

If ergebnis(0) Then
    Print "ERFOLG: " & ergebnis(1)
Else
    Print "FEHLER: Schlüssel passt nicht oder Daten wurden manipuliert!"
End If