package options

import (
	"context"
	"strings"
	"unicode/utf8"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

// Wave 1 社区策略：注册/用户名/登录锁定/新人信任/维护模式/论坛阅读与行为开关。
// 与 site_identity 一样用 init 追加定义，避免继续膨胀 service.go。

const (
	usernameMinLengthMin  = 2
	usernameMinLengthMax  = 32
	usernameMaxLengthMin  = 2
	usernameMaxLengthMax  = 64
	loginMaxFailuresMin   = 0
	loginMaxFailuresMax   = 50
	loginLockoutMin       = 0
	loginLockoutMax       = 1440
	trustNewUserDaysMin   = 0
	trustNewUserDaysMax   = 365
	hotWindowDaysMin      = 1
	hotWindowDaysMax      = 90
	autoLockIdleDaysMin   = 0
	autoLockIdleDaysMax   = 3650
	mentionsMaxMin        = 0
	mentionsMaxMax        = 50
	maintenanceMsgMax     = 500
	reservedNamesMaxRunes = 2000
)

var (
	registrationModes = []string{"open", "invite", "approval", "closed"}
	usernameCharsets  = []string{"unicode_letters_numbers", "ascii"}
	guestReadModes    = []string{"public", "login_required"}
	listDefaultSorts  = []string{"latest", "active", "hot"}
	// warn 保留为历史兼容值，运行时按 off 处理（无独立 warn 合同）。
	duplicateTitlePolicies   = []string{"off", "warn", "block"}
	softDeleteVisibilities   = []string{"author_and_staff", "staff_only", "hidden"}
	recommendedReservedNames = "admin,administrator,system,sforum,root,support,moderator,mod,official,null,undefined"
)

func init() {
	optionDefinitions = append(optionDefinitions, communityPolicyOptionDefinitions()...)
}

func communityPolicyOptionDefinitions() []optionDefinition {
	site := identity.PermissionSettingsSiteManage
	forum := identity.PermissionForumSettingsManage
	return []optionDefinition{
		// 注册扩展：mode 为 public（登录页展示）；邮箱验证相关 public 便于注册 UX 提示。
		{name: NameIdentityRegistrationMode, public: true, managePermission: site},
		{name: NameIdentityRegistrationRequireEmailVerification, public: true, managePermission: site},
		{name: NameIdentityRegistrationBlockPostingUntilVerified, public: true, managePermission: site},
		{name: NameIdentityUsernameMinLength, public: true, managePermission: site},
		{name: NameIdentityUsernameMaxLength, public: true, managePermission: site},
		{name: NameIdentityUsernameCharset, public: true, managePermission: site},
		{name: NameIdentityUsernameReserved, public: false, managePermission: site},
		// 登录锁定：非 public，仅后端读取。
		{name: NameIdentityLoginMaxFailures, public: false, managePermission: site},
		{name: NameIdentityLoginLockoutMinutes, public: false, managePermission: site},
		// 新人信任：冷却/日限/外链对前端 composer 提示公开。
		{name: NameTrustNewUserDays, public: true, managePermission: forum},
		{name: NameTrustNewUserTopicCooldownSeconds, public: true, managePermission: forum},
		{name: NameTrustNewUserCommentCooldownSeconds, public: true, managePermission: forum},
		{name: NameTrustNewUserDailyTopicLimit, public: true, managePermission: forum},
		{name: NameTrustNewUserDailyCommentLimit, public: true, managePermission: forum},
		{name: NameTrustNewUserForbidOutboundLinks, public: true, managePermission: forum},
		{name: NameTrustNewUserForbidAttachments, public: true, managePermission: forum},
		// 维护模式：public，前台可展示横幅/拦截写操作提示。
		{name: NameSiteMaintenanceEnabled, public: true, managePermission: site},
		{name: NameSiteMaintenanceMessage, public: true, managePermission: site},
		// 论坛阅读与行为
		{name: NameForumGuestRead, public: true, managePermission: forum},
		{name: NameForumListDefaultSort, public: true, managePermission: forum},
		{name: NameForumListHotWindowDays, public: true, managePermission: forum},
		{name: NameForumTopicsAllowAuthorCloseReplies, public: true, managePermission: forum},
		{name: NameForumTopicsAllowAuthorDelete, public: true, managePermission: forum},
		{name: NameForumTopicsAutoLockIdleDays, public: true, managePermission: forum},
		{name: NameForumTopicsShowEditMark, public: true, managePermission: forum},
		{name: NameForumTopicsDuplicateTitlePolicy, public: true, managePermission: forum},
		{name: NameForumCommentsShowEditMark, public: true, managePermission: forum},
		{name: NameForumCommentsSoftDeleteVisibility, public: true, managePermission: forum},
		{name: NameForumMentionsEnabled, public: true, managePermission: forum},
		{name: NameForumMentionsMaxPerPost, public: true, managePermission: forum},
	}
}

func communityPolicyRecommendedDefaults() map[string]string {
	return map[string]string{
		NameIdentityRegistrationMode:                      "open",
		NameIdentityRegistrationRequireEmailVerification:  enabledOptionValue(false),
		NameIdentityRegistrationBlockPostingUntilVerified: enabledOptionValue(true),
		NameIdentityUsernameMinLength:                     "3",
		NameIdentityUsernameMaxLength:                     "20",
		NameIdentityUsernameCharset:                       "unicode_letters_numbers",
		NameIdentityUsernameReserved:                      recommendedReservedNames,
		NameIdentityLoginMaxFailures:                      "10",
		NameIdentityLoginLockoutMinutes:                   "15",
		NameTrustNewUserDays:                              "7",
		NameTrustNewUserTopicCooldownSeconds:              "300",
		NameTrustNewUserCommentCooldownSeconds:            "60",
		NameTrustNewUserDailyTopicLimit:                   "3",
		NameTrustNewUserDailyCommentLimit:                 "30",
		NameTrustNewUserForbidOutboundLinks:               enabledOptionValue(true),
		NameTrustNewUserForbidAttachments:                 enabledOptionValue(false),
		NameSiteMaintenanceEnabled:                        enabledOptionValue(false),
		NameSiteMaintenanceMessage:                        "",
		NameForumGuestRead:                                "public",
		NameForumListDefaultSort:                          "latest",
		NameForumListHotWindowDays:                        "7",
		NameForumTopicsAllowAuthorCloseReplies:            enabledOptionValue(true),
		NameForumTopicsAllowAuthorDelete:                  enabledOptionValue(true),
		NameForumTopicsAutoLockIdleDays:                   "0",
		NameForumTopicsShowEditMark:                       enabledOptionValue(true),
		NameForumTopicsDuplicateTitlePolicy:               "off",
		NameForumCommentsShowEditMark:                     enabledOptionValue(true),
		NameForumCommentsSoftDeleteVisibility:             "author_and_staff",
		NameForumMentionsEnabled:                          enabledOptionValue(true),
		NameForumMentionsMaxPerPost:                       "10",
	}
}

func mergeCommunityPolicyDefaults(values map[string]string) {
	for name, value := range communityPolicyRecommendedDefaults() {
		if _, exists := values[name]; !exists {
			values[name] = value
		}
	}
	values[NameMailWelcomeEnabled] = enabledOptionValue(false)
}

func coerceCommunityPolicyOptions(coerced, defaults map[string]string) {
	if value, ok := normalizeRegistrationMode(coerced[NameIdentityRegistrationMode]); ok {
		coerced[NameIdentityRegistrationMode] = value
	} else {
		coerced[NameIdentityRegistrationMode] = defaults[NameIdentityRegistrationMode]
	}
	// mode=closed 时同步 enabled=disabled，保持旧开关与 mode 一致。
	if coerced[NameIdentityRegistrationMode] == "closed" {
		coerced[NameIdentityRegistrationEnabled] = enabledOptionValue(false)
	} else if coerced[NameIdentityRegistrationMode] == "open" {
		// open 时若 enabled 未显式关闭则保持；若 enabled 关闭则 mode 也视为 closed 语义由 RegistrationEnabled 处理。
	}
	for _, name := range []string{
		NameIdentityRegistrationRequireEmailVerification,
		NameIdentityRegistrationBlockPostingUntilVerified,
		NameTrustNewUserForbidOutboundLinks,
		NameTrustNewUserForbidAttachments,
		NameSiteMaintenanceEnabled,
		NameForumTopicsAllowAuthorCloseReplies,
		NameForumTopicsAllowAuthorDelete,
		NameForumTopicsShowEditMark,
		NameForumCommentsShowEditMark,
		NameForumMentionsEnabled,
	} {
		if value, ok := normalizeEnabledOption(coerced[name]); ok {
			coerced[name] = value
		} else {
			coerced[name] = defaults[name]
		}
	}
	if value, ok := normalizeBoundedInt(coerced[NameIdentityUsernameMinLength], usernameMinLengthMin, usernameMinLengthMax); ok {
		coerced[NameIdentityUsernameMinLength] = value
	} else {
		coerced[NameIdentityUsernameMinLength] = defaults[NameIdentityUsernameMinLength]
	}
	if value, ok := normalizeBoundedInt(coerced[NameIdentityUsernameMaxLength], usernameMaxLengthMin, usernameMaxLengthMax); ok {
		coerced[NameIdentityUsernameMaxLength] = value
	} else {
		coerced[NameIdentityUsernameMaxLength] = defaults[NameIdentityUsernameMaxLength]
	}
	minLen, minOK := strictAtoi(coerced[NameIdentityUsernameMinLength])
	maxLen, maxOK := strictAtoi(coerced[NameIdentityUsernameMaxLength])
	if !minOK || !maxOK || maxLen < minLen {
		coerced[NameIdentityUsernameMinLength] = defaults[NameIdentityUsernameMinLength]
		coerced[NameIdentityUsernameMaxLength] = defaults[NameIdentityUsernameMaxLength]
	}
	if value, ok := normalizeChoice(coerced[NameIdentityUsernameCharset], usernameCharsets); ok {
		coerced[NameIdentityUsernameCharset] = value
	} else {
		coerced[NameIdentityUsernameCharset] = defaults[NameIdentityUsernameCharset]
	}
	if value, ok := normalizeReservedUsernames(coerced[NameIdentityUsernameReserved]); ok {
		coerced[NameIdentityUsernameReserved] = value
	} else {
		coerced[NameIdentityUsernameReserved] = defaults[NameIdentityUsernameReserved]
	}
	if value, ok := normalizeBoundedInt(coerced[NameIdentityLoginMaxFailures], loginMaxFailuresMin, loginMaxFailuresMax); ok {
		coerced[NameIdentityLoginMaxFailures] = value
	} else {
		coerced[NameIdentityLoginMaxFailures] = defaults[NameIdentityLoginMaxFailures]
	}
	if value, ok := normalizeBoundedInt(coerced[NameIdentityLoginLockoutMinutes], loginLockoutMin, loginLockoutMax); ok {
		coerced[NameIdentityLoginLockoutMinutes] = value
	} else {
		coerced[NameIdentityLoginLockoutMinutes] = defaults[NameIdentityLoginLockoutMinutes]
	}
	for name, bounds := range map[string][2]int{
		NameTrustNewUserDays:                   {trustNewUserDaysMin, trustNewUserDaysMax},
		NameTrustNewUserTopicCooldownSeconds:   {forumCooldownMin, forumCooldownMax},
		NameTrustNewUserCommentCooldownSeconds: {forumCooldownMin, forumCooldownMax},
		NameTrustNewUserDailyTopicLimit:        {forumDailyLimitMin, forumDailyLimitMax},
		NameTrustNewUserDailyCommentLimit:      {forumDailyLimitMin, forumDailyLimitMax},
		NameForumListHotWindowDays:             {hotWindowDaysMin, hotWindowDaysMax},
		NameForumTopicsAutoLockIdleDays:        {autoLockIdleDaysMin, autoLockIdleDaysMax},
		NameForumMentionsMaxPerPost:            {mentionsMaxMin, mentionsMaxMax},
	} {
		if value, ok := normalizeBoundedInt(coerced[name], bounds[0], bounds[1]); ok {
			coerced[name] = value
		} else {
			coerced[name] = defaults[name]
		}
	}
	if value, ok := normalizeMaintenanceMessage(coerced[NameSiteMaintenanceMessage]); ok {
		coerced[NameSiteMaintenanceMessage] = value
	} else {
		coerced[NameSiteMaintenanceMessage] = defaults[NameSiteMaintenanceMessage]
	}
	if value, ok := normalizeChoice(coerced[NameForumGuestRead], guestReadModes); ok {
		coerced[NameForumGuestRead] = value
	} else {
		coerced[NameForumGuestRead] = defaults[NameForumGuestRead]
	}
	if value, ok := normalizeChoice(coerced[NameForumListDefaultSort], listDefaultSorts); ok {
		coerced[NameForumListDefaultSort] = value
	} else {
		coerced[NameForumListDefaultSort] = defaults[NameForumListDefaultSort]
	}
	if value, ok := normalizeChoice(coerced[NameForumTopicsDuplicateTitlePolicy], duplicateTitlePolicies); ok {
		coerced[NameForumTopicsDuplicateTitlePolicy] = value
	} else {
		coerced[NameForumTopicsDuplicateTitlePolicy] = defaults[NameForumTopicsDuplicateTitlePolicy]
	}
	if value, ok := normalizeChoice(coerced[NameForumCommentsSoftDeleteVisibility], softDeleteVisibilities); ok {
		coerced[NameForumCommentsSoftDeleteVisibility] = value
	} else {
		coerced[NameForumCommentsSoftDeleteVisibility] = defaults[NameForumCommentsSoftDeleteVisibility]
	}
}

func isValidCommunityPolicyOptions(values map[string]string) bool {
	if _, ok := normalizeRegistrationMode(values[NameIdentityRegistrationMode]); !ok {
		return false
	}
	for _, name := range []string{
		NameIdentityRegistrationRequireEmailVerification,
		NameIdentityRegistrationBlockPostingUntilVerified,
		NameTrustNewUserForbidOutboundLinks,
		NameTrustNewUserForbidAttachments,
		NameSiteMaintenanceEnabled,
		NameForumTopicsAllowAuthorCloseReplies,
		NameForumTopicsAllowAuthorDelete,
		NameForumTopicsShowEditMark,
		NameForumCommentsShowEditMark,
		NameForumMentionsEnabled,
	} {
		if _, ok := normalizeEnabledOption(values[name]); !ok {
			return false
		}
	}
	minLen, minOK := parseBoundedInt(values[NameIdentityUsernameMinLength], usernameMinLengthMin, usernameMinLengthMax)
	maxLen, maxOK := parseBoundedInt(values[NameIdentityUsernameMaxLength], usernameMaxLengthMin, usernameMaxLengthMax)
	if !minOK || !maxOK || maxLen < minLen {
		return false
	}
	if _, ok := normalizeChoice(values[NameIdentityUsernameCharset], usernameCharsets); !ok {
		return false
	}
	if _, ok := normalizeReservedUsernames(values[NameIdentityUsernameReserved]); !ok {
		return false
	}
	if _, ok := parseBoundedInt(values[NameIdentityLoginMaxFailures], loginMaxFailuresMin, loginMaxFailuresMax); !ok {
		return false
	}
	if _, ok := parseBoundedInt(values[NameIdentityLoginLockoutMinutes], loginLockoutMin, loginLockoutMax); !ok {
		return false
	}
	for name, bounds := range map[string][2]int{
		NameTrustNewUserDays:                   {trustNewUserDaysMin, trustNewUserDaysMax},
		NameTrustNewUserTopicCooldownSeconds:   {forumCooldownMin, forumCooldownMax},
		NameTrustNewUserCommentCooldownSeconds: {forumCooldownMin, forumCooldownMax},
		NameTrustNewUserDailyTopicLimit:        {forumDailyLimitMin, forumDailyLimitMax},
		NameTrustNewUserDailyCommentLimit:      {forumDailyLimitMin, forumDailyLimitMax},
		NameForumListHotWindowDays:             {hotWindowDaysMin, hotWindowDaysMax},
		NameForumTopicsAutoLockIdleDays:        {autoLockIdleDaysMin, autoLockIdleDaysMax},
		NameForumMentionsMaxPerPost:            {mentionsMaxMin, mentionsMaxMax},
	} {
		if _, ok := parseBoundedInt(values[name], bounds[0], bounds[1]); !ok {
			return false
		}
	}
	if _, ok := normalizeMaintenanceMessage(values[NameSiteMaintenanceMessage]); !ok {
		return false
	}
	if _, ok := normalizeChoice(values[NameForumGuestRead], guestReadModes); !ok {
		return false
	}
	if _, ok := normalizeChoice(values[NameForumListDefaultSort], listDefaultSorts); !ok {
		return false
	}
	if _, ok := normalizeChoice(values[NameForumTopicsDuplicateTitlePolicy], duplicateTitlePolicies); !ok {
		return false
	}
	if _, ok := normalizeChoice(values[NameForumCommentsSoftDeleteVisibility], softDeleteVisibilities); !ok {
		return false
	}
	return true
}

func normalizeCommunityPolicyOption(name, value string) (string, bool) {
	value = strings.TrimSpace(value)
	switch name {
	case NameIdentityRegistrationMode:
		return normalizeRegistrationMode(value)
	case NameIdentityRegistrationRequireEmailVerification,
		NameIdentityRegistrationBlockPostingUntilVerified,
		NameTrustNewUserForbidOutboundLinks,
		NameTrustNewUserForbidAttachments,
		NameSiteMaintenanceEnabled,
		NameForumTopicsAllowAuthorCloseReplies,
		NameForumTopicsAllowAuthorDelete,
		NameForumTopicsShowEditMark,
		NameForumCommentsShowEditMark,
		NameForumMentionsEnabled:
		return normalizeEnabledOption(value)
	case NameIdentityUsernameMinLength:
		return normalizeBoundedInt(value, usernameMinLengthMin, usernameMinLengthMax)
	case NameIdentityUsernameMaxLength:
		return normalizeBoundedInt(value, usernameMaxLengthMin, usernameMaxLengthMax)
	case NameIdentityUsernameCharset:
		return normalizeChoice(value, usernameCharsets)
	case NameIdentityUsernameReserved:
		return normalizeReservedUsernames(value)
	case NameIdentityLoginMaxFailures:
		return normalizeBoundedInt(value, loginMaxFailuresMin, loginMaxFailuresMax)
	case NameIdentityLoginLockoutMinutes:
		return normalizeBoundedInt(value, loginLockoutMin, loginLockoutMax)
	case NameTrustNewUserDays:
		return normalizeBoundedInt(value, trustNewUserDaysMin, trustNewUserDaysMax)
	case NameTrustNewUserTopicCooldownSeconds, NameTrustNewUserCommentCooldownSeconds:
		return normalizeBoundedInt(value, forumCooldownMin, forumCooldownMax)
	case NameTrustNewUserDailyTopicLimit, NameTrustNewUserDailyCommentLimit:
		return normalizeBoundedInt(value, forumDailyLimitMin, forumDailyLimitMax)
	case NameSiteMaintenanceMessage:
		return normalizeMaintenanceMessage(value)
	case NameForumGuestRead:
		return normalizeChoice(value, guestReadModes)
	case NameForumListDefaultSort:
		return normalizeChoice(value, listDefaultSorts)
	case NameForumListHotWindowDays:
		return normalizeBoundedInt(value, hotWindowDaysMin, hotWindowDaysMax)
	case NameForumTopicsAutoLockIdleDays:
		return normalizeBoundedInt(value, autoLockIdleDaysMin, autoLockIdleDaysMax)
	case NameForumTopicsDuplicateTitlePolicy:
		return normalizeChoice(value, duplicateTitlePolicies)
	case NameForumCommentsSoftDeleteVisibility:
		return normalizeChoice(value, softDeleteVisibilities)
	case NameForumMentionsMaxPerPost:
		return normalizeBoundedInt(value, mentionsMaxMin, mentionsMaxMax)
	default:
		return "", false
	}
}

func normalizeRegistrationMode(value string) (string, bool) {
	return normalizeChoice(value, registrationModes)
}

func normalizeMaintenanceMessage(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if utf8.RuneCountInString(value) > maintenanceMsgMax {
		return "", false
	}
	return value, true
}

func normalizeReservedUsernames(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if utf8.RuneCountInString(value) > reservedNamesMaxRunes {
		return "", false
	}
	if value == "" {
		return "", true
	}
	parts := strings.Split(value, ",")
	cleaned := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		name := strings.ToLower(strings.TrimSpace(part))
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		cleaned = append(cleaned, name)
	}
	return strings.Join(cleaned, ","), true
}

