package extensions

import "context"

// ListOpenLifecycleOperations exposes the durable lifecycle fence used by the
// plugin runtime coordinator. A full runtime set cannot be reconciled through
// a direct lifecycle publication while an operation is still open.
func (s *PostgresStore) ListOpenLifecycleOperations(ctx context.Context, limit int) ([]LifecycleOperation, error) {
	if s == nil || s.pool == nil {
		return nil, ErrLifecycleInvalidInput
	}
	return NewPostgresLifecycleRepository(s.pool).ListOpenOperations(ctx, limit)
}

var _ pluginRuntimeLifecycleFenceRepository = (*PostgresStore)(nil)
