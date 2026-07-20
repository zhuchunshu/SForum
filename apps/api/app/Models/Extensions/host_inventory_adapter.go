package extensions

import (
	"context"
	"time"
)

// ListRedactedInventory 实现 hostapi.ExtensionInventoryReader。
// 将去敏结构体转为 map[string]any，便于 Host Query 编码。
func (s *Service) ListRedactedInventoryMaps(ctx context.Context) ([]map[string]any, error) {
	items, err := s.ListRedactedInventory(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, redactedInventoryItemMap(item))
	}
	return out, nil
}

// HostInventoryAdapter 把 Service 适配为 HostAPI ExtensionInventoryReader。
type HostInventoryAdapter struct {
	Service *Service
}

func (a HostInventoryAdapter) ListRedactedInventory(ctx context.Context) ([]map[string]any, error) {
	if a.Service == nil {
		return nil, ErrRuntimeUnavailable
	}
	return a.Service.ListRedactedInventoryMaps(ctx)
}

func redactedInventoryItemMap(item RedactedExtensionInventoryItem) map[string]any {
	row := map[string]any{
		"id":          item.ID,
		"name":        item.Name,
		"version":     item.Version,
		"type":        item.Type,
		"status":      item.Status,
		"source":      item.Source,
		"isSystem":    item.IsSystem,
		"installedAt": item.InstalledAt.UTC().Format(time.RFC3339Nano),
		"updatedAt":   item.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if item.PackageDigest != "" {
		row["packageDigest"] = item.PackageDigest
	}
	if len(item.Capabilities) > 0 {
		// 复制切片，避免调用方持有内部引用。
		caps := make([]any, len(item.Capabilities))
		for i, key := range item.Capabilities {
			caps[i] = key
		}
		row["capabilities"] = caps
	}
	if item.RuntimeState != "" {
		row["runtimeState"] = item.RuntimeState
	}
	if item.Protocol != "" {
		row["protocolTransport"] = item.Protocol
	}
	return row
}
