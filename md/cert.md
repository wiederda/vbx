# cert.* – Zertifikat-Modul

Das Modul `cert` stellt Funktionen zur PKI-Verwaltung bereit: Schlüsselerzeugung, CSR-Erstellung, Zertifikat-Signierung sowie Export in verschiedene Formate (PEM, DER, PFX, PKCS#7).

---

### `cert.GenerateKey(outFile, algo, bits)`

Erzeugt einen privaten Schlüssel im PKCS#8-Format.

| Parameter | Typ | Beschreibung |
|-----------|-----|--------------|
| `outFile` | String | Zieldatei für den Key (PEM) |
| `algo` | String | Algorithmus: `rsa` oder `ecdsa` |
| `bits` | String/Int | RSA: Mindestens 4096. ECDSA: 256, 384, 521 (Standard: 384) |

**Rückgabe:** `Bool` – `true` bei Erfolg.

**Hinweise:**
- RSA-Schlüssel unter 4096 Bit werden automatisch auf 4096 angehoben.
- Die Datei wird mit Berechtigung `0600` gespeichert.

---

### `cert.CreateCSR(subject, keyPath, outFile [, SANs])`

Erstellt eine Certificate Signing Request (CSR).

| Parameter | Typ | Beschreibung |
|-----------|-----|--------------|
| `subject` | String | Common Name (CN) |
| `keyPath` | String | Pfad zum privaten Schlüssel |
| `outFile` | String | Ausgabedatei (PEM) |
| `SANs` | String (optional) | Kommagetrennte DNS-Namen oder IP-Adressen |

**Rückgabe:** `Bool`

---

### `cert.CreateCSRConf(confPath, keyPath, outCSR)`

Erstellt eine CSR auf Basis einer OpenSSL-Konfigurationsdatei. Liest `CN` und `DNS.*`-Einträge aus der Datei.

| Parameter | Typ | Beschreibung |
|-----------|-----|--------------|
| `confPath` | String | Pfad zur `.conf`-Datei |
| `keyPath` | String | Pfad zum privaten Schlüssel |
| `outCSR` | String | Ausgabedatei (PEM) |

**Rückgabe:** `Bool`

---

### `cert.CreateSelfSigned(subject, keyPath, outCert [, days, SANs, isCA])`

Erstellt ein selbstsigniertes Zertifikat.

| Parameter | Typ | Beschreibung |
|-----------|-----|--------------|
| `subject` | String | Common Name (CN) |
| `keyPath` | String | Pfad zum privaten Schlüssel |
| `outCert` | String | Ausgabedatei (PEM) |
| `days` | Int (optional) | Gültigkeitsdauer in Tagen (Standard: 365) |
| `SANs` | String (optional) | Kommagetrennte DNS-Namen oder IPs |
| `isCA` | String (optional) | `"true"` für CA-Zertifikat |

**Rückgabe:** `Null` bei Erfolg, `Error` bei Fehler.

**Hinweise:**
- Unterstützte Key-Typen: RSA (`RSA PRIVATE KEY`), ECDSA (`EC PRIVATE KEY`).
- PKCS#8-Keys (`PRIVATE KEY`) werden **nicht** unterstützt – dafür `cert.GenerateKey` verwenden und den erzeugten Key direkt nutzen.

---

### `cert.SignCSR(csrPath, caCert, caKey, outCert [, days])`

Signiert eine CSR mit einer CA und stellt ein gültiges Zertifikat aus.

| Parameter | Typ | Beschreibung |
|-----------|-----|--------------|
| `csrPath` | String | Pfad zur CSR-Datei |
| `caCert` | String | Pfad zum CA-Zertifikat |
| `caKey` | String | Pfad zum CA-Schlüssel |
| `outCert` | String | Ausgabedatei (PEM) |
| `days` | Int (optional) | Gültigkeitsdauer in Tagen (Standard: 365) |

**Rückgabe:** `Bool`

**Hinweise:**
- Die CSR-Signatur wird vor der Verarbeitung validiert.
- Das ausgestellte Zertifikat erhält `ExtKeyUsageServerAuth`.

---

### `cert.ExportPEM(certPath, outFile)`

Kopiert ein Zertifikat unverändert als PEM-Datei.

| Parameter | Typ | Beschreibung |
|-----------|-----|--------------|
| `certPath` | String | Quelldatei |
| `outFile` | String | Zieldatei |

**Rückgabe:** `Null` bei Erfolg, `Error` bei Fehler.

---

### `cert.ExportDER(certPath, outFile)`

Konvertiert ein PEM-Zertifikat in das binäre DER-Format.

| Parameter | Typ | Beschreibung |
|-----------|-----|--------------|
| `certPath` | String | PEM-Quelldatei |
| `outFile` | String | DER-Ausgabedatei |

**Rückgabe:** `Null` bei Erfolg, `Error` bei Fehler.

**Hinweise:**
- Nur PEM-Blöcke vom Typ `CERTIFICATE` werden akzeptiert. Private Keys werden explizit abgelehnt.

---

### `cert.GetPublicKey(certPath)`

Extrahiert den öffentlichen Schlüssel aus einem Zertifikat.

| Parameter | Typ | Beschreibung |
|-----------|-----|--------------|
| `certPath` | String | Pfad zum Zertifikat (PEM) |

**Rückgabe:** `String` – öffentlicher Schlüssel im PEM-Format (`PUBLIC KEY`).

---

### `cert.ExportPFX(certPath, keyPath, outFile, password [, friendlyName, mode])`

Exportiert Zertifikat und Key in einen PKCS#12-Container (`.pfx`/`.p12`).

| Parameter | Typ | Beschreibung |
|-----------|-----|--------------|
| `certPath` | String | PEM-Datei (kann Zertifikatskette enthalten) |
| `keyPath` | String | Pfad zum privaten Schlüssel |
| `outFile` | String | Ausgabedatei |
| `password` | String | Passwort für den Container |
| `friendlyName` | String (optional) | Anzeigename (wird geloggt, nicht eingebettet) |
| `mode` | String (optional) | `"legacy"` (Standard, RC2) oder `"modern"` |

**Rückgabe:** `Bool`

**Hinweise:**
- Das erste Zertifikat in der PEM-Datei wird als Leaf-Zertifikat verwendet, alle weiteren als Chain.

---

### `cert.ExportPKCS7(certPath, outFile)`

Erstellt einen PKCS#7-Container aus einem oder mehreren Zertifikaten.

| Parameter | Typ | Beschreibung |
|-----------|-----|--------------|
| `certPath` | String | Quelldatei (PEM, auch Bundles) |
| `outFile` | String | Ausgabedatei (binär) |

**Rückgabe:** `Bool`

---

### `cert.Combine(cert1, ..., outFile)`

Kombiniert mehrere Zertifikate zu einer PEM-Kette.

| Parameter | Typ | Beschreibung |
|-----------|-----|--------------|
| `cert1, ...` | String | Beliebig viele Zertifikatspfade (PEM oder DER) |
| `outFile` | String | Ausgabedatei (letztes Argument) |

**Rückgabe:** `Bool`

**Hinweise:**
- DER-Dateien werden automatisch erkannt und konvertiert.

---

### `cert.CreateConf([cn, dnsArray, outFile])`

Erstellt eine OpenSSL-Konfigurationsdatei mit SAN-Einträgen.

**Ohne Argumente:** Interaktiver Modus mit Eingabe-Prompts.

| Parameter | Typ | Beschreibung |
|-----------|-----|--------------|
| `cn` | String (optional) | Common Name |
| `dnsArray` | Array (optional) | Zusätzliche DNS-Namen und/oder IP-Adressen |
| `outFile` | String (optional) | Zielordner oder Dateiname |

**Rückgabe:** `Bool`

**Hinweise:**
- Ist `outFile` ein Ordner (oder leer), wird `<CN>.conf` als Dateiname verwendet.
- Existiert die Zieldatei bereits, wird eine Überschreib-Bestätigung eingeholt.

---

## Interne Hilfsfunktionen

| Funktion | Beschreibung |
|----------|--------------|
| `loadPrivateKey(path)` | Lädt RSA-, EC- und PKCS#8-Keys aus PEM |
| `loadAllCerts(path)` | Liest alle `CERTIFICATE`-Blöcke aus einer PEM-Datei |
| `parseAndValidateCSR(data)` | Dekodiert und validiert eine CSR inkl. Signaturprüfung |
| `newSerial()` | Erzeugt eine kryptografisch sichere 128-Bit-Seriennummer |
| `ensureDir(path)` | Erstellt übergeordnete Verzeichnisse falls nötig |

---

## Abhängigkeiten

| Paket | Verwendung |
|-------|------------|
| `github.com/fullsailor/pkcs7` | PKCS#7-Export |
| `software.sslmate.com/src/go-pkcs12` | PFX/PKCS#12-Export |
| Go Standardbibliothek (`crypto/x509`, `crypto/rsa`, `crypto/ecdsa`, …) | Kern-PKI-Operationen |