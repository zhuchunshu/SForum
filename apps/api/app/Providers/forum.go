package providers

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"

	forumcontroller "github.com/zhuchunshu/sforum/apps/api/app/Http/Controllers/Forum"
	forum "github.com/zhuchunshu/sforum/apps/api/app/Models/Forum"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	options "github.com/zhuchunshu/sforum/apps/api/app/Models/Options"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
)

type ForumProvider struct {
	controller *forumcontroller.Controller
}

func NewForumProvider(store forum.Store, users identity.ActorStore, sessions *authsession.Manager) *ForumProvider {
	return NewForumProviderWithEvents(store, users, sessions, nil)
}

func NewForumProviderWithEvents(store forum.Store, users identity.ActorStore, sessions *authsession.Manager, publisher appevents.Publisher) *ForumProvider {
	return NewForumProviderWithOptionsAndEvents(store, nil, users, sessions, publisher)
}

func NewForumProviderWithOptionsAndEvents(store forum.Store, optionsService *options.Service, users identity.ActorStore, sessions *authsession.Manager, publisher appevents.Publisher) *ForumProvider {
	return &ForumProvider{
		controller: forumcontroller.NewController(forum.NewServiceWithSettingsAndEvents(store, ForumSettingsResolver{options: optionsService}, publisher), users, sessions),
	}
}

func (p *ForumProvider) RegisterRoutes(api fiber.Router) {
	p.controller.RegisterRoutes(api)
}

type ForumSettingsResolver struct {
	options *options.Service
}

func (r ForumSettingsResolver) ForumSettings(ctx context.Context) (forum.ForumSettings, error) {
	settings := recommendedForumSettings()
	if r.options == nil {
		return settings, nil
	}
	if value, err := r.options.WebOption(ctx, options.NameForumDefaultCategorySlug); err == nil {
		if slug, ok := normalizeForumSlug(value); ok {
			settings.DefaultCategorySlug = slug
		}
	}
	if value, err := r.options.WebOption(ctx, options.NameForumTagCreationMode); err == nil {
		if mode, ok := normalizeForumTagCreationMode(value); ok {
			settings.TagCreationMode = mode
		}
	}
	if value, err := r.options.WebOption(ctx, options.NameForumTagPublicPages); err == nil {
		if enabled, ok := normalizeForumEnabled(value); ok {
			settings.TagPublicPages = enabled
		}
	}
	if value, err := r.options.WebOption(ctx, options.NameForumTagMaxPerTopic); err == nil {
		if max, ok := normalizeForumMaxTags(value); ok {
			settings.TagMaxPerTopic = max
		}
	}
	return settings, nil
}

func (r ForumSettingsResolver) UpdateForumSettings(ctx context.Context, actor identity.Actor, input forum.UpdateForumSettingsInput) (forum.ForumSettings, error) {
	if r.options == nil {
		return forum.ForumSettings{}, forum.ErrInvalidSettings
	}
	updates := []options.UpdateInput{}
	if input.DefaultCategorySlug != nil {
		updates = append(updates, options.UpdateInput{Name: options.NameForumDefaultCategorySlug, Value: *input.DefaultCategorySlug})
	}
	if input.TagCreationMode != nil {
		updates = append(updates, options.UpdateInput{Name: options.NameForumTagCreationMode, Value: *input.TagCreationMode})
	}
	if input.TagPublicPages != nil {
		updates = append(updates, options.UpdateInput{Name: options.NameForumTagPublicPages, Value: enabledOptionValue(*input.TagPublicPages)})
	}
	if input.TagMaxPerTopic != nil {
		updates = append(updates, options.UpdateInput{Name: options.NameForumTagMaxPerTopic, Value: strconv.Itoa(*input.TagMaxPerTopic)})
	}
	if len(updates) > 0 {
		if _, err := r.options.UpdateMany(ctx, actor, updates); err != nil {
			return forum.ForumSettings{}, err
		}
	}
	return r.ForumSettings(ctx)
}

func (r ForumSettingsResolver) ResetForumSettings(ctx context.Context, actor identity.Actor) (forum.ForumSettings, error) {
	input := forum.UpdateForumSettingsInput{}
	if actor.Can(identity.PermissionCategoryManage) {
		value := "general"
		input.DefaultCategorySlug = &value
	}
	if actor.Can(identity.PermissionTagManage) {
		mode := forum.TagCreationModeControlled
		publicPages := true
		maxTags := 5
		input.TagCreationMode = &mode
		input.TagPublicPages = &publicPages
		input.TagMaxPerTopic = &maxTags
	}
	return r.UpdateForumSettings(ctx, actor, input)
}

func recommendedForumSettings() forum.ForumSettings {
	return forum.ForumSettings{
		DefaultCategorySlug: "general",
		TagCreationMode:     forum.TagCreationModeControlled,
		TagPublicPages:      true,
		TagMaxPerTopic:      5,
	}
}

var providerForumSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func normalizeForumSlug(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	return value, providerForumSlugPattern.MatchString(value)
}

func normalizeForumTagCreationMode(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case forum.TagCreationModeControlled, forum.TagCreationModeReview, forum.TagCreationModeOpen:
		return value, true
	default:
		return "", false
	}
}

func normalizeForumEnabled(value string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "enabled", "true", "1", "yes", "on":
		return true, true
	case "disabled", "false", "0", "no", "off":
		return false, true
	default:
		return false, false
	}
}

func enabledOptionValue(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}

func normalizeForumMaxTags(value string) (int, bool) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	return parsed, err == nil && parsed >= 0 && parsed <= 10
}
