package notifications

import (
	"context"
	"encoding/json"
)

type TargetVisibilityResolver interface {
	ResolveNotificationTarget(context.Context, int64, string, int64) (available bool, path string, err error)
}

type TargetPreviewResolver interface {
	ResolveNotificationTargetPreview(context.Context, int64, string, int64) (TargetPreview, bool, error)
}

// ResolveSafeTargets 在通知进入任一呈现层前重新核验目标。解析失败按不可见
// 处理，避免临时授权/数据错误退化成 payload 或路由泄漏。
func ResolveSafeTargets(ctx context.Context, resolver TargetVisibilityResolver, userID int64, page Page) (Page, error) {
	for index := range page.Items {
		item := page.Items[index]
		available, path := false, ""
		if resolver != nil {
			var err error
			available, path, err = resolver.ResolveNotificationTarget(ctx, userID, item.TargetType, item.TargetID)
			if err != nil && ctx.Err() != nil {
				return Page{}, ctx.Err()
			}
			if err != nil {
				available, path = false, ""
			}
		}
		if available && path != "" {
			item.TargetAvailable = true
			item.TargetPath = path
			page.Items[index] = item
			continue
		}
		item.ActorUserID = nil
		item.TargetType = "unavailable"
		item.TargetID = 0
		item.TargetAvailable = false
		item.TargetPath = ""
		item.Payload = json.RawMessage(`{}`)
		page.Items[index] = item
	}
	return page, nil
}

func ResolveNotificationDetail(ctx context.Context, visibility TargetVisibilityResolver, previews TargetPreviewResolver, userID int64, item Notification) (NotificationDetail, error) {
	page, err := ResolveSafeTargets(ctx, visibility, userID, Page{Items: []Notification{item}})
	if err != nil {
		return NotificationDetail{}, err
	}
	item = page.Items[0]
	detail := NotificationDetail{Notification: item}
	if !item.TargetAvailable || previews == nil {
		return detail, nil
	}
	preview, available, err := previews.ResolveNotificationTargetPreview(ctx, userID, item.TargetType, item.TargetID)
	if err != nil {
		if ctx.Err() != nil {
			return NotificationDetail{}, ctx.Err()
		}
		return detail, nil
	}
	if available {
		detail.Preview = &preview
	}
	return detail, nil
}
