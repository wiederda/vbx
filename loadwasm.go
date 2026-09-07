package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

const expectedABIVersion = 1

// ------------------------------------------------------------
// Plugin-Debug
// ------------------------------------------------------------
//
// Im normalen Betrieb false.
// Für detaillierte Plugin-Diagnose auf true setzen.
//

const pluginDebug = false

func pluginDebugf(format string, args ...interface{}) {
	if pluginDebug {
		fmt.Printf("[PLUGIN DEBUG] "+format+"\n", args...)
	}
}

// ------------------------------------------------------------
// Plugin-Beschreibung
// ------------------------------------------------------------

type pluginFuncDesc struct {
	Namespace   string `json:"namespace"`
	Name        string `json:"name"`
	Params      string `json:"params"`
	Description string `json:"description"`
}

type pluginHandle struct {
	runtime wazero.Runtime
	module  api.Module
}

var pluginBaseDir string
var loadedPlugins = map[string]*pluginHandle{}

// ------------------------------------------------------------
// Plugin-Basisverzeichnis
// ------------------------------------------------------------

func resolvePluginBaseDir() (string, error) {
	if pluginBaseDir != "" {
		return pluginBaseDir, nil
	}

	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf(
			"Arbeitsverzeichnis konnte nicht ermittelt werden: %w",
			err,
		)
	}

	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}

	pluginBaseDir = abs

	return pluginBaseDir, nil
}

// ------------------------------------------------------------
// Sicheren Pfad erzeugen
// ------------------------------------------------------------
//
// Wird ausschließlich von write_file genutzt (siehe Hinweis
// dort). Arbeitet rein mit Host-Pfaden (kein WASI-Guest-Pfad
// beteiligt), da write_file vom Plugin nur relative
// Unterpfade unterhalb von pluginBaseDir entgegennimmt.
//

func safeJoin(base, requested string) (string, error) {
	joined := filepath.Join(base, requested)

	cleanBase := filepath.Clean(base)
	cleanJoined := filepath.Clean(joined)

	if cleanJoined != cleanBase &&
		!strings.HasPrefix(
			cleanJoined+string(os.PathSeparator),
			cleanBase+string(os.PathSeparator),
		) {
		return "", fmt.Errorf(
			"Pfad %q liegt außerhalb des erlaubten Verzeichnisses",
			requested,
		)
	}

	return cleanJoined, nil
}

// ------------------------------------------------------------
// Guest-Dateisystem (WASI)
// ------------------------------------------------------------
//
// Damit Plugins direkte Dateizugriffe (os.Stat, os.Open, ...)
// nutzen können - z.B. git.IsRepo/git.Status, die intern
// go-gits PlainOpen aufrufen - braucht das instanziierte
// WASI-Modul mindestens eine Preopened-Directory. Ohne das
// scheitert JEDER Dateizugriff des Plugins unabhängig vom
// übergebenen Pfad (siehe git.IsRepo-Bug).
//
// Linux/macOS: Host- und Gast-Pfade sind ohnehin POSIX-artig,
// daher genügt ein 1:1-Mount von "/" auf "/".
//
// Windows: WASI-Gastpfade kennen keine Laufwerksbuchstaben.
// Jedes vorhandene Laufwerk wird unter einem eigenen
// Guest-Wurzelpfad gemountet, z.B. C:\ -> /c, D:\ -> /d
// (WSL-Konvention) - siehe hostToGuestPath für die
// dazugehörige Pfad-Übersetzung der Argumente.
//
// WICHTIGER HINWEIS zur Sandbox-Konsistenz:
// Dieses Mounting öffnet dem Plugin vollen Lese-/Schreibzugriff
// auf das gesamte gemountete Verzeichnis (praktisch die ganze
// Festplatte) für alle direkten os.*-Aufrufe im Plugin-Code.
// Das ist bewusst breiter als die Beschränkung, die
// safeJoin/resolvePluginBaseDir für write_file durchsetzen -
// ein Plugin kann also über einen normalen os.WriteFile()-Aufruf
// diese Beschränkung umgehen. Das ist hier akzeptiert, weil
// Plugins aktuell nur selbstgeschrieben/vertrauenswürdig sind
// und VBX-Skripte ohnehin nativen Vollzugriff aufs Dateisystem
// haben. write_file bleibt trotzdem bestehen für Plugins, die
// bewusst NUR über die Host-Bridge schreiben wollen (z.B. weil
// sie kein Preopened-Directory für ihren Zielpfad haben).
//

