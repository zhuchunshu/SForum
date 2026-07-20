package hostapi

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Capabilities"
	hostv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/host/v2"
	protocolv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/protocol/v2"
)

const (
	CommandExtensionPluginDisableID           = "sforum.extensions.plugin.disable"
	CommandExtensionPluginDisableVersion      = "1"
	CommandExtensionPluginDisableInputSchema  = "sforum.extensions.plugin.disable.input"
	CommandExtensionPluginDisableOutputSchema = "sforum.extensions.plugin.disable.result"
	CommandExtensionPluginDisableSchemaV1     = "1"

	CommandExtensionSettingsResetID           = "sforum.extensions.settings.reset"
	CommandExtensionSettingsResetVersion      = "1"
	CommandExtensionSettingsResetInputSchema  = "sforum.extensions.settings.reset.input"
	CommandExtensionSettingsResetOutputSchema = "sforum.extensions.settings.reset.result"
	CommandExtensionSettingsResetSchemaV1     = "1"
)

type protocolV2ExtensionPluginDisableInput struct {
	TargetExtensionID string `json:"targetExtensionId"`
}

type protocolV2ExtensionPluginDisableMutation struct {
	targetExtensionID string
}

// newProtocolV2ExtensionPluginDisableCommandDefinition 实现 extensions.manage
// 允许清单中的“禁用已信任非系统非自身插件”。状态写入在 Host Command 事务内完成；
// 运行时 drain 由后续 lifecycle reconcile 收敛。
func newProtocolV2ExtensionPluginDisableCommandDefinition() protocolV2CommandDefinition {
	definition := protocolV2CommandDefinition{
		ID: CommandExtensionPluginDisableID, Version: CommandExtensionPluginDisableVersion,
		InputSchemaID: CommandExtensionPluginDisableInputSchema, InputSchemaVersion: CommandExtensionPluginDisableSchemaV1,
		OutputSchemaID: CommandExtensionPluginDisableOutputSchema, OutputSchemaVersion: CommandExtensionPluginDisableSchemaV1,
		ActorMode:           protocolV2CommandActorDelegated,
		RequiredPermissions: []string{identity.PermissionExtensionPluginManage},
		RequiredCapability:  capabilities.ExtensionsManage,
	}
	definition.Preview = func(_ context.Context, request *hostv2.CommandRequest) (*protocolV2CommandPreparation, error) {
		mutation, err := protocolV2ExtensionPluginDisableMutationFromRequest(request)
		if err != nil {
			return nil, err
		}
		return protocolV2ExtensionPluginDisablePreparation(mutation, "enabled")
	}
	definition.Prepare = func(ctx context.Context, tx pgx.Tx, request *hostv2.CommandRequest) (*protocolV2CommandPreparation, error) {
		mutation, err := protocolV2ExtensionPluginDisableMutationFromRequest(request)
		if err != nil {
			return nil, err
		}
		runtime := ProtocolV2RuntimeIdentityFromContext(ctx)
		if runtime == nil || strings.TrimSpace(runtime.GetExtensionId()) == "" {
			return nil, invalidProtocolV2CommandActorDelegation()
		}
		callerID := strings.TrimSpace(runtime.GetExtensionId())
		if mutation.targetExtensionID == callerID {
			return nil, newProtocolV2CommandError(
				protocolv2.ErrorCode_ERROR_CODE_PERMISSION_DENIED,
				"host.extensions_manage_self_denied",
				"A plugin cannot disable itself through extensions.manage.",
				false,
			)
		}
		var (
			extensionType string
			status        string
			isSystem      bool
			source        string
		)
		err = tx.QueryRow(ctx, `
			SELECT type, status, is_system, source
			FROM extensions
			WHERE id = $1
			FOR UPDATE
		`, mutation.targetExtensionID).Scan(&extensionType, &status, &isSystem, &source)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, newProtocolV2CommandError(
				protocolv2.ErrorCode_ERROR_CODE_NOT_FOUND,
				"host.extension_not_found",
				"The target extension does not exist.",
				false,
			)
		}
		if err != nil {
			return nil, fmt.Errorf("lock target extension for disable: %w", err)
		}
		if extensionType != "plugin" {
			return nil, newProtocolV2CommandError(
				protocolv2.ErrorCode_ERROR_CODE_FAILED_PRECONDITION,
				"host.extensions_manage_theme_denied",
				"extensions.manage can only disable plugins.",
				false,
			)
		}
		if isSystem {
			return nil, newProtocolV2CommandError(
				protocolv2.ErrorCode_ERROR_CODE_PERMISSION_DENIED,
				"host.extensions_manage_system_denied",
				"System plugins cannot be disabled through extensions.manage.",
				false,
			)
		}
		if status != "enabled" {
			return nil, newProtocolV2CommandError(
				protocolv2.ErrorCode_ERROR_CODE_FAILED_PRECONDITION,
				"host.extensions_manage_not_enabled",
				"Only an already-enabled trusted plugin can be disabled through extensions.manage.",
				false,
			)
		}
		// 已信任：builtin 非系统，或 uploaded 存在未撤销 enable grant。
		if source == "uploaded" {
			var trusted bool
			if err := tx.QueryRow(ctx, `
				SELECT EXISTS (
				  SELECT 1 FROM extension_trust_grants
				  WHERE extension_id = $1 AND action = 'enable' AND revoked_at IS NULL
				)
			`, mutation.targetExtensionID).Scan(&trusted); err != nil {
				return nil, fmt.Errorf("check target trust grant: %w", err)
			}
			if !trusted {
				return nil, newProtocolV2CommandError(
					protocolv2.ErrorCode_ERROR_CODE_FAILED_PRECONDITION,
					"host.extensions_manage_untrusted",
					"Only already-trusted plugins can be disabled through extensions.manage.",
					false,
				)
			}
		}
		return protocolV2ExtensionPluginDisablePreparation(mutation, status)
	}
	definition.Execute = func(ctx context.Context, tx pgx.Tx, request *hostv2.CommandRequest, _ *protocolV2CommandPreparation) (*protocolV2CommandExecution, error) {
		mutation, err := protocolV2ExtensionPluginDisableMutationFromRequest(request)
		if err != nil {
			return nil, err
		}
		actorUserID, ok := ProtocolV2CommandActorUserID(ctx)
		if !ok {
			return nil, invalidProtocolV2CommandActorDelegation()
		}
		command, err := tx.Exec(ctx, `
			UPDATE extensions
			SET status = 'disabled', updated_at = now()
			WHERE id = $1 AND type = 'plugin' AND is_system = FALSE AND status = 'enabled'
		`, mutation.targetExtensionID)
		if err != nil {
			return nil, fmt.Errorf("disable target extension: %w", err)
		}
		if command.RowsAffected() != 1 {
			return nil, newProtocolV2CommandError(
				protocolv2.ErrorCode_ERROR_CODE_CONFLICT,
				"host.extensions_manage_race",
				"The target extension changed before disable committed.",
				false,
			)
		}
		// 清理邮件提供商选择，与 store.Disable 行为对齐。
		if _, err := tx.Exec(ctx, `DELETE FROM mail_provider_selection WHERE extension_id = $1`, mutation.targetExtensionID); err != nil {
			return nil, fmt.Errorf("clear mail provider selection: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO extension_events (extension_id, actor_user_id, action, message)
			VALUES ($1, $2, 'disabled', 'Extension disabled via extensions.manage Host Command.')
		`, mutation.targetExtensionID, actorUserID); err != nil {
			return nil, fmt.Errorf("append disable event: %w", err)
		}
		var revision time.Time
		if err := tx.QueryRow(ctx, `SELECT updated_at FROM extensions WHERE id = $1`, mutation.targetExtensionID).Scan(&revision); err != nil {
			return nil, fmt.Errorf("read disable revision: %w", err)
		}
		output, err := protocolV2Document(CommandExtensionPluginDisableOutputSchema, CommandExtensionPluginDisableSchemaV1, map[string]any{
			"targetExtensionId": mutation.targetExtensionID,
			"status":            "disabled",
			"revision":          revision.UTC().Format(time.RFC3339Nano),
		})
		if err != nil {
			return nil, fmt.Errorf("encode disable command result: %w", err)
		}
		return &protocolV2CommandExecution{
			Output: output, CommittedRevision: revision.UTC().Format(time.RFC3339Nano),
		}, nil
	}
	return definition
}

func protocolV2ExtensionPluginDisableMutationFromRequest(request *hostv2.CommandRequest) (protocolV2ExtensionPluginDisableMutation, error) {
	input, err := decodeProtocolV2CommandInput[protocolV2ExtensionPluginDisableInput](request)
	if err != nil || strings.TrimSpace(request.GetExpectedRevision()) != "" {
		return protocolV2ExtensionPluginDisableMutation{}, invalidProtocolV2DomainCommandInput()
	}
	target := strings.TrimSpace(input.TargetExtensionID)
	if target == "" || len(target) > 200 {
		return protocolV2ExtensionPluginDisableMutation{}, invalidProtocolV2DomainCommandInput()
	}
	return protocolV2ExtensionPluginDisableMutation{targetExtensionID: target}, nil
}

func protocolV2ExtensionPluginDisablePreparation(
	mutation protocolV2ExtensionPluginDisableMutation,
	currentStatus string,
) (*protocolV2CommandPreparation, error) {
	output, err := protocolV2Document(CommandExtensionPluginDisableOutputSchema, CommandExtensionPluginDisableSchemaV1, map[string]any{
		"targetExtensionId": mutation.targetExtensionID,
		"status":            "disabled",
		"previousStatus":    currentStatus,
	})
	if err != nil {
		return nil, err
	}
	return &protocolV2CommandPreparation{
		Policy: []*hostv2.PolicyDecision{{
			PolicyId: "sforum.extensions.manage@1", Allowed: true,
			Reason: "Delegated actor holds extension.plugin.manage and caller holds extensions.manage.",
		}},
		Impact: []*hostv2.ImpactItem{{
			Module: "extensions", Action: "disable", ResourceType: "plugin",
			ResourceId: mutation.targetExtensionID, Summary: "Disable an already-trusted non-system non-self plugin.",
		}},
		ProjectedResult: output,
	}, nil
}

type protocolV2ExtensionSettingsResetInput struct {
	TargetExtensionID string `json:"targetExtensionId"`
}

type protocolV2ExtensionSettingsResetMutation struct {
	targetExtensionID string
}

// newProtocolV2ExtensionSettingsResetCommandDefinition 重置已信任非系统非自身
// 插件的设置。仅删除 extension_settings 行（回到 Manifest 默认），不执行上传/
// 启用/信任变更。运行时热重载由后续 reconcile 收敛。
func newProtocolV2ExtensionSettingsResetCommandDefinition() protocolV2CommandDefinition {
	definition := protocolV2CommandDefinition{
		ID: CommandExtensionSettingsResetID, Version: CommandExtensionSettingsResetVersion,
		InputSchemaID: CommandExtensionSettingsResetInputSchema, InputSchemaVersion: CommandExtensionSettingsResetSchemaV1,
		OutputSchemaID: CommandExtensionSettingsResetOutputSchema, OutputSchemaVersion: CommandExtensionSettingsResetSchemaV1,
		ActorMode:           protocolV2CommandActorDelegated,
		RequiredPermissions: []string{identity.PermissionExtensionPluginManage},
		RequiredCapability:  capabilities.ExtensionsManage,
	}
	definition.Preview = func(_ context.Context, request *hostv2.CommandRequest) (*protocolV2CommandPreparation, error) {
		mutation, err := protocolV2ExtensionSettingsResetMutationFromRequest(request)
		if err != nil {
			return nil, err
		}
		return protocolV2ExtensionSettingsResetPreparation(mutation)
	}
	definition.Prepare = func(ctx context.Context, tx pgx.Tx, request *hostv2.CommandRequest) (*protocolV2CommandPreparation, error) {
		mutation, err := protocolV2ExtensionSettingsResetMutationFromRequest(request)
		if err != nil {
			return nil, err
		}
		if err := protocolV2ExtensionsManageLockTarget(ctx, tx, request, mutation.targetExtensionID, false); err != nil {
			return nil, err
		}
		return protocolV2ExtensionSettingsResetPreparation(mutation)
	}
	definition.Execute = func(ctx context.Context, tx pgx.Tx, request *hostv2.CommandRequest, _ *protocolV2CommandPreparation) (*protocolV2CommandExecution, error) {
		mutation, err := protocolV2ExtensionSettingsResetMutationFromRequest(request)
		if err != nil {
			return nil, err
		}
		actorUserID, ok := ProtocolV2CommandActorUserID(ctx)
		if !ok {
			return nil, invalidProtocolV2CommandActorDelegation()
		}
		if _, err := tx.Exec(ctx, `DELETE FROM extension_settings WHERE extension_id = $1`, mutation.targetExtensionID); err != nil {
			return nil, fmt.Errorf("reset target extension settings: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO extension_events (extension_id, actor_user_id, action, message)
			VALUES ($1, $2, 'settings_reset', 'Extension settings reset via extensions.manage Host Command.')
		`, mutation.targetExtensionID, actorUserID); err != nil {
			return nil, fmt.Errorf("append settings reset event: %w", err)
		}
		var revision time.Time
		if err := tx.QueryRow(ctx, `SELECT transaction_timestamp()`).Scan(&revision); err != nil {
			return nil, fmt.Errorf("read settings reset revision: %w", err)
		}
		output, err := protocolV2Document(CommandExtensionSettingsResetOutputSchema, CommandExtensionSettingsResetSchemaV1, map[string]any{
			"targetExtensionId": mutation.targetExtensionID,
			"status":            "reset",
			"revision":          revision.UTC().Format(time.RFC3339Nano),
		})
		if err != nil {
			return nil, fmt.Errorf("encode settings reset result: %w", err)
		}
		return &protocolV2CommandExecution{
			Output: output, CommittedRevision: revision.UTC().Format(time.RFC3339Nano),
		}, nil
	}
	return definition
}

