package hostapi

import (
	"context"
	"errors"
	"slices"
	"sync"
	"time"
)

type fakeHostCacheItem struct {
	value     HostCacheStoredValue
	expiresAt time.Time
}

type fakeHostCacheCounter struct {
	value     int64
	expiresAt time.Time
}

type fakeHostCacheLock struct {
	owner     string
	expiresAt time.Time
}

type fakeHostCacheBackend struct {
	mu              sync.Mutex
	items           map[string]fakeHostCacheItem
	tags            map[string]map[string]struct{}
	counters        map[string]fakeHostCacheCounter
	locks           map[string]fakeHostCacheLock
	keys            []string
	failure         error
	poison          *HostCacheStoredValue
	afterGet        func()
	afterSet        func()
	afterLock       func()
	afterDelete     func()
	afterInvalidate func()
	afterIncrement  func()
	operationLag    time.Duration
}

func newFakeHostCacheBackend() *fakeHostCacheBackend {
	return &fakeHostCacheBackend{
		items: map[string]fakeHostCacheItem{}, tags: map[string]map[string]struct{}{},
		counters: map[string]fakeHostCacheCounter{}, locks: map[string]fakeHostCacheLock{},
	}
}

func hostCacheTestServiceOptions(extra ...HostCacheServiceOption) []HostCacheServiceOption {
	options := []HostCacheServiceOption{
		WithHostCacheInstallationID("sforum-cache-test-installation"),
		WithHostCacheRuntimeAdmission(&testServiceProviderAdmission{}),
		WithHostCacheTraceSink(NewHostCacheInspector(256)),
	}
	return append(options, extra...)
}

func (b *fakeHostCacheBackend) Get(ctx context.Context, key string) (HostCacheStoredValue, bool, error) {
	if err := b.before(ctx); err != nil {
		return HostCacheStoredValue{}, false, err
	}
	b.mu.Lock()
	if b.poison != nil {
		value := cloneHostCacheStoredValue(*b.poison)
		callback := b.afterGet
		b.mu.Unlock()
		if callback != nil {
			callback()
		}
		return value, true, nil
	}
	item, found := b.items[key]
	if found && !item.expiresAt.After(time.Now()) {
		b.deleteLocked(key)
		found = false
	}
	callback := b.afterGet
	b.mu.Unlock()
	if callback != nil {
		callback()
	}
	if !found {
		return HostCacheStoredValue{}, false, nil
	}
	return cloneHostCacheStoredValue(item.value), true, nil
}

