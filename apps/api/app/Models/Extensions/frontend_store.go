package extensions

import (
	"context"
	"errors"
)

var (
	ErrFrontendGrantNotFound      = errors.New("extensions: frontend trust grant not found")
	ErrFrontendGrantConflict      = errors.New("extensions: frontend trust grant conflicts with existing immutable declaration")
	ErrFrontendGrantStateConflict = errors.New("extensions: frontend trust grant state conflict")
)

type FrontendTrustStore interface {
	FrontendGrant(context.Context, string, string, string) (FrontendTrustGrant, error)
	CreateFrontendGrant(context.Context, FrontendTrustGrantInput) (FrontendTrustGrant, error)
	RequestFrontendRevocation(context.Context, FrontendRevocationInput) (FrontendTrustGrant, error)
	FinalizeFrontendRevocations(context.Context, int64) error
}
