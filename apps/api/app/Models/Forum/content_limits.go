package forum

import (
	"strings"
	"time"
	"unicode/utf8"
)

// 论坛内容限制硬边界：管理员可配范围，防止极端配置拖垮存储或列表。
const (
	HardTitleMinRunes     = 1
	HardTitleMaxRunes     = 200
	HardContentMinRunes   = 0
	HardContentMaxRunes   = 200000
	HardCommentMinRunes   = 0
	HardCommentMaxRunes   = 50000
	HardNestingMin        = 0
	HardNestingMax        = 20
	HardEditWindowMaxMin  = 10080 // 7 天
	HardCooldownMaxSec     = 86400 // 24 小时
	HardDailyLimitMax     = 10000
	HardExcerptMinRunes   = 40
	HardExcerptMaxRunes   = 500
	HardTagMinPerTopic    = 0
	HardTagMaxPerTopic    = 10
)

// Recommended* 与 defaultForumSettings / web_options 推荐值保持一致。
const (
	RecommendedTopicTitleMinRunes       = 2
	RecommendedTopicTitleMaxRunes       = 100
	RecommendedTopicContentMinRunes     = 0
	RecommendedTopicContentMaxRunes     = 50000
	RecommendedTopicEditWindowMinutes   = 0
	RecommendedTopicCooldownSeconds      = 0
	RecommendedDailyTopicLimit          = 0
	RecommendedCommentMinRunes          = 1
	RecommendedCommentMaxRunes          = 10000
	RecommendedCommentMaxNestingDepth   = 5
	RecommendedCommentEditWindowMinutes = 0
	RecommendedCommentCooldownSeconds     = 0
	RecommendedDailyCommentLimit        = 0
	RecommendedExcerptRuneLimit         = 180
	RecommendedTagMinPerTopic           = 0
	RecommendedTagMaxPerTopic           = 5
)

func defaultForumSettings() ForumSettings {
	return ForumSettings{
		DefaultCategorySlug:          "general",
		TagCreationMode:              TagCreationModeControlled,
		TagPublicPages:               true,
		TagMinPerTopic:               RecommendedTagMinPerTopic,
		TagMaxPerTopic:               RecommendedTagMaxPerTopic,
		TopicsPerPage:                20,
		CommentsPerPage:              20,
		TopicTitleMinRunes:           RecommendedTopicTitleMinRunes,
		TopicTitleMaxRunes:           RecommendedTopicTitleMaxRunes,
		TopicContentMinRunes:         RecommendedTopicContentMinRunes,
		TopicContentMaxRunes:         RecommendedTopicContentMaxRunes,
		TopicEditWindowMinutes:       RecommendedTopicEditWindowMinutes,
		TopicCooldownSeconds:          RecommendedTopicCooldownSeconds,
		DailyTopicLimit:              RecommendedDailyTopicLimit,
		CommentMinRunes:              RecommendedCommentMinRunes,
		CommentMaxRunes:              RecommendedCommentMaxRunes,
		CommentMaxNestingDepth:       RecommendedCommentMaxNestingDepth,
		CommentEditWindowMinutes:     RecommendedCommentEditWindowMinutes,
		CommentCooldownSeconds:         RecommendedCommentCooldownSeconds,
		DailyCommentLimit:            RecommendedDailyCommentLimit,
		ExcerptRuneLimit:             RecommendedExcerptRuneLimit,
		GuestRead:                    "public",
		ListDefaultSort:              "latest",
		ListHotWindowDays:            7,
		AllowAuthorCloseReplies:      true,
		AllowAuthorDelete:            true,
		AutoLockIdleDays:             0,
		ShowTopicEditMark:            true,
		DuplicateTitlePolicy:         "warn",
		ShowCommentEditMark:          true,
		SoftDeleteVisibility:         "author_and_staff",
		MentionsEnabled:              true,
		MentionsMaxPerPost:           10,
	}
}

func runeCount(value string) int {
	return utf8.RuneCountInString(value)
}

func validateTopicTitle(title string, settings ForumSettings) error {
	count := runeCount(title)
	if count < settings.TopicTitleMinRunes {
		return ErrTitleTooShort
	}
	if settings.TopicTitleMaxRunes > 0 && count > settings.TopicTitleMaxRunes {
		return ErrTitleTooLong
	}
	return nil
}

// validateTopicContent 按编辑器原文 rune 计数，与用户所见字数一致。
func validateTopicContent(raw string, settings ForumSettings) error {
	count := runeCount(strings.TrimSpace(raw))
	if count < settings.TopicContentMinRunes {
		return ErrContentTooShort
	}
	if settings.TopicContentMaxRunes > 0 && count > settings.TopicContentMaxRunes {
		return ErrContentTooLong
	}
	return nil
}

func validateCommentContent(raw string, settings ForumSettings) error {
	count := runeCount(strings.TrimSpace(raw))
	if count < settings.CommentMinRunes {
		return ErrCommentTooShort
	}
	if settings.CommentMaxRunes > 0 && count > settings.CommentMaxRunes {
		return ErrCommentTooLong
	}
	return nil
}

