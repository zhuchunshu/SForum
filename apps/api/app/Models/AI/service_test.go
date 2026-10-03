package ai_test

import (
	"context"
	"errors"
	"testing"
	"time"

	modelsai "github.com/zhuchunshu/sforum/apps/api/app/Models/AI"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	supportai "github.com/zhuchunshu/sforum/apps/api/app/Support/AI"
)

type stubSettings struct {
	stored    supportai.Settings
	saves     int
	resets    int
	lastActor int64
}

func (s *stubSettings) GetSettings(context.Context) (supportai.Settings, error) { return s.stored, nil }
func (s *stubSettings) SaveSettings(_ context.Context, settings supportai.Settings, actorUserID int64) (supportai.Settings, error) {
	s.saves++
	s.lastActor = actorUserID
	s.stored = settings
	return settings, nil
}
func (s *stubSettings) ResetSettings(_ context.Context, settings supportai.Settings, actorUserID int64) (supportai.Settings, error) {
	s.resets++
	s.stored = settings
	return settings, nil
}

func managerActor() identity.Actor {
	return identity.Actor{
		ID:          42,
		Status:      identity.UserStatusActive,
		Permissions: map[string]bool{identity.PermissionAIManage: true},
	}
}

func memberActor() identity.Actor {
	return identity.Actor{ID: 7, Status: identity.UserStatusActive, Permissions: map[string]bool{}}
}

func TestServiceRequiresManagePermission(t *testing.T) {
	store := &stubSettings{stored: supportai.RecommendedSettings()}
	service := modelsai.NewService(modelsai.Config{Settings: store})
	ctx := context.Background()

	if _, err := service.GetSettings(ctx, memberActor()); !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("get settings: %v", err)
	}
	if _, err := service.UpdateSettings(ctx, memberActor(), supportai.RecommendedSettings()); !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("update settings: %v", err)
	}
	if _, err := service.ResetSettings(ctx, memberActor()); !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("reset settings: %v", err)
	}
	if _, err := service.Usage(ctx, memberActor(), time.Now()); !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("usage: %v", err)
	}
	if _, err := service.Executions(ctx, memberActor(), 10); !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("executions: %v", err)
	}
	if store.saves != 0 || store.resets != 0 {
		t.Fatal("denied callers must not reach the store")
	}
}

func TestServicePersistsValidatedSettingsWithTheActor(t *testing.T) {
	store := &stubSettings{stored: supportai.RecommendedSettings()}
	service := modelsai.NewService(modelsai.Config{Settings: store})
	settings := supportai.RecommendedSettings()
	settings.Enabled = true
	settings.Profiles[0].Enabled = true

	saved, err := service.UpdateSettings(context.Background(), managerActor(), settings)
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if !saved.Enabled || store.lastActor != 42 {
		t.Fatalf("saved = %+v actor = %d", saved, store.lastActor)
	}
}

func TestServiceRejectsInvalidSettingsBeforeWriting(t *testing.T) {
	store := &stubSettings{stored: supportai.RecommendedSettings()}
	service := modelsai.NewService(modelsai.Config{Settings: store})
	settings := supportai.RecommendedSettings()
	settings.Enabled = true
	// 越界值才是真正非法的：天花板不可调，必须拒绝。
	settings.Gates.ExtensionDailyQuota = supportai.MaxExtensionDailyQuota + 1

	if _, err := service.UpdateSettings(context.Background(), managerActor(), settings); !errors.Is(err, supportai.ErrSettingsInvalid) {
		t.Fatalf("err = %v", err)
	}
	if store.saves != 0 {
		t.Fatal("invalid settings must not be persisted")
	}
}

func TestServiceResetRestoresRecommendedDefaults(t *testing.T) {
	enabled := supportai.RecommendedSettings()
	enabled.Enabled = true
	enabled.Profiles[0].Enabled = true
	store := &stubSettings{stored: enabled}
	service := modelsai.NewService(modelsai.Config{Settings: store})

	reset, err := service.ResetSettings(context.Background(), managerActor())
	if err != nil {
		t.Fatalf("reset failed: %v", err)
	}
	if reset.Enabled {
		t.Fatal("reset must return the disabled gateway default")
	}
	if reset.Gates.ExtensionDailyQuota != supportai.RecommendedSettings().Gates.ExtensionDailyQuota {
		t.Fatalf("quota not restored: %+v", reset.Gates)
	}
	if store.resets != 1 {
		t.Fatalf("resets = %d", store.resets)
	}
}

func TestServiceWorksWithoutOptionalStores(t *testing.T) {
	service := modelsai.NewService(modelsai.Config{})
	ctx := context.Background()
	settings, err := service.GetSettings(ctx, managerActor())
	if err != nil || settings.Enabled {
		t.Fatalf("settings = %+v err = %v", settings, err)
	}
	usage, err := service.Usage(ctx, managerActor(), time.Now())
	if err != nil || usage.Site.DayCalls != 0 {
		t.Fatalf("usage = %+v err = %v", usage, err)
	}
	items, err := service.Executions(ctx, managerActor(), 10)
	if err != nil || len(items) != 0 {
		t.Fatalf("executions = %+v err = %v", items, err)
	}
}
