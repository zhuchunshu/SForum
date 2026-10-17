package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestSyncInlineDeclarationDigestsRefreshesEditorL2Module(t *testing.T) {
	manifest := map[string]any{
		"editor": []any{
			map[string]any{
				"kind": "command", "l2Module": "frontend/editor/demo.mjs", "l2Digest": "old",
			},
			map[string]any{
				"kind": "toolbar", "commandId": "demo.command",
			},
		},
	}
	syncInlineDeclarationDigests(manifest, map[string]string{
		"frontend/editor/demo.mjs": "new-digest",
	})
	items := manifest["editor"].([]any)
	command := items[0].(map[string]any)
	if command["l2Digest"] != "new-digest" {
		t.Fatalf("l2Digest = %v", command["l2Digest"])
	}
	toolbar := items[1].(map[string]any)
	if _, ok := toolbar["l2Digest"]; ok {
		t.Fatalf("toolbar unexpectedly received l2Digest: %#v", toolbar)
	}
}

// digest --write 必须顺带重算已声明 integrity 的资产 SRI：资产文件一改旧值必然
// 过期（Host 校验 fail closed），此前作者只能像 prepare-package.sh 那样手写脚本重算。
// 从缺省（空）开始的清单不被悄悄改写，既有包基线不受影响。
func TestRefreshAssetIntegrityRewritesStaleSRI(t *testing.T) {
	root := t.TempDir()
	assetDir := filepath.Join(root, "assets")
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := []byte(".sf-theme{color:rebeccapurple}\n")
	if err := os.WriteFile(filepath.Join(assetDir, "theme.css"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	staleDigest := sha256.Sum256(body)
	want := "sha256-" + base64.StdEncoding.EncodeToString(staleDigest[:])
	manifest := map[string]any{
		"assets": []any{
			map[string]any{"path": "assets/theme.css", "digest": "0", "integrity": "sha256-StaleValue"},
			map[string]any{"path": "assets/theme.css", "digest": "0", "integrity": want},
			map[string]any{"path": "assets/fresh.css", "digest": "0"},
		},
	}
	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)
	if err := refreshAssetIntegrity(cmd, root, manifest); err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	assets := manifest["assets"].([]any)
	first := assets[0].(map[string]any)
	if first["integrity"] != want {
		t.Fatalf("stale integrity = %q; want %q", first["integrity"], want)
	}
	if !strings.Contains(out.String(), "integrity refreshed: assets/theme.css") {
		t.Fatalf("refresh notice missing: %q", out.String())
	}
	// 已是最新值的资产不被改写也无需打印；从缺省开始的清单保持原样。
	second := assets[1].(map[string]any)
	if second["integrity"] != want {
		t.Fatalf("current integrity unexpectedly rewritten: %v", second["integrity"])
	}
	if strings.Contains(out.String(), "integrity refreshed: assets/fresh.css") {
		t.Fatalf("undeclared integrity must stay untouched")
	}
}

func TestRefreshAssetIntegrityReportsMissingAsset(t *testing.T) {
	root := t.TempDir()
	manifest := map[string]any{
		"assets": []any{
			map[string]any{"path": "assets/absent.css", "integrity": "sha256-whatever"},
		},
	}
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	err := refreshAssetIntegrity(cmd, root, manifest)
	if err == nil || !strings.Contains(err.Error(), "recompute integrity for assets/absent.css") {
		t.Fatalf("missing asset must fail with its path: %v", err)
	}
}

func TestRefreshAssetIntegrityLeavesUndeclaredIntegrityUntouched(t *testing.T) {
	root := t.TempDir()
	assetDir := filepath.Join(root, "assets")
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assetDir, "fresh.css"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	entry := map[string]any{"path": "assets/fresh.css"}
	manifest := map[string]any{"assets": []any{entry}}
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	if err := refreshAssetIntegrity(cmd, root, manifest); err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	if _, ok := entry["integrity"]; ok {
		t.Fatalf("integrity must not be invented for undeclared assets: %#v", entry)
	}
}
