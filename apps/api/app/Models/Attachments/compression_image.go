package attachments

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image"
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
		if settings.Strength >= 67 {
			level = png.BestCompression
		} else if settings.Strength >= 34 {
			level = png.DefaultCompression
		}
		encoder := png.Encoder{CompressionLevel: level}
		err = encoder.Encode(&output, img)
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
