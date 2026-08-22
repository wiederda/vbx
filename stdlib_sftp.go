// ------------------------
// stdlib_sftp.go
// ------------------------

package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

type SftpConn struct {
	client  *sftp.Client
	sshConn *ssh.Client
}

var (
	sftpConnections = make(map[string]*SftpConn)
	sftpMu          sync.RWMutex
)

func InitSftpFunctions() {
	if builtins == nil {
		builtins = make(map[string]BuiltinInfo)
	}

	ns := "sftp."

	// ------------------------
	// sftp.Connect
	// ------------------------
	Register(ns+"Connect", "sftp", "host, user, password, alias, [knownHostsPath], [port]",
		"Öffnet eine SFTP-Verbindung per Passwort-Authentifizierung und speichert sie unter einem Alias. Mit knownHostsPath wird der Host-Key gegen eine known_hosts-Datei geprüft (empfohlen bei Verbindungen außerhalb des eigenen Netzes). port ist der SSH-Port (Default 22, nur bei Abweichung angeben).",
		func(args []Value) Value {
			if len(args) < 4 {
				return ErrorVal("sftp.Connect(host, user, password, alias [, knownHostsPath, port]) benötigt mindestens 4 Argumente")
			}

			host := args[0].Str
			user := args[1].Str
			password := args[2].Str
			alias := strings.ToLower(args[3].Str)

			port := 22
			if len(args) >= 6 {
				port = int(toNumVal(args[5]))
			}

			hostKeyCallback, errVal := buildHostKeyCallback(args, 4)
			if errVal != nil {
				return *errVal
			}

			config := &ssh.ClientConfig{
				User: user,
				Auth: []ssh.AuthMethod{
					ssh.Password(password),
				},
				HostKeyCallback: hostKeyCallback,
				Timeout:         10 * time.Second,
			}

			addr := fmt.Sprintf("%s:%d", host, port)
			sshConn, err := ssh.Dial("tcp", addr, config)
			if err != nil {
				return ErrorVal("SSH-Verbindung fehlgeschlagen: " + err.Error())
			}

			client, err := sftp.NewClient(sshConn)
			if err != nil {
				sshConn.Close()
				return ErrorVal("SFTP-Client konnte nicht erstellt werden: " + err.Error())
			}

			sftpMu.Lock()
			if old, ok := sftpConnections[alias]; ok {
				old.client.Close()
				old.sshConn.Close()
			}
			sftpConnections[alias] = &SftpConn{client: client, sshConn: sshConn}
			sftpMu.Unlock()

			return BoolVal(true)
		})

	// ------------------------
	// sftp.ConnectWithKey
	// ------------------------
	Register(ns+"ConnectWithKey", "sftp", "host, user, keyPath, alias, [knownHostsPath], [port]",
		"Öffnet eine SFTP-Verbindung per Private-Key-Authentifizierung (z.B. aus global.GenerateSSHKey) und speichert sie unter einem Alias. port ist der SSH-Port (Default 22, nur bei Abweichung angeben).",
		func(args []Value) Value {
			if len(args) < 4 {
				return ErrorVal("sftp.ConnectWithKey(host, user, keyPath, alias [, knownHostsPath, port]) benötigt mindestens 4 Argumente")
			}

			host := args[0].Str
			user := args[1].Str
			keyPath := args[2].Str
			alias := strings.ToLower(args[3].Str)

			port := 22
			if len(args) >= 6 {
				port = int(toNumVal(args[5]))
			}

			absKeyPath, errVal := absPathVal(keyPath)
			if errVal != nil {
				return *errVal
			}

			keyBytes, err := os.ReadFile(absKeyPath)
			if err != nil {
				return ErrorVal("Private Key konnte nicht gelesen werden: " + err.Error())
			}

			signer, err := ssh.ParsePrivateKey(keyBytes)
			if err != nil {
				return ErrorVal("Private Key konnte nicht geparst werden: " + err.Error())
			}

			hostKeyCallback, errVal := buildHostKeyCallback(args, 4)
			if errVal != nil {
				return *errVal
			}

			config := &ssh.ClientConfig{
				User: user,
				Auth: []ssh.AuthMethod{
					ssh.PublicKeys(signer),
				},
				HostKeyCallback: hostKeyCallback,
				Timeout:         10 * time.Second,
			}

			addr := fmt.Sprintf("%s:%d", host, port)
			sshConn, err := ssh.Dial("tcp", addr, config)
			if err != nil {
				return ErrorVal("SSH-Verbindung fehlgeschlagen: " + err.Error())
			}

			client, err := sftp.NewClient(sshConn)
			if err != nil {
				sshConn.Close()
				return ErrorVal("SFTP-Client konnte nicht erstellt werden: " + err.Error())
			}

			sftpMu.Lock()
			if old, ok := sftpConnections[alias]; ok {
				old.client.Close()
				old.sshConn.Close()
			}
			sftpConnections[alias] = &SftpConn{client: client, sshConn: sshConn}
			sftpMu.Unlock()

			return BoolVal(true)
		})

	// ------------------------
	// sftp.Close
	// ------------------------
	Register(ns+"Close", "sftp", "alias",
		"Schließt eine SFTP-Verbindung und entfernt den Alias.",
		func(args []Value) Value {
			if len(args) < 1 {
				return ErrorVal("sftp.Close(alias) benötigt einen Alias")
			}
			alias := strings.ToLower(args[0].Str)

			sftpMu.Lock()
			defer sftpMu.Unlock()

			conn, ok := sftpConnections[alias]
			if !ok {
				return BoolVal(false)
			}

			conn.client.Close()
			conn.sshConn.Close()
			delete(sftpConnections, alias)

			return BoolVal(true)
		})

	// ------------------------
	// sftp.Upload
	// ------------------------
	Register(ns+"Upload", "sftp", "alias, localPath, remotePath",
		"Lädt eine lokale Datei auf den SFTP-Server hoch.",
		func(args []Value) Value {
			if len(args) < 3 {
				return ErrorVal("sftp.Upload(alias, localPath, remotePath) benötigt 3 Argumente")
			}
			alias := strings.ToLower(args[0].Str)
			localPath := args[1].Str
			remotePath := args[2].Str

			conn, errVal := getSftpConn(alias)
			if errVal != nil {
				return *errVal
			}

			absLocal, errVal := absPathVal(localPath)
			if errVal != nil {
				return *errVal
			}

			localFile, err := os.Open(absLocal)
			if err != nil {
				return ErrorVal("Lokale Datei konnte nicht geöffnet werden: " + err.Error())
			}
			defer localFile.Close()

			remoteFile, err := conn.client.Create(remotePath)
			if err != nil {
				return ErrorVal("Remote-Datei konnte nicht erstellt werden: " + err.Error())
			}
			defer remoteFile.Close()

			written, err := io.Copy(remoteFile, localFile)
			if err != nil {
				return ErrorVal("Upload fehlgeschlagen: " + err.Error())
			}

			return NumVal(float64(written))
		})

	// ------------------------
	// sftp.Download
	// ------------------------
	Register(ns+"Download", "sftp", "alias, remotePath, localPath",
		"Lädt eine Datei vom SFTP-Server herunter.",
		func(args []Value) Value {
			if len(args) < 3 {
				return ErrorVal("sftp.Download(alias, remotePath, localPath) benötigt 3 Argumente")
			}
			alias := strings.ToLower(args[0].Str)
			remotePath := args[1].Str
			localPath := args[2].Str

			conn, errVal := getSftpConn(alias)
			if errVal != nil {
				return *errVal
			}

			absLocal, errVal := absPathVal(localPath)
			if errVal != nil {
				return *errVal
			}

			if _, err := os.Stat(filepath.Dir(absLocal)); err != nil {
				return ErrorVal("Zielverzeichnis existiert nicht: " + filepath.Dir(absLocal))
			}

			remoteFile, err := conn.client.Open(remotePath)
			if err != nil {
				return ErrorVal("Remote-Datei konnte nicht geöffnet werden: " + err.Error())
			}
			defer remoteFile.Close()

			localFile, err := os.Create(absLocal)
			if err != nil {
				return ErrorVal("Lokale Datei konnte nicht erstellt werden: " + err.Error())
			}
			defer localFile.Close()

			written, err := io.Copy(localFile, remoteFile)
			if err != nil {
				return ErrorVal("Download fehlgeschlagen: " + err.Error())
			}

			return NumVal(float64(written))
		})

	// ------------------------
	// sftp.DownloadFolder
	// ------------------------
	Register(ns+"DownloadFolder", "sftp", "alias, remotePath, localBasePath, [recursive], [progress]",
		"Lädt einen kompletten Remote-Ordner herunter. Legt lokal einen Ordner mit demselben Namen wie das letzte Segment von remotePath an (falls nicht vorhanden) und lädt alle enthaltenen Dateien hinein.",
		func(args []Value) Value {
			if len(args) < 3 {
				return ErrorVal("sftp.DownloadFolder(alias, remotePath, localBasePath [, recursive, progress]) benötigt mindestens 3 Argumente")
			}
			alias := strings.ToLower(args[0].Str)
			remotePath := strings.TrimSuffix(args[1].Str, "/")
			localBasePath := args[2].Str

			recursive := false
			if len(args) >= 4 {
				recursive = isTruthy(args[3])
			}

			showProgress := false
			if len(args) >= 5 {
				showProgress = isTruthy(args[4])
			}

			conn, errVal := getSftpConn(alias)
			if errVal != nil {
				return *errVal
			}

			absLocalBase, errVal := absPathVal(localBasePath)
			if errVal != nil {
				return *errVal
			}

			folderName := path.Base(remotePath)
			localTarget := filepath.Join(absLocalBase, folderName)

			if err := os.MkdirAll(localTarget, 0755); err != nil {
				return ErrorVal("Lokaler Zielordner konnte nicht angelegt werden: " + err.Error())
			}

			// Erst zählen, damit die Fortschrittsanzeige "X von Y" zeigen kann,
			// statt nur einen laufenden Zähler ohne Gesamtzahl.
			total := 0
			if showProgress {
				total, _ = countRemoteFiles(conn, remotePath, recursive)
			}

			current := 0
			count, err := downloadFolderRecursiveP(conn, remotePath, localTarget, recursive, showProgress, total, &current)
			if err != nil {
				return ErrorVal(err.Error())
			}

			return NumVal(float64(count))
		})

	// ------------------------
	// sftp.FindByExt
	// ------------------------
	Register(ns+"FindByExt", "sftp", "alias, remotePath, ext, [all]",
		"Sucht Dateien mit einer bestimmten Endung in einem Remote-Verzeichnis, ohne den genauen Dateinamen zu kennen.",
		func(args []Value) Value {
			if len(args) < 3 {
				return ErrorVal("sftp.FindByExt(alias, remotePath, ext [, all]) benötigt mindestens 3 Argumente")
			}
			alias := strings.ToLower(args[0].Str)
			remotePath := args[1].Str
			ext := strings.ToLower(strings.TrimPrefix(args[2].Str, "."))

			all := false
			if len(args) >= 4 {
				all = isTruthy(args[3])
			}

			conn, errVal := getSftpConn(alias)
			if errVal != nil {
				return *errVal
			}

			entries, err := conn.client.ReadDir(remotePath)
			if err != nil {
				return ErrorVal("Verzeichnis konnte nicht gelesen werden: " + err.Error())
			}

			var treffer []string
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				name := e.Name()
				fileExt := strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
				if fileExt == ext {
					treffer = append(treffer, remotePath+"/"+name)
				}
			}

			sort.SliceStable(treffer, func(i, j int) bool {
				return naturalLess(treffer[i], treffer[j])
			})

			if all {
				result := make([]Value, len(treffer))
				for i, t := range treffer {
					result[i] = StrVal(t)
				}
				return Value{Kind: KindArr, Arr: result}
			}

			if len(treffer) == 0 {
				return StrVal("")
			}
			return StrVal(treffer[0])
		})

	// ------------------------
	// sftp.DownloadByExt
	// ------------------------
	Register(ns+"DownloadByExt", "sftp", "alias, remotePath, ext, localBasePath, [all]",
		"Sucht Dateien mit einer bestimmten Endung in einem Remote-Verzeichnis und lädt sie direkt herunter, ohne die genauen Dateinamen zu kennen. Kombiniert sftp.FindByExt und sftp.Download in einem Aufruf.",
		func(args []Value) Value {
			if len(args) < 4 {
				return ErrorVal("sftp.DownloadByExt(alias, remotePath, ext, localBasePath [, all]) benötigt mindestens 4 Argumente")
			}
			alias := strings.ToLower(args[0].Str)
			remotePath := args[1].Str
			ext := strings.ToLower(strings.TrimPrefix(args[2].Str, "."))
			localBasePath := args[3].Str

			all := false
			if len(args) >= 5 {
				all = isTruthy(args[4])
			}

			conn, errVal := getSftpConn(alias)
			if errVal != nil {
				return *errVal
			}

			absLocalBase, errVal := absPathVal(localBasePath)
			if errVal != nil {
				return *errVal
			}

			if err := os.MkdirAll(absLocalBase, 0755); err != nil {
				return ErrorVal("Lokales Zielverzeichnis konnte nicht angelegt werden: " + err.Error())
			}

			entries, err := conn.client.ReadDir(remotePath)
			if err != nil {
				return ErrorVal("Verzeichnis konnte nicht gelesen werden: " + err.Error())
			}

			var treffer []string
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				fileExt := strings.ToLower(strings.TrimPrefix(filepath.Ext(e.Name()), "."))
				if fileExt == ext {
					treffer = append(treffer, e.Name())
				}
			}

			sort.SliceStable(treffer, func(i, j int) bool {
				return naturalLess(treffer[i], treffer[j])
			})

			if len(treffer) == 0 {
				return NumVal(0)
			}

			// Ohne 'all': nur die erste (natürlich sortierte) Datei laden
			if !all {
				treffer = treffer[:1]
			}

			downloaded := 0
			for _, name := range treffer {
				remoteFilePath := remotePath + "/" + name
				localFilePath := filepath.Join(absLocalBase, name)

				remoteFile, err := conn.client.Open(remoteFilePath)
				if err != nil {
					return ErrorVal(fmt.Sprintf("Remote-Datei konnte nicht geöffnet werden (%s): %s", name, err.Error()))
				}

				localFile, err := os.Create(localFilePath)
				if err != nil {
					remoteFile.Close()
					return ErrorVal(fmt.Sprintf("Lokale Datei konnte nicht erstellt werden (%s): %s", name, err.Error()))
				}

				_, copyErr := io.Copy(localFile, remoteFile)
				remoteFile.Close()
				localFile.Close()

				if copyErr != nil {
					return ErrorVal(fmt.Sprintf("Download fehlgeschlagen (%s): %s", name, copyErr.Error()))
				}

				downloaded++
			}

			return NumVal(float64(downloaded))
		})

	// ------------------------
	// sftp.List
	// ------------------------
	Register(ns+"List", "sftp", "alias, remotePath, [sortBy], [desc]",
		"Listet den Inhalt eines Remote-Verzeichnisses. Gibt ein Array von Maps zurück (name, size, isDir, modTime). Optional direkt sortiert.",
		func(args []Value) Value {
			if len(args) < 2 {
				return ErrorVal("sftp.List(alias, remotePath [, sortBy, desc]) benötigt Alias und Pfad")
			}
			alias := strings.ToLower(args[0].Str)
			remotePath := args[1].Str

			sortBy := ""
			if len(args) >= 3 {
				sortBy = args[2].Str
			}
			desc := false
			if len(args) >= 4 {
				desc = isTruthy(args[3])
			}

			conn, errVal := getSftpConn(alias)
			if errVal != nil {
				return *errVal
			}

			entries, err := conn.client.ReadDir(remotePath)
			if err != nil {
				return ErrorVal("Verzeichnis konnte nicht gelesen werden: " + err.Error())
			}

			if sortBy != "" {
				sort.SliceStable(entries, func(i, j int) bool {
					var less bool
					switch strings.ToLower(sortBy) {
					case "size":
						less = entries[i].Size() < entries[j].Size()
					case "modtime":
						less = entries[i].ModTime().Before(entries[j].ModTime())
					default: // "name"
						less = entries[i].Name() < entries[j].Name()
					}
					if desc {
						return !less
					}
					return less
				})
			}

			result := make([]Value, len(entries))
			for i, e := range entries {
				m := map[string]Value{
					"name":    StrVal(e.Name()),
					"size":    NumVal(float64(e.Size())),
					"isDir":   BoolVal(e.IsDir()),
					"modTime": StrVal(e.ModTime().Format(time.RFC3339)),
				}
				result[i] = Value{Kind: KindMap, Map: m}
			}

			return Value{Kind: KindArr, Arr: result}
		})
}

