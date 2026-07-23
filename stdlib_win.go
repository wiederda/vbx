//go:build windows
// +build windows

// ------------------------
// stdlib_win.go
// ------------------------

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	procGetWindowTextW      = user32.NewProc("GetWindowTextW")
)

func InitWinFunctions() {
	if builtins == nil {
		builtins = make(map[string]BuiltinInfo)
	}

	ns := "win."

	// win.GetActiveTitle() -> Gibt den Titel des aktuell fokussierten Fensters zurück
	Register(ns+"GetActiveTitle", "win", "-", "Gibt den Titel des aktuell im Vordergrund befindlichen Fensters zurück.", func(args []Value) Value {
		hwnd, _, _ := procGetForegroundWindow.Call()
		if hwnd == 0 {
			return ErrorVal("Kein Fenster im Vordergrund")
		}

		b := make([]uint16, 256)
		n, _, _ := procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)))
		if n == 0 {
			return ErrorVal("Fenster-Titel konnte nicht gelesen werden")
		}
		return StrVal(syscall.UTF16ToString(b))
	})

	// event.Get(logName, levelName [, count])
	Register(ns+"GetEvent", "win", "logName, level, [count]", "Liest Ereignisse eines Typs (Error, Warn, Info) aus dem Windows-EventLog.", func(args []Value) Value {
		if len(args) < 2 {
			return ErrorVal("getevent benötigt Log-Name und Level (Fehler, Warnung, Info)")
		}

		levelInput := strings.ToLower(ToString(args[1]))
		levelID, style, label := 4, 37, "INFO "

		switch levelInput {
		case "fehler", "error", "err", "f", "2":
			levelID, style, label = 2, 31, "ERROR" // Rot
		case "warnung", "warn", "warning", "w", "3":
			levelID, style, label = 3, 33, "WARN " // Gelb
		case "information", "info", "i", "4":
			levelID, style, label = 4, 37, "INFO " // Weiß
		}

		count := 5
		if len(args) >= 3 {
			count = int(toNumVal(args[2]))
		}

		query := fmt.Sprintf("*[System[(Level=%d)]]", levelID)
		return runEventQuery(ToString(args[0]), query, count, label, style, true)
	})

	// win.SearchEventLog(log, text [, count])
	Register(ns+"SearchEventLog", "win", "log, text, [count]", "Durchsucht ein EventLog nach einem bestimmten Text in den Event-Daten oder Providern.", func(args []Value) Value {
		if len(args) < 2 {
			return ErrorVal("searchevent benötigt Log-Name und Suchtext")
		}
		searchText := ToString(args[1])
		count := 5
		if len(args) >= 3 {
			count = int(toNumVal(args[2]))
		}

		// Suche in Datenfeldern oder Provider-Namen
		query := fmt.Sprintf("*[EventData[Data[contains(.,'%s')]]] or *[System[Provider[@Name='%s']]]", searchText, searchText)
		return runEventQuery(ToString(args[0]), query, count, "SEARCH: "+searchText, 36, true)
	})

	// win.FindEventID(log, id [, count])
	Register(ns+"FindEventID", "win", "log, id, [count]", "Sucht im angegebenen Log nach einer spezifischen Event-ID.", func(args []Value) Value {
		if len(args) < 2 {
			return ErrorVal("findeventid benötigt Log-Name und Event-ID")
		}
		id := ToString(args[1])
		count := 5
		if len(args) >= 3 {
			count = int(toNumVal(args[2]))
		}

		query := fmt.Sprintf("*[System[(EventID=%s)]]", id)
		// Hier showID = false, da die ID bereits im Header steht
		return runEventQuery(ToString(args[0]), query, count, "ID: "+id, 95, false)
	})
}

func runEventQuery(logInput string, query string, count int, label string, style int, showID bool) Value {
	// 1. LOG-NAMEN ÜBERSETZUNG (Deutsch -> Windows Intern)
	logName := "System"
	switch strings.ToLower(logInput) {
	case "anwendung", "application", "app":
		logName = "Application"
	case "sicherheit", "security", "sec":
		logName = "Security"
	case "installation", "setup":
		logName = "Setup"
	case "system", "sys":
		logName = "System"
	}

	// wevtutil Befehl zusammenbauen
	cmd := exec.Command("wevtutil", "qe", logName, "/q:"+query, "/c:"+strconv.Itoa(count), "/rd:true", "/f:text")
	output, err := cmd.CombinedOutput()

	if err != nil {
		return ErrorVal("Fehler: Log '" + logInput + "' nicht gefunden oder keine Admin-Rechte.")
	}

	// Header-Ausgabe
	fmt.Printf("\033[90m--- [%s] %s (%d Einträge) ---\033[0m\n", strings.ToUpper(logName), label, count)

	// Events parsen (doppelter Zeilenumbruch trennt Einträge im Text-Format)
	events := strings.Split(string(output), "\r\n\r\n")
	for _, ev := range events {
		if strings.TrimSpace(ev) == "" {
			continue
		}

		lines := strings.Split(ev, "\r\n")
		var ts, src, msg, id, cat string

		for _, l := range lines {
			l = strings.TrimSpace(l)
			if strings.HasPrefix(l, "Date:") {
				ts = strings.TrimPrefix(l, "Date: ")
			}
			if strings.HasPrefix(l, "Source:") {
				src = strings.TrimPrefix(l, "Source: ")
			}
			if strings.HasPrefix(l, "Event ID:") {
				id = strings.TrimPrefix(l, "Event ID: ")
			}
			if strings.HasPrefix(l, "Task Category:") {
				cat = strings.TrimPrefix(l, "Task Category: ")
			}
			if strings.HasPrefix(l, "Message:") {
				msg = strings.TrimPrefix(l, "Message: ")
			}
		}

		// Zeitstempel kürzen für bessere Lesbarkeit
		if len(ts) > 19 {
			ts = ts[:19]
		}
		// Leere Kategorien abfangen
		if cat == "None" || cat == "Keine" || cat == "" {
			cat = "-"
		}

		// Formatierte Zeilenausgabe
		if showID {
			// Mit ID (für GetEvent und Search) - ID in Hell-Cyan (96)
			fmt.Printf("\033[%dm%s\033[0m [%s] \033[96mID:%s\033[0m \033[90m(%s)\033[0m %s: %s\n",
				style, label, ts, id, cat, src, msg)
		} else {
			// Ohne ID (für FindEventID)
			fmt.Printf("\033[%dm%s\033[0m [%s] \033[90m(%s)\033[0m %s: %s\n",
				style, label, ts, cat, src, msg)
		}
	}
	return BoolVal(true)
}

// Wird für Main gebraucht
func enableWindowsANSI() {
	stdout := windows.Handle(os.Stdout.Fd())
	var mode uint32
	err := windows.GetConsoleMode(stdout, &mode)
	if err == nil {
		// ENABLE_VIRTUAL_TERMINAL_PROCESSING = 0x0004
		windows.SetConsoleMode(stdout, mode|0x0004)
	}
}
