package webreleasecoordinator

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

type ReleaseStore interface {
	TransitionWebRelease(context.Context, extensions.WebReleaseTransitionInput) (extensions.WebRelease, error)
	WebRelease(context.Context, int64) (extensions.WebReleaseDetail, error)
	ListWebReleases(context.Context, extensions.WebReleaseListInput) (extensions.WebReleasePage, error)
}

// ExtensionStateStore 仅用于读取状态与 RuntimeAdapter；
// 状态变更必须走 ApprovedLifecycleApplier，不得直接 Enable/Disable 绕过 Page Registry。
type ExtensionStateStore interface {
	Get(context.Context, string) (extensions.Extension, error)
}

// ApprovedLifecycleApplier 执行已批准 Web Release 效果的完整扩展生命周期。
// 实现方（extensions.Service）负责：改状态、启停 runtime、注册/清除页面贡献与补偿。
// 不需要伪造管理员 actor。
type ApprovedLifecycleApplier interface {
	ApplyApprovedLifecycleEffect(ctx context.Context, extensionID string, targetStatus string) error
}

type PostgresStore struct {
	pool       *pgxpool.Pool
	releases   ReleaseStore
	extensions ExtensionStateStore
	lifecycle  ApprovedLifecycleApplier
}

func NewPostgresStore(pool *pgxpool.Pool, releases ReleaseStore, extensionStore ExtensionStateStore) *PostgresStore {
	return &PostgresStore{pool: pool, releases: releases, extensions: extensionStore}
}

// WithLifecycle 注入完整生命周期执行器（含 Page Registry 同步）。
func (s *PostgresStore) WithLifecycle(applier ApprovedLifecycleApplier) *PostgresStore {
	if s != nil {
		s.lifecycle = applier
	}
	return s
}

func (s *PostgresStore) NextActivation(ctx context.Context) (extensions.WebReleaseDetail, error) {
	page, err := s.releases.ListWebReleases(ctx, extensions.WebReleaseListInput{Status: extensions.WebReleaseActivating, Page: 1, PerPage: 1})
	if err != nil {
		return extensions.WebReleaseDetail{}, err
	}
	if len(page.Items) > 0 {
		return s.releases.WebRelease(ctx, page.Items[0].ID)
	}
	page, err = s.releases.ListWebReleases(ctx, extensions.WebReleaseListInput{Status: extensions.WebReleaseReady, Page: 1, PerPage: 100})
	if err != nil {
		return extensions.WebReleaseDetail{}, err
	}
	if len(page.Items) == 0 {
		return extensions.WebReleaseDetail{}, ErrNoActivation
	}
	for _, stale := range page.Items[1:] {
		if _, err := s.releases.TransitionWebRelease(ctx, extensions.WebReleaseTransitionInput{
			ID: stale.ID, ExpectedStatus: extensions.WebReleaseReady, NextStatus: extensions.WebReleaseSuperseded,
			Reason: "web_release.superseded", Message: fmt.Sprintf("Superseded by web release %d", page.Items[0].ID),
		}); err != nil {
			return extensions.WebReleaseDetail{}, err
		}
	}
	return s.releases.WebRelease(ctx, page.Items[0].ID)
}

func (s *PostgresStore) Transition(ctx context.Context, input extensions.WebReleaseTransitionInput) (extensions.WebRelease, error) {
	return s.releases.TransitionWebRelease(ctx, input)
}

