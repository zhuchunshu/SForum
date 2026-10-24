package main

import (
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// 品牌资产契约（docs/extensions/brand-assets.md）：每个随产品分发的扩展包都带
// brand/icon.svg、brand/icon.png、brand/social-preview.png。这些文件不进
// packageFiles —— 品牌不是运行时制品，但它是包的一部分，所以随包树摘要变化。
const (
	brandDirName      = "brand"
	brandIconSVGName  = "icon.svg"
	brandIconPNGName  = "icon.png"
	brandSocialName   = "social-preview.png"
	brandIconSide     = 512
	brandSocialWidth  = 1280
	brandSocialHeight = 640
	brandIconSVGMax   = 40 << 10
	brandIconPNGMax   = 120 << 10
	brandSocialMax    = 600 << 10
)

// brandFinding 是一条可定位的品牌资产问题。Code 稳定，便于与文档和 CI 输出对齐。
type brandFinding struct {
	Package string
	Code    string
	Detail  string
}

func (f brandFinding) String() string {
	return fmt.Sprintf("%s: %s: %s", f.Package, f.Code, f.Detail)
}

func newExtensionBrandCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "brand",
		Short: "Brand asset helpers (icon and social preview)",
	}
	cmd.AddCommand(newExtensionBrandCheckCommand())
	return cmd
}

