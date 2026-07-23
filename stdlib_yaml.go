package main

import (
	"errors"
	"io"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

func InitYamlFunctions() {
	if builtins == nil {
		builtins = make(map[string]BuiltinInfo)
	}

	ns := "yaml."

	Register(ns+"Parse", "yaml", "yamlContent",
		"Prüft die YAML-Struktur. Gibt True zurück, wenn valide, oder eine Fehlermeldung mit Zeilenangabe bei Syntaxfehlern.",
		func(args []Value) Value {
			if len(args) < 1 || ToString(args[0]) == "" {
				return StrVal("ERROR: Kein Inhalt zum Prüfen übergeben")
			}

			var data interface{}
			err := yaml.Unmarshal([]byte(ToString(args[0])), &data)

			if err != nil {
				// yaml.v3 liefert Fehlermeldungen wie: "yaml: line 4: found character that cannot start any token"
				return StrVal("ERROR: " + err.Error())
			}

			return BoolVal(true)
		})

	Register(ns+"ParseAll", "yaml", "yamlContent",
		"Prüft den gesamten YAML-Stream (alle Dokumente). Gibt True zurück, wenn der gesamte Inhalt valide ist, oder die Fehlermeldung inkl. Zeilenangabe bei Syntaxfehlern.",
		func(args []Value) Value {
			if len(args) < 1 || args[0].Str == "" {
				return StrVal("ERROR: Kein Inhalt zum Parsen übergeben")
			}

			// Wir nutzen den Decoder, um über alle YAML-Dokumente im String zu laufen
			decoder := yaml.NewDecoder(strings.NewReader(ToString(args[0])))

			for {
				var data interface{}
				err := decoder.Decode(&data)

				// Wenn der Decoder am Ende des Dokuments/Streams ist, ist alles OK
				if err == io.EOF {
					break
				}

				// Sobald ein Dokument im Stream einen Fehler hat, geben wir diesen zurück
				if err != nil {
					return StrVal("ERROR: " + err.Error())
				}
			}

			// Wenn der Loop ohne Fehler durchläuft, ist der gesamte Stream valide
			return BoolVal(true)
		})

	Register(ns+"Get", "yaml", "yaml, path, v", "Liest Wert via Pfad.", func(args []Value) Value {
		if len(args) < 2 {
			return ErrorVal("Argumente fehlen")
		}
		var data interface{}
		if err := yaml.Unmarshal([]byte(args[0].Str), &data); err != nil {
			return ErrorVal(err.Error())
		}

		// Korrektur: 2 Rückgabewerte & splitPath
		val, ok := getValueByPath(data, splitPath(args[1].Str))
		if !ok {
			return Value{}
		}
		return interfaceToValue(val)
	})

	Register(ns+"Set", "yaml", "yaml, path, val", "Setzt Wert und gibt YAML zurück.", func(args []Value) Value {
		if len(args) < 3 {
			return ErrorVal("Argumente fehlen")
		}
		var data interface{}
		if err := yaml.Unmarshal([]byte(args[0].Str), &data); err != nil {
			return ErrorVal(err.Error())
		}

		updated, err := setValueByPath(data, splitPath(args[1].Str), valToInterface(args[2]))
		if err != nil {
			return ErrorVal(err.Error())
		}

		out, _ := yaml.Marshal(updated)
		return StrVal(string(out))
	})

	Register(ns+"Stringify", "yaml", "value",
		"Konvertiert eine vbmini-Struktur (Map/Array) in einen formatierten YAML-String.",
		func(args []Value) Value {
			if len(args) < 1 {
				return StrVal("")
			}

			// interfaceFromValue ist das Gegenstück zu interfaceToValue
			raw := valToInterface(args[0])
			out, err := yaml.Marshal(raw)
			if err != nil {
				return ErrorVal(err.Error())
			}

			return StrVal(string(out))
		})
}

// GET: Navigiert rekursiv. Rückgabe: (Wert, Erfolg)
func getValueByPath(current interface{}, keys []string) (interface{}, bool) {
	if len(keys) == 0 {
		return current, true
	}

	switch node := current.(type) {
	case map[string]interface{}:
		next, ok := node[keys[0]]
		if !ok {
			return nil, false
		}
		return getValueByPath(next, keys[1:])
	case []interface{}:
		index, err := strconv.Atoi(keys[0])
		if err != nil || index < 0 || index >= len(node) {
			return nil, false
		}
		return getValueByPath(node[index], keys[1:])
	}
	return nil, false
}

// SET: Schreibt Werte und baut Pfade ggf. aus
func setValueByPath(current interface{}, keys []string, value interface{}) (interface{}, error) {
	if len(keys) == 0 {
		return value, nil
	}

	switch node := current.(type) {
	case map[string]interface{}:
		key := keys[0]
		if len(keys) == 1 {
			node[key] = value
			return node, nil
		}
		next, ok := node[key]
		if !ok {
			next = make(map[string]interface{})
		}
		updated, err := setValueByPath(next, keys[1:], value) // NEU
		node[key] = updated                                   // NEU
		return node, err

	case []interface{}:
		index, err := strconv.Atoi(keys[0])
		if err != nil || index < 0 || index >= len(node) {
			return nil, errors.New("index out of range")
		}
		if len(keys) == 1 {
			node[index] = value
			return node, nil
		}
		updated, err := setValueByPath(node[index], keys[1:], value) // NEU
		node[index] = updated                                        // NEU
		return node, err
	}
	return current, errors.New("invalid path structure")
}
