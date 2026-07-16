package hostapi

import (
	"context"
	"slices"
	"strings"
	"time"
)

// InvalidateDeclared executes one declaration-bound invalidator against every
// tag owned by that cache contract. Event adapters remain Host-owned: callers
// cannot supply physical tags or use an invalidator declared by another cache.
func (s *HostCacheService) InvalidateDeclared(
	ctx context.Context,
	request HostCacheInvalidatorRequest,
) (invalidated uint64, err error) {
	started := time.Now()
	prepared, err := s.prepare(ctx, request.HostCacheRequestBase)
	if err != nil {
		s.recordRejectedHostCacheTrace(started, request.HostCacheRequestBase, "invalidate_declared", err)
		return 0, err
	}
	defer prepared.release()
	var candidate HostCacheProviderCandidate
	attempts := 0
	affected := uint64(0)
	defer func() {
		s.recordHostCacheTrace(started, prepared, candidate, "invalidate_declared", err, false, affected, attempts)
	}()

	invalidatorID := strings.ToLower(strings.TrimSpace(request.InvalidatorID))
	if invalidatorID == "" || !slices.Contains(prepared.plan.Cache.Invalidators, invalidatorID) {
		return 0, ErrHostCacheDenied
	}
	prepared.traceInvalidator = invalidatorID
	if len(prepared.plan.Cache.Tags) == 0 {
		return 0, nil
	}
	physicalTags, err := prepared.physicalTags(prepared.plan.Cache.Tags)
	if err != nil {
		return 0, err
	}
	prepared.setTraceTags(physicalTags)
	candidate, attempts = prepared.providers.Candidates[0], 1
	providerLease, providerCtx, err := s.acquireHostCacheProvider(prepared, candidate)
	if err != nil {
		return 0, err
	}
	if providerLease != nil {
		defer providerLease.Release()
	}
	invalidated, backendErr := candidate.Backend.InvalidateTags(providerCtx, physicalTags, prepared.tagPrefix)
	affected = invalidated
	if fenceErr := s.validateHostCacheExecution(prepared, providerLease, providerCtx); fenceErr != nil {
		return 0, fenceErr
	}
	if backendErr != nil {
		return 0, hostCacheBackendError(backendErr)
	}
	return invalidated, nil
}
