// ------------------------
// stdlib_xml.go
// ------------------------

package main

import (
	"encoding/xml"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
)

// ------------------------
// Typen
// ------------------------

type XmlEngine struct {
	mu   sync.RWMutex // FIX: Mutex für thread-sichere Zugriffe
	Root *Node
	Path string
}

type Node struct {
	XMLName xml.Name
	Attrs   []xml.Attr `xml:",any,attr"`
	Content string     `xml:",chardata"`
	Nodes   []*Node    `xml:",any"`
}

var xmlEngine = &XmlEngine{}

// ------------------------
// Hilfsfunktionen (Pfad-Parsing)
// ------------------------

func splitPath(path string) []string {
	if path == "" {
		return []string{}
	}
	return strings.Split(path, ".")
}

// parseStep trennt "user[1]" in "user" und den Index 1
func parseStep(step string) (string, int) {
	if idxStart := strings.Index(step, "["); idxStart != -1 {
		idxEnd := strings.Index(step, "]")
		if idxEnd != -1 {
			name := step[:idxStart]
			idx, err := strconv.Atoi(step[idxStart+1 : idxEnd])
			if err == nil {
				return name, idx
			}
		}
	}
	return step, 0
}

// ------------------------
// Builtins für den Interpreter
// ------------------------

func InitXmlFunctions() {
	if builtins == nil {
		builtins = make(map[string]BuiltinInfo)
	}

	ns := "xml."

	Register(ns+"Load", "xml", "path",
		"Lädt eine XML-Datei vom angegebenen Pfad in den Speicher.", func(args []Value) Value {
			if len(args) < 1 {
				return ErrorVal("Pfad fehlt")
			}
			return xmlEngine.Load(args[0].Str)
		})

	Register(ns+"Save", "xml", "[path]",
		"Speichert die aktuelle XML-Struktur. Wird ein Pfad angegeben, wird dieser als neues Ziel gesetzt.", func(args []Value) Value {
			if len(args) >= 1 && args[0].Str != "" {
				xmlEngine.mu.Lock()
				xmlEngine.Path = args[0].Str
				xmlEngine.mu.Unlock()
			}
			return xmlEngine.Save()
		})

	// FIX: Parse validiert jetzt korrekt durch vollständiges Dekodieren bis EOF,
	// statt xml.Unmarshal(&interface{}) das fast alles akzeptiert.
	Register(ns+"Parse", "xml", "xmlContent",
		"Prüft die XML-Struktur. Gibt True zurück wenn valide, oder eine Fehlermeldung bei Syntaxfehlern.",
		func(args []Value) Value {
			if len(args) < 1 {
				return ErrorVal("Kein XML-Inhalt übergeben")
			}
			dec := xml.NewDecoder(strings.NewReader(ToString(args[0])))
			for {
				_, err := dec.Token()
				if err == io.EOF {
					break
				}
				if err != nil {
					return ErrorVal(err.Error())
				}
			}
			return BoolVal(true)
		})

	// Get: Unterstützt @attribute
	Register(ns+"Get", "xml", "xpath", "Liest Content oder Attribute (@attr).", func(args []Value) Value {
		if len(args) < 1 {
			return StrVal("")
		}
		path := args[0].Str

		attrName := ""
		if idx := strings.Index(path, "@"); idx != -1 {
			attrName = path[idx+1:]
			path = strings.TrimSuffix(path[:idx], ".")
		}

		xmlEngine.mu.RLock()
		node, _, err := xmlEngine.find(path)
		xmlEngine.mu.RUnlock()

		if err != nil || node == nil {
			return StrVal("")
		}
		if attrName != "" {
			return StrVal(node.GetAttr(attrName))
		}
		return StrVal(node.Content)
	})

	// Set: Unterstützt @attribute
	// FIX: Attribute werden jetzt gesetzt ohne den Content des Knotens zu löschen.
	// Vorher: xmlEngine.Set(nodePath, StrVal("")) überschrieb bestehenden Content.
	Register(ns+"Set", "xml", "xpath, value", "Setzt Content oder Attribute (@attr).", func(args []Value) Value {
		if len(args) < 2 {
			return ErrorVal("Pfad/Wert fehlt")
		}
		path := args[0].Str
		val := args[1].Str

		if idx := strings.Index(path, "@"); idx != -1 {
			attrName := path[idx+1:]
			nodePath := strings.TrimSuffix(path[:idx], ".")

			xmlEngine.mu.Lock()
			defer xmlEngine.mu.Unlock()

			// FIX: Node nur anlegen wenn er noch nicht existiert —
			// kein Set("") mehr das den Content überschreibt.
			node, _, _ := xmlEngine.find(nodePath)
			if node == nil {
				// Pfad anlegen ohne Content zu setzen
				if err := xmlEngine.ensurePath(nodePath); err != nil {
					return ErrorVal("Knoten konnte nicht erstellt werden: " + err.Error())
				}
				node, _, _ = xmlEngine.find(nodePath)
			}
			if node == nil {
				return ErrorVal("Knoten konnte nicht erstellt werden")
			}
			node.SetAttr(attrName, val)
			return StrVal("OK")
		}

		xmlEngine.mu.Lock()
		defer xmlEngine.mu.Unlock()
		return xmlEngine.Set(path, args[1])
	})

	Register(ns+"Delete", "xml", "xpath",
		"Löscht den spezifizierten Knoten und alle seine Unterknoten.", func(args []Value) Value {
			if len(args) < 1 {
				return ErrorVal("Pfad fehlt")
			}
			xmlEngine.mu.Lock()
			defer xmlEngine.mu.Unlock()
			return xmlEngine.Delete(args[0].Str)
		})

	Register(ns+"ToMap", "xml", "[xpath]",
		"Konvertiert den geladenen XML-Baum (oder einen Teilbaum ab xpath) in eine Map-Struktur.", func(args []Value) Value {
			path := ""
			if len(args) > 0 {
				path = args[0].Str
			}

			xmlEngine.mu.RLock()
			defer xmlEngine.mu.RUnlock()

			var target *Node
			if path == "" {
				target = xmlEngine.Root
			} else {
				node, _, err := xmlEngine.find(path)
				if err != nil {
					return ErrorVal("xml.ToMap: " + err.Error())
				}
				target = node
			}

			if target == nil {
				return ErrorVal("xml.ToMap: kein XML geladen oder Pfad nicht gefunden")
			}

			return nodeToValue(target)
		})

	Register(ns+"Count", "xml", "xpath",
		"Zählt, wie viele Geschwister-Knoten mit demselben Namen unter dem Parent-Pfad existieren.", func(args []Value) Value {
			if len(args) < 1 {
				return NumVal(0)
			}
			parts := splitPath(args[0].Str)
			if len(parts) == 0 {
				return NumVal(0)
			}

			xmlEngine.mu.RLock()
			defer xmlEngine.mu.RUnlock()

			if len(parts) == 1 {
				if xmlEngine.Root != nil && xmlEngine.Root.XMLName.Local == parts[0] {
					return NumVal(1)
				}
				return NumVal(0)
			}

			parentPath := strings.Join(parts[:len(parts)-1], ".")
			tagName, _ := parseStep(parts[len(parts)-1])

			parent, _, err := xmlEngine.find(parentPath)
			if err != nil {
				return NumVal(0)
			}

			count := 0
			for _, n := range parent.Nodes {
				if n.XMLName.Local == tagName {
					count++
				}
			}
			return NumVal(float64(count))
		})

	Register(ns+"Keys", "xml", "[xpath]",
		"Gibt ein Array mit den Namen aller direkten Unterknoten des angegebenen Pfads zurück.", func(args []Value) Value {
			path := ""
			if len(args) > 0 {
				path = args[0].Str
			}

			xmlEngine.mu.RLock()
			defer xmlEngine.mu.RUnlock()

			var target *Node
			if path == "" {
				target = xmlEngine.Root
			} else {
				node, _, _ := xmlEngine.find(path)
				target = node
			}

			if target == nil {
				return Value{Kind: KindArr}
			}

			arr := make([]Value, len(target.Nodes))
			for i, n := range target.Nodes {
				arr[i] = StrVal(n.XMLName.Local)
			}
			return Value{Kind: KindArr, Arr: arr}
		})
}

