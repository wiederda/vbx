// ------------------------
// stdlib_steg.go
// ------------------------

package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"

	"golang.org/x/image/bmp"
)

// =============================================================================
// HELPER-FUNKTIONEN
// =============================================================================

// strideFromSeed berechnet den deterministischen Stride-Wert aus einem Seed.
// Zentralisiert die 5-fach duplizierte Berechnung.
func strideFromSeed(seedStr string) int {
	h := sha256.Sum256([]byte(seedStr))
	return 2 + int(h[0]%3)
}

// isBMP erkennt BMP-Dateien anhand des Magic-Headers.
func isBMP(data []byte) bool {
	return len(data) >= 2 && data[0] == 'B' && data[1] == 'M'
}

// decodeImageSafe dekodiert ein Bild und gibt einen sprechenden Fehler zurück.
// Verhindert Nil-Pointer-Panics bei fehlerhaften Bilddaten.
func decodeImageSafe(data []byte) (image.Image, string, error) {
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("bild konnte nicht dekodiert werden: %w", err)
	}
	return img, format, nil
}

// toRGBA konvertiert ein beliebiges image.Image in ein *image.RGBA.
func toRGBA(img image.Image) *image.RGBA {
	b := img.Bounds()
	out := image.NewRGBA(b)
	draw.Draw(out, b, img, b.Min, draw.Src)
	return out
}

// buildPayload erzeugt den vollständigen Payload: STEG-Header (4 Bytes) +
// Länge (4 Bytes Little-Endian) + Nutzdaten.
func buildPayload(data []byte) []byte {
	header := make([]byte, 8)
	copy(header[:4], "STEG")
	binary.LittleEndian.PutUint32(header[4:], uint32(len(data)))
	return append(header, data...)
}

// checkBounds prüft ob genug Einheiten für einen Payload vorhanden sind.
// Gibt einen Fehler zurück, bevor irgendetwas geschrieben wird.
func checkBounds(availableUnits, payloadBytes, stride int) error {
	required := payloadBytes * 8 * stride
	if required > availableUnits {
		return fmt.Errorf(
			"payload zu groß: %d Bits benötigt, %d verfügbar (stride=%d)",
			required, availableUnits, stride,
		)
	}
	return nil
}

// securePermutation erzeugt eine AES-CTR-basierte Permutation der Indizes 0..n-1.
func securePermutation(n int, seedStr string) []int {
	key := sha256.Sum256([]byte(seedStr))
	block, _ := aes.NewCipher(key[:])
	stream := cipher.NewCTR(block, make([]byte, aes.BlockSize))

	arr := make([]int, n)
	for i := range arr {
		arr[i] = i
	}

	for i := n - 1; i > 0; i-- {
		j := int(nextUint64(stream) % uint64(i+1))
		arr[i], arr[j] = arr[j], arr[i]
	}
	return arr
}

func nextUint64(s cipher.Stream) uint64 {
	b := make([]byte, 8)
	s.XORKeyStream(b, b)
	return binary.LittleEndian.Uint64(b)
}

// bytesToBits wandelt Bytes in eine Bit-Slice (MSB first) um.
func bytesToBits(data []byte) []uint8 {
	out := make([]uint8, len(data)*8)
	for i, b := range data {
		for j := 0; j < 8; j++ {
			out[i*8+j] = (b >> (7 - j)) & 1
		}
	}
	return out
}

// bitsToBytes wandelt eine Bit-Slice (MSB first) in Bytes um.
func bitsToBytes(bits []uint8) []byte {
	out := make([]byte, len(bits)/8)
	for i := range out {
		for j := 0; j < 8; j++ {
			out[i] |= bits[i*8+j] << (7 - j)
		}
	}
	return out
}

// =============================================================================
// BMP-IMPLEMENTIERUNG
// =============================================================================

func injectCore(bmpData []byte, secret []byte, seedStr string) ([]byte, error) {
	offset := int(binary.LittleEndian.Uint32(bmpData[10:14]))
	available := len(bmpData) - offset

	payload := buildPayload(secret)
	stride := strideFromSeed(seedStr)

	if err := checkBounds(available, len(payload), stride); err != nil {
		return nil, err
	}

	indices := securePermutation(available, seedStr)

	for i := 0; i < len(payload)*8; i++ {
		idx := offset + indices[i*stride]
		bit := (payload[i/8] >> (7 - (i % 8))) & 1
		bmpData[idx] = (bmpData[idx] & 0xFE) | bit
	}

	return bmpData, nil
}

