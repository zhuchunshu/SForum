package extensionsruntime

import (
	"context"
	"errors"
	"fmt"
	"strings"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

func (a *ManagerPluginRuntimeFullSetApplier) allowsInitialProtocolV1Compatibility() bool {
	return a != nil && a.initialProtocolV1Compatibility.Load()
}

func (a *ManagerPluginRuntimeFullSetApplier) disarmInitialProtocolV1Compatibility() {
	if a == nil {
		return
	}
	a.initialProtocolV1Compatibility.Store(false)
}

// initialProtocolV1StartedMember 记录本轮 cold-start 的 exact 身份与冻结 artifact。
// 回滚只允许停止该 identity，绝不以 extension ID 盲目 stop 替换实例。
type initialProtocolV1StartedMember struct {
	identity  RuntimeInstanceIdentity
	extension extensions.Extension
	member    extensions.PluginRuntimeMember
}

// startInitialProtocolV1CompatibilityLocked 在已持有 Manager runtime-set barrier
// 的前提下，仅 cold-start 本 publication 中缺失的 exact Protocol V1 成员。
// 启动循环失败时返回已累计账本与原始 cause，由 Apply 外层 defer 唯一回滚。
func (a *ManagerPluginRuntimeFullSetApplier) startInitialProtocolV1CompatibilityLocked(
	ctx context.Context,
	desired []pluginRuntimeFullSetDesired,
) ([]initialProtocolV1StartedMember, error) {
	if !a.allowsInitialProtocolV1Compatibility() {
		return nil, nil
	}
	started := make([]initialProtocolV1StartedMember, 0)
	for _, item := range desired {
		if err := ctx.Err(); err != nil {
			return started, err
		}
		if manifestProtocolVersion(item.extension) != 1 {
			// Protocol V2 / 非成员 / 未启用 artifact 从不在此 prestart。
			continue
		}
		matches, hasActive, err := a.inspectActiveProtocolV1MatchLocked(item.extension, item.member)
		if err != nil {
			return started, err
		}
		if matches {
			// exact active V1 => 复用，不重启、不入回滚账本。
			continue
		}
		if hasActive {
			// 活动实例与 durable exact 不匹配：fail-closed，绝不隐式替换。
			return started, fmt.Errorf(
				"%w: Protocol V1 runtime %s is not an exact reusable artifact",
				ErrProtocolInstanceTransitionBlocked, item.member.ExtensionID,
			)
		}
		if err := a.manager.startRuntimeSetLocked(ctx, item.extension); err != nil {
			return started, fmt.Errorf(
				"start Protocol V1 compatibility runtime %s@%s: %w",
				item.member.ExtensionID, item.member.ExtensionVersion, err,
			)
		}
		// start 成功后立刻捕获/校验 exact 活动身份，写入 identity-bound 账本。
		identity, err := a.captureStartedProtocolV1IdentityLocked(item.extension, item.member)
		if identity.InstanceID != "" {
			started = append(started, initialProtocolV1StartedMember{
				identity:  identity,
				extension: item.extension,
				member:    item.member,
			})
		}
		if err != nil {
			return started, err
		}
	}
	return started, nil
}

// inspectActiveProtocolV1MatchLocked 在已持有 runtime-set barrier 时检查活动实例。
func (a *ManagerPluginRuntimeFullSetApplier) inspectActiveProtocolV1MatchLocked(
	extension extensions.Extension,
	member extensions.PluginRuntimeMember,
) (matches bool, hasActive bool, err error) {
	if a == nil || a.manager == nil || strings.TrimSpace(member.ExtensionID) == "" {
		return false, false, ErrPluginRuntimeFullSetInvalid
	}
	m := a.manager
	m.mu.RLock()
	defer m.mu.RUnlock()
	instanceID := m.activeInstances[member.ExtensionID]
	if instanceID == "" {
		return false, false, nil
	}
	instance, err := m.runtimeInstanceLocked(RuntimeInstanceIdentity{
		ExtensionID: member.ExtensionID,
		InstanceID:  instanceID,
	})
	if err != nil {
		return false, true, err
	}
	return pluginRuntimeActiveMatchesDesired(instance, extension, member), true, nil
}

// captureStartedProtocolV1IdentityLocked 在 startRuntimeSetLocked 成功后读取活动
// exact 身份；返回的 identity 即使校验失败也应写入账本供外层 identity-bound 回滚。
func (a *ManagerPluginRuntimeFullSetApplier) captureStartedProtocolV1IdentityLocked(
	extension extensions.Extension,
	member extensions.PluginRuntimeMember,
) (RuntimeInstanceIdentity, error) {
	if a == nil || a.manager == nil {
		return RuntimeInstanceIdentity{}, ErrPluginRuntimeFullSetInvalid
	}
	m := a.manager
	m.mu.RLock()
	defer m.mu.RUnlock()
	instanceID := m.activeInstances[member.ExtensionID]
	if instanceID == "" {
		return RuntimeInstanceIdentity{}, fmt.Errorf(
			"%w: Protocol V1 runtime %s missing active identity after start",
			ErrPluginRuntimeFullSetConflict, member.ExtensionID,
		)
	}
	identity := RuntimeInstanceIdentity{ExtensionID: member.ExtensionID, InstanceID: instanceID}
	instance, err := m.runtimeInstanceLocked(identity)
	if err != nil {
		return identity, err
	}
	if !pluginRuntimeActiveMatchesDesired(instance, extension, member) {
		return identity, fmt.Errorf(
			"%w: Protocol V1 runtime %s/%s does not match frozen exact artifact after start",
			ErrPluginRuntimeFullSetConflict, identity.ExtensionID, identity.InstanceID,
		)
	}
	return identity, nil
}

// rollbackInitialProtocolV1Starts 在同一 barrier 下按逆序 identity-bound 停止本轮新启动的 V1。
// 使用有界后台 cleanup context，避免调用方 ctx 已取消时无法回收进程。
// 仅 activeID=="" 视为已清理；身份漂移或损坏的活动指针返回 conflict 且绝不 stop。
func (a *ManagerPluginRuntimeFullSetApplier) rollbackInitialProtocolV1Starts(
	started []initialProtocolV1StartedMember,
) error {
	if a == nil || a.manager == nil || len(started) == 0 {
		return nil
	}
	cleanupCtx, cancel := pluginRuntimeFullSetCleanupContext(context.Background())
	defer cancel()
	var errs []error
	for index := len(started) - 1; index >= 0; index-- {
		entry := started[index]
		shouldStop, inspectErr := a.inspectInitialProtocolV1RollbackTargetLocked(entry)
		if inspectErr != nil {
			errs = append(errs, inspectErr)
			continue
		}
		if !shouldStop {
			// 账本身份已不在 active 集合：视为已清理，跳过。
			continue
		}
		if err := a.manager.stopRuntimeSetLocked(cleanupCtx, entry.extension); err != nil {
			errs = append(errs, fmt.Errorf(
				"rollback Protocol V1 compatibility runtime %s@%s (%s): %w",
				entry.extension.ID, entry.extension.Version, entry.identity.InstanceID, err,
			))
		}
	}
	return errors.Join(errs...)
}

// inspectInitialProtocolV1RollbackTargetLocked 校验回滚目标仍是账本中的 exact 实例。
// 返回 shouldStop=false 且 err==nil 仅表示 active 已清空；error 表示冲突，调用方不得 stop。
func (a *ManagerPluginRuntimeFullSetApplier) inspectInitialProtocolV1RollbackTargetLocked(
	entry initialProtocolV1StartedMember,
) (shouldStop bool, err error) {
	if a == nil || a.manager == nil || entry.identity.ExtensionID == "" || entry.identity.InstanceID == "" {
		return false, ErrPluginRuntimeFullSetInvalid
	}
	m := a.manager
	m.mu.RLock()
	defer m.mu.RUnlock()
	activeID := m.activeInstances[entry.identity.ExtensionID]
	if activeID == "" {
		// 仅空 active 指针才是已清理证明。
		return false, nil
	}
	if activeID != entry.identity.InstanceID {
		// 替换实例已上位：绝不按 extension ID 盲目 stop。
		return false, fmt.Errorf(
			"%w: Protocol V1 rollback target %s drifted (want instance %s, active %s)",
			ErrPluginRuntimeFullSetConflict,
			entry.identity.ExtensionID, entry.identity.InstanceID, activeID,
		)
	}
	instance, inspectErr := m.runtimeInstanceLocked(entry.identity)
	if inspectErr != nil {
		// 活动指针存在但实例查找失败：损坏状态，fail-closed，绝不 stop、绝不静默成功。
		return false, fmt.Errorf(
			"%w: Protocol V1 rollback target %s/%s has corrupted active pointer: %w",
			ErrPluginRuntimeFullSetConflict,
			entry.identity.ExtensionID, entry.identity.InstanceID, inspectErr,
		)
	}
	if !pluginRuntimeActiveMatchesDesired(instance, entry.extension, entry.member) {
		return false, fmt.Errorf(
			"%w: Protocol V1 rollback target %s/%s no longer matches frozen exact artifact",
			ErrPluginRuntimeFullSetConflict, entry.identity.ExtensionID, entry.identity.InstanceID,
		)
	}
	return true, nil
}
