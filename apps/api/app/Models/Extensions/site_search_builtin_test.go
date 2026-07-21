package extensions

import (
	"context"
	"errors"
	"testing"
)

// 确认站内搜索扩展 id 走真实 Uninstall 门禁（与其它 protected builtin 相同）。
func TestUninstallRejectsProtectedSiteSearchBuiltin(t *testing.T) {
	item := Extension{
		ID:          "sforum.search-site",
		Type:        TypePlugin,
		Status:      StatusInstalled,
		Source:      SourceBuiltin,
		IsSystem:    true,
		IsDeletable: false,
		Name:        "Site Search",
		Version:     "1.0.0",
		Manifest: Manifest{
			ID:   "sforum.search-site",
			Type: TypePlugin,
			Providers: []ManifestProvider{{
				Slot:  "search.provider",
				Label: "Site Search",
			}},
		},
	}
	store := newFakeExtensionStore(map[string]Extension{item.ID: item})
	service := NewService(store, t.TempDir())
	err := service.Uninstall(context.Background(), extensionManager(), item.ID, UninstallInput{})
	if !errors.Is(err, ErrNotDeletable) {
		t.Fatalf("uninstall site search: %v, want ErrNotDeletable", err)
	}
	got, getErr := store.Get(context.Background(), item.ID)
	if getErr != nil {
		t.Fatalf("Get after rejected uninstall: %v", getErr)
	}
	if got.ID != item.ID {
		t.Fatalf("extension disappeared after rejected uninstall")
	}
}
