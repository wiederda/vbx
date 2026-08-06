package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

type Shortcut struct {
	Internal     string
	Args         string // z.B. "[pfad]" oder "[länge]"
	Beschreibung string
}

var TotalFunctions int
var ProductName string // wird via ldflags gesetzt
var Version string     // wird via ldflags gesetzt
var loadedModules = make(map[string]bool)

// var BuildDate string   // wird via ldflags gesetzt
var optionalModuleFunctions = map[string]int{
	"ad":       8,
	"cert":     12,
	"convert":  5,
	"computer": 14,
	"crypt":    6,
	"data":     15,
	"db":       30,
	"debug":    11,
	"docker":   31,
	"env":      3,
	"fin":      7,
	"geo":      1,
	"hash":     8,
	"ini":      8,
	"json":     24,
	"kuma":     5,
	"map":      11,
	"net":      17,
	"pgp":      11,
	"picture":  3,
	"pqc":      14,
	"rand":     6,
	"reg":      4,
	"service":  8,
	"smtp":     1,
	"steg":     4,
	"string":   31,
	"tar":      13,
	"template": 3,
	"yaml":     5,
	"xml":      9,
	"zip":      5,
	"win":      4,
}

func main() {
	enableWindowsANSI()
	// --- 1. Vor-Analyse der Argumente ---
	var positionalArgs []string // NEU: Sammelt alles ohne "-"
	var filter string
	showShell, isHelp, showReg := false, false, false
	regC()

	for _, arg := range os.Args[1:] {
		low := strings.ToLower(arg)
		switch {
		case low == "-h" || low == "--help" || low == "/?":
			isHelp = true
		case low == "-shell":
			showShell = true
		case low == "-reg":
			showReg = true
		case low == "-v" || low == "version":
			fmt.Println(Version)
			return
		case strings.HasPrefix(low, "-modules="):
			continue
		case strings.HasPrefix(low, "-"):
			filter = strings.TrimPrefix(low, "-")
		default:
			positionalArgs = append(positionalArgs, arg) // Speichere JEDES Argument
		}
	}

	var filename string
	if len(positionalArgs) > 0 {
		filename = positionalArgs[0] // Das erste ist dein Befehl oder Skript
	}

	// --- 2. Module & Umgebung für CLI/Hilfe vorbereiten ---
	// Das laden wir, damit Hilfe-Texte und Shortcuts funktionieren
	optionalModules := parseOptionalModulesFromCLI()
	if filename != "" && !isHelp && strings.HasSuffix(strings.ToLower(filename), ".vb") {
		// Nur bei Klartext-Dateien können wir hier schon Module für die Hilfe laden
		vbModules := parseOptionalModulesFromVBFile(filename)
		optionalModules = append(optionalModules, vbModules...)
	}

	mainEnv := NewEnvironment(nil)
	LoadModules(mainEnv, optionalModules)
	TotalFunctions = len(builtins)

	// --- 3. CLI Shortcuts ---
	var cliShortcuts = getCLIShortcuts()

	// --- 4. ENTSCHEIDUNGSLOGIK ---

	// FALL A: Hilfe-Modus
	if isHelp || (filename == "" && filter == "" && !showShell && !showReg) {
		if showReg {
			displayHelp()
			return
		}
		if filter == "" && len(optionalModules) == 1 {
			filter = optionalModules[0]
		}
		printHelp(filter, cliShortcuts, showShell)
		return
	}

	// FALL B: CLI-Shortcut (z.B. vbmini zipcreate ...)
	if filename != "" {
		// Wir normalisieren den Namen (entfernen evtl. Klammern)
		lookupName := strings.ToLower(strings.TrimSuffix(filename, "()"))

		if sh, ok := cliShortcuts[lookupName]; ok {
			// Wir müssen herausfinden, ab welchem Index die echten Argumente beginnen.
			// Wenn 'filename' os.Args[1] war, dann sind die Args ab os.Args[2:].
			// Wir suchen die Position von filename in os.Args:
			argStart := 1
			for i, a := range os.Args {
				if a == filename {
					argStart = i + 1
					break
				}
			}
			executeShortcut(sh, os.Args[argStart:])
			os.Exit(0)
		}
	}

	// FALL C: Normales Skript ausführen
	if filename != "" {
		// HIER passiert jetzt die magische Prüfung (Version + Module + Decrypt)
		if err := RunFile(filename); err != nil {
			fmt.Printf("\n\033[31m!---------------------------------------------------------!\033[0m")
			fmt.Printf("\n  \033[1mFEHLER:\033[0m %v\n", err)
			fmt.Printf("\033[31m!---------------------------------------------------------!\033[0m\n\n")
			os.Exit(1)
		}
	}
}

