// ------------------------
// stdlib_git.go
// ------------------------

package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

// =============================================================================
// Git-Funktionen
// =============================================================================

// registerGitFuncs registriert alle git.* Funktionen im VBX-Namespace.
// Basiert auf go-git (reines Go, kein externes Git-Binary nötig).
func InitGitFunctions() {

	if builtins == nil {
		builtins = make(map[string]BuiltinInfo)
	}

	ns := "git."

	// ---------------------------------------------------------------
	// Clone / CloneWithKey
	// ---------------------------------------------------------------

	Register(ns+"Clone", "git", "url [, path]",
		"Klont ein Git-Repository ohne Authentifizierung.\n\n"+
			"Das Zielverzeichnis darf noch nicht existieren. Wird path weggelassen, wird der\n"+
			"Verzeichnisname wie bei 'git clone' aus dem letzten Pfadsegment der URL abgeleitet\n"+
			"(ohne '.git'-Endung), relativ zum aktuellen Arbeitsverzeichnis. Der Pfad wird\n"+
			"anschließend über pathutils absolut aufgelöst.\n"+
			"Parameter: url (String) - Repository-URL; path (String, optional) - Zielverzeichnis\n"+
			"Rückgabe: Bool - true bei Erfolg",
		func(args []Value) Value {

			if len(args) < 1 {
				return ErrorVal("git.Clone: Erwartet mindestens url.")
			}

			url := args[0].Str

			rawPath := ""
			if len(args) >= 2 {
				rawPath = args[1].Str
			}

			path, errVal := cloneTargetPath(rawPath, url, "Clone")
			if errVal != nil {
				return *errVal
			}

			_, err := git.PlainClone(path, false, &git.CloneOptions{
				URL: url,
			})
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.Clone: Klonen von %q nach %q fehlgeschlagen: %v",
					url, path, err))
			}

			return Value{
				Kind: KindBool,
				Bool: true,
			}
		})

	Register(ns+"CloneWithToken", "git", "url, token [, path] [, username]",
		"Klont ein Git-Repository per HTTPS mit Token-Authentifizierung.\n\n"+
			"Das Zielverzeichnis darf noch nicht existieren. Wird path weggelassen, wird der\n"+
			"Verzeichnisname wie bei 'git clone' aus dem letzten Pfadsegment der URL abgeleitet\n"+
			"(ohne '.git'-Endung), relativ zum aktuellen Arbeitsverzeichnis. Der Pfad wird\n"+
			"anschließend über pathutils absolut aufgelöst.\n"+
			"Der Username ist optional. Wenn kein Username angegeben wird, wird\n"+
			"'x-access-token' verwendet.\n"+
			"Parameter: url (String) - HTTPS-Repository-URL; token (String) - Zugriffstoken;\n"+
			"path (String, optional) - Zielverzeichnis; username (String, optional) - Benutzername\n"+
			"für Basic-Auth\n"+
			"Rückgabe: Bool - true bei Erfolg",
		func(args []Value) Value {

			if len(args) < 2 {
				return ErrorVal("git.CloneWithToken: Erwartet mindestens url und token.")
			}

			url := args[0].Str
			token := args[1].Str

			rawPath := ""
			if len(args) >= 3 {
				rawPath = args[2].Str
			}

			path, errVal := cloneTargetPath(rawPath, url, "CloneWithToken")
			if errVal != nil {
				return *errVal
			}

			username := "x-access-token"

			if len(args) >= 4 {
				if args[3].Kind != KindStr {
					return ErrorVal(
						"git.CloneWithToken: username muss ein String sein.")
				}

				if args[3].Str != "" {
					username = args[3].Str
				}
			}

			_, err := git.PlainClone(path, false, &git.CloneOptions{
				URL: url,
				Auth: &http.BasicAuth{
					Username: username,
					Password: token,
				},
			})

			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.CloneWithToken: Klonen von %q nach %q fehlgeschlagen: %v",
					url, path, err))
			}

			return Value{
				Kind: KindBool,
				Bool: true,
			}
		})

	Register(ns+"CloneWithKey", "git", "url, keyPath [, path] [, knownHostsPath]",
		"Klont ein Git-Repository per SSH mit Schlüssel-Authentifizierung.\n\n"+
			"Das Zielverzeichnis darf noch nicht existieren. Wird path weggelassen, wird der\n"+
			"Verzeichnisname wie bei 'git clone' aus dem letzten Pfadsegment der URL abgeleitet\n"+
			"(ohne '.git'-Endung), relativ zum aktuellen Arbeitsverzeichnis. Ziel- und\n"+
			"Schlüsselpfad werden über pathutils absolut aufgelöst.\n"+
			"Konkret: Nutzt einen privaten SSH-Key statt Passwort oder Agent. Der Host-Key wird\n"+
			"wie bei sftp.ConnectWithKey / ssh.RebootWithKey behandelt: Mit knownHostsPath wird\n"+
			"gegen eine known_hosts-Datei geprüft (TOFU - unbekannte Hosts werden beim ersten\n"+
			"Kontakt automatisch eingetragen). Ohne knownHostsPath wird der Host-Key ungeprüft\n"+
			"akzeptiert.\n"+
			"Parameter: url (String) - SSH-URL (git@host:user/repo.git); keyPath (String) - Pfad\n"+
			"zum privaten SSH-Schlüssel; path (String, optional) - Zielverzeichnis;\n"+
			"knownHostsPath (String, optional) - Pfad zur known_hosts-Datei\n"+
			"Rückgabe: Bool - true bei Erfolg",
		func(args []Value) Value {

			if len(args) < 2 {
				return ErrorVal(
					"git.CloneWithKey: Erwartet mindestens url und keyPath.")
			}

			url := args[0].Str

			keyPath, errVal := absPathVal(args[1].Str)
			if errVal != nil {
				return *errVal
			}

			rawPath := ""
			if len(args) >= 3 {
				rawPath = args[2].Str
			}

			path, errVal := cloneTargetPath(rawPath, url, "CloneWithKey")
			if errVal != nil {
				return *errVal
			}

			hostKeyCallback, errVal := buildHostKeyCallback(args, 3)
			if errVal != nil {
				return *errVal
			}

			auth, err := gitssh.NewPublicKeysFromFile(
				"git",
				keyPath,
				"",
			)
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.CloneWithKey: Key %q konnte nicht geladen werden: %v",
					keyPath, err))
			}
			auth.HostKeyCallback = hostKeyCallback

			_, err = git.PlainClone(path, false, &git.CloneOptions{
				URL:  url,
				Auth: auth,
			})
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.CloneWithKey: Klonen von %q nach %q fehlgeschlagen: %v",
					url, path, err))
			}

			return Value{
				Kind: KindBool,
				Bool: true,
			}
		})

	// ---------------------------------------------------------------
	// Pull / Push / Fetch
	// ---------------------------------------------------------------

	Register(ns+"Pull", "git", "[path] [, keyPath] [, knownHostsPath] [, token] [, username]",
		"Holt Änderungen vom Remote und führt sie in den aktuellen Branch zusammen.\n\n"+
			"Konkret: Entspricht 'git pull' auf dem aktuell ausgecheckten Branch (Fast-Forward).\n"+
			"Wird path weggelassen, wird das aktuelle Arbeitsverzeichnis verwendet. Für\n"+
			"authentifizierte Remotes: entweder keyPath (SSH, optional mit knownHostsPath für\n"+
			"TOFU-Host-Key-Prüfung wie bei git.CloneWithKey) oder token (HTTPS, optional mit\n"+
			"username, Standard 'x-access-token') angeben. Ist keins von beiden gesetzt, läuft\n"+
			"der Pull ohne Auth (nur für unauthentifizierte Remotes ausreichend). Um einen\n"+
			"späteren Parameter zu setzen, ohne einen davorliegenden zu verwenden, einen leeren\n"+
			"String übergeben.\n"+
			"Parameter: path (String, optional) - Repository-Pfad; keyPath (String, optional) -\n"+
			"Pfad zum privaten SSH-Schlüssel; knownHostsPath (String, optional) - Pfad zur\n"+
			"known_hosts-Datei; token (String, optional) - HTTPS-Zugriffstoken; username\n"+
			"(String, optional) - Benutzername für Basic-Auth\n"+
			"Rückgabe: Bool - true bei Erfolg (auch wenn bereits aktuell)",
		func(args []Value) Value {
			rawPath := ""
			if len(args) >= 1 {
				rawPath = args[0].Str
			}

			path, errVal := repoPathOrDefault(rawPath)
			if errVal != nil {
				return *errVal
			}

			auth, errVal := gitAuthFromArgs(args, 1, 2, 3, 4, "git.Pull")
			if errVal != nil {
				return *errVal
			}

			repo, err := git.PlainOpen(path)
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.Pull: %q ist kein gültiges Repository: %v",
					path, err))
			}

			wt, err := repo.Worktree()
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.Pull: Worktree nicht lesbar: %v",
					err))
			}

			err = wt.Pull(&git.PullOptions{
				Auth: auth,
			})
			if err != nil && err != git.NoErrAlreadyUpToDate {
				return ErrorVal(fmt.Sprintf(
					"git.Pull: Pull fehlgeschlagen: %v",
					err))
			}

			return Value{
				Kind: KindBool,
				Bool: true,
			}
		})

	Register(ns+"Push", "git", "[path] [, keyPath] [, knownHostsPath] [, token] [, username]",
		"Sendet lokale Commits an den Remote.\n\n"+
			"Konkret: Entspricht 'git push' auf dem aktuellen Branch zum konfigurierten Remote\n"+
			"'origin'. Wird path weggelassen, wird das aktuelle Arbeitsverzeichnis verwendet. Für\n"+
			"authentifizierte Remotes: entweder keyPath (SSH, optional mit knownHostsPath für\n"+
			"TOFU-Host-Key-Prüfung wie bei git.CloneWithKey) oder token (HTTPS, optional mit\n"+
			"username, Standard 'x-access-token') angeben. Ist keins von beiden gesetzt, läuft\n"+
			"der Push ohne Auth (nur für unauthentifizierte Remotes ausreichend). Um einen\n"+
			"späteren Parameter zu setzen, ohne einen davorliegenden zu verwenden, einen leeren\n"+
			"String übergeben.\n"+
			"Parameter: path (String, optional) - Repository-Pfad; keyPath (String, optional) -\n"+
			"Pfad zum privaten SSH-Schlüssel; knownHostsPath (String, optional) - Pfad zur\n"+
			"known_hosts-Datei; token (String, optional) - HTTPS-Zugriffstoken; username\n"+
			"(String, optional) - Benutzername für Basic-Auth\n"+
			"Rückgabe: Bool - true bei Erfolg (auch wenn nichts zu pushen war)",
		func(args []Value) Value {
			rawPath := ""
			if len(args) >= 1 {
				rawPath = args[0].Str
			}

			path, errVal := repoPathOrDefault(rawPath)
			if errVal != nil {
				return *errVal
			}

			auth, errVal := gitAuthFromArgs(args, 1, 2, 3, 4, "git.Push")
			if errVal != nil {
				return *errVal
			}

			repo, err := git.PlainOpen(path)
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.Push: %q ist kein gültiges Repository: %v",
					path, err))
			}

			err = repo.Push(&git.PushOptions{
				Auth: auth,
			})
			if err != nil && err != git.NoErrAlreadyUpToDate {
				return ErrorVal(fmt.Sprintf(
					"git.Push: Push fehlgeschlagen: %v",
					err))
			}

			return Value{
				Kind: KindBool,
				Bool: true,
			}
		})

	Register(ns+"Fetch", "git", "[path] [, keyPath] [, knownHostsPath] [, token] [, username]",
		"Holt Änderungen vom Remote, ohne sie zu übernehmen.\n\n"+
			"Konkret: Entspricht 'git fetch'. Aktualisiert Remote-Tracking-Branches, ohne den\n"+
			"Arbeitsbereich zu verändern. Wird path weggelassen, wird das aktuelle\n"+
			"Arbeitsverzeichnis verwendet. Für authentifizierte Remotes: entweder keyPath (SSH,\n"+
			"optional mit knownHostsPath für TOFU-Host-Key-Prüfung wie bei git.CloneWithKey)\n"+
			"oder token (HTTPS, optional mit username, Standard 'x-access-token') angeben. Ist\n"+
			"keins von beiden gesetzt, läuft der Fetch ohne Auth (nur für unauthentifizierte\n"+
			"Remotes ausreichend). Um einen späteren Parameter zu setzen, ohne einen\n"+
			"davorliegenden zu verwenden, einen leeren String übergeben.\n"+
			"Parameter: path (String, optional) - Repository-Pfad; keyPath (String, optional) -\n"+
			"Pfad zum privaten SSH-Schlüssel; knownHostsPath (String, optional) - Pfad zur\n"+
			"known_hosts-Datei; token (String, optional) - HTTPS-Zugriffstoken; username\n"+
			"(String, optional) - Benutzername für Basic-Auth\n"+
			"Rückgabe: Bool - true bei Erfolg (auch wenn bereits aktuell)",
		func(args []Value) Value {
			rawPath := ""
			if len(args) >= 1 {
				rawPath = args[0].Str
			}

			path, errVal := repoPathOrDefault(rawPath)
			if errVal != nil {
				return *errVal
			}

			auth, errVal := gitAuthFromArgs(args, 1, 2, 3, 4, "git.Fetch")
			if errVal != nil {
				return *errVal
			}

			repo, err := git.PlainOpen(path)
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.Fetch: %q ist kein gültiges Repository: %v",
					path, err))
			}

			err = repo.Fetch(&git.FetchOptions{
				Auth: auth,
			})
			if err != nil && err != git.NoErrAlreadyUpToDate {
				return ErrorVal(fmt.Sprintf(
					"git.Fetch: Fetch fehlgeschlagen: %v",
					err))
			}

			return Value{
				Kind: KindBool,
				Bool: true,
			}
		})

	// ---------------------------------------------------------------
	// Add / Commit / Remove
	// ---------------------------------------------------------------

	Register(ns+"Add", "git", "pattern [, path]",
		"Fügt Dateien zum Git-Index hinzu (staging).\n\n"+
			"Konkret: Entspricht 'git add'. Nimmt einen Dateipfad relativ zum Repo-Root.\n"+
			"Wird path weggelassen, wird das aktuelle Arbeitsverzeichnis verwendet.\n"+
			"Parameter: pattern (String) - Dateipfad relativ zum Repo-Root;\n"+
			"path (String, optional) - Repository-Pfad\n"+
			"Rückgabe: Bool - true bei Erfolg",
		func(args []Value) Value {
			if len(args) < 1 {
				return ErrorVal("git.Add: Erwartet mindestens pattern.")
			}
			pattern := args[0].Str

			rawPath := ""
			if len(args) >= 2 {
				rawPath = args[1].Str
			}

			path, errVal := repoPathOrDefault(rawPath)
			if errVal != nil {
				return *errVal
			}

			repo, err := git.PlainOpen(path)
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.Add: %q ist kein gültiges Repository: %v",
					path, err))
			}

			wt, err := repo.Worktree()
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.Add: Worktree nicht lesbar: %v",
					err))
			}

			_, err = wt.Add(pattern)
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.Add: %q konnte nicht hinzugefügt werden: %v",
					pattern, err))
			}

			return Value{
				Kind: KindBool,
				Bool: true,
			}
		})

	Register(ns+"Commit", "git", "message [, path]",
		"Erstellt einen Commit mit allen gestagten Änderungen.\n\n"+
			"Konkret: Entspricht 'git commit -m'. Autor wird als 'VBX <vbx@localhost>' gesetzt.\n"+
			"Wird path weggelassen, wird das aktuelle Arbeitsverzeichnis verwendet.\n"+
			"Parameter: message (String) - Commit-Nachricht; path (String, optional) - Repository-Pfad\n"+
			"Rückgabe: String - Hash des erzeugten Commits",
		func(args []Value) Value {
			if len(args) < 1 {
				return ErrorVal("git.Commit: Erwartet mindestens message.")
			}
			message := args[0].Str

			rawPath := ""
			if len(args) >= 2 {
				rawPath = args[1].Str
			}

			path, errVal := repoPathOrDefault(rawPath)
			if errVal != nil {
				return *errVal
			}

			repo, err := git.PlainOpen(path)
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.Commit: %q ist kein gültiges Repository: %v",
					path, err))
			}

			wt, err := repo.Worktree()
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.Commit: Worktree nicht lesbar: %v",
					err))
			}

			hash, err := wt.Commit(message, &git.CommitOptions{
				Author: &object.Signature{
					Name:  "VBX",
					Email: "vbx@localhost",
					When:  time.Now(),
				},
			})
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.Commit: Commit fehlgeschlagen: %v",
					err))
			}

			return Value{
				Kind: KindStr,
				Str:  hash.String(),
			}
		})

	Register(ns+"Remove", "git", "pattern [, path]",
		"Entfernt eine Datei aus Git-Index und Arbeitsverzeichnis.\n\n"+
			"Konkret: Entspricht 'git rm'. Löscht die Datei physisch und entfernt sie aus dem Index.\n"+
			"Wird path weggelassen, wird das aktuelle Arbeitsverzeichnis verwendet.\n"+
			"Parameter: pattern (String) - Dateipfad relativ zum Repo-Root;\n"+
			"path (String, optional) - Repository-Pfad\n"+
			"Rückgabe: Bool - true bei Erfolg",
		func(args []Value) Value {
			if len(args) < 1 {
				return ErrorVal("git.Remove: Erwartet mindestens pattern.")
			}
			pattern := args[0].Str

			rawPath := ""
			if len(args) >= 2 {
				rawPath = args[1].Str
			}

			path, errVal := repoPathOrDefault(rawPath)
			if errVal != nil {
				return *errVal
			}

			repo, err := git.PlainOpen(path)
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.Remove: %q ist kein gültiges Repository: %v",
					path, err))
			}

			wt, err := repo.Worktree()
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.Remove: Worktree nicht lesbar: %v",
					err))
			}

			_, err = wt.Remove(pattern)
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.Remove: %q konnte nicht entfernt werden: %v",
					pattern, err))
			}

			return Value{
				Kind: KindBool,
				Bool: true,
			}
		})

	Register(ns+"QuickPush", "git", "message [, pattern] [, path] [, keyPath] [, knownHostsPath] [, token] [, username]",
		"Stagt alle Änderungen, erstellt einen Commit und pusht ihn - in einem Aufruf.\n\n"+
			"Konkret: Entspricht dem Dreischritt 'git add' + 'git commit -m' + 'git push',\n"+
			"wie es z.B. der 'Commit & Push'/'Sync'-Knopf in VSCode macht. KEIN Force-Push -\n"+
			"schlägt der Push fehl (z.B. weil der Remote neuere Commits hat), wird ein Fehler\n"+
			"zurückgegeben; der lokale Commit bleibt in diesem Fall bestehen. Autor wird wie bei\n"+
			"git.Commit als 'VBX <vbx@localhost>' gesetzt. Wird path weggelassen, wird das\n"+
			"aktuelle Arbeitsverzeichnis verwendet. Für authentifizierte Remotes: entweder\n"+
			"keyPath (SSH, optional mit knownHostsPath für TOFU-Host-Key-Prüfung wie bei\n"+
			"git.CloneWithKey) oder token (HTTPS, optional mit username, Standard\n"+
			"'x-access-token') angeben. Um einen späteren Parameter zu setzen, ohne einen\n"+
			"davorliegenden zu verwenden, einen leeren String übergeben.\n"+
			"Parameter: message (String) - Commit-Nachricht; pattern (String, optional) -\n"+
			"zu stagender Pfad relativ zum Repo-Root (Standard: '.', also alle Änderungen);\n"+
			"path (String, optional) - Repository-Pfad; keyPath (String, optional) - Pfad zum\n"+
			"privaten SSH-Schlüssel; knownHostsPath (String, optional) - Pfad zur\n"+
			"known_hosts-Datei; token (String, optional) - HTTPS-Zugriffstoken; username\n"+
			"(String, optional) - Benutzername für Basic-Auth\n"+
			"Rückgabe: String - Hash des erzeugten Commits",
		func(args []Value) Value {
			if len(args) < 1 {
				return ErrorVal("git.QuickPush: Erwartet mindestens message.")
			}
			message := args[0].Str

			pattern := "."
			if len(args) >= 2 && args[1].Str != "" {
				pattern = args[1].Str
			}

			rawPath := ""
			if len(args) >= 3 {
				rawPath = args[2].Str
			}

			path, errVal := repoPathOrDefault(rawPath)
			if errVal != nil {
				return *errVal
			}

			auth, errVal := gitAuthFromArgs(args, 3, 4, 5, 6, "git.QuickPush")
			if errVal != nil {
				return *errVal
			}

			repo, err := git.PlainOpen(path)
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.QuickPush: %q ist kein gültiges Repository: %v",
					path, err))
			}

			wt, err := repo.Worktree()
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.QuickPush: Worktree nicht lesbar: %v",
					err))
			}

			_, err = wt.Add(pattern)
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.QuickPush: %q konnte nicht gestaged werden: %v",
					pattern, err))
			}

			hash, err := wt.Commit(message, &git.CommitOptions{
				Author: &object.Signature{
					Name:  "VBX",
					Email: "vbx@localhost",
					When:  time.Now(),
				},
			})
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.QuickPush: Commit fehlgeschlagen: %v",
					err))
			}

			err = repo.Push(&git.PushOptions{
				Auth: auth,
			})
			if err != nil && err != git.NoErrAlreadyUpToDate {
				return ErrorVal(fmt.Sprintf(
					"git.QuickPush: Commit %q erstellt, aber Push fehlgeschlagen: %v",
					hash.String(), err))
			}

			return Value{
				Kind: KindStr,
				Str:  hash.String(),
			}
		})

	// ---------------------------------------------------------------
	// Status / Log / Diff
	// ---------------------------------------------------------------

	Register(ns+"Status", "git", "[path]",
		"Liefert den Status des Arbeitsverzeichnisses.\n\n"+
			"Konkret: Entspricht 'git status'. Zeigt geänderte, neu hinzugefügte, gelöschte und\n"+
			"unversionierte Dateien. Wird path weggelassen, wird das aktuelle Arbeitsverzeichnis\n"+
			"verwendet.\n"+
			"Parameter: path (String, optional) - Repository-Pfad\n"+
			"Rückgabe: Map mit Arrays 'modified', 'added', 'deleted', 'untracked' (jeweils Dateipfade)",
		func(args []Value) Value {
			rawPath := ""
			if len(args) >= 1 {
				rawPath = args[0].Str
			}

			path, errVal := repoPathOrDefault(rawPath)
			if errVal != nil {
				return *errVal
			}

			repo, err := git.PlainOpen(path)
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.Status: %q ist kein gültiges Repository: %v",
					path, err))
			}

			wt, err := repo.Worktree()
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.Status: Worktree nicht lesbar: %v",
					err))
			}

			status, err := wt.Status()
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.Status: Status-Abfrage fehlgeschlagen: %v",
					err))
			}

			var modified, added, deleted, untracked []Value

			for file, s := range status {
				v := Value{
					Kind: KindStr,
					Str:  file,
				}

				switch s.Worktree {
				case git.Modified:
					modified = append(modified, v)

				case git.Added:
					added = append(added, v)

				case git.Deleted:
					deleted = append(deleted, v)

				case git.Untracked:
					untracked = append(untracked, v)
				}
			}

			result := map[string]Value{
				"modified": {
					Kind: KindArr,
					Arr:  modified,
				},
				"added": {
					Kind: KindArr,
					Arr:  added,
				},
				"deleted": {
					Kind: KindArr,
					Arr:  deleted,
				},
				"untracked": {
					Kind: KindArr,
					Arr:  untracked,
				},
			}

			return Value{
				Kind: KindMap,
				Map:  result,
			}
		})

	Register(ns+"Log", "git", "[limit] [, path]",
		"Liefert die Commit-Historie.\n\n"+
			"Konkret: Entspricht 'git log --oneline' (erweitert um Autor und Datum), ausgehend von HEAD.\n"+
			"Wird path weggelassen, wird das aktuelle Arbeitsverzeichnis verwendet.\n"+
			"Parameter: limit (Zahl, optional) - maximale Anzahl Commits (0 oder weggelassen = alle);\n"+
			"path (String, optional) - Repository-Pfad\n"+
			"Rückgabe: Array von Maps mit 'hash', 'author', 'message', 'date'",
		func(args []Value) Value {
			limit := 0
			if len(args) >= 1 {
				limit = int(args[0].Num)
			}

			rawPath := ""
			if len(args) >= 2 {
				rawPath = args[1].Str
			}

			path, errVal := repoPathOrDefault(rawPath)
			if errVal != nil {
				return *errVal
			}

			repo, err := git.PlainOpen(path)
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.Log: %q ist kein gültiges Repository: %v",
					path, err))
			}

			ref, err := repo.Head()
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.Log: HEAD nicht auflösbar: %v",
					err))
			}

			iter, err := repo.Log(&git.LogOptions{
				From: ref.Hash(),
			})
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.Log: Historie nicht lesbar: %v",
					err))
			}
			defer iter.Close()

			var entries []Value
			count := 0

			for {
				if limit > 0 && count >= limit {
					break
				}

				c, err := iter.Next()
				if err != nil {
					break
				}

				entries = append(entries, Value{
					Kind: KindMap,
					Map: map[string]Value{
						"hash": {
							Kind: KindStr,
							Str:  c.Hash.String(),
						},
						"author": {
							Kind: KindStr,
							Str:  c.Author.Name,
						},
						"message": {
							Kind: KindStr,
							Str:  c.Message,
						},
						"date": {
							Kind: KindStr,
							Str:  c.Author.When.Format("2006-01-02 15:04:05"),
						},
					},
				})

				count++
			}

			return Value{
				Kind: KindArr,
				Arr:  entries,
			}
		})

	Register(ns+"Diff", "git", "commitA [, commitB] [, path]",
		"Zeigt den Diff zwischen zwei Commits.\n\n"+
			"Konkret: Entspricht 'git diff <commitA> <commitB>'. Wird commitB weggelassen oder\n"+
			"leer gelassen, bedeutet das: Vergleich gegen HEAD. Ein Diff gegen das unversionierte\n"+
			"Arbeitsverzeichnis (wie 'git diff' ohne Argumente) wird nicht unterstützt. Wird path\n"+
			"weggelassen, wird das aktuelle Arbeitsverzeichnis verwendet.\n"+
			"Parameter: commitA (String) - älterer Commit-Hash; commitB (String, optional) -\n"+
			"neuerer Commit-Hash, leer/weggelassen für HEAD; path (String, optional) - Repository-Pfad\n"+
			"Rückgabe: String - unified Diff",
		func(args []Value) Value {
			if len(args) < 1 {
				return ErrorVal("git.Diff: Erwartet mindestens commitA.")
			}
			hashA := args[0].Str

			hashB := ""
			if len(args) >= 2 {
				hashB = args[1].Str
			}

			rawPath := ""
			if len(args) >= 3 {
				rawPath = args[2].Str
			}

			path, errVal := repoPathOrDefault(rawPath)
			if errVal != nil {
				return *errVal
			}

			repo, err := git.PlainOpen(path)
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.Diff: %q ist kein gültiges Repository: %v",
					path, err))
			}

			revA, err := repo.ResolveRevision(plumbing.Revision(hashA))
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.Diff: Commit %q nicht auflösbar: %v",
					hashA, err))
			}

			commitA, err := repo.CommitObject(*revA)
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.Diff: Commit %q nicht lesbar: %v",
					hashA, err))
			}

			var commitB *object.Commit

			if hashB == "" {
				head, err := repo.Head()
				if err != nil {
					return ErrorVal(fmt.Sprintf(
						"git.Diff: HEAD nicht auflösbar: %v",
						err))
				}

				commitB, err = repo.CommitObject(head.Hash())
				if err != nil {
					return ErrorVal(fmt.Sprintf(
						"git.Diff: HEAD-Commit nicht lesbar: %v",
						err))
				}
			} else {
				revB, err := repo.ResolveRevision(plumbing.Revision(hashB))
				if err != nil {
					return ErrorVal(fmt.Sprintf(
						"git.Diff: Commit %q nicht auflösbar: %v",
						hashB, err))
				}

				commitB, err = repo.CommitObject(*revB)
				if err != nil {
					return ErrorVal(fmt.Sprintf(
						"git.Diff: Commit %q nicht lesbar: %v",
						hashB, err))
				}
			}

			patch, err := commitA.Patch(commitB)
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.Diff: Patch-Erstellung fehlgeschlagen: %v",
					err))
			}

			return Value{
				Kind: KindStr,
				Str:  patch.String(),
			}
		})

	// ---------------------------------------------------------------
	// Reset / ResetHard
	// ---------------------------------------------------------------

	Register(ns+"ResetHard", "git", "commitHash [, path]",
		"Setzt das Repository hart auf einen bestimmten Commit zurück.\n\n"+
			"Konkret: Entspricht 'git reset --hard <hash>'. Alle lokalen Änderungen und Commits\n"+
			"nach diesem Hash gehen unwiderruflich verloren. Typischer Anwendungsfall:\n"+
			"'git log --oneline' -> letzten funktionierenden Commit finden -> hierher zurücksetzen.\n"+
			"Wird path weggelassen, wird das aktuelle Arbeitsverzeichnis verwendet.\n"+
			"Parameter: commitHash (String) - Ziel-Commit-Hash (voll oder gekürzt);\n"+
			"path (String, optional) - Repository-Pfad\n"+
			"Rückgabe: Bool - true bei Erfolg",
		func(args []Value) Value {
			if len(args) < 1 {
				return ErrorVal("git.ResetHard: Erwartet mindestens commitHash.")
			}
			hashStr := args[0].Str

			rawPath := ""
			if len(args) >= 2 {
				rawPath = args[1].Str
			}

			path, errVal := repoPathOrDefault(rawPath)
			if errVal != nil {
				return *errVal
			}

			repo, err := git.PlainOpen(path)
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.ResetHard: %q ist kein gültiges Repository: %v",
					path, err))
			}

			wt, err := repo.Worktree()
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.ResetHard: Worktree nicht lesbar: %v",
					err))
			}

			rev, err := repo.ResolveRevision(plumbing.Revision(hashStr))
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.ResetHard: Commit %q nicht auflösbar: %v",
					hashStr, err))
			}

			err = wt.Reset(&git.ResetOptions{
				Commit: *rev,
				Mode:   git.HardReset,
			})
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.ResetHard: Reset auf %q fehlgeschlagen: %v",
					hashStr, err))
			}

			return Value{
				Kind: KindBool,
				Bool: true,
			}
		})

	Register(ns+"Reset", "git", "[mode] [, path]",
		"Hebt das Staging von Änderungen auf (unstage), ohne Dateien im Arbeitsverzeichnis\n"+
			"zu verändern.\n\n"+
			"Konkret: Entspricht 'git reset' (mixed, Standard) oder 'git reset --soft'. Wird path\n"+
			"weggelassen, wird das aktuelle Arbeitsverzeichnis verwendet.\n"+
			"Parameter: mode (String, optional) - 'mixed' (Standard) oder 'soft';\n"+
			"path (String, optional) - Repository-Pfad\n"+
			"Rückgabe: Bool - true bei Erfolg",
		func(args []Value) Value {
			mode := "mixed"
			if len(args) >= 1 && args[0].Str != "" {
				mode = args[0].Str
			}

			rawPath := ""
			if len(args) >= 2 {
				rawPath = args[1].Str
			}

			path, errVal := repoPathOrDefault(rawPath)
			if errVal != nil {
				return *errVal
			}

			repo, err := git.PlainOpen(path)
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.Reset: %q ist kein gültiges Repository: %v",
					path, err))
			}

			wt, err := repo.Worktree()
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.Reset: Worktree nicht lesbar: %v",
					err))
			}

			resetMode := git.MixedReset

			if mode == "soft" {
				resetMode = git.SoftReset
			}

			head, err := repo.Head()
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.Reset: HEAD nicht auflösbar: %v",
					err))
			}

			err = wt.Reset(&git.ResetOptions{
				Commit: head.Hash(),
				Mode:   resetMode,
			})
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.Reset: Reset fehlgeschlagen: %v",
					err))
			}

			return Value{
				Kind: KindBool,
				Bool: true,
			}
		})

	// ---------------------------------------------------------------
	// Branch / Checkout / Sonstiges
	// ---------------------------------------------------------------

	Register(ns+"CurrentBranch", "git", "[path]",
		"Liefert den Namen des aktuell ausgecheckten Branches.\n\n"+
			"Konkret: Liest den kurzen Branch-Namen aus HEAD. Wird path weggelassen, wird das\n"+
			"aktuelle Arbeitsverzeichnis verwendet.\n"+
			"Parameter: path (String, optional) - Repository-Pfad\n"+
			"Rückgabe: String - Branch-Name (z.B. 'main')",
		func(args []Value) Value {
			rawPath := ""
			if len(args) >= 1 {
				rawPath = args[0].Str
			}

			path, errVal := repoPathOrDefault(rawPath)
			if errVal != nil {
				return *errVal
			}

			repo, err := git.PlainOpen(path)
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.CurrentBranch: %q ist kein gültiges Repository: %v",
					path, err))
			}

			head, err := repo.Head()
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.CurrentBranch: HEAD nicht auflösbar: %v",
					err))
			}

			return Value{
				Kind: KindStr,
				Str:  head.Name().Short(),
			}
		})

	Register(ns+"Checkout", "git", "branch [, create] [, path]",
		"Wechselt den Branch oder erstellt einen neuen.\n\n"+
			"Konkret: Entspricht 'git checkout <branch>' bzw. mit create=true 'git checkout -b <branch>'.\n"+
			"Wird path weggelassen, wird das aktuelle Arbeitsverzeichnis verwendet.\n"+
			"Parameter: branch (String) - Branch-Name; create (Bool, optional) - true, um den\n"+
			"Branch neu anzulegen (Standard: false); path (String, optional) - Repository-Pfad\n"+
			"Rückgabe: Bool - true bei Erfolg",
		func(args []Value) Value {
			if len(args) < 1 {
				return ErrorVal("git.Checkout: Erwartet mindestens branch.")
			}
			branch := args[0].Str

			create := false
			if len(args) >= 2 {
				create = args[1].Bool
			}

			rawPath := ""
			if len(args) >= 3 {
				rawPath = args[2].Str
			}

			path, errVal := repoPathOrDefault(rawPath)
			if errVal != nil {
				return *errVal
			}

			repo, err := git.PlainOpen(path)
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.Checkout: %q ist kein gültiges Repository: %v",
					path, err))
			}

			wt, err := repo.Worktree()
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.Checkout: Worktree nicht lesbar: %v",
					err))
			}

			err = wt.Checkout(&git.CheckoutOptions{
				Branch: plumbing.NewBranchReferenceName(branch),
				Create: create,
			})
			if err != nil {
				return ErrorVal(fmt.Sprintf(
					"git.Checkout: Wechsel zu Branch %q fehlgeschlagen: %v",
					branch, err))
			}

			return Value{
				Kind: KindBool,
				Bool: true,
			}
		})

	Register(ns+"IsRepo", "git", "[path]",
		"Prüft, ob ein Verzeichnis ein gültiges Git-Repository ist.\n\n"+
			"Konkret: Prüft auf Vorhandensein eines gültigen '.git'-Verzeichnisses. Wird path\n"+
			"weggelassen, wird das aktuelle Arbeitsverzeichnis geprüft.\n"+
			"Parameter: path (String, optional) - zu prüfender Pfad\n"+
			"Rückgabe: Bool - true, wenn gültiges Repository",
		func(args []Value) Value {
			rawPath := ""
			if len(args) >= 1 {
				rawPath = args[0].Str
			}

			path, errVal := repoPathOrDefault(rawPath)
			if errVal != nil {
				return *errVal
			}

			_, err := git.PlainOpen(path)

			return Value{
				Kind: KindBool,
				Bool: err == nil,
			}
		})
}

