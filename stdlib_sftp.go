// ------------------------
// stdlib_sftp.go
// ------------------------

package main

import (
	"fmt"
	"io"
	"os"
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
	Register(ns+"Connect", "sftp", "host, port, user, password, alias, [knownHostsPath]",
		"Öffnet eine SFTP-Verbindung per Passwort-Authentifizierung und speichert sie unter einem Alias. Mit knownHostsPath wird der Host-Key gegen eine known_hosts-Datei geprüft (empfohlen bei Verbindungen außerhalb des eigenen Netzes).",
		func(args []Value) Value {
			if len(args) < 5 {
				return ErrorVal("sftp.Connect(host, port, user, password, alias [, knownHostsPath]) benötigt mindestens 5 Argumente")
			}

			host := args[0].Str
			port := int(toNumVal(args[1]))
			user := args[2].Str
			password := args[3].Str
			alias := strings.ToLower(args[4].Str)

			hostKeyCallback, errVal := buildHostKeyCallback(args, 5)
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
	Register(ns+"ConnectWithKey", "sftp", "host, port, user, keyPath, alias",
		"Öffnet eine SFTP-Verbindung per Private-Key-Authentifizierung (z.B. aus global.GenerateSSHKey) und speichert sie unter einem Alias.",
		func(args []Value) Value {
			if len(args) < 5 {
				return ErrorVal("sftp.ConnectWithKey(host, port, user, keyPath, alias) benötigt 5 Argumente")
			}

			host := args[0].Str
			port := int(toNumVal(args[1]))
			user := args[2].Str
			keyPath := args[3].Str
			alias := strings.ToLower(args[4].Str)

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

			config := &ssh.ClientConfig{
				User: user,
				Auth: []ssh.AuthMethod{
					ssh.PublicKeys(signer),
				},
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
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
	// sftp.List
	// ------------------------
	Register(ns+"List", "sftp", "alias, remotePath",
		"Listet den Inhalt eines Remote-Verzeichnisses. Gibt ein Array von Maps zurück (name, size, isDir, modTime).",
		func(args []Value) Value {
			if len(args) < 2 {
				return ErrorVal("sftp.List(alias, remotePath) benötigt Alias und Pfad")
			}
			alias := strings.ToLower(args[0].Str)
			remotePath := args[1].Str

			conn, errVal := getSftpConn(alias)
			if errVal != nil {
				return *errVal
			}

			entries, err := conn.client.ReadDir(remotePath)
			if err != nil {
				return ErrorVal("Verzeichnis konnte nicht gelesen werden: " + err.Error())
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
// Ist er angegeben, wird echte Host-Key-Verifikation genutzt (via
// golang.org/x/crypto/ssh/knownhosts). Ist er leer/nicht angegeben, wird
// weiterhin InsecureIgnoreHostKey verwendet - das bleibt der Standard, um
// bestehende Skripte nicht zu brechen, ist aber NICHT empfohlen für
// Verbindungen außerhalb des eigenen, vertrauten Netzes.
func buildHostKeyCallback(args []Value, idx int) (ssh.HostKeyCallback, *Value) {
	if len(args) <= idx || args[idx].Str == "" {
		return ssh.InsecureIgnoreHostKey(), nil
	}

	knownHostsPath, errVal := absPathVal(args[idx].Str)
	if errVal != nil {
		return nil, errVal
	}

	callback, err := knownhosts.New(knownHostsPath)
	if err != nil {
		v := ErrorVal("known_hosts-Datei konnte nicht geladen werden: " + err.Error())
		return nil, &v
	}

	return callback, nil
}
