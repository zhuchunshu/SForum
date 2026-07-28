package attachments

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"strconv"
	"testing"

	"golang.org/x/image/tiff"
)

func TestApplyEXIFOrientation(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	for index := 0; index < 6; index++ {
		src.SetNRGBA(index%3, index/3, color.NRGBA{R: uint8(index + 1), A: 255})
	}

	tests := []struct {
		orientation int
		width       int
		height      int
		pixels      []uint8
	}{
		{orientation: 1, width: 3, height: 2, pixels: []uint8{1, 2, 3, 4, 5, 6}},
		{orientation: 2, width: 3, height: 2, pixels: []uint8{3, 2, 1, 6, 5, 4}},
		{orientation: 3, width: 3, height: 2, pixels: []uint8{6, 5, 4, 3, 2, 1}},
		{orientation: 4, width: 3, height: 2, pixels: []uint8{4, 5, 6, 1, 2, 3}},
		{orientation: 5, width: 2, height: 3, pixels: []uint8{1, 4, 2, 5, 3, 6}},
		{orientation: 6, width: 2, height: 3, pixels: []uint8{4, 1, 5, 2, 6, 3}},
		{orientation: 7, width: 2, height: 3, pixels: []uint8{6, 3, 5, 2, 4, 1}},
		{orientation: 8, width: 2, height: 3, pixels: []uint8{3, 6, 2, 5, 1, 4}},
	}

	for _, tt := range tests {
		t.Run(strconv.Itoa(tt.orientation), func(t *testing.T) {
			result := applyEXIFOrientation(src, tt.orientation)
			if result.Bounds().Dx() != tt.width || result.Bounds().Dy() != tt.height {
				t.Fatalf("orientation %d bounds=%v", tt.orientation, result.Bounds())
			}
			for index, expected := range tt.pixels {
				x, y := index%tt.width, index/tt.width
				actual := color.NRGBAModel.Convert(result.At(x, y)).(color.NRGBA).R
				if actual != expected {
					t.Fatalf("orientation %d pixel (%d,%d)=%d, want %d", tt.orientation, x, y, actual, expected)
				}
			}
		})
	}
}

func TestDecodeAutoOrientedImageAppliesJPEGEXIF(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, src, nil); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	withEXIF := addJPEGEXIFOrientation(t, encoded.Bytes(), 6)

	result, format, err := decodeAutoOrientedImage(bytes.NewReader(withEXIF))
	if err != nil {
		t.Fatalf("decode oriented jpeg: %v", err)
	}
	if format != "jpeg" || result.Bounds().Dx() != 2 || result.Bounds().Dy() != 3 {
		t.Fatalf("oriented jpeg format=%q bounds=%v", format, result.Bounds())
	}
}

func TestDecodeAutoOrientedImageRejectsTIFF(t *testing.T) {
	var encoded bytes.Buffer
	if err := tiff.Encode(&encoded, image.NewNRGBA(image.Rect(0, 0, 2, 2)), nil); err != nil {
		t.Fatalf("encode tiff: %v", err)
	}
	if _, _, err := decodeAutoOrientedImage(bytes.NewReader(encoded.Bytes())); err == nil {
		t.Fatal("expected TIFF to be rejected")
	}
}

func TestCenterFillImageUsesRequestedDimensions(t *testing.T) {
	result := centerFillImage(image.NewNRGBA(image.Rect(0, 0, 320, 240)), 128, 128)
	if result.Bounds() != image.Rect(0, 0, 128, 128) {
		t.Fatalf("center fill bounds=%v", result.Bounds())
	}
}

func addJPEGEXIFOrientation(t *testing.T, jpegBytes []byte, orientation uint16) []byte {
	t.Helper()
	if len(jpegBytes) < 2 || jpegBytes[0] != 0xff || jpegBytes[1] != 0xd8 {
		t.Fatal("invalid test jpeg")
	}

	tiffHeader := make([]byte, 26)
	copy(tiffHeader[0:2], "II")
	binary.LittleEndian.PutUint16(tiffHeader[2:4], 42)
	binary.LittleEndian.PutUint32(tiffHeader[4:8], 8)
	binary.LittleEndian.PutUint16(tiffHeader[8:10], 1)
	binary.LittleEndian.PutUint16(tiffHeader[10:12], 0x0112)
	binary.LittleEndian.PutUint16(tiffHeader[12:14], 3)
	binary.LittleEndian.PutUint32(tiffHeader[14:18], 1)
	binary.LittleEndian.PutUint16(tiffHeader[18:20], orientation)

	payload := append([]byte("Exif\x00\x00"), tiffHeader...)
	segment := make([]byte, 4+len(payload))
	segment[0], segment[1] = 0xff, 0xe1
	binary.BigEndian.PutUint16(segment[2:4], uint16(len(payload)+2))
	copy(segment[4:], payload)

	result := make([]byte, 0, len(jpegBytes)+len(segment))
	result = append(result, jpegBytes[:2]...)
	result = append(result, segment...)
	return append(result, jpegBytes[2:]...)
}
