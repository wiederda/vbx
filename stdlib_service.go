// ------------------------
// stdlib_windows.go
//go:build windows
// +build windows

// ------------------------

package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

func serviceStateToString(state svc.State) string {
	switch state {
	case svc.Stopped:
		return "Stopped"
	case svc.StartPending:
		return "StartPending"
	case svc.StopPending:
		return "StopPending"
	case svc.Running:
		return "Running"
	case svc.ContinuePending:
		return "ContinuePending"
	case svc.PausePending:
		return "PausePending"
	case svc.Paused:
		return "Paused"
	default:
		return "Unknown"
	}
}

func stringToStartType(s string) uint32 {
	// Windows Service Start Types (Standardwerte aus der Win32 API)
	const (
		SERVICE_BOOT_START   = 0x00000000
		SERVICE_SYSTEM_START = 0x00000001
		SERVICE_AUTO_START   = 0x00000002
		SERVICE_DEMAND_START = 0x00000003 // "Manual"
		SERVICE_DISABLED     = 0x00000004
	)

	switch strings.ToLower(s) {
	case "auto", "automatic":
		return SERVICE_AUTO_START
	case "manual":
		return SERVICE_DEMAND_START
	case "disabled":
		return SERVICE_DISABLED
	case "delayed":
		// Falls dein mgr Paket StartDelayedAutoStart nicht kennt,
		// geben wir Auto_Start zurück. Das 'Delayed' Flag ist
		// technisch ein separater Config-Eintrag (DelayedAutoStart).
		return SERVICE_AUTO_START
	default:
		return 0
	}
}

func connectMgr(server string) (*mgr.Mgr, error) {
	if server == "" {
		return mgr.Connect()
	}
	return mgr.ConnectRemote(server)
}

