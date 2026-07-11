package notificationjobs

import (
	"encoding/json"
	"fmt"

	notifications "github.com/zhuchunshu/sforum/apps/api/app/Models/Notifications"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
)

func renderDelivery(delivery notifications.MailDelivery) extensionsruntime.MailProviderRequest {
	data := map[string]any{}
	_ = json.Unmarshal(delivery.TemplateData, &data)
	subject, _ := data["subject"].(string)
	body, _ := data["textBody"].(string)
	if subject == "" {
		switch delivery.TemplateKey {
		case "forum.reply":
			subject, body = "[SForum] 你收到了新回复 / New reply", forumBody("有人回复了你的内容。", "Someone replied to your content.", data)
		case "forum.mention":
			subject, body = "[SForum] 你被提及了 / You were mentioned", forumBody("有人在内容中提到了你。", "Someone mentioned you in a post.", data)
		case "forum.moderation_approved":
			subject, body = "[SForum] 内容已通过审核 / Content approved", forumBody("你的内容已通过审核。", "Your content was approved.", data)
		case "forum.moderation_rejected":
			subject, body = "[SForum] 内容未通过审核 / Content rejected", forumBody("你的内容未通过审核。", "Your content was rejected.", data)
		default:
			subject, body = "[SForum] 通知 / Notification", "你有一条新的站内通知。\n\nYou have a new in-app notification."
		}
	}
	return extensionsruntime.MailProviderRequest{DeliveryID: fmt.Sprint(delivery.ID), CorrelationID: delivery.CorrelationID, To: []string{delivery.Recipient}, Subject: subject, TextBody: body}
}

func forumBody(zh, en string, data map[string]any) string {
	return fmt.Sprintf("%s\n\n%s\n\nTopic ID: %v\nTarget ID: %v", zh, en, data["topicId"], data["targetId"])
}
