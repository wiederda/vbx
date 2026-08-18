//go:build windows
// +build windows

package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

func regRoot(root string) registry.Key {
	switch root {
	case "HKCU":
		return registry.CURRENT_USER
	case "HKLM":
		return registry.LOCAL_MACHINE
	default:
		return registry.CURRENT_USER
	}
}

// löscht einen Registry-Key rekursiv inkl. Unterkeys
func deleteKeyRecursive(root registry.Key, path string) error {
	k, err := registry.OpenKey(root, path, registry.ENUMERATE_SUB_KEYS|registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	subkeys, err := k.ReadSubKeyNames(-1)
	if err != nil {
		return err
	}
	for _, sub := range subkeys {
		if err := deleteKeyRecursive(k, sub); err != nil {
			return err
		}
	}

	k.Close()
	return registry.DeleteKey(root, path)
}

// --- DPAPI ---

var (
	modcrypt32             = windows.NewLazySystemDLL("crypt32.dll")
	modkernel32            = windows.NewLazySystemDLL("kernel32.dll")
	procCryptProtectData   = modcrypt32.NewProc("CryptProtectData")
	procCryptUnprotectData = modcrypt32.NewProc("CryptUnprotectData")
	procLocalFree          = modkernel32.NewProc("LocalFree")
)

type dataBlob struct {
	cbData uint32
	pbData *byte
}

// CRYPTPROTECT_UI_FORBIDDEN: keine UI-Prompts, entspricht DataProtectionScope.CurrentUser
const cryptprotectUIForbidden = 0x1

func dpapiProtect(data []byte) ([]byte, error) {
	if len(data) == 0 {
		data = []byte{0}
	}
	in := dataBlob{cbData: uint32(len(data)), pbData: &data[0]}
	var out dataBlob
	r, _, err := procCryptProtectData.Call(
		uintptr(unsafe.Pointer(&in)),
		0, 0, 0, 0,
		cryptprotectUIForbidden,
		uintptr(unsafe.Pointer(&out)),
	)
	if r == 0 {
		return nil, err
	}
	defer procLocalFree.Call(uintptr(unsafe.Pointer(out.pbData)))
	result := make([]byte, out.cbData)
	copy(result, unsafe.Slice(out.pbData, out.cbData))
	return result, nil
}

func dpapiUnprotect(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("leere Daten")
	}
	in := dataBlob{cbData: uint32(len(data)), pbData: &data[0]}
	var out dataBlob
	r, _, err := procCryptUnprotectData.Call(
		uintptr(unsafe.Pointer(&in)),
		0, 0, 0, 0,
		cryptprotectUIForbidden,
		uintptr(unsafe.Pointer(&out)),
	)
	if r == 0 {
		return nil, err
	}
	defer procLocalFree.Call(uintptr(unsafe.Pointer(out.pbData)))
	result := make([]byte, out.cbData)
	copy(result, unsafe.Slice(out.pbData, out.cbData))
	return result, nil
}

