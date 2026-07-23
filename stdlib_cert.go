// ------------------------
// stdlib_cert.go
// ------------------------

package main

import (
	"bufio"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fullsailor/pkcs7"
	"software.sslmate.com/src/go-pkcs12"
)

func InitCertFunctions() {
	if builtins == nil {
		builtins = make(map[string]BuiltinInfo)
	}

	ns := "cert."

	// ---------------- cert.GenerateKey ----------------
	Register(ns+"GenerateKey", "cert",
		"outFile, algo, bits",
		"Erzeugt einen Private Key (RSA ≥4096 oder ECDSA P256/P384/P521) im PKCS#8-Format.",
		func(args []Value) Value {

			if len(args) < 3 {
				return ErrorVal("usage: cert.GenerateKey(outFile, algo, bits)")
			}

			outFile, errVal := absPathVal(args[0].Str)
			if errVal != nil {
				return *errVal
			}

			algo := strings.ToLower(args[1].Str)
			bits, _ := strconv.Atoi(args[2].Str)

			var key crypto.PrivateKey
			var err error

			switch algo {
			case "rsa":
				if bits < 4096 {
					bits = 4096
				}
				key, err = rsa.GenerateKey(rand.Reader, bits)

			case "ecdsa":
				var curve elliptic.Curve
				switch bits {
				case 256:
					curve = elliptic.P256()
				case 384:
					curve = elliptic.P384()
				case 521:
					curve = elliptic.P521()
				default:
					curve = elliptic.P384()
				}
				key, err = ecdsa.GenerateKey(curve, rand.Reader)

			default:
				return ErrorVal("Unsupported algorithm")
			}

			if err != nil {
				return ErrorVal(err.Error())
			}

			der, err := x509.MarshalPKCS8PrivateKey(key)
			if err != nil {
				return ErrorVal(err.Error())
			}

			err = os.WriteFile(outFile, pem.EncodeToMemory(&pem.Block{
				Type:  "PRIVATE KEY",
				Bytes: der,
			}), 0600)

			return BoolVal(err == nil)
		})

	Register(ns+"CreateCSR", "cert",
		"subject, keyPath, outFile [, SANs]",
		"Erstellt eine Certificate Signing Request (CSR) mit optionalen DNS/IP SANs.",
		func(args []Value) Value {

			if len(args) < 3 {
				return ErrorVal("usage: cert.CreateCSR(subject, keyPath, outFile [, SANs])")
			}

			keyPath, _ := absPathVal(args[1].Str)
			outPath, _ := absPathVal(args[2].Str)

			priv, err := loadPrivateKey(keyPath)
			if err != nil {
				return ErrorVal(err.Error())
			}

			var dns []string
			var ips []net.IP

			if len(args) >= 4 {
				for _, s := range strings.Split(args[3].Str, ",") {
					s = strings.TrimSpace(s)
					if ip := net.ParseIP(s); ip != nil {
						ips = append(ips, ip)
					} else {
						dns = append(dns, s)
					}
				}
			}

			tmpl := x509.CertificateRequest{
				Subject:     pkix.Name{CommonName: args[0].Str},
				DNSNames:    dns,
				IPAddresses: ips,
			}

			csrBytes, err := x509.CreateCertificateRequest(rand.Reader, &tmpl, priv)
			if err != nil {
				return ErrorVal(err.Error())
			}

			err = os.WriteFile(outPath, pem.EncodeToMemory(&pem.Block{
				Type:  "CERTIFICATE REQUEST",
				Bytes: csrBytes,
			}), 0644)

			return BoolVal(err == nil)
		})

	// ---------------- ExportPEM ----------------
	Register(ns+"ExportPEM", "cert", "certPath, outFile", "Kopiert ein Zertifikat unverändert ins PEM-Format.", func(args []Value) Value {
		if len(args) < 2 {
			return ErrorVal("usage: cert.ExportPEM(certPath, outFile)")
		}

		// 1. Pfade sicher auflösen (Sandbox-Check)
		// inP = Input Path, outP = Output Path
		inP, errVal1 := absPathVal(args[0].Str)
		if errVal1 != nil {
			return *errVal1
		}

		outP, errVal2 := absPathVal(args[1].Str)
		if errVal2 != nil {
			return *errVal2
		}

		// 2. Quelldaten lesen (nutzt validierten inP)
		certData, err := os.ReadFile(inP)
		if err != nil {
			return ErrorVal("Zertifikat konnte nicht gelesen werden: " + err.Error())
		}

		// 3. Datei schreiben (nutzt validierten outP)
		// 0644 ist der Standard für öffentliche Zertifikate
		if err := os.WriteFile(outP, certData, 0644); err != nil {
			return ErrorVal("Fehler beim Schreiben der PEM-Datei: " + err.Error())
		}

		// 4. Erfolg zurückgeben
		return NullVal()
	})

	// ---------------- ExportDER ----------------
	Register(ns+"ExportDER", "cert", "certPath, outFile", "Konvertiert ein PEM-Zertifikat in das binäre DER-Format.", func(args []Value) Value {
		if len(args) < 2 {
			return ErrorVal("usage: cert.ExportDER(certPath, outFile)")
		}

		// 1. Pfade sicher auflösen (Sandbox-Check)
		inP, errVal1 := absPathVal(args[0].Str)
		if errVal1 != nil {
			return *errVal1
		}

		outP, errVal2 := absPathVal(args[1].Str)
		if errVal2 != nil {
			return *errVal2
		}

		// 2. Quelldatei lesen
		certData, err := os.ReadFile(inP)
		if err != nil {
			return ErrorVal("Fehler beim Lesen der Zertifikatsdatei: " + err.Error())
		}

		// 3. PEM-Dekodierung
		block, _ := pem.Decode(certData)
		if block == nil {
			return ErrorVal("Datei ist kein gültiges PEM-Format")
		}

		// 4. Typ-Prüfung (Sicherheits-Feature)
		// Wir stellen sicher, dass wir nur Zertifikate exportieren, keine privaten Schlüssel!
		if block.Type != "CERTIFICATE" {
			return ErrorVal("Export abgebrochen: PEM-Block ist vom Typ '" + block.Type + "', erwartet wurde 'CERTIFICATE'")
		}

		// 5. Schreiben der binären DER-Daten
		if err := os.WriteFile(outP, block.Bytes, 0644); err != nil {
			return ErrorVal("Fehler beim Schreiben der DER-Datei: " + err.Error())
		}

		return NullVal()
	})

	// ---------------- GetPublicKey ----------------
	Register(ns+"GetPublicKey", "cert", "certPath", "Extrahiert den öffentlichen Schlüssel aus einem Zertifikat.", func(args []Value) Value {
		if len(args) < 1 {
			return ErrorVal("usage: cert.GetPublicKey(certPath)")
		}

		// 1. Pfad sicher auflösen (Sandbox-Check)
		path, errVal := absPathVal(args[0].Str)
		if errVal != nil {
			return *errVal
		}

		// 2. Zertifikatsdatei lesen
		data, err := os.ReadFile(path)
		if err != nil {
			return ErrorVal("Zertifikat konnte nicht gelesen werden: " + err.Error())
		}

		// 3. PEM-Block dekodieren
		block, _ := pem.Decode(data)
		if block == nil || block.Type != "CERTIFICATE" {
			return ErrorVal("Kein gültiges Zertifikat im PEM-Format gefunden")
		}

		// 4. Zertifikat parsen
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return ErrorVal("Zertifikat-Parse-Fehler: " + err.Error())
		}

		// 5. Öffentlichen Schlüssel extrahieren und in PKIX/DER formatieren
		pubBytes, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
		if err != nil {
			return ErrorVal("Fehler beim Exportieren des Public Keys: " + err.Error())
		}

		// 6. In PEM-Format umwandeln (für bessere Lesbarkeit im Skript)
		pubPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: pubBytes,
		})

		return StrVal(string(pubPEM))
	})

	Register(ns+"CreateSelfSigned", "cert", "subject, keyPath, outCert [, days, SANs, isCA]", "Erstellt ein selbstsigniertes Zertifikat.", func(args []Value) Value {
		if len(args) < 3 {
			return ErrorVal("usage: cert.CreateSelfSigned(subject, keyPath, outCert [, days, SANs, isCA])")
		}

		// 1. Pfade sicher auflösen (Sandbox-Check)
		kPath, errVal1 := absPathVal(args[1].Str)
		if errVal1 != nil {
			return *errVal1
		}

		oPath, errVal2 := absPathVal(args[2].Str)
		if errVal2 != nil {
			return *errVal2
		}

		subject := args[0].Str
		days := 365
		if len(args) >= 4 {
			if d, err := strconv.Atoi(args[3].Str); err == nil {
				days = d
			}
		}

		// 2. Key laden (nutzt validierten kPath)
		keyData, err := os.ReadFile(kPath)
		if err != nil {
			return ErrorVal("Key-Lesefehler: " + err.Error())
		}

		block, _ := pem.Decode(keyData)
		if block == nil {
			return ErrorVal("Ungültiger Private Key (PEM erwartet)")
		}

		var priv crypto.PrivateKey
		switch block.Type {
		case "RSA PRIVATE KEY":
			priv, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		case "EC PRIVATE KEY":
			priv, err = x509.ParseECPrivateKey(block.Bytes)
		default:
			return ErrorVal("Nicht unterstützter Key-Typ: " + block.Type)
		}
		if err != nil {
			return ErrorVal("Key-Parse-Fehler: " + err.Error())
		}

		// 3. SANs (Subject Alternative Names) verarbeiten
		var dnsNames []string
		var ipAddresses []net.IP
		if len(args) >= 5 && args[4].Str != "" {
			for _, s := range strings.Split(args[4].Str, ",") {
				s = strings.TrimSpace(s)
				if ip := net.ParseIP(s); ip != nil {
					ipAddresses = append(ipAddresses, ip)
				} else {
					dnsNames = append(dnsNames, s)
				}
			}
		}

		isCA := false
		if len(args) >= 6 && strings.ToLower(args[5].Str) == "true" {
			isCA = true
		}

		// 4. Zertifikat-Template
		serial, _ := rand.Int(rand.Reader, big.NewInt(1<<62))
		tmpl := x509.Certificate{
			SerialNumber: serial,
			Subject: pkix.Name{
				CommonName: subject,
			},
			NotBefore:             time.Now().Add(-5 * time.Minute),
			NotAfter:              time.Now().Add(time.Duration(days) * 24 * time.Hour),
			DNSNames:              dnsNames,
			IPAddresses:           ipAddresses,
			BasicConstraintsValid: true,
			IsCA:                  isCA,
			KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		}

		if isCA {
			tmpl.KeyUsage |= x509.KeyUsageCertSign
		}

		// 5. Selbst signieren
		certBytes, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, publicKey(priv), priv)
		if err != nil {
			return ErrorVal("Zertifikat-Erstellung fehlgeschlagen: " + err.Error())
		}

		// 6. Speichern (nutzt validierten oPath)
		certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
		if err := os.WriteFile(oPath, certPEM, 0644); err != nil {
			return ErrorVal("Speicherfehler: " + err.Error())
		}

		return NullVal()
	})

	Register(ns+"SignCSR", "cert",
		"csrPath, caCert, caKey, outCert [, days]",
		"Signiert eine CSR mit einer CA und erzeugt ein gültiges Zertifikat.",
		func(args []Value) Value {

			if len(args) < 4 {
				return ErrorVal("usage: cert.SignCSR(...)")
			}

			csrPath, _ := absPathVal(args[0].Str)
			caCertPath, _ := absPathVal(args[1].Str)
			caKeyPath, _ := absPathVal(args[2].Str)
			outPath, _ := absPathVal(args[3].Str)

			days := 365
			if len(args) >= 5 {
				days, _ = strconv.Atoi(args[4].Str)
			}

			csrData, _ := os.ReadFile(csrPath)
			csr, err := parseAndValidateCSR(csrData)
			if err != nil {
				return ErrorVal(err.Error())
			}

			caCertData, _ := os.ReadFile(caCertPath)
			block, _ := pem.Decode(caCertData)
			caCert, _ := x509.ParseCertificate(block.Bytes)

			caKey, err := loadPrivateKey(caKeyPath)
			if err != nil {
				return ErrorVal(err.Error())
			}

			tmpl := &x509.Certificate{
				SerialNumber: newSerial(),
				Subject:      csr.Subject,
				DNSNames:     csr.DNSNames,
				IPAddresses:  csr.IPAddresses,
				NotBefore:    time.Now().Add(-5 * time.Minute),
				NotAfter:     time.Now().AddDate(0, 0, days),
				KeyUsage:     x509.KeyUsageDigitalSignature,
				ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			}

			certBytes, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, csr.PublicKey, caKey)
			if err != nil {
				return ErrorVal(err.Error())
			}

			err = os.WriteFile(outPath, pem.EncodeToMemory(&pem.Block{
				Type:  "CERTIFICATE",
				Bytes: certBytes,
			}), 0644)

			return BoolVal(err == nil)
		})

	Register(ns+"ExportPFX", "cert", "certP, keyP, outP, pass [, fName, mode]", "Exportiert Zertifikat und Key in einen PKCS#12 (PFX) Container.", func(args []Value) Value {
		if len(args) < 4 {
			return ErrorVal("usage: cert.ExportPFX(certP, keyP, outP, pass [, fName, mode])")
		}

		certP, keyP, outP, password := args[0].Str, args[1].Str, args[2].Str, args[3].Str
		fName := ""
		mode := "legacy"
		if len(args) >= 5 {
			fName = args[4].Str
		}
		if len(args) >= 6 {
			mode = strings.ToLower(args[5].Str)
		}

		// --- DER OPENSSL-TRICK: Alles aus der Datei lesen ---
		certData, err := os.ReadFile(certP)
		if err != nil {
			return ErrorVal("Datei nicht lesbar: " + err.Error())
		}

		var allCerts []*x509.Certificate
		rest := certData
		for {
			var block *pem.Block
			block, rest = pem.Decode(rest) // Sucht den nächsten Block
			if block == nil {
				break
			} // Keine Blöcke mehr da

			if block.Type == "CERTIFICATE" {
				c, err := x509.ParseCertificate(block.Bytes)
				if err != nil {
					return ErrorVal("Zertifikat-Fehler: " + err.Error())
				}
				allCerts = append(allCerts, c)
			}
		}

		if len(allCerts) == 0 {
			return ErrorVal("Keine Zertifikate in der Datei gefunden!")
		}

		// Erstes Zertifikat = Hauptzertifikat (Leaf)
		// Alle weiteren = Kette (Chain)
		leaf := allCerts[0]
		var chain []*x509.Certificate
		if len(allCerts) > 1 {
			chain = allCerts[1:]
		}
		// ---------------------------------------------------

		key, err := loadPrivateKey(keyP)
		if err != nil {
			return ErrorVal(err.Error())
		}

		var encoder *pkcs12.Encoder
		if mode == "modern" {
			encoder = pkcs12.Modern
		} else {
			encoder = pkcs12.LegacyRC2
		}

		// FriendlyName Info (für den Compiler)
		if fName != "" {
			fmt.Printf("PFX Export: %s\n", fName)
		}

		// Jetzt bekommt der Encoder den Key, das Leaf UND die restliche Chain
		pfxData, err := encoder.Encode(key, leaf, chain, password)
		if err != nil {
			return ErrorVal("Encoding Fehler: " + err.Error())
		}

		if err := os.WriteFile(outP, pfxData, 0644); err != nil {
			return ErrorVal("Schreibfehler: " + err.Error())
		}

		return BoolVal(true)
	})

	Register(ns+"Combine", "cert", "cert1, ..., outFile", "Kombiniert mehrere Zertifikate zu einer PEM-Kette (Chain).", func(args []Value) Value {
		if len(args) < 2 {
			return ErrorVal("usage: cert.Combine(cert1, ..., outFile)")
		}

		// Pfad-Intelligenz für das Ziel
		outP, _ := absPathVal(args[len(args)-1].Str)
		if err := ensureDir(outP); err != nil {
			return ErrorVal("Ordner-Fehler: " + err.Error())
		}

		sourceFiles := args[:len(args)-1]
		var combinedPEM []byte

		for _, fileVal := range sourceFiles {
			path, _ := absPathVal(fileVal.Str) // Auch Quellpfade auflösen
			data, err := os.ReadFile(path)
			if err != nil {
				return ErrorVal(fmt.Sprintf("Fehler beim Lesen von %s: %v", path, err))
			}

			// ... (PEM/DER Erkennung wie zuvor) ...
			block, _ := pem.Decode(data)
			if block != nil {
				current := data
				for {
					var b *pem.Block
					var rest []byte
					b, rest = pem.Decode(current)
					if b == nil {
						break
					}
					if b.Type == "CERTIFICATE" {
						combinedPEM = append(combinedPEM, pem.EncodeToMemory(b)...)
					}
					current = rest
				}
			} else {
				cert, err := x509.ParseCertificate(data)
				if err == nil {
					combinedPEM = append(combinedPEM, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})...)
				}
			}
		}

		err := os.WriteFile(outP, combinedPEM, 0644)
		return BoolVal(err == nil)
	})

	Register(ns+"ExportPKCS7", "cert", "certPath, outFile", "Erstellt einen PKCS#7 Container aus einem oder mehreren Zertifikaten.", func(args []Value) Value {
		if len(args) < 2 {
			return ErrorVal("usage: cert.ExportPKCS7(certPath, outFile)")
		}

		certPath, _ := absPathVal(args[0].Str)
		outFile, _ := absPathVal(args[1].Str)

		if err := ensureDir(outFile); err != nil {
			return ErrorVal("Ordner-Fehler: " + err.Error())
		}

		// Wir laden hier alle Zertifikate aus der Datei (für Bundles)
		// Nutze hier die Logik aus ExportPFX, um den gesamten Stapel zu lesen
		certs, err := loadAllCerts(certPath)
		if err != nil {
			return ErrorVal("Zertifikat-Ladefehler: " + err.Error())
		}

		// Neues PKCS7 Objekt
		p7, err := pkcs7.NewSignedData([]byte{})
		if err != nil {
			return ErrorVal("PKCS7-Init Fehler: " + err.Error())
		}

		// Alle gefundenen Zertifikate zum Container hinzufügen
		for _, c := range certs {
			p7.AddCertificate(c)
		}

		// Den binären PKCS7 Block generieren
		out, err := p7.Finish()
		if err != nil {
			return ErrorVal("PKCS7-Finish Fehler: " + err.Error())
		}

		// Speichern als binäre Datei
		if err := os.WriteFile(outFile, out, 0644); err != nil {
			return ErrorVal("Schreibfehler: " + err.Error())
		}

		return BoolVal(true)
	})

	Register(ns+"CreateConf", "cert", "[cn, dnsArray, outFile]", "Assistent (interaktiv) oder Funktion zum Erstellen einer OpenSSL-Konfigurationsdatei.", func(args []Value) Value {
		var cn string
		var dnsNames []string
		var ipNames []string
		var outFile string
		scanner := bufio.NewScanner(os.Stdin)

		// Helper für interaktive Abfragen
		ask := func(prompt string, defaultVal string) string {
			suffix := ""
			if defaultVal != "" {
				suffix = " [" + defaultVal + "]"
			}
			fmt.Print(prompt + suffix + ": ")
			if scanner.Scan() {
				input := strings.TrimSpace(scanner.Text())
				if input == "" {
					return defaultVal
				}
				return input
			}
			return defaultVal
		}

		// --- LOGIK-ZWEIG: INTERAKTIV ODER DIREKT ---
		if len(args) == 0 {
			fmt.Println("\n--- VBMini Zertifikats-Assistent ---")
			cn = ask("Common Name (CN)", "localhost")
			dnsNames = append(dnsNames, cn) // CN ist immer der erste DNS

			fmt.Print("Möchten Sie weitere DNS/IP-Adressen hinzufügen? (j/n): ")
			if scanner.Scan() && strings.ToLower(strings.TrimSpace(scanner.Text())) == "j" {
				for {
					entry := ask("Zusätzlicher Eintrag (leer zum Beenden)", "")
					if entry == "" {
						break
					}

					if ip := net.ParseIP(entry); ip != nil {
						ipNames = append(ipNames, entry)
					} else {
						dnsNames = append(dnsNames, entry)
					}
				}
			}
			outFile = ask("Zielordner oder Dateiname", "")
		} else {
			// Direkte Übergabe aus einem Skript: CreateConf(cn, dnsArray, [outFile])
			cn = args[0].Str
			dnsNames = append(dnsNames, cn)
			if len(args) > 1 {
				for _, v := range args[1].Arr {
					if ip := net.ParseIP(v.Str); ip != nil {
						ipNames = append(ipNames, v.Str)
					} else {
						dnsNames = append(dnsNames, v.Str)
					}
				}
			}
			if len(args) > 2 {
				outFile = args[2].Str
			}
		}

		// --- TEMPLATE BAUEN ---
		var sb strings.Builder
		for i, d := range dnsNames {
			sb.WriteString(fmt.Sprintf("DNS.%d = %s\n", i+1, d))
		}
		for i, ip := range ipNames {
			sb.WriteString(fmt.Sprintf("IP.%d = %s\n", i+1, ip))
		}

		confTemplate := fmt.Sprintf(`[ req ]
distinguished_name = req_distinguished_name
req_extensions = v3_req
prompt = no

[ req_distinguished_name ]
CN = %s

[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
subjectAltName = @alt_names

[ alt_names ]
%s`, cn, sb.String())

		// --- INTELLIGENTE PFAD-LOGIK ---
		finalPath := ""
		cwd, _ := os.Getwd()

		if outFile == "" {
			// Fall 1: Gar kein Pfad -> Aktueller Ordner + CN.conf
			finalPath = filepath.Join(cwd, cn+".conf")
		} else {
			absOut, _ := filepath.Abs(outFile)
			fi, err := os.Stat(absOut)

			// Fall 2: Pfad ist ein Ordner oder endet auf Slash
			if (err == nil && fi.IsDir()) || strings.HasSuffix(outFile, "/") || strings.HasSuffix(outFile, "\\") {
				finalPath = filepath.Join(absOut, cn+".conf")
			} else {
				// Fall 3: Pfad ist ein direkter Dateiname
				finalPath = absOut
			}
		}

		// --- OVERWRITE-SCHUTZ ---
		if _, err := os.Stat(finalPath); err == nil {
			fmt.Printf("\x1b[33mDatei existiert bereits:\x1b[0m %s\n", filepath.Base(finalPath))
			fmt.Print("Überschreiben? (j/n): ")
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(response) != "j" && strings.ToLower(response) != "y" {
				fmt.Println("Vorgang abgebrochen.")
				return BoolVal(false)
			}
		}

		// --- SPEICHERN ---
		dir := filepath.Dir(finalPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return ErrorVal("Ordner-Fehler: " + err.Error())
		}
		if err := os.WriteFile(finalPath, []byte(confTemplate), 0644); err != nil {
			return ErrorVal("Schreibfehler: " + err.Error())
		}

		fmt.Printf("\x1b[32m✔ Konfiguration erfolgreich gespeichert:\x1b[0m %s\n", finalPath)
		return BoolVal(true)
	})

	Register(ns+"CreateCSRConf", "cert", "confPath, keyPath, outCSR", "Erstellt eine OpenSSL-Konfigurationsdatei mit SAN-Einträgen.", func(args []Value) Value {
		if len(args) < 3 {
			return ErrorVal("usage: cert.CreateCSRConf(confPath, keyPath, outCSR)")
		}

		confP, _ := absPathVal(args[0].Str)
		keyP, _ := absPathVal(args[1].Str)
		outP, _ := absPathVal(args[2].Str)

		if err := ensureDir(outP); err != nil {
			return ErrorVal(err.Error())
		}

		// 1. Config-Datei einlesen
		confData, err := os.ReadFile(confP)
		if err != nil {
			return ErrorVal("Config-Ladefehler: " + err.Error())
		}
		confStr := string(confData)

		// 2. Key laden
		priv, err := loadPrivateKey(keyP)
		if err != nil {
			return ErrorVal("Key-Ladefehler: " + err.Error())
		}

		// 3. Einfacher Parser für CN und DNS (SAN)
		var cn string
		var dnsNames []string

		lines := strings.Split(confStr, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "CN =") {
				cn = strings.TrimSpace(strings.TrimPrefix(line, "CN ="))
			} else if strings.Contains(line, "DNS.") && strings.Contains(line, "=") {
				parts := strings.Split(line, "=")
				if len(parts) == 2 {
					dnsNames = append(dnsNames, strings.TrimSpace(parts[1]))
				}
			}
		}

		// 4. CSR Template füllen
		csrTemplate := x509.CertificateRequest{
			Subject:  pkix.Name{CommonName: cn},
			DNSNames: dnsNames,
		}

		// 5. CSR erstellen
		csrBytes, err := x509.CreateCertificateRequest(rand.Reader, &csrTemplate, priv)
		if err != nil {
			return ErrorVal("CSR-Fehler: " + err.Error())
		}

		// 6. Als PEM speichern
		pemBlock := &pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrBytes}
		err = os.WriteFile(outP, pem.EncodeToMemory(pemBlock), 0644)

		return BoolVal(err == nil)
	})
}

