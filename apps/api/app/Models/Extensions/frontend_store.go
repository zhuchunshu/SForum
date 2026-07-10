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
	LiveFrontendGrants(context.Context, string) ([]FrontendTrustGrant, error)
	CreateFrontendGrant(context.Context, FrontendTrustGrantInput) (FrontendTrustGrant, error)
	RequestFrontendRevocation(context.Context, FrontendRevocationInput) (FrontendTrustGrant, error)
	RequestAllFrontendRevocations(context.Context, int64) ([]FrontendTrustGrant, error)
	FinalizeFrontendRevocation(context.Context, FrontendFinalizeInput) (FrontendTrustGrant, error)
	FinalizeFrontendRevocations(context.Context, int64) error
}
