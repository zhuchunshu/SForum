package main

import (
	"bytes"
	"image"
	"image/color"
	"testing"

	"golang.org/x/image/tiff"
)

func TestProcessImageResizesSupportedPNG(t *testing.T) {
	payload, err := samplePNG(640, 480, color.NRGBA{R: 20, G: 40, B: 60, A: 255})
	if err != nil {
		t.Fatalf("build png: %v", err)
	}
	variant, metadata, err := processImage(payload, processOptions{MaxWidth: 640, MaxHeight: 480})
	if err != nil {
		t.Fatalf("process png: %v", err)
	}
	if metadata.Format != "png" || variant.Width != 320 || variant.Height != 240 || variant.Bytes == 0 {
		t.Fatalf("unexpected image result: metadata=%#v variant=%#v", metadata, variant)
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
