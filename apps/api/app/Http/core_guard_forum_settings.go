package http

// 论坛设置路由守卫的请求 DTO 与字段分组判定。
// 从 core_guard_authorizer.go 抽出，避免该文件的 1000 行硬上限被新字段突破。

type forumSettingsGuardInput struct {
	DefaultCategorySlug *string `json:"defaultCategorySlug"`
	TagCreationMode     *string `json:"tagCreationMode"`
	TagPublicPages      *bool   `json:"tagPublicPages"`
	TagMinPerTopic      *int    `json:"tagMinPerTopic"`
	TagMaxPerTopic      *int    `json:"tagMaxPerTopic"`
	TopicsPerPage       *int    `json:"topicsPerPage"`
	CommentsPerPage     *int    `json:"commentsPerPage"`

	TopicTitleMinRunes       *int `json:"topicTitleMinRunes"`
	TopicTitleMaxRunes       *int `json:"topicTitleMaxRunes"`
	TopicContentMinRunes     *int `json:"topicContentMinRunes"`
	TopicContentMaxRunes     *int `json:"topicContentMaxRunes"`
	TopicEditWindowMinutes   *int `json:"topicEditWindowMinutes"`
	TopicCooldownSeconds     *int `json:"topicCooldownSeconds"`
	DailyTopicLimit          *int `json:"dailyTopicLimit"`
	CommentMinRunes          *int `json:"commentMinRunes"`
	CommentMaxRunes          *int `json:"commentMaxRunes"`
	CommentMaxNestingDepth   *int `json:"commentMaxNestingDepth"`
	TreeDescendantsPerRoot   *int `json:"treeDescendantsPerRoot"`
	CommentEditWindowMinutes *int `json:"commentEditWindowMinutes"`
	CommentCooldownSeconds   *int `json:"commentCooldownSeconds"`
	DailyCommentLimit        *int `json:"dailyCommentLimit"`
	ExcerptRuneLimit         *int `json:"excerptRuneLimit"`

	GuestRead               *string `json:"guestRead"`
	ListDefaultSort         *string `json:"listDefaultSort"`
	ListHotWindowDays       *int    `json:"listHotWindowDays"`
	AllowAuthorCloseReplies *bool   `json:"allowAuthorCloseReplies"`
	AllowAuthorDelete       *bool   `json:"allowAuthorDelete"`
	AutoLockIdleDays        *int    `json:"autoLockIdleDays"`
	ShowTopicEditMark       *bool   `json:"showTopicEditMark"`
	DuplicateTitlePolicy    *string `json:"duplicateTitlePolicy"`
	ShowCommentEditMark     *bool   `json:"showCommentEditMark"`
	SoftDeleteVisibility    *string `json:"softDeleteVisibility"`
	MentionsEnabled         *bool   `json:"mentionsEnabled"`
	MentionsMaxPerPost      *int    `json:"mentionsMaxPerPost"`

	// 正文外链呈现（点击时判定，写权限随论坛设置）
	ExternalInterstitial *bool   `json:"externalInterstitial"`
	TrustedDomains       *string `json:"trustedDomains"`
}

func decodeForumSettingsGuardInput(body []byte) (forumSettingsGuardInput, error) {
	var input forumSettingsGuardInput
	if err := decodeGuardJSON(body, &input); err != nil {
		return forumSettingsGuardInput{}, err
	}
	return input, nil
}

func forumTagSettingsPresent(input forumSettingsGuardInput) bool {
	return input.TagCreationMode != nil || input.TagPublicPages != nil ||
		input.TagMinPerTopic != nil || input.TagMaxPerTopic != nil
}

func forumRuntimeSettingsPresent(input forumSettingsGuardInput) bool {
	return input.TopicsPerPage != nil || input.CommentsPerPage != nil ||
		input.TopicTitleMinRunes != nil || input.TopicTitleMaxRunes != nil ||
		input.TopicContentMinRunes != nil || input.TopicContentMaxRunes != nil ||
		input.TopicEditWindowMinutes != nil || input.TopicCooldownSeconds != nil ||
		input.DailyTopicLimit != nil || input.CommentMinRunes != nil ||
		input.CommentMaxRunes != nil || input.CommentMaxNestingDepth != nil ||
		input.TreeDescendantsPerRoot != nil ||
		input.CommentEditWindowMinutes != nil || input.CommentCooldownSeconds != nil ||
		input.DailyCommentLimit != nil || input.ExcerptRuneLimit != nil ||
		input.GuestRead != nil || input.ListDefaultSort != nil || input.ListHotWindowDays != nil ||
		input.AllowAuthorCloseReplies != nil || input.AllowAuthorDelete != nil || input.AutoLockIdleDays != nil ||
		input.ShowTopicEditMark != nil || input.DuplicateTitlePolicy != nil || input.ShowCommentEditMark != nil ||
		input.SoftDeleteVisibility != nil || input.MentionsEnabled != nil || input.MentionsMaxPerPost != nil ||
		input.ExternalInterstitial != nil || input.TrustedDomains != nil
}