func (s *Service) UsernamePolicy(ctx context.Context) (identity.UsernamePolicy, error) {
	values, err := s.loadMap(ctx)
	if err != nil {
		return identity.UsernamePolicy{}, err
	}
	minLen, _ := strictAtoi(values[NameIdentityUsernameMinLength])
	maxLen, _ := strictAtoi(values[NameIdentityUsernameMaxLength])
	reserved := []string{}
	if raw := strings.TrimSpace(values[NameIdentityUsernameReserved]); raw != "" {
		reserved = strings.Split(raw, ",")
	}
	return identity.UsernamePolicy{
		MinLength: minLen,
		MaxLength: maxLen,
		Charset:   values[NameIdentityUsernameCharset],
		Reserved:  reserved,
	}, nil
}

func (s *Service) LoginLockoutPolicy(ctx context.Context) (identity.LoginLockoutPolicy, error) {
	values, err := s.loadMap(ctx)
	if err != nil {
		return identity.LoginLockoutPolicy{}, err
	}
	maxFailures, _ := strictAtoi(values[NameIdentityLoginMaxFailures])
	lockoutMinutes, _ := strictAtoi(values[NameIdentityLoginLockoutMinutes])
	return identity.LoginLockoutPolicy{MaxFailures: maxFailures, LockoutMinutes: lockoutMinutes}, nil
}

