package extensions

import (
	"context"
	"strings"
	"testing"
)

func TestListRedactedInventoryOmitsSensitiveFields(t *testing.T) {
	plugin := uploadedExtension("demo.plugin", TypePlugin)
	plugin.Status = StatusEnabled
	plugin.PackagePath = "/secret/packages/demo.zip"
	plugin.Manifest.Capabilities = []string{"extensions.read", "settings.own"}
	plugin.PackageDigest = strings.Repeat("a", 64)

	store := &fakeExtensionStore{items: map[string]Extension{plugin.ID: plugin}}
	service := NewService(store, t.TempDir())

	rows, err := service.ListRedactedInventory(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows=%#v", rows)
	}
	row := rows[0]
	if row.ID != "demo.plugin" || row.PackageDigest != plugin.PackageDigest || row.Status != StatusEnabled {
		t.Fatalf("row=%#v", row)
	}
	if len(row.Capabilities) != 2 {
		t.Fatalf("capabilities=%#v", row.Capabilities)
	}
	// 结构体本身不得携带 packagePath；拼装字符串也不应泄漏路径片段。
	encoded := strings.ToLower(row.ID + row.Name + row.PackageDigest + strings.Join(row.Capabilities, ","))
	if strings.Contains(encoded, "secret") || strings.Contains(encoded, ".zip") {
		t.Fatalf("redacted row leaked path material: %#v", row)
	}
}

func TestListRedactedInventoryDeniesSafeMode(t *testing.T) {
	plugin := uploadedExtension("demo.plugin", TypePlugin)
	store := &fakeExtensionStore{items: map[string]Extension{plugin.ID: plugin}}
	service := NewServiceWithOptions(store, t.TempDir(), "", nil, WithSafeMode(true))
	if _, err := service.ListRedactedInventory(context.Background()); err != ErrSafeModeActive {
		t.Fatalf("err=%v", err)
	}
}
