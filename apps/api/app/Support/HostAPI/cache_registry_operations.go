package hostapi

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"time"
)

func (s *HostCacheService) Get(ctx context.Context, request HostCacheGetRequest) (result HostCacheGetResult, err error) {
	started := time.Now()
	prepared, err := s.prepare(ctx, request.HostCacheRequestBase)
	if err != nil {
		s.recordRejectedHostCacheTrace(started, request.HostCacheRequestBase, "get", err)
		return HostCacheGetResult{}, err
	}
	defer prepared.release()
	var candidate HostCacheProviderCandidate
	attempts := 0
	defer func() { s.recordHostCacheTrace(started, prepared, candidate, "get", err, result.Found, 0, attempts) }()
	if validateHostCacheKey(request.Key) != nil || validateHostCacheSchema(request.Schema) != nil {
		return HostCacheGetResult{}, ErrHostCacheInvalid
	}
	physicalKey := prepared.valueKey(request.Key)
	var lastErr error
	for index, current := range prepared.providers.Candidates {
		candidate, attempts = current, index+1
		providerLease, providerCtx, validateErr := s.acquireHostCacheProvider(prepared, current)
		if validateErr != nil {
			lastErr = validateErr
			return HostCacheGetResult{}, validateErr
		}
		stored, found, backendErr := current.Backend.Get(providerCtx, physicalKey)
		validateErr = s.validateHostCacheExecution(prepared, providerLease, providerCtx)
		if providerLease != nil {
			providerLease.Release()
		}
		if validateErr != nil {
			return HostCacheGetResult{}, validateErr
		}
		if backendErr != nil {
			lastErr = hostCacheBackendError(backendErr)
			return HostCacheGetResult{}, lastErr
		}
		if !found {
			return HostCacheGetResult{}, nil
		}
		if validateHostCacheStoredValue(prepared, stored, request.Schema) != nil {
			return HostCacheGetResult{}, ErrHostCachePoisoned
		}
		return HostCacheGetResult{Found: true, Value: slices.Clone(stored.Value), Revision: stored.Revision}, nil
	}
	if lastErr == nil {
		lastErr = ErrHostCacheProviderUnavailable
	}
	return HostCacheGetResult{}, lastErr
}

func (s *HostCacheService) Set(ctx context.Context, request HostCacheSetRequest) (revision string, err error) {
	started := time.Now()
	prepared, err := s.prepare(ctx, request.HostCacheRequestBase)
	if err != nil {
		s.recordRejectedHostCacheTrace(started, request.HostCacheRequestBase, "set", err)
		return "", err
	}
	defer prepared.release()
	var candidate HostCacheProviderCandidate
	attempts := 0
	defer func() { s.recordHostCacheTrace(started, prepared, candidate, "set", err, false, 0, attempts) }()
	if validateHostCacheKey(request.Key) != nil || validateHostCacheSchema(request.Schema) != nil ||
		validateHostCacheTTL(request.TTL) != nil || len(request.Value) == 0 || len(request.Value) > HostCacheMaximumValueBytes ||
		validateHostCacheRevision(strings.TrimSpace(request.ExpectedRevision), true) != nil {
		return "", ErrHostCacheInvalid
	}
	request.ExpectedRevision = strings.TrimSpace(request.ExpectedRevision)
	physicalTags := []string{}
	if len(request.Tags) > 0 {
		physicalTags, err = prepared.physicalTags(request.Tags)
		if err != nil {
			return "", err
		}
	}
	prepared.setTraceTags(physicalTags)
	revision, err = newHostCacheRevision()
	if err != nil {
		return "", ErrHostCacheProviderUnavailable
	}
	stored := HostCacheStoredValue{
		Value: slices.Clone(request.Value), SchemaID: strings.TrimSpace(request.Schema.ID),
		SchemaVersion: strings.TrimSpace(request.Schema.Version), Revision: revision, Tags: physicalTags,
	}
	physicalKey := prepared.valueKey(request.Key)
	var lastErr error
	for index, current := range prepared.providers.Candidates {
		candidate, attempts = current, index+1
		providerLease, providerCtx, validateErr := s.acquireHostCacheProvider(prepared, current)
		if validateErr != nil {
			lastErr = validateErr
			return "", validateErr
		}
		backendErr := current.Backend.Set(providerCtx, physicalKey, stored, request.TTL, request.ExpectedRevision, prepared.tagPrefix)
		validateErr = s.validateHostCacheExecution(prepared, providerLease, providerCtx)
		if providerLease != nil {
			providerLease.Release()
		}
		if validateErr != nil {
			return "", validateErr
		}
		if backendErr == nil {
			return revision, nil
		}
		if errors.Is(backendErr, ErrHostCacheConflict) {
			return "", ErrHostCacheConflict
		}
		lastErr = hostCacheBackendError(backendErr)
		return "", lastErr
	}
	if lastErr == nil {
		lastErr = ErrHostCacheProviderUnavailable
	}
	return "", lastErr
}

