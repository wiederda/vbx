// ------------------------
// vbx_acme.go
// ------------------------

package main

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
	"time"
)

var LEStaging = "https://acme-staging-v02.api.letsencrypt.org/directory"
var LEProduction = "https://acme-v02.api.letsencrypt.org/directory"

// InitAcmeFunctions initialisiert den acme.-Namespace.
func InitAcmeFunctions() {
	if builtins == nil {
		builtins = make(map[string]BuiltinInfo)
	}

	ns := "acme."

	// ---------------------------------------------------------------
	// Account
	// ---------------------------------------------------------------

	Register(ns+"Account", "acme", "directory, keyPath, statePath, email, acceptTOS",
		"Erstellt oder lädt ein ACME-Konto und gibt die Account-URL zurück.",
		func(args []Value) Value {
			if len(args) < 5 {
				return ErrorVal(
					"acme.Account(directory, keyPath, statePath, email, acceptTOS) erwartet 5 Parameter",
				)
			}

			directory := ResolveACMEDirectory(strings.TrimSpace(args[0].Str))
			keyPath := strings.TrimSpace(args[1].Str)
			statePath := strings.TrimSpace(args[2].Str)
			email := strings.TrimSpace(args[3].Str)
			acceptTOS := ToBool(args[4])

			if directory == "" {
				return ErrorVal("ACME Directory darf nicht leer sein")
			}

			if keyPath == "" {
				return ErrorVal("Account-Key-Pfad darf nicht leer sein")
			}

			if statePath == "" {
				return ErrorVal("Account-State-Pfad darf nicht leer sein")
			}

			if email == "" {
				return ErrorVal("E-Mail-Adresse darf nicht leer sein")
			}

			if !acceptTOS {
				return ErrorVal("Die ACME-Nutzungsbedingungen müssen bestätigt werden")
			}

			ctx := context.Background()

			client, err := NewACMEClient(ctx, directory, keyPath)
			if err != nil {
				return ErrorVal("ACME Client konnte nicht erstellt werden: " + err.Error())
			}

			if err := client.LoadDirectory(ctx); err != nil {
				return ErrorVal("ACME Directory konnte nicht geladen werden: " + err.Error())
			}

			account, err := LoadOrCreateAccount(
				ctx,
				client,
				keyPath,
				statePath,
				email,
				acceptTOS,
			)
			if err != nil {
				return ErrorVal(
					"ACME Account konnte nicht geladen/erstellt werden: " + err.Error(),
				)
			}

			return StrVal(account.URL)
		})

	// ---------------------------------------------------------------
	// Issue
	// ---------------------------------------------------------------

	Register(ns+"Issue", "acme",
		"directory, keyPath, statePath, email, acceptTOS, domain, csrPath, outputPath, provider, token, [dnsServers]",
		"Fordert vollautomatisch ein Zertifikat per ACME DNS-01 an: lädt/erstellt das ACME-Konto, "+
			"setzt den DNS-TXT-Eintrag, wartet auf Validierung und speichert das Zertifikat. Gibt True bei Erfolg zurück.",
		func(args []Value) Value {
			if len(args) < 10 {
				return ErrorVal(
					"acme.Issue(directory, keyPath, statePath, email, acceptTOS, domain, csrPath, outputPath, provider, token, [dnsServers]) erwartet mindestens 10 Parameter",
				)
			}

			directory := ResolveACMEDirectory(strings.TrimSpace(args[0].Str))
			keyPath := strings.TrimSpace(args[1].Str)
			statePath := strings.TrimSpace(args[2].Str)
			email := strings.TrimSpace(args[3].Str)
			acceptTOS := ToBool(args[4])
			domain := strings.TrimSpace(args[5].Str)
			csrPath := strings.TrimSpace(args[6].Str)
			outputPath := strings.TrimSpace(args[7].Str)
			providerName := strings.TrimSpace(args[8].Str)
			token := strings.TrimSpace(args[9].Str)

			dnsServers := ""
			if len(args) > 10 {
				dnsServers = strings.TrimSpace(args[10].Str)
			}

			if directory == "" {
				return ErrorVal("ACME Directory darf nicht leer sein")
			}
			if keyPath == "" {
				return ErrorVal("Account-Key-Pfad darf nicht leer sein")
			}
			if statePath == "" {
				return ErrorVal("Account-State-Pfad darf nicht leer sein")
			}
			if email == "" {
				return ErrorVal("E-Mail-Adresse darf nicht leer sein")
			}
			if !acceptTOS {
				return ErrorVal("Die ACME-Nutzungsbedingungen müssen bestätigt werden")
			}
			if domain == "" {
				return ErrorVal("Domain darf nicht leer sein")
			}
			if csrPath == "" {
				return ErrorVal("CSR-Pfad darf nicht leer sein")
			}
			if outputPath == "" {
				return ErrorVal("Ausgabepfad darf nicht leer sein")
			}
			if providerName == "" {
				return ErrorVal("DNS-Provider darf nicht leer sein")
			}
			if token == "" {
				return ErrorVal("API-Token des DNS-Providers darf nicht leer sein")
			}

			// -------------------------------------------------------
			// CSR laden
			// -------------------------------------------------------

			csrData, err := os.ReadFile(csrPath)
			if err != nil {
				return ErrorVal(fmt.Sprintf("CSR konnte nicht gelesen werden: %v", err))
			}

			csrDER := csrData

			if block, _ := pem.Decode(csrData); block != nil {
				if block.Type != "CERTIFICATE REQUEST" &&
					block.Type != "NEW CERTIFICATE REQUEST" {
					return ErrorVal("CSR-Datei enthält keinen CERTIFICATE REQUEST Block")
				}
				csrDER = block.Bytes
			}

			if _, err := x509.ParseCertificateRequest(csrDER); err != nil {
				return ErrorVal(fmt.Sprintf("CSR ist ungültig: %v", err))
			}

			// -------------------------------------------------------
			// DNS Provider
			// -------------------------------------------------------

			providerConfig := map[string]string{"token": token}

			provider, ok := GetDNSProvider(providerName, providerConfig)
			if !ok {
				return ErrorVal(fmt.Sprintf("Unbekannter DNS-Provider: %s", providerName))
			}

			config := provider.Config()

			if !config.SupportsTXT {
				return ErrorVal(fmt.Sprintf("DNS-Provider %q unterstützt keine TXT-Einträge", providerName))
			}
			if !config.SupportsDelete {
				return ErrorVal(fmt.Sprintf("DNS-Provider %q unterstützt das Löschen von TXT-Einträgen nicht", providerName))
			}

			// -------------------------------------------------------
			// ACME Client + Account
			// -------------------------------------------------------

			ctx := context.Background()

			client, err := NewACMEClient(ctx, directory, keyPath)
			if err != nil {
				return ErrorVal("ACME Client konnte nicht erstellt werden: " + err.Error())
			}

			if err := client.LoadDirectory(ctx); err != nil {
				return ErrorVal("ACME Directory konnte nicht geladen werden: " + err.Error())
			}

			account, err := LoadOrCreateAccount(ctx, client, keyPath, statePath, email, acceptTOS)
			if err != nil {
				return ErrorVal("ACME Account konnte nicht geladen/erstellt werden: " + err.Error())
			}

			client.AccountURL = account.URL

			// -------------------------------------------------------
			// Order + Authorization
			// -------------------------------------------------------

			order, err := client.CreateOrder(ctx, []string{domain})
			if err != nil {
				return ErrorVal("ACME Order konnte nicht erstellt werden: " + err.Error())
			}

			if len(order.Authorizations) == 0 {
				return ErrorVal("ACME Order enthält keine Authorization")
			}

			authz, err := client.GetAuthorization(ctx, order.Authorizations[0])
			if err != nil {
				return ErrorVal(err.Error())
			}

			challenge, err := FindDNS01Challenge(authz)
			if err != nil {
				return ErrorVal(err.Error())
			}

			txtValue, err := client.DNS01Value(challenge.Token)
			if err != nil {
				return ErrorVal(err.Error())
			}

			fqdn := DNS01FQDN(domain)

			// -------------------------------------------------------
			// TXT setzen, validieren, aufräumen
			// -------------------------------------------------------

			if err := provider.Present(ctx, fqdn, txtValue); err != nil {
				return ErrorVal("TXT-Eintrag konnte nicht gesetzt werden: " + err.Error())
			}

			cleanup := true
			defer func() {
				if cleanup {
					_ = provider.Cleanup(context.Background(), fqdn, txtValue)
				}
			}()

			resolvers := ParseDNSResolvers(dnsServers)

			if err := WaitForTXT(ctx, fqdn, txtValue, resolvers, 120*time.Second, 5*time.Second); err != nil {
				return ErrorVal("DNS-01 TXT-Eintrag ist nicht öffentlich erreichbar: " + err.Error())
			}

			if err := client.AcceptChallenge(ctx, challenge.URL); err != nil {
				return ErrorVal(err.Error())
			}

			if err := client.WaitAuthorization(ctx, authz.URL); err != nil {
				return ErrorVal(err.Error())
			}

			if err := provider.Cleanup(ctx, fqdn, txtValue); err != nil {
				return ErrorVal("TXT-Eintrag konnte nach erfolgreicher Validierung nicht gelöscht werden: " + err.Error())
			}
			cleanup = false

			// -------------------------------------------------------
			// Finalisieren + Zertifikat abrufen
			// -------------------------------------------------------

			if err := client.FinalizeOrder(ctx, order.FinalizeURL, csrDER); err != nil {
				return ErrorVal(err.Error())
			}

			finalOrder, err := client.WaitOrder(ctx, order.URL)
			if err != nil {
				return ErrorVal(err.Error())
			}

			if finalOrder.CertificateURL == "" {
				return ErrorVal("ACME Order wurde abgeschlossen, enthält aber keine Certificate URL")
			}

			certificate, err := client.FetchCertificate(ctx, finalOrder.CertificateURL)
			if err != nil {
				return ErrorVal(err.Error())
			}

			if err := WriteCertificate(outputPath, certificate); err != nil {
				return ErrorVal(err.Error())
			}

			return BoolVal(true)
		})
}
