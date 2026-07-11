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

type ExtensionStateStore interface {
	Get(context.Context, string) (extensions.Extension, error)
	Enable(context.Context, string, string) (extensions.Extension, error)
	Disable(context.Context, string) (extensions.Extension, error)
	ActivateTheme(context.Context, string) (extensions.Extension, error)
}

type PostgresStore struct {
	pool       *pgxpool.Pool
	releases   ReleaseStore
	extensions ExtensionStateStore
}

func NewPostgresStore(pool *pgxpool.Pool, releases ReleaseStore, extensionStore ExtensionStateStore) *PostgresStore {
	return &PostgresStore{pool: pool, releases: releases, extensions: extensionStore}
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

func (s *PostgresStore) ApplyEffects(ctx context.Context, detail extensions.WebReleaseDetail, forward bool) error {
	for _, effect := range detail.Effects {
		status := effect.TargetStatus
		if !forward {
			status = effect.PreviousStatus
		}
		item, err := s.extensions.Get(ctx, effect.ExtensionID)
		if err != nil {
			return err
		}
		switch status {
		case extensions.StatusEnabled:
			if item.Type == extensions.TypeTheme {
				_, err = s.extensions.ActivateTheme(ctx, item.ID)
			} else {
				_, err = s.extensions.Enable(ctx, item.ID, item.Type)
			}
		case extensions.StatusDisabled, extensions.StatusInstalled:
			_, err = s.extensions.Disable(ctx, item.ID)
		default:
			return fmt.Errorf("unsupported extension target status %q", status)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *PostgresStore) FinalizeRevocations(ctx context.Context, releaseID int64) error {
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

type RuntimeAdapter struct {
	store   ExtensionStateStore
	runtime ExtensionRuntime
}

func NewRuntimeAdapter(store ExtensionStateStore, runtime ExtensionRuntime) *RuntimeAdapter {
	return &RuntimeAdapter{store: store, runtime: runtime}
}

func (a *RuntimeAdapter) Prepare(ctx context.Context, detail extensions.WebReleaseDetail) error {
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
