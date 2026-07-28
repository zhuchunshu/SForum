package options

import "context"

type ChannelPolicy struct {
	InAppEnabled bool `json:"inAppEnabled"`
	EmailEnabled bool `json:"emailEnabled"`
}

type NotificationPolicy struct {
	Reply      ChannelPolicy `json:"reply"`
	Mention    ChannelPolicy `json:"mention"`
	Moderation ChannelPolicy `json:"moderation"`
}

func notificationOptionNames() []string {
	return []string{
		NameNotificationReplyInApp,
		NameNotificationReplyEmail,
		NameNotificationMentionInApp,
		NameNotificationMentionEmail,
		NameNotificationModerationInApp,
		NameNotificationModerationEmail,
	}
}

func (s *Service) NotificationPolicy(ctx context.Context) (NotificationPolicy, error) {
	values, err := s.loadMap(ctx)
	if err != nil {
		return NotificationPolicy{}, err
	}
	return notificationPolicyFromValues(values), nil
}

func notificationPolicyFromValues(values map[string]string) NotificationPolicy {
	enabled := func(name string, recommended bool) bool {
		value, ok := values[name]
		if !ok || value == "" {
			return recommended
		}
		return isEnabledOption(value)
	}
	return NotificationPolicy{
		Reply:      ChannelPolicy{InAppEnabled: enabled(NameNotificationReplyInApp, true), EmailEnabled: enabled(NameNotificationReplyEmail, false)},
		Mention:    ChannelPolicy{InAppEnabled: enabled(NameNotificationMentionInApp, true), EmailEnabled: enabled(NameNotificationMentionEmail, false)},
		Moderation: ChannelPolicy{InAppEnabled: enabled(NameNotificationModerationInApp, true), EmailEnabled: enabled(NameNotificationModerationEmail, false)},
	}
}

func NotificationPolicyUpdateInputs(policy NotificationPolicy) []UpdateInput {
	return []UpdateInput{
		{Name: NameNotificationReplyInApp, Value: enabledOptionValue(policy.Reply.InAppEnabled)},
		{Name: NameNotificationReplyEmail, Value: enabledOptionValue(policy.Reply.EmailEnabled)},
		{Name: NameNotificationMentionInApp, Value: enabledOptionValue(policy.Mention.InAppEnabled)},
		{Name: NameNotificationMentionEmail, Value: enabledOptionValue(policy.Mention.EmailEnabled)},
		{Name: NameNotificationModerationInApp, Value: enabledOptionValue(policy.Moderation.InAppEnabled)},
		{Name: NameNotificationModerationEmail, Value: enabledOptionValue(policy.Moderation.EmailEnabled)},
	}
}

func NotificationPolicyRecommendedInputs() []UpdateInput {
	recommended := ChannelPolicy{InAppEnabled: true, EmailEnabled: false}
	return NotificationPolicyUpdateInputs(NotificationPolicy{Reply: recommended, Mention: recommended, Moderation: recommended})
}
