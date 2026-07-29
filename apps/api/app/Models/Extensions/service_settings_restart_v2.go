package extensions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	settingslifecycle "github.com/zhuchunshu/sforum/apps/api/app/Support/SettingsLifecycle"
)

// preflightLifecycleV2SettingsRestart 在设置文档或 SecretStore 发生修改前
// 检查停用、启用两个阶段；实际执行时协调器仍会重新校验运行时状态。
func (s *serviceCore) preflightLifecycleV2SettingsRestart(
	ctx context.Context,
	actor identity.Actor,
	extension Extension,
) error {
	if !canManageExtensionSettings(actor, extension) {
		return identity.ErrPermissionDenied
	}
	if s.lifecycleCoordinator == nil || s.lifecyclePreflight == nil || s.lifecycleAuthority == nil {
		return ErrLifecycleCoordinatorUnavailable
	}
	if err := s.preflightRestartTarget(ctx, extension); err != nil {
		return err
	}
	disable := lifecycleServiceRequest{
		operation: LifecycleMachineDisable,
		source:    exactLifecycleCopy(extension), target: extension,
		idempotencyKey: "settings-preflight-disable", frozenAuthority: true,
	}
	if _, err := s.lifecycleServiceAuthority(ctx, actor, disable); err != nil {
		return err
	}
	if err := s.lifecyclePreflight(ctx, disable.operation, disable.source, disable.target); err != nil {
		return errors.Join(ErrPreflightFailed, err)
	}
	enable := lifecycleServiceRequest{
		operation: LifecycleMachineEnable, target: extension,
		idempotencyKey: "settings-preflight-enable",
	}
	if _, err := s.lifecycleServiceAuthority(ctx, actor, enable); err != nil {
		return err
	}
	if err := s.lifecyclePreflight(ctx, enable.operation, nil, enable.target); err != nil {
		return errors.Join(ErrPreflightFailed, err)
	}
	return nil
}

// restartLifecycleV2ForSettings 只重启精确 active 制品。设置保存不是升级，
// 因此即使存在 staged 制品也不得顺便晋升。
func (s *serviceCore) restartLifecycleV2ForSettings(
	ctx context.Context,
	actor identity.Actor,
	extension Extension,
	mutationKey string,
) error {
	if mutationKey == "" {
		return fmt.Errorf("%w: settings mutation identity is missing", ErrLifecycleCoordinatorInvalid)
	}
	if _, err := s.disableLifecycleV2(ctx, actor, extension, LifecycleRequestInput{
		IdempotencyKey: settingsRestartPhaseKey(mutationKey, "disable"),
	}); err != nil {
		return fmt.Errorf("disable exact plugin for settings restart: %w", err)
	}

	disabled, err := s.store.Get(ctx, extension.ID)
	if err != nil {
		return fmt.Errorf("reload plugin after settings disable: %w", err)
	}
	if disabled.Status != StatusDisabled || !sameRestartArtifact(disabled, extension) {
		return fmt.Errorf("%w: settings restart exact artifact changed while disabled", ErrLifecycleCoordinatorInvalid)
	}
	if _, err := s.enableLifecycleV2(ctx, actor, disabled, EnableInput{
		IdempotencyKey: settingsRestartPhaseKey(mutationKey, "enable"),
	}); err != nil {
		return fmt.Errorf("enable exact plugin after settings change: %w", err)
	}
	return nil
}

func settingsRestartMutationKey(extension Extension, doc settingslifecycle.Document) string {
	updatedAt := doc.UpdatedAt.UTC()
	if updatedAt.IsZero() {
		updatedAt = time.Unix(0, 0).UTC()
	}
	digest := sha256.Sum256([]byte(
		extension.ID + "\x00" + extension.Version + "\x00" + extension.PackageDigest + "\x00" +
			doc.UpdatedBy + "\x00" + strconv.FormatInt(updatedAt.UnixNano(), 10),
	))
	return hex.EncodeToString(digest[:])
}

func legacySettingsRestartMutationKey(extension Extension, actor identity.Actor) string {
	doc := settingslifecycle.Document{
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: settingsActorID(actor),
	}
	return settingsRestartMutationKey(extension, doc)
}

func settingsRestartPhaseKey(mutationKey string, phase string) string {
	digest := sha256.Sum256([]byte("sforum.extension.settings.restart\x00" + phase + "\x00" + mutationKey))
	return "settings-" + phase + "-" + hex.EncodeToString(digest[:])
}
