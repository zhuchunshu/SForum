package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func digestFile(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// CLI 与运行时必须对路由 schema 目录给出一致判断（2026-10-16）。
//
// 背景：路由声明了 responseSchema，但 OpenAPI 片段无法解析该 schema 时，
// `extension validate` / `extension test` 曾报 0 error，而 API **启动失败**
// （`prepare startup route schema publication: … route schema is missing`），
// dev API 因此被拖挂。现在 CLI 构建同一份目录，作者在命令行就能拿到可理解的失败。
func TestExtensionValidateBuildsRouteSchemaCatalog(t *testing.T) {
	const manifestTemplate = `{
  "manifestVersion": 3,
  "id": "acme.catalog-probe",
  "name": "Acme Catalog Probe",
  "description": "Route schema catalog parity probe.",
  "url": "https://example.com/catalog-probe",
  "author": {"name": "Acme"},
  "version": "1.0.0",
  "type": "plugin",
  "sforumVersion": ">=0.1.0",
  "routes": [{
    "id": "acme.catalog-probe.route.data",
    "contractVersion": "acme.catalog-probe.route.data@1",
    "action": "add",
    "path": "/acme/data",
    "methods": ["GET"],
    "guard": "core.guard.public",
    "fallback": "closed",
    "mode": "http",
    "handler": "acme.catalog-probe.route.data",
    "responseSchema": "acme.catalog-probe.data.response@1"
  }]%s
}
`
	// 拒绝侧：没有 OpenAPI 片段 → schema 无法解析 → CLI 必须失败（否则 API 启动会失败）。
	missing := filepath.Join(t.TempDir(), "missing-fragment")
	if err := os.MkdirAll(missing, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(missing, "sforum.extension.json"),
		[]byte(strings.Replace(manifestTemplate, "%s", "", 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := newRootCommand()
	cmd.SetArgs([]string{"extension", "validate", missing})
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	err := cmd.Execute()
	if err == nil {
		t.Fatalf("validate accepted a route schema that the runtime cannot resolve:\n%s", out.String())
	}
	if !strings.Contains(err.Error(), "route schema catalog") {
		t.Fatalf("expected a route-schema-catalog failure, got: %v", err)
	}

	// 允许侧：声明可解析的 OpenAPI 片段后，同一个包必须通过。
	resolved := filepath.Join(t.TempDir(), "with-fragment")
	if err := os.MkdirAll(filepath.Join(resolved, "openapi"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(resolved, "openapi", "routes.yaml"), []byte(`openapi: 3.1.0
info:
  title: Acme Catalog Probe
  version: 1.0.0
paths:
  /acme/data:
    get:
      operationId: acme.catalog-probe.data
      x-sforum-route-id: acme.catalog-probe.route.data
      x-sforum-contract-version: acme.catalog-probe.route.data@1
      x-sforum-guard: core.guard.public
      x-sforum-response-schema: acme.catalog-probe.data.response@1
      responses:
        "200":
          description: Data document
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/acme.catalog-probe.data.response@1"
components:
  schemas:
    "acme.catalog-probe.data.response@1":
      type: object
      required: [title]
      additionalProperties: false
      properties:
        title: {type: string}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	extra := `,
  "openapi": [{
    "id": "acme.catalog-probe.openapi.routes",
    "contractVersion": "acme.catalog-probe.openapi.routes@1",
    "path": "openapi/routes.yaml",
    "namespace": "acme.catalog-probe"
  }],
  "packageFiles": [{
    "id": "acme.catalog-probe.file.openapi",
    "kind": "openapi",
    "path": "openapi/routes.yaml"
  }]`
	body := strings.Replace(manifestTemplate, "%s", extra, 1)
	body = strings.ReplaceAll(body, `"path": "openapi/routes.yaml"`, `"path": "openapi/routes.yaml", "digest": "`+digestFile(t, filepath.Join(resolved, "openapi", "routes.yaml"))+`"`)
	if err := os.WriteFile(filepath.Join(resolved, "sforum.extension.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd = newRootCommand()
	cmd.SetArgs([]string{"extension", "validate", resolved})
	out.Reset()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("validate rejected a resolvable route schema: %v\n%s", err, out.String())
	}
}