func newExtensionBrandCheckCommand() *cobra.Command {
	var roots []string
	cmd := &cobra.Command{
		Use:   "check [package...]",
		Short: "Verify brand/icon.svg, brand/icon.png and brand/social-preview.png",
		Long: `Verify the brand assets of one or more extension packages.

Every package that ships with the product carries three files:

  brand/icon.svg          viewBox 0 0 512 512, self-contained, at most 40 KiB
  brand/icon.png          512x512 RGBA, at most 120 KiB
  brand/social-preview.png 1280x640, at most 600 KiB

Use --root to scan an extension collection (a directory holding plugins/ and
themes/) instead of listing packages by hand.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			dirs, err := brandPackageDirs(args, roots)
			if err != nil {
				return err
			}
			if len(dirs) == 0 {
				return fmt.Errorf("brand: 没有可检查的扩展包（给包路径，或用 --root 指向扩展集合根）")
			}

			findings := 0
			for _, dir := range dirs {
				for _, finding := range checkBrandAssets(dir) {
					findings++
					cmd.PrintErrln(finding.String())
				}
			}
			if findings > 0 {
				return fmt.Errorf("brand: %d 个问题，%d 个包未通过", findings, len(dirs))
			}
			cmd.Printf("brand: %d 个包通过\n", len(dirs))
			return nil
		},
	}
	cmd.Flags().StringSliceVar(&roots, "root", nil, "扩展集合根（含 plugins/ 与 themes/），可重复")
	return cmd
}

// brandPackageDirs 把显式包路径与集合根合并成去重、排序后的包目录列表。
func brandPackageDirs(packages, roots []string) ([]string, error) {
	seen := make(map[string]struct{})
	dirs := make([]string, 0, len(packages))

	add := func(dir string) {
		abs, err := filepath.Abs(dir)
		if err != nil {
			return
		}
		if _, ok := seen[abs]; ok {
			return
		}
		seen[abs] = struct{}{}
		dirs = append(dirs, abs)
	}

	for _, pkg := range packages {
		info, err := os.Stat(pkg)
		if err != nil || !info.IsDir() {
			return nil, fmt.Errorf("brand: %s 不是目录", pkg)
		}
		add(pkg)
	}
	for _, root := range roots {
		for _, collection := range []string{"plugins", "themes"} {
			entries, err := os.ReadDir(filepath.Join(root, collection))
			if err != nil {
				continue
			}
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				dir := filepath.Join(root, collection, entry.Name())
				// 只认真正的扩展包：有清单才算，避免把说明目录当成包。
				if _, err := os.Stat(filepath.Join(dir, "sforum.extension.json")); err != nil {
					continue
				}
				add(dir)
			}
		}
	}
	sort.Strings(dirs)
	return dirs, nil
}

func checkBrandAssets(packageDir string) []brandFinding {
	brandDir := filepath.Join(packageDir, brandDirName)
	findings := make([]brandFinding, 0, 3)

	findings = append(findings, checkBrandSVG(packageDir, filepath.Join(brandDir, brandIconSVGName))...)
	findings = append(findings, checkBrandPNG(packageDir, filepath.Join(brandDir, brandIconPNGName),
		"brand.missing_icon_png", "brand.icon_dimensions", brandIconSide, brandIconSide, brandIconPNGMax)...)
	findings = append(findings, checkBrandPNG(packageDir, filepath.Join(brandDir, brandSocialName),
		"brand.missing_social", "brand.social_dimensions", brandSocialWidth, brandSocialHeight, brandSocialMax)...)

	return findings
}

func checkBrandSVG(packageDir, path string) []brandFinding {
	missing := brandFinding{Package: packageDir, Code: "brand.missing_icon", Detail: "brand/icon.svg 不存在"}
	body, err := os.ReadFile(path)
	if err != nil {
		return []brandFinding{missing}
	}
	if len(body) > brandIconSVGMax {
		return []brandFinding{{Package: packageDir, Code: "brand.too_large",
			Detail: fmt.Sprintf("brand/icon.svg 为 %d 字节，上限 %d", len(body), brandIconSVGMax)}}
	}

	decoder := xml.NewDecoder(bytes.NewReader(body))
	sawRoot := false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return []brandFinding{{Package: packageDir, Code: "brand.svg_invalid", Detail: err.Error()}}
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		if !sawRoot {
			sawRoot = true
			if start.Name.Local != "svg" {
				return []brandFinding{{Package: packageDir, Code: "brand.svg_invalid",
					Detail: fmt.Sprintf("根元素是 <%s>，期望 <svg>", start.Name.Local)}}
			}
			if viewBox := svgAttribute(start, "viewBox"); viewBox != "0 0 512 512" {
				return []brandFinding{{Package: packageDir, Code: "brand.svg_invalid",
					Detail: fmt.Sprintf("viewBox 为 %q，期望 \"0 0 512 512\"", viewBox)}}
			}
		}
		switch start.Name.Local {
		case "script", "image", "foreignObject", "iframe":
			return []brandFinding{{Package: packageDir, Code: "brand.svg_external_ref",
				Detail: fmt.Sprintf("<%s> 不允许出现在自包含图标里", start.Name.Local)}}
		}
		for _, attr := range start.Attr {
			if attr.Name.Local != "href" {
				continue
			}
			if strings.HasPrefix(attr.Value, "http") || strings.HasPrefix(attr.Value, "//") || strings.HasPrefix(attr.Value, "data:") {
				return []brandFinding{{Package: packageDir, Code: "brand.svg_external_ref",
					Detail: fmt.Sprintf("href=%q 指向包外资源", attr.Value)}}
			}
		}
	}
	if !sawRoot {
		return []brandFinding{{Package: packageDir, Code: "brand.svg_invalid", Detail: "没有 <svg> 根元素"}}
	}
	return nil
}

func svgAttribute(element xml.StartElement, name string) string {
	for _, attr := range element.Attr {
		if attr.Name.Local == name {
			return strings.TrimSpace(attr.Value)
		}
	}
	return ""
}

func checkBrandPNG(packageDir, path, missingCode, dimensionsCode string, wantWidth, wantHeight, maxBytes int) []brandFinding {
	body, err := os.ReadFile(path)
	if err != nil {
		return []brandFinding{{Package: packageDir, Code: missingCode,
			Detail: filepath.Join(brandDirName, filepath.Base(path)) + " 不存在"}}
	}
	label := filepath.Join(brandDirName, filepath.Base(path))
	if len(body) > maxBytes {
		return []brandFinding{{Package: packageDir, Code: "brand.too_large",
			Detail: fmt.Sprintf("%s 为 %d 字节，上限 %d", label, len(body), maxBytes)}}
	}
	width, height, ok := pngDimensions(body)
	if !ok {
		return []brandFinding{{Package: packageDir, Code: "brand.not_png", Detail: label + " 不是有效 PNG"}}
	}
	if width != wantWidth || height != wantHeight {
		return []brandFinding{{Package: packageDir, Code: dimensionsCode,
			Detail: fmt.Sprintf("%s 为 %dx%d，期望 %dx%d", label, width, height, wantWidth, wantHeight)}}
	}
	return nil
}

// pngDimensions 读 IHDR，避免为一次尺寸校验引入图像解码依赖。
func pngDimensions(body []byte) (int, int, bool) {
	signature := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}
	if len(body) < 24 || !bytes.Equal(body[:8], signature) {
		return 0, 0, false
	}
	if string(body[12:16]) != "IHDR" {
		return 0, 0, false
	}
	width := int(binary.BigEndian.Uint32(body[16:20]))
	height := int(binary.BigEndian.Uint32(body[20:24]))
	return width, height, true
}
