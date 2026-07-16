package hostapi

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
)

var ErrServiceRuntimeSetConflict = errors.New("service runtime set changed while publication was prepared")

// ServiceRuntimeSetTransaction is a validated, complete desired runtime graph.
// Preparing never changes reader visibility; Commit publishes one immutable
// snapshot only when no independent writer changed the captured base graph.
type ServiceRuntimeSetTransaction struct {
	mu       sync.Mutex
	registry *ServiceRegistry
	base     *serviceRegistrySnapshot
	next     *serviceRegistrySnapshot
	done     bool
}

// ServiceRuntimeSetLease prevents service writers from changing a verified
// complete runtime graph while another host-owned registry commit finishes.
type ServiceRuntimeSetLease struct {
	mu       sync.Mutex
	registry *ServiceRegistry
	held     bool
}

func (l *ServiceRuntimeSetLease) Release() {
	if l == nil {
		return
	}
	l.mu.Lock()
	if !l.held || l.registry == nil {
		l.mu.Unlock()
		return
	}
	l.held = false
	registry := l.registry
	l.mu.Unlock()
	registry.writeMu.Unlock()
}

// ReplaceRuntimeSet swaps to another complete graph without releasing the
// writer fence. It is the rollback half of an aggregate runtime transition.
func (l *ServiceRuntimeSetLease) ReplaceRuntimeSet(publications []ServiceRuntimePublication) error {
	if l == nil {
		return fmt.Errorf("%w: runtime set lease is nil", ErrInvalidServiceRegistration)
	}
	next, err := prepareServiceRuntimeSetSnapshot(publications)
	if err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.held || l.registry == nil {
		return fmt.Errorf("%w: runtime set lease is closed", ErrInvalidServiceRegistration)
	}
	current := l.registry.loadSnapshot()
	next.revision = current.revision + 1
	l.registry.snapshot.Store(next)
	return nil
}

// PrepareRuntimeSet validates the complete desired Protocol V2 runtime set.
// The resulting transaction replaces every runtime, dependency and service
// registration in the registry; omitted extensions are deliberately removed.
func (r *ServiceRegistry) PrepareRuntimeSet(publications []ServiceRuntimePublication) (*ServiceRuntimeSetTransaction, error) {
	if r == nil {
		return nil, fmt.Errorf("%w: registry is nil", ErrInvalidServiceRegistration)
	}
	next, err := prepareServiceRuntimeSetSnapshot(publications)
	if err != nil {
		return nil, err
	}
	return &ServiceRuntimeSetTransaction{registry: r, base: r.loadSnapshot(), next: next}, nil
}

func prepareServiceRuntimeSetSnapshot(publications []ServiceRuntimePublication) (*serviceRegistrySnapshot, error) {
	runtimes := make(map[string]preparedServiceRuntime, len(publications))
	registrations := make([]preparedServiceRegistration, 0)
	for _, publication := range publications {
		preparedRuntime, preparedServices, err := prepareServiceRuntimePublication(publication)
		if err != nil {
			return nil, err
		}
		extensionID := preparedRuntime.publication.ExtensionID
		if _, exists := runtimes[extensionID]; exists {
			return nil, fmt.Errorf("%w: runtime set declares extension %q more than once", ErrInvalidServiceRegistration, extensionID)
		}
		runtimes[extensionID] = preparedRuntime
		registrations = append(registrations, preparedServices...)
	}
	sortPreparedServices(registrations)
	return &serviceRegistrySnapshot{
		registrations: registrations,
		runtimes:      runtimes,
		// A complete runtime publication always enables exact dependency
		// enforcement, including the intentionally empty desired set.
		dependencyEnforced: true,
	}, nil
}

// ReplaceRuntimeSet is the one-shot complete-set compatibility entrypoint.
// Callers that coordinate other registries should use PrepareRuntimeSet and
// defer Commit until their remaining validation has succeeded.
func (r *ServiceRegistry) ReplaceRuntimeSet(publications []ServiceRuntimePublication) error {
	return r.ReplaceRuntimeSetContext(context.Background(), publications)
}

