package extensions

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var errThemeRuntimeSafetyApply = errors.New("theme runtime safety apply failed")

type themeRuntimeSafetyRepository struct {
	mu sync.Mutex

	identity    ThemeRuntimeNodeIdentity
	publication ThemeRuntimePublication
	node        ThemeRuntimeNode
	ackRevision int64

	ensureCalls   atomic.Int32
	registerCalls atomic.Int32

	ensureFn    func(context.Context) (ThemeRuntimePublication, error)
	heartbeatFn func(context.Context, ThemeRuntimeNodeIdentity, time.Duration) (ThemeRuntimeNode, error)
	beginFn     func(context.Context, ThemeRuntimeNodeIdentity, int64) (ThemeRuntimePublicationAck, error)
	completeFn  func(context.Context, ThemeRuntimeNodeIdentity, ThemeRuntimePublication, int64) (ThemeRuntimePublicationAck, error)
	failFn      func(context.Context, ThemeRuntimeNodeIdentity, int64, int64, string) (ThemeRuntimePublicationAck, error)
}

func newThemeRuntimeSafetyRepository(
	identity ThemeRuntimeNodeIdentity,
	publication ThemeRuntimePublication,
) *themeRuntimeSafetyRepository {
	return &themeRuntimeSafetyRepository{
		identity: identity, publication: publication,
		node: ThemeRuntimeNode{ThemeRuntimeNodeIdentity: identity},
	}
}

