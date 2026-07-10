package moderation

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	ModeOff   = "off"
	ModeRules = "rules"
	ModeAll   = "all"

	TriggerAllContent   = "all_content"
	TriggerNewUser      = "new_user"
	TriggerExternalLink = "external_link"
)

var (
	ErrSettingsInvalid = errors.New("moderation: invalid settings")
	httpURLPattern     = regexp.MustCompile(`https?://[^\s<>()]+`)
)

type Settings struct {
	Mode                string    `json:"mode"`
	ReviewNewUsers      bool      `json:"reviewNewUsers"`
	NewUserMaxAgeDays   int       `json:"newUserMaxAgeDays"`
	ReviewExternalLinks bool      `json:"reviewExternalLinks"`
	UpdatedByUserID     *int64    `json:"updatedByUserId,omitempty"`
	UpdatedAt           time.Time `json:"updatedAt,omitempty"`
}

type PublicationInput struct {
	Now           time.Time
	UserCreatedAt time.Time
	RawContent    string
	SiteURL       string
}

type PublicationDecision struct {
	Pending  bool     `json:"pending"`
	Triggers []string `json:"triggers"`
}

type SettingsStore interface {
	GetSettings(ctx context.Context) (Settings, error)
	SaveSettings(ctx context.Context, settings Settings, actorUserID int64) (Settings, error)
	ResetSettings(ctx context.Context, settings Settings, actorUserID int64) (Settings, error)
}

func RecommendedSettings() Settings {
	return Settings{
		Mode:                ModeOff,
		ReviewNewUsers:      true,
		NewUserMaxAgeDays:   7,
		ReviewExternalLinks: true,
	}
}

func (settings Settings) Validate() error {
	if settings.Mode != ModeOff && settings.Mode != ModeRules && settings.Mode != ModeAll {
		return ErrSettingsInvalid
	}
	if settings.NewUserMaxAgeDays < 0 || settings.NewUserMaxAgeDays > 3650 {
		return ErrSettingsInvalid
	}
	return nil
}

func (settings Settings) Evaluate(input PublicationInput) PublicationDecision {
	switch settings.Mode {
	case ModeAll:
		return PublicationDecision{Pending: true, Triggers: []string{TriggerAllContent}}
	case ModeRules:
		triggers := make([]string, 0, 2)
		if settings.ReviewNewUsers && isNewUser(input, settings.NewUserMaxAgeDays) {
			triggers = append(triggers, TriggerNewUser)
		}
		if settings.ReviewExternalLinks && containsExternalLink(input.RawContent, input.SiteURL) {
			triggers = append(triggers, TriggerExternalLink)
		}
		return PublicationDecision{Pending: len(triggers) > 0, Triggers: triggers}
	default:
		return PublicationDecision{Triggers: []string{}}
	}
}

func isNewUser(input PublicationInput, maxAgeDays int) bool {
	if input.UserCreatedAt.IsZero() || maxAgeDays <= 0 {
		return false
	}
	now := input.Now
	if now.IsZero() {
		now = time.Now()
	}
	return input.UserCreatedAt.After(now.Add(-time.Duration(maxAgeDays) * 24 * time.Hour))
}

func containsExternalLink(content, siteURL string) bool {
	site, err := url.Parse(strings.TrimSpace(siteURL))
	if err != nil || site.Hostname() == "" {
		return len(httpURLPattern.FindAllString(content, -1)) > 0
	}
	siteHost := strings.ToLower(site.Hostname())
	for _, raw := range httpURLPattern.FindAllString(content, -1) {
		candidate, err := url.Parse(raw)
		if err == nil && candidate.Hostname() != "" && strings.ToLower(candidate.Hostname()) != siteHost {
			return true
		}
	}
	return false
}
