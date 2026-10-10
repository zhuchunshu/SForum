package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	pluginv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2"
	protocolwire "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/protocol/v2"
)

// 庆祝通知类型（manifest notificationTypes 声明，命名空间必须为本插件）。
const notificationBadgeGranted = "sforum.gamification.badge_granted"

// celebrationEnabled 读取插件设置项的进程内覆盖；未配置时默认开启。
// 正式设置由 M4 的 settings schema 提供，这里只做开发通道。
func celebrationEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("SFORUM_GAMIFICATION_CELEBRATION"))) {
	case "0", "false", "off":
		return false
	}
	return true
}

// emitBadgeGranted 经 Host notifications.emit@1 发送庆祝通知。
// 失败只记日志：通知是仪式感，不能影响授予结果（授予已落库）。
func emitBadgeGranted(ctx context.Context, host *pluginv2.Host, parent *protocolwire.RequestContext, userID int64, badge BadgeDefinition, level string) {
	if host == nil || !celebrationEnabled() {
		return
	}
	payload := map[string]any{
		"badgeSlug":     badge.Slug,
		"badgeName":     badge.Name,
		"icon":          badge.Icon,
		"pointsGranted": badge.Points,
	}
	if level != "" {
		payload["levelTitle"] = level
	}
	// 幂等键：同一用户同一徽章只发一次。
	result, err := host.EmitNotification(ctx, parent, pluginv2.NotificationEmitInput{
		Type:             notificationBadgeGranted,
		PayloadVersion:   1,
		Payload:          payload,
		RecipientUserIDs: []int64{userID},
		IdempotencyKey:   fmt.Sprintf("badge-granted:%d:%s", userID, badge.Slug),
	})
	if err != nil {
		log.Printf("gamification notification %s user=%d failed: %v", notificationBadgeGranted, userID, err)
		return
	}
	if result.GetError() != nil {
		log.Printf("gamification notification %s user=%d rejected: %s", notificationBadgeGranted, userID, result.GetError().GetReason())
	}
}
