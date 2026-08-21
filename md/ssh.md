# ssh.*

Funktionen für Remote-SSH-Zugriff: einzelne Befehlsausführung, Mehrfach-Exec über eine
offene Verbindung, sowie ein spezialisierter Reboot-Befehl. Authentifizierung ausschließlich
per SSH-Key (kein Klartext-Passwort in Skripten).

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
- `knownHostsPath` *(optional)* – Pfad zu einer known_hosts-Datei. Ohne Angabe wird der
  Host-Key nicht geprüft (`InsecureIgnoreHostKey`). Mit Angabe: Trust-on-First-Use – ein
  bislang unbekannter Host wird beim ersten Connect automatisch eingetragen, eine spätere
  Änderung des Host-Keys (möglicher Angriff oder Server-Neuaufsetzung ohne neue known_hosts)
  wird abgelehnt.
- `port` *(optional, Default 22)* – SSH-Port

**Rückgabe:** `BoolVal(true)` bei erfolgreichem Verbindungsaufbau, sonst `ErrorVal` mit
Fehlermeldung.

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
- `printOutput` *(optional, Default True)* – bei `False` keine Konsolenausgabe, nur Rückgabe-Map
- `knownHostsPath` *(optional)* – wie bei `ssh.Connect`
- `port` *(optional, Default 22)* – SSH-Port

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
- `delay` *(optional, Default 30)* – Sekunden bis zum Reboot
- `knownHostsPath` *(optional)* – wie bei `ssh.Connect`
- `port` *(optional, Default 22)*

**Rückgabe:** `BoolVal(true)` bei erfolgreich ausgelöstem Reboot, sonst `ErrorVal`.


## Sicherheitshinweise

- Ausschließlich Key-Auth, kein Passwort-Parameter in der gesamten `ssh.*`-Bibliothek.
- `knownHostsPath` ist bei allen Funktionen optional und schützt vor Man-in-the-Middle
  während des Verbindungsaufbaus. Ohne Angabe wird jedem Host vertraut, der behauptet,
  die angegebene Adresse zu sein – in einem kontrollierten internen Netz ein akzeptables
  Risiko, in Netzen mit geringerem Vertrauen sollte `knownHostsPath` gesetzt werden.
- `ssh.Exec`/`ssh.ExecOnce` führen beliebige Befehle aus. Wer Skript und privaten Key hat,
  hat vollen Shell-Zugriff auf jeden Server, für den der zugehörige Public Key hinterlegt ist.

## Passphrase-geschützte Keys werden NICHT unterstützt

Alle `ssh.*`-Funktionen erwarten einen **unverschlüsselten** privaten Key (`keyPath`). Ist
der Key mit einer Passphrase geschützt, schlägt `ssh.ParsePrivateKey` fehl und die Funktion
gibt einen entsprechenden Fehler zurück (sinngemäß "this private key is passphrase
protected").

**Das ist bewusst so, kein Versehen:** Für den Automatisierungs-Anwendungsfall (Cron,
CLI-Shortcuts, Reboot-Trigger) müsste die Passphrase ohnehin irgendwo im Skript/Aufruf
hinterlegt werden – damit wäre der Schutz, den eine Passphrase bieten soll, wieder aufgehoben:
Wer Skript und Passphrase hat, hat denselben Zugriff wie mit einem ungeschützten Key. Der
einzige verbleibende Vorteil einer Passphrase wäre Schutz gegen den Fall, dass *nur* die
Key-Datei entwendet wird (z.B. aus einem Backup) – ohne das Skript.

**Empfohlene Alternative für Automatisierung:** ein dedizierter, ungeschützter Key nur für
diesen einen Zweck (z.B. nur Reboot), auf dem Zielserver in `authorized_keys` mit einer
`command=`-Einschränkung versehen. Das begrenzt den Schaden bei Diebstahl der Key-Datei
deutlich wirksamer als eine Passphrase, die im Skript ohnehin wieder auftauchen müsste:

```
command="/sbin/shutdown -r now",no-port-forwarding,no-X11-forwarding,no-agent-forwarding ssh-ed25519 AAAA... automation-reboot-key
```