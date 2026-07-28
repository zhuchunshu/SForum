package options

import (
	"reflect"
	"testing"
)

func TestNotificationPolicyDefaultsToInAppOnly(t *testing.T) {
	got := notificationPolicyFromValues(nil)
	want := NotificationPolicy{
		Reply:      ChannelPolicy{InAppEnabled: true, EmailEnabled: false},
		Mention:    ChannelPolicy{InAppEnabled: true, EmailEnabled: false},
		Moderation: ChannelPolicy{InAppEnabled: true, EmailEnabled: false},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("policy=%#v want=%#v", got, want)
	}
}

func TestNotificationPolicyResolvesChannelsIndependently(t *testing.T) {
	got := notificationPolicyFromValues(map[string]string{
		NameNotificationReplyInApp:   "disabled",
		NameNotificationReplyEmail:   "enabled",
		NameNotificationMentionEmail: "disabled",
	})
	if got.Reply.InAppEnabled || !got.Reply.EmailEnabled {
		t.Fatalf("unexpected reply policy: %#v", got.Reply)
	}
	if !got.Mention.InAppEnabled || got.Mention.EmailEnabled {
		t.Fatalf("unexpected mention policy: %#v", got.Mention)
	}
	if !got.Moderation.InAppEnabled || got.Moderation.EmailEnabled {
		t.Fatalf("unexpected moderation policy: %#v", got.Moderation)
	}
}

func TestNotificationPolicyUpdateAndRecommendedInputs(t *testing.T) {
	policy := NotificationPolicy{
		Reply:      ChannelPolicy{InAppEnabled: false, EmailEnabled: true},
		Mention:    ChannelPolicy{InAppEnabled: true, EmailEnabled: false},
		Moderation: ChannelPolicy{InAppEnabled: false, EmailEnabled: false},
	}
	inputs := NotificationPolicyUpdateInputs(policy)
	values := make(map[string]string, len(inputs))
	for _, input := range inputs {
		values[input.Name] = input.Value
	}
	if values[NameNotificationReplyInApp] != "disabled" || values[NameNotificationReplyEmail] != "enabled" || values[NameNotificationMentionEmail] != "disabled" {
		t.Fatalf("unexpected update inputs: %#v", values)
	}

	for _, input := range NotificationPolicyRecommendedInputs() {
		want := "enabled"
		if input.Name == NameNotificationReplyEmail || input.Name == NameNotificationMentionEmail || input.Name == NameNotificationModerationEmail {
			want = "disabled"
		}
		if input.Value != want {
			t.Fatalf("recommended input %s=%q want=%q", input.Name, input.Value, want)
		}
	}
}