func (b *fakeHostCacheBackend) Set(
	ctx context.Context,
	key string,
	value HostCacheStoredValue,
	ttl time.Duration,
	expectedRevision string,
	_ string,
) error {
	if err := b.before(ctx); err != nil {
		return err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	err := b.setLocked(key, value, ttl, expectedRevision)
	callback := b.afterSet
	if err != nil {
		return err
	}
	if callback != nil {
		b.mu.Unlock()
		callback()
		b.mu.Lock()
	}
	return nil
}

func (b *fakeHostCacheBackend) setLocked(
	key string,
	value HostCacheStoredValue,
	ttl time.Duration,
	expectedRevision string,
) error {
	current, found := b.items[key]
	if found && !current.expiresAt.After(time.Now()) {
		b.deleteLocked(key)
		found = false
	}
	if expectedRevision != "" && (!found || current.value.Revision != expectedRevision) {
		return ErrHostCacheConflict
	}
	if found {
		b.removeTagMembershipsLocked(key, current.value.Tags)
	}
	value = cloneHostCacheStoredValue(value)
	b.items[key] = fakeHostCacheItem{value: value, expiresAt: time.Now().Add(ttl)}
	b.keys = append(b.keys, key)
	for _, tag := range value.Tags {
		if b.tags[tag] == nil {
			b.tags[tag] = map[string]struct{}{}
		}
		b.tags[tag][key] = struct{}{}
	}
	return nil
}

func (b *fakeHostCacheBackend) Delete(ctx context.Context, key, _ string) (bool, error) {
	if err := b.before(ctx); err != nil {
		return false, err
	}
	b.mu.Lock()
	_, found := b.items[key]
	b.deleteLocked(key)
	callback := b.afterDelete
	b.mu.Unlock()
	if callback != nil {
		callback()
	}
	return found, nil
}

func (b *fakeHostCacheBackend) InvalidateTags(ctx context.Context, tags []string, _ string) (uint64, error) {
	if err := b.before(ctx); err != nil {
		return 0, err
	}
	b.mu.Lock()
	keys := map[string]struct{}{}
	for _, tag := range tags {
		for key := range b.tags[tag] {
			keys[key] = struct{}{}
		}
	}
	for key := range keys {
		b.deleteLocked(key)
	}
	for _, tag := range tags {
		delete(b.tags, tag)
	}
	callback := b.afterInvalidate
	b.mu.Unlock()
	if callback != nil {
		callback()
	}
	return uint64(len(keys)), nil
}

func (b *fakeHostCacheBackend) Increment(
	ctx context.Context,
	key string,
	delta int64,
	ttl time.Duration,
) (int64, error) {
	if err := b.before(ctx); err != nil {
		return 0, err
	}
	b.mu.Lock()
	current := b.counters[key]
	if !current.expiresAt.After(time.Now()) {
		current = fakeHostCacheCounter{}
	}
	current.value += delta
	if current.expiresAt.IsZero() {
		current.expiresAt = time.Now().Add(ttl)
	}
	b.counters[key] = current
	b.keys = append(b.keys, key)
	callback := b.afterIncrement
	b.mu.Unlock()
	if callback != nil {
		callback()
	}
	return current.value, nil
}

func (b *fakeHostCacheBackend) AcquireLock(
	ctx context.Context,
	key string,
	owner string,
	ttl time.Duration,
) (bool, error) {
	if err := b.before(ctx); err != nil {
		return false, err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	current, found := b.locks[key]
	if found && current.expiresAt.After(time.Now()) {
		return false, nil
	}
	b.locks[key] = fakeHostCacheLock{owner: owner, expiresAt: time.Now().Add(ttl)}
	b.keys = append(b.keys, key)
	callback := b.afterLock
	if callback != nil {
		b.mu.Unlock()
		callback()
		b.mu.Lock()
	}
	return true, nil
}

func (b *fakeHostCacheBackend) ReleaseLock(ctx context.Context, key, owner string) (bool, error) {
	if err := b.before(ctx); err != nil {
		return false, err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	current, found := b.locks[key]
	if !found || !current.expiresAt.After(time.Now()) || current.owner != owner {
		if found && !current.expiresAt.After(time.Now()) {
			delete(b.locks, key)
		}
		return false, nil
	}
	delete(b.locks, key)
	return true, nil
}

func (b *fakeHostCacheBackend) RenewLock(
	ctx context.Context,
	key string,
	owner string,
	ttl time.Duration,
) (bool, error) {
	if err := b.before(ctx); err != nil {
		return false, err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	current, found := b.locks[key]
	if !found || !current.expiresAt.After(time.Now()) || current.owner != owner {
		if found && !current.expiresAt.After(time.Now()) {
			delete(b.locks, key)
		}
		return false, nil
	}
	current.expiresAt = time.Now().Add(ttl)
	b.locks[key] = current
	return true, nil
}

func (b *fakeHostCacheBackend) SetAndReleaseLock(
	ctx context.Context,
	key string,
	value HostCacheStoredValue,
	ttl time.Duration,
	expectedRevision string,
	_ string,
	lockKey string,
	owner string,
) error {
	if err := b.before(ctx); err != nil {
		return err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	lock, found := b.locks[lockKey]
	if !found || !lock.expiresAt.After(time.Now()) || lock.owner != owner {
		if found && !lock.expiresAt.After(time.Now()) {
			delete(b.locks, lockKey)
		}
		return ErrHostCacheLockNotOwned
	}
	if err := b.setLocked(key, value, ttl, expectedRevision); err != nil {
		return err
	}
	delete(b.locks, lockKey)
	return nil
}

func (b *fakeHostCacheBackend) before(ctx context.Context) error {
	if ctx == nil {
		return ErrHostCacheInvalid
	}
	if b.operationLag > 0 {
		timer := time.NewTimer(b.operationLag)
		select {
		case <-ctx.Done():
			timer.Stop()
			return context.Cause(ctx)
		case <-timer.C:
		}
	}
	b.mu.Lock()
	err := b.failure
	b.mu.Unlock()
	return err
}

func (b *fakeHostCacheBackend) deleteLocked(key string) {
	item, found := b.items[key]
	if found {
		b.removeTagMembershipsLocked(key, item.value.Tags)
	}
	delete(b.items, key)
}

func (b *fakeHostCacheBackend) removeTagMembershipsLocked(key string, tags []string) {
	for _, tag := range tags {
		delete(b.tags[tag], key)
		if len(b.tags[tag]) == 0 {
			delete(b.tags, tag)
		}
	}
}

func (b *fakeHostCacheBackend) observedKeys() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return slices.Clone(b.keys)
}

type fakeHostCacheProviderResolver struct {
	mu         sync.Mutex
	resolution HostCacheProviderResolution
	resolveErr error
	stale      bool
	requests   []HostCacheProviderRequest
}

func (r *fakeHostCacheProviderResolver) ResolveHostCacheProvider(
	_ context.Context,
	request HostCacheProviderRequest,
) (HostCacheProviderResolution, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requests = append(r.requests, request)
	if r.resolveErr != nil {
		return HostCacheProviderResolution{}, r.resolveErr
	}
	return cloneHostCacheProviderResolution(r.resolution), nil
}

func (r *fakeHostCacheProviderResolver) ValidateHostCacheProvider(
	_ context.Context,
	_ HostCacheProviderRequest,
	resolution HostCacheProviderResolution,
	candidate HostCacheProviderCandidate,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.stale || resolution.Revision != r.resolution.Revision {
		return errors.New("stale provider")
	}
	for _, active := range r.resolution.Candidates {
		if active.ProviderID == candidate.ProviderID && active.ExtensionID == candidate.ExtensionID &&
			active.ArtifactDigest == candidate.ArtifactDigest && active.VersionID == candidate.VersionID &&
			active.RuntimeInstance == candidate.RuntimeInstance &&
			active.Backend == candidate.Backend {
			return nil
		}
	}
	return errors.New("provider missing")
}

// delayedResponseHostCacheBackend models a backend that applies the lock TTL
// before network or queue latency delays the response observed by the Host.
type delayedResponseHostCacheBackend struct {
	*fakeHostCacheBackend
	acquireDelay       time.Duration
	renewDelay         time.Duration
	commitDelay        time.Duration
	acquireApplied     chan struct{}
	renewApplied       chan struct{}
	acquireAppliedOnce sync.Once
	renewAppliedOnce   sync.Once
	releaseMu          sync.Mutex
	releaseCalls       int
}

func (b *delayedResponseHostCacheBackend) AcquireLock(
	ctx context.Context,
	key string,
	owner string,
	ttl time.Duration,
) (bool, error) {
	acquired, err := b.fakeHostCacheBackend.AcquireLock(ctx, key, owner, ttl)
	if err != nil || !acquired {
		return acquired, err
	}
	if b.acquireApplied != nil {
		b.acquireAppliedOnce.Do(func() { close(b.acquireApplied) })
	}
	if err := waitHostCacheBackendResponse(ctx, b.acquireDelay); err != nil {
		return acquired, err
	}
	return true, nil
}

func (b *delayedResponseHostCacheBackend) RenewLock(
	ctx context.Context,
	key string,
	owner string,
	ttl time.Duration,
) (bool, error) {
	renewed, err := b.fakeHostCacheBackend.RenewLock(ctx, key, owner, ttl)
	if err != nil || !renewed {
		return renewed, err
	}
	if b.renewApplied != nil {
		b.renewAppliedOnce.Do(func() { close(b.renewApplied) })
	}
	if err := waitHostCacheBackendResponse(ctx, b.renewDelay); err != nil {
		return renewed, err
	}
	return true, nil
}

func (b *delayedResponseHostCacheBackend) SetAndReleaseLock(
	ctx context.Context,
	key string,
	value HostCacheStoredValue,
	ttl time.Duration,
	expectedRevision string,
	tagPrefix string,
	lockKey string,
	owner string,
) error {
	if err := b.fakeHostCacheBackend.SetAndReleaseLock(
		ctx, key, value, ttl, expectedRevision, tagPrefix, lockKey, owner,
	); err != nil {
		return err
	}
	// 模拟服务端已经原子提交，但成功响应晚于调用方的本地保守期限。
	if b.commitDelay > 0 {
		time.Sleep(b.commitDelay)
	}
	return nil
}

func (b *delayedResponseHostCacheBackend) ReleaseLock(ctx context.Context, key, owner string) (bool, error) {
	b.releaseMu.Lock()
	b.releaseCalls++
	b.releaseMu.Unlock()
	return b.fakeHostCacheBackend.ReleaseLock(ctx, key, owner)
}

func (b *delayedResponseHostCacheBackend) observedReleaseCalls() int {
	b.releaseMu.Lock()
	defer b.releaseMu.Unlock()
	return b.releaseCalls
}

func waitHostCacheBackendResponse(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	case <-timer.C:
		return nil
	}
}

type blockingHostCacheLockBackend struct {
	*fakeHostCacheBackend
	releaseStarted chan struct{}
	renewStarted   chan struct{}
	releaseOnce    sync.Once
	renewOnce      sync.Once
}

func (b *blockingHostCacheLockBackend) ReleaseLock(ctx context.Context, key, owner string) (bool, error) {
	if b.releaseStarted == nil {
		return b.fakeHostCacheBackend.ReleaseLock(ctx, key, owner)
	}
	blocked := false
	b.releaseOnce.Do(func() {
		blocked = true
		close(b.releaseStarted)
	})
	if blocked {
		<-ctx.Done()
		return false, context.Cause(ctx)
	}
	return b.fakeHostCacheBackend.ReleaseLock(ctx, key, owner)
}

func (b *blockingHostCacheLockBackend) RenewLock(
	ctx context.Context,
	key string,
	owner string,
	ttl time.Duration,
) (bool, error) {
	if b.renewStarted == nil {
		return b.fakeHostCacheBackend.RenewLock(ctx, key, owner, ttl)
	}
	b.renewOnce.Do(func() { close(b.renewStarted) })
	<-ctx.Done()
	return false, context.Cause(ctx)
}

type cancelableHostCacheAdmission struct {
	mu     sync.Mutex
	cancel context.CancelFunc
}

func (a *cancelableHostCacheAdmission) AcquireServiceProvider(
	ctx context.Context,
	_ ServiceProviderIdentity,
) (ServiceProviderAdmissionLease, error) {
	leaseCtx, cancel := context.WithCancel(ctx)
	a.mu.Lock()
	a.cancel = cancel
	a.mu.Unlock()
	return &cancelableHostCacheLease{ctx: leaseCtx, cancel: cancel}, nil
}

func (a *cancelableHostCacheAdmission) cancelRuntime() {
	a.mu.Lock()
	cancel := a.cancel
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

type cancelableHostCacheLease struct {
	ctx    context.Context
	cancel context.CancelFunc
	once   sync.Once
}

func (l *cancelableHostCacheLease) Context() context.Context { return l.ctx }
func (l *cancelableHostCacheLease) Release() {
	l.once.Do(l.cancel)
}

var _ HostCacheBackend = (*fakeHostCacheBackend)(nil)
var _ HostCacheBackend = (*delayedResponseHostCacheBackend)(nil)
var _ HostCacheBackend = (*blockingHostCacheLockBackend)(nil)
var _ HostCacheProviderResolver = (*fakeHostCacheProviderResolver)(nil)
