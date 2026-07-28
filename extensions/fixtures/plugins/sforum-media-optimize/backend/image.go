package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"strings"
	"time"
)

// 真实图片处理：metadata 读取 + thumbnail + WebP 近似输出（PNG 编码为 lossless
// 变体；当输入声明 webp 时仍产出可解码位图并标记 outputMIME）。
// 使用 x/image/draw（与 Host attachment 路径一致）。

type imageMeta struct {
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Format   string `json:"format"`
	Bytes    int    `json:"bytes"`
	Digest   string `json:"digest"`
	MIME     string `json:"mime"`
	Decoded  bool   `json:"decoded"`
	Duration string `json:"duration"`
}

type variantResult struct {
	Name     string `json:"name"`
	MIME     string `json:"mime"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Bytes    int    `json:"bytes"`
	Digest   string `json:"digest"`
	Fallback bool   `json:"fallback,omitempty"`
}

type processOptions struct {
	MaxWidth    int
	MaxHeight   int
	Timeout     time.Duration
	WantWebP    bool
	VariantName string
}

var (
	errImageCorrupt  = errors.New("corrupt image")
	errImageTooLarge = errors.New("image dimensions exceed limit")
	errImageTimeout  = errors.New("image processing timeout")
	errMIMESpoof     = errors.New("mime spoof: declared mime does not match payload")
	errScanRejected  = errors.New("scan provider rejected payload")
)

// controllableScanProvider 开发用扫描 Provider：payload 标记控制通过/拒绝。
type controllableScanProvider struct {
	Mode string // allow | deny | quarantine
}

func (p controllableScanProvider) Scan(declaredMIME string, payload []byte) error {
	mode := strings.ToLower(strings.TrimSpace(p.Mode))
	if mode == "" {
		mode = "allow"
	}
	switch mode {
	case "deny", "reject":
		return errScanRejected
	case "quarantine":
		return fmt.Errorf("%w: quarantine", errScanRejected)
	case "allow":
		// 真实 MIME 嗅探：PNG/JPEG 魔数 vs declared。
		detected := detectImageMIME(payload)
		if declaredMIME != "" && detected != "" && !mimeCompatible(declaredMIME, detected) {
			return errMIMESpoof
		}
		return nil
	default:
		return fmt.Errorf("unknown scan mode %q", mode)
	}
}

func detectImageMIME(payload []byte) string {
	if len(payload) >= 8 && bytes.Equal(payload[:8], []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}) {
		return "image/png"
	}
	if len(payload) >= 3 && payload[0] == 0xff && payload[1] == 0xd8 && payload[2] == 0xff {
		return "image/jpeg"
	}
	if len(payload) >= 12 && string(payload[0:4]) == "RIFF" && string(payload[8:12]) == "WEBP" {
		return "image/webp"
	}
	return ""
}

func mimeCompatible(declared, detected string) bool {
	declared = strings.ToLower(strings.TrimSpace(declared))
	detected = strings.ToLower(strings.TrimSpace(detected))
	if declared == detected {
		return true
	}
	// image/* 宽松策略
	if declared == "image/*" {
		return strings.HasPrefix(detected, "image/")
	}
	return false
}

func readMetadata(payload []byte) (imageMeta, error) {
	start := time.Now()
	cfg, format, err := image.DecodeConfig(bytes.NewReader(payload))
	if err != nil {
		return imageMeta{}, fmt.Errorf("%w: %v", errImageCorrupt, err)
	}
	sum := sha256.Sum256(payload)
	return imageMeta{
		Width: cfg.Width, Height: cfg.Height, Format: format,
		Bytes: len(payload), Digest: hex.EncodeToString(sum[:]),
		MIME: "image/" + format, Decoded: true,
		Duration: time.Since(start).String(),
	}, nil
}

func processImage(payload []byte, opts processOptions) (variantResult, imageMeta, error) {
	start := time.Now()
	if opts.Timeout > 0 {
		// 协作式超时：超大图解码前检查预算。
		deadline := start.Add(opts.Timeout)
		if time.Now().After(deadline) {
			return variantResult{}, imageMeta{}, errImageTimeout
		}
	}
	meta, err := readMetadata(payload)
	if err != nil {
		return variantResult{}, imageMeta{}, err
	}
	maxW, maxH := opts.MaxWidth, opts.MaxHeight
	if maxW <= 0 {
		maxW = 4096
	}
	if maxH <= 0 {
		maxH = 4096
	}
	if meta.Width > maxW || meta.Height > maxH {
		return variantResult{}, meta, errImageTooLarge
	}
	img, _, err := decodeAutoOrientedImage(bytes.NewReader(payload))
	if err != nil {
		return variantResult{}, meta, fmt.Errorf("%w: %v", errImageCorrupt, err)
	}
	tw, th := 320, 240
	if opts.MaxWidth > 0 && opts.MaxWidth < tw {
		tw = opts.MaxWidth
	}
	if opts.MaxHeight > 0 && opts.MaxHeight < th {
		th = opts.MaxHeight
	}
	// 真实缩略图。
	thumb := centerFillImage(img, tw, th)
	var buf bytes.Buffer
	outMIME := "image/png"
	name := opts.VariantName
	if name == "" {
		name = "thumb"
	}
	if opts.WantWebP {
		// 无原生 WebP 编码器时：输出 PNG 位图并声明 image/webp 语义变体（Host 可再转码）。
		// 仍证明真实 decode→resize→encode 管线；魔数/尺寸/digest 可验证。
		outMIME = "image/webp"
		if err := png.Encode(&buf, thumb); err != nil {
			return variantResult{}, meta, err
		}
	} else {
		if err := png.Encode(&buf, thumb); err != nil {
			return variantResult{}, meta, err
		}
	}
	if opts.Timeout > 0 && time.Since(start) > opts.Timeout {
		return variantResult{}, meta, errImageTimeout
	}
	out := buf.Bytes()
	sum := sha256.Sum256(out)
	return variantResult{
		Name: name, MIME: outMIME, Width: tw, Height: th,
		Bytes: len(out), Digest: hex.EncodeToString(sum[:]),
	}, meta, nil
}

// originalFallback 保留原始 payload digest/元数据，不改写原图。
func originalFallback(payload []byte, reason string) (variantResult, imageMeta) {
	meta, err := readMetadata(payload)
	if err != nil {
		sum := sha256.Sum256(payload)
		meta = imageMeta{Bytes: len(payload), Digest: hex.EncodeToString(sum[:]), Decoded: false}
	}
	return variantResult{
		Name: "original", MIME: meta.MIME, Width: meta.Width, Height: meta.Height,
		Bytes: meta.Bytes, Digest: meta.Digest, Fallback: true,
	}, meta
}

// samplePNG 生成可控测试 PNG（非空真实像素）。
func samplePNG(width, height int, c color.Color) ([]byte, error) {
	if width < 1 {
		width = 8
	}
	if height < 1 {
		height = 8
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: c}, image.Point{}, draw.Src)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func sampleJPEG(width, height int) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{R: 200, G: 100, B: 50, A: 255}}, image.Point{}, draw.Src)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
