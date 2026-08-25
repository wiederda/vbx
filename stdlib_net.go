// ------------------------
// stdlib_net.go
// ------------------------

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ------------------------------------------------------------
// net.Get / net.Post / net.Download
//
// Alle drei um Retry-with-Backoff erweitert:
//   - Neue optionale Parameter: retries, baseDelayMs, maxDelayMs
//   - Backoff: exponentiell (baseDelayMs, 2x, 4x, 8x, ... gedeckelt bei maxDelayMs)
//   - Retry-Trigger: Netzwerkfehler, 5xx, 429 (Rate-Limit)
//   - Andere 4xx-Fehler (400, 401, 403, 404, ...) werden NICHT wiederholt,
//     da ein erneuter Versuch am Ergebnis nichts ändern würde.
//
// BREAKING CHANGE (bewusst gewollt): Rückgabe bei Fehler ist jetzt
// durchgehend ErrorVal (per IsError()/ErrorText() prüfbar) statt:
//   - Get:      vorher stiller Rückgabewert "" bei Fehler
//   - Post:     vorher roher String "HTTP ERROR: ..." (kein ErrorVal)
//   - Download: vorher [bool, msg]-Array
// Bestehende Aufrufstellen müssen entsprechend angepasst werden.
// ------------------------------------------------------------

// Defaults für die neuen Parameter, falls nicht angegeben oder <= 0
const (
	defaultBaseDelayMs = 500
	defaultMaxDelayMs  = 30000
)

func wolValidateMAC(mac string) bool {
	regex := regexp.MustCompile(`^([0-9A-Fa-f]{2}([-:])?){5}[0-9A-Fa-f]{2}$`)
	return regex.MatchString(mac)
}

func wolParseMAC(mac string) ([]byte, error) {
	mac = strings.ReplaceAll(mac, ":", "")
	mac = strings.ReplaceAll(mac, "-", "")

	if len(mac) != 12 {
		return nil, errors.New("ungültige MAC-Adresse")
	}

	macBytes := make([]byte, 6)
	for i := 0; i < 6; i++ {
		byteValue, err := strconv.ParseUint(mac[i*2:i*2+2], 16, 8)
		if err != nil {
			return nil, errors.New("fehler beim Parsen der MAC-Adresse")
		}
		macBytes[i] = byte(byteValue)
	}
	return macBytes, nil
}

func wolCreateMagicPacket(macBytes []byte) []byte {
	// 6x 0xFF, gefolgt von 16 Wiederholungen der MAC-Adresse
	magicPacket := make([]byte, 102)
	for i := 0; i < 6; i++ {
		magicPacket[i] = 0xFF
	}
	for i := 1; i <= 16; i++ {
		copy(magicPacket[i*6:(i+1)*6], macBytes)
	}
	return magicPacket
}

func retryDelays(baseDelayMs, maxDelayMs int) (int, int) {
	if baseDelayMs <= 0 {
		baseDelayMs = defaultBaseDelayMs
	}
	if maxDelayMs <= 0 {
		maxDelayMs = defaultMaxDelayMs
	}
	return baseDelayMs, maxDelayMs
}

// shouldRetryStatus: 5xx oder 429 -> Retry sinnvoll, alles andere nicht
func shouldRetryStatus(code int) bool {
	return code >= 500 || code == 429
}

var lastHttpStatus int = 0

