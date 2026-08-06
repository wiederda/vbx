// ------------------------
// stdlib_docker.go
// ------------------------

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// getString extrahiert sicher einen String aus einer map[string]interface{}
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// dockerExec führt einen docker-Befehl aus und gibt (stdout, stderr, exitCode) zurück
func dockerExec(args ...string) (string, string, int) {
	cmd := exec.Command("docker", args...)
	var stdout, stderr strings.Builder
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
	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), code
}

// dockerOk gibt "OK" oder "error: ..." zurück
func dockerOk(stderr string, code int) Value {
	if code != 0 {
		return Value{Kind: KindStr, Str: "error: " + stderr}
	}
	return Value{Kind: KindStr, Str: "OK"}
}

func dockerImageExists(image string) bool {
	_, _, code := dockerExec(
		"image",
		"inspect",
		image,
	)

	return code == 0
}

func InitDockerFunctions() {
	if builtins == nil {
		builtins = make(map[string]BuiltinInfo)
	}

	ns := "docker."

	// =========================================================
	// CONTAINER
	// =========================================================

	Register(ns+"List", "docker", "[all]", "Gibt ein Array aller Container zurück. all=true zeigt auch gestoppte.", func(args []Value) Value {
		all := false
		if len(args) >= 1 {
			all = (args[0].Str == "true" || (args[0].Kind == KindBool && args[0].Bool))
		}

		cmdArgs := []string{"ps", "--format", "{{.ID}}|{{.Names}}|{{.Image}}|{{.Status}}|{{.Ports}}"}
		if all {
			cmdArgs = append(cmdArgs, "-a")
		}

		stdout, stderr, code := dockerExec(cmdArgs...)
		if code != 0 {
			return Value{Kind: KindStr, Str: "error: " + stderr}
		}

		var result []Value
		for _, line := range strings.Split(stdout, "\n") {
			if line == "" {
				continue
			}
			result = append(result, Value{Kind: KindStr, Str: line})
		}
		return Value{Kind: KindArr, Arr: result}
	})

	Register(ns+"Run", "docker", "image, [name], [options], [command]",
		"Startet einen neuen Container aus einem Image.", func(args []Value) Value {

			if len(args) < 1 {
				return Value{Kind: KindStr, Str: "error: Image-Name benötigt"}
			}

			// Prüfen, ob Image vorhanden ist
			if !dockerImageExists(args[0].Str) {

				stdout, stderr, code := dockerExec(
					"pull",
					args[0].Str,
				)

				if code != 0 {
					return Value{
						Kind: KindStr,
						Str:  "error: " + stderr,
					}
				}

				_ = stdout
			}

			cmdArgs := []string{"run", "-d"}

			argIndex := 1

			// Optionaler Containername
			if len(args) > argIndex && args[argIndex].Kind == KindStr && args[argIndex].Str != "" {
				cmdArgs = append(cmdArgs, "--name", args[argIndex].Str)
				argIndex++
			}

			// Docker Optionen als Array
			if len(args) > argIndex && args[argIndex].Kind == KindArr {
				for _, v := range args[argIndex].Arr {
					cmdArgs = append(cmdArgs, ToString(v))
				}
				argIndex++
			}

			// Image
			cmdArgs = append(cmdArgs, args[0].Str)

			// Command und weitere Parameter
			for ; argIndex < len(args); argIndex++ {
				cmdArgs = append(cmdArgs, ToString(args[argIndex]))
			}

			stdout, stderr, code := dockerExec(cmdArgs...)
			if code != 0 {
				return Value{Kind: KindStr, Str: "error: " + stderr}
			}

			return Value{Kind: KindStr, Str: stdout}
		})

	Register(ns+"Login", "docker", "registry, username, token",
		"Authentifiziert an einer Docker Registry.", func(args []Value) Value {

			if len(args) < 3 {
				return Value{Kind: KindStr, Str: "error: registry, username und token benötigt"}
			}

			cmd := exec.Command(
				"docker",
				"login",
				args[0].Str,
				"-u",
				args[1].Str,
				"--password-stdin",
			)

			stdin, err := cmd.StdinPipe()
			if err != nil {
				return Value{Kind: KindStr, Str: "error: " + err.Error()}
			}

			if err := cmd.Start(); err != nil {
				return Value{Kind: KindStr, Str: "error: " + err.Error()}
			}

			_, _ = stdin.Write([]byte(args[2].Str))
			stdin.Close()

			output, err := cmd.CombinedOutput()

			if err != nil {
				return Value{Kind: KindStr, Str: "error: " + string(output)}
			}

			return Value{Kind: KindStr, Str: "ok"}
		})

	Register(ns+"Logout", "docker", "registry",
		"Abmelden von einer Docker Registry.", func(args []Value) Value {

			if len(args) < 1 {
				return Value{Kind: KindStr, Str: "error: Registry benötigt"}
			}

			stdout, stderr, code := dockerExec(
				"logout",
				args[0].Str,
			)

			if code != 0 {
				return Value{Kind: KindStr, Str: "error: " + stderr}
			}

			return Value{Kind: KindStr, Str: stdout}
		})

	Register(ns+"Start", "docker", "name", "Startet einen Container.", func(args []Value) Value {
		if len(args) < 1 {
			return Value{Kind: KindStr, Str: "error: Container-Name benötigt"}
		}
		_, stderr, code := dockerExec("start", args[0].Str)
		return dockerOk(stderr, code)
	})

	Register(ns+"Stop", "docker", "name, [timeout]", "Stoppt einen Container. Timeout in Sekunden (Default: 10).", func(args []Value) Value {
		if len(args) < 1 {
			return Value{Kind: KindStr, Str: "error: Container-Name benötigt"}
		}
		timeout := "10"
		if len(args) >= 2 && args[1].Str != "" {
			timeout = args[1].Str
		}
		_, stderr, code := dockerExec("stop", "-t", timeout, args[0].Str)
		return dockerOk(stderr, code)
	})

	Register(ns+"Restart", "docker", "name, [timeout]", "Startet einen Container neu.", func(args []Value) Value {
		if len(args) < 1 {
			return Value{Kind: KindStr, Str: "error: Container-Name benötigt"}
		}
		timeout := "10"
		if len(args) >= 2 && args[1].Str != "" {
			timeout = args[1].Str
		}
		_, stderr, code := dockerExec("restart", "-t", timeout, args[0].Str)
		return dockerOk(stderr, code)
	})

	Register(ns+"Remove", "docker", "name, [force]", "Entfernt einen Container. force=true erzwingt das Löschen.", func(args []Value) Value {
		if len(args) < 1 {
			return Value{Kind: KindStr, Str: "error: Container-Name benötigt"}
		}
		force := false
		if len(args) >= 2 {
			force = (args[1].Str == "true" || (args[1].Kind == KindBool && args[1].Bool))
		}
		cmdArgs := []string{"rm"}
		if force {
			cmdArgs = append(cmdArgs, "-f")
		}
		cmdArgs = append(cmdArgs, args[0].Str)
		_, stderr, code := dockerExec(cmdArgs...)
		return dockerOk(stderr, code)
	})

	Register(ns+"Status", "docker", "name", "Gibt den Status eines Containers zurück (running, exited, etc.).", func(args []Value) Value {
		if len(args) < 1 {
			return Value{Kind: KindStr, Str: "error: Container-Name benötigt"}
		}
		stdout, stderr, code := dockerExec("inspect", "--format", "{{.State.Status}}", args[0].Str)
		if code != 0 {
			return Value{Kind: KindStr, Str: "error: " + stderr}
		}
		return Value{Kind: KindStr, Str: stdout}
	})

	Register(ns+"IsRunning", "docker", "name", "Gibt True zurück wenn der Container läuft.", func(args []Value) Value {
		if len(args) < 1 {
			return BoolVal(false)
		}
		stdout, _, code := dockerExec("inspect", "--format", "{{.State.Running}}", args[0].Str)
		if code != 0 {
			return BoolVal(false)
		}
		return BoolVal(strings.TrimSpace(stdout) == "true")
	})

	Register(ns+"Inspect", "docker", "name", "Gibt die vollständigen Metadaten eines Containers als String zurück.", func(args []Value) Value {
		if len(args) < 1 {
			return Value{Kind: KindStr, Str: "error: Container-Name benötigt"}
		}
		stdout, stderr, code := dockerExec("inspect", args[0].Str)
		if code != 0 {
			return Value{Kind: KindStr, Str: "error: " + stderr}
		}
		return Value{Kind: KindStr, Str: stdout}
	})

	Register(ns+"ExportCompose", "docker", "name, [outputPath]", "Exportiert einen laufenden Container als docker-compose.yml. Gibt YAML-String zurück oder schreibt in Datei.", func(args []Value) Value {
		if len(args) < 1 {
			return Value{Kind: KindStr, Str: "error: Container-Name benötigt"}
		}
		name := args[0].Str
		outputPath := ""
		if len(args) >= 2 {
			outputPath = args[1].Str
		}

		// docker inspect aufrufen
		stdout, stderr, code := dockerExec("inspect", name)
		if code != 0 {
			return Value{Kind: KindStr, Str: "error: " + stderr}
		}

		// JSON parsen
		var containers []map[string]interface{}
		if err := json.Unmarshal([]byte(stdout), &containers); err != nil {
			return Value{Kind: KindStr, Str: "error: JSON parsen fehlgeschlagen: " + err.Error()}
		}
		if len(containers) == 0 {
			return Value{Kind: KindStr, Str: "error: Keine Container-Daten gefunden"}
		}
		c := containers[0]

		// Name
		containerName := strings.TrimPrefix(getString(c, "Name"), "/")

		// Image
		config, _ := c["Config"].(map[string]interface{})
		image := ""
		if config != nil {
			image = getString(config, "Image")
		}

		// Environment
		var envLines []string
		if config != nil {
			if envRaw, ok := config["Env"].([]interface{}); ok {
				for _, e := range envRaw {
					if s, ok := e.(string); ok {
						envLines = append(envLines, "      - "+s)
					}
				}
			}
		}

		// Labels
		var labelLines []string
		if config != nil {
			if labelsRaw, ok := config["Labels"].(map[string]interface{}); ok {
				for k, v := range labelsRaw {
					labelLines = append(labelLines, fmt.Sprintf("      - %s=%v", k, v))
				}
			}
		}

		// Ports
		var portLines []string
		hostConfig, _ := c["HostConfig"].(map[string]interface{})
		if hostConfig != nil {
			if portBindings, ok := hostConfig["PortBindings"].(map[string]interface{}); ok {
				for containerPort, bindingsRaw := range portBindings {
					if bindings, ok := bindingsRaw.([]interface{}); ok {
						for _, bRaw := range bindings {
							if b, ok := bRaw.(map[string]interface{}); ok {
								hostPort := getString(b, "HostPort")
								portLines = append(portLines, fmt.Sprintf("      - \"%s:%s\"", hostPort, containerPort))
							}
						}
					}
				}
			}
		}

		// Volumes
		var volumeLines []string
		if mounts, ok := c["Mounts"].([]interface{}); ok {
			for _, mRaw := range mounts {
				if m, ok := mRaw.(map[string]interface{}); ok {
					src := getString(m, "Source")
					dst := getString(m, "Destination")
					if src != "" && dst != "" {
						volumeLines = append(volumeLines, fmt.Sprintf("      - %s:%s", src, dst))
					}
				}
			}
		}

		// Restart Policy
		restartPolicy := "no"
		if hostConfig != nil {
			if rp, ok := hostConfig["RestartPolicy"].(map[string]interface{}); ok {
				restartPolicy = getString(rp, "Name")
				if restartPolicy == "" {
					restartPolicy = "no"
				}
			}
		}

		// YAML zusammenbauen
		var sb strings.Builder
		sb.WriteString("version: '3'\n")
		sb.WriteString("services:\n")
		sb.WriteString(fmt.Sprintf("  %s:\n", containerName))
		sb.WriteString(fmt.Sprintf("    image: %s\n", image))
		sb.WriteString(fmt.Sprintf("    container_name: %s\n", containerName))
		sb.WriteString(fmt.Sprintf("    restart: %s\n", restartPolicy))

		if len(portLines) > 0 {
			sb.WriteString("    ports:\n")
			for _, p := range portLines {
				sb.WriteString(p)
				sb.WriteString("\n")
			}
		}
		if len(volumeLines) > 0 {
			sb.WriteString("    volumes:\n")
			for _, v := range volumeLines {
				sb.WriteString(v)
				sb.WriteString("\n")
			}
		}
		if len(envLines) > 0 {
			sb.WriteString("    environment:\n")
			for _, e := range envLines {
				sb.WriteString(e)
				sb.WriteString("\n")
			}
		}
		if len(labelLines) > 0 {
			sb.WriteString("    labels:\n")
			for _, l := range labelLines {
				sb.WriteString(l)
				sb.WriteString("\n")
			}
		}

		yaml := sb.String()

		// In Datei schreiben falls Pfad angegeben
		if outputPath != "" {
			if err := os.WriteFile(outputPath, []byte(yaml), 0644); err != nil {
				return Value{Kind: KindStr, Str: "error: Datei konnte nicht geschrieben werden: " + err.Error()}
			}
			return Value{Kind: KindStr, Str: "OK"}
		}

		return Value{Kind: KindStr, Str: yaml}
	})

	Register(ns+"Logs", "docker", "name, [lines]", "Gibt die letzten Logzeilen eines Containers zurück. Default: 50 Zeilen.", func(args []Value) Value {
		if len(args) < 1 {
			return Value{Kind: KindStr, Str: "error: Container-Name benötigt"}
		}
		lines := "50"
		if len(args) >= 2 && args[1].Str != "" {
			lines = args[1].Str
		}
		stdout, stderr, code := dockerExec("logs", "--tail", lines, args[0].Str)
		if code != 0 {
			return Value{Kind: KindStr, Str: "error: " + stderr}
		}
		return Value{Kind: KindStr, Str: stdout}
	})

	Register(ns+"Exec", "docker", "name, cmd, [args...]", "Führt einen Befehl in einem laufenden Container aus.", func(args []Value) Value {
		if len(args) < 2 {
			return Value{Kind: KindStr, Str: "error: Container-Name und Befehl benötigt"}
		}
		cmdArgs := []string{"exec", args[0].Str, args[1].Str}
		for i := 2; i < len(args); i++ {
			cmdArgs = append(cmdArgs, ToString(args[i]))
		}
		stdout, stderr, code := dockerExec(cmdArgs...)
		if code != 0 {
			return Value{Kind: KindStr, Str: "error: " + stderr}
		}
		return Value{Kind: KindStr, Str: stdout}
	})

	Register(ns+"Stats", "docker", "[name]", "Gibt CPU/RAM-Nutzung zurück. Ohne Name alle Container.", func(args []Value) Value {
		cmdArgs := []string{"stats", "--no-stream", "--format", "{{.Name}}|{{.CPUPerc}}|{{.MemUsage}}|{{.MemPerc}}"}
		if len(args) >= 1 && args[0].Str != "" {
			cmdArgs = append(cmdArgs, args[0].Str)
		}
		stdout, stderr, code := dockerExec(cmdArgs...)
		if code != 0 {
			return Value{Kind: KindStr, Str: "error: " + stderr}
		}
		var result []Value
		for _, line := range strings.Split(stdout, "\n") {
			if line == "" {
				continue
			}
			result = append(result, Value{Kind: KindStr, Str: line})
		}
		return Value{Kind: KindArr, Arr: result}
	})

	// =========================================================
	// IMAGES
	// =========================================================

	Register(ns+"ImageList", "docker", "-", "Gibt ein Array aller lokalen Images zurück.", func(args []Value) Value {
		stdout, stderr, code := dockerExec("images", "--format", "{{.Repository}}|{{.Tag}}|{{.ID}}|{{.Size}}")
		if code != 0 {
			return Value{Kind: KindStr, Str: "error: " + stderr}
		}
		var result []Value
		for _, line := range strings.Split(stdout, "\n") {
			if line == "" {
				continue
			}
			result = append(result, Value{Kind: KindStr, Str: line})
		}
		return Value{Kind: KindArr, Arr: result}
	})

	Register(ns+"ImagePull", "docker", "image", "Lädt ein Image von Docker Hub oder einer Registry.", func(args []Value) Value {
		if len(args) < 1 {
			return Value{Kind: KindStr, Str: "error: Image-Name benötigt"}
		}
		_, stderr, code := dockerExec("pull", args[0].Str)
		return dockerOk(stderr, code)
	})

	Register(ns+"ImageRemove", "docker", "image, [force]", "Löscht ein lokales Image.", func(args []Value) Value {
		if len(args) < 1 {
			return Value{Kind: KindStr, Str: "error: Image-Name benötigt"}
		}
		force := false
		if len(args) >= 2 {
			force = (args[1].Str == "true" || (args[1].Kind == KindBool && args[1].Bool))
		}
		cmdArgs := []string{"rmi"}
		if force {
			cmdArgs = append(cmdArgs, "-f")
		}
		cmdArgs = append(cmdArgs, args[0].Str)
		_, stderr, code := dockerExec(cmdArgs...)
		return dockerOk(stderr, code)
	})

	Register(ns+"ImageBuild", "docker", "path, tag, [dockerfile]", "Baut ein Image aus einem Dockerfile. dockerfile optional (Default: Dockerfile).", func(args []Value) Value {
		if len(args) < 2 {
			return Value{Kind: KindStr, Str: "error: path und tag benötigt"}
		}
		path := args[0].Str
		tag := args[1].Str
		dockerfile := "Dockerfile"
		if len(args) >= 3 && args[2].Str != "" {
			dockerfile = args[2].Str
		}
		_, stderr, code := dockerExec("build", "-t", tag, "-f", dockerfile, path)
		return dockerOk(stderr, code)
	})

	Register(ns+"ImagePrune", "docker", "-", "Löscht alle ungenutzten (dangling) Images.", func(args []Value) Value {
		_, stderr, code := dockerExec("image", "prune", "-f")
		return dockerOk(stderr, code)
	})

	Register(ns+"ImageRemoveAll", "docker", "-", "Löscht alle ungenutzten Images (auch nicht-dangling). Container müssen gestoppt sein.", func(args []Value) Value {
		_, stderr, code := dockerExec("image", "prune", "-a", "-f")
		return dockerOk(stderr, code)
	})

	// =========================================================
	// COMPOSE
	// =========================================================

	Register(ns+"ComposeUp", "docker", "path, [detach]", "Startet Dienste via docker compose. detach=true (Default) für Hintergrund.", func(args []Value) Value {
		if len(args) < 1 {
			return Value{Kind: KindStr, Str: "error: Pfad zur compose-Datei benötigt"}
		}
		detach := true
		if len(args) >= 2 {
			detach = !(args[1].Str == "false" || (args[1].Kind == KindBool && !args[1].Bool))
		}
		cmdArgs := []string{"compose", "-f", args[0].Str, "up"}
		if detach {
			cmdArgs = append(cmdArgs, "-d")
		}
		_, stderr, code := dockerExec(cmdArgs...)
		return dockerOk(stderr, code)
	})

	Register(ns+"ComposeDown", "docker", "path, [removeVolumes]", "Stoppt und entfernt Compose-Dienste. removeVolumes=true löscht auch Volumes.", func(args []Value) Value {
		if len(args) < 1 {
			return Value{Kind: KindStr, Str: "error: Pfad zur compose-Datei benötigt"}
		}
		removeVolumes := false
		if len(args) >= 2 {
			removeVolumes = (args[1].Str == "true" || (args[1].Kind == KindBool && args[1].Bool))
		}
		cmdArgs := []string{"compose", "-f", args[0].Str, "down"}
		if removeVolumes {
			cmdArgs = append(cmdArgs, "-v")
		}
		_, stderr, code := dockerExec(cmdArgs...)
		return dockerOk(stderr, code)
	})

	Register(ns+"ComposePull", "docker", "path", "Aktualisiert alle Images einer Compose-Datei.", func(args []Value) Value {
		if len(args) < 1 {
			return Value{Kind: KindStr, Str: "error: Pfad zur compose-Datei benötigt"}
		}
		_, stderr, code := dockerExec("compose", "-f", args[0].Str, "pull")
		return dockerOk(stderr, code)
	})

	Register(ns+"ComposeRestart", "docker", "path", "Startet alle Dienste einer Compose-Datei neu.", func(args []Value) Value {
		if len(args) < 1 {
			return Value{Kind: KindStr, Str: "error: Pfad zur compose-Datei benötigt"}
		}
		_, stderr, code := dockerExec("compose", "-f", args[0].Str, "restart")
		return dockerOk(stderr, code)
	})

	Register(ns+"ComposeLogs", "docker", "path, [lines]", "Gibt die Logs aller Compose-Dienste zurück.", func(args []Value) Value {
		if len(args) < 1 {
			return Value{Kind: KindStr, Str: "error: Pfad zur compose-Datei benötigt"}
		}
		lines := "50"
		if len(args) >= 2 && args[1].Str != "" {
			lines = args[1].Str
		}
		stdout, stderr, code := dockerExec("compose", "-f", args[0].Str, "logs", "--tail", lines)
		if code != 0 {
			return Value{Kind: KindStr, Str: "error: " + stderr}
		}
		return Value{Kind: KindStr, Str: stdout}
	})

	// =========================================================
	// SYSTEM
	// =========================================================

	Register(ns+"Version", "docker", "-", "Gibt die installierte Docker-Version zurück.", func(args []Value) Value {
		stdout, stderr, code := dockerExec("version", "--format", "{{.Server.Version}}")
		if code != 0 {
			return Value{Kind: KindStr, Str: "error: " + stderr}
		}
		return Value{Kind: KindStr, Str: stdout}
	})

	Register(ns+"IsInstalled", "docker", "-", "Gibt True zurück wenn Docker installiert und erreichbar ist.", func(args []Value) Value {
		_, _, code := dockerExec("info")
		return BoolVal(code == 0)
	})

	Register(ns+"SystemPrune", "docker", "[all]", "Räumt ungenutzte Ressourcen auf. all=true entfernt auch ungenutzte Images.", func(args []Value) Value {
		all := false
		if len(args) >= 1 {
			all = (args[0].Str == "true" || (args[0].Kind == KindBool && args[0].Bool))
		}
		cmdArgs := []string{"system", "prune", "-f"}
		if all {
			cmdArgs = append(cmdArgs, "-a")
		}
		_, stderr, code := dockerExec(cmdArgs...)
		return dockerOk(stderr, code)
	})

	Register(ns+"NetworkList", "docker", "-", "Gibt ein Array aller Docker-Netzwerke zurück.", func(args []Value) Value {
		stdout, stderr, code := dockerExec("network", "ls", "--format", "{{.ID}}|{{.Name}}|{{.Driver}}|{{.Scope}}")
		if code != 0 {
			return Value{Kind: KindStr, Str: "error: " + stderr}
		}
		var result []Value
		for _, line := range strings.Split(stdout, "\n") {
			if line == "" {
				continue
			}
			result = append(result, Value{Kind: KindStr, Str: line})
		}
		return Value{Kind: KindArr, Arr: result}
	})

	Register(ns+"VolumeList", "docker", "-", "Gibt ein Array aller Docker-Volumes zurück.", func(args []Value) Value {
		stdout, stderr, code := dockerExec("volume", "ls", "--format", "{{.Name}}|{{.Driver}}")
		if code != 0 {
			return Value{Kind: KindStr, Str: "error: " + stderr}
		}
		var result []Value
		for _, line := range strings.Split(stdout, "\n") {
			if line == "" {
				continue
			}
			result = append(result, Value{Kind: KindStr, Str: line})
		}
		return Value{Kind: KindArr, Arr: result}
	})

}
