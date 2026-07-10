package extensions

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

var (
	ErrWebReleaseNotFound                   = errors.New("extensions: web release not found")
	ErrWebReleaseStale                      = errors.New("extensions: stale web release state")
	ErrWebReleaseCompositionMismatch        = errors.New("extensions: web release composition hash mismatch")
	ErrWebReleaseDependencySnapshotConflict = errors.New("extensions: web release dependency snapshot conflict")
)

type WebReleaseStore interface {
	CreateWebRelease(context.Context, WebReleaseCreateInput) (WebRelease, error)
	CreateWebReleaseTx(context.Context, pgx.Tx, WebReleaseCreateInput) (WebRelease, error)
	TransitionWebRelease(context.Context, WebReleaseTransitionInput) (WebRelease, error)
	ActiveWebRelease(context.Context) (WebRelease, error)
	WebRelease(context.Context, int64) (WebReleaseDetail, error)
	ListWebReleases(context.Context, WebReleaseListInput) (WebReleasePage, error)
	RecordWebReleaseDependencySnapshot(context.Context, WebReleaseDependencySnapshotInput) error
}