// TrustPolicy 新人信任阶梯。
type TrustPolicy struct {
	NewUserDays            int
	TopicCooldownSeconds   int
	CommentCooldownSeconds int
	DailyTopicLimit        int
	DailyCommentLimit      int
	ForbidOutboundLinks    bool
	ForbidAttachments      bool
}

func (s *Service) TrustPolicy(ctx context.Context) (TrustPolicy, error) {
	values, err := s.loadMap(ctx)
	if err != nil {
		return TrustPolicy{}, err
	}
	days, _ := strictAtoi(values[NameTrustNewUserDays])
	topicCooldown, _ := strictAtoi(values[NameTrustNewUserTopicCooldownSeconds])
	commentCooldown, _ := strictAtoi(values[NameTrustNewUserCommentCooldownSeconds])
	dailyTopic, _ := strictAtoi(values[NameTrustNewUserDailyTopicLimit])
	dailyComment, _ := strictAtoi(values[NameTrustNewUserDailyCommentLimit])
	return TrustPolicy{
		NewUserDays:            days,
		TopicCooldownSeconds:   topicCooldown,
		CommentCooldownSeconds: commentCooldown,
		DailyTopicLimit:        dailyTopic,
		DailyCommentLimit:      dailyComment,
		ForbidOutboundLinks:    isEnabledOption(values[NameTrustNewUserForbidOutboundLinks]),
		ForbidAttachments:      isEnabledOption(values[NameTrustNewUserForbidAttachments]),
	}, nil
}

// MaintenancePolicy 站点维护模式。
type MaintenancePolicy struct {
	Enabled bool
	Message string
}

func (s *Service) MaintenancePolicy(ctx context.Context) (MaintenancePolicy, error) {
	values, err := s.loadMap(ctx)
	if err != nil {
		return MaintenancePolicy{}, err
	}
	return MaintenancePolicy{
		Enabled: isEnabledOption(values[NameSiteMaintenanceEnabled]),
		Message: values[NameSiteMaintenanceMessage],
	}, nil
}

// RegistrationMode 返回 open|invite|approval|closed。
func (s *Service) RegistrationMode(ctx context.Context) (string, error) {
	values, err := s.loadMap(ctx)
	if err != nil {
		return "open", err
	}
	if mode, ok := normalizeRegistrationMode(values[NameIdentityRegistrationMode]); ok {
		return mode, nil
	}
	return "open", nil
}