func (s *HostCacheService) Delete(ctx context.Context, request HostCacheDeleteRequest) (deleted bool, err error) {
	started := time.Now()
	prepared, err := s.prepare(ctx, request.HostCacheRequestBase)
	if err != nil {
		s.recordRejectedHostCacheTrace(started, request.HostCacheRequestBase, "delete", err)
		return false, err
	}
	defer prepared.release()
	var candidate HostCacheProviderCandidate
	attempts := 0
	defer func() {
		affected := uint64(0)
		if deleted {
			affected = 1
		}
		s.recordHostCacheTrace(started, prepared, candidate, "delete", err, false, affected, attempts)
	}()
	if validateHostCacheKey(request.Key) != nil {
		return false, ErrHostCacheInvalid
	}
	physicalKey := prepared.valueKey(request.Key)
	var lastErr error
	for index, current := range prepared.providers.Candidates {
		candidate, attempts = current, index+1
		providerLease, providerCtx, validateErr := s.acquireHostCacheProvider(prepared, current)
		if validateErr != nil {
			lastErr = validateErr
			return false, validateErr
		}
		deleted, backendErr := current.Backend.Delete(providerCtx, physicalKey, prepared.tagPrefix)
		validateErr = s.validateHostCacheExecution(prepared, providerLease, providerCtx)
		if providerLease != nil {
			providerLease.Release()
		}
		if validateErr != nil {
			return false, validateErr
		}
		if backendErr == nil {
			return deleted, nil
		}
		lastErr = hostCacheBackendError(backendErr)
		return false, lastErr
	}
	if lastErr == nil {
		lastErr = ErrHostCacheProviderUnavailable
	}
	return false, lastErr
}

func (s *HostCacheService) InvalidateTags(
	ctx context.Context,
	request HostCacheInvalidateTagsRequest,
) (invalidated uint64, err error) {
	started := time.Now()
	prepared, err := s.prepare(ctx, request.HostCacheRequestBase)
	if err != nil {
		s.recordRejectedHostCacheTrace(started, request.HostCacheRequestBase, "invalidate_tags", err)
		return 0, err
	}
	defer prepared.release()
	var candidate HostCacheProviderCandidate
	attempts := 0
	affected := uint64(0)
	defer func() {
		s.recordHostCacheTrace(started, prepared, candidate, "invalidate_tags", err, false, affected, attempts)
	}()
	physicalTags, err := prepared.physicalTags(request.Tags)
	if err != nil {
		return 0, err
	}
	prepared.setTraceTags(physicalTags)
	var lastErr error
	for index, current := range prepared.providers.Candidates {
		candidate, attempts = current, index+1
		providerLease, providerCtx, validateErr := s.acquireHostCacheProvider(prepared, current)
		if validateErr != nil {
			lastErr = validateErr
			return 0, validateErr
		}
		invalidated, backendErr := current.Backend.InvalidateTags(providerCtx, physicalTags, prepared.tagPrefix)
		affected = invalidated
		validateErr = s.validateHostCacheExecution(prepared, providerLease, providerCtx)
		if providerLease != nil {
			providerLease.Release()
		}
		if validateErr != nil {
			return 0, validateErr
		}
		if backendErr == nil {
			return invalidated, nil
		}
		lastErr = hostCacheBackendError(backendErr)
		return 0, lastErr
	}
	if lastErr == nil {
		lastErr = ErrHostCacheProviderUnavailable
	}
	return 0, lastErr
}

