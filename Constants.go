package main

import (
	"fmt"
)

// Hier werden alle Konstanten gespeichert, die via regC registriert werden
var globalRegistry []RegInfo

func getAnsiCode(val float64) string {
	v := int(val)
	if v == -1 {
		return "\033[0m"
	} // vbNormal

	// Stile (100, 200, 300)
	if v >= 100 && v < 1000 {
		switch v {
		case 100:
			return "\033[1m" // Bold
		case 200:
			return "\033[4m" // Underline
		case 300:
			return "\033[7m" // Invert
		}
	}

	// Hintergrund (1000, 2000, ...)
	if v >= 1000 {
		colorIdx := v / 1000
		if colorIdx <= 7 {
			return fmt.Sprintf("\033[%dm", 40+colorIdx)
		}
		return fmt.Sprintf("\033[48;5;%dm", colorIdx) // Extended Colors
	}

	// Vordergrund (0-15)
	if v >= 0 && v <= 7 {
		return fmt.Sprintf("\033[%dm", 30+v)
	}
	if v >= 8 && v <= 15 {
		return fmt.Sprintf("\033[%dm", 90+(v-8))
	}

	return ""
}

func regC() {
	// Der interne Helfer bleibt gleich, registriert nun aber die sauberen Werte
	register := func(category, name, desc string, val Value) {
		globalRegistry = append(globalRegistry, RegInfo{
			Name:        name,
			Kind:        "const",
			Description: "[" + category + "] " + desc,
		})

		Register(name, "const", "", desc, func(args []Value) Value {
			return val
		})
	}

	// --- GRUPPE: LOGIK & TYPEN ---
	// Hier nutzen wir jetzt echte Booleans für maximale Konsistenz
	register("Logic", "vbTrue", "Wahr", BoolVal(true))
	register("Logic", "vbFalse", "Falsch", BoolVal(false))
	register("Logic", "vbNullString", "Leerer String", StrVal(""))

	// --- GRUPPE: SYSTEM & FORMAT ---
	register("System", "vbCrLf", "Windows-Zeilenumbruch (CRLF)", StrVal("\r\n"))
	register("System", "vbNewLine", "System-Zeilenumbruch (LF)", StrVal("\n"))
	register("System", "vbTab", "Tabulator-Zeichen", StrVal("\t"))

	// --- GRUPPE: VORDERGRUNDFARBEN (ANSI-Strings) ---
	register("Color", "vbBlack", "Farbe: Schwarz", StrVal("\033[30m"))
	register("Color", "vbRed", "Farbe: Rot", StrVal("\033[91m"))
	register("Color", "vbGreen", "Farbe: Grün", StrVal("\033[92m"))
	register("Color", "vbYellow", "Farbe: Gelb", StrVal("\033[93m"))
	register("Color", "vbBlue", "Farbe: Blau", StrVal("\033[94m"))
	register("Color", "vbWhite", "Farbe: Weiß", StrVal("\033[37m"))
	register("Color", "vbCyan", "Farbe: Cyan (Hellblau)", StrVal("\033[36m"))
	register("Color", "vbMagenta", "Farbe: Magenta (Lila)", StrVal("\033[35m"))
	register("Color", "vbLightGray", "Farbe: Hellgrau", StrVal("\033[37m"))
	register("Color", "vbGray", "Farbe: Grau", StrVal("\033[90m"))

	// --- GRUPPE: HINTERGRUNDFARBEN (ANSI-Strings) ---
	register("Background", "vbBgBlack", "Hintergrund: Schwarz", StrVal("\033[40m"))
	register("Background", "vbBgRed", "Hintergrund: Rot", StrVal("\033[41m"))
	register("Background", "vbBgGreen", "Hintergrund: Grün", StrVal("\033[42m"))
	register("Background", "vbBgYellow", "Hintergrund: Gelb", StrVal("\033[43m"))
	register("Background", "vbBgBlue", "Hintergrund: Blau", StrVal("\033[44m"))
	register("Background", "vbBgWhite", "Hintergrund: Weiß", StrVal("\033[47m"))
	register("Background", "vbBgCyan", "Hintergrund: Cyan", StrVal("\033[46m"))
	register("Background", "vbBgMagenta", "Hintergrund: Magenta", StrVal("\033[45m"))

	// --- GRUPPE: STILE (ANSI-Strings) ---
	register("Style", "vbBold", "Stil: Fett", StrVal("\033[1m"))
	register("Style", "vbUnderline", "Stil: Unterstrichen", StrVal("\033[4m"))
	register("Style", "vbNormal", "Reset: Stile & Farben zurücksetzen", StrVal("\033[0m"))

}
