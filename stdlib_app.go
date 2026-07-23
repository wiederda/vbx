// ------------------------
// stdlib_app.go
// ------------------------

package main

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
)

// InitAppFunctions initialisiert den app.-Namespace
func InitAppFunctions() {
	if builtins == nil {
		builtins = make(map[string]BuiltinInfo)
	}

	ns := "app."

	// Interner Helper für Home-Verzeichnis
	getHomeDir := func() string {
		// 1. user.Current() ist die beste Methode
		usr, err := user.Current()
		if err == nil {
			return usr.HomeDir
		}
		// 2. Fallback für Windows
		if runtime.GOOS == "windows" {
			if profile := os.Getenv("USERPROFILE"); profile != "" {
				return profile
			}
		}
		// 3. Fallback für Unix-ähnliche Systeme
		if home := os.Getenv("HOME"); home != "" {
			return home
		}
		// Letzter Ausweg
		return ""
	}

	// app.StartupPath: Ordner der ausführbaren Datei
	Register(ns+"StartupPath", "app", "", "Gibt das Verzeichnis der ausführbaren Datei zurück.", func(args []Value) Value {
		exe, _ := os.Executable()
		return Value{Kind: KindStr, Str: filepath.Dir(exe)}
	})

	// app.ExecutablePath: Vollständiger Pfad
	Register(ns+"ExecutablePath", "app", "", "Gibt den vollständigen Pfad der ausführbaren Datei zurück.", func(args []Value) Value {
		exe, _ := os.Executable()
		return Value{Kind: KindStr, Str: exe}
	})

	// app.CurrentDirectory: aktuelles Arbeitsverzeichnis
	Register(ns+"CurrentDirectory", "app", "", "Gibt das aktuelle Arbeitsverzeichnis (Working Directory) zurück.", func(args []Value) Value {
		dir, err := os.Getwd()
		if err != nil {
			return Value{Kind: KindStr, Str: ""}
		}
		return Value{Kind: KindStr, Str: dir}
	})

	// app.TempPath() als bequemer Alias
	Register(ns+"TempPath", "app", "", "Gibt den Pfad zum temporären Verzeichnis des Systems zurück.", func(args []Value) Value {
		return Value{Kind: KindStr, Str: os.TempDir()}
	})

	// app.SpecialFolder(name)
	Register(ns+"SpecialFolder", "app", "name", "Gibt Standardpfade zurück (desktop, home, temp, etc.).", func(args []Value) Value {
		if len(args) < 1 {
			return StrVal("")
		}

		name := strings.ToLower(args[0].Str)
		home := getHomeDir()
		var folder string

		// 1. Windows-spezifische Pfade (Priorität)
		if runtime.GOOS == "windows" {
			switch name {
			case "appdata":
				folder = os.Getenv("APPDATA")
			case "localappdata":
				folder = os.Getenv("LOCALAPPDATA")
			case "programdata":
				folder = os.Getenv("PROGRAMDATA")
			}
		} else {
			// 2. Plattformunabhängige Pfade (Unix/macOS)
			switch name {
			case "home":
				folder = home
			case "desktop":
				folder = filepath.Join(home, "Desktop")
			case "documents":
				folder = filepath.Join(home, "Documents")
			case "downloads":
				folder = filepath.Join(home, "Downloads")
			case "pictures":
				folder = filepath.Join(home, "Pictures")
			case "music":
				folder = filepath.Join(home, "Music")
			case "videos":
				folder = filepath.Join(home, "Videos")
			case "temp":
				folder = os.TempDir()
			// Hier könnten weitere plattformspezifische Pfade hinzugefügt werden
			default:
				folder = ""
			}
		}

		// Falls der Pfad im Windows-Fall nicht gesetzt wurde, aber der Name "temp" ist,
		// falls wir uns nicht auf Windows befinden, nutzen wir os.TempDir() als Fallback.
		if folder == "" && name == "temp" {
			folder = os.TempDir()
		}

		return StrVal(folder)
	})

}
