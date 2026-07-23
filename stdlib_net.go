// ------------------------
// stdlib_net.go
// ------------------------

package main

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

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

	// =============================
	// Get(url, [token])
	// =============================
	Register(ns+"Get", "net", "url, [token]", "Führt eine GET-Anfrage aus (optional mit Auth-Präfix gl:, gh:, b:).", func(args []Value) Value {
		if len(args) < 1 {
			return StrVal("")
		}
		u := strings.TrimSpace(ToString(args[0]))

		req, err := http.NewRequest("GET", u, nil)
		if err != nil {
			return StrVal("")
		}
		req.Header.Set("User-Agent", "VBX/1.0")

		if len(args) >= 2 {
			setAuthHeader(req, ToString(args[1]))
		}

		client := &http.Client{Timeout: 15 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			lastHttpStatus = 0
			return StrVal("")
		}
		defer resp.Body.Close()

		lastHttpStatus = resp.StatusCode
		body, _ := io.ReadAll(resp.Body)
		return StrVal(string(body))
	})

	// =============================
	// Post(url, data, [token])
	// =============================
	Register(ns+"Post", "net", "url, data, [token]", "POST-Anfrage mit Auto-JSON-Erkennung und optionaler Auth.", func(args []Value) Value {
		if len(args) < 2 {
			return StrVal("")
		}
		u := strings.TrimSpace(ToString(args[0]))
		rawBody := ToString(args[1])

		req, err := http.NewRequest("POST", u, strings.NewReader(rawBody))
		if err != nil {
			return StrVal("")
		}

		// Content-Type Logik
		trimmed := strings.TrimSpace(rawBody)
		if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
			req.Header.Set("Content-Type", "application/json")
		} else {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}

		req.Header.Set("User-Agent", "VBX/1.0")

		if len(args) >= 3 {
			setAuthHeader(req, ToString(args[2]))
		}

		client := &http.Client{Timeout: 15 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			lastHttpStatus = 0
			return StrVal("HTTP ERROR: " + err.Error())
		}
		defer resp.Body.Close()

		lastHttpStatus = resp.StatusCode
		body, _ := io.ReadAll(resp.Body)
		return StrVal(string(body))
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

	Register(ns+"PostForm", "net", "url, token, title, message", "Sendet eine Multipart-POST-Anfrage (z.B. für Push-Dienste).", func(args []Value) Value {
		if len(args) < 4 {
			return ErrorVal("net.PostForm(url, token, title, message) benötigt 4 Argumente")
		}

		baseURL := strings.TrimRight(strings.TrimSpace(ToString(args[0])), "/")
		token := strings.TrimSpace(ToString(args[1]))
		title := ToString(args[2])
		message := ToString(args[3])

		postURL := fmt.Sprintf("%s/message?token=%s", baseURL, token)

		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		_ = writer.WriteField("title", title)
		_ = writer.WriteField("message", message)
		writer.Close()

		req, err := http.NewRequest("POST", postURL, &buf)
		if err != nil {
			return ErrorVal("Request-Erstellung fehlgeschlagen: " + err.Error())
		}

		// Header mit Boundary setzen
		req.Header.Set("Content-Type", writer.FormDataContentType())

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)

		// Sicherheit-Check: Wenn Netzwerkfehler (err != nil), ist resp meist nil
		if err != nil {
			lastHttpStatus = 0
			return ErrorVal("Netzwerkfehler: " + err.Error())
		}

		// Ab hier ist resp garantiert nicht nil
		defer resp.Body.Close()
		lastHttpStatus = resp.StatusCode

		body, _ := io.ReadAll(resp.Body)

		// Fehlerbehandlung für HTTP-Statuscodes (4xx, 5xx)
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

	// =============================
	// Download(url, [path], [token])
	// =============================
	Register(ns+"Download", "net", "url, [path], [token]", "Lädt eine Datei herunter. Gibt True bei Erfolg, False bei Fehler zurück.", func(args []Value) Value {
		if len(args) < 1 {
			return BoolVal(false)
		}
		u := strings.TrimSpace(ToString(args[0]))
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

		req, _ := http.NewRequest("GET", u, nil)
		req.Header.Set("User-Agent", "VBX/1.0")

		if len(args) >= 3 {
			setAuthHeader(req, ToString(args[2]))
		}

		dlClient := &http.Client{Timeout: 0} // Timeout für Downloads auf 0 (unendlich) oder via Context
		resp, err := dlClient.Do(req)
		if err != nil || resp.StatusCode != 200 {
			return BoolVal(false)
		}
		defer resp.Body.Close()

		file, err := os.Create(p)
		if err != nil {
			return BoolVal(false)
		}
		defer file.Close()

		_, err = io.Copy(file, resp.Body)
		if err != nil {
			return BoolVal(false)
		}
		return BoolVal(true)
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

func setAuthHeader(req *http.Request, token string) {
	token = strings.TrimSpace(token)
	if token == "" {
		return
	}

	// 1. Explizite Präfixe prüfen
	if strings.HasPrefix(token, "gl:") {
		// GitLab Spezial-Header
		req.Header.Set("PRIVATE-TOKEN", strings.TrimPrefix(token, "gl:"))
	} else if strings.HasPrefix(token, "gh:") {
		// GitHub Standard
		req.Header.Set("Authorization", "token "+strings.TrimPrefix(token, "gh:"))
	} else if strings.HasPrefix(token, "b:") {
		// Erzwungener Bearer (FastAPI)
		req.Header.Set("Authorization", "Bearer "+strings.TrimPrefix(token, "b:"))
	} else {
		// 2. Fallback: Wenn nichts angegeben ist, ist es ein Bearer (FastAPI Standard)
		// Das deckt 90% der modernen APIs ab.
		req.Header.Set("Authorization", "Bearer "+token)
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