func Register(name string, module string, params string, desc string, fn func([]Value) Value) {
	pCount := 0

	p := strings.TrimSpace(params)
	if p != "" && p != "-" {
		if strings.Contains(p, "[") || strings.Contains(p, "...") {
			pCount = -1
		} else {
			pCount = len(strings.Split(p, ","))
		}
	}

	builtins[name] = BuiltinInfo{
		Fn:           fn,
		Module:       module,
		Params:       params,
		Beschreibung: desc,
		ParamCount:   pCount,
	}
}

func printHelp(filter string, shortcuts map[string]Shortcut, showShell bool) {
	// ANSI-Farben
	bold := "\033[1m"
	green := "\033[32m"
	cyan := "\033[36m"
	reset := "\033[0m"
	boldMagenta := "\033[1;35m"

	// Filter bereinigen (aus "-tar" wird "tar")
	cleanFilter := strings.TrimPrefix(strings.ToLower(filter), "-")

	// --- Dynamische Zählung für den Header ---
	mandatoryNames := map[string]bool{
		"app": true, "array": true, "date": true, "file": true, "folder": true, "global": true, "math": true,
	}

	counts := make(map[string]int)
	for k := range builtins {
		parts := strings.Split(k, ".")
		ns := "global"
		if len(parts) > 1 {
			ns = parts[0]
		}
		counts[ns]++
	}

	standardCount := 0
	for ns, c := range counts {
		if mandatoryNames[ns] {
			standardCount += c
		}
	}

	optionalCount := 0
	for name, staticCount := range optionalModuleFunctions {
		if realCount, loaded := counts[name]; loaded {
			optionalCount += realCount
		} else {
			optionalCount += staticCount
		}
	}

	totalCount := standardCount + optionalCount

	fmt.Printf("Version: "+bold+"%s"+bold+" | Funktionen: "+reset+"Standard "+green+"%d"+reset+" / Optional "+cyan+"%d"+reset+" / Gesamt "+boldMagenta+"%d"+reset+"\n",
		Version, standardCount, optionalCount, totalCount)

	if !showShell {
		fmt.Println("\n" + bold + "Kurz-Übersicht:" + reset)
		fmt.Println("  vbx <quelle.vb>                  - Skript ausführen")
		fmt.Println("  vbx -modules=zip <quelle.vb>     - optionales Modul vor Skriptstart laden")
		fmt.Println("  #use zip                         - optionales Modul im Skript laden")
		fmt.Println("                                     (erste Zeile, auch in include-Dateien)")
		fmt.Println("  #requires 1.0.23                 - Benötigte Mindestversion")
		fmt.Println("  include \"datei.vb\"               - Weitere VB-Datei einbinden")
		fmt.Println("  vbx -h                           - Diese Übersicht")
		fmt.Println("  vbx -v                           - Aktuelle Versionsnummer")
		fmt.Println("  vbx -shell -h                    - Verfügbare Shell-Befehle")
		fmt.Println("  vbx -reg -h                      - Verfügbare Konstanten")
		fmt.Println("  ' oder /' ... '/                 - Eine Zeile oder einen ganzen Block auskommentieren")

		if cleanFilter != "" {
			fmt.Printf("\n"+bold+"Aktivierter Filter: "+cyan+"%s"+reset+"\n", cleanFilter)
		}
	}

	printModulesHelp(cleanFilter)

	if showShell {
		printShellHelp(shortcuts, cleanFilter)
	}
}

