package notificationjobs

import (
	"encoding/json"
	"strings"
	"testing"

	notifications "github.com/zhuchunshu/sforum/apps/api/app/Models/Notifications"
)

func TestRenderDeliveryUsesStoredPasswordResetContent(t *testing.T) {
	data, _ := json.Marshal(map[string]string{"subject": "Reset", "textBody": "secret link"})
	request := renderDelivery(notifications.MailDelivery{ID: 1, Recipient: "u@example.com", TemplateKey: "identity.password_reset", TemplateData: data})
	if request.Subject != "Reset" || request.TextBody != "secret link" {
		t.Fatalf("unexpected request: %#v", request)
	}
}
func TestRenderDeliveryCreatesBilingualForumMail(t *testing.T) {
	request := renderDelivery(notifications.MailDelivery{ID: 2, Recipient: "u@example.com", TemplateKey: "forum.mention", TemplateData: json.RawMessage(`{"topicId":42}`)})
	if !strings.Contains(request.Subject, "mentioned") || !strings.Contains(request.TextBody, "提到") {
		t.Fatalf("unexpected request: %#v", request)
	}
}
