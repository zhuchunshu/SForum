package hostapi

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"
)

func (s *HostCacheService) Remember(ctx context.Context, request HostCacheRememberRequest) (result HostCacheGetResult, err error) {
	started := time.Now()
	prepared, err := s.prepare(ctx, request.HostCacheRequestBase)
	if err != nil {
		s.recordRejectedHostCacheTrace(started, request.HostCacheRequestBase, "remember", err)
		return HostCacheGetResult{}, err
	}
	defer prepared.release()
	var candidate HostCacheProviderCandidate
	attempts := 0
	defer func() {
		s.recordHostCacheTrace(started, prepared, candidate, "remember", err, result.Found, 0, attempts)
	}()

	if request.Load == nil || validateHostCacheKey(request.Key) != nil || validateHostCacheSchema(request.Schema) != nil ||
		validateHostCacheTTL(request.TTL) != nil ||
		validateHostCacheRevision(strings.TrimSpace(request.ExpectedRevision), true) != nil {
		return HostCacheGetResult{}, ErrHostCacheInvalid
	}
	request.ExpectedRevision = strings.TrimSpace(request.ExpectedRevision)
	if request.LockTTL == 0 {
		request.LockTTL = HostCacheMaximumLockTTL
	}
	if request.Wait == 0 {
		request.Wait = HostCacheDefaultRememberWait
	}
	if request.Wait < 0 || request.Wait > HostCacheMaximumLockTTL ||
		request.LockTTL < HostCacheMinimumLockTTL || request.LockTTL > HostCacheMaximumLockTTL {
		return HostCacheGetResult{}, ErrHostCacheInvalid
	}
	physicalTags := []string{}
	if len(request.Tags) > 0 {
		physicalTags, err = prepared.physicalTags(request.Tags)
		if err != nil {
			return HostCacheGetResult{}, err
		}
	}
	prepared.setTraceTags(physicalTags)

	candidate, attempts = prepared.providers.Candidates[0], 1
	providerLease, providerCtx, err := s.acquireHostCacheProvider(prepared, candidate)
	if err != nil {
		return HostCacheGetResult{}, err
	}
	if providerLease != nil {
		defer providerLease.Release()
	}
	valueKey, lockKey := prepared.valueKey(request.Key), prepared.lockKey(request.Key)
	if result, err = s.getRememberValue(prepared, candidate, providerLease, providerCtx, valueKey, request.Schema); err != nil || result.Found {
		return result, err
	}

	deadline := time.Now().Add(request.Wait)
	for {
		token, tokenErr := newHostCacheLockToken()
		if tokenErr != nil {
			return HostCacheGetResult{}, ErrHostCacheProviderUnavailable
		}
		acquireStarted := time.Now()
		acquired, acquireErr := candidate.Backend.AcquireLock(providerCtx, lockKey, token, request.LockTTL)
		lockExpiresAt := acquireStarted.Add(request.LockTTL)
		if fenceErr := s.validateHostCacheExecution(prepared, providerLease, providerCtx); fenceErr != nil {
			if acquired || acquireErr != nil {
				releaseHostCacheLockToken(candidate, providerLease, providerCtx, lockKey, token)
			}
			return HostCacheGetResult{}, fenceErr
		}
		if acquireErr != nil {
			releaseHostCacheLockToken(candidate, providerLease, providerCtx, lockKey, token)
			return HostCacheGetResult{}, hostCacheBackendError(acquireErr)
		}
		if acquired {
			if time.Until(lockExpiresAt) <= 0 {
				releaseHostCacheLockToken(candidate, providerLease, providerCtx, lockKey, token)
				return HostCacheGetResult{}, ErrHostCacheLockNotOwned
			}
			return s.loadRememberValue(
				prepared, candidate, providerLease, providerCtx, request,
				valueKey, lockKey, token, lockExpiresAt, physicalTags,
			)
		}
		if !time.Now().Before(deadline) {
			return HostCacheGetResult{}, ErrHostCacheLockNotOwned
		}
		timer := time.NewTimer(25 * time.Millisecond)
		select {
		case <-providerCtx.Done():
			timer.Stop()
			return HostCacheGetResult{}, context.Cause(providerCtx)
		case <-timer.C:
		}
		if result, err = s.getRememberValue(prepared, candidate, providerLease, providerCtx, valueKey, request.Schema); err != nil || result.Found {
			return result, err
		}
	}
}

func (s *HostCacheService) getRememberValue(
	prepared preparedHostCache,
	candidate HostCacheProviderCandidate,
	providerLease ServiceProviderAdmissionLease,
	providerCtx context.Context,
	key string,
	schema HostCacheSchema,
) (HostCacheGetResult, error) {
	stored, found, err := candidate.Backend.Get(providerCtx, key)
	if fenceErr := s.validateHostCacheExecution(prepared, providerLease, providerCtx); fenceErr != nil {
		return HostCacheGetResult{}, fenceErr
	}
	if err != nil {
		return HostCacheGetResult{}, hostCacheBackendError(err)
	}
	if !found {
		return HostCacheGetResult{}, nil
	}
	if validateHostCacheStoredValue(prepared, stored, schema) != nil {
		return HostCacheGetResult{}, ErrHostCachePoisoned
	}
	return HostCacheGetResult{Found: true, Value: slices.Clone(stored.Value), Revision: stored.Revision}, nil
}