// ReplaceRuntimeSetContext is the cancellable complete-set compatibility
// entrypoint. Cancellation only applies before the immutable snapshot swap.
func (r *ServiceRegistry) ReplaceRuntimeSetContext(ctx context.Context, publications []ServiceRuntimePublication) error {
	transaction, err := r.PrepareRuntimeSet(publications)
	if err != nil {
		return err
	}
	return transaction.CommitContext(ctx)
}

// RuntimeSetMatches verifies the complete exact runtime graph without changing
// the revision. It is used by idempotent full-set convergence after ambiguous
// acknowledgements or instance-bound compensation.
func (r *ServiceRegistry) RuntimeSetMatches(publications []ServiceRuntimePublication) (bool, error) {
	if r == nil {
		return false, fmt.Errorf("%w: registry is nil", ErrInvalidServiceRegistration)
	}
	desired, err := prepareServiceRuntimeSetSnapshot(publications)
	if err != nil {
		return false, err
	}
	return serviceRuntimeSetSnapshotsEqual(r.loadSnapshot(), desired), nil
}

// AcquireRuntimeSet verifies the exact complete graph and holds the writer
// boundary until Release. Lock-free readers continue using the immutable
// snapshot while raw lifecycle writers wait for the coordinated commit.
func (r *ServiceRegistry) AcquireRuntimeSet(publications []ServiceRuntimePublication) (*ServiceRuntimeSetLease, error) {
	return r.AcquireRuntimeSetContext(context.Background(), publications)
}

// AcquireRuntimeSetContext verifies and pins the exact complete graph while
// allowing a queued writer-fence wait to stop with the caller context.
func (r *ServiceRegistry) AcquireRuntimeSetContext(ctx context.Context, publications []ServiceRuntimePublication) (*ServiceRuntimeSetLease, error) {
	if r == nil {
		return nil, fmt.Errorf("%w: registry is nil", ErrInvalidServiceRegistration)
	}
	desired, err := prepareServiceRuntimeSetSnapshot(publications)
	if err != nil {
		return nil, err
	}
	if err := lockServiceRuntimeSetWriter(ctx, r); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		r.writeMu.Unlock()
		return nil, err
	}
	if !serviceRuntimeSetSnapshotsEqual(r.loadSnapshot(), desired) {
		r.writeMu.Unlock()
		return nil, ErrServiceRuntimeSetConflict
	}
	return &ServiceRuntimeSetLease{registry: r, held: true}, nil
}

func serviceRuntimeSetSnapshotsEqual(left, right *serviceRegistrySnapshot) bool {
	if left == nil || right == nil || left.dependencyEnforced != right.dependencyEnforced ||
		len(left.runtimes) != len(right.runtimes) || len(left.registrations) != len(right.registrations) {
		return false
	}
	for extensionID, rightRuntime := range right.runtimes {
		leftRuntime, ok := left.runtimes[extensionID]
		if !ok || !serviceRuntimeEqual(leftRuntime, rightRuntime) {
			return false
		}
	}
	for index := range right.registrations {
		leftRegistration := left.registrations[index]
		rightRegistration := right.registrations[index]
		if leftRegistration.publishedID != rightRegistration.publishedID ||
			leftRegistration.registration.ExtensionID != rightRegistration.registration.ExtensionID ||
			leftRegistration.registration.InstanceID != rightRegistration.registration.InstanceID ||
			leftRegistration.registration.Action != rightRegistration.registration.Action ||
			leftRegistration.registration.TargetID != rightRegistration.registration.TargetID ||
			leftRegistration.registration.Priority != rightRegistration.registration.Priority ||
			!proto.Equal(leftRegistration.registration.Descriptor, rightRegistration.registration.Descriptor) ||
			!sameServiceProvider(leftRegistration.registration.Provider, rightRegistration.registration.Provider) {
			return false
		}
	}
	return true
}