func InitServiceFunctions() {
	if builtins == nil {
		builtins = make(map[string]BuiltinInfo)
	}

	ns := "service."

	Register(ns+"List", "service", "[server], [showDisplayNames]", "Gibt ein Array aller Dienste zurück. Wenn showDisplayNames true ist, im Format 'Name:Anzeigename'.", func(args []Value) Value {
		server := ""
		showDisplayNames := false

		// Parameter-Zuordnung
		if len(args) >= 1 && args[0].Str != "" {
			server = args[0].Str
		}
		if len(args) >= 2 {
			showDisplayNames = (args[1].Str == "true" || (args[1].Kind == KindBool && args[1].Bool))
		}

		m, err := connectMgr(server)
		if err != nil {
			return Value{Kind: KindStr, Str: "error: " + err.Error()}
		}
		defer m.Disconnect()

		names, err := m.ListServices()
		if err != nil {
			return Value{Kind: KindStr, Str: "error: " + err.Error()}
		}

		var resultValues []Value
		for _, name := range names {
			entry := name

			if showDisplayNames {
				s, err := m.OpenService(name)
				if err == nil {
					config, err := s.Config()
					if err == nil {
						entry = fmt.Sprintf("%s:%s", name, config.DisplayName)
					}
					s.Close()
				} else {
					entry = name + ":?" // Zugriff verweigert
				}
			}

			// Füge den Eintrag als neuen Value zum Array hinzu
			resultValues = append(resultValues, Value{Kind: KindStr, Str: entry})
		}

		// Rückgabe als echtes Array-Objekt
		return Value{Kind: KindArr, Arr: resultValues}
	})

	Register(ns+"Status", "service", "name, [server]", "Ruft den aktuellen Betriebsstatus eines Dienstes ab.", func(args []Value) Value {
		if len(args) < 1 {
			return Value{Kind: KindStr, Str: "Unknown"}
		}
		name := args[0].Str
		server := ""
		if len(args) >= 2 {
			server = args[1].Str
		}

		m, err := connectMgr(server)
		if err != nil {
			return Value{Kind: KindStr, Str: "error: " + err.Error()}
		}
		defer m.Disconnect()

		s, err := m.OpenService(name)
		if err != nil {
			return Value{Kind: KindStr, Str: "error: " + err.Error()}
		}
		defer s.Close()

		status, err := s.Query()
		if err != nil {
			return Value{Kind: KindStr, Str: "error: " + err.Error()}
		}

		return Value{Kind: KindStr, Str: serviceStateToString(status.State)}
	})

	Register(ns+"SetStartType", "service", "name, startType, [server]", "Setzt den automatischen Starttyp (auto, manuell, deaktiviert, verzögert) für einen Dienst.", func(args []Value) Value {
		if len(args) < 2 {
			return Value{Kind: KindStr, Str: "error: missing service name or start type"}
		}
		name := args[0].Str
		startTypeStr := args[1].Str
		server := ""
		if len(args) >= 3 {
			server = args[2].Str
		}

		startType := stringToStartType(startTypeStr)
		if startType == 0 && strings.ToLower(startTypeStr) != "unchanged" {
			return Value{Kind: KindStr, Str: "error: invalid start type (use auto, manual, disabled, delayed)"}
		}

		m, err := connectMgr(server)
		if err != nil {
			return Value{Kind: KindStr, Str: "error: " + err.Error()}
		}
		defer m.Disconnect()

		s, err := m.OpenService(name)
		if err != nil {
			return Value{Kind: KindStr, Str: "error: " + err.Error()}
		}
		defer s.Close()

		// Aktuelle Konfiguration abrufen
		config, err := s.Config()
		if err != nil {
			return Value{Kind: KindStr, Str: "error: " + err.Error()}
		}

		if strings.ToLower(startTypeStr) == "delayed" {
			config.StartType = 2 // Auto Start
			config.DelayedAutoStart = true
		} else {
			config.StartType = startType
			config.DelayedAutoStart = false
		}

		err = s.UpdateConfig(config)
		if err != nil {
			return Value{Kind: KindStr, Str: "error: " + err.Error()}
		}

		return Value{Kind: KindStr, Str: "OK"}
	})

	Register(ns+"Start", "service", "name, [server]", "Startet einen angegebenen Dienst.", func(args []Value) Value {
		if len(args) < 1 {
			return Value{Kind: KindStr, Str: "error: missing service name"}
		}
		name := args[0].Str
		server := ""
		if len(args) >= 2 {
			server = args[1].Str
		}

		m, err := connectMgr(server)
		if err != nil {
			return Value{Kind: KindStr, Str: "error: " + err.Error()}
		}
		defer m.Disconnect()

		s, err := m.OpenService(name)
		if err != nil {
			return Value{Kind: KindStr, Str: "error: " + err.Error()}
		}
		defer s.Close()

		if err := s.Start(); err != nil {
			return Value{Kind: KindStr, Str: "error: " + err.Error()}
		}

		status, _ := s.Query()
		return Value{Kind: KindStr, Str: serviceStateToString(status.State)}
	})

	Register(ns+"Stop", "service", "name, [server]", "Stoppt einen angegebenen Dienst.", func(args []Value) Value {
		if len(args) < 1 {
			return Value{Kind: KindStr, Str: "error: missing service name"}
		}
		name := args[0].Str
		server := ""
		if len(args) >= 2 {
			server = args[1].Str
		}

		m, err := connectMgr(server)
		if err != nil {
			return Value{Kind: KindStr, Str: "error: " + err.Error()}
		}
		defer m.Disconnect()

		s, err := m.OpenService(name)
		if err != nil {
			return Value{Kind: KindStr, Str: "error: " + err.Error()}
		}
		defer s.Close()

		if _, err := s.Control(svc.Stop); err != nil {
			return Value{Kind: KindStr, Str: "error: " + err.Error()}
		}

		status, _ := s.Query()
		return Value{Kind: KindStr, Str: serviceStateToString(status.State)}
	})

	Register(ns+"Restart", "service", "name, [server]", "Stoppt und startet einen Dienst anschließend neu.", func(args []Value) Value {
		if len(args) < 1 {
			return Value{Kind: KindStr, Str: "error: missing service name"}
		}
		name := args[0].Str
		server := ""
		if len(args) >= 2 {
			server = args[1].Str
		}

		go func() {
			m, err := connectMgr(server)
			if err != nil {
				log.Println("Restart error:", err)
				return
			}
			defer m.Disconnect()

			s, err := m.OpenService(name)
			if err != nil {
				log.Println("Restart error:", err)
				return
			}
			defer s.Close()

			if _, err := s.Control(svc.Stop); err != nil {
				log.Println("Stop error:", err)
				return
			}

			// Ticker statt Tick, um Memory Leaks zu vermeiden
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()

			stopTimeout := time.After(10 * time.Second)

			// Phase 1: Warten auf Stop
			stopped := false
			for !stopped {
				select {
				case <-stopTimeout:
					log.Println("Stop timeout reached for service:", name)
					return
				case <-ticker.C:
					status, err := s.Query()
					if err != nil {
						log.Println("Query error:", err)
						return
					}
					if status.State == svc.Stopped {
						stopped = true
					}
				}
			}

			// Phase 2: Starten
			if err := s.Start(); err != nil {
				log.Println("Start error:", err)
				return
			}

			startTimeout := time.After(10 * time.Second)
			for {
				select {
				case <-startTimeout:
					log.Println("Start timeout reached for service:", name)
					return
				case <-ticker.C:
					status, err := s.Query()
					if err != nil {
						return
					}
					if status.State == svc.Running {
						return
					}
				}
			}
		}()

		return Value{Kind: KindStr, Str: "Restarting"}
	})

	Register(ns+"Install", "service", "name, displayName, exePath, [startType], [server]", "Installiert einen neuen Windows-Dienst.", func(args []Value) Value {
		if len(args) < 3 {
			return Value{Kind: KindStr, Str: "error: name, displayName und exePath benötigt"}
		}
		name := args[0].Str
		displayName := args[1].Str
		exePath := args[2].Str
		startTypeStr := "auto"
		if len(args) >= 4 && args[3].Str != "" {
			startTypeStr = args[3].Str
		}
		server := ""
		if len(args) >= 5 {
			server = args[4].Str
		}

		startType := stringToStartType(startTypeStr)

		m, err := connectMgr(server)
		if err != nil {
			return Value{Kind: KindStr, Str: "error: " + err.Error()}
		}
		defer m.Disconnect()

		// Prüfen ob Dienst bereits existiert
		existing, err := m.OpenService(name)
		if err == nil {
			existing.Close()
			return Value{Kind: KindStr, Str: "error: Dienst '" + name + "' existiert bereits"}
		}

		cfg := mgr.Config{
			DisplayName: displayName,
			StartType:   startType,
		}

		s, err := m.CreateService(name, exePath, cfg)
		if err != nil {
			return Value{Kind: KindStr, Str: "error: " + err.Error()}
		}
		defer s.Close()

		return Value{Kind: KindStr, Str: "OK"}
	})

	Register(ns+"Delete", "service", "name, [server]", "Löscht einen Windows-Dienst. Stoppt ihn vorher falls er noch läuft.", func(args []Value) Value {
		if len(args) < 1 {
			return Value{Kind: KindStr, Str: "error: missing service name"}
		}
		name := args[0].Str
		server := ""
		if len(args) >= 2 {
			server = args[1].Str
		}

		m, err := connectMgr(server)
		if err != nil {
			return Value{Kind: KindStr, Str: "error: " + err.Error()}
		}
		defer m.Disconnect()

		s, err := m.OpenService(name)
		if err != nil {
			return Value{Kind: KindStr, Str: "error: " + err.Error()}
		}
		defer s.Close()

		// Dienst stoppen falls er läuft
		status, err := s.Query()
		if err != nil {
			return Value{Kind: KindStr, Str: "error: " + err.Error()}
		}

		if status.State != svc.Stopped {
			if _, err := s.Control(svc.Stop); err != nil {
				return Value{Kind: KindStr, Str: "error: Dienst konnte nicht gestoppt werden: " + err.Error()}
			}

			// Warten bis gestoppt (max 10 Sekunden)
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()
			timeout := time.After(10 * time.Second)

			stopped := false
			for !stopped {
				select {
				case <-timeout:
					return Value{Kind: KindStr, Str: "error: Timeout beim Stoppen des Dienstes"}
				case <-ticker.C:
					status, err := s.Query()
					if err != nil {
						return Value{Kind: KindStr, Str: "error: " + err.Error()}
					}
					if status.State == svc.Stopped {
						stopped = true
					}
				}
			}
		}

		if err := s.Delete(); err != nil {
			return Value{Kind: KindStr, Str: "error: " + err.Error()}
		}

		return Value{Kind: KindStr, Str: "OK"}
	})
}