func (s *HostCacheService) loadRememberValue(
	prepared preparedHostCache,
	candidate HostCacheProviderCandidate,
	providerLease ServiceProviderAdmissionLease,
	providerCtx context.Context,
	request HostCacheRememberRequest,
	valueKey string,
	lockKey string,
	token string,
	lockExpiresAt time.Time,
	physicalTags []string,
) (result HostCacheGetResult, err error) {
	lockOwned := true
	defer func() {
		if !lockOwned {
			return
		}
		releaseHostCacheLockToken(candidate, providerLease, providerCtx, lockKey, token)
	}()
	if result, err = s.getRememberValue(prepared, candidate, providerLease, providerCtx, valueKey, request.Schema); err != nil || result.Found {
		return result, err
	}

	loaded, lockExpiresAt, err := s.runRememberLoader(
		providerCtx, candidate, lockKey, token, request.LockTTL, lockExpiresAt, request.Load,
	)
	if err != nil {
		if fenceErr := s.validateHostCacheExecution(prepared, providerLease, providerCtx); fenceErr != nil {
			return HostCacheGetResult{}, fenceErr
		}
		return HostCacheGetResult{}, err
	}
	if len(loaded) == 0 || len(loaded) > HostCacheMaximumValueBytes {
		return HostCacheGetResult{}, ErrHostCacheInvalid
	}
	revision, err := newHostCacheRevision()
	if err != nil {
		return HostCacheGetResult{}, ErrHostCacheProviderUnavailable
	}
	stored := HostCacheStoredValue{
		Value: slices.Clone(loaded), SchemaID: strings.TrimSpace(request.Schema.ID),
		SchemaVersion: strings.TrimSpace(request.Schema.Version), Revision: revision, Tags: physicalTags,
	}
	if err := s.validateHostCacheExecution(prepared, providerLease, providerCtx); err != nil {
		return HostCacheGetResult{}, err
	}
	if time.Until(lockExpiresAt) <= 0 {
		return HostCacheGetResult{}, ErrHostCacheLockNotOwned
	}
	commitCtx, cancelCommit := context.WithDeadline(providerCtx, lockExpiresAt)
	defer cancelCommit()
	backendErr := candidate.Backend.SetAndReleaseLock(
		commitCtx, valueKey, stored, request.TTL, request.ExpectedRevision, prepared.tagPrefix, lockKey, token,
	)
	if backendErr == nil {
		// nil is the linearization result: the backend atomically stored the value
		// and deleted this exact token. Response latency must not turn that commit
		// into a false lock-expiry failure.
		lockOwned = false
		return HostCacheGetResult{Found: true, Value: slices.Clone(loaded), Revision: revision}, nil
	}
	if fenceErr := s.validateHostCacheExecution(prepared, providerLease, providerCtx); fenceErr != nil {
		return HostCacheGetResult{}, fenceErr
	}
	return HostCacheGetResult{}, hostCacheBackendError(backendErr)
}

func (s *HostCacheService) runRememberLoader(
	ctx context.Context,
	candidate HostCacheProviderCandidate,
	lockKey string,
	token string,
	lockTTL time.Duration,
	initialExpiresAt time.Time,
	load func(context.Context) ([]byte, error),
) ([]byte, time.Time, error) {
	loadCtx, cancel := context.WithCancelCause(ctx)
	done := make(chan struct{})
	type renewalResult struct {
		expiresAt time.Time
		err       error
	}
	renewed := make(chan renewalResult, 1)
	go func() {
		expiresAt := initialExpiresAt
		for {
			remaining := time.Until(expiresAt)
			if remaining <= 0 {
				err := ErrHostCacheLockNotOwned
				cancel(err)
				renewed <- renewalResult{expiresAt: expiresAt, err: err}
				return
			}
			timer := time.NewTimer(remaining / 3)
			select {
			case <-done:
				timer.Stop()
				if time.Until(expiresAt) <= 0 {
					renewed <- renewalResult{expiresAt: expiresAt, err: ErrHostCacheLockNotOwned}
					return
				}
				renewed <- renewalResult{expiresAt: expiresAt}
				return
			case <-loadCtx.Done():
				timer.Stop()
				renewed <- renewalResult{expiresAt: expiresAt, err: context.Cause(loadCtx)}
				return
			case <-timer.C:
				renewCtx, cancelRenew := context.WithDeadline(loadCtx, expiresAt)
				renewStarted := time.Now()
				ok, err := candidate.Backend.RenewLock(renewCtx, lockKey, token, lockTTL)
				cancelRenew()
				renewedExpiresAt := renewStarted.Add(lockTTL)
				if err != nil || !ok {
					if time.Until(expiresAt) <= 0 {
						err = ErrHostCacheLockNotOwned
					} else if err == nil {
						err = ErrHostCacheLockNotOwned
					} else {
						err = hostCacheBackendError(err)
					}
					cancel(err)
					renewed <- renewalResult{expiresAt: expiresAt, err: err}
					return
				}
				if time.Until(renewedExpiresAt) <= 0 {
					err = ErrHostCacheLockNotOwned
					_, _ = candidate.Backend.ReleaseLock(loadCtx, lockKey, token)
					cancel(err)
					renewed <- renewalResult{expiresAt: renewedExpiresAt, err: err}
					return
				}
				expiresAt = renewedExpiresAt
			}
		}
	}()
	value, loadErr := load(loadCtx)
	close(done)
	renewResult := <-renewed
	cancel(nil)
	if loadErr != nil {
		return nil, renewResult.expiresAt, loadErr
	}
	if renewResult.err != nil && !errors.Is(renewResult.err, context.Canceled) {
		return nil, renewResult.expiresAt, renewResult.err
	}
	if time.Until(renewResult.expiresAt) <= 0 {
		return nil, renewResult.expiresAt, ErrHostCacheLockNotOwned
	}
	return slices.Clone(value), renewResult.expiresAt, nil
}