func ensureDir(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0755)
}

func loadAllCerts(certPath string) ([]*x509.Certificate, error) {
	data, err := os.ReadFile(certPath)
	if err != nil {
		return nil, err
	}

	var certs []*x509.Certificate
	rest := data
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type == "CERTIFICATE" {
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, err
			}
			certs = append(certs, cert)
		}
	}

	if len(certs) == 0 {
		return nil, errors.New("keine Zertifikate in der Datei gefunden")
	}
	return certs, nil
}

func loadPrivateKey(keyPath string) (crypto.PrivateKey, error) {
	data, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("kein gültiger PEM-Key")
	}

	switch block.Type {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(block.Bytes)

	case "EC PRIVATE KEY":
		return x509.ParseECPrivateKey(block.Bytes)

	case "PRIVATE KEY": // PKCS#8
		return x509.ParsePKCS8PrivateKey(block.Bytes)

	default:
		return nil, fmt.Errorf("unbekannter Key-Typ: %s", block.Type)
	}
}

func newSerial() *big.Int {
	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	return serial
}

func parseAndValidateCSR(data []byte) (*x509.CertificateRequest, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("CSR ist kein PEM")
	}

	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		return nil, err
	}

	if err := csr.CheckSignature(); err != nil {
		return nil, fmt.Errorf("ungültige CSR-Signatur: %w", err)
	}

	return csr, nil
}
