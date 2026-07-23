Dim path
path = "testfolder/"
' Zertifikat laden
cert.Load("mycert.pem")

' Subject und Issuer auslesen
cert.Subject("mycert.pem")
cert.Issuer("mycert.pem")

' Gültigkeitsdaten
cert.ValidFrom("mycert.pem")
cert.ValidTo("mycert.pem")

' Prüfen, ob aktuell gültig
cert.IsValid("mycert.pem")

' Fingerprint berechnen
cert.Fingerprint("mycert.pem", "sha256")

' Privaten Schlüssel erzeugen
cert.GenerateKey("private.key", "rsa", 2048)
cert.GenerateKey("private.key", "ecdsa", 256)

' CSR erstellen
cert.CreateCSR("CN=example.com,O=MyCompany,C=DE", "private.key", testfolder & "request.csr")
cert.CreateCSR("CN=example.com,O=MyCompany,C=DE", "private.key", testfolder & "request.csr", "DNS:www.example.com,DNS:mail.example.com")

' Zertifikat exportieren
cert.ExportPEM("mycert.pem", "exported.pem")
cert.ExportDER("mycert.pem", "exported.der")
cert.ExportPFX("mycert.pem", "private.key", "exported.pfx", "password123")
cert.ExportPKCS7(["cert1.pem", "cert2.pem"], "bundle.p7b")

' Öffentlichen Schlüssel auslesen
cert.GetPublicKey("mycert.pem")
