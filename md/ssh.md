# ssh.*

Funktionen für Remote-SSH-Zugriff: einzelne Befehlsausführung, Mehrfach-Exec über eine
offene Verbindung, sowie ein spezialisierter Reboot-Befehl. Authentifizierung ausschließlich
per SSH-Key (kein Klartext-Passwort in Skripten).

---

## GenerateSSHKey([outDir, algo, bits, pass])
- **Konkret:**
  Erstellt ein SSH-Schlüsselpaar und schreibt es auf die Festplatte.
  Private Key mit Rechten `0600`, Public Key mit `0644`.
  Bricht ab, wenn die Zieldateien bereits existieren.
- **Parameter:**
  - `outDir`: Optional. Zielverzeichnis (Standard: `~/.ssh`).
  - `algo`: Optional. `"rsa"` (Standard) oder `"ed25519"`.
  - `bits`: Optional. Schlüssellänge für RSA (Standard: 4096, Minimum wird erzwungen).
  - `pass`: Optional. Passphrase (aktuell nicht verwendet).
- **Rückgabe:**
  `StrVal`
  Basispfad des erstellten Schlüsselpaares (ohne `.pub`).

---

## ssh.Connect

Öffnet eine SSH-Verbindung per Key-Authentifizierung und speichert sie unter einem Alias
für nachfolgende `ssh.Exec`-Aufrufe.

**Konkret:**
Für Skripte gedacht, die mehrere Befehle nacheinander auf derselben Verbindung ausführen
wollen (Alias bleibt im Speicher, solange der VBX-Prozess läuft). Für Einzelbefehle bzw.
CLI-Shortcuts stattdessen `ssh.ExecOnce` verwenden – der Alias ist über getrennte
Prozessaufrufe hinweg nicht wiederverwendbar.

**Parameter:**
- `host` – Zielhost (ohne Port)
- `user` – SSH-User
- `keyPath` – Pfad zum privaten Key lokal (z.B. aus `GenerateSSHKey`)
- `alias` – Bezeichner, unter dem die Verbindung gespeichert wird (Groß-/Kleinschreibung wird ignoriert)
- `port` *(optional, Default 22)* – SSH-Port
- `knownHostsPath` *(optional)* – Pfad zu einer known_hosts-Datei. Ohne Angabe wird der
  Host-Key nicht geprüft (`InsecureIgnoreHostKey`). Mit Angabe: Trust-on-First-Use – ein
  bislang unbekannter Host wird beim ersten Connect automatisch eingetragen, eine spätere
  Änderung des Host-Keys (möglicher Angriff oder Server-Neuaufsetzung ohne neue known_hosts)
  wird abgelehnt.

---

## ssh.Exec

Führt einen beliebigen Shell-Befehl auf einer per `ssh.Connect` geöffneten Verbindung aus.

**Konkret:**
Öffnet pro Aufruf eine neue Session auf der bestehenden Verbindung – mehrere `ssh.Exec`-Aufrufe
auf denselben Alias sind beliebig oft möglich, ohne erneut zu authentifizieren.

**Parameter:**
- `alias` – zuvor per `ssh.Connect` angelegter Alias
- `cmd` – auszuführender Shell-Befehl

**Rückgabe:** Map mit:
- `stdout` – kombinierte Ausgabe von stdout und stderr (wie bei einer interaktiven Shell), whitespace-getrimmt
- `exitCode` – Exit-Code des Befehls (0 = Erfolg)

Bei Verbindungs- oder Session-Fehlern (nicht bei einem regulären Nicht-0-Exit-Code des
Befehls selbst): `ErrorVal`.

---

## ssh.Close

Schließt eine per `ssh.Connect` geöffnete Verbindung und entfernt den Alias.

**Konkret:**
Sollte am Ende jedes Skripts aufgerufen werden, das `ssh.Connect` genutzt hat, um die
Verbindung sauber zu beenden, statt sie bis zum Prozessende offen zu halten.

**Parameter:**
- `alias` – zu schließender Alias

**Rückgabe:** `BoolVal(true)` wenn eine Verbindung unter diesem Alias gefunden und
geschlossen wurde, `BoolVal(false)` wenn kein Alias mit diesem Namen existierte.

---

## ssh.ExecOnce

Verbindet, führt genau einen Befehl aus und schließt die Verbindung wieder – alles in
einem Aufruf.

