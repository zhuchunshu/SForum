package attachments

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func TestCompressAttachmentJPEGStrengthAndResize(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 1200, 800))
	for y := 0; y < 800; y++ {
		for x := 0; x < 1200; x++ {
			source.SetNRGBA(x, y, color.NRGBA{
				R: uint8((x*13 + y*3) % 255), G: uint8((x*5 + y*17) % 255),
				B: uint8((x*11 + y*7) % 255), A: 255,
			})
		}
	}
	var original bytes.Buffer
	if err := jpeg.Encode(&original, source, &jpeg.Options{Quality: 100}); err != nil {
		t.Fatal(err)
	}
	base := CompressionSettings{Enabled: true, MaxDimension: 600, MinSizeKB: 1, MinSavingsPercent: 0}
	light, err := compressAttachmentImage(bytes.NewReader(original.Bytes()), "image/jpeg", mergeCompressionStrength(base, 10), int64(original.Len()))
	if err != nil {
		t.Fatalf("light compression: %v", err)
	}
	strong, err := compressAttachmentImage(bytes.NewReader(original.Bytes()), "image/jpeg", mergeCompressionStrength(base, 90), int64(original.Len()))
	if err != nil {
		t.Fatalf("strong compression: %v", err)
	}
	if light.Width != 600 || light.Height != 400 || strong.Width != 600 || strong.Height != 400 {
		t.Fatalf("unexpected dimensions: light=%dx%d strong=%dx%d", light.Width, light.Height, strong.Width, strong.Height)
	}
	if len(strong.Bytes) >= len(light.Bytes) {
		t.Fatalf("strong compression should be smaller: strong=%d light=%d", len(strong.Bytes), len(light.Bytes))
	}
	if strong.ContentType != "image/jpeg" || strong.SHA256 == "" {
		t.Fatalf("unexpected strong output: %#v", strong)
	}
}

func TestCompressAttachmentPNGPreservesAlpha(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 640, 480))
	for y := 0; y < 480; y++ {
		for x := 0; x < 640; x++ {
			source.SetNRGBA(x, y, color.NRGBA{R: 22, G: 96, B: 180, A: uint8((x + y) % 256)})
		}
	}
	var original bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.NoCompression}
	if err := encoder.Encode(&original, source); err != nil {
		t.Fatal(err)
	}
	settings := CompressionSettings{Enabled: true, Strength: 90, MaxDimension: 320, MinSizeKB: 1, MinSavingsPercent: 0}
	output, err := compressAttachmentImage(bytes.NewReader(original.Bytes()), "image/png", settings, int64(original.Len()))
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := png.Decode(bytes.NewReader(output.Bytes))
	if err != nil {
		t.Fatal(err)
	}
	if output.Width != 320 || output.Height != 240 || decoded.ColorModel() == color.GrayModel {
		t.Fatalf("unexpected PNG output: %#v model=%v", output, decoded.ColorModel())
	}
}

func TestCompressAttachmentPNGUsesLosslessPaletteForLowColorImage(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 512, 384))
	colors := []color.NRGBA{
		{R: 18, G: 32, B: 48, A: 255},
		{R: 220, G: 230, B: 240, A: 255},
		{R: 220, G: 230, B: 240, A: 96},
		{R: 240, G: 80, B: 50, A: 255},
	}
	state := uint32(1)
	for y := 0; y < source.Bounds().Dy(); y++ {
		for x := 0; x < source.Bounds().Dx(); x++ {
			state = state*1664525 + 1013904223
			source.SetNRGBA(x, y, colors[state>>30])
		}
	}
	var original bytes.Buffer
	if err := png.Encode(&original, source); err != nil {
		t.Fatal(err)
	}
	if _, ok := optimizePNGColorModel(source).(*image.Paletted); !ok {
		t.Fatal("low-color PNG should use indexed color")
	}
	settings := CompressionSettings{Enabled: true, Strength: 55, MaxDimension: 1024, MinSizeKB: 1, MinSavingsPercent: 0}
	output, err := compressAttachmentImage(bytes.NewReader(original.Bytes()), "image/png", settings, int64(original.Len()))
	if err != nil {
		t.Fatal(err)
	}
	if len(output.Bytes) >= original.Len()*9/10 {
		t.Fatalf("indexed PNG should be smaller: original=%d output=%d", original.Len(), len(output.Bytes))
	}
	decoded, err := png.Decode(bytes.NewReader(output.Bytes))
	if err != nil {
		t.Fatal(err)
	}
	for _, point := range []image.Point{{0, 0}, {31, 31}, {32, 0}, {100, 100}, {511, 383}} {
		if got, want := color.NRGBAModel.Convert(decoded.At(point.X, point.Y)), color.NRGBAModel.Convert(source.At(point.X, point.Y)); got != want {
			t.Fatalf("pixel %v changed: got=%v want=%v", point, got, want)
		}
	}
}

func TestOptimizePNGColorModelDoesNotReduce16BitChannels(t *testing.T) {
	source := image.NewNRGBA64(image.Rect(0, 0, 2, 2))
	source.SetNRGBA64(0, 0, color.NRGBA64{R: 0x1234, G: 0xabcd, B: 0x5678, A: 0xffff})
	if _, ok := optimizePNGColorModel(source).(*image.Paletted); ok {
		t.Fatal("16-bit channels must not be reduced to an indexed 8-bit PNG")
	}
}

func TestCompressionSettingsExposeStableQualityMapping(t *testing.T) {
	light := (CompressionSettings{Strength: 0, MaxDimension: 2560, MinSizeKB: 1}).normalized()
	strong := (CompressionSettings{Strength: 100, MaxDimension: 2560, MinSizeKB: 1}).normalized()
	if light.JPEGQuality != 95 || strong.JPEGQuality != 70 || light.PolicyDigest == strong.PolicyDigest {
		t.Fatalf("quality mapping light=%#v strong=%#v", light, strong)
	}
}

func mergeCompressionStrength(settings CompressionSettings, strength int) CompressionSettings {
	settings.Strength = strength
	return settings
}