func buildGuestFSConfig() (wazero.FSConfig, error) {

	fsConfig := wazero.NewFSConfig()

	if runtime.GOOS != "windows" {
		return fsConfig.WithDirMount("/", "/"), nil
	}

	mounted := false

	for letter := 'A'; letter <= 'Z'; letter++ {

		drive := string(letter) + `:\`

		if _, err := os.Stat(drive); err != nil {
			continue // Laufwerk existiert nicht
		}

		guestRoot := "/" + strings.ToLower(string(letter))

		fsConfig = fsConfig.WithDirMount(drive, guestRoot)
		mounted = true
	}

	if !mounted {
		return nil, fmt.Errorf(
			"kein Laufwerk zum Mounten gefunden",
		)
	}

	return fsConfig, nil
}

// ------------------------------------------------------------
// Host-Pfad <-> Gast-Pfad Übersetzung
// ------------------------------------------------------------
//
// Plugin-Funktionsargumente laufen generisch über toJSONValue
// (siehe unten). Da der Host nicht weiß, welches Argument
// einer Plugin-Funktion "ein Pfad" ist (nur Freitext in
// Params, z.B. "url, path"), wird heuristisch erkannt: sieht
// ein String wie ein Host-Dateipfad aus, wird er auf das
// Guest-FS-Layout aus buildGuestFSConfig übersetzt.
//
// Bewusste Einschränkung: nur absolute Pfade (und ".") werden
// erkannt. Ein relativer Pfad wie "meinrepo" wird NICHT
// übersetzt, da sonst z.B. Commit-Messages, die zufällig mit
// "/" beginnen, fälschlich als Pfad behandelt würden. Für
// Plugin-Aufrufe mit relativen Pfaden gilt: absolute Pfade
// verwenden.
//

func looksLikeHostPath(s string) bool {

	if s == "." {
		return true
	}

	if strings.HasPrefix(s, "/") {
		return true
	}

	// Windows: "C:\" oder "C:/"
	if len(s) >= 3 &&
		((s[0] >= 'A' && s[0] <= 'Z') ||
			(s[0] >= 'a' && s[0] <= 'z')) &&
		s[1] == ':' &&
		(s[2] == '\\' || s[2] == '/') {
		return true
	}

	return false
}

func hostToGuestPath(hostPath string) (string, error) {

	abs, err := filepath.Abs(hostPath)
	if err != nil {
		return "", err
	}

	if runtime.GOOS != "windows" {
		return filepath.ToSlash(abs), nil
	}

	if len(abs) < 2 || abs[1] != ':' {
		return "", fmt.Errorf(
			"kein gültiger Windows-Pfad: %q",
			abs,
		)
	}

	drive := strings.ToLower(string(abs[0]))
	rest := strings.ReplaceAll(abs[2:], `\`, "/")

	return "/" + drive + rest, nil
}

// ------------------------------------------------------------
// docker_exec Validierung
// ------------------------------------------------------------

var dockerDeniedFlags = []string{
	"--privileged",
	"--cap-add",
	"--security-opt",
	"--pid=host",
	"--pid",
	"--network=host",
	"--ipc=host",
	"--uts=host",
	"--userns=host",
	"--device",
}

var dockerDeniedMountSources = []string{
	"/",
	"/etc",
	"/root",
	"/var/run/docker.sock",
	"/proc",
	"/sys",
}

func validateDockerArgs(args []string) error {
	for i, arg := range args {
		lower := strings.ToLower(arg)

		for _, denied := range dockerDeniedFlags {
			if lower == denied ||
				strings.HasPrefix(lower, denied+"=") {
				return fmt.Errorf(
					"Flag %q ist nicht erlaubt",
					arg,
				)
			}
		}

		if lower == "--pid" &&
			i+1 < len(args) &&
			strings.EqualFold(args[i+1], "host") {
			return fmt.Errorf(
				"--pid host ist nicht erlaubt",
			)
		}

		if lower == "-v" ||
			lower == "--volume" ||
			lower == "--mount" {

			if i+1 >= len(args) {
				continue
			}

			if err := validateMountSpec(args[i+1]); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateMountSpec(spec string) error {
	var source string

	// --mount source=...,target=...
	if strings.Contains(spec, "=") &&
		!strings.Contains(spec, ":") {

		for _, part := range strings.Split(spec, ",") {
			kv := strings.SplitN(part, "=", 2)

			if len(kv) == 2 &&
				strings.EqualFold(kv[0], "source") {
				source = kv[1]
			}
		}

	} else {
		// -v source:target
		parts := strings.Split(spec, ":")

		if len(parts) >= 1 {
			source = parts[0]
		}
	}

	source = filepath.Clean(source)

	for _, denied := range dockerDeniedMountSources {
		if source == denied {
			return fmt.Errorf(
				"Mount-Quelle %q ist nicht erlaubt",
				source,
			)
		}
	}

	return nil
}

// ------------------------------------------------------------
// Plugin-Verzeichnis auflösen
// ------------------------------------------------------------

func pluginDir() (string, error) {
	if p := os.Getenv("VBX_PLUGIN_PATH"); p != "" {
		abs, err := filepath.Abs(p)
		if err != nil {
			return "", fmt.Errorf(
				"VBX_PLUGIN_PATH konnte nicht aufgelöst werden: %w",
				err,
			)
		}

		return abs, nil
	}

	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf(
			"Pfad der vbx-Executable konnte nicht ermittelt werden: %w",
			err,
		)
	}

	return filepath.Join(
		filepath.Dir(exePath),
		"plugins",
	), nil
}

// ------------------------------------------------------------
// Host Bridge
// ------------------------------------------------------------

type hostDockerRequest struct {
	Args  []string `json:"args"`
	Stdin string   `json:"stdin,omitempty"`
}

type hostDockerResponse struct {
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
	Code   int    `json:"code"`
}

type hostWriteFileRequest struct {
	Path string `json:"path"`
	Data string `json:"data"`
}

type hostWriteFileResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// ------------------------------------------------------------
// Docker auf dem Host ausführen
// ------------------------------------------------------------

func dockerExecHost(
	req hostDockerRequest,
) hostDockerResponse {

	if err := validateDockerArgs(req.Args); err != nil {
		return hostDockerResponse{
			Stdout: "",
			Stderr: "docker_exec: " + err.Error(),
			Code:   -1,
		}
	}

	cmd := exec.Command(
		"docker",
		req.Args...,
	)

	if req.Stdin != "" {
		cmd.Stdin = strings.NewReader(req.Stdin)
	}

	var stdout strings.Builder
	var stderr strings.Builder

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	code := 0

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		} else {
			code = -1
		}
	}

	return hostDockerResponse{
		Stdout: strings.TrimSpace(stdout.String()),
		Stderr: strings.TrimSpace(stderr.String()),
		Code:   code,
	}
}

// ------------------------------------------------------------
// Host Bridge in WASM bereitstellen
// ------------------------------------------------------------

func instantiateHostBridge(
	ctx context.Context,
	rt wazero.Runtime,
) error {

	builder := rt.NewHostModuleBuilder(
		"vbx_host",
	)

	// ============================================================
	// docker_exec
	// ============================================================

	builder.NewFunctionBuilder().
		WithFunc(func(
			ctx context.Context,
			mod api.Module,
			reqPtr uint32,
			reqLen uint32,
		) uint64 {

			reqBytes, ok := mod.Memory().Read(
				reqPtr,
				reqLen,
			)

			if !ok {
				return hostBridgeError(
					ctx,
					mod,
					"docker_exec: Request konnte nicht aus WASM-Speicher gelesen werden",
				)
			}

			var req hostDockerRequest

			if err := json.Unmarshal(
				reqBytes,
				&req,
			); err != nil {
				return hostBridgeError(
					ctx,
					mod,
					"docker_exec: ungültiger Request: "+err.Error(),
				)
			}

			result := dockerExecHost(req)

			resultBytes, err := json.Marshal(result)
			if err != nil {
				return hostBridgeError(
					ctx,
					mod,
					"docker_exec: Host-Antwort konnte nicht kodiert werden: "+err.Error(),
				)
			}

			allocFn := mod.ExportedFunction("alloc")
			if allocFn == nil {
				return hostBridgeError(
					ctx,
					mod,
					"docker_exec: Plugin exportiert keine Funktion 'alloc'",
				)
			}

			res, err := allocFn.Call(
				ctx,
				uint64(len(resultBytes)),
			)

			if err != nil {
				return hostBridgeError(
					ctx,
					mod,
					"docker_exec: Plugin-alloc fehlgeschlagen: "+err.Error(),
				)
			}

			if len(res) == 0 {
				return hostBridgeError(
					ctx,
					mod,
					"docker_exec: Plugin-alloc liefert keinen Pointer",
				)
			}

			ptr := uint32(res[0])

			if !mod.Memory().Write(
				ptr,
				resultBytes,
			) {
				return hostBridgeError(
					ctx,
					mod,
					"docker_exec: Host-Antwort konnte nicht in WASM-Speicher geschrieben werden",
				)
			}

			return (uint64(ptr) << 32) |
				uint64(len(resultBytes))

		}).
		Export("docker_exec")

	// ============================================================
	// write_file
	// ============================================================
	//
	// Bleibt als Fallback bestehen für Plugins, die bewusst
	// nur über die Host-Bridge schreiben (statt direkt über
	// das gemountete Guest-FS). Arbeitet ausschließlich mit
	// Host-Pfaden relativ zu pluginBaseDir - siehe safeJoin.
	//

	builder.NewFunctionBuilder().
		WithFunc(func(
			ctx context.Context,
			mod api.Module,
			reqPtr uint32,
			reqLen uint32,
		) uint64 {

			reqBytes, ok := mod.Memory().Read(
				reqPtr,
				reqLen,
			)

			if !ok {
				return hostBridgeError(
					ctx,
					mod,
					"write_file: Request konnte nicht aus WASM-Speicher gelesen werden",
				)
			}

			var req hostWriteFileRequest

			if err := json.Unmarshal(
				reqBytes,
				&req,
			); err != nil {
				return hostBridgeError(
					ctx,
					mod,
					"write_file: ungültiger Request: "+err.Error(),
				)
			}

			if strings.TrimSpace(req.Path) == "" {
				return hostBridgeError(
					ctx,
					mod,
					"write_file: kein Dateipfad angegeben",
				)
			}

			base, err := resolvePluginBaseDir()
			if err != nil {
				return hostBridgeError(
					ctx,
					mod,
					"write_file: "+err.Error(),
				)
			}

			safePath, err := safeJoin(
				base,
				req.Path,
			)
			if err != nil {
				return hostBridgeError(
					ctx,
					mod,
					"write_file: "+err.Error(),
				)
			}

			if err := os.WriteFile(
				safePath,
				[]byte(req.Data),
				0644,
			); err != nil {
				return hostBridgeError(
					ctx,
					mod,
					"write_file: Datei konnte nicht geschrieben werden: "+err.Error(),
				)
			}

			result := hostWriteFileResponse{
				OK: true,
			}

			resultBytes, err := json.Marshal(result)
			if err != nil {
				return hostBridgeError(
					ctx,
					mod,
					"write_file: Antwort konnte nicht kodiert werden: "+err.Error(),
				)
			}

			allocFn := mod.ExportedFunction("alloc")
			if allocFn == nil {
				return hostBridgeError(
					ctx,
					mod,
					"write_file: Plugin exportiert keine Funktion 'alloc'",
				)
			}

			res, err := allocFn.Call(
				ctx,
				uint64(len(resultBytes)),
			)

			if err != nil {
				return hostBridgeError(
					ctx,
					mod,
					"write_file: Plugin-alloc fehlgeschlagen: "+err.Error(),
				)
			}

			if len(res) == 0 {
				return hostBridgeError(
					ctx,
					mod,
					"write_file: Plugin-alloc liefert keinen Pointer",
				)
			}

			ptr := uint32(res[0])

			if !mod.Memory().Write(
				ptr,
				resultBytes,
			) {
				return hostBridgeError(
					ctx,
					mod,
					"write_file: Antwort konnte nicht in WASM-Speicher geschrieben werden",
				)
			}

			return (uint64(ptr) << 32) |
				uint64(len(resultBytes))

		}).
		Export("write_file")

	// ============================================================
	// Host-Modul instanziieren
	// ============================================================

	_, err := builder.Instantiate(ctx)

	return err
}

// ------------------------------------------------------------
// Fehlerantwort des Host-Bridge-Aufrufs
// ------------------------------------------------------------

func hostBridgeError(
	ctx context.Context,
	mod api.Module,
	message string,
) uint64 {

	result := hostDockerResponse{
		Stdout: "",
		Stderr: message,
		Code:   -1,
	}

	data, err := json.Marshal(result)
	if err != nil {
		return 0
	}

	allocFn := mod.ExportedFunction("alloc")
	if allocFn == nil {
		return 0
	}

	res, err := allocFn.Call(
		ctx,
		uint64(len(data)),
	)

	if err != nil || len(res) == 0 {
		return 0
	}

	ptr := uint32(res[0])

	if !mod.Memory().Write(
		ptr,
		data,
	) {
		return 0
	}

	return (uint64(ptr) << 32) |
		uint64(len(data))
}

// ------------------------------------------------------------
// Plugin laden
// ------------------------------------------------------------

func LoadWasmPlugin(
	e *Environment,
	ns string,
) error {

	ctx := context.Background()

	pluginDebugf(
		"LoadWasmPlugin(%q)",
		ns,
	)

	if _, exists := loadedPlugins[ns]; exists {
		pluginDebugf(
			"%s bereits geladen",
			ns,
		)

		return nil
	}

	dir, err := pluginDir()
	if err != nil {
		pluginDebugf(
			"pluginDir FEHLER: %v",
			err,
		)

		return err
	}

	pluginDebugf(
		"Plugin-Verzeichnis: %s",
		dir,
	)

	wasmPath := filepath.Join(
		dir,
		ns+".wasm",
	)

	pluginDebugf(
		"WASM-Pfad: %s",
		wasmPath,
	)

	wasmBytes, err := os.ReadFile(wasmPath)
	if err != nil {
		pluginDebugf(
			"ReadFile FEHLER: %v",
			err,
		)

		return fmt.Errorf(
			"Plugin %q konnte nicht gelesen werden: %w",
			ns,
			err,
		)
	}

	pluginDebugf(
		"WASM geladen: %d Bytes",
		len(wasmBytes),
	)

	cache, err := getCompilationCache()
	if err != nil {
		pluginDebugf(
			"CompilationCache FEHLER: %v",
			err,
		)

		return err
	}

	pluginDebugf(
		"Runtime wird erstellt",
	)

	rt := wazero.NewRuntimeWithConfig(
		ctx,
		wazero.NewRuntimeConfig().
			WithCompilationCache(cache),
	)

	success := false

	defer func() {
		if !success {
			pluginDebugf(
				"Runtime wird wegen Fehler geschlossen",
			)

			_ = rt.Close(ctx)
		}
	}()

	// ------------------------------------------------------------
	// WASI
	// ------------------------------------------------------------

	pluginDebugf(
		"WASI wird geladen",
	)

	if _, err := wasi_snapshot_preview1.Instantiate(
		ctx,
		rt,
	); err != nil {

		pluginDebugf(
			"WASI FEHLER: %v",
			err,
		)

		return fmt.Errorf(
			"Plugin %q: WASI konnte nicht geladen werden: %w",
			ns,
			err,
		)
	}

	// ------------------------------------------------------------
	// Host Bridge
	// ------------------------------------------------------------

	pluginDebugf(
		"Host-Bridge wird geladen",
	)

	if err := instantiateHostBridge(
		ctx,
		rt,
	); err != nil {

		pluginDebugf(
			"Host-Bridge FEHLER: %v",
			err,
		)

		return err
	}

	// ------------------------------------------------------------
	// WASM kompilieren
	// ------------------------------------------------------------

	pluginDebugf(
		"WASM wird kompiliert",
	)

	compiled, err := rt.CompileModule(
		ctx,
		wasmBytes,
	)

	if err != nil {
		pluginDebugf(
			"Compile FEHLER: %v",
			err,
		)

		return fmt.Errorf(
			"Plugin %q konnte nicht kompiliert werden: %w",
			ns,
			err,
		)
	}

	pluginDebugf(
		"WASM kompiliert",
	)

	// ------------------------------------------------------------
	// Guest-Dateisystem konfigurieren
	// ------------------------------------------------------------

	pluginDebugf(
		"Guest-Dateisystem wird konfiguriert",
	)

	fsConfig, err := buildGuestFSConfig()
	if err != nil {
		pluginDebugf(
			"FS-Config FEHLER: %v",
			err,
		)

		return fmt.Errorf(
			"Plugin %q: Dateisystem-Konfiguration fehlgeschlagen: %w",
			ns,
			err,
		)
	}

	// ------------------------------------------------------------
	// WASI Reactor instanziieren
	//
	// Wichtig:
	// Normal-Go-WASM-Plugins (GOOS=wasip1) besitzen _initialize
	// statt automatischem _start-Durchlauf.
	//
	// _start darf hier NICHT automatisch ausgeführt werden,
	// da main() ansonsten das WASI-Modul beendet.
	// ------------------------------------------------------------

	pluginDebugf(
		"WASM wird instanziiert",
	)

	mod, err := rt.InstantiateModule(
		ctx,
		compiled,
		wazero.NewModuleConfig().
			WithStartFunctions().
			WithFSConfig(fsConfig),
	)

	if err != nil {
		pluginDebugf(
			"Instantiate FEHLER: %v",
			err,
		)

		return fmt.Errorf(
			"Plugin %q konnte nicht instanziiert werden: %w",
			ns,
			err,
		)
	}

	pluginDebugf(
		"WASM instanziiert",
	)

	// ------------------------------------------------------------
	// Go WASI Reactor initialisieren
	// ------------------------------------------------------------

	initializeFn := mod.ExportedFunction(
		"_initialize",
	)

	if initializeFn == nil {
		return fmt.Errorf(
			"Plugin %q: _initialize Export fehlt",
			ns,
		)
	}

	pluginDebugf(
		"_initialize wird aufgerufen",
	)

	if _, err := initializeFn.Call(ctx); err != nil {
		pluginDebugf(
			"_initialize FEHLER: %v",
			err,
		)

		return fmt.Errorf(
			"Plugin %q: _initialize fehlgeschlagen: %w",
			ns,
			err,
		)
	}

	pluginDebugf(
		"_initialize erfolgreich",
	)

	// ------------------------------------------------------------
	// ABI-Exports prüfen
	// ------------------------------------------------------------

	abiVerFn := mod.ExportedFunction(
		"vbx_abi_version",
	)

	allocFn := mod.ExportedFunction(
		"alloc",
	)

	deallocFn := mod.ExportedFunction(
		"dealloc",
	)

	callFn := mod.ExportedFunction(
		"vbx_call",
	)

	describeFn := mod.ExportedFunction(
		"vbx_describe",
	)

	pluginDebugf(
		"Exports: abi=%v alloc=%v dealloc=%v call=%v describe=%v",
		abiVerFn != nil,
		allocFn != nil,
		deallocFn != nil,
		callFn != nil,
		describeFn != nil,
	)

	if abiVerFn == nil ||
		allocFn == nil ||
		deallocFn == nil ||
		callFn == nil ||
		describeFn == nil {

		return fmt.Errorf(
			"Plugin %q: erforderliche Exports fehlen",
			ns,
		)
	}

	// ------------------------------------------------------------
	// ABI-Version prüfen
	// ------------------------------------------------------------

	pluginDebugf(
		"ABI wird geprüft",
	)

	res, err := abiVerFn.Call(ctx)

	if err != nil {
		pluginDebugf(
			"ABI Call FEHLER: %v",
			err,
		)

		return fmt.Errorf(
			"Plugin %q: vbx_abi_version fehlgeschlagen: %w",
			ns,
			err,
		)
	}

	if len(res) == 0 {
		return fmt.Errorf(
			"Plugin %q: vbx_abi_version liefert keinen Rückgabewert",
			ns,
		)
	}

	abiVersion := uint32(res[0])

	pluginDebugf(
		"ABI-Version: %d",
		abiVersion,
	)

	if abiVersion != expectedABIVersion {
		return fmt.Errorf(
			"Plugin %q: inkompatible ABI-Version %d, erwartet %d",
			ns,
			abiVersion,
			expectedABIVersion,
		)
	}

	// ------------------------------------------------------------
	// Plugin-Beschreibung lesen
	// ------------------------------------------------------------

	pluginDebugf(
		"vbx_describe wird aufgerufen",
	)

	res, err = describeFn.Call(ctx)

	if err != nil {
		pluginDebugf(
			"vbx_describe FEHLER: %v",
			err,
		)

		return fmt.Errorf(
			"Plugin %q: vbx_describe fehlgeschlagen: %w",
			ns,
			err,
		)
	}

	if len(res) == 0 {
		return fmt.Errorf(
			"Plugin %q: vbx_describe liefert keinen Rückgabewert",
			ns,
		)
	}

	descPtr, descLen := unpackPtrLen(
		res[0],
	)

	pluginDebugf(
		"%s: ptr=%d len=%d raw=0x%016x",
		ns,
		descPtr,
		descLen,
		res[0],
	)

	if descLen == 0 {
		return fmt.Errorf(
			"Plugin %q: vbx_describe liefert eine leere Beschreibung",
			ns,
		)
	}

	descBytes, ok := mod.Memory().Read(
		descPtr,
		descLen,
	)

	if !ok {
		return fmt.Errorf(
			"Plugin %q: Beschreibung nicht lesbar (ptr=%d, len=%d)",
			ns,
			descPtr,
			descLen,
		)
	}

	pluginDebugf(
		"description=%q",
		string(descBytes),
	)

	var entries []pluginFuncDesc

	if err := json.Unmarshal(
		descBytes,
		&entries,
	); err != nil {

		return fmt.Errorf(
			"Plugin %q: ungültige Beschreibung: %w",
			ns,
			err,
		)
	}

	pluginDebugf(
		"%d Funktionen beschrieben",
		len(entries),
	)

	if len(entries) == 0 {
		return fmt.Errorf(
			"Plugin %q: keine Funktionen beschrieben",
			ns,
		)
	}

	// ------------------------------------------------------------
	// Funktionen registrieren
	// ------------------------------------------------------------

	for _, entry := range entries {
		entry := entry

		pluginDebugf(
			"Registrierung: %s.%s",
			entry.Namespace,
			entry.Name,
		)

		wrapperFn := func(args []Value) Value {
			return callPluginFunction(
				ctx,
				ns,
				mod,
				allocFn,
				deallocFn,
				callFn,
				entry.Name,
				args,
			)
		}

		fullName := entry.Name

		if entry.Namespace != "" {
			fullName =
				entry.Namespace +
					"." +
					entry.Name
		}

		Register(
			fullName,
			entry.Namespace,
			entry.Params,
			entry.Description,
			wrapperFn,
		)
	}

	// ------------------------------------------------------------
	// Plugin als geladen markieren
	// ------------------------------------------------------------

	loadedPlugins[ns] = &pluginHandle{
		runtime: rt,
		module:  mod,
	}

	success = true

	pluginDebugf(
		"Plugin %q erfolgreich geladen",
		ns,
	)

	return nil
}

// ------------------------------------------------------------
// Plugin-Funktion aufrufen
// ------------------------------------------------------------

func callPluginFunction(
	ctx context.Context,
	ns string,
	mod api.Module,
	allocFn api.Function,
	deallocFn api.Function,
	callFn api.Function,
	fnName string,
	args []Value,
) Value {

	argsJSON, err := json.Marshal(
		toJSONValues(args),
	)

	if err != nil {
		return ErrorVal(
			fmt.Sprintf(
				"Plugin %q, Funktion %q: Argumente konnten nicht kodiert werden: %v",
				ns,
				fnName,
				err,
			),
		)
	}

	// ------------------------------------------------------------
	// Argumente in Plugin-Speicher schreiben
	// ------------------------------------------------------------

	argsPtr, err := writeToPluginMemory(
		ctx,
		mod,
		allocFn,
		argsJSON,
	)

	if err != nil {
		return ErrorVal(
			fmt.Sprintf(
				"Plugin %q, Funktion %q: %v",
				ns,
				fnName,
				err,
			),
		)
	}

	defer freePluginMemory(
		ctx,
		deallocFn,
		argsPtr,
		uint32(len(argsJSON)),
	)

	// ------------------------------------------------------------
	// Funktionsnamen in Plugin-Speicher schreiben
	// ------------------------------------------------------------

	namePtr, err := writeToPluginMemory(
		ctx,
		mod,
		allocFn,
		[]byte(fnName),
	)

	if err != nil {
		return ErrorVal(
			fmt.Sprintf(
				"Plugin %q, Funktion %q: %v",
				ns,
				fnName,
				err,
			),
		)
	}

	defer freePluginMemory(
		ctx,
		deallocFn,
		namePtr,
		uint32(len(fnName)),
	)

	// ------------------------------------------------------------
	// Plugin-Funktion aufrufen
	// ------------------------------------------------------------

	res, err := callFn.Call(
		ctx,
		uint64(namePtr),
		uint64(len(fnName)),
		uint64(argsPtr),
		uint64(len(argsJSON)),
	)

	if err != nil {
		return ErrorVal(
			fmt.Sprintf(
				"Plugin %q, Funktion %q: %v",
				ns,
				fnName,
				err,
			),
		)
	}

	if len(res) == 0 {
		return ErrorVal(
			fmt.Sprintf(
				"Plugin %q, Funktion %q: kein Rückgabewert",
				ns,
				fnName,
			),
		)
	}

	// ------------------------------------------------------------
	// Rückgabe-Pointer und Länge auspacken
	// ------------------------------------------------------------

	resPtr, resLen := unpackPtrLen(
		res[0],
	)

	resultJSON, ok := mod.Memory().Read(
		resPtr,
		resLen,
	)

	if !ok {
		return ErrorVal(
			fmt.Sprintf(
				"Plugin %q, Funktion %q: Rückgabewert nicht lesbar (ptr=%d, len=%d)",
				ns,
				fnName,
				resPtr,
				resLen,
			),
		)
	}

	// ------------------------------------------------------------
	// Rückgabespeicher des Plugins freigeben
	// ------------------------------------------------------------

	defer freePluginMemory(
		ctx,
		deallocFn,
		resPtr,
		resLen,
	)

	return fromJSONValue(
		resultJSON,
	)
}

// ------------------------------------------------------------
// Compilation Cache
// ------------------------------------------------------------

var wasmCache wazero.CompilationCache

func getCompilationCache() (
	wazero.CompilationCache,
	error,
) {

	if wasmCache != nil {
		return wasmCache, nil
	}

	base, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf(
			"Cache-Verzeichnis konnte nicht ermittelt werden: %w",
			err,
		)
	}

	cacheDir := filepath.Join(
		base,
		"vbx",
		"plugin-cache",
	)

	if err := os.MkdirAll(
		cacheDir,
		0755,
	); err != nil {

		return nil, fmt.Errorf(
			"Cache-Verzeichnis %q konnte nicht angelegt werden: %w",
			cacheDir,
			err,
		)
	}

	cache, err := wazero.NewCompilationCacheWithDir(
		cacheDir,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"Plugin-Compilation-Cache konnte nicht angelegt werden: %w",
			err,
		)
	}

	wasmCache = cache

	return cache, nil
}

// ------------------------------------------------------------
// Speicherhilfsfunktionen
// ------------------------------------------------------------

func unpackPtrLen(
	packed uint64,
) (ptr, length uint32) {

	return uint32(packed >> 32),
		uint32(packed)
}

func writeToPluginMemory(
	ctx context.Context,
	mod api.Module,
	allocFn api.Function,
	data []byte,
) (uint32, error) {

	res, err := allocFn.Call(
		ctx,
		uint64(len(data)),
	)

	if err != nil {
		return 0, fmt.Errorf(
			"alloc fehlgeschlagen: %w",
			err,
		)
	}

	if len(res) == 0 {
		return 0, fmt.Errorf(
			"alloc liefert keinen Rückgabewert",
		)
	}

	ptr := uint32(res[0])

	if !mod.Memory().Write(
		ptr,
		data,
	) {
		return 0, fmt.Errorf(
			"Schreiben in Plugin-Speicher fehlgeschlagen (ptr=%d, len=%d)",
			ptr,
			len(data),
		)
	}

	return ptr, nil
}

func freePluginMemory(
	ctx context.Context,
	deallocFn api.Function,
	ptr,
	size uint32,
) {

	_, _ = deallocFn.Call(
		ctx,
		uint64(ptr),
		uint64(size),
	)
}

// ------------------------------------------------------------
// Value <-> JSON
// ------------------------------------------------------------

type jsonValue struct {
	Type    string               `json:"type"`
	Num     float64              `json:"num,omitempty"`
	Str     string               `json:"str,omitempty"`
	Bool    bool                 `json:"bool,omitempty"`
	Arr     []jsonValue          `json:"arr,omitempty"`
	Arr2D   [][]jsonValue        `json:"arr2d,omitempty"`
	Map     map[string]jsonValue `json:"map,omitempty"`
	Bytes   []byte               `json:"bytes,omitempty"`
	Message string               `json:"message,omitempty"`
}

func toJSONValue(
	v Value,
) jsonValue {

	switch v.Kind {

	case KindNum:

		return jsonValue{
			Type: "num",
			Num:  v.Num,
		}

	case KindStr:

		s := v.Str

		if looksLikeHostPath(s) {
			if translated, err := hostToGuestPath(s); err == nil {
				s = translated
			}
			// Bei Übersetzungsfehler: Original-String
			// durchreichen, das Plugin liefert dann einen
			// erklärenden Fehler statt eines stillen
			// Fehlschlags.
		}

		return jsonValue{
			Type: "str",
			Str:  s,
		}

	case KindBool:

		return jsonValue{
			Type: "bool",
			Bool: v.Bool,
		}

	case KindArr:

		arr := make(
			[]jsonValue,
			len(v.Arr),
		)

		for i, el := range v.Arr {
			arr[i] = toJSONValue(el)
		}

		return jsonValue{
			Type: "arr",
			Arr:  arr,
		}

	case KindArr2D:

		arr2d := make(
			[][]jsonValue,
			len(v.Arr2D),
		)

		for i, row := range v.Arr2D {

			r := make(
				[]jsonValue,
				len(row),
			)

			for j, el := range row {
				r[j] = toJSONValue(el)
			}

			arr2d[i] = r
		}

		return jsonValue{
			Type:  "arr2d",
			Arr2D: arr2d,
		}

	case KindMap:

		m := make(
			map[string]jsonValue,
			len(v.Map),
		)

		for k, el := range v.Map {
			m[k] = toJSONValue(el)
		}

		return jsonValue{
			Type: "map",
			Map:  m,
		}

	case KindBytes:

		return jsonValue{
			Type:  "bytes",
			Bytes: v.Bytes,
		}

	case KindError:

		return jsonValue{
			Type:    "error",
			Message: v.Str,
		}

	case KindObj:

		return jsonValue{
			Type:    "error",
			Message: "Objects werden von Plugin-Funktionen noch nicht unterstützt",
		}

	case KindNull:

		return jsonValue{
			Type: "null",
		}

	default:

		return jsonValue{
			Type: "empty",
		}
	}
}

func toJSONValues(
	vals []Value,
) []jsonValue {

	out := make(
		[]jsonValue,
		len(vals),
	)

	for i, v := range vals {
		out[i] = toJSONValue(v)
	}

	return out
}

func fromJSONValue(
	data []byte,
) Value {

	var jv jsonValue

	if err := json.Unmarshal(
		data,
		&jv,
	); err != nil {

		return ErrorVal(
			"Ungültige Plugin-Antwort: " +
				err.Error(),
		)
	}

	return jsonValueToValue(
		jv,
	)
}

func jsonValueToValue(
	jv jsonValue,
) Value {

	switch jv.Type {

	case "num":

		return NumVal(
			jv.Num,
		)

	case "str":

		return StrVal(
			jv.Str,
		)

	case "bool":

		return BoolVal(
			jv.Bool,
		)

	case "arr":

		arr := make(
			[]Value,
			len(jv.Arr),
		)

		for i, el := range jv.Arr {
			arr[i] = jsonValueToValue(el)
		}

		return Value{
			Kind: KindArr,
			Arr:  arr,
		}

	case "arr2d":

		arr2d := make(
			[][]Value,
			len(jv.Arr2D),
		)

		for i, row := range jv.Arr2D {

			r := make(
				[]Value,
				len(row),
			)

			for j, el := range row {
				r[j] = jsonValueToValue(el)
			}

			arr2d[i] = r
		}

		return Value{
			Kind:  KindArr2D,
			Arr2D: arr2d,
		}

	case "map":

		m := make(
			map[string]Value,
			len(jv.Map),
		)

		for k, el := range jv.Map {
			m[k] = jsonValueToValue(el)
		}

		return Value{
			Kind: KindMap,
			Map:  m,
		}

	case "bytes":

		return Value{
			Kind:  KindBytes,
			Bytes: jv.Bytes,
		}

	case "empty":

		return NullVal()

	case "error":

		return ErrorVal(
			jv.Message,
		)

	case "null":

		return NullVal()

	default:

		return NilValue
	}
}