// getSftpConn holt eine bestehende Verbindung anhand des Alias.
func getSftpConn(alias string) (*SftpConn, *Value) {
	sftpMu.RLock()
	defer sftpMu.RUnlock()

	conn, ok := sftpConnections[alias]
	if !ok {
		v := ErrorVal("SFTP-Verbindung '" + alias + "' nicht gefunden. Zuerst sftp.Connect() aufrufen.")
		return nil, &v
	}
	return conn, nil
}

// buildHostKeyCallback liest optional einen knownHostsPath-Parameter aus args[idx].
// Ist er angegeben, wird echte Host-Key-Verifikation genutzt: bei einem
// bisher UNBEKANNTEN Host wird der Key automatisch zur known_hosts-Datei
// hinzugefügt (Trust-on-First-Use, wie ssh es beim ersten interaktiven
// Connect auch tut - nur ohne Nachfrage). Bei einem Host, dessen Key sich
// GEÄNDERT hat (möglicher Angriff oder Server wurde neu aufgesetzt), wird
// weiterhin abgelehnt - das automatische Hinzufügen gilt NUR für neue,
// bisher nicht verzeichnete Hosts.
// Ist knownHostsPath leer/nicht angegeben, bleibt InsecureIgnoreHostKey
// der Standard (unverändertes Verhalten für den lokalen Netzbetrieb).
func buildHostKeyCallback(args []Value, idx int) (ssh.HostKeyCallback, *Value) {
	if len(args) <= idx || args[idx].Str == "" {
		return ssh.InsecureIgnoreHostKey(), nil
	}

	knownHostsPath, errVal := absPathVal(args[idx].Str)
	if errVal != nil {
		return nil, errVal
	}

	if _, err := os.Stat(knownHostsPath); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(knownHostsPath), 0700); err != nil {
			v := ErrorVal("known_hosts-Verzeichnis konnte nicht erstellt werden: " + err.Error())
			return nil, &v
		}
		f, err := os.OpenFile(knownHostsPath, os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			v := ErrorVal("known_hosts-Datei konnte nicht erstellt werden: " + err.Error())
			return nil, &v
		}
		f.Close()
	}

	baseCallback, err := knownhosts.New(knownHostsPath)
	if err != nil {
		v := ErrorVal("known_hosts-Datei konnte nicht geladen werden: " + err.Error())
		return nil, &v
	}

	autoAddCallback := ssh.HostKeyCallback(func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := baseCallback(hostname, remote, key)
		if err == nil {
			return nil
		}

		var keyErr *knownhosts.KeyError
		if errors.As(err, &keyErr) {
			if len(keyErr.Want) == 0 {
				// Backup anlegen, bevor die Datei verändert wird - analog
				// zum .old-Muster, das an anderer Stelle im Projekt bereits
				// vor Schreibvorgängen genutzt wird.
				if backupErr := backupFileOld(knownHostsPath); backupErr != nil {
					return fmt.Errorf("Backup von known_hosts fehlgeschlagen: %w", backupErr)
				}

				line := knownhosts.Line([]string{knownhosts.Normalize(hostname)}, key)

				f, openErr := os.OpenFile(knownHostsPath, os.O_APPEND|os.O_WRONLY, 0600)
				if openErr != nil {
					return fmt.Errorf("known_hosts konnte nicht zum Schreiben geöffnet werden: %w", openErr)
				}
				defer f.Close()

				if _, writeErr := f.WriteString(line + "\n"); writeErr != nil {
					return fmt.Errorf("Host-Key konnte nicht gespeichert werden: %w", writeErr)
				}

				return nil
			}
		}

		return err
	})

	return autoAddCallback, nil
}

// backupFileOld kopiert eine bestehende Datei nach <path>.old, bevor sie
// verändert wird. Überschreibt eine evtl. vorhandene .old-Datei, damit sie
// immer den Stand direkt vor der letzten Änderung zeigt. Existiert die
// Originaldatei (noch) nicht oder ist leer, wird kein Backup angelegt.
func backupFileOld(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.Size() == 0 {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return os.WriteFile(path+".old", data, 0600)
}