// ---------------- Module anzeigen ----------------
func printModulesHelp(filter string) {
	bold := "\033[1m"
	green := "\033[32m"
	cyan := "\033[36m"
	white := "\033[37m"
	reset := "\033[0m"

	if filter != "" {
		var found []string
		filterLower := strings.ToLower(filter)

		for name, info := range builtins {
			lowName := strings.ToLower(name)
			hasDot := strings.Contains(lowName, ".")
			isMatch := false

			switch filterLower {
			case "global":
				// Nur Funktionen: Kein Punkt UND kein "vb"-Präfix
				if !hasDot && !strings.HasPrefix(lowName, "vb") {
					isMatch = true
				}

			case "reg":
				// Nur Register/Konstanten: Kein Punkt UND "vb"-Präfix
				if !hasDot && strings.HasPrefix(lowName, "vb") {
					isMatch = true
				}

			default:
				// Normaler Modul-Filter (math, file, etc.)
				lowModule := strings.ToLower(info.Module)
				if lowModule == filterLower || strings.HasPrefix(lowName, filterLower+".") {
					isMatch = true
				}
			}

			if isMatch {
				found = append(found, name)
			}
		}

		// Dubletten verhindern (falls beide Bedingungen zutreffen)
		found = uniqueStrings(found)
		sort.Strings(found)

		if len(found) == 0 {
			fmt.Println("  " + white + "(Keine exakten Treffer gefunden)" + reset)
		}

		for _, name := range found {
			info := builtins[name]
			params := ""
			if info.Params != "" {
				params = "(" + info.Params + ")"
			}
			fmt.Printf("  %s%-30s%s %s\n", green, name+params, reset, info.Beschreibung)
		}
		return
	}

	// 2. Wenn KEIN Filter gesetzt ist: Die gewohnte Modul-Übersicht anzeigen
	counts := make(map[string]int)
	for k := range builtins {
		parts := strings.Split(k, ".")
		ns := "global"
		if len(parts) > 1 {
			ns = parts[0]
		}
		counts[ns]++
	}

	// --- Standard Module ---
	mandatoryModules := []string{"app", "array", "date", "file", "folder", "global", "math"}
	fmt.Println("\n" + bold + "Module/Funktionsgruppen (Standard):" + reset)
	for _, ns := range mandatoryModules {
		if c, ok := counts[ns]; ok {
			fmt.Printf("  "+green+"%-10s"+reset+" ("+white+"%2d Funktionen"+reset+")\n", ns, c)
		}
	}

	// --- Optionale Module ---
	fmt.Println("\n" + bold + "Optionale Module (via -modules= oder #use):" + reset)
	var optKeys []string
	for k := range optionalModuleFunctions {
		optKeys = append(optKeys, k)
	}
	sort.Strings(optKeys)

	for _, name := range optKeys {
		displayCount := optionalModuleFunctions[name]
		if realCount, loaded := counts[name]; loaded {
			displayCount = realCount
		}
		fmt.Printf("  "+cyan+"%-10s"+reset+" ("+white+"%2d Funktionen"+reset+")\n", name, displayCount)
	}
}

