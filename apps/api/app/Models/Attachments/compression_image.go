package attachments

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"

	golangdraw "golang.org/x/image/draw"
)

const (
	maxCompressionSourceBytes = 64 << 20
	maxCompressionPixels      = 80_000_000
)

var (
	errCompressionUnsupported = errors.New("attachments: compression format unsupported")
	errCompressionTooLarge    = errors.New("attachments: compression input too large")
	errCompressionNotUseful   = errors.New("attachments: compression output not useful")
)

type compressedImage struct {
	Bytes       []byte
	ContentType string
	Extension   string
	Width       int
	Height      int
	SHA256      string
}

func compressAttachmentImage(reader io.Reader, contentType string, settings CompressionSettings, sourceSize int64) (compressedImage, error) {
	if contentType != "image/jpeg" && contentType != "image/png" {
		return compressedImage{}, errCompressionUnsupported
	}
	if sourceSize <= 0 || sourceSize > maxCompressionSourceBytes {
		return compressedImage{}, errCompressionTooLarge
	}
	payload, err := io.ReadAll(io.LimitReader(reader, maxCompressionSourceBytes+1))
	if err != nil {
		return compressedImage{}, err
	}
	if int64(len(payload)) > maxCompressionSourceBytes || int64(len(payload)) != sourceSize {
		return compressedImage{}, errCompressionTooLarge
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(payload))
	if err != nil || config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > maxCompressionPixels {
		return compressedImage{}, errCompressionTooLarge
	}
	img, format, err := decodeAutoOrientedImage(bytes.NewReader(payload))
	if err != nil {
		return compressedImage{}, err
	}
	settings = settings.normalized()
	img = containImage(img, settings.MaxDimension)
	bounds := img.Bounds()
	var output bytes.Buffer
	result := compressedImage{Width: bounds.Dx(), Height: bounds.Dy()}
	switch format {
	case "jpeg":
		result.ContentType = "image/jpeg"
		result.Extension = ".jpg"
		err = jpeg.Encode(&output, img, &jpeg.Options{Quality: settings.JPEGQuality})
	case "png":
		result.ContentType = "image/png"
		result.Extension = ".png"
		level := png.BestSpeed
		if settings.Strength >= 50 {
			level = png.BestCompression
		} else if settings.Strength >= 34 {
			level = png.DefaultCompression
		}
		encoder := png.Encoder{CompressionLevel: level}
		err = encoder.Encode(&output, optimizePNGColorModel(img))
	default:
		return compressedImage{}, errCompressionUnsupported
	}
	if err != nil {
		return compressedImage{}, err
	}
	result.Bytes = output.Bytes()
	minimumSaved := sourceSize * int64(settings.MinSavingsPercent) / 100
	if int64(len(result.Bytes)) >= sourceSize-minimumSaved {
		return compressedImage{}, errCompressionNotUseful
	}
	digest := sha256.Sum256(result.Bytes)
	result.SHA256 = hex.EncodeToString(digest[:])
	return result, nil
}

// optimizePNGColorModel keeps PNG pixels lossless while allowing simple images
// to use indexed color. The standard encoder cannot discover this opportunity
// after decoding a true-color source, even when it contains only a few colors.
func optimizePNGColorModel(src image.Image) image.Image {
	bounds := src.Bounds()
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		return src
	}
	palette := make([]color.Color, 0, 256)
	indices := make(map[[4]uint8]uint8, 256)
	indexed := image.NewPaletted(image.Rect(0, 0, bounds.Dx(), bounds.Dy()), nil)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			nrgba, ok := pngPixel8(src.At(x, y))
			if !ok {
				return src
			}
			key := [4]uint8{nrgba.R, nrgba.G, nrgba.B, nrgba.A}
			index, ok := indices[key]
			if !ok {
				if len(palette) >= 256 {
					return src
				}
				index = uint8(len(palette))
				indices[key] = index
				palette = append(palette, nrgba)
			}
			indexed.SetColorIndex(x-bounds.Min.X, y-bounds.Min.Y, index)
		}
	}
	indexed.Palette = color.Palette(palette)
	return indexed
}

func isRepeated8Bit(value uint16) bool {
	byteValue := uint16(uint8(value >> 8))
	return value == byteValue|byteValue<<8
}

func pngPixel8(pixel color.Color) (color.NRGBA, bool) {
	switch value := pixel.(type) {
	case color.NRGBA:
		return value, true
	case color.NRGBA64:
		if !isRepeated8Bit(value.R) || !isRepeated8Bit(value.G) || !isRepeated8Bit(value.B) || !isRepeated8Bit(value.A) {
			return color.NRGBA{}, false
		}
		return color.NRGBA{R: uint8(value.R >> 8), G: uint8(value.G >> 8), B: uint8(value.B >> 8), A: uint8(value.A >> 8)}, true
	default:
		converted := color.NRGBA64Model.Convert(pixel).(color.NRGBA64)
		if !isRepeated8Bit(converted.R) || !isRepeated8Bit(converted.G) || !isRepeated8Bit(converted.B) || !isRepeated8Bit(converted.A) {
			return color.NRGBA{}, false
		}
		return color.NRGBA{R: uint8(converted.R >> 8), G: uint8(converted.G >> 8), B: uint8(converted.B >> 8), A: uint8(converted.A >> 8)}, true
	}
}

func containImage(src image.Image, maxDimension int) image.Image {
	bounds := src.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if maxDimension <= 0 || (width <= maxDimension && height <= maxDimension) {
		return src
	}
	targetWidth, targetHeight := width, height
	if width >= height {
		targetWidth = maxDimension
		targetHeight = max(1, int(int64(height)*int64(maxDimension)/int64(width)))
	} else {
		targetHeight = maxDimension
		targetWidth = max(1, int(int64(width)*int64(maxDimension)/int64(height)))
	}
	dst := image.NewNRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	golangdraw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, golangdraw.Over, nil)
	return dst
}
