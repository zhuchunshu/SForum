package extensionsruntime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

func TestPackageLocalComponentSSRPublishAndRender(t *testing.T) {
	root := t.TempDir()
	body := `<aside data-e2e="plugin-ssr">{{index .Props "title"}}</aside>`
	writePackageLocalTestFile(t, root, "templates/card.html", body)
	digest := sha256.Sum256([]byte(body))
	digestHex := hex.EncodeToString(digest[:])
	packageDigest := strings.Repeat("a", 64)
	extension := extensions.Extension{
		ID: "demo.ssr", Version: "1.0.0", Type: extensions.TypePlugin,
		PackageDigest: packageDigest, PackagePath: root,
		Manifest: extensions.Manifest{
			ManifestVersion: 3, ID: "demo.ssr", Version: "1.0.0", Type: extensions.TypePlugin,
			Components: []extensions.ManifestComponent{{
				ID: "demo.ssr.component.card", ContractVersion: "demo.ssr.component.card@1",
				Action: extensionmanifest.ComponentActionBefore, TargetID: componentTestCoreTarget,
				TargetContractVersion: componentTestCoreContract, Priority: 10,
				SSRTemplate: "demo.ssr.template.card", PropsSchema: "demo.ssr.schema.props@1",
			}},
			Templates: []extensions.ManifestTemplate{{
				ID: "demo.ssr.template.card", ContractVersion: "demo.ssr.template.card@1",
				Action: "add", Path: "templates/card.html", Digest: digestHex,
				ViewModelSchema: "demo.ssr.schema.props@1",
			}},
		},
	}
	renderer := NewPackageLocalComponentSSRRenderer()
	if err := renderer.Publish(extension); err != nil {
		t.Fatal(err)
	}
	if renderer.Count() != 1 {
		t.Fatalf("count=%d", renderer.Count())
	}
	response, err := renderer.RenderComponent(context.Background(), ComponentRenderCall{
		TargetID: componentTestCoreTarget, TargetContractVersion: componentTestCoreContract,
		Contribution: ComponentContribution{
			ID: "demo.ssr.component.card", Action: extensionmanifest.ComponentActionBefore,
			SSRTemplate: "demo.ssr.template.card",
			Artifact: HookArtifact{
				ExtensionID: "demo.ssr", ExtensionVersion: "1.0.0",
				PackageDigest: packageDigest, RuntimeInstanceID: "host-component-package:demo.ssr",
			},
		},
		Artifact: HookArtifact{
			ExtensionID: "demo.ssr", ExtensionVersion: "1.0.0",
			PackageDigest: packageDigest, RuntimeInstanceID: "host-component-package:demo.ssr",
		},
		Props: map[string]any{"title": "Hello SSR"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Fragments) != 1 ||
		!strings.Contains(response.Fragments[0].ReviewedHTML, `data-e2e="plugin-ssr"`) ||
		!strings.Contains(response.Fragments[0].ReviewedHTML, "Hello SSR") {
		t.Fatalf("fragments=%#v", response.Fragments)
	}

	// 请求路径不再读包文件：删除源文件后仍可渲染。
	if err := os.Remove(filepath.Join(root, "templates/card.html")); err != nil {
		t.Fatal(err)
	}
	again, err := renderer.RenderComponent(context.Background(), ComponentRenderCall{
		TargetID: componentTestCoreTarget,
		Contribution: ComponentContribution{
			ID: "demo.ssr.component.card", Action: extensionmanifest.ComponentActionBefore,
			SSRTemplate: "demo.ssr.template.card",
			Artifact: HookArtifact{
				ExtensionID: "demo.ssr", PackageDigest: packageDigest,
				RuntimeInstanceID: "host-component-package:demo.ssr",
			},
		},
		Artifact: HookArtifact{
			ExtensionID: "demo.ssr", PackageDigest: packageDigest,
			RuntimeInstanceID: "host-component-package:demo.ssr",
		},
		Props: map[string]any{"title": "Cached"},
	})
	if err != nil || !strings.Contains(again.Fragments[0].ReviewedHTML, "Cached") {
		t.Fatalf("cached render err=%v fragments=%#v", err, again.Fragments)
	}

	// digest 漂移 fail closed。
	bad := extension
	bad.Manifest.Templates[0].Digest = strings.Repeat("b", 64)
	writePackageLocalTestFile(t, root, "templates/card.html", body)
	if err := renderer.Publish(bad); err == nil {
		t.Fatal("expected digest mismatch")
	}

	renderer.RemovePackage(packageDigest)
	if renderer.Count() != 0 {
		t.Fatalf("after remove count=%d", renderer.Count())
	}
	if _, err := renderer.RenderComponent(context.Background(), ComponentRenderCall{
		Contribution: ComponentContribution{
			ID: "demo.ssr.component.card", SSRTemplate: "demo.ssr.template.card",
			Action: extensionmanifest.ComponentActionBefore,
			Artifact: HookArtifact{ExtensionID: "demo.ssr", PackageDigest: packageDigest},
		},
		Artifact: HookArtifact{ExtensionID: "demo.ssr", PackageDigest: packageDigest},
		Props:    map[string]any{"title": "gone"},
	}); err == nil {
		t.Fatal("expected crash after remove")
	}
}

func TestProductionComponentCompositionUsesPackageLocalSSRByDefault(t *testing.T) {
	id := "production.package.ssr"
	packageDigest := strings.Repeat("c", 64)
	extension := componentTestExtension(t, id, extensions.TypePlugin,
		componentTestContribution(
			id, "before", extensionmanifest.ComponentActionBefore, 10,
			componentTestCoreTarget, componentTestCoreContract,
		),
	)
	extension.PackageDigest = packageDigest
	// 在 extension 自己的 PackagePath 写入 SSR 模板并同步 digest。
	body := `<div class="plugin-before">{{index .Props "scope"}}</div>`
	writePackageLocalTestFile(t, extension.PackagePath, "templates/default.html", body)
	digest := sha256.Sum256([]byte(body))
	extension.Manifest.Templates[0].Path = "templates/default.html"
	extension.Manifest.Templates[0].Digest = hex.EncodeToString(digest[:])

	registry := NewComponentRegistry()
	if err := registry.ReplaceRuntime(extension, componentPackageRuntimeInstanceID(extension)); err != nil {
		t.Fatal(err)
	}
	service, err := NewProductionComponentComposition(ProductionComponentCompositionConfig{
		Registry: registry,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.PublishPackageSSR(extension); err != nil {
		t.Fatal(err)
	}
	result, err := service.Compose(context.Background(), ComponentCompositionRequest{
		TargetID: componentTestCoreTarget, TargetContractVersion: componentTestCoreContract,
		Props: map[string]any{"scope": "home"},
	})
	if err != nil {
		t.Fatal(err)
	}
	// 应包含插件 before 片段（HTML 已消毒）与 Core primary。
	joined := flattenComponentHTML(result.Segments)
	if !strings.Contains(joined, "plugin-before") || !strings.Contains(joined, "home") {
		t.Fatalf("composed HTML=%q segments=%#v", joined, result.Segments)
	}
}

func writePackageLocalTestFile(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func flattenComponentHTML(segments []ComponentRenderSegment) string {
	var builder strings.Builder
	var walk func([]ComponentRenderSegment)
	walk = func(items []ComponentRenderSegment) {
		for _, item := range items {
			builder.WriteString(item.HTML)
			walk(item.Children)
		}
	}
	walk(segments)
	return builder.String()
}