// cloneTargetPath löst den Zielpfad für Clone-Varianten absolut auf und prüft,
// dass dort noch kein Verzeichnis existiert (go-git bricht sonst mit einem
// weniger sprechenden Fehler ab).
//
// Ist raw leer, wird der Zielverzeichnisname wie bei 'git clone' aus der URL
// abgeleitet (letztes Pfadsegment ohne '.git'-Endung), relativ zum aktuellen
// Arbeitsverzeichnis.
func cloneTargetPath(raw string, url string, funcName string) (string, *Value) {

	target := raw
	if target == "" {
		target = deriveCloneDirName(url)
	}

	path, errVal := absPathVal(target)
	if errVal != nil {
		return "", errVal
	}

	if _, err := os.Stat(path); err == nil {
		v := ErrorVal(fmt.Sprintf(
			"git.%s: Zielverzeichnis %q existiert bereits.",
			funcName, path))
		return "", &v

	} else if !os.IsNotExist(err) {
		v := ErrorVal(fmt.Sprintf(
			"git.%s: Zielverzeichnis %q kann nicht geprüft werden: %v",
			funcName, path, err))
		return "", &v
	}

	return path, nil
}

// deriveCloneDirName leitet aus einer Git-URL den Standard-Verzeichnisnamen ab,
// so wie es 'git clone <url>' ohne explizites Zielverzeichnis tut:
// letztes Pfadsegment, '.git'-Endung entfernt.
//
// Beispiele:
//
//	https://github.com/user/repo.git -> repo
//	git@github.com:user/repo.git     -> repo
//	https://github.com/user/repo     -> repo
func deriveCloneDirName(url string) string {

	u := strings.TrimSuffix(strings.TrimSpace(url), "/")

	// Letztes Segment nach '/' oder ':' (SSH-Kurzform user@host:pfad)
	idx := strings.LastIndexAny(u, "/:")

	name := u
	if idx >= 0 {
		name = u[idx+1:]
	}

	name = strings.TrimSuffix(name, ".git")

	if name == "" {
		name = "repo"
	}

	return name
}