func extractCore(bmpData []byte, seedStr string) ([]byte, error) {
	offset := int(binary.LittleEndian.Uint32(bmpData[10:14]))
	available := len(bmpData) - offset

	stride := strideFromSeed(seedStr)
	indices := securePermutation(available, seedStr)

	// Bounds-Prüfung für den Header (8 Bytes = 64 Bits)
	if err := checkBounds(available, 8, stride); err != nil {
		return nil, fmt.Errorf("bild zu klein für Header: %w", err)
	}

	header := make([]byte, 8)
	for i := 0; i < 64; i++ {
		idx := offset + indices[i*stride]
		header[i/8] |= (bmpData[idx] & 1) << (7 - (i % 8))
	}

	if string(header[:4]) != "STEG" {
		return nil, fmt.Errorf("falscher Seed oder kein STEG-Header")
	}

	length := int(binary.LittleEndian.Uint32(header[4:]))

	// Bounds-Prüfung für den vollen Payload
	if err := checkBounds(available, 8+length, stride); err != nil {
		return nil, fmt.Errorf("angegebene payload-länge überschreitet bild: %w", err)
	}

	out := make([]byte, length)
	for i := 0; i < length*8; i++ {
		idx := offset + indices[(64+i)*stride]
		out[i/8] |= (bmpData[idx] & 1) << (7 - (i % 8))
	}

	return out, nil
}

// =============================================================================
// PNG / UNIVERSAL-IMPLEMENTIERUNG
// =============================================================================

func injectImageUniversal(fileData []byte, secret []byte, seedStr string) ([]byte, error) {
	img, format, err := decodeImageSafe(fileData)
	if err != nil {
		return nil, err
	}

	rgba := toRGBA(img)
	b := rgba.Bounds()
	total := b.Dx() * b.Dy()

	payload := buildPayload(secret)
	stride := strideFromSeed(seedStr)

	if err := checkBounds(total, len(payload), stride); err != nil {
		return nil, err
	}

	indices := securePermutation(total, seedStr)
	bits := bytesToBits(payload)

	for i, bit := range bits {
		idx := indices[i*stride]
		x := b.Min.X + (idx % b.Dx())
		y := b.Min.Y + (idx / b.Dx())

		c := rgba.RGBAAt(x, y)
		if bit == 1 {
			c.R |= 1
		} else {
			c.R &^= 1
		}
		rgba.SetRGBA(x, y, c)
	}

	var buf bytes.Buffer
	if format == "png" {
		if err := png.Encode(&buf, rgba); err != nil {
			return nil, fmt.Errorf("PNG-kodierung fehlgeschlagen: %w", err)
		}
	} else {
		if err := bmp.Encode(&buf, rgba); err != nil {
			return nil, fmt.Errorf("BMP-kodierung fehlgeschlagen: %w", err)
		}
	}

	return buf.Bytes(), nil
}

func extractImageUniversal(fileData []byte, seedStr string) ([]byte, error) {
	img, _, err := decodeImageSafe(fileData)
	if err != nil {
		return nil, err
	}

	rgba := toRGBA(img)
	b := rgba.Bounds()
	total := b.Dx() * b.Dy()

	stride := strideFromSeed(seedStr)
	indices := securePermutation(total, seedStr)

	// Schritt 1: Header (64 Bits = 8 Bytes) lesen
	if err := checkBounds(total, 8, stride); err != nil {
		return nil, fmt.Errorf("bild zu klein für Header: %w", err)
	}

	headerBits := make([]uint8, 64)
	for i := 0; i < 64; i++ {
		idx := indices[i*stride]
		x := b.Min.X + (idx % b.Dx())
		y := b.Min.Y + (idx / b.Dx())
		headerBits[i] = rgba.RGBAAt(x, y).R & 1
	}

	if string(bitsToBytes(headerBits[:32])) != "STEG" {
		return nil, fmt.Errorf("kein STEG-Header — falscher Seed?")
	}

	length := int(binary.LittleEndian.Uint32(bitsToBytes(headerBits[32:64])))

	// Schritt 2: Payload lesen
	if err := checkBounds(total, 8+length, stride); err != nil {
		return nil, fmt.Errorf("angegebene payload-länge überschreitet bild: %w", err)
	}

	totalBits := (8 + length) * 8
	payloadBits := make([]uint8, totalBits)
	copy(payloadBits, headerBits)

	// Separater Index für den Payload-Teil (kein Mutation der Schleifenvariable)
	for i := 64; i < totalBits; i++ {
		idx := indices[i*stride]
		x := b.Min.X + (idx % b.Dx())
		y := b.Min.Y + (idx / b.Dx())
		payloadBits[i] = rgba.RGBAAt(x, y).R & 1
	}

	full := bitsToBytes(payloadBits)
	return full[8:], nil
}