func validateTagCount(count int, settings ForumSettings) error {
	if count < settings.TagMinPerTopic {
		return ErrTagMinRequired
	}
	if count > settings.TagMaxPerTopic {
		return ErrInvalidTag
	}
	return nil
}

func validateCommentNesting(parentDepth int, settings ForumSettings) error {
	// parentDepth 为父评论深度；新评论 depth = parentDepth+1（根评论 parent 为 nil 时 depth=0）。
	// maxNestingDepth=0 表示只允许根评论；=5 表示 depth 最大为 5。
	nextDepth := 0
	if parentDepth >= 0 {
		nextDepth = parentDepth + 1
	}
	if nextDepth > settings.CommentMaxNestingDepth {
		return ErrCommentNestingDeep
	}
	return nil
}

// withinEditWindow：windowMinutes=0 表示不限时；moderator/any 权限由调用方绕过。
func withinEditWindow(createdAt time.Time, windowMinutes int, now time.Time) bool {
	if windowMinutes <= 0 {
		return true
	}
	if createdAt.IsZero() {
		return true
	}
	deadline := createdAt.Add(time.Duration(windowMinutes) * time.Minute)
	return !now.After(deadline)
}

func cooldownElapsed(lastAt time.Time, ok bool, cooldownSeconds int, now time.Time) bool {
	if cooldownSeconds <= 0 || !ok {
		return true
	}
	return !now.Before(lastAt.Add(time.Duration(cooldownSeconds) * time.Second))
}

func dayStartUTC(now time.Time) time.Time {
	y, m, d := now.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func validForumContentLimits(settings ForumSettings) bool {
	if settings.TagMinPerTopic < HardTagMinPerTopic || settings.TagMinPerTopic > HardTagMaxPerTopic {
		return false
	}
	if settings.TagMaxPerTopic < HardTagMinPerTopic || settings.TagMaxPerTopic > HardTagMaxPerTopic {
		return false
	}
	if settings.TagMinPerTopic > settings.TagMaxPerTopic {
		return false
	}
	if settings.TopicTitleMinRunes < HardTitleMinRunes || settings.TopicTitleMinRunes > HardTitleMaxRunes {
		return false
	}
	if settings.TopicTitleMaxRunes < HardTitleMinRunes || settings.TopicTitleMaxRunes > HardTitleMaxRunes {
		return false
	}
	if settings.TopicTitleMinRunes > settings.TopicTitleMaxRunes {
		return false
	}
	if settings.TopicContentMinRunes < HardContentMinRunes || settings.TopicContentMinRunes > HardContentMaxRunes {
		return false
	}
	if settings.TopicContentMaxRunes < 1 || settings.TopicContentMaxRunes > HardContentMaxRunes {
		return false
	}
	if settings.TopicContentMinRunes > settings.TopicContentMaxRunes {
		return false
	}
	if settings.CommentMinRunes < HardCommentMinRunes || settings.CommentMinRunes > HardCommentMaxRunes {
		return false
	}
	if settings.CommentMaxRunes < 1 || settings.CommentMaxRunes > HardCommentMaxRunes {
		return false
	}
	if settings.CommentMinRunes > settings.CommentMaxRunes {
		return false
	}
	if settings.CommentMaxNestingDepth < HardNestingMin || settings.CommentMaxNestingDepth > HardNestingMax {
		return false
	}
	if settings.TopicEditWindowMinutes < 0 || settings.TopicEditWindowMinutes > HardEditWindowMaxMin {
		return false
	}
	if settings.CommentEditWindowMinutes < 0 || settings.CommentEditWindowMinutes > HardEditWindowMaxMin {
		return false
	}
	if settings.TopicCooldownSeconds < 0 || settings.TopicCooldownSeconds > HardCooldownMaxSec {
		return false
	}
	if settings.CommentCooldownSeconds < 0 || settings.CommentCooldownSeconds > HardCooldownMaxSec {
		return false
	}
	if settings.DailyTopicLimit < 0 || settings.DailyTopicLimit > HardDailyLimitMax {
		return false
	}
	if settings.DailyCommentLimit < 0 || settings.DailyCommentLimit > HardDailyLimitMax {
		return false
	}
	if settings.ExcerptRuneLimit < HardExcerptMinRunes || settings.ExcerptRuneLimit > HardExcerptMaxRunes {
		return false
	}
	switch settings.GuestRead {
	case "public", "login_required", "":
	default:
		return false
	}
	switch settings.ListDefaultSort {
	case "latest", "active", "hot", "":
	default:
		return false
	}
	if settings.ListHotWindowDays < 0 || settings.ListHotWindowDays > 90 {
		return false
	}
	if settings.AutoLockIdleDays < 0 || settings.AutoLockIdleDays > 3650 {
		return false
	}
	switch settings.DuplicateTitlePolicy {
	case "off", "warn", "block", "":
	default:
		return false
	}
	switch settings.SoftDeleteVisibility {
	case "author_and_staff", "staff_only", "hidden", "":
	default:
		return false
	}
	if settings.MentionsMaxPerPost < 0 || settings.MentionsMaxPerPost > 50 {
		return false
	}
	return true
}