func protocolV2ExtensionSettingsResetMutationFromRequest(request *hostv2.CommandRequest) (protocolV2ExtensionSettingsResetMutation, error) {
	input, err := decodeProtocolV2CommandInput[protocolV2ExtensionSettingsResetInput](request)
	if err != nil || strings.TrimSpace(request.GetExpectedRevision()) != "" {
		return protocolV2ExtensionSettingsResetMutation{}, invalidProtocolV2DomainCommandInput()
	}
	target := strings.TrimSpace(input.TargetExtensionID)
	if target == "" || len(target) > 200 {
		return protocolV2ExtensionSettingsResetMutation{}, invalidProtocolV2DomainCommandInput()
	}
	return protocolV2ExtensionSettingsResetMutation{targetExtensionID: target}, nil
}

func protocolV2ExtensionSettingsResetPreparation(
	mutation protocolV2ExtensionSettingsResetMutation,
) (*protocolV2CommandPreparation, error) {
	output, err := protocolV2Document(CommandExtensionSettingsResetOutputSchema, CommandExtensionSettingsResetSchemaV1, map[string]any{
		"targetExtensionId": mutation.targetExtensionID,
		"status":            "reset",
	})
	if err != nil {
		return nil, err
	}
	return &protocolV2CommandPreparation{
		Policy: []*hostv2.PolicyDecision{{
			PolicyId: "sforum.extensions.manage@1", Allowed: true,
			Reason: "Delegated actor holds extension.plugin.manage and caller holds extensions.manage.",
		}},
		Impact: []*hostv2.ImpactItem{{
			Module: "extensions", Action: "settings_reset", ResourceType: "plugin_settings",
			ResourceId: mutation.targetExtensionID, Summary: "Reset settings of an already-trusted non-system non-self plugin.",
		}},
		ProjectedResult: output,
	}, nil
}