func printShellHelp(shortcuts map[string]Shortcut, filter string) {
	fmt.Println("\n\x1b[1mVerfügbare Shell-Befehle:\x1b[0m")

	sKeys := make([]string, 0, len(shortcuts))
	maxArgs, maxFunc := 0, 0
	filterLower := strings.ToLower(filter)

	// 1. Sammeln und Breiten berechnen
	for sk, sh := range shortcuts {
		match := filter == "" ||
			strings.Contains(strings.ToLower(sk), filterLower) ||
			strings.Contains(strings.ToLower(sh.Internal), filterLower) ||
			strings.Contains(strings.ToLower(sh.Beschreibung), filterLower)

		if match {
			sKeys = append(sKeys, sk)
			if len(sh.Args) > maxArgs {
				maxArgs = len(sh.Args)
			}
			if len(sh.Internal) > maxFunc {
				maxFunc = len(sh.Internal)
			}
		}
	}

	// --- NEU: Sortierung nach Namespace (sh.Internal), dann nach Name (sk) ---
	sort.Slice(sKeys, func(i, j int) bool {
		shI, shJ := shortcuts[sKeys[i]], shortcuts[sKeys[j]]
		if shI.Internal != shJ.Internal {
			return shI.Internal < shJ.Internal
		}
		return sKeys[i] < sKeys[j]
	})

	if len(sKeys) == 0 {
		fmt.Printf("  \x1b[31m(Keine Shell-Befehle für Filter '%s' gefunden)\x1b[0m\n", filter)
		return
	}

	paddingArgs := maxArgs + 2
	paddingFunc := maxFunc + 2

	// --- NEU: Ausgabe mit Gruppen-Trennern ---
	var lastNamespace string
	for _, sk := range sKeys {
		sh := shortcuts[sk]

		// Namespace extrahieren (alles vor dem ersten Punkt in sh.Internal)
		currentNamespace := "sys"
		if idx := strings.Index(sh.Internal, "."); idx != -1 {
			currentNamespace = sh.Internal[:idx]
		}

		// Trenner einfügen, wenn eine neue Gruppe beginnt
		if lastNamespace != "" && currentNamespace != lastNamespace {
			fmt.Printf("  \x1b[90m%s\x1b[0m\n", strings.Repeat("-", 80))
		}

		fmt.Printf("  \x1b[36m%-15s\x1b[0m \x1b[33m%-*s\x1b[0m -> \x1b[32m%-*s\x1b[0m %s\n",
			sk,
			paddingArgs,
			sh.Args,
			paddingFunc,
			sh.Internal,
			sh.Beschreibung)

		lastNamespace = currentNamespace
	}
	fmt.Println()
}

func totalOptionalFunctions() int {
	total := 0
	for _, v := range optionalModuleFunctions {
		total += v
	}
	return total
}

func moduleFromInternal(internal string) string {
	parts := strings.Split(internal, ".")
	if len(parts) > 1 {
		return parts[0]
	}
	return ""
}

// Prüft, ob ein Modul schon geladen wurde
func isModuleLoaded(ns string) bool {
	loaded, ok := loadedModules[ns]
	return ok && loaded
}

func handleShortcutResult(result Value) {
	switch result.Kind {
	case KindStr:
		fmt.Println(result.Str)
	case KindNum:
		fmt.Println(result.Num)
	case KindBool:
		// Wir machen hier einfach nichts.
		// Wenn ein Shortcut true/false liefert, hat er seine Arbeit
		// (wie das Drucken der Logs) meist schon erledigt.
	case KindArr:
		for _, v := range result.Arr {
			fmt.Println(v.Str)
		}
	case KindError:
		fmt.Printf("\x1b[31mFehler: %s\x1b[0m\n", result.Str)
	}
}

