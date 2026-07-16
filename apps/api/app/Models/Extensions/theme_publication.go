package extensions

import (
	"context"
	"errors"
)

var (
	ErrThemePublicationNotFound = errors.New("extensions: theme runtime publication not found")
	ErrThemePublicationConflict = errors.New("extensions: theme runtime publication conflict")
)

type ThemeRuntimePublicationRepository interface {
	LatestThemeRuntimePublication(context.Context) (ThemeRuntimePublication, error)
	ThemeRuntimePublicationByRevision(context.Context, int64) (ThemeRuntimePublication, error)
}

// InitialThemeRuntimePublicationEnsurer imports the pre-publication active
// theme exactly once. Watchers use it before registering a boot lease so an
// empty desired-state ledger can never be mistaken for successful convergence.
type InitialThemeRuntimePublicationEnsurer interface {
	EnsureInitialThemeRuntimePublication(context.Context) (ThemeRuntimePublication, error)
}
