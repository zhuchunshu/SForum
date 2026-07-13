package extensions

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// BenchmarkExtensionEnableV1Baseline 固化 V3 P0 迁移前的同步启用成本：
// 包含包目录/Manifest 复核、Store 状态切换、事件记录与运行时状态装饰。
func BenchmarkExtensionEnableV1Baseline(b *testing.B) {
	ctx := context.Background()
	item := installedExtension("benchmark.plugin", TypePlugin, ManifestBackend{})
	item.Source = SourceUploaded
	root := filepath.Join(b.TempDir(), item.ID, item.Version)
	if err := os.MkdirAll(root, 0o755); err != nil {
		b.Fatal(err)
	}
	item.PackagePath = filepath.Join(root, "package.zip")
	if err := os.WriteFile(item.PackagePath, []byte("zip"), 0o600); err != nil {
		b.Fatal(err)
	}
	if err := writeManifest(root, item.Manifest); err != nil {
		b.Fatal(err)
	}
	store := &fakeExtensionStore{items: map[string]Extension{item.ID: item}}
	service := NewServiceWithRuntime(store, b.TempDir(), &fakeRuntimeManager{})
	actor := extensionManager()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		current := store.items[item.ID]
		current.Status = StatusInstalled
		store.items[item.ID] = current
		if _, err := service.Enable(ctx, actor, item.ID, EnableInput{}); err != nil {
			b.Fatal(err)
		}
	}
}