func (s *HostCacheService) Increment(ctx context.Context, request HostCacheIncrementRequest) (value int64, err error) {
	started := time.Now()
	prepared, err := s.prepare(ctx, request.HostCacheRequestBase)
	if err != nil {
		s.recordRejectedHostCacheTrace(started, request.HostCacheRequestBase, "increment", err)
		return 0, err
	}
	defer prepared.release()
	var candidate HostCacheProviderCandidate
	attempts := 0
	defer func() { s.recordHostCacheTrace(started, prepared, candidate, "increment", err, false, 0, attempts) }()
	if validateHostCacheKey(request.Key) != nil || validateHostCacheTTL(request.TTL) != nil {
		return 0, ErrHostCacheInvalid
	}
	if request.Delta == 0 {
		request.Delta = 1
	}
	if request.Delta < -1_000_000 || request.Delta > 1_000_000 {
		return 0, ErrHostCacheInvalid
	}
	var lastErr error
	for index, current := range prepared.providers.Candidates {
		candidate, attempts = current, index+1
		providerLease, providerCtx, validateErr := s.acquireHostCacheProvider(prepared, current)
		if validateErr != nil {
			lastErr = validateErr
			return 0, validateErr
		}
		value, backendErr := current.Backend.Increment(providerCtx, prepared.counterKey(request.Key), request.Delta, request.TTL)
		validateErr = s.validateHostCacheExecution(prepared, providerLease, providerCtx)
		if providerLease != nil {
			providerLease.Release()
		}
		if validateErr != nil {
			return 0, validateErr
		}
		if backendErr != nil {
			// Retrying an ambiguous increment can double-apply it, so fail closed.
			return 0, hostCacheBackendError(backendErr)
		}
		return value, nil
	}
	if lastErr == nil {
		lastErr = ErrHostCacheProviderUnavailable
	}
	return 0, lastErr
}

type HostCacheLock struct {
	mu         sync.Mutex
	service    *HostCacheService
	prepared   preparedHostCache
	candidate  HostCacheProviderCandidate
	provider   ServiceProviderAdmissionLease
	ctx        context.Context
	key        string
	token      string
	released   bool
	timer      *time.Timer
	expiresAt  time.Time
	cancelStop func() bool
}

