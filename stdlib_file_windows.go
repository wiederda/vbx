//go:build windows

package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"
)

func watchLogInternal(path, pattern, ansiSequence string) Value {
	// Wir nutzen hier kein defer file.Close(), da wir das Handle bei Rotation selbst schließen müssen
	var file *os.File
	var lastPos int64

	// Hilfsfunktion zum "Shared" Öffnen
	openShared := func(p string) (*os.File, error) {
		pathPtr, _ := syscall.UTF16PtrFromString(p)
		handle, err := syscall.CreateFile(
			pathPtr,
			syscall.GENERIC_READ,
			syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE,
			nil,
			syscall.OPEN_EXISTING,
			syscall.FILE_ATTRIBUTE_NORMAL,
			0,
		)
		if err != nil {
			return nil, err
		}
		return os.NewFile(uintptr(handle), p), nil
	}

	// Initiales Öffnen
	f, err := openShared(path)
	if err != nil {
		return ErrorVal("Datei konnte nicht geöffnet werden: " + err.Error())
	}
	file = f
	info, _ := file.Stat()
	lastPos = info.Size()

	fmt.Printf("\033[90m[WATCH] Aktiv: %s | Filter: '%s'\033[0m\n", path, pattern)

	for {
		currInfo, err := os.Stat(path)
		if err != nil {
			// Datei kurzzeitig weg (z.B. während Rotation)? Warten.
			time.Sleep(200 * time.Millisecond)
			continue
		}

		currSize := currInfo.Size()

		// ROTATION CHECK
		if currSize < lastPos {
			fmt.Println("\033[90m[WATCH] Log-Rotation erkannt. Starte neu...\033[0m")
			file.Close()
			f, err := openShared(path)
			if err == nil {
				file = f
				lastPos = 0 // Von vorn anfangen in der neuen Datei
			}
			continue
		}

		if currSize > lastPos {
			newBytes := make([]byte, currSize-lastPos)
			// ReadAt ist super, weil es den internen Cursor nicht bewegt
			_, err := file.ReadAt(newBytes, lastPos)
			if err == nil {
				lines := strings.Split(string(newBytes), "\n")
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line != "" && strings.Contains(strings.ToLower(line), strings.ToLower(pattern)) {
						ts := time.Now().Format("15:04:05")
						if ansiSequence != "" {
							fmt.Printf("%s[%s] %s\033[0m\n", ansiSequence, ts, line)
						} else {
							fmt.Printf("[%s] %s\n", ts, line)
						}
					}
				}
			}
			lastPos = currSize
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// openFileShared öffnet eine Datei unter Windows so, dass andere Prozesse
// (wie Wildfly) weiterhin darin schreiben, lesen oder sie löschen können.
func openFileShared(path string) (*os.File, error) {
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}

	handle, err := syscall.CreateFile(
		pathPtr,
		syscall.GENERIC_READ,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE,
		nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_ATTRIBUTE_NORMAL,
		0,
	)

	if err != nil {
		return nil, err
	}

	// Verwandelt das Win32-Handle in ein Standard Go *os.File Objekt
	return os.NewFile(uintptr(handle), path), nil
}