// repoPathOrDefault löst einen Repository-Pfad absolut auf. Ist raw leer,
// wird das aktuelle Arbeitsverzeichnis (".") verwendet - analog dazu, wie
// 'git status'/'git pull'/etc. ohne Pfadangabe im aktuellen Verzeichnis
// arbeiten.
func repoPathOrDefault(raw string) (string, *Value) {

	target := raw
	if target == "" {
		target = "."
	}

	return absPathVal(target)
}

// gitAuthFromArgs baut optionale Authentifizierung für Remote-Operationen
// (Pull/Push/Fetch/QuickPush) aus positionellen Argumenten. keyIdx hat
// Vorrang vor tokenIdx, falls beide gesetzt sind. Sind weder keyPath noch
// token angegeben, wird (nil, nil) zurückgegeben - die Operation läuft dann
// ohne explizite Auth (funktioniert nur bei unauthentifizierten Remotes).
func gitAuthFromArgs(args []Value, keyIdx int, knownHostsIdx int, tokenIdx int, usernameIdx int, funcName string) (transport.AuthMethod, *Value) {

	keyPathRaw := ""
	if len(args) > keyIdx {
		keyPathRaw = args[keyIdx].Str
	}

	token := ""
	if len(args) > tokenIdx {
		token = args[tokenIdx].Str
	}

	if keyPathRaw != "" {
		keyPath, errVal := absPathVal(keyPathRaw)
		if errVal != nil {
			return nil, errVal
		}

		hostKeyCallback, errVal := buildHostKeyCallback(args, knownHostsIdx)
		if errVal != nil {
			return nil, errVal
		}

		auth, err := gitssh.NewPublicKeysFromFile("git", keyPath, "")
		if err != nil {
			v := ErrorVal(fmt.Sprintf(
				"%s: Key %q konnte nicht geladen werden: %v",
				funcName, keyPath, err))
			return nil, &v
		}
		auth.HostKeyCallback = hostKeyCallback

		return auth, nil
	}

	if token != "" {
		username := "x-access-token"
		if len(args) > usernameIdx && args[usernameIdx].Str != "" {
			username = args[usernameIdx].Str
		}

		return &http.BasicAuth{
			Username: username,
			Password: token,
		}, nil
	}

	return nil, nil
}
