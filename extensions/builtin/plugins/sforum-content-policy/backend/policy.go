package main

import (
	"os"
	"strconv"
	"strings"
)

// 动作模式：reject 拒绝发布；tag 仅对主题追加强制标签。
const (
	modeReject = "reject"
	modeTag    = "tag"
)

// policyConfig 来自宿主注入的 SFORUM_SETTING_* 环境变量。
type policyConfig struct {
	Enabled       bool
	Keywords      []string
	Mode          string
	ForceTag      string
	MatchTitle    bool
	MatchContent  bool
	CaseSensitive bool
}

func loadPolicyConfigFromEnv() policyConfig {
	return policyConfig{
		Enabled:       envBool("SFORUM_SETTING_ENABLED", true),
		Keywords:      parseKeywords(os.Getenv("SFORUM_SETTING_KEYWORDS")),
		Mode:          normalizeMode(os.Getenv("SFORUM_SETTING_MODE")),
		ForceTag:      strings.TrimSpace(os.Getenv("SFORUM_SETTING_FORCE_TAG")),
		MatchTitle:    envBool("SFORUM_SETTING_MATCH_TITLE", true),
		MatchContent:  envBool("SFORUM_SETTING_MATCH_CONTENT", true),
		CaseSensitive: envBool("SFORUM_SETTING_CASE_SENSITIVE", false),
	}
}

func envBool(key string, defaultValue bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultValue
	}
	// 兼容 boolean 设置的 true/false/1/0。
	if v, err := strconv.ParseBool(raw); err == nil {
		return v
	}
	switch strings.ToLower(raw) {
	case "yes", "on", "y":
		return true
	case "no", "off", "n":
		return false
	default:
		return defaultValue
	}
}

func normalizeMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case modeTag:
		return modeTag
	default:
		return modeReject
	}
}

// parseKeywords 支持换行或逗号分隔；# 行注释；空白忽略。
func parseKeywords(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	// 统一分隔符：逗号 → 换行，再按行切。
	normalized := strings.ReplaceAll(raw, ",", "\n")
	lines := strings.Split(normalized, "\n")
	out := make([]string, 0, len(lines))
	seen := map[string]bool{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// 去重时用原样 key，匹配时再按 case 规则处理。
		if seen[line] {
			continue
		}
		seen[line] = true
		out = append(out, line)
	}
	return out
}

// findKeyword 在 haystack 中查找第一个命中关键词；未命中返回空串。
func findKeyword(haystack string, keywords []string, caseSensitive bool) string {
	if haystack == "" || len(keywords) == 0 {
		return ""
	}
	search := haystack
	if !caseSensitive {
		search = strings.ToLower(search)
	}
	for _, kw := range keywords {
		needle := kw
		if !caseSensitive {
			needle = strings.ToLower(needle)
		}
		if needle == "" {
			continue
		}
		if strings.Contains(search, needle) {
			return kw
		}
	}
	return ""
}

// policyDecision 是 filter 决策结果（不依赖 go-plugin 类型，便于单测）。
type policyDecision struct {
	OK      bool
	Reason  string
	Message string
	// PatchTag 非空时表示应对主题 tagSlugs 追加该标签（mode=tag）。
	PatchTag string
}

// evaluateContent 对 title/content 做关键词扫描。
// eventName 用于区分主题 vs 评论（tag 模式对评论仍 reject）。
func evaluateContent(cfg policyConfig, eventName, title, content string) policyDecision {
	if !cfg.Enabled {
		return policyDecision{OK: true}
	}
	if len(cfg.Keywords) == 0 {
		return policyDecision{OK: true}
	}

	var parts []string
	if cfg.MatchTitle && strings.TrimSpace(title) != "" {
		parts = append(parts, title)
	}
	if cfg.MatchContent && strings.TrimSpace(content) != "" {
		parts = append(parts, content)
	}
	if len(parts) == 0 {
		return policyDecision{OK: true}
	}

	matched := findKeyword(strings.Join(parts, "\n"), cfg.Keywords, cfg.CaseSensitive)
	if matched == "" {
		return policyDecision{OK: true}
	}

	// 评论：tag 模式无 patch 字段，统一拒绝。
	isTopic := eventName == "topic.before_create" || eventName == "topic.before_update"
	if cfg.Mode == modeTag && isTopic {
		tag := strings.TrimSpace(cfg.ForceTag)
		if tag == "" {
			// 配置不完整时 fail closed，避免 silent pass。
			return policyDecision{
				OK:      false,
				Reason:  "content_policy.force_tag_missing",
				Message: "Content policy is set to force-tag mode but force_tag is empty.",
			}
		}
		return policyDecision{
			OK:       true,
			Reason:   "content_policy.tagged",
			Message:  "Keyword matched; force tag will be applied.",
			PatchTag: tag,
		}
	}

	return policyDecision{
		OK:      false,
		Reason:  "content_policy.keyword_blocked",
		Message: "Content matches a blocked keyword.",
	}
}

// mergeTagSlugs 将 forceTag 追加到 payload 中的 tagSlugs（去重、保序）。
func mergeTagSlugs(existing any, forceTag string) []string {
	forceTag = strings.TrimSpace(forceTag)
	out := make([]string, 0, 8)
	seen := map[string]bool{}
	appendOne := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		out = append(out, s)
	}
	switch v := existing.(type) {
	case []string:
		for _, s := range v {
			appendOne(s)
		}
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				appendOne(s)
			}
		}
	}
	appendOne(forceTag)
	return out
}

func payloadString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return ""
	}
	switch v := raw.(type) {
	case string:
		return v
	case map[string]any:
		// Host 经 Protocol V2 规范化后，content 多为 ContentInput JSON 对象。
		// 关键词扫描优先 plainText（无标记），否则回退 rawContent。
		if plain, ok := v["plainText"].(string); ok && strings.TrimSpace(plain) != "" {
			return plain
		}
		if rawContent, ok := v["rawContent"].(string); ok {
			return rawContent
		}
		return ""
	default:
		return ""
	}
}
