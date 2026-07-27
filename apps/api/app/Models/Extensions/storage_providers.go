package extensions

import (
	"context"
	"sort"
	"strings"

	storage "github.com/zhuchunshu/sforum/apps/api/app/Support/Storage"
)

// ListStorageProviderCandidates 返回已启用且声明 attachment.storage.provider 的插件候选（E6.1）。
// 不要求 actor：由附件设置 API 在已鉴权路径调用。
func (s *CatalogService) ListStorageProviderCandidates(ctx context.Context) ([]storage.Candidate, error) {
	if s.safeMode {
		return []storage.Candidate{}, nil
	}
	items, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]storage.Candidate, 0)
	for _, item := range items {
		if item.Type != TypePlugin || item.Status != StatusEnabled {
			continue
		}
		label, ok := storageProviderLabel(item)
		if !ok {
			continue
		}
		settingsPath := ""
		if item.Manifest.Admin.Entry != "" {
			// 管理端扩展设置页固定命名空间。
			settingsPath = "/extensions/" + item.ID + "/pages/settings"
		}
		out = append(out, storage.PluginCandidate(item.ID, label, settingsPath))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ExtensionID < out[j].ExtensionID
	})
	return out, nil
}

// IsStorageProviderAvailable 判断扩展是否可作为存储提供方被选中。
func (s *CatalogService) IsStorageProviderAvailable(ctx context.Context, extensionID string) (bool, error) {
	if s.safeMode {
		return false, nil
	}
	extensionID = normalizeID(extensionID)
	if extensionID == "" {
		return false, nil
	}
	item, err := s.store.Get(ctx, extensionID)
	if err != nil {
		if err == ErrExtensionNotFound {
			return false, nil
		}
		return false, err
	}
	if item.Type != TypePlugin || item.Status != StatusEnabled {
		return false, nil
	}
	_, ok := storageProviderLabel(item)
	return ok, nil
}

func storageProviderLabel(item Extension) (string, bool) {
	for _, provider := range item.Manifest.Providers {
		if strings.TrimSpace(provider.Slot) != storage.ProviderSlot {
			continue
		}
		label := strings.TrimSpace(provider.Label)
		if label == "" {
			label = item.Name
		}
		if label == "" {
			label = item.ID
		}
		return label, true
	}
	return "", false
}