// =============================================================================
// REGISTRIERUNG
// =============================================================================

func InitStegFunctions() {
	ns := "steg."

	// ---------------------------------------------------------------------------
	// steg.Inject(inPath, outPath, dataB64, seed)  →  [bool, seed, err]
	// ---------------------------------------------------------------------------
	Register(ns+"Inject", "steg", "inPath, outPath, dataB64, seed",
		"Bettet Base64-kodierte Daten per LSB-Steganografie in ein Bild ein.",
		func(args []Value) Value {
			if len(args) < 4 {
				return errResult("Parameter fehlen: steg.Inject(inPath, outPath, dataB64, seed)")
			}

			inPath, errVal := absPathVal(args[0].Str)
			if errVal != nil {
				return errResult(fmt.Sprintf("ungültiger Eingabepfad '%s'", args[0].Str))
			}

			outPath, errVal := absPathVal(args[1].Str)
			if errVal != nil {
				return errResult(fmt.Sprintf("ungültiger Ausgabepfad '%s'", args[1].Str))
			}

			data, err := base64.StdEncoding.DecodeString(args[2].Str)
			if err != nil {
				return errResult("ungültiges Base64: " + err.Error())
			}

			seedStr := args[3].Str

			fileBytes, err := os.ReadFile(inPath)
			if err != nil {
				return errResult("eingabedatei nicht lesbar: " + err.Error())
			}

			var outBytes []byte
			if isBMP(fileBytes) {
				outBytes, err = injectCore(fileBytes, data, seedStr)
			} else {
				outBytes, err = injectImageUniversal(fileBytes, data, seedStr)
			}

			if err != nil {
				return errResult(err.Error())
			}

			// Atomar schreiben
			tmpPath := outPath + ".tmp"
			if err := os.WriteFile(tmpPath, outBytes, 0644); err != nil {
				return errResult("ausgabedatei konnte nicht geschrieben werden: " + err.Error())
			}
			if err := os.Rename(tmpPath, outPath); err != nil {
				os.Remove(tmpPath)
				return errResult("umbenennen fehlgeschlagen: " + err.Error())
			}

			return okResult(seedStr)
		})

	// ---------------------------------------------------------------------------
	// steg.Extract(path, seed)  →  [bool, dataB64, err]
	// ---------------------------------------------------------------------------
	Register(ns+"Extract", "steg", "path, seed",
		"Extrahiert eingebettete Daten aus einem Bild und gibt sie als Base64 zurück.",
		func(args []Value) Value {
			if len(args) < 2 {
				return errResult("Parameter fehlen: steg.Extract(path, seed)")
			}

			path, errVal := absPathVal(args[0].Str)
			if errVal != nil {
				return errResult(fmt.Sprintf("ungültiger Pfad '%s'", args[0].Str))
			}

			seedStr := args[1].Str

			fileBytes, err := os.ReadFile(path)
			if err != nil {
				return errResult("datei nicht lesbar: " + err.Error())
			}

			var data []byte
			if isBMP(fileBytes) {
				data, err = extractCore(fileBytes, seedStr)
			} else {
				data, err = extractImageUniversal(fileBytes, seedStr)
			}

			if err != nil {
				return errResult(err.Error())
			}

			return okResult(base64.StdEncoding.EncodeToString(data))
		})

	// ---------------------------------------------------------------------------
	// steg.GenerateSeed(pass, salt)  →  str hexSeed
	// ---------------------------------------------------------------------------
	Register(ns+"GenerateSeed", "steg", "pass, salt",
		"Erzeugt einen deterministischen Seed aus Passwort und Salt (HMAC-SHA256).",
		func(args []Value) Value {
			if len(args) < 2 {
				return StrVal("")
			}
			h := hmac.New(sha256.New, []byte(args[0].Str))
			h.Write([]byte(args[1].Str))
			return StrVal(hex.EncodeToString(h.Sum(nil)))
		})

	// ---------------------------------------------------------------------------
	// steg.GetCapacity(path, dataLen, seed)  →  [bool, nettoBytes, err, bruttoBytes]
	//
	// Hinweis: dataLen wird als rohe Byte-Länge (nicht Base64) erwartet.
	// ---------------------------------------------------------------------------
	Register(ns+"GetCapacity", "steg", "path, dataLen, seed",
		"Prüft ob ein Bild groß genug für einen Payload ist. dataLen = rohe Byte-Länge.",
		func(args []Value) Value {
			res := make([]Value, 4)

			if len(args) < 3 {
				res[0], res[2] = BoolVal(false), StrVal("Parameter fehlen: steg.GetCapacity(path, dataLen, seed)")
				return Value{Kind: KindArr, Arr: res}
			}

			path, errVal := absPathVal(args[0].Str)
			if errVal != nil {
				res[0], res[2] = BoolVal(false), StrVal(fmt.Sprintf("ungültiger Pfad '%s'", args[0].Str))
				return Value{Kind: KindArr, Arr: res}
			}

			dataLen := int64(args[1].Num)
			seedStr := args[2].Str
			stride := strideFromSeed(seedStr)

			fileBytes, err := os.ReadFile(path)
			if err != nil {
				res[0], res[2] = BoolVal(false), StrVal("datei nicht lesbar: "+err.Error())
				return Value{Kind: KindArr, Arr: res}
			}

			// Kapazität in Pixel/Bytes ermitteln — einheitlich als "Einheiten pro Bit"
			var availableUnits int

			if isBMP(fileBytes) {
				if len(fileBytes) < 14 {
					res[0], res[2] = BoolVal(false), StrVal("ungültiges BMP: zu kurz für Header")
					return Value{Kind: KindArr, Arr: res}
				}
				offset := int(binary.LittleEndian.Uint32(fileBytes[10:14]))
				// BMP: 1 Byte pro Einheit, 1 Bit pro Einheit genutzt → korrekt
				availableUnits = len(fileBytes) - offset
				if availableUnits <= 0 {
					res[0], res[2] = BoolVal(false), StrVal("ungültiges BMP: pixel-offset außerhalb der datei")
					return Value{Kind: KindArr, Arr: res}
				}
			} else {
				img, _, err := decodeImageSafe(fileBytes)
				if err != nil {
					res[0], res[2] = BoolVal(false), StrVal(err.Error())
					return Value{Kind: KindArr, Arr: res}
				}
				b := img.Bounds()
				// PNG: 1 Pixel pro Einheit, 1 Bit (R-LSB) pro Einheit genutzt → korrekt
				availableUnits = b.Dx() * b.Dy()
			}

			// Kapazität berechnen (Header = 8 Bytes)
			maxBits := availableUnits / stride
			maxBytes := int64(maxBits/8) - 8
			if maxBytes < 0 {
				maxBytes = 0
			}

			// Netto mit 35% Sicherheitspuffer
			nettoBytes := (maxBytes * 65) / 100

			res[1] = IntVal(nettoBytes)
			res[3] = IntVal(maxBytes)

			if dataLen > maxBytes {
				res[0], res[2] = BoolVal(false), StrVal(
					fmt.Sprintf("bild zu klein: %d Bytes benötigt, %d verfügbar", dataLen, maxBytes),
				)
			} else {
				res[0], res[2] = BoolVal(true), StrVal("OK")
			}

			return Value{Kind: KindArr, Arr: res}
		})
}