func (s *HostCacheService) AcquireLock(
	ctx context.Context,
	request HostCacheLockRequest,
) (lock *HostCacheLock, acquired bool, err error) {
	started := time.Now()
	prepared, err := s.prepare(ctx, request.HostCacheRequestBase)
	if err != nil {
		s.recordRejectedHostCacheTrace(started, request.HostCacheRequestBase, "lock_acquire", err)
		return nil, false, err
	}
	releasePrepared := true
	defer func() {
		if releasePrepared {
			prepared.release()
		}
	}()
	var candidate HostCacheProviderCandidate
	attempts := 0
	traceRecorded := false
	defer func() {
		if traceRecorded {
			return
		}
		affected := uint64(0)
		if acquired {
			affected = 1
		}
		s.recordHostCacheTrace(started, prepared, candidate, "lock_acquire", err, false, affected, attempts)
	}()
	if validateHostCacheKey(request.Key) != nil || request.TTL < HostCacheMinimumLockTTL ||
		request.TTL > HostCacheMaximumLockTTL {
		return nil, false, ErrHostCacheInvalid
	}
	token, err := newHostCacheLockToken()
	if err != nil {
		return nil, false, ErrHostCacheProviderUnavailable
	}
	physicalKey := prepared.lockKey(request.Key)
	var lastErr error
	for index, current := range prepared.providers.Candidates {
		candidate, attempts = current, index+1
		providerLease, providerCtx, validateErr := s.acquireHostCacheProvider(prepared, current)
		if validateErr != nil {
			lastErr = validateErr
			return nil, false, validateErr
		}
		acquireStarted := time.Now()
		acquired, backendErr := current.Backend.AcquireLock(providerCtx, physicalKey, token, request.TTL)
		expiresAt := acquireStarted.Add(request.TTL)
		validateErr = s.validateHostCacheExecution(prepared, providerLease, providerCtx)
		if validateErr != nil {
			if acquired || backendErr != nil {
				releaseHostCacheLockToken(current, providerLease, providerCtx, physicalKey, token)
			}
			if providerLease != nil {
				providerLease.Release()
			}
			return nil, false, validateErr
		}
		if backendErr != nil {
			releaseHostCacheLockToken(current, providerLease, providerCtx, physicalKey, token)
			if providerLease != nil {
				providerLease.Release()
			}
			// A timeout may hide a successful SET NX; never acquire elsewhere.
			return nil, false, hostCacheBackendError(backendErr)
		}
		if !acquired {
			if providerLease != nil {
				providerLease.Release()
			}
			return nil, false, nil
		}
		remaining := time.Until(expiresAt)
		if remaining <= 0 {
			// 后端可能在请求期间任意时刻开始 TTL；按调用开始时间保守计时。
			releaseHostCacheLockToken(current, providerLease, providerCtx, physicalKey, token)
			if providerLease != nil {
				providerLease.Release()
			}
			return nil, false, ErrHostCacheLockNotOwned
		}
		releasePrepared = false
		lock := &HostCacheLock{
			service: s, prepared: prepared, candidate: current, provider: providerLease, ctx: providerCtx,
			key: physicalKey, token: token, expiresAt: expiresAt,
		}
		lock.mu.Lock()
		lock.timer = time.AfterFunc(remaining, lock.expire)
		lock.cancelStop = context.AfterFunc(providerCtx, lock.cancel)
		lock.mu.Unlock()
		// Audit is mandatory and synchronous. Recheck ownership after it so a slow
		// sink cannot let the backend TTL expire and still hand a dead lock to the
		// caller as acquired.
		s.recordHostCacheTrace(started, prepared, candidate, "lock_acquire", nil, false, 1, attempts)
		lock.mu.Lock()
		if lock.released {
			lock.mu.Unlock()
			if cause := context.Cause(providerCtx); cause != nil {
				return nil, false, cause
			}
			return nil, false, ErrHostCacheLockNotOwned
		}
		if validateErr := s.validateHostCacheExecution(prepared, providerLease, providerCtx); validateErr != nil {
			releaseHostCacheLockToken(current, providerLease, providerCtx, physicalKey, token)
			lock.released = true
			lock.releaseAdmissionsLocked()
			lock.mu.Unlock()
			return nil, false, validateErr
		}
		if !time.Now().Before(lock.expiresAt) {
			releaseHostCacheLockToken(current, providerLease, providerCtx, physicalKey, token)
			lock.released = true
			lock.releaseAdmissionsLocked()
			lock.mu.Unlock()
			return nil, false, ErrHostCacheLockNotOwned
		}
		lock.mu.Unlock()
		traceRecorded = true
		return lock, true, nil
	}
	if lastErr == nil {
		lastErr = ErrHostCacheProviderUnavailable
	}
	return nil, false, lastErr
}