// protocolV2ExtensionsManageLockTarget 共享目标门禁：非自身、插件、非系统；
// requireEnabled 为 true 时还要求 enabled + 已信任。
func protocolV2ExtensionsManageLockTarget(
	ctx context.Context,
	tx pgx.Tx,
	request *hostv2.CommandRequest,
	targetExtensionID string,
	requireEnabled bool,
) error {
	runtime := ProtocolV2RuntimeIdentityFromContext(ctx)
	if runtime == nil || strings.TrimSpace(runtime.GetExtensionId()) == "" {
		return invalidProtocolV2CommandActorDelegation()
	}
	callerID := strings.TrimSpace(runtime.GetExtensionId())
	if targetExtensionID == callerID {
		return newProtocolV2CommandError(
			protocolv2.ErrorCode_ERROR_CODE_PERMISSION_DENIED,
			"host.extensions_manage_self_denied",
			"A plugin cannot manage itself through extensions.manage.",
			false,
		)
	}
	var (
		extensionType string
		status        string
		isSystem      bool
		source        string
	)
	err := tx.QueryRow(ctx, `
		SELECT type, status, is_system, source
		FROM extensions
		WHERE id = $1
		FOR UPDATE
	`, targetExtensionID).Scan(&extensionType, &status, &isSystem, &source)
	if errors.Is(err, pgx.ErrNoRows) {
		return newProtocolV2CommandError(
			protocolv2.ErrorCode_ERROR_CODE_NOT_FOUND,
			"host.extension_not_found",
			"The target extension does not exist.",
			false,
		)
	}
	if err != nil {
		return fmt.Errorf("lock target extension for manage: %w", err)
	}
	if extensionType != "plugin" {
		return newProtocolV2CommandError(
			protocolv2.ErrorCode_ERROR_CODE_FAILED_PRECONDITION,
			"host.extensions_manage_theme_denied",
			"extensions.manage can only target plugins.",
			false,
		)
	}
	if isSystem {
		return newProtocolV2CommandError(
			protocolv2.ErrorCode_ERROR_CODE_PERMISSION_DENIED,
			"host.extensions_manage_system_denied",
			"System plugins cannot be managed through extensions.manage.",
			false,
		)
	}
	if requireEnabled && status != "enabled" {
		return newProtocolV2CommandError(
			protocolv2.ErrorCode_ERROR_CODE_FAILED_PRECONDITION,
			"host.extensions_manage_not_enabled",
			"Only an already-enabled trusted plugin can be disabled through extensions.manage.",
			false,
		)
	}
	if source == "uploaded" {
		var trusted bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
			  SELECT 1 FROM extension_trust_grants
			  WHERE extension_id = $1 AND action = 'enable' AND revoked_at IS NULL
			)
		`, targetExtensionID).Scan(&trusted); err != nil {
			return fmt.Errorf("check target trust grant: %w", err)
		}
		if !trusted {
			return newProtocolV2CommandError(
				protocolv2.ErrorCode_ERROR_CODE_FAILED_PRECONDITION,
				"host.extensions_manage_untrusted",
				"Only already-trusted plugins can be managed through extensions.manage.",
				false,
			)
		}
	}
	_ = request
	return nil
}
