package attachments

import (
	"fmt"
	"image"
	"io"

	"github.com/rwcarlsen/goexif/exif"
	golangdraw "golang.org/x/image/draw"
)

const maxEXIFScanBytes = 1 << 20

func decodeAutoOrientedImage(reader io.ReadSeeker) (image.Image, string, error) {
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return nil, "", err
	}
	img, format, err := image.Decode(reader)
	if err != nil {
		return nil, "", err
	}
	if format != "jpeg" && format != "png" {
		return nil, "", fmt.Errorf("unsupported image format %q", format)
	}
	if format == "jpeg" {
		img = applyEXIFOrientation(img, readEXIFOrientation(reader))
	}
	return img, format, nil
}

func readEXIFOrientation(reader io.ReadSeeker) (orientation int) {
	orientation = 1
	// EXIF metadata is untrusted. A malformed optional tag must not crash upload.
	defer func() {
		if recover() != nil {
			orientation = 1
		}
	}()

	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return orientation
	}
	metadata, err := exif.Decode(io.LimitReader(reader, maxEXIFScanBytes))
	if err != nil || metadata == nil {
		return orientation
	}
	tag, err := metadata.Get(exif.Orientation)
	if err != nil || tag == nil || tag.Count != 1 {
		return orientation
	}
	value, err := tag.Int(0)
	if err != nil || value < 1 || value > 8 {
		return orientation
	}
	return value
}

func applyEXIFOrientation(src image.Image, orientation int) image.Image {
	if orientation <= 1 || orientation > 8 {
		return src
	}
	bounds := src.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	dstWidth, dstHeight := width, height
	if orientation >= 5 {
		dstWidth, dstHeight = height, width
	}
	dst := image.NewNRGBA(image.Rect(0, 0, dstWidth, dstHeight))
	for y := 0; y < dstHeight; y++ {
		for x := 0; x < dstWidth; x++ {
			sourceX, sourceY := orientedSourcePoint(x, y, width, height, orientation)
			dst.Set(x, y, src.At(bounds.Min.X+sourceX, bounds.Min.Y+sourceY))
		}
	}
	return dst
}

func orientedSourcePoint(x int, y int, width int, height int, orientation int) (int, int) {
	switch orientation {
	case 2:
		return width - 1 - x, y
	case 3:
		return width - 1 - x, height - 1 - y
	case 4:
		return x, height - 1 - y
	case 5:
		return y, x
	case 6:
		return y, height - 1 - x
	case 7:
		return width - 1 - y, height - 1 - x
	case 8:
		return width - 1 - y, x
	default:
		return x, y
	}
}

func centerFillImage(src image.Image, targetWidth int, targetHeight int) image.Image {
	bounds := src.Bounds()
	sourceWidth, sourceHeight := bounds.Dx(), bounds.Dy()
	if sourceWidth <= 0 || sourceHeight <= 0 || targetWidth <= 0 || targetHeight <= 0 {
		return image.NewNRGBA(image.Rect(0, 0, 0, 0))
	}

	crop := bounds
	if int64(sourceWidth)*int64(targetHeight) > int64(sourceHeight)*int64(targetWidth) {
		cropWidth := max(1, int(int64(sourceHeight)*int64(targetWidth)/int64(targetHeight)))
		crop.Min.X += (sourceWidth - cropWidth) / 2
		crop.Max.X = crop.Min.X + cropWidth
	} else {
		cropHeight := max(1, int(int64(sourceWidth)*int64(targetHeight)/int64(targetWidth)))
		crop.Min.Y += (sourceHeight - cropHeight) / 2
		crop.Max.Y = crop.Min.Y + cropHeight
	}

	dst := image.NewNRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	golangdraw.CatmullRom.Scale(dst, dst.Bounds(), src, crop, golangdraw.Over, nil)
	return dst
}
