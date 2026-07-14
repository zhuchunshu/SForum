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
