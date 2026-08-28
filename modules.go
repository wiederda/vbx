package main

import (
	"bufio"
	"os"
	"strings"
)

// ---------------- Modul-Loader ----------------

var mandatoryModulesMap = map[string]func(*Environment){
	"global": func(e *Environment) { InitGlobal() },
	"math":   func(e *Environment) { InitMathFunctions() },
	"app":    func(e *Environment) { InitAppFunctions() },
	"array":  func(e *Environment) { InitArrayFunctions() },
	"date":   func(e *Environment) { InitDateFunctions() },
	"file":   func(e *Environment) { InitFileFunctions() },
	"folder": func(e *Environment) { InitFolderFunctions() },
}

var optionalModulesMap = map[string]func(*Environment){
	"debug":   InitDebugFunctions,
	"7z":      func(e *Environment) { InitSevenZipFunctions() },
	"ad":      func(e *Environment) { InitADFunctions() },
	"db":      func(e *Environment) { InitDBFunctions() },
	"env":     func(e *Environment) { InitEnvFunctions() },
	"cert":    func(e *Environment) { InitCertFunctions() },
	"docker":  func(e *Environment) { InitDockerFunctions() },
	"string":  func(e *Environment) { InitStringFunctions() },
	"convert": func(e *Environment) { InitConvertFunctions() },
	"crypt":   func(e *Environment) { InitCryptFunctions() },
	"geo":     func(e *Environment) { InitGeoFunctions() },
	"git":     func(e *Environment) { InitGitFunctions() },
	"yaml":    func(e *Environment) { InitYamlFunctions() },
	"smtp":    func(e *Environment) { InitSmtpFunctions() },
	"service": func(e *Environment) { InitServiceFunctions() },
	//"gui":      func(e *Environment) { InitGUIFunctions() },
	"data":     func(e *Environment) { InitDataFunctions() },
	"fin":      func(e *Environment) { InitFinFunctions() },
	"computer": func(e *Environment) { InitComputerFunctions() },
	"map":      func(e *Environment) { InitMapFunctions() },
	"ini":      func(e *Environment) { InitIniFunctions() },
	"json":     func(e *Environment) { InitJsonFunctions() },
	"kuma":     func(e *Environment) { InitKumaFunctions() },
	"tar":      func(e *Environment) { InitTarFunctions() },
	"zip":      func(e *Environment) { InitZipFunctions() },
	"reg":      func(e *Environment) { InitRegistryFunctions() },
	"picture":  func(e *Environment) { InitPictureFunctions() },
	"pqc":      func(e *Environment) { InitPQCFunctions() },
	"pgp":      func(e *Environment) { InitPGPFunctions() },
	"xml":      func(e *Environment) { InitXmlFunctions() },
	"net":      func(e *Environment) { InitNetFunctions() },
	"rand":     func(e *Environment) { InitRandFunctions() },
	"proc":     func(e *Environment) { InitProcFunctions() },
	"sftp":     func(e *Environment) { InitSftpFunctions() },
	"ssh":      func(e *Environment) { InitSSHFunctions() },
	"steg":     func(e *Environment) { InitStegFunctions() },
	"template": func(e *Environment) { InitTemplateFunctions() },
	"win":      func(e *Environment) { InitWinFunctions() },
}

// LoadModules lädt zuerst die Pflichtmodule und anschließend nur die optionalen Module, die angegeben wurden
func LoadModules(e *Environment, optionals []string) {
	// 1. Pflichtmodule IMMER laden
	for mod, initFn := range mandatoryModulesMap {
		// Wir reichen das Environment 'e' an die (ggf. gewrappte) Funktion weiter
		initFn(e)
		loadedModules[mod] = true
	}

	// 2. Optionale Module (aus -modules= oder #use)
	for _, opt := range optionals {
		opt = strings.TrimSpace(strings.ToLower(opt))
		if opt != "" {
			// Auch hier muss 'e' mit auf die Reise
			LoadOptionalModule(e, opt)
		}
	}
}

// Hilfsfunktion: optionales Modul laden, wenn noch nicht geladen
func LoadOptionalModule(e *Environment, ns string) {
	// Falls schon geladen, nichts tun
	if loaded, ok := loadedModules[ns]; ok && loaded {
		return
	}

	// In der Map nachschauen und mit 'e' ausführen
	if initFn, ok := optionalModulesMap[ns]; ok {
		initFn(e)
		loadedModules[ns] = true
	}
}

// ---------------- CLI Parsing ----------------

// parseOptionalModulesFromCLI liest optional angegebene Module über "-modules=cert,sqlite"
func parseOptionalModulesFromCLI() []string {
	var modules []string
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "-modules=") {
			list := strings.TrimPrefix(arg, "-modules=")
			for _, m := range strings.Split(list, ",") {
				if m != "" {
					modules = append(modules, strings.ToLower(strings.TrimSpace(m)))
				}
			}
		}
	}
	return modules
}

// parseOptionalModulesFromVBFile liest die ersten 10 Zeilen einer VB-Datei
// und gibt alle Module zurück, die mit "#use" angegeben wurden
func parseOptionalModulesFromVBFile(filename string) []string {
	file, err := os.Open(filename)
	if err != nil {
		return nil
	}
	defer file.Close()

	var modules []string
	scanner := bufio.NewScanner(file)
	for i := 0; i < 10 && scanner.Scan(); i++ {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(strings.ToLower(line), "#use") {
			line = strings.TrimSpace(line[4:]) // "#use" entfernen
			if line != "" {
				for _, m := range strings.Split(line, ",") {
					modules = append(modules, strings.ToLower(strings.TrimSpace(m)))
				}
			}
			break
		}
	}

	if err := scanner.Err(); err != nil {
		// Lesefehler ignorieren wir bewusst - bereits gesammelte Module bleiben gültig,
		// und die Funktion hat ohnehin nur eine Best-Effort-Semantik (siehe Rückgabe nil bei Open-Fehler)
		return modules
	}

	return modules
}

func parseRequiredVersionFromVBFile(filename string) string {
	file, err := os.Open(filename)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// 1. Zeile überspringen (#use)
	if !scanner.Scan() {
		return ""
	}

	// 2. Zeile prüfen (#requires)
	if scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(strings.ToLower(line), "#requires") {
			// " #requires " abschneiden und Rest trimmen
			return strings.TrimSpace(line[9:])
		}
	}

	return ""
}
