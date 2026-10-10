package main

import (
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/KarpelesLab/gowebp"
	ico "github.com/Kodeworks/golang-image-ico"
	"github.com/disintegration/imaging"
	"github.com/gen2brain/jpegxl"
	"github.com/kbinani/screenshot"
)

// Liefert Argument i als String, oder "" wenn es fehlt.
func argStr(args []Value, i int) string {
	if i < len(args) {
		return ToString(args[i])
	}
	return ""
}

func init() {
	// Registriert WebP global bei Go, damit imaging.Open (und image.Decode
	// generell) .webp-Dateien automatisch erkennt und liest.
	image.RegisterFormat("webp", "RIFF????WEBP", gowebp.Decode, decodeWebpConfig)
}

func decodeWebpConfig(r io.Reader) (image.Config, error) {
	// gowebp bietet kein separates DecodeConfig, daher einmal voll decodieren.
	// Für Massenkonvertierung ggf. später durch echtes Header-Parsing ersetzen,
	// falls Performance bei sehr vielen/großen Dateien relevant wird.
	img, err := gowebp.Decode(r)
	if err != nil {
		return image.Config{}, err
	}
	b := img.Bounds()
	return image.Config{ColorModel: img.ColorModel(), Width: b.Dx(), Height: b.Dy()}, nil
}

func InitPictureFunctions() {
	if builtins == nil {
		builtins = make(map[string]BuiltinInfo)
	}

	ns := "picture."

	Register(ns+"Convert", "picture", "in, out, fmt [, w, h, q]",
		"Konvertiert ein Bild. Formate: jpg, png, webp (auch als Input). Bei ico: w/h/q durch Größen-Liste (16,32) ersetzen.",
		func(args []Value) Value {
			if len(args) < 3 {
				return ErrorVal("usage: picture.Convert(infile, outfile, format [, width, height, quality | icoSizes])")
			}

			inFile, errIn := absPathVal(args[0].Str)
			if errIn != nil {
				return *errIn
			}

			outFile, errOut := absPathVal(args[1].Str)
			if errOut != nil {
				return *errOut
			}

			format := strings.ToLower(args[2].Str)
			src, err := imaging.Open(inFile) // erkennt jetzt auch .webp automatisch
			if err != nil {
				return ErrorVal("Bild-Ladefehler: " + err.Error())
			}

			if format == "ico" {
				if len(args) < 4 || args[3].Str == "" {
					return ErrorVal("Fehlende ICO-Größen (z.B. '16,32,48')")
				}
				return handleIcoExport(src, outFile, args[3].Str)
			}

			width, _ := strconv.Atoi(args[3].Str)
			height, _ := strconv.Atoi(args[4].Str)
			quality := 100
			if len(args) >= 6 {
				if q, err := strconv.Atoi(args[5].Str); err == nil {
					quality = q
				}
			}

			if width > 0 || height > 0 {
				if width > 0 && height > 0 {
					src = imaging.Fill(src, width, height, imaging.Center, imaging.Lanczos)
				} else {
					src = imaging.Resize(src, width, height, imaging.Lanczos)
				}
			}

			var saveErr error
			switch format {
			case "jpg", "jpeg":
				saveErr = imaging.Save(src, outFile, imaging.JPEGQuality(quality))
			case "jxl":
				saveErr = saveJxl(src, outFile, quality)
			case "png":
				saveErr = imaging.Save(src, outFile)
			case "webp":
				saveErr = saveWebp(src, outFile, quality)
			default:
				return ErrorVal("Nicht unterstütztes Format: " + format)
			}

			if saveErr != nil {
				return ErrorVal("Speicherfehler: " + saveErr.Error())
			}
			return NullVal()
		})

	// ---------------- picture.ConvertAll ---------------- (unverändert, profitiert automatisch mit)
	Register(ns+"Convert", "picture", "in, out, fmt [, w, h, q]",
		"Konvertiert ein Bild. Formate: jpg, png, webp, jxl (auch als Input). Bei ico: w/h/q durch Größen-Liste (16,32) ersetzen.",
		func(args []Value) Value {
			if len(args) < 3 {
				return ErrorVal("usage: picture.Convert(infile, outfile, format [, width, height, quality | icoSizes])")
			}

			inFile, errIn := absPathVal(argStr(args, 0))
			if errIn != nil {
				return *errIn
			}

			outFile, errOut := absPathVal(argStr(args, 1))
			if errOut != nil {
				return *errOut
			}

			format := strings.ToLower(argStr(args, 2))
			src, err := imaging.Open(inFile)
			if err != nil {
				return ErrorVal("Bild-Ladefehler: " + err.Error())
			}

			if format == "ico" {
				sizes := argStr(args, 3)
				if sizes == "" {
					return ErrorVal("Fehlende ICO-Größen (z.B. '16,32,48')")
				}
				return handleIcoExport(src, outFile, sizes)
			}

			width, _ := strconv.Atoi(argStr(args, 3))
			height, _ := strconv.Atoi(argStr(args, 4))
			quality := 100
			if q, err := strconv.Atoi(argStr(args, 5)); err == nil {
				quality = q
			}

			if width > 0 || height > 0 {
				if width > 0 && height > 0 {
					src = imaging.Fill(src, width, height, imaging.Center, imaging.Lanczos)
				} else {
					src = imaging.Resize(src, width, height, imaging.Lanczos)
				}
			}

			var saveErr error
			switch format {
			case "jpg", "jpeg":
				saveErr = imaging.Save(src, outFile, imaging.JPEGQuality(quality))
			case "png":
				saveErr = imaging.Save(src, outFile)
			case "webp":
				saveErr = saveWebp(src, outFile, quality)
			case "jxl":
				saveErr = saveJxl(src, outFile, quality)
			default:
				return ErrorVal("Nicht unterstütztes Format: " + format)
			}

			if saveErr != nil {
				return ErrorVal("Speicherfehler: " + saveErr.Error())
			}
			return NullVal()
		})

	Register(ns+"Snapshot", "picture", "outFile [, monitorIndex]",
		"Erstellt einen Screenshot eines Monitors.",
		func(args []Value) Value {
			if len(args) < 1 {
				return ErrorVal("usage: picture.Snapshot(outFile, [idx])")
			}
			outFile, errVal := absPathVal(args[0].Str)
			if errVal != nil {
				return *errVal
			}
			idx := 0
			if len(args) >= 2 {
				idx = int(args[1].Num)
			}
			if idx >= screenshot.NumActiveDisplays() {
				return ErrorVal("Monitor nicht gefunden")
			}
			bounds := screenshot.GetDisplayBounds(idx)
			img, err := screenshot.CaptureRect(bounds)
			if err != nil {
				return ErrorVal("Screenshot fehlgeschlagen: " + err.Error())
			}
			if err := imaging.Save(img, outFile); err != nil {
				return ErrorVal("Fehler beim Speichern des Screenshots: " + err.Error())
			}
			return NullVal()
		})
}