func tcpCheck(host, port string, timeout time.Duration) bool {
	addr := net.JoinHostPort(host, port)
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func InitNetFunctions() {
	if builtins == nil {
		builtins = make(map[string]BuiltinInfo)
	}

	ns := "net."

	// =============================
	// ResolveAll
	// Rückgabe:
	// [1, 4, "ip", 6, "ip", ...]
	// [0] bei Fehler
	// =============================
	Register(ns+"ResolveAll", "net", "host", "Löst einen Hostnamen in alle verfügbaren IPv4/IPv6-Adressen auf.", func(args []Value) Value {

		if len(args) < 1 {
			return Value{Kind: KindArr, Arr: []Value{NumVal(0)}}
		}

		host := strings.TrimSpace(ToString(args[0]))
		if host == "" {
			return Value{Kind: KindArr, Arr: []Value{NumVal(0)}}
		}

		ips, err := net.LookupIP(host)
		if err != nil || len(ips) == 0 {
			return Value{Kind: KindArr, Arr: []Value{NumVal(0)}}
		}

		var arr []Value
		arr = append(arr, NumVal(1))

		for _, ip := range ips {
			if ip.To4() != nil {
				arr = append(arr, NumVal(4))
			} else {
				arr = append(arr, NumVal(6))
			}
			arr = append(arr, StrVal(ip.String()))
		}

		return Value{Kind: KindArr, Arr: arr}
	})

	Register(ns+"Resolve", "net",
		"host,type[,dnsServer]",
		"Löst einen Hostnamen in eine IPv4 oder IPv6 Adresse auf. Optional kann ein DNS-Server angegeben werden.",
		func(args []Value) Value {

			if len(args) < 2 {
				return ErrorVal("net.Resolve: mindestens 2 Argumente erforderlich (host, type)")
			}

			host := strings.TrimSpace(ToString(args[0]))
			ipType := strings.ToUpper(strings.TrimSpace(ToString(args[1])))

			if host == "" {
				return ErrorVal("net.Resolve: host ist leer")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			resolver := net.DefaultResolver

			// Optionaler DNS-Server
			if len(args) >= 3 {
				if dnsServer := strings.TrimSpace(ToString(args[2])); dnsServer != "" {
					resolver = &net.Resolver{
						PreferGo: true,
						Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
							d := net.Dialer{Timeout: 5 * time.Second}
							return d.DialContext(ctx, "udp", dnsServer+":53")
						},
					}
				}
			}

			ips, err := resolver.LookupIP(ctx, "ip", host)
			if err != nil {
				return ErrorVal("net.Resolve: Auflösung fehlgeschlagen: " + err.Error())
			}
			if len(ips) == 0 {
				return StrVal("")
			}

			for _, ip := range ips {
				switch ipType {
				case "IP4":
					if ip4 := ip.To4(); ip4 != nil {
						return StrVal(ip4.String())
					}
				case "IP6":
					if ip.To4() == nil {
						return StrVal(ip.String())
					}
				default:
					// Falls kein Typ angegeben wurde,
					// erste gefundene Adresse zurückgeben
					return StrVal(ip.String())
				}
			}

			// Adressen vorhanden, aber keine passte zum gewünschten Typ
			return StrVal("")
		})

	// =============================
	// Connect(host, port)
	// =============================
	Register(ns+"Connect", "net", "host, port", "Prüft eine TCP-Verbindung zu einem Host und Port.", func(args []Value) Value {

		if len(args) < 2 {
			return BoolVal(false)
		}

		host := strings.TrimSpace(ToString(args[0]))
		port := strings.TrimSpace(ToString(args[1]))

		if host == "" || port == "" {
			return BoolVal(false)
		}

		if tcpCheck(host, port, 3*time.Second) {
			return BoolVal(true)
		}
		return BoolVal(false)
	})

	// =============================
	// Ping(host, optional port)
	// Default: 443
	// =============================
	Register(ns+"Ping", "net", "host, [port]", "Prüft die Erreichbarkeit eines Hosts (Default Port 443).", func(args []Value) Value {

		if len(args) < 1 {
			return BoolVal(false)
		}

		host := strings.TrimSpace(ToString(args[0]))
		port := "443"

		if len(args) > 1 {
			p := strings.TrimSpace(ToString(args[1]))
			if p != "" {
				port = p
			}
		}

		if tcpCheck(host, port, 2*time.Second) {
			return BoolVal(true)
		}
		return BoolVal(false)
	})

	Register(ns+"Available", "net", "url", "Prüft, ob eine URL erreichbar ist (HTTP 200-299).", func(args []Value) Value {
		if len(args) < 1 {
			return BoolVal(false)
		}

		rawUrl := strings.TrimSpace(ToString(args[0]))

		// Kleiner Timeout, damit das Script nicht hängt
		client := &http.Client{
			Timeout: 5 * time.Second,
		}

		// HEAD ist schneller als GET, da kein Body übertragen wird
		resp, err := client.Head(rawUrl)
		if err != nil {
			// Falls HEAD nicht erlaubt ist, probieren wir ein kurzes GET
			resp, err = client.Get(rawUrl)
			if err != nil {
				return BoolVal(false)
			}
		}
		defer resp.Body.Close()

		// Prüfen, ob der Status-Code im Bereich 200-299 liegt
		return BoolVal(resp.StatusCode >= 200 && resp.StatusCode < 300)
	})

	// ------------------------------------------------------------
	// net.Get(url, [token], [retries], [baseDelayMs], [maxDelayMs])
	// ------------------------------------------------------------

	Register(ns+"Get", "net", "url, [token], [retries], [baseDelayMs], [maxDelayMs]",
		"Führt eine GET-Anfrage aus (optional mit Auth-Präfix gl:, gh:, b:). Bei Netzwerkfehlern, 5xx oder 429 wird bis zu 'retries'-mal mit exponentiellem Backoff wiederholt. Gibt bei Erfolg den Body zurück, sonst ErrorVal.",
		func(args []Value) Value {
			if len(args) < 1 {
				return ErrorVal("net.Get: URL fehlt")
			}
			u := strings.TrimSpace(ToString(args[0]))

			var token string
			if len(args) >= 2 {
				token = ToString(args[1])
			}
			retries := 0
			if len(args) >= 3 {
				retries = int(toNumVal(args[2]))
			}
			baseDelayMs := 0
			if len(args) >= 4 {
				baseDelayMs = int(toNumVal(args[3]))
			}
			maxDelayMs := 0
			if len(args) >= 5 {
				maxDelayMs = int(toNumVal(args[4]))
			}
			baseDelayMs, maxDelayMs = retryDelays(baseDelayMs, maxDelayMs)

			client := &http.Client{Timeout: 15 * time.Second}
			delay := baseDelayMs

			var lastErrMsg string
			var lastStatusCode int
			var lastBody []byte

			for attempt := 0; attempt <= retries; attempt++ {
				req, err := http.NewRequest("GET", u, nil)
				if err != nil {
					return ErrorVal("net.Get: ungültige Anfrage: " + err.Error())
				}
				req.Header.Set("User-Agent", "VBX/1.0")
				if token != "" {
					setAuthHeader(req, token)
				}

				resp, err := client.Do(req)
				if err != nil {
					lastHttpStatus = 0
					lastErrMsg = err.Error()
					lastStatusCode = 0
				} else {
					lastHttpStatus = resp.StatusCode
					lastStatusCode = resp.StatusCode
					b, _ := io.ReadAll(resp.Body)
					resp.Body.Close()

					if resp.StatusCode >= 200 && resp.StatusCode < 300 {
						return StrVal(string(b))
					}

					lastBody = b
					if !shouldRetryStatus(resp.StatusCode) {
						// Kein Retry-würdiger Status (z.B. 404, 401) -> sofort abbrechen
						return ErrorVal(fmt.Sprintf("net.Get: HTTP %d: %s", resp.StatusCode, string(b)))
					}
					lastErrMsg = fmt.Sprintf("HTTP %d", resp.StatusCode)
				}

				if attempt < retries {
					time.Sleep(time.Duration(delay) * time.Millisecond)
					delay *= 2
					if delay > maxDelayMs {
						delay = maxDelayMs
					}
				}
			}

			if lastStatusCode > 0 {
				return ErrorVal(fmt.Sprintf("net.Get: nach %d Versuch(en) fehlgeschlagen, letzter Status %d: %s", retries+1, lastStatusCode, string(lastBody)))
			}
			return ErrorVal(fmt.Sprintf("net.Get: nach %d Versuch(en) fehlgeschlagen: %s", retries+1, lastErrMsg))
		})

	// ------------------------------------------------------------
	// net.Post(url, data, [token], [retries], [baseDelayMs], [maxDelayMs])
	// ------------------------------------------------------------

	Register(ns+"Post", "net", "url, data, [token], [retries], [baseDelayMs], [maxDelayMs]",
		"POST-Anfrage mit Auto-JSON-Erkennung und optionaler Auth. Bei Netzwerkfehlern, 5xx oder 429 wird bis zu 'retries'-mal mit exponentiellem Backoff wiederholt. Gibt bei Erfolg den Body zurück, sonst ErrorVal.",
		func(args []Value) Value {
			if len(args) < 2 {
				return ErrorVal("net.Post: url und data benötigt")
			}
			u := strings.TrimSpace(ToString(args[0]))
			rawBody := ToString(args[1])

			var token string
			if len(args) >= 3 {
				token = ToString(args[2])
			}
			retries := 0
			if len(args) >= 4 {
				retries = int(toNumVal(args[3]))
			}
			baseDelayMs := 0
			if len(args) >= 5 {
				baseDelayMs = int(toNumVal(args[4]))
			}
			maxDelayMs := 0
			if len(args) >= 6 {
				maxDelayMs = int(toNumVal(args[5]))
			}
			baseDelayMs, maxDelayMs = retryDelays(baseDelayMs, maxDelayMs)

			contentType := "application/x-www-form-urlencoded"
			trimmed := strings.TrimSpace(rawBody)
			if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
				contentType = "application/json"
			}

			client := &http.Client{Timeout: 15 * time.Second}
			delay := baseDelayMs

			var lastErrMsg string
			var lastStatusCode int
			var lastBody []byte

			for attempt := 0; attempt <= retries; attempt++ {
				// Neuer Reader pro Versuch nötig, da der Body pro Request konsumiert wird
				req, err := http.NewRequest("POST", u, strings.NewReader(rawBody))
				if err != nil {
					return ErrorVal("net.Post: ungültige Anfrage: " + err.Error())
				}
				req.Header.Set("Content-Type", contentType)
				req.Header.Set("User-Agent", "VBX/1.0")
				if token != "" {
					setAuthHeader(req, token)
				}

				resp, err := client.Do(req)
				if err != nil {
					lastHttpStatus = 0
					lastErrMsg = err.Error()
					lastStatusCode = 0
				} else {
					lastHttpStatus = resp.StatusCode
					lastStatusCode = resp.StatusCode
					b, _ := io.ReadAll(resp.Body)
					resp.Body.Close()

					if resp.StatusCode >= 200 && resp.StatusCode < 300 {
						return StrVal(string(b))
					}

					lastBody = b
					if !shouldRetryStatus(resp.StatusCode) {
						return ErrorVal(fmt.Sprintf("net.Post: HTTP %d: %s", resp.StatusCode, string(b)))
					}
					lastErrMsg = fmt.Sprintf("HTTP %d", resp.StatusCode)
				}

				if attempt < retries {
					time.Sleep(time.Duration(delay) * time.Millisecond)
					delay *= 2
					if delay > maxDelayMs {
						delay = maxDelayMs
					}
				}
			}

			if lastStatusCode > 0 {
				return ErrorVal(fmt.Sprintf("net.Post: nach %d Versuch(en) fehlgeschlagen, letzter Status %d: %s", retries+1, lastStatusCode, string(lastBody)))
			}
			return ErrorVal(fmt.Sprintf("net.Post: nach %d Versuch(en) fehlgeschlagen: %s", retries+1, lastErrMsg))
		})

	Register(ns+"Status", "[url]", "Gibt den HTTP-Status der letzten Anfrage zurück oder prüft eine neue URL (HEAD-Request).", "If net.Status(\"https://google.de\") = 200 Then Print \"Online\"", func(args []Value) Value {
		// 1. FALL: Passiv (keine Argumente) -> Letzten gespeicherten Status liefern
		if len(args) == 0 {
			return NumVal(float64(lastHttpStatus))
		}

		// 2. FALL: Aktiv (URL übergeben) -> Server kurz anpingen (HEAD-Request)
		u := strings.TrimSpace(ToString(args[0]))
		if u == "" {
			return NumVal(0)
		}

		// HEAD ist schneller als GET, da nur der Header geladen wird, nicht der Inhalt
		resp, err := http.Head(u)
		if err != nil {
			return NumVal(0) // Host nicht erreichbar
		}
		defer resp.Body.Close()

		return NumVal(float64(resp.StatusCode))
	})

	Register(ns+"PostForm", "net", "url, token, title, message [, link]", "Sendet eine JSON-POST-Anfrage (z.B. für Push-Dienste wie Gotify). Optional kann ein Link angehängt werden, der als Markdown-Link und als klickbare Notification-Aktion gesetzt wird.", func(args []Value) Value {
		if len(args) < 4 {
			return ErrorVal("net.PostForm(url, token, title, message [, link]) benötigt mindestens 4 Argumente")
		}

		baseURL := strings.TrimRight(strings.TrimSpace(ToString(args[0])), "/")
		token := strings.TrimSpace(ToString(args[1]))
		title := ToString(args[2])
		message := ToString(args[3])

		var link string
		if len(args) >= 5 {
			link = strings.TrimSpace(ToString(args[4]))
			if link != "" {
				message += fmt.Sprintf("\n\n[%s](%s)", link, link)
			}
		}

		postURL := fmt.Sprintf("%s/message?token=%s", baseURL, token)

		payload := map[string]interface{}{
			"title":   title,
			"message": message,
		}

		if link != "" {
			payload["extras"] = map[string]interface{}{
				"client::display": map[string]string{
					"contentType": "text/markdown",
				},
				"client::notification": map[string]interface{}{
					"click": map[string]string{
						"url": link,
					},
				},
			}
		}

		jsonBody, err := json.Marshal(payload)
		if err != nil {
			return ErrorVal("JSON-Kodierung fehlgeschlagen: " + err.Error())
		}

		req, err := http.NewRequest("POST", postURL, bytes.NewReader(jsonBody))
		if err != nil {
			return ErrorVal("Request-Erstellung fehlgeschlagen: " + err.Error())
		}

		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			lastHttpStatus = 0
			return ErrorVal("Netzwerkfehler: " + err.Error())
		}
		defer resp.Body.Close()

		lastHttpStatus = resp.StatusCode

		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return ErrorVal(fmt.Sprintf("Status %d: %s", resp.StatusCode, string(body)))
		}

		return StrVal(string(body))
	})

	Register(ns+"CanResolve", "net", "host, [dnsServer]", "Prüft, ob ein Host aufgelöst werden kann (optional über spez. DNS).", func(args []Value) Value {
		if len(args) < 1 {
			return BoolVal(false)
		}

		// 1. Host säubern (Protokolle und Pfade entfernen)
		host := strings.TrimSpace(ToString(args[0]))
		if strings.Contains(host, "://") {
			parts := strings.SplitN(host, "://", 2)
			host = parts[1]
		}
		if idx := strings.Index(host, "/"); idx != -1 {
			host = host[:idx]
		}

		// 2. Resolver auswählen
		var resolver *net.Resolver

		if len(args) > 1 && strings.TrimSpace(ToString(args[1])) != "" {
			// Spezifischer DNS Server wurde angegeben
			dnsServer := strings.TrimSpace(ToString(args[1]))
			if !strings.Contains(dnsServer, ":") {
				dnsServer += ":53"
			}

			resolver = &net.Resolver{
				PreferGo: true,
				Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
					d := net.Dialer{Timeout: 2 * time.Second}
					return d.DialContext(ctx, "udp", dnsServer)
				},
			}
		} else {
			// Standard System-Resolver
			resolver = net.DefaultResolver
		}

		// 3. Auflösung versuchen
		_, err := resolver.LookupHost(context.Background(), host)
		return BoolVal(err == nil)
	})

	// ------------------------------------------------------------
	// net.Download(url, [path], [token], [retries], [baseDelayMs], [maxDelayMs])
	// ------------------------------------------------------------

	Register(ns+"Download", "net", "url, [path], [token], [retries], [baseDelayMs], [maxDelayMs]",
		"Lädt eine Datei herunter. Bei Netzwerkfehlern, 5xx oder 429 wird bis zu 'retries'-mal mit exponentiellem Backoff wiederholt. Gibt bei Erfolg den Zielpfad zurück, sonst ErrorVal.",
		func(args []Value) Value {
			if len(args) < 1 {
				return ErrorVal("net.Download: URL fehlt")
			}
			u := strings.TrimSpace(ToString(args[0]))
			if u == "" {
				return ErrorVal("net.Download: URL ist leer")
			}

			var p string
			if len(args) >= 2 && strings.TrimSpace(ToString(args[1])) != "" {
				absP, eVal := absPathVal(ToString(args[1]))
				if eVal != nil {
					return *eVal
				}
				p = absP
			} else {
				parts := strings.Split(u, "/")
				p = parts[len(parts)-1]
				if p == "" {
					p = "downloaded_file"
				}
			}

			var token string
			if len(args) >= 3 {
				token = ToString(args[2])
			}
			retries := 0
			if len(args) >= 4 {
				retries = int(toNumVal(args[3]))
			}
			baseDelayMs := 0
			if len(args) >= 5 {
				baseDelayMs = int(toNumVal(args[4]))
			}
			maxDelayMs := 0
			if len(args) >= 6 {
				maxDelayMs = int(toNumVal(args[5]))
			}
			baseDelayMs, maxDelayMs = retryDelays(baseDelayMs, maxDelayMs)

			dlClient := &http.Client{Timeout: 0}
			delay := baseDelayMs

			var lastErrMsg string

			for attempt := 0; attempt <= retries; attempt++ {
				req, err := http.NewRequest("GET", u, nil)
				if err != nil {
					return ErrorVal("net.Download: ungültige Anfrage: " + err.Error())
				}
				req.Header.Set("User-Agent", "VBX/1.0")
				if token != "" {
					setAuthHeader(req, token)
				}

				resp, err := dlClient.Do(req)
				if err != nil {
					lastErrMsg = err.Error()
				} else {
					if resp.StatusCode == http.StatusOK {
						file, ferr := os.Create(p)
						if ferr != nil {
							resp.Body.Close()
							return ErrorVal("net.Download: " + ferr.Error())
						}
						_, cerr := io.Copy(file, resp.Body)
						file.Close()
						resp.Body.Close()

						if cerr == nil {
							return StrVal(p)
						}
						lastErrMsg = cerr.Error()
					} else {
						body, _ := io.ReadAll(resp.Body)
						resp.Body.Close()

						if !shouldRetryStatus(resp.StatusCode) {
							return ErrorVal(fmt.Sprintf("net.Download: HTTP %d: %s", resp.StatusCode, string(body)))
						}
						lastErrMsg = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))
					}
				}

				if attempt < retries {
					time.Sleep(time.Duration(delay) * time.Millisecond)
					delay *= 2
					if delay > maxDelayMs {
						delay = maxDelayMs
					}
				}
			}

			return ErrorVal(fmt.Sprintf("net.Download: nach %d Versuch(en) fehlgeschlagen: %s", retries+1, lastErrMsg))
		})

	Register(ns+"Html", "Encode", "text",
		"Ersetzt Sonderzeichen wie Umlaute und HTML-Tags durch Entities (z.B. &auml;).",
		func(args []Value) Value {
			input := ToString(args[0])
			// Nutzt intern eine Map für Umlaute + html.EscapeString
			return StrVal(CustomHtmlEscape(input))
		})

	Register(ns+"LastStatus", "net", "-", "Gibt den HTTP-Statuscode der letzten Anfrage zurück.", func(args []Value) Value {
		// Direkt als Zahl zurückgeben, nicht als formatierten String
		return NumVal(float64(lastHttpStatus))
	})

	// --- Eigene IP-Konfiguration ---
	Register(ns+"LocalIPs", "net", "-", "Gibt ein Array aller aktiven lokalen IPv4-Adressen zurück.", func(args []Value) Value {
		var arr []Value
		for _, ip := range IPAddresses() { // Nutzt deine Logik aus der computer.go
			arr = append(arr, StrVal(ip))
		}
		return Value{Kind: KindArr, Arr: arr}
	})

	Register(ns+"IPIsValid", "net", "ip", "Prüft, ob ein String eine gültige IPv4- oder IPv6-Adresse ist.", func(args []Value) Value {
		if len(args) < 1 {
			return BoolVal(false)
		}
		ip := strings.TrimSpace(ToString(args[0]))
		return BoolVal(net.ParseIP(ip) != nil)
	})

	Register(ns+"MACs", "net", "-", "Listet die Hardware-Adressen (MAC) aller aktiven Schnittstellen auf.", func(args []Value) Value {
		var arr []Value
		ifaces, _ := net.Interfaces()
		for _, iface := range ifaces {
			if (iface.Flags&net.FlagUp) != 0 && len(iface.HardwareAddr) > 0 {
				mac := iface.HardwareAddr.String()
				if mac != "00:00:00:00:00:00" {
					arr = append(arr, StrVal(mac))
				}
			}
		}
		return Value{Kind: KindArr, Arr: arr}
	})

	Register(ns+"PublicIP", "net", "-", "Ermittelt die externe IP-Adresse über globale Provider.", func(args []Value) Value {
		return StrVal(PublicIP()) // Nutzt deine robuste Standalone-Funktion
	})

	Register(ns+"WakeOnLan", "net", "mac, [broadcastIP], [port]",
		"Sendet ein Wake-on-LAN Magic Packet per UDP-Broadcast. mac akzeptiert die Formate XX:XX:XX:XX:XX:XX, XX-XX-XX-XX-XX-XX oder ohne Trennzeichen.",
		func(args []Value) Value {
			if len(args) < 1 {
				return ErrorVal("net.WakeOnLan: MAC-Adresse fehlt")
			}

			mac := strings.TrimSpace(ToString(args[0]))
			if !wolValidateMAC(mac) {
				return ErrorVal("net.WakeOnLan: ungültige MAC-Adresse: " + mac)
			}

			broadcastIP := "255.255.255.255"
			if len(args) >= 2 && strings.TrimSpace(ToString(args[1])) != "" {
				broadcastIP = strings.TrimSpace(ToString(args[1]))
			}

			port := 9
			if len(args) >= 3 {
				p := int(toNumVal(args[2]))
				if p > 0 {
					port = p
				}
			}

			macBytes, err := wolParseMAC(mac)
			if err != nil {
				return ErrorVal("net.WakeOnLan: " + err.Error())
			}

			magicPacket := wolCreateMagicPacket(macBytes)

			conn, err := net.Dial("udp", fmt.Sprintf("%s:%d", broadcastIP, port))
			if err != nil {
				return ErrorVal("net.WakeOnLan: Verbindung fehlgeschlagen: " + err.Error())
			}
			defer conn.Close()

			if _, err := conn.Write(magicPacket); err != nil {
				return ErrorVal("net.WakeOnLan: Senden fehlgeschlagen: " + err.Error())
			}

			return BoolVal(true)
		})
}