// ------------------------
// Engine Kern-Logik
// ------------------------

func (x *XmlEngine) Load(path string) Value {
	data, err := os.ReadFile(path)
	if err != nil {
		x.mu.Lock()
		x.Root = nil
		x.Path = path
		x.mu.Unlock()
		return ErrorVal(err.Error())
	}

	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	var stack []*Node
	var root *Node

	// FIX: Fehler beim Dekodieren werden jetzt unterschieden:
	// io.EOF = normales Ende, alles andere = korruptes XML → ErrorVal statt stilles Abbrechen.
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return ErrorVal("XML Parse-Fehler: " + err.Error())
		}

		switch t := tok.(type) {
		case xml.StartElement:
			node := &Node{XMLName: t.Name, Attrs: t.Attr}
			if len(stack) == 0 {
				root = node
			} else {
				parent := stack[len(stack)-1]
				parent.Nodes = append(parent.Nodes, node)
			}
			stack = append(stack, node)
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case xml.CharData:
			if len(stack) > 0 {
				text := strings.TrimSpace(string(t))
				if text != "" {
					stack[len(stack)-1].Content = text
				}
			}
		}
	}

	x.mu.Lock()
	x.Root = root
	x.Path = path
	x.mu.Unlock()

	return StrVal("OK")
}