func InitRegistryFunctions() {
	if builtins == nil {
		builtins = make(map[string]BuiltinInfo)
	}

	ns := "reg."

	Register(ns+"Read", "reg", "root, path, name",
		"Liest einen String-Wert aus der Registry. 'root' kann 'HKCU' oder 'HKLM' sein.", func(args []Value) Value {
			if len(args) < 3 {
				return ErrorVal("reg.Read erwartet 3 Argumente")
			}
			root := regRoot(args[0].Str)
			keyPath := args[1].Str
			valueName := args[2].Str

			k, err := registry.OpenKey(root, keyPath, registry.QUERY_VALUE)
			if err != nil {
				return ErrorVal(fmt.Sprintf("reg.Read: Key '%s\\%s' nicht gefunden oder nicht zugreifbar: %s", args[0].Str, keyPath, err.Error()))
			}
			defer k.Close()

			val, _, err := k.GetStringValue(valueName)
			if err != nil {
				return ErrorVal(fmt.Sprintf("reg.Read: Wert '%s' in '%s\\%s' nicht gefunden: %s", valueName, args[0].Str, keyPath, err.Error()))
			}
			return StrVal(val)
		})

	Register(ns+"Write", "reg", "root, path, name, data",
		"Schreibt einen String-Wert in die Registry. Erstellt den Key automatisch, falls er nicht existiert.", func(args []Value) Value {
			if len(args) < 4 {
				return ErrorVal("reg.Write erwartet 4 Argumente")
			}
			root := regRoot(args[0].Str)
			keyPath := args[1].Str
			valueName := args[2].Str
			valueData := args[3].Str

			k, _, err := registry.CreateKey(root, keyPath, registry.SET_VALUE)
			if err != nil {
				return ErrorVal(fmt.Sprintf("reg.Write: Key '%s\\%s' konnte nicht erstellt/geöffnet werden: %s", args[0].Str, keyPath, err.Error()))
			}
			defer k.Close()

			err = k.SetStringValue(valueName, valueData)
			if err != nil {
				return ErrorVal(fmt.Sprintf("reg.Write: Wert '%s' in '%s\\%s' konnte nicht geschrieben werden: %s", valueName, args[0].Str, keyPath, err.Error()))
			}
			return StrVal("OK")
		})

	// reg.WriteProtectedValue
	Register(ns+"WriteProtectedValue", "reg", "root, path, name, data",
		"Speichert einen String DPAPI-verschlüsselt in der Registry. Der Wert kann nur vom aktuell angemeldeten Windows-Benutzer auf diesem Rechner wieder entschlüsselt werden.", func(args []Value) Value {
			if len(args) < 4 {
				return ErrorVal("reg.WriteProtectedValue erwartet 4 Argumente")
			}
			root := regRoot(args[0].Str)
			keyPath := args[1].Str
			valueName := args[2].Str
			valueData := args[3].Str

			protected, err := dpapiProtect([]byte(valueData))
			if err != nil {
				return ErrorVal("reg.WriteProtectedValue: DPAPI-Verschlüsselung fehlgeschlagen: " + err.Error())
			}
			b64 := base64.StdEncoding.EncodeToString(protected)

			k, _, err := registry.CreateKey(root, keyPath, registry.SET_VALUE)
			if err != nil {
				return ErrorVal(fmt.Sprintf("reg.WriteProtectedValue: Key '%s\\%s' konnte nicht erstellt/geöffnet werden: %s", args[0].Str, keyPath, err.Error()))
			}
			defer k.Close()

			if err := k.SetStringValue(valueName, b64); err != nil {
				return ErrorVal(fmt.Sprintf("reg.WriteProtectedValue: Wert '%s' in '%s\\%s' konnte nicht geschrieben werden: %s", valueName, args[0].Str, keyPath, err.Error()))
			}
			return StrVal("OK")
		})

	// reg.ReadProtectedValue
	Register(ns+"ReadProtectedValue", "reg", "root, path, name",
		"Liest einen DPAPI-verschlüsselten String aus der Registry und entschlüsselt ihn für den aktuell angemeldeten Windows-Benutzer.", func(args []Value) Value {
			if len(args) < 3 {
				return ErrorVal("reg.ReadProtectedValue erwartet 3 Argumente")
			}
			root := regRoot(args[0].Str)
			keyPath := args[1].Str
			valueName := args[2].Str

			k, err := registry.OpenKey(root, keyPath, registry.QUERY_VALUE)
			if err != nil {
				return ErrorVal(fmt.Sprintf("reg.ReadProtectedValue: Key '%s\\%s' nicht gefunden oder nicht zugreifbar: %s", args[0].Str, keyPath, err.Error()))
			}
			defer k.Close()

			b64, _, err := k.GetStringValue(valueName)
			if err != nil {
				return ErrorVal(fmt.Sprintf("reg.ReadProtectedValue: Wert '%s' in '%s\\%s' nicht gefunden: %s", valueName, args[0].Str, keyPath, err.Error()))
			}

			raw, err := base64.StdEncoding.DecodeString(b64)
			if err != nil {
				return ErrorVal("reg.ReadProtectedValue: Base64-Dekodierung fehlgeschlagen (Wert beschädigt?): " + err.Error())
			}

			plain, err := dpapiUnprotect(raw)
			if err != nil {
				return ErrorVal("reg.ReadProtectedValue: DPAPI-Entschlüsselung fehlgeschlagen (falscher Benutzer/Rechner?): " + err.Error())
			}
			return StrVal(string(plain))
		})

	// reg.ReadProtectedValueBytes
	Register(ns+"ReadProtectedValueBytes", "reg", "root, path, name",
		"Liest einen DPAPI-verschlüsselten Wert und entschlüsselt ihn für den aktuell angemeldeten Windows-Benutzer. Gibt die Rohdaten als Byte-Array zurück (statt String), damit der Wert nach Gebrauch gezielt mit crypt.Wipe überschrieben werden kann.", func(args []Value) Value {
			if len(args) < 3 {
				return ErrorVal("reg.ReadProtectedValueBytes erwartet 3 Argumente")
			}
			root := regRoot(args[0].Str)
			keyPath := args[1].Str
			valueName := args[2].Str

			k, err := registry.OpenKey(root, keyPath, registry.QUERY_VALUE)
			if err != nil {
				return ErrorVal(fmt.Sprintf("reg.ReadProtectedValueBytes: Key '%s\\%s' nicht gefunden oder nicht zugreifbar: %s", args[0].Str, keyPath, err.Error()))
			}
			defer k.Close()

			b64, _, err := k.GetStringValue(valueName)
			if err != nil {
				return ErrorVal(fmt.Sprintf("reg.ReadProtectedValueBytes: Wert '%s' in '%s\\%s' nicht gefunden: %s", valueName, args[0].Str, keyPath, err.Error()))
			}

			raw, err := base64.StdEncoding.DecodeString(b64)
			if err != nil {
				return ErrorVal("reg.ReadProtectedValueBytes: Base64-Dekodierung fehlgeschlagen (Wert beschädigt?): " + err.Error())
			}

			plain, err := dpapiUnprotect(raw)
			if err != nil {
				return ErrorVal("reg.ReadProtectedValueBytes: DPAPI-Entschlüsselung fehlgeschlagen (falscher Benutzer/Rechner?): " + err.Error())
			}

			// plain als VBX-Byte-Array zurückgeben (jedes Element 0-255)
			// WICHTIG: plain selbst danach nullen, da wir es hier kopieren.
			arr := make([]Value, len(plain))
			for i, b := range plain {
				arr[i] = NumVal(float64(b))
			}
			for i := range plain {
				plain[i] = 0
			}

			return Value{Kind: KindArr, Arr: arr}
		})

	// reg.Exists
	Register(ns+"Exists", "reg", "root, path [, name]",
		"Prüft, ob ein Registry-Key oder ein spezifischer Wert existiert. Gibt true oder false zurück.", func(args []Value) Value {

			if len(args) < 2 || len(args) > 3 {
				return ErrorVal("reg.Exists erwartet 2 oder 3 Argumente")
			}

			if args[0].Kind != KindStr || args[1].Kind != KindStr {
				return ErrorVal("reg.Exists: 'root' und 'path' müssen Strings sein")
			}

			root := regRoot(args[0].Str)
			keyPath := args[1].Str

			// Key prüfen
			if len(args) == 2 {
				k, err := registry.OpenKey(root, keyPath, registry.QUERY_VALUE)
				if err != nil {
					return BoolVal(false)
				}
				k.Close()
				return BoolVal(true)
			}

			// Value prüfen
			if args[2].Kind != KindStr {
				return ErrorVal(fmt.Sprintf("reg.Exists: 'name' muss ein String sein (Key: '%s\\%s')", args[0].Str, keyPath))
			}

			valueName := args[2].Str

			k, err := registry.OpenKey(root, keyPath, registry.QUERY_VALUE)
			if err != nil {
				return BoolVal(false)
			}
			defer k.Close()

			_, _, err = k.GetValue(valueName, nil)
			return BoolVal(err == nil)
		})

	Register(ns+"Delete", "reg", "root, path [, name]",
		"Löscht einen Wert oder einen ganzen Key rekursiv (inkl. aller Unterkeys!). Ohne 'name' wird der gesamte Pfad gelöscht. Vorsicht geboten.", func(args []Value) Value {

			if len(args) < 2 || len(args) > 3 {
				return ErrorVal("reg.Delete erwartet 2 oder 3 Argumente")
			}

			root := regRoot(args[0].Str)
			keyPath := args[1].Str

			// --- reg.Delete(root, keyPath) ---
			if len(args) == 2 {
				err := deleteKeyRecursive(root, keyPath)
				if err != nil {
					return ErrorVal(fmt.Sprintf("reg.Delete: Key '%s\\%s' konnte nicht rekursiv gelöscht werden: %s", args[0].Str, keyPath, err.Error()))
				}
				return StrVal("OK")
			}

			// --- reg.Delete(root, keyPath, valueName) ---
			valueName := args[2].Str

			k, err := registry.OpenKey(root, keyPath, registry.SET_VALUE)
			if err != nil {
				return ErrorVal(fmt.Sprintf("reg.Delete: Key '%s\\%s' nicht gefunden oder nicht zugreifbar: %s", args[0].Str, keyPath, err.Error()))
			}
			defer k.Close()

			err = k.DeleteValue(valueName)
			if err != nil {
				return ErrorVal(fmt.Sprintf("reg.Delete: Wert '%s' in '%s\\%s' konnte nicht gelöscht werden: %s", valueName, args[0].Str, keyPath, err.Error()))
			}

			return StrVal("OK")
		})
}
