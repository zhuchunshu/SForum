package notifications

import (
	"testing"

	options "github.com/zhuchunshu/sforum/apps/api/app/Models/Options"
)

func TestNotificationPolicyChannelsForType(t *testing.T) {
	policy := options.NotificationPolicy{
		Reply:      options.ChannelPolicy{InAppEnabled: false, EmailEnabled: true},
		Mention:    options.ChannelPolicy{InAppEnabled: true, EmailEnabled: false},
		Moderation: options.ChannelPolicy{InAppEnabled: false, EmailEnabled: false},
	}
	cases := []struct {
		kind         string
		inApp, email bool
	}{
		{TypeReply, false, true},
		{TypeMention, true, false},
		{TypeModerationApproved, false, false},
		{TypeModerationRejected, false, false},
	}
	for _, tc := range cases {
		inApp, email := channelsForType(policy, tc.kind)
		if inApp != tc.inApp || email != tc.email {
			t.Fatalf("%s channels=(%v,%v) want=(%v,%v)", tc.kind, inApp, email, tc.inApp, tc.email)
		}
	}
}