**Konkret:**
Für Einzelaufrufe gedacht, insbesondere CLI-Shortcuts (`remoteexec`): ein Shortcut kann pro
Prozessaufruf nur eine Funktion mappen, daher lohnt sich der Connect/Exec/Close-Dreischritt
dort nicht – `ssh.ExecOnce` kapselt alles in einem Funktionsaufruf. Kein explizites `Close`
nötig, die Verbindung wird am Ende der Funktion in jedem Fall geschlossen.

Standardmäßig wird die Ausgabe zusätzlich direkt auf der Konsole ausgegeben (`stdout` +
`Exit-Code: N`) – praktisch für den Shortcut-Fall ohne eigenes `Print` im Skript. Innerhalb
eines Skripts, das die Rückgabe-Map selbst auswertet, sollte `printOutput` auf `False`
gesetzt werden, um doppelte bzw. unerwünschte Konsolenausgaben zu vermeiden.

**Parameter:**
- `host` – Zielhost (ohne Port)
- `user` – SSH-User
- `keyPath` – Pfad zum privaten Key lokal
- `cmd` – auszuführender Shell-Befehl
- `port` *(optional, Default 22)* – SSH-Port
- `knownHostsPath` *(optional)* – wie bei `ssh.Connect`
- `printOutput` *(optional, Default True)* – bei `False` keine Konsolenausgabe, nur Rückgabe-Map

**Rückgabe:** Map mit `stdout`, `exitCode` (siehe `ssh.Exec`).

---

## ssh.RebootWithKey

Löst per SSH-Key-Authentifizierung einen Reboot auf einem Zielsystem aus.

**Konkret:**
Nutzt `shutdown -r +N` statt eines direkten `reboot`, damit die SSH-Session sauber schließt,
bevor das System herunterfährt (ein direktes `reboot` kappt die Session mitten im Befehl und
liefert dann fälschlich einen Verbindungsfehler zurück, obwohl der Reboot funktioniert hat).
`delay=0` löst stattdessen einen sofortigen Reboot (`shutdown -r now`) aus – dort ist ein
Abbruch der Verbindung während des Befehls erwartetes Verhalten und wird nicht als Fehler
gewertet.

Der öffentliche Key (`.pub`) muss vorher in `~/.ssh/authorized_keys` auf dem Zielserver
eingetragen sein (siehe `GenerateSSHKey`).

**Parameter:**
- `host` – Zielhost (ohne Port)
- `user` – SSH-User
- `keyPath` – Pfad zum privaten Key lokal
- `port` *(optional, Default 22)*
- `delay` *(optional, Default 30)* – Sekunden bis zum Reboot
- `knownHostsPath` *(optional)* – wie bei `ssh.Connect`

**Rückgabe:** `BoolVal(true)` bei erfolgreich ausgelöstem Reboot, sonst `ErrorVal`.

---

## CLI-Shortcuts

```go
"remoteexec": {"ssh.ExecOnce", "host, user, keyPath, cmd, [port], [knownHostsPath], [printOutput]",
    "Führt per SSH-Key-Auth einen einzelnen Befehl aus und schließt die Verbindung wieder."},
"remoteboot": {"ssh.RebootWithKey", "host, user, keyPath, [port], [delay], [knownHostsPath]",
    "Löst per SSH-Key-Auth einen Reboot auf einem Zielsystem aus."},
```

`ssh.Connect`/`ssh.Exec`/`ssh.Close` haben bewusst keinen eigenen Shortcut-Eintrag: der
Alias ist nur innerhalb eines einzelnen laufenden VBX-Prozesses gültig und über getrennte
Shortcut-Aufrufe (= getrennte Prozesse) nicht wiederverwendbar.

## Sicherheitshinweise

- Ausschließlich Key-Auth, kein Passwort-Parameter in der gesamten `ssh.*`-Bibliothek.
- `knownHostsPath` ist bei allen Funktionen optional und schützt vor Man-in-the-Middle
  während des Verbindungsaufbaus. Ohne Angabe wird jedem Host vertraut, der behauptet,
  die angegebene Adresse zu sein – in einem kontrollierten internen Netz ein akzeptables
  Risiko, in Netzen mit geringerem Vertrauen sollte `knownHostsPath` gesetzt werden.
- `ssh.Exec`/`ssh.ExecOnce` führen beliebige Befehle aus. Wer Skript und privaten Key hat,
  hat vollen Shell-Zugriff auf jeden Server, für den der zugehörige Public Key hinterlegt ist.