// find navigiert den Node-Tree anhand eines Punktpfads.
// Muss unter x.mu (Read oder Write) aufgerufen werden.
func (x *XmlEngine) find(path string) (*Node, *Node, error) {
	parts := splitPath(path)
	if x.Root == nil || len(parts) == 0 {
		return nil, nil, errors.New("not found")
	}

	rootName, rootIdx := parseStep(parts[0])

	// FIX: rootIdx != 0 verhinderte vorher immer root[0] zu finden.
	// Korrektur: Root-Element hat immer Index 0, andere Indizes sind ungültig.
	if x.Root.XMLName.Local != rootName || rootIdx != 0 {
		return nil, nil, errors.New("root mismatch")
	}

	current := x.Root
	var parent *Node

	for _, p := range parts[1:] {
		tagName, targetIdx := parseStep(p)
		found := false
		count := 0

		for _, n := range current.Nodes {
			if n.XMLName.Local == tagName {
				if count == targetIdx {
					parent = current
					current = n
					found = true
					break
				}
				count++
			}
		}
		if !found {
			return nil, nil, errors.New("path not found: " + p)
		}
	}
	return current, parent, nil
}

// ensurePath legt alle fehlenden Knoten entlang eines Pfads an,
// ohne bestehenden Content zu verändern.
// Muss unter x.mu.Lock() aufgerufen werden.
func (x *XmlEngine) ensurePath(path string) error {
	parts := splitPath(path)
	if len(parts) == 0 {
		return errors.New("invalid path")
	}

	rootName, _ := parseStep(parts[0])
	if x.Root == nil {
		x.Root = &Node{XMLName: xml.Name{Local: rootName}}
	}
	if x.Root.XMLName.Local != rootName {
		return errors.New("root mismatch")
	}

	current := x.Root
	for _, p := range parts[1:] {
		tagName, targetIdx := parseStep(p)
		var next *Node
		count := 0

		for _, n := range current.Nodes {
			if n.XMLName.Local == tagName {
				if count == targetIdx {
					next = n
					break
				}
				count++
			}
		}
		if next == nil {
			for i := count; i <= targetIdx; i++ {
				newNode := &Node{XMLName: xml.Name{Local: tagName}}
				current.Nodes = append(current.Nodes, newNode)
				if i == targetIdx {
					next = newNode
				}
			}
		}
		current = next
	}
	return nil
}

func (x *XmlEngine) Set(path string, val Value) Value {
	parts := splitPath(path)
	if len(parts) == 0 {
		return ErrorVal("invalid path")
	}

	if err := x.ensurePath(path); err != nil {
		return ErrorVal(err.Error())
	}

	node, _, err := x.find(path)
	if err != nil {
		return ErrorVal(err.Error())
	}
	node.Content = val.Str
	return StrVal("OK")
}

func (x *XmlEngine) Delete(path string) Value {
	node, parent, err := x.find(path)
	if err != nil || parent == nil {
		return ErrorVal("not found")
	}

	for i, n := range parent.Nodes {
		if n == node {
			parent.Nodes = append(parent.Nodes[:i], parent.Nodes[i+1:]...)
			return StrVal("OK")
		}
	}
	return ErrorVal("delete failed")
}

func (x *XmlEngine) Save() Value {
	x.mu.RLock()
	root := x.Root
	path := x.Path
	x.mu.RUnlock()

	if root == nil {
		return ErrorVal("nothing to save")
	}
	file, err := os.Create(path)
	if err != nil {
		return ErrorVal(err.Error())
	}
	defer file.Close()

	enc := xml.NewEncoder(file)
	enc.Indent("", "  ")
	if err := enc.Encode(root); err != nil {
		return ErrorVal(err.Error())
	}
	return StrVal("OK")
}

func (n *Node) GetAttr(name string) string {
	for _, a := range n.Attrs {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}

func nodeToValue(n *Node) Value {
	// Blattknoten ohne Kinder und ohne Attribute: einfacher String
	if len(n.Nodes) == 0 && len(n.Attrs) == 0 {
		return StrVal(n.Content)
	}

	m := make(map[string]Value)

	for _, a := range n.Attrs {
		m["@"+a.Name.Local] = StrVal(a.Value)
	}

	if strings.TrimSpace(n.Content) != "" {
		m["_text"] = StrVal(n.Content)
	}

	// Kinder gruppieren: gleicher Name -> Array, sonst einzelner Wert
	childGroups := make(map[string][]Value)
	var order []string
	for _, child := range n.Nodes {
		name := child.XMLName.Local
		if _, exists := childGroups[name]; !exists {
			order = append(order, name)
		}
		childGroups[name] = append(childGroups[name], nodeToValue(child))
	}

	for _, name := range order {
		vals := childGroups[name]
		if len(vals) == 1 {
			m[name] = vals[0]
		} else {
			m[name] = Value{Kind: KindArr, Arr: vals}
		}
	}

	return Value{Kind: KindMap, Map: m}
}

func (n *Node) SetAttr(name, value string) {
	for i, a := range n.Attrs {
		if a.Name.Local == name {
			n.Attrs[i].Value = value
			return
		}
	}
	n.Attrs = append(n.Attrs, xml.Attr{Name: xml.Name{Local: name}, Value: value})
}