func saveWebp(img image.Image, outFile string, quality int) error {
	f, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer f.Close()

	if quality <= 0 || quality >= 100 {
		return gowebp.Encode(f, img, nil) // VP8L, verlustfrei
	}
	return gowebp.Encode(f, img, &gowebp.Options{
		Lossy:   true,
		Quality: float32(quality),
		Method:  4,
	})
}

func saveJxl(img image.Image, outFile string, quality int) error {
	f, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer f.Close()

	return jpegxl.Encode(f, img, jpegxl.Options{
		Quality: quality,
		Effort:  jpegxl.DefaultEffort,
	})
}

func handleIcoExport(src image.Image, outFile, sizesStr string) Value {
	base := strings.TrimSuffix(outFile, filepath.Ext(outFile))
	failed := 0
	for _, s := range strings.Split(sizesStr, ",") {
		size, _ := strconv.Atoi(strings.TrimSpace(s))
		if size <= 0 || size > 256 {
			continue
		}
		resized := imaging.Resize(src, size, size, imaging.Lanczos)
		f, err := os.Create(fmt.Sprintf("%s_%d.ico", base, size))
		if err == nil {
			ico.Encode(f, resized)
			f.Close()
		} else {
			failed++
		}
	}
	if failed > 0 {
		return ErrorVal("Einige ICOs fehlgeschlagen")
	}
	return NullVal()
}
