# 🌐 net.* – Netzwerkfunktionen

Dient zur Kommunikation über HTTP, TCP und DNS sowie zur Abfrage lokaler und externer Netzwerkinformationen.

---

## net.Ping(host [, port])
- **Konkret:**
  Prüft die TCP-Erreichbarkeit eines Hosts.
  Timeout: 2 Sekunden.
- **Parameter:**
  - `host`: Hostname oder IP.
  - `port`: Optional. Port (Standard: `443`).
- **Rückgabe:**
  `BoolVal`

---

## net.Connect(host, port)
- **Konkret:**
  Prüft eine TCP-Verbindung zu einem Host und Port.
  Timeout: 3 Sekunden.
- **Parameter:**
  - `host`: Hostname oder IP.
  - `port`: Port als String oder Zahl.
- **Rückgabe:**
  `BoolVal`

---

## net.Available(url)
- **Konkret:**
  Prüft ob eine URL erreichbar ist (HTTP-Statuscode 200–299).
  Versucht zuerst HEAD, dann GET als Fallback.
  Timeout: 5 Sekunden.
- **Parameter:**
  - `url`: Vollständige URL.
- **Rückgabe:**
  `BoolVal`

---

## net.Get(url [, token])
- **Konkret:**
  Führt eine HTTP-GET-Anfrage aus.
  Speichert den HTTP-Statuscode intern (abrufbar via `net.LastStatus`).