func (l *HostCacheLock) Release(ctx context.Context) (err error) {
	if l == nil || l.service == nil || ctx == nil {
		return ErrHostCacheLockNotOwned
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.released {
		return ErrHostCacheLockNotOwned
	}
	l.released = true
	defer l.releaseAdmissionsLocked()
	started := time.Now()
	defer func() {
		l.service.recordHostCacheTrace(started, l.prepared, l.candidate, "lock_release", err, false, 0, 1)
	}()
	if err := ctx.Err(); err != nil {
		releaseHostCacheLockToken(l.candidate, l.provider, l.ctx, l.key, l.token)
		return context.Cause(ctx)
	}
	if err := l.service.validateHostCacheLease(l.provider, l.ctx); err != nil {
		releaseHostCacheLockToken(l.candidate, l.provider, l.ctx, l.key, l.token)
		return err
	}
	if err := l.service.validateHostCacheLease(l.prepared.ownerLease, l.prepared.executionCtx); err != nil {
		releaseHostCacheLockToken(l.candidate, l.provider, l.ctx, l.key, l.token)
		return err
	}
	operationCtx, finishOperation := hostCacheLockOperationContext(l.ctx, ctx)
	defer finishOperation()
	released, backendErr := l.candidate.Backend.ReleaseLock(operationCtx, l.key, l.token)
	if validateErr := l.service.validateHostCacheExecution(l.prepared, l.provider, l.ctx); validateErr != nil {
		releaseHostCacheLockToken(l.candidate, l.provider, l.ctx, l.key, l.token)
		return validateErr
	}
	if cancellationErr := hostCacheContextCancellation(operationCtx); cancellationErr != nil {
		releaseHostCacheLockToken(l.candidate, l.provider, l.ctx, l.key, l.token)
		return cancellationErr
	}
	if backendErr != nil {
		releaseHostCacheLockToken(l.candidate, l.provider, l.ctx, l.key, l.token)
		return hostCacheBackendError(backendErr)
	}
	if !released {
		return ErrHostCacheLockNotOwned
	}
	return nil
}

func (l *HostCacheLock) Renew(ctx context.Context, ttl time.Duration) (err error) {
	if l == nil || l.service == nil || ctx == nil || ttl < HostCacheMinimumLockTTL || ttl > HostCacheMaximumLockTTL {
		return ErrHostCacheInvalid
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.released {
		return ErrHostCacheLockNotOwned
	}
	started := time.Now()
	defer func() {
		l.service.recordHostCacheTrace(started, l.prepared, l.candidate, "lock_renew", err, false, 0, 1)
	}()
	if err := ctx.Err(); err != nil {
		return context.Cause(ctx)
	}
	if err := l.service.validateHostCacheExecution(l.prepared, l.provider, l.ctx); err != nil {
		return err
	}
	if !time.Now().Before(l.expiresAt) {
		releaseHostCacheLockToken(l.candidate, l.provider, l.ctx, l.key, l.token)
		l.released = true
		l.releaseAdmissionsLocked()
		return ErrHostCacheLockNotOwned
	}
	operationCtx, finishOperation := hostCacheLockOperationContext(l.ctx, ctx)
	renewCtx, cancelAtExpiry := context.WithDeadline(operationCtx, l.expiresAt)
	defer func() {
		cancelAtExpiry()
		finishOperation()
	}()
	renewStarted := time.Now()
	renewed, backendErr := l.candidate.Backend.RenewLock(renewCtx, l.key, l.token, ttl)
	expiresAt := renewStarted.Add(ttl)
	if validateErr := l.service.validateHostCacheExecution(l.prepared, l.provider, l.ctx); validateErr != nil {
		releaseHostCacheLockToken(l.candidate, l.provider, l.ctx, l.key, l.token)
		l.released = true
		l.releaseAdmissionsLocked()
		return validateErr
	}
	if !time.Now().Before(l.expiresAt) {
		releaseHostCacheLockToken(l.candidate, l.provider, l.ctx, l.key, l.token)
		l.released = true
		l.releaseAdmissionsLocked()
		return ErrHostCacheLockNotOwned
	}
	if cancellationErr := hostCacheContextCancellation(operationCtx); cancellationErr != nil {
		releaseHostCacheLockToken(l.candidate, l.provider, l.ctx, l.key, l.token)
		l.released = true
		l.releaseAdmissionsLocked()
		return cancellationErr
	}
	if backendErr != nil {
		releaseHostCacheLockToken(l.candidate, l.provider, l.ctx, l.key, l.token)
		l.released = true
		l.releaseAdmissionsLocked()
		return hostCacheBackendError(backendErr)
	}
	if !renewed {
		l.released = true
		l.releaseAdmissionsLocked()
		return ErrHostCacheLockNotOwned
	}
	remaining := time.Until(expiresAt)
	if remaining <= 0 {
		releaseHostCacheLockToken(l.candidate, l.provider, l.ctx, l.key, l.token)
		l.released = true
		l.releaseAdmissionsLocked()
		return ErrHostCacheLockNotOwned
	}
	l.expiresAt = expiresAt
	if l.timer != nil {
		l.timer.Reset(remaining)
	}
	return nil
}

func hostCacheLockOperationContext(leaseCtx, callerCtx context.Context) (context.Context, func()) {
	operationCtx, cancel := context.WithCancelCause(leaseCtx)
	stopCaller := context.AfterFunc(callerCtx, func() {
		cancel(context.Cause(callerCtx))
	})
	deadlineCancel := func() {}
	if deadline, ok := callerCtx.Deadline(); ok {
		operationCtx, deadlineCancel = context.WithDeadline(operationCtx, deadline)
	}
	return operationCtx, func() {
		deadlineCancel()
		stopCaller()
		cancel(nil)
	}
}

func hostCacheContextCancellation(ctx context.Context) error {
	if ctx == nil || ctx.Err() == nil {
		return nil
	}
	if errors.Is(context.Cause(ctx), context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	return context.Canceled
}

func releaseHostCacheLockToken(
	candidate HostCacheProviderCandidate,
	lease ServiceProviderAdmissionLease,
	leaseCtx context.Context,
	key string,
	token string,
) {
	if candidate.Backend == nil || leaseCtx == nil {
		return
	}
	cleanupBase := leaseCtx
	if !candidate.Core {
		// An executable provider must never be called after its exact-runtime lease
		// has been revoked; its TTL is the fail-closed reclamation boundary then.
		if lease == nil || leaseCtx.Err() != nil {
			return
		}
		if failure, ok := lease.(ServiceProviderAdmissionLeaseFailure); ok && failure.ServiceProviderAdmissionFailure() != nil {
			return
		}
	} else if leaseCtx.Err() != nil {
		// Core cleanup is Host-owned and may complete after caller/runtime cancel.
		cleanupBase = context.WithoutCancel(leaseCtx)
	}
	cleanupCtx, cancel := context.WithTimeout(cleanupBase, time.Second)
	defer cancel()
	_, _ = candidate.Backend.ReleaseLock(cleanupCtx, key, token)
}

func (l *HostCacheLock) releaseAdmissionsLocked() {
	if l.timer != nil {
		l.timer.Stop()
		l.timer = nil
	}
	if l.cancelStop != nil {
		l.cancelStop()
		l.cancelStop = nil
	}
	if l.provider != nil {
		l.provider.Release()
		l.provider = nil
	}
	l.prepared.release()
}

func (l *HostCacheLock) expire() {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.released {
		return
	}
	if remaining := time.Until(l.expiresAt); remaining > 0 {
		l.timer = time.AfterFunc(remaining, l.expire)
		return
	}
	releaseHostCacheLockToken(l.candidate, l.provider, l.ctx, l.key, l.token)
	l.released = true
	l.service.recordHostCacheTrace(time.Now(), l.prepared, l.candidate, "lock_expire", nil, false, 1, 1)
	l.releaseAdmissionsLocked()
}

func (l *HostCacheLock) cancel() {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.released {
		return
	}
	releaseHostCacheLockToken(l.candidate, l.provider, l.ctx, l.key, l.token)
	l.released = true
	l.service.recordHostCacheTrace(time.Now(), l.prepared, l.candidate, "lock_cancel", context.Cause(l.ctx), false, 1, 1)
	l.releaseAdmissionsLocked()
}

func hostCacheBackendError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, ErrHostCacheConflict) {
		return ErrHostCacheConflict
	}
	if errors.Is(err, ErrHostCachePoisoned) {
		return ErrHostCachePoisoned
	}
	if errors.Is(err, ErrHostCacheInvalid) {
		return ErrHostCacheInvalid
	}
	if errors.Is(err, ErrHostCacheLockNotOwned) {
		return ErrHostCacheLockNotOwned
	}
	return ErrHostCacheProviderUnavailable
}
