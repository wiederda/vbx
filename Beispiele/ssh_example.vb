#use ssh

' ============================
' SSH Beispiele für VBX
' ============================

Dim keyBasePath, result

Print "=== ssh.GenerateSSHKey ==="
keyBasePath = ssh.GenerateSSHKey("/home/vbx/.ssh", "ed25519")
Print "Schlüsselpaar erstellt: " & keyBasePath
Print "(Public Key " & keyBasePath & ".pub muss vorher/danach in authorized_hosts")
Print " auf dem Zielserver eingetragen werden)"
Print ""

Print "=== ssh.Connect ==="
result = ssh.Connect("server1.local", "vbx", keyBasePath, "server1", "/home/vbx/.ssh/known_hosts")
Print "Verbindung 'server1' aufgebaut: " & result
Print ""

Print "=== ssh.Exec (mehrfach über dieselbe Verbindung) ==="
result = ssh.Exec("server1", "uptime")
Print "uptime -> " & result("stdout") & " (Exit " & result("exitCode") & ")"

result = ssh.Exec("server1", "df -h /")
Print "df -h / -> " & result("stdout") & " (Exit " & result("exitCode") & ")"
Print ""

Print "=== ssh.Close ==="
result = ssh.Close("server1")
Print "Verbindung 'server1' geschlossen: " & result
Print ""

Print "=== ssh.ExecOnce ==="
result = ssh.ExecOnce("server2.local", "vbx", keyBasePath, "hostname", true, "/home/vbx/.ssh/known_hosts")
Print "Ergebnis-Map: stdout=" & result("stdout") & ", exitCode=" & result("exitCode")
Print ""

Print "=== ssh.ExecOnce (ohne Konsolenausgabe) ==="
result = ssh.ExecOnce("server2.local", "vbx", keyBasePath, "whoami", false, "/home/vbx/.ssh/known_hosts")
Print "whoami (still) -> " & result("stdout")
Print ""

Print "=== ssh.RebootWithKey ==="
' Vorsicht: löst einen echten Reboot auf dem Zielsystem aus - hier auskommentiert
' result = ssh.RebootWithKey("server2.local", "vbx", keyBasePath, 30, "/home/vbx/.ssh/known_hosts")
' Print "Reboot ausgelöst: " & result
Print "(auskommentiert - löst einen echten Reboot aus)"
Print ""

Print "=== Test abgeschlossen ==="