func serviceRuntimeEqual(left, right preparedServiceRuntime) bool {
	leftPublication := left.publication
	rightPublication := right.publication
	if leftPublication.ExtensionID != rightPublication.ExtensionID ||
		leftPublication.ExtensionVersion != rightPublication.ExtensionVersion ||
		leftPublication.ArtifactDigest != rightPublication.ArtifactDigest ||
		leftPublication.TrustGrantID != rightPublication.TrustGrantID ||
		leftPublication.RuntimeEpoch != rightPublication.RuntimeEpoch ||
		leftPublication.InstanceID != rightPublication.InstanceID ||
		len(left.dependencies) != len(right.dependencies) || len(left.provides) != len(right.provides) {
		return false
	}
	for index := range right.dependencies {
		if left.dependencies[index].dependency != right.dependencies[index].dependency {
			return false
		}
	}
	for index := range right.provides {
		if left.provides[index].capability != right.provides[index].capability {
			return false
		}
	}
	return true
}

func sameServiceProvider(left, right ServiceProvider) bool {
	leftValue := reflect.ValueOf(left)
	rightValue := reflect.ValueOf(right)
	if !leftValue.IsValid() || !rightValue.IsValid() || leftValue.Type() != rightValue.Type() || !leftValue.Comparable() {
		return false
	}
	return leftValue.Interface() == rightValue.Interface()
}

// Commit publishes the prepared graph at one revision. A stale transaction
// fails closed and leaves the independently published graph untouched.
func (t *ServiceRuntimeSetTransaction) Commit() error {
	return t.CommitContext(context.Background())
}

// CommitContext publishes the prepared graph unless the writer-fence wait is
// canceled first.
func (t *ServiceRuntimeSetTransaction) CommitContext(ctx context.Context) error {
	lease, err := t.commit(ctx, false)
	lease.Release()
	return err
}

// CommitAndAcquire publishes the prepared graph and retains the registry
// writer fence. The caller must Release the returned lease.
func (t *ServiceRuntimeSetTransaction) CommitAndAcquire() (*ServiceRuntimeSetLease, error) {
	return t.CommitAndAcquireContext(context.Background())
}

// CommitAndAcquireContext publishes the graph and retains the writer fence.
// A canceled waiter leaves the transaction reusable until Abort is called.
func (t *ServiceRuntimeSetTransaction) CommitAndAcquireContext(ctx context.Context) (*ServiceRuntimeSetLease, error) {
	return t.commit(ctx, true)
}

func (t *ServiceRuntimeSetTransaction) commit(ctx context.Context, retain bool) (*ServiceRuntimeSetLease, error) {
	if t == nil {
		return nil, fmt.Errorf("%w: runtime set transaction is nil", ErrInvalidServiceRegistration)
	}
	if ctx == nil {
		return nil, fmt.Errorf("%w: context is nil", ErrInvalidServiceRegistration)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.done || t.registry == nil || t.base == nil || t.next == nil {
		return nil, fmt.Errorf("%w: runtime set transaction is closed", ErrInvalidServiceRegistration)
	}

	if err := lockServiceRuntimeSetWriter(ctx, t.registry); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		t.registry.writeMu.Unlock()
		return nil, err
	}
	current := t.registry.loadSnapshot()
	if current != t.base {
		t.done = true
		t.registry.writeMu.Unlock()
		return nil, ErrServiceRuntimeSetConflict
	}
	t.next.revision = current.revision + 1
	t.registry.snapshot.Store(t.next)
	t.done = true
	lease := &ServiceRuntimeSetLease{registry: t.registry, held: true}
	if !retain {
		lease.Release()
	}
	return lease, nil
}

// sync.Mutex cannot remove a canceled waiter. TryLock keeps raw legacy writers
// on the same fence while making aggregate publication admission cancellable.
func lockServiceRuntimeSetWriter(ctx context.Context, registry *ServiceRegistry) error {
	if ctx == nil {
		return fmt.Errorf("%w: context is nil", ErrInvalidServiceRegistration)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if registry.writeMu.TryLock() {
		return nil
	}

	timer := time.NewTimer(time.Millisecond)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
		if registry.writeMu.TryLock() {
			return nil
		}
		timer.Reset(time.Millisecond)
	}
}

// Abort closes a prepared transaction without changing the active snapshot.
// It is safe to call more than once and after a failed Commit.
func (t *ServiceRuntimeSetTransaction) Abort() {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.done = true
	t.mu.Unlock()
}
