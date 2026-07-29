// ------------------------
// script_sanity.go
// ------------------------
//
// Interner Helfer für runFileInternal (main_run.go). Keine eigene VBX-Builtin-
// Funktion, keine Registrierung, kein Modul - wird automatisch von RunFile /
// CanExecuteFile mitgenutzt.

package main

import (
	"bytes"
	"strings"
	"unicode/utf8"
)

// looksLikeScript prüft einen Byte-Sample auf typische Anzeichen dafür, dass es sich
// NICHT um ein Textskript handelt (z.B. HTML-Fehlerseite, GitHub/GitLab-API-JSON-
// Antwort, falsches Encoding). Dient als schneller Vorfilter mit spezifischer
// Fehlermeldung, bevor der eigentliche Parser bemüht wird.
func looksLikeScript(sample []byte) (reason string, ok bool) {
	// Nur die ersten paar KB betrachten - reicht für alle Heuristiken
	const sniffLimit = 8192
	if len(sample) > sniffLimit {
		sample = sample[:sniffLimit]
	}

	// 1. Muss gültiges UTF-8 sein (reines ASCII ist eine Teilmenge davon)
	if !utf8.Valid(sample) {
		return "Datei ist kein gültiges UTF-8 - vermutlich falsches Encoding oder beschädigter Download", false
	}

	// 2. Bekannte "das ist eindeutig kein Skript"-Signaturen/Präfixe
	trimmed := bytes.TrimLeft(sample, " \t\r\n\ufeff")
	lowerPrefix := strings.ToLower(string(trimmed))
	if len(lowerPrefix) > 512 {
		lowerPrefix = lowerPrefix[:512]
	}

	badPrefixes := []string{
		"<!doctype",
		"<html",
		"<?xml",
		"%pdf-",
	}
	for _, p := range badPrefixes {
		if strings.HasPrefix(lowerPrefix, p) {
			return "Inhalt sieht nach HTML/XML/PDF aus, nicht nach einem Skript " +
				"(typisch bei fehlgeschlagenem Git-Raw-Download, z.B. Fehlerseite statt Rohdatei)", false
		}
	}

	// 3. Spezialfall: GitHub-/GitLab-"Contents"-API-Antwort statt Rohdatei.
	// Erkennbar als JSON-Objekt mit den typischen Metadaten-Feldern dieses
	// Endpoints (der eigentliche Skriptinhalt steckt dann Base64-kodiert im
	// "content"-Feld). Häufigste Ursache: net.Download wurde mit der
	// api.github.com/.../contents/...-URL aufgerufen statt mit der im JSON
	// enthaltenen "download_url" (raw.githubusercontent.com/...).
	if strings.HasPrefix(lowerPrefix, "{") {
		full := strings.ToLower(string(sample))
		if strings.Contains(full, `"encoding":"base64"`) &&
			strings.Contains(full, `"content":`) &&
			strings.Contains(full, `"download_url"`) {
			return "Inhalt ist eine GitHub/GitLab-API-JSON-Antwort (contents-Endpoint), keine Skriptdatei - " +
				"vermutlich wurde die API-URL statt der 'download_url' (Raw-URL) aus dem JSON für den Download verwendet", false
		}
	}

	return "", true
}