func CustomHtmlEscape(s string) string {
	// Erst die Standard-Zeichen escapen
	s = html.EscapeString(s)

	// Dann gezielt Umlaute ersetzen
	replacer := strings.NewReplacer(
		"ä", "&auml;", "ö", "&ouml;", "ü", "&uuml;",
		"Ä", "&Auml;", "Ö", "&Ouml;", "Ü", "&Uuml;",
		"ß", "&szlig;",
	)
	return replacer.Replace(s)
}

func setAuthHeader(req *http.Request, auth string) {
	auth = strings.TrimSpace(auth)

	if auth == "" {
		return
	}

	switch {
	case strings.HasPrefix(auth, "gl:"):
		// GitLab Personal Access Token
		req.Header.Set("PRIVATE-TOKEN", strings.TrimPrefix(auth, "gl:"))

	case strings.HasPrefix(auth, "gh:"):
		// GitHub Token
		req.Header.Set("Authorization", "Bearer "+strings.TrimPrefix(auth, "gh:"))

	case strings.HasPrefix(auth, "b:"):
		// Bearer Token
		req.Header.Set("Authorization", "Bearer "+strings.TrimPrefix(auth, "b:"))

	case strings.HasPrefix(auth, "a:"):
		// kompletter Authorization Header
		req.Header.Set("Authorization", strings.TrimPrefix(auth, "a:"))

	default:
		// Standard: Bearer
		req.Header.Set("Authorization", "Bearer "+auth)
	}
}

func PublicIP() string {
	// Ein lokaler Client mit kurzem Timeout ist hier besser,
	// damit das Skript nicht 30 Sekunden hängt, wenn das Internet weg ist.
	localClient := &http.Client{
		Timeout: 5 * time.Second,
	}

	providers := []string{
		"https://api.ipify.org",
		"https://ifconfig.me/ip",
		"https://checkip.amazonaws.com",
	}

	for _, url := range providers {
		resp, err := localClient.Get(url)
		if err != nil {
			continue
		}

		// Body lesen
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close() // Sofort schließen, kein defer in der Schleife!

		if err == nil && len(body) > 0 {
			// TrimSpace ist wichtig, da viele Provider ein "\n" mitschicken
			return strings.TrimSpace(string(body))
		}
	}
	return ""
}

func IPAddresses() []string {
	var ips []string
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		if (iface.Flags & net.FlagUp) == 0 {
			continue // Interface down
		}
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // nur IPv4 zurückgeben
			}
			if ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
				continue
			}
			ips = append(ips, ip.String())
		}
	}
	return ips
}