func (r *themeRuntimeSafetyRepository) EnsureInitialThemeRuntimePublication(
	ctx context.Context,
) (ThemeRuntimePublication, error) {
	r.ensureCalls.Add(1)
	if r.ensureFn != nil {
		return r.ensureFn(ctx)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.publication, nil
}

func (r *themeRuntimeSafetyRepository) RegisterThemeRuntimeNode(
	_ context.Context,
	identity ThemeRuntimeNodeIdentity,
	_ time.Duration,
) (ThemeRuntimeNode, error) {
	r.registerCalls.Add(1)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.identity = identity
	r.node.ThemeRuntimeNodeIdentity = identity
	return r.node, nil
}

func (r *themeRuntimeSafetyRepository) HeartbeatThemeRuntimeNode(
	ctx context.Context,
	identity ThemeRuntimeNodeIdentity,
	lease time.Duration,
) (ThemeRuntimeNode, error) {
	if r.heartbeatFn != nil {
		return r.heartbeatFn(ctx, identity, lease)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.node, nil
}

func (r *themeRuntimeSafetyRepository) GetThemeRuntimeNode(
	context.Context,
	ThemeRuntimeNodeIdentity,
) (ThemeRuntimeNode, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.node, nil
}

func (r *themeRuntimeSafetyRepository) BeginThemeRuntimePublicationApply(
	ctx context.Context,
	identity ThemeRuntimeNodeIdentity,
	publicationRevision int64,
) (ThemeRuntimePublicationAck, error) {
	if r.beginFn != nil {
		return r.beginFn(ctx, identity, publicationRevision)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ackRevision++
	return ThemeRuntimePublicationAck{
		PublicationRevision:      publicationRevision,
		ThemeRuntimeNodeIdentity: identity,
		Status:                   ThemeRuntimeAckApplying, AttemptCount: 1, Revision: r.ackRevision,
	}, nil
}

func (r *themeRuntimeSafetyRepository) CompleteThemeRuntimePublicationApply(
	ctx context.Context,
	identity ThemeRuntimeNodeIdentity,
	publication ThemeRuntimePublication,
	expectedAckRevision int64,
) (ThemeRuntimePublicationAck, error) {
	if r.completeFn != nil {
		return r.completeFn(ctx, identity, publication, expectedAckRevision)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.node.LastAppliedRevision = publication.Revision
	r.ackRevision = max(r.ackRevision, expectedAckRevision) + 1
	return completedThemeRuntimeSafetyAck(identity, publication, expectedAckRevision, r.ackRevision), nil
}

func (r *themeRuntimeSafetyRepository) FailThemeRuntimePublicationApply(
	ctx context.Context,
	identity ThemeRuntimeNodeIdentity,
	publicationRevision int64,
	expectedAckRevision int64,
	reason string,
) (ThemeRuntimePublicationAck, error) {
	if r.failFn != nil {
		return r.failFn(ctx, identity, publicationRevision, expectedAckRevision, reason)
	}
	return ThemeRuntimePublicationAck{
		PublicationRevision:      publicationRevision,
		ThemeRuntimeNodeIdentity: identity,
		Status:                   ThemeRuntimeAckFailed, ErrorReason: reason,
		AttemptCount: 1, Revision: expectedAckRevision + 1,
	}, nil
}

func (r *themeRuntimeSafetyRepository) LatestThemeRuntimePublication(
	context.Context,
) (ThemeRuntimePublication, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.publication, nil
}

func (*themeRuntimeSafetyRepository) ThemeRuntimePublicationByRevision(
	context.Context,
	int64,
) (ThemeRuntimePublication, error) {
	return ThemeRuntimePublication{}, ErrThemePublicationNotFound
}

type themeRuntimeSafetyApplier func(context.Context, ThemeRuntimePublication) error

func (apply themeRuntimeSafetyApplier) ApplyThemeRuntimePublication(
	ctx context.Context,
	publication ThemeRuntimePublication,
) error {
	return apply(ctx, publication)
}

func TestThemeRuntimeWatcherHeartbeatFailureCancelsInFlightApply(t *testing.T) {
	for _, test := range []struct {
		name          string
		heartbeat     func(context.Context) error
		expectedError error
	}{
		{
			name: "database error",
			heartbeat: func(context.Context) error {
				return ErrThemeRuntimeNodeLeaseLost
			},
			expectedError: ErrThemeRuntimeNodeLeaseLost,
		},
		{
			name: "database timeout",
			heartbeat: func(ctx context.Context) error {
				<-ctx.Done()
				return ctx.Err()
			},
			expectedError: context.DeadlineExceeded,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			identity := ThemeRuntimeNodeIdentity{NodeID: "api-heartbeat", BootID: "boot-heartbeat"}
			repository := newThemeRuntimeSafetyRepository(identity, themeRuntimeSafetyPublication(1, "a"))
			heartbeatStarted := make(chan struct{})
			var heartbeatOnce sync.Once
			repository.heartbeatFn = func(
				ctx context.Context,
				_ ThemeRuntimeNodeIdentity,
				_ time.Duration,
			) (ThemeRuntimeNode, error) {
				heartbeatOnce.Do(func() { close(heartbeatStarted) })
				return ThemeRuntimeNode{}, test.heartbeat(ctx)
			}
			applyStarted := make(chan struct{})
			applyCanceled := make(chan struct{})
			applier := themeRuntimeSafetyApplier(func(ctx context.Context, _ ThemeRuntimePublication) error {
				close(applyStarted)
				<-ctx.Done()
				close(applyCanceled)
				return ctx.Err()
			})
			var ready atomic.Bool
			watcher := newThemeRuntimeSafetyWatcher(t, repository, applier, ThemeRuntimeWatcherConfig{
				Identity: identity, LeaseDuration: 90 * time.Millisecond,
				HeartbeatInterval: 20 * time.Millisecond, PollInterval: time.Hour,
				OnReady: func() { ready.Store(true) },
			})
			result := make(chan error, 1)
			go func() { result <- watcher.Run(context.Background()) }()

			waitThemeRuntimeSafetySignal(t, applyStarted)
			waitThemeRuntimeSafetySignal(t, heartbeatStarted)
			waitThemeRuntimeSafetySignal(t, applyCanceled)
			if err := waitThemeRuntimeSafetyResult(t, result); !errors.Is(err, test.expectedError) {
				t.Fatalf("watcher terminal error=%v, want %v", err, test.expectedError)
			}
			if ready.Load() {
				t.Fatal("watcher became ready after heartbeat ownership was lost")
			}
		})
	}
}

func TestThemeRuntimeWatcherAckLeaseLossIsTerminal(t *testing.T) {
	for _, operation := range []string{"begin", "complete", "fail"} {
		t.Run(operation, func(t *testing.T) {
			identity := ThemeRuntimeNodeIdentity{NodeID: "api-ack-" + operation, BootID: "boot-ack"}
			repository := newThemeRuntimeSafetyRepository(identity, themeRuntimeSafetyPublication(1, "b"))
			apply := themeRuntimeSafetyApplier(func(context.Context, ThemeRuntimePublication) error { return nil })
			switch operation {
			case "begin":
				repository.beginFn = func(
					context.Context,
					ThemeRuntimeNodeIdentity,
					int64,
				) (ThemeRuntimePublicationAck, error) {
					return ThemeRuntimePublicationAck{}, ErrThemeRuntimeNodeLeaseLost
				}
			case "complete":
				repository.completeFn = func(
					context.Context,
					ThemeRuntimeNodeIdentity,
					ThemeRuntimePublication,
					int64,
				) (ThemeRuntimePublicationAck, error) {
					return ThemeRuntimePublicationAck{}, ErrThemeRuntimeNodeLeaseLost
				}
			case "fail":
				apply = func(context.Context, ThemeRuntimePublication) error { return errThemeRuntimeSafetyApply }
				repository.failFn = func(
					context.Context,
					ThemeRuntimeNodeIdentity,
					int64,
					int64,
					string,
				) (ThemeRuntimePublicationAck, error) {
					return ThemeRuntimePublicationAck{}, ErrThemeRuntimeNodeLeaseLost
				}
			}
			var ready atomic.Bool
			watcher := newThemeRuntimeSafetyWatcher(t, repository, apply, ThemeRuntimeWatcherConfig{
				Identity: identity, LeaseDuration: time.Second,
				HeartbeatInterval: 100 * time.Millisecond, PollInterval: time.Hour,
				OnReady: func() { ready.Store(true) },
			})
			if err := watcher.Run(context.Background()); !errors.Is(err, ErrThemeRuntimeNodeLeaseLost) {
				t.Fatalf("%s lease-loss error=%v", operation, err)
			}
			if ready.Load() {
				t.Fatalf("watcher became ready after %s lost its lease", operation)
			}
		})
	}
}

func TestThemeRuntimeWatcherReadyWaitsForRevisionPublishedDuringInitialApply(t *testing.T) {
	identity := ThemeRuntimeNodeIdentity{NodeID: "api-catch-up", BootID: "boot-catch-up"}
	first := themeRuntimeSafetyPublication(40, "c")
	second := themeRuntimeSafetyPublication(41, "d")
	repository := newThemeRuntimeSafetyRepository(identity, first)
	var ready atomic.Bool
	var applyCount atomic.Int32
	var completeCount atomic.Int32
	eventsMu := sync.Mutex{}
	events := make([]string, 0, 5)
	record := func(event string) {
		eventsMu.Lock()
		events = append(events, event)
		eventsMu.Unlock()
	}
	applier := themeRuntimeSafetyApplier(func(_ context.Context, publication ThemeRuntimePublication) error {
		if ready.Load() {
			t.Error("OnReady ran before the initial apply returned")
		}
		applyCount.Add(1)
		record("apply-" + strconv.FormatInt(publication.Revision, 10))
		if publication.Revision == first.Revision {
			repository.mu.Lock()
			repository.publication = second
			repository.mu.Unlock()
		}
		return nil
	})
	repository.completeFn = func(
		_ context.Context,
		ackIdentity ThemeRuntimeNodeIdentity,
		publication ThemeRuntimePublication,
		expectedAckRevision int64,
	) (ThemeRuntimePublicationAck, error) {
		if ready.Load() {
			t.Error("OnReady ran before the durable acknowledgement")
		}
		completeCount.Add(1)
		record("complete-" + strconv.FormatInt(publication.Revision, 10))
		repository.mu.Lock()
		defer repository.mu.Unlock()
		repository.node.LastAppliedRevision = publication.Revision
		repository.ackRevision = max(repository.ackRevision, expectedAckRevision) + 1
		return completedThemeRuntimeSafetyAck(
			ackIdentity, publication, expectedAckRevision, repository.ackRevision,
		), nil
	}
	readySignal := make(chan struct{})
	watcher := newThemeRuntimeSafetyWatcher(t, repository, applier, ThemeRuntimeWatcherConfig{
		Identity: identity, LeaseDuration: time.Second,
		HeartbeatInterval: 100 * time.Millisecond, PollInterval: time.Hour,
		OnReady: func() {
			if applyCount.Load() != 2 || completeCount.Load() != 2 {
				t.Errorf("ready at applies=%d completes=%d", applyCount.Load(), completeCount.Load())
			}
			ready.Store(true)
			record("ready")
			close(readySignal)
		},
	})
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { result <- watcher.Run(ctx) }()
	waitThemeRuntimeSafetySignal(t, readySignal)
	cancel()
	if err := waitThemeRuntimeSafetyResult(t, result); err != nil {
		t.Fatalf("watcher shutdown error=%v", err)
	}
	eventsMu.Lock()
	defer eventsMu.Unlock()
	want := []string{"apply-40", "complete-40", "apply-41", "complete-41", "ready"}
	if strings.Join(events, ",") != strings.Join(want, ",") {
		t.Fatalf("convergence order=%q want=%q", events, want)
	}
}

func TestThemeRuntimeWatcherGenesisFailurePreventsRegistrationAndReady(t *testing.T) {
	for _, test := range []struct {
		name          string
		ensure        func(context.Context) (ThemeRuntimePublication, error)
		expectedError error
	}{
		{
			name: "missing publication",
			ensure: func(context.Context) (ThemeRuntimePublication, error) {
				return ThemeRuntimePublication{}, ErrThemePublicationNotFound
			},
			expectedError: ErrThemePublicationNotFound,
		},
		{
			name: "invalid publication",
			ensure: func(context.Context) (ThemeRuntimePublication, error) {
				return ThemeRuntimePublication{Revision: 1}, nil
			},
			expectedError: ErrThemePublicationConflict,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			identity := ThemeRuntimeNodeIdentity{NodeID: "api-genesis", BootID: "boot-genesis"}
			repository := newThemeRuntimeSafetyRepository(identity, themeRuntimeSafetyPublication(1, "e"))
			repository.ensureFn = test.ensure
			var ready atomic.Bool
			var applied atomic.Int32
			watcher := newThemeRuntimeSafetyWatcher(t, repository, themeRuntimeSafetyApplier(func(
				context.Context,
				ThemeRuntimePublication,
			) error {
				applied.Add(1)
				return nil
			}), ThemeRuntimeWatcherConfig{
				Identity: identity, LeaseDuration: time.Second,
				HeartbeatInterval: 100 * time.Millisecond, PollInterval: time.Hour,
				OnReady: func() { ready.Store(true) },
			})
			if err := watcher.Run(context.Background()); !errors.Is(err, test.expectedError) {
				t.Fatalf("genesis error=%v, want %v", err, test.expectedError)
			}
			if repository.ensureCalls.Load() != 1 || repository.registerCalls.Load() != 0 ||
				applied.Load() != 0 || ready.Load() {
				t.Fatalf(
					"ensure=%d register=%d apply=%d ready=%t",
					repository.ensureCalls.Load(), repository.registerCalls.Load(), applied.Load(), ready.Load(),
				)
			}
		})
	}
}

func TestServiceFailClosedThemeRuntimeWaitsForInFlightActivationAndPermanentlyClosesAdmission(t *testing.T) {
	defaultTheme := withInstalledPackage(t, protectedBuiltinExtension(DefaultThemeID, TypeTheme))
	target := exactThemeRuntimeExtensionFixture(t, "fail-closed.target", "/target")
	target.Status = StatusInstalled
	store := newFakeExtensionStore(map[string]Extension{
		defaultTheme.ID: defaultTheme,
		target.ID:       target,
	})
	store.activeThemeID = defaultTheme.ID
	registry := newBlockingThemeRuntimeSafetyRegistry()
	service := NewServiceWithOptions(
		store, t.TempDir(), "", LocalRuntimeManager{}, WithPageRegistry(registry),
	)
	activationInput := ThemeActivationInput{
		Version: target.Version, PackageDigest: target.PackageDigest,
		CurrentThemeID: defaultTheme.ID, CurrentThemeVersion: defaultTheme.Version,
		CurrentThemeDigest: defaultTheme.PackageDigest,
	}
	activationResult := make(chan error, 1)
	go func() {
		_, err := service.ActivateThemeFromPreview(
			context.Background(), extensionManager(), target.ID, activationInput,
		)
		activationResult <- err
	}()
	waitThemeRuntimeSafetySignal(t, registry.activationStarted)

	failClosedStarted := make(chan struct{})
	failClosedResult := make(chan error, 1)
	go func() {
		close(failClosedStarted)
		failClosedResult <- service.FailClosedThemeRuntime(context.Background())
	}()
	waitThemeRuntimeSafetySignal(t, failClosedStarted)
	close(registry.releaseActivation)
	if err := waitThemeRuntimeSafetyResult(t, activationResult); err != nil {
		t.Fatalf("in-flight activation error=%v", err)
	}
	if err := waitThemeRuntimeSafetyResult(t, failClosedResult); err != nil {
		t.Fatalf("fail-closed fallback error=%v", err)
	}
	if active := registry.activeTheme(); active != DefaultThemeID {
		t.Fatalf("fail-closed runtime active=%q, want protected default", active)
	}

	if _, err := service.ActivateThemeFromPreview(
		context.Background(), extensionManager(), target.ID, activationInput,
	); !errors.Is(err, ErrThemeRuntimeUnavailable) {
		t.Fatalf("activation admission error=%v", err)
	}
	publication := themeRuntimeSafetyPublication(99, "f")
	publication.ThemeID = target.ID
	publication.ThemeVersion = target.Version
	publication.PackageDigest = target.PackageDigest
	if err := service.ApplyThemeRuntimePublication(context.Background(), publication); !errors.Is(err, ErrThemeRuntimeUnavailable) {
		t.Fatalf("watcher apply admission error=%v", err)
	}
	if active := registry.activeTheme(); active != DefaultThemeID {
		t.Fatalf("blocked mutations changed fail-closed runtime to %q", active)
	}
}

type blockingThemeRuntimeSafetyRegistry struct {
	mu sync.Mutex

	active            string
	activationStarted chan struct{}
	releaseActivation chan struct{}
	activationOnce    sync.Once
}

func newBlockingThemeRuntimeSafetyRegistry() *blockingThemeRuntimeSafetyRegistry {
	return &blockingThemeRuntimeSafetyRegistry{
		activationStarted: make(chan struct{}),
		releaseActivation: make(chan struct{}),
	}
}

func (*blockingThemeRuntimeSafetyRegistry) PreflightThemePackage(
	context.Context,
	Extension,
	string,
) error {
	return nil
}

func (r *blockingThemeRuntimeSafetyRegistry) RegisterThemePackage(
	_ context.Context,
	extension Extension,
) error {
	r.mu.Lock()
	r.active = extension.ID
	r.mu.Unlock()
	return nil
}

func (r *blockingThemeRuntimeSafetyRegistry) RegisterThemePackageRestoring(
	ctx context.Context,
	extension Extension,
	_ []string,
) error {
	return r.RegisterThemePackage(ctx, extension)
}

func (*blockingThemeRuntimeSafetyRegistry) RegisterDefaultThemeFallback(
	context.Context,
	Extension,
) error {
	return nil
}

func (r *blockingThemeRuntimeSafetyRegistry) RegisterThemePackageReplacing(
	_ context.Context,
	extension Extension,
	_ string,
) error {
	r.activationOnce.Do(func() { close(r.activationStarted) })
	<-r.releaseActivation
	r.mu.Lock()
	r.active = extension.ID
	r.mu.Unlock()
	return nil
}

func (r *blockingThemeRuntimeSafetyRegistry) RegisterThemePackageReplacingApproved(
	ctx context.Context,
	extension Extension,
	previous string,
	_ int64,
) error {
	return r.RegisterThemePackageReplacing(ctx, extension, previous)
}

func (*blockingThemeRuntimeSafetyRegistry) RegisterPluginPackage(context.Context, Extension) error {
	return nil
}

func (r *blockingThemeRuntimeSafetyRegistry) ClearExtension(extensionID string) {
	r.mu.Lock()
	if r.active == extensionID {
		r.active = ""
	}
	r.mu.Unlock()
}

func (r *blockingThemeRuntimeSafetyRegistry) activeTheme() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.active
}

func newThemeRuntimeSafetyWatcher(
	t *testing.T,
	repository themeRuntimeWatcherRepository,
	applier ThemeRuntimePublicationApplier,
	config ThemeRuntimeWatcherConfig,
) *ThemeRuntimeWatcher {
	t.Helper()
	config.StopTimeout = time.Second
	watcher, err := NewThemeRuntimeWatcher(repository, applier, nil, config)
	if err != nil {
		t.Fatal(err)
	}
	return watcher
}

func themeRuntimeSafetyPublication(revision int64, digestRune string) ThemeRuntimePublication {
	return ThemeRuntimePublication{
		Revision: revision, DesiredState: ThemeRuntimePublicationActive,
		ThemeID: DefaultThemeID, ThemeVersion: "1.0.0", PackageDigest: strings.Repeat(digestRune, 64),
		Reason: ThemeRuntimePublicationStartupRepair,
	}
}

func completedThemeRuntimeSafetyAck(
	identity ThemeRuntimeNodeIdentity,
	publication ThemeRuntimePublication,
	expectedAckRevision int64,
	completedRevision int64,
) ThemeRuntimePublicationAck {
	return ThemeRuntimePublicationAck{
		PublicationRevision:      publication.Revision,
		ThemeRuntimeNodeIdentity: identity,
		Status:                   ThemeRuntimeAckApplied, AttemptCount: 1,
		Revision:     max(expectedAckRevision+1, completedRevision),
		AppliedState: publication.DesiredState, AppliedThemeID: publication.ThemeID,
		AppliedThemeVersion: publication.ThemeVersion, AppliedPackageDigest: publication.PackageDigest,
	}
}

func waitThemeRuntimeSafetySignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for theme runtime safety signal")
	}
}

func waitThemeRuntimeSafetyResult(t *testing.T, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for theme runtime safety result")
		return nil
	}
}