- **Parameter:**
  - `url`: Ziel-URL.
  - `token`: Optional. Auth-Token, siehe [Auth-Token-Präfixe](#auth-token-präfixe).
- **Rückgabe:**
  `StrVal` (Response-Body), leerer String bei Fehler.

---

## net.Post(url, data [, token])
- **Konkret:**
  Führt eine HTTP-POST-Anfrage aus.
  Erkennt JSON automatisch (`{` oder `[` am Anfang) und setzt `Content-Type: application/json`.
  Sonst: `application/x-www-form-urlencoded`.
- **Parameter:**
  - `url`: Ziel-URL.
  - `data`: Request-Body als String.
  - `token`: Optional. Auth-Token (gleiche Präfixe wie `net.Get`).
- **Rückgabe:**
  `StrVal` (Response-Body), leerer String bei Fehler.

---

## net.PostForm(url, token, title, message [, link])
- **Konkret:**
  Sendet eine JSON-POST-Anfrage mit `title` und `message` als Felder.
  Geeignet für Push-Dienste wie Gotify.
  Optional kann ein `link` übergeben werden: Er wird als Markdown-Link (`[link](link)`) an `message` angehängt und zusätzlich über `extras` gesetzt – `client::display` (Content-Type `text/markdown`, damit der Link in der Nachrichtenliste als echter Hyperlink gerendert wird) sowie `client::notification` (macht die Notification selbst anklickbar, öffnet `link` beim Antippen).
  Achtung: `token` wird hier direkt als URL-Parameter angehängt – **keine** Präfix-Logik wie bei `net.Get`/`net.Download`.
- **Parameter:**
  - `url`: Basis-URL des Dienstes.
  - `token`: Auth-Token (wird als URL-Parameter angehängt).
  - `title`: Nachrichtentitel.
  - `message`: Nachrichtentext.
  - `link`: Optional. Link, der an die Nachricht angehängt und über `extras` klickbar gemacht wird (siehe oben).
- **Rückgabe:**
  `StrVal` (Response-Body), `ErrorVal` bei Fehler oder HTTP 4xx/5xx.

```vb
' Ohne Link
net.PostForm(url, token, "Titel", "Nachricht")

' Mit klickbarem Link
net.PostForm(url, token, "Titel", "Nachricht", "https://example.com")
```

---

## net.Download(url [, path, token])
- **Konkret:**
  Lädt eine Datei per HTTP-GET herunter und speichert sie lokal.
  Ohne Pfadangabe wird der Dateiname aus dem letzten Segment der URL abgeleitet (Fallback: `"downloaded_file"`).
  Kein Timeout (`Timeout: 0`) – geeignet für große Dateien, ein hängender Download blockiert aber unbegrenzt.
  Es wird kein Verzeichnis automatisch angelegt; das Zielverzeichnis muss existieren. Eine bereits vorhandene Datei am Zielpfad wird ohne Rückfrage überschrieben.
  Der `User-Agent`-Header ist fest auf `"VBX/1.0"` gesetzt.
- **Parameter:**
  - `url`: Download-URL.
  - `path`: Optional. Zielpfad inkl. Dateiname. Relative Pfade werden über `absPathVal` aufgelöst.
  - `token`: Optional. Auth-Token, siehe [Auth-Token-Präfixe](#auth-token-präfixe).
- **Rückgabe:**
  `ArrVal` `[Bool, String]`
  Erfolg: `[True, ""]`
  Fehler: `[False, Fehlermeldung]` – u. a. bei fehlender/leerer URL, ungültigem Zielpfad, HTTP-Statuscode ≠ 200, oder Lese-/Schreibfehlern.

---

## Auth-Token-Präfixe

Gilt für den `token`-Parameter von `net.Get` und `net.Download` (intern: `setAuthHeader`). Die Präfixe sind case-sensitive und werden per `strings.HasPrefix` erkannt, danach wird der Präfix entfernt und der Rest als Header-Wert verwendet.

| Präfix | Header | Beispiel | Verwendung |
|--------|--------|----------|------------|
| `gl:`  | `PRIVATE-TOKEN: <wert>` | `"gl:glpat-xxx"` | GitLab Personal Access Token |
| `gh:`  | `Authorization: Bearer <wert>` | `"gh:ghp_xxx"` | GitHub Token |
| `b:`   | `Authorization: Bearer <wert>` | `"b:mein-token"` | Bearer-Token erzwingen |
| `a:`   | `Authorization: <wert>` (unverändert) | `"a:Basic dXNlcjpwYXNz"` | Kompletter, selbst zusammengesetzter Authorization-Header |
| *(kein Präfix)* | `Authorization: Bearer <wert>` | `"mein-token"` | Standardfall: Bearer |

Ist `token` leer (nach Trim), wird **kein** Auth-Header gesetzt.

```vb
result = net.Get(url, "gh:" & githubToken)
result = net.Download(url, path, "gl:" & gitlabToken)
result = net.Download(url, path, "a:" & "Basic " & base64Cred)
```

---

## net.Status([url])
- **Konkret:**
  Ohne Parameter: gibt den HTTP-Statuscode der letzten `Get`/`Post`-Anfrage zurück.
  Mit URL: führt einen HEAD-Request durch und gibt dessen Statuscode zurück.
- **Parameter:**
  - `url`: Optional. URL für aktive Prüfung.
- **Rückgabe:**
  `NumVal` (HTTP-Statuscode, `0` bei Fehler oder noch keiner Anfrage)

---

## net.LastStatus()
- **Konkret:**
  Gibt den HTTP-Statuscode der letzten `Get`/`Post`/`PostForm`-Anfrage zurück.
- **Rückgabe:**
  `NumVal`

---

## net.Resolve(host, type [, dnsServer])
- **Konkret:**
  Löst einen Hostnamen in eine einzelne IPv4- oder IPv6-Adresse auf.
  Optional kann ein spezifischer DNS-Server für die Anfrage verwendet werden. Die Auflösung ist mit einem Timeout von 5 Sekunden abgesichert.
- **Parameter:**
  - `host`: Hostname.
  - `type`: Adresstyp – `"IP4"` oder `"IP6"`. Bei anderen/leeren Werten wird die erste gefundene Adresse zurückgegeben (unabhängig vom Typ).
  - `dnsServer`: Optional. DNS-Server für die Anfrage (z. B. `"8.8.8.8"`).
- **Rückgabe:**
  `StrVal` (aufgelöste Adresse). Leerer String wenn keine Adresse zum gewünschten Typ gefunden wurde. `ErrorVal` bei fehlenden Pflichtargumenten oder fehlgeschlagener Auflösung.

---

## net.ResolveAll(host)
- **Konkret:**
  Löst einen Hostnamen in alle verfügbaren IPv4- und IPv6-Adressen auf.
- **Parameter:**
  - `host`: Hostname.
- **Rückgabe:**
  `ArrVal`
  Format: `[1, 4, "ip", 6, "ip", ...]`
  Index 0 = `1` bei Erfolg, `0` bei Fehler. Danach abwechselnd IP-Version (`4` oder `6`) und IP-String.

---

## net.CanResolve(host [, dnsServer])
- **Konkret:**
  Prüft ob ein Hostname aufgelöst werden kann.
  Protokoll-Präfixe und Pfade werden automatisch entfernt.
  Optional kann ein spezifischer DNS-Server angegeben werden.
- **Parameter:**
  - `host`: Hostname oder URL.
  - `dnsServer`: Optional. DNS-Server (z. B. `"8.8.8.8"` oder `"8.8.8.8:53"`).
- **Rückgabe:**
  `BoolVal`

---

## net.LocalIPs()
- **Konkret:**
  Gibt alle aktiven lokalen IPv4-Adressen zurück.
  Loopback, Link-Local und nicht-spezifizierte Adressen werden ignoriert.
- **Rückgabe:**
  `ArrVal`
  Array von `StrVal`-Einträgen.

---

## net.MACs()
- **Konkret:**
  Listet die MAC-Adressen aller aktiven Netzwerkinterfaces auf.
  Null-Adressen werden ignoriert.
- **Rückgabe:**
  `ArrVal`
  Array von `StrVal`-Einträgen im Format `"xx:xx:xx:xx:xx:xx"`.

---

## net.PublicIP()
- **Konkret:**
  Ermittelt die externe (öffentliche) IP-Adresse über externe Provider.
  Fragt nacheinander `ipify.org`, `ifconfig.me` und `checkip.amazonaws.com` ab.
  Timeout: 5 Sekunden je Provider.
- **Rückgabe:**
  `StrVal`
  Leerer String wenn kein Provider erreichbar ist.

---

## net.IPIsValid(ip)

- **Konkret:** Prüft, ob ein String eine gültige IPv4- oder IPv6-Adresse ist.
- **Parameter:** `ip` – zu prüfender String
- **Rückgabe:** `True` wenn gültig, sonst `False`

```vb
If net.IPIsValid("192.168.1.1") Then Print "OK"
If net.IPIsValid(net.PublicIP()) Then Print "Externe IP ist gültig"
```

---

## net.Html(text)
- **Konkret:**
  Ersetzt HTML-Sonderzeichen und deutsche Umlaute durch HTML-Entities.
  Beispiele: `<` → `&lt;`, `ä` → `&auml;`, `ß` → `&szlig;`.
- **Parameter:**
  - `text`: Zu kodierender String.
- **Rückgabe:**
  `StrVal`