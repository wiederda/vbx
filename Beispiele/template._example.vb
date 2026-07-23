#use json, template

' ----------------------------------------
' Beispiel 1: E-Mail-Text generieren
' ----------------------------------------

Dim data = json.FromJSON("{""name"":""Max"",""server"":""srv01"",""fehler"":""Dienst nicht erreichbar""}")

Dim tmpl = "Hallo {{.name}}," & vbNewLine() & _
           "" & vbNewLine() & _
           "auf Server {{.server}} wurde folgender Fehler erkannt:" & vbNewLine() & _
           "{{.fehler}}" & vbNewLine() & _
           "" & vbNewLine() & _
           "Bitte prüfen."

Print template.Render(tmpl, data)


' ----------------------------------------
' Beispiel 2: Config-Datei generieren
' nginx.conf.tmpl Inhalt:
'   server {
'       listen {{.port}};
'       server_name {{.domain}};
'       root {{.webroot}};
'   }
' ----------------------------------------

Dim cfg = json.FromJSON("{""port"":""443"",""domain"":""example.com"",""webroot"":""/var/www/html""}")

template.RenderToFile("nginx.conf.tmpl", "/etc/nginx/sites-available/example.conf", cfg)
Print "Konfiguration geschrieben."


' ----------------------------------------
' Beispiel 3: Liste iterieren
' ----------------------------------------

Dim data2 = json.FromJSON("{""user"":""admin"",""rechte"":[""lesen"",""schreiben"",""loeschen""]}")

Dim tmpl2 = "Benutzer: {{.user}}" & vbNewLine() & _
            "Rechte:" & vbNewLine() & _
            "{{range .rechte}}- {{.}}" & vbNewLine() & "{{end}}"

Print template.Render(tmpl2, data2)


' ----------------------------------------
' Beispiel 4: Bedingte Ausgabe
' ----------------------------------------

Dim srv = json.FromJSON("{""name"":""srv02"",""online"":true}")

Dim tmpl3 = "Server {{.name}}: {{if .online}}✔ erreichbar{{else}}✘ nicht erreichbar{{end}}"

Print template.Render(tmpl3, srv)

' ----------------------------------------
' Beispiel 5: Batch-Report
' ----------------------------------------
'#use json, template
Dim data = json.FromJSON("{" & _
    """datum"":""15.07.2026""," & _
    """user"":""admin""," & _
    """server"":""srv01""," & _
    """ok"":true," & _
    """anzahl"":42," & _
    """dateien"":[""backup.zip"",""log.txt"",""config.ini""]" & _
"}")

template.RenderToFile("report.tmpl", "bericht_heute.txt", data)
Print "Bericht erstellt."