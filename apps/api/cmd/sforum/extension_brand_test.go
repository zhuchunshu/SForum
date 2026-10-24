package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const testBrandSVG = `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512" viewBox="0 0 512 512"><rect width="512" height="512" fill="#0b1220"/></svg>`

func writeTestBrandPNG(t *testing.T, path string, width, height int, noCompression bool) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 0x40, A: 0xff})
		}
	}
	var buf bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.DefaultCompression}
	if noCompression {
		encoder.CompressionLevel = png.NoCompression
	}
	if err := encoder.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("write png: %v", err)
	}
}

// writeTestBrandPackage 造一个合规包，返回包目录；调用方按需破坏其中某个文件。
func writeTestBrandPackage(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	brandDir := filepath.Join(dir, "brand")
	if err := os.MkdirAll(brandDir, 0o755); err != nil {
		t.Fatalf("mkdir brand: %v", err)
	}
	if err := os.WriteFile(filepath.Join(brandDir, "icon.svg"), []byte(testBrandSVG), 0o644); err != nil {
		t.Fatalf("write icon.svg: %v", err)
	}
	writeTestBrandPNG(t, filepath.Join(brandDir, "icon.png"), brandIconSide, brandIconSide, false)
	writeTestBrandPNG(t, filepath.Join(brandDir, "social-preview.png"), brandSocialWidth, brandSocialHeight, false)
	return dir
}

func brandCodes(findings []brandFinding) []string {
	codes := make([]string, 0, len(findings))
	for _, finding := range findings {
		codes = append(codes, finding.Code)
	}
	return codes
}

func TestCheckBrandAssets(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(t *testing.T, dir string)
		want   string
	}{
		{
			name:   "compliant package",
			mutate: func(t *testing.T, dir string) {},
		},
		{
			name: "missing svg",
			mutate: func(t *testing.T, dir string) {
				os.Remove(filepath.Join(dir, "brand", "icon.svg"))
			},
			want: "brand.missing_icon",
		},
		{
			name: "missing icon png",
			mutate: func(t *testing.T, dir string) {
				os.Remove(filepath.Join(dir, "brand", "icon.png"))
			},
			want: "brand.missing_icon_png",
		},
		{
			name: "missing social preview",
			mutate: func(t *testing.T, dir string) {
				os.Remove(filepath.Join(dir, "brand", "social-preview.png"))
			},
			want: "brand.missing_social",
		},
		{
			name: "icon png wrong size",
			mutate: func(t *testing.T, dir string) {
				writeTestBrandPNG(t, filepath.Join(dir, "brand", "icon.png"), 256, 256, false)
			},
			want: "brand.icon_dimensions",
		},
		{
			name: "social preview wrong size",
			mutate: func(t *testing.T, dir string) {
				writeTestBrandPNG(t, filepath.Join(dir, "brand", "social-preview.png"), 1200, 630, false)
			},
			want: "brand.social_dimensions",
		},
		{
			name: "icon png over budget",
			mutate: func(t *testing.T, dir string) {
				writeTestBrandPNG(t, filepath.Join(dir, "brand", "icon.png"), brandIconSide, brandIconSide, true)
			},
			want: "brand.too_large",
		},
		{
			name: "svg is not xml",
			mutate: func(t *testing.T, dir string) {
				os.WriteFile(filepath.Join(dir, "brand", "icon.svg"), []byte("not xml"), 0o644)
			},
			want: "brand.svg_invalid",
		},
		{
			name: "svg wrong viewBox",
			mutate: func(t *testing.T, dir string) {
				os.WriteFile(filepath.Join(dir, "brand", "icon.svg"),
					[]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 256 256"></svg>`), 0o644)
			},
			want: "brand.svg_invalid",
		},
		{
			name: "svg references an external image",
			mutate: func(t *testing.T, dir string) {
				os.WriteFile(filepath.Join(dir, "brand", "icon.svg"),
					[]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512"><image href="https://example.com/a.png"/></svg>`), 0o644)
			},
			want: "brand.svg_external_ref",
		},
		{
			name: "png is actually text",
			mutate: func(t *testing.T, dir string) {
				os.WriteFile(filepath.Join(dir, "brand", "social-preview.png"), []byte("nope"), 0o644)
			},
			want: "brand.not_png",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := writeTestBrandPackage(t)
			tc.mutate(t, dir)
			codes := brandCodes(checkBrandAssets(dir))
			if tc.want == "" {
				if len(codes) != 0 {
					t.Fatalf("expected no findings, got %v", codes)
				}
				return
			}
			found := false
			for _, code := range codes {
				if code == tc.want {
					found = true
				}
			}
			if !found {
				t.Fatalf("expected finding %q, got %v", tc.want, codes)
			}
		})
	}
}

func TestBrandPackageDirsScansCollections(t *testing.T) {
	root := t.TempDir()
	for _, collection := range []string{"plugins", "themes"} {
		pkg := filepath.Join(root, collection, collection+"-one")
		if err := os.MkdirAll(pkg, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(pkg, "sforum.extension.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("write manifest: %v", err)
		}
	}
	// 没有清单的目录不是扩展包，必须被跳过。
	if err := os.MkdirAll(filepath.Join(root, "plugins", "notes"), 0o755); err != nil {
		t.Fatalf("mkdir notes: %v", err)
	}

	dirs, err := brandPackageDirs(nil, []string{root})
	if err != nil {
		t.Fatalf("brandPackageDirs: %v", err)
	}
	if len(dirs) != 2 {
		t.Fatalf("expected 2 packages, got %d: %v", len(dirs), dirs)
	}
	if !strings.HasSuffix(dirs[0], "plugins-one") || !strings.HasSuffix(dirs[1], "themes-one") {
		t.Fatalf("unexpected order: %v", dirs)
	}
}

// TestShippedPackagesCarryBrandAssets 把契约钉在仓库上：内置与可选包缺品牌资产
// 时 api 线直接红，而不是等发版前才想起来。
func TestShippedPackagesCarryBrandAssets(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "..", ".."))

	for _, root := range []string{
		filepath.Join(repoRoot, "extensions", "builtin"),
		filepath.Join(repoRoot, "extensions", "optional"),
	} {
		if _, err := os.Stat(root); err != nil {
			continue
		}
		dirs, err := brandPackageDirs(nil, []string{root})
		if err != nil {
			t.Fatalf("scan %s: %v", root, err)
		}
		if len(dirs) == 0 {
			continue
		}
		for _, dir := range dirs {
			if findings := checkBrandAssets(dir); len(findings) > 0 {
				t.Errorf("%s 缺少合规品牌资产: %v", dir, brandCodes(findings))
			}
		}
	}
}
