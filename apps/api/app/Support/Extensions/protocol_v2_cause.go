package extensionsruntime

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// protocolV2OperationCause replaces only cancellation-shaped wire errors with
// the exact Host context cause. Distinguishable runtime failures remain intact.
func protocolV2OperationCause(ctx context.Context, err error) error {
	if err == nil || !protocolV2CancellationLike(err) || ctx == nil {
		return err
	}
	if cause := context.Cause(ctx); cause != nil {
		return cause
	}
	return err
}

func protocolV2CancellationLike(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	switch status.Code(err) {
	case codes.Canceled, codes.DeadlineExceeded:
		return true
	default:
		return false
	}
}