func getCLIShortcuts() map[string]Shortcut {
	// Deine bestehende Map...
	m := map[string]Shortcut{
		"aesdecrypt":     {"crypt.AESDecrypt", "ciphertext, [password]", "AES-256-GCM Entschlüsselung, Rückgabe Text"},
		"aesencrypt":     {"crypt.AESEncrypt", "text, [password]", "AES-256-GCM Verschlüsselung, Rückgabe Base64"},
		"agestring":      {"date.AgeString", "tag, monat, jahr [, stunde [, minute]]", "Alter als lesbarer Text, Jahre, Monate, Tag"},
		"build":          {"Build", "path", "Kompiliert ein Projekt im Zielverzeichnis als vbc"},
		"createconf":     {"cert.CreateConf", "", "Keine Argumente. Generiert ein openssl.conf"},
		"distro":         {"computer.Distro", "", "Keine Parameter, gibt die Distribution zuück"},
		"file64decode":   {"file.Base64Decode", "infile, outfile", "Base64-Datei dekodieren"},
		"file64encode":   {"file.Base64Encode", "infile, outfile", "Datei Base64-kodieren (Standard-Variante)"},
		"filereplaceall": {"file.ReplaceAll", "path, old, new", ""},
		"findeventid":    {"win.FindEventID", "log, id, cnt", "Sucht gezielt nach einer Windows Event-ID"},
		"foldercount":    {"folder.Count", "path", "Zählt alle Dateien und Unterordner..."},
		"getevent":       {"win.GetEvent", "log, lvl, cnt", "Ruft die letzten Windows-Events ab (Error, Warn, Info)"},
		"gzcreate":       {"tar.GzCreate", "tarGzPath, paths...", "TAR.GZ erstellen, Ordnerstruktur wird beibehalten"},
		"gzcreateflat":   {"tar.GzCreateFlat", "tarGzPath, paths...", "TAR.GZ erstellen ohne Ordnerstruktur"},
		"categories":     {"convert.Categories", "", "Gibt alle Unit-Kategorien (Convert) zurück"},
		"units":          {"convert.Units", "category", "Gibt alle Units einer Kategorie (Convert) zurück"},
		"gzextract":      {"tar.GzExtract", "tarGzPath, paths...", "TAR.GZ entpacken"},
		"randomstring":   {"crypt.RandomString", "[length]", "zufälliger alphanumerischer String (mindestens 10 Zeichen)"},
		"searchevent":    {"win.SearchEventLog", "log, text, cnt", "Durchsucht das Event-Log nach Volltext"},
		"string64decode": {"DecodeBase64", "text", "Dekodiert Base64-String"},
		"string64encode": {"EncodeBase64", "text", "Kodiert String nach Base64"},
		"tail":           {"file.Tail", "path, lines, [refresh]", "Zeigt Dateiende..."},
		"unblock":        {"Unblock", "path", "Entfernt Windows-Zone.Identifier (nur Windows)"},
		"watchlog":       {"file.WatchLog", "path, pat, sty", "Überwacht eine Textdatei live auf Suchmuster"},
		"zipcreate":      {"zip.Create", "zipPath, paths..., [password]", "ZIP erstellen, Ordnerstruktur bleibt erhalten"},
		"zipcreateflat":  {"zip.CreateFlat", "zipPath, paths..., [password]", "ZIP erstellen ohne Ordnerstruktur"},
		"zipextract":     {"zip.Extract", "zipPath, dest, [password]", "ZIP entpacken"},
		"worker":         {"Worker", "vbPath", "Startet ein VB Skript autark im Hintergrund."},
		"optimizefile":   {"db.OptimizeFile", "path, dialect [, outPath]", "Optimiert eine SQL-Datei direkt."},
		"uptime":         {"UptimeString", "", "System-Laufzeit als formatierter Text"},
		"convnewline":    {"ConvNewLine", "content, [target]", "Normalisiert Zeilenumbrüche. Standard: Aktuelles OS (auto). Optional: 'crlf', 'lf', 'cr'."},
		"reboot":         {"computer.Reboot", "", "Startet das System sofort neu."},
		"shutdown":       {"computer.Shutdown", "", "Fährt das System sofort herunter."},
		"pqcidentity":    {"pqc.SetupIdentity", "path, encrypt", "Erzeugt Keys.(id_pqc.pub / id_pqc) Bei 'encrypt=false' erfolgt eine Sicherheitswarnung!"},
		"pgpidentity":    {"pgp.SetupIdentity", "folder, name, email, [password]", "Erstellt PGP-Keys (id_pgp.pub / id_pgp). Optional mit Passwort."},
		"printformat":    {"PrintFormat", "", "Gibt eine Übersicht der Format-Optionen auf der Konsole aus."},
		"copyfolder":     {"folder.Copy", "src, dst, [progress]", "Kopiert einen Ordner rekursiv mit parallelen Workern an einen neuen Ort. progress=True zeigt Fortschritt auf der Konsole."},
		"rename":         {"file.Rename", "alt, neu", "Benennt eine Datei um."},
		"qcsignidentity": {"pqc.SetupSignIdentity", "path, encrypt", "Erzeugt Keys.(id_sig.pub / id_sig) Bei 'encrypt=false' erfolgt eine Sicherheitswarnung!"},
		"generatesshkey": {"GenerateSSHKey", "[outFile, algo, bits, pass]",
			"Erstellt ein SSH-Paar. RSA mit min. 4096 Bit. Ed25519 nutzt fix 256 Bit. Überschreibt keine vorhandenen Dateien im .ssh Ordner der Users."},
	}
	return m
}

