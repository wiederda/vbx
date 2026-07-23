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
  - `token`: Optional. Auth-Token. Unterstützte Präfixe: `"gl:"` (GitLab), `"gh:"` (GitHub), `"b:"` (Bearer erzwungen). Ohne Präfix: Bearer.
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

## net.PostForm(url, token, title, message)
- **Konkret:**
  Sendet eine Multipart-POST-Anfrage mit `title` und `message` als Formularfelder.
  Geeignet für Push-Dienste wie Gotify.
- **Parameter:**
  - `url`: Basis-URL des Dienstes.
  - `token`: Auth-Token (wird als URL-Parameter angehängt).
  - `title`: Nachrichtentitel.
  - `message`: Nachrichtentext.
- **Rückgabe:**
  `StrVal` (Response-Body), `ErrorVal` bei Fehler oder HTTP 4xx/5xx.

---

## net.Download(url [, path, token])
- **Konkret:**
  Lädt eine Datei herunter.
  Ohne Pfadangabe wird der Dateiname aus der URL abgeleitet.
  Kein Timeout – geeignet für große Dateien.
- **Parameter:**
  - `url`: Download-URL.
  - `path`: Optional. Zielpfad inkl. Dateiname.
  - `token`: Optional. Auth-Token (gleiche Präfixe wie `net.Get`).
- **Rückgabe:**
  `BoolVal`

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