func (s *PostgresStore) SetCheckpoint(ctx context.Context, releaseID int64, expected, next string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE web_releases SET activation_checkpoint = $3, updated_at = now()
		WHERE id = $1 AND status = 'activating' AND activation_checkpoint = $2
	`, releaseID, expected, next)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return extensions.ErrWebReleaseStale
	}
	return nil
}

// ApplyEffects 通过完整生命周期应用/回滚效果，同步 Page Registry。
// 禁止直接调用底层 Store.Enable/Disable（会绕过 RegisterPluginPackage/ClearExtension）。
func (s *PostgresStore) ApplyEffects(ctx context.Context, detail extensions.WebReleaseDetail, forward bool) error {
	if s.lifecycle == nil {
		return fmt.Errorf("web release: lifecycle applier not configured")
	}
	for _, effect := range detail.Effects {
		status := effect.TargetStatus
		if !forward {
			status = effect.PreviousStatus
		}
		// 确认扩展仍存在
		if _, err := s.extensions.Get(ctx, effect.ExtensionID); err != nil {
			return err
		}
		if err := s.lifecycle.ApplyApprovedLifecycleEffect(ctx, effect.ExtensionID, status); err != nil {
			return err
		}
	}
	return nil
}

func (s *PostgresStore) FinalizeRevocations(ctx context.Context, releaseID int64) error {
	// 信任吊销完成后：不在此直接清页面；disable effect 已同步 ClearExtension。
	// 若某扩展仅被吊销信任仍保持 enabled，页面贡献保持（管理端 Vue 不可用，公开页仍可）。
	_, err := s.pool.Exec(ctx, `
		UPDATE extension_frontend_trust_grants AS grants
		SET revoked_at = COALESCE(revoked_at, now())
		WHERE grants.revocation_requested_at IS NOT NULL
		  AND grants.revoked_at IS NULL
		  AND NOT EXISTS (
		    SELECT 1 FROM web_release_extensions AS snapshot
		    WHERE snapshot.web_release_id = $1
		      AND snapshot.extension_id = grants.extension_id
		      AND snapshot.extension_version = grants.extension_version
		      AND snapshot.package_digest = grants.package_digest
		  )
	`, releaseID)
	return err
}

type ExtensionRuntime interface {
	Start(context.Context, extensions.Extension) error
	Stop(context.Context, extensions.Extension) error
}

// RuntimeAdapter 在 ApplyEffects 之前/之后协调 runtime。
// 注意：完整 enable 生命周期（含 Start）已在 ApplyApprovedLifecycleEffect 内执行；
// Prepare 仅对「即将 enable 且需要先起 runtime 再切指针」的旧路径做预热。
// 为避免双重 Start，Prepare 在 lifecycle applier 已配置时跳过 Start（由 effect 负责）。
type RuntimeAdapter struct {
	store              ExtensionStateStore
	runtime            ExtensionRuntime
	lifecycleOwnsStart bool
}

func NewRuntimeAdapter(store ExtensionStateStore, runtime ExtensionRuntime) *RuntimeAdapter {
	return &RuntimeAdapter{store: store, runtime: runtime}
}

// WithLifecycleOwnsStart 标记 Start/Stop 由 lifecycle applier 负责，避免双重启停。
func (a *RuntimeAdapter) WithLifecycleOwnsStart(v bool) *RuntimeAdapter {
	if a != nil {
		a.lifecycleOwnsStart = v
	}
	return a
}

func (a *RuntimeAdapter) Prepare(ctx context.Context, detail extensions.WebReleaseDetail) error {
	if a.lifecycleOwnsStart {
		// lifecycle 在 ApplyEffects 内 Start；此处仅校验扩展存在
		for _, effect := range detail.Effects {
			if _, err := a.store.Get(ctx, effect.ExtensionID); err != nil {
				return err
			}
		}
		return nil
	}
	for _, effect := range detail.Effects {
		if effect.TargetStatus == extensions.StatusEnabled && effect.PreviousStatus != extensions.StatusEnabled {
			item, err := a.store.Get(ctx, effect.ExtensionID)
			if err != nil {
				return err
			}
			if item.Type == extensions.TypePlugin {
				if err := a.runtime.Start(ctx, item); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (a *RuntimeAdapter) Finalize(ctx context.Context, detail extensions.WebReleaseDetail) error {
	if a.lifecycleOwnsStart {
		return nil
	}
	for _, effect := range detail.Effects {
		if effect.TargetStatus != extensions.StatusEnabled && effect.PreviousStatus == extensions.StatusEnabled {
			item, err := a.store.Get(ctx, effect.ExtensionID)
			if err != nil {
				return err
			}
			if item.Type == extensions.TypePlugin {
				if err := a.runtime.Stop(ctx, item); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (a *RuntimeAdapter) Compensate(ctx context.Context, detail extensions.WebReleaseDetail) error {
	if a.lifecycleOwnsStart {
		// reverse effects 已在 ApplyEffects(forward=false) 中通过 lifecycle 处理
		return nil
	}
	var result error
	for _, effect := range detail.Effects {
		item, err := a.store.Get(ctx, effect.ExtensionID)
		if err != nil {
			result = errors.Join(result, err)
			continue
		}
		if item.Type != extensions.TypePlugin {
			continue
		}
		if effect.PreviousStatus == extensions.StatusEnabled && effect.TargetStatus != extensions.StatusEnabled {
			err = a.runtime.Start(ctx, item)
		} else if effect.PreviousStatus != extensions.StatusEnabled && effect.TargetStatus == extensions.StatusEnabled {
			err = a.runtime.Stop(ctx, item)
		}
		result = errors.Join(result, err)
	}
	return result
}