func executeShortcut(sh Shortcut, rawArgs []string) {
	ns := moduleFromInternal(sh.Internal)
	if ns != "" && !isModuleLoaded(ns) {
		LoadOptionalModule(nil, ns)
	}

	if info, ok := builtins[sh.Internal]; ok {
		var cmdArgs []Value
		for _, arg := range rawArgs {
			cleanArg := strings.Trim(arg, " \"")
			if cleanArg != "" {
				cmdArgs = append(cmdArgs, Value{Kind: KindStr, Str: cleanArg})
			}
		}

		// --- DER WOLF-SCHUTZ FÜR SHORTCUTS ---
		if info.ParamCount != -1 && len(cmdArgs) != info.ParamCount {
			fmt.Printf("\n\033[31m[PARAMETER FEHLER]\033[0m\n")
			fmt.Printf("Der Befehl '%s' erwartet %d Argumente, du hast %d angegeben.\n",
				sh.Internal, info.ParamCount, len(cmdArgs))
			fmt.Printf("Syntax: %s %s\n", sh.Internal, info.Params)
			os.Exit(1)
		}

		handleShortcutResult(info.Fn(cmdArgs))
	} else {
		fmt.Printf("\n\033[31m[SHORTCUT FEHLER]\033[0m\n")
		fmt.Printf("Funktion '%s' im Modul '%s' nicht gefunden!\n", sh.Internal, ns)
		os.Exit(1)
	}
}

func displayHelp() {
	bold := "\033[1m"
	cyan := "\033[36m"
	green := "\033[32m"
	reset := "\033[0m"
	white := "\033[37m"

	// Hilfsfunktion zum Extrahieren der Kategorie [Kat]
	getCat := func(s string) string {
		if strings.HasPrefix(s, "[") && strings.Contains(s, "]") {
			return s[1:strings.Index(s, "]")]
		}
		return "Z-Sonstiges"
	}

	fmt.Printf("\n%svbx %s - Verfügbare Konstanten%s\n", bold, Version, reset)
	fmt.Println(strings.Repeat("-", 65))
	fmt.Printf("%s%-22s%s | %s\n", cyan, "Konstante", reset, "Beschreibung")
	fmt.Println(strings.Repeat("-", 65))

	// 1. Sortierung: Erst nach Kategorie, dann alphabetisch nach Name
	sort.Slice(globalRegistry, func(i, j int) bool {
		catI := getCat(globalRegistry[i].Description)
		catJ := getCat(globalRegistry[j].Description)
		if catI != catJ {
			return catI < catJ
		}
		return globalRegistry[i].Name < globalRegistry[j].Name
	})

	var lastCat string
	for _, entry := range globalRegistry {
		currentCat := getCat(entry.Description)

		// 2. Trenner einfügen bei Kategoriewechsel
		if lastCat != "" && currentCat != lastCat {
			fmt.Printf("%s%s%s\n", white, strings.Repeat("-", 65), reset)
		}

		displayName := entry.Name + "()"

		fmt.Printf("%s%-22s%s | %s\n",
			green,
			displayName,
			reset,
			entry.Description)

		lastCat = currentCat
	}

	fmt.Println(strings.Repeat("-", 65))
}

// Hilfsfunktion zum Aufräumen der Liste
func uniqueStrings(strSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range strSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
