package extensions

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"
)

func TestPluginRuntimeCoordinatorDoesNotReadyBeforeDesiredPublicationExists(t *testing.T) {
	identity := uniquePluginRuntimeCoordinatorTestIdentity("unseeded", PluginRuntimeProcessAPI)
	repository := newPluginRuntimeCoordinatorTestRepository()
	var readyCount atomic.Int32
	config := pluginRuntimeCoordinatorConfigFixture(identity)
	config.OnReady = func() { readyCount.Add(1) }
	coordinator, err := NewPluginRuntimeCoordinator(repository, &pluginRuntimeCoordinatorTestApplier{}, nil, config)
	if err != nil {
		t.Fatal(err)
	}
	runCtx, stop := context.WithCancel(t.Context())
	result := make(chan error, 1)
	go func() { result <- coordinator.Run(runCtx) }()
	waitCtx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	for range 3 {
		if err := waitPluginRuntimeCoordinatorSignal(waitCtx, repository.latestSignal); err != nil {
			t.Fatal(err)
		}
	}
	if readyCount.Load() != 0 {
		t.Fatalf("unseeded coordinator became ready %d times", readyCount.Load())
	}
	stop()
	if err := assertPluginRuntimeCoordinatorRunStopped(waitCtx, result); err != nil {
		t.Fatalf("Run returned %v", err)
	}
}

func TestPluginRuntimeCoordinatorHeartbeatsDuringLongInitialApply(t *testing.T) {
	identity := uniquePluginRuntimeCoordinatorTestIdentity("long-apply", PluginRuntimeProcessWorker)
	repository := newPluginRuntimeCoordinatorTestRepository()
	repository.addPublication(pluginRuntimeCoordinatorPublicationFixture(1, []PluginRuntimeMember{
		pluginRuntimeCoordinatorMemberFixture("long.plugin", 1, "1"),
	}))
	entered := make(chan struct{})
	release := make(chan struct{})
	applier := &pluginRuntimeCoordinatorTestApplier{apply: func(
		ctx context.Context,
		publication PluginRuntimePublication,
		_ int,
	) ([]PluginRuntimeAppliedMember, error) {
		close(entered)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-release:
			return pluginRuntimeCoordinatorAppliedFixture(publication), nil
		}
	}}
	ready := make(chan struct{})
	var readyCount atomic.Int32
	config := pluginRuntimeCoordinatorConfigFixture(identity)
	config.PollInterval = time.Hour
	config.OnReady = func() {
		if readyCount.Add(1) == 1 {
			close(ready)
		}
	}
	coordinator, err := NewPluginRuntimeCoordinator(repository, applier, nil, config)
	if err != nil {
		t.Fatal(err)
	}
	runCtx, stop := context.WithCancel(t.Context())
	result := make(chan error, 1)
	go func() { result <- coordinator.Run(runCtx) }()
	waitCtx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	select {
	case <-entered:
	case <-waitCtx.Done():
		t.Fatal(waitCtx.Err())
	}
	for range 3 {
		if err := waitPluginRuntimeCoordinatorSignal(waitCtx, repository.heartbeatSignal); err != nil {
			t.Fatal(err)
		}
	}
	if readyCount.Load() != 0 || repository.snapshot().completeCalls != 0 {
		t.Fatal("coordinator readied or completed while the full-set apply was blocked")
	}
	close(release)
	select {
	case <-ready:
	case <-waitCtx.Done():
		t.Fatal(waitCtx.Err())
	}
	stop()
	if err := assertPluginRuntimeCoordinatorRunStopped(waitCtx, result); err != nil {
		t.Fatalf("Run returned %v", err)
	}
	if snapshot := repository.snapshot(); snapshot.heartbeatCalls < 3 || snapshot.completeCalls != 1 {
		t.Fatalf("snapshot=%#v", snapshot)
	}
}

func TestPluginRuntimeCoordinatorCancellationNeverWritesFalseEvidence(t *testing.T) {
	identity := uniquePluginRuntimeCoordinatorTestIdentity("cancel", PluginRuntimeProcessAPI)
	repository := newPluginRuntimeCoordinatorTestRepository()
	publication := pluginRuntimeCoordinatorPublicationFixture(1, []PluginRuntimeMember{
		pluginRuntimeCoordinatorMemberFixture("cancel.plugin", 1, "2"),
	})
	repository.addPublication(publication)
	entered := make(chan struct{})
	applierReturned := make(chan struct{})
	applier := &pluginRuntimeCoordinatorTestApplier{apply: func(
		ctx context.Context,
		_ PluginRuntimePublication,
		_ int,
	) ([]PluginRuntimeAppliedMember, error) {
		close(entered)
		<-ctx.Done()
		close(applierReturned)
		// Even a misbehaving applier that reports success after cancellation
		// must not cause Complete or a synthetic failed acknowledgement.
		return pluginRuntimeCoordinatorAppliedFixture(publication), nil
	}}
	config := pluginRuntimeCoordinatorConfigFixture(identity)
	config.PollInterval = time.Hour
	coordinator, err := NewPluginRuntimeCoordinator(repository, applier, nil, config)
	if err != nil {
		t.Fatal(err)
	}
	runCtx, stop := context.WithCancel(t.Context())
	result := make(chan error, 1)
	go func() { result <- coordinator.Run(runCtx) }()
	waitCtx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	select {
	case <-entered:
	case <-waitCtx.Done():
		t.Fatal(waitCtx.Err())
	}
	stop()
	select {
	case <-applierReturned:
	case <-waitCtx.Done():
		t.Fatal("applier context was not cancelled")
	}
	if err := assertPluginRuntimeCoordinatorRunStopped(waitCtx, result); err != nil {
		t.Fatalf("Run returned %v", err)
	}
	snapshot := repository.snapshot()
	if snapshot.completeCalls != 0 || snapshot.failCalls != 0 || snapshot.acks[1].Status != PluginRuntimeAckApplying {
		t.Fatalf("snapshot=%#v", snapshot)
	}
}

func TestPluginRuntimeCoordinatorCancellationDuringHeartbeatRetiresBootCleanly(t *testing.T) {
	identity := uniquePluginRuntimeCoordinatorTestIdentity("cancel-heartbeat", PluginRuntimeProcessAPI)
	firstRepository := &pluginRuntimeCoordinatorCancellationHeartbeatRepository{
		pluginRuntimeCoordinatorTestRepository: newPluginRuntimeCoordinatorTestRepository(),
		entered:                                make(chan struct{}),
	}
	config := pluginRuntimeCoordinatorConfigFixture(identity)
	config.HeartbeatInterval = 100 * time.Millisecond
	config.PollInterval = time.Hour
	first, err := NewPluginRuntimeCoordinator(
		firstRepository, &pluginRuntimeCoordinatorTestApplier{}, nil, config,
	)
	if err != nil {
		t.Fatal(err)
	}
	runCtx, stop := context.WithCancel(t.Context())
	result := make(chan error, 1)
	go func() { result <- first.Run(runCtx) }()
	waitCtx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	select {
	case <-firstRepository.entered:
	case <-waitCtx.Done():
		t.Fatal("heartbeat did not start")
	}
	stop()
	if err := assertPluginRuntimeCoordinatorRunStopped(waitCtx, result); err != nil {
		t.Fatalf("cancelled Run returned %v", err)
	}

	// The registered boot is one-shot even after graceful shutdown. Reusing its
	// durable last_applied_revision with a replacement Manager could otherwise
	// claim readiness for runtime instance ids that no longer exist locally.
	second, err := NewPluginRuntimeCoordinator(
		firstRepository, &pluginRuntimeCoordinatorTestApplier{}, nil, config,
	)
	if err != nil {
		t.Fatal(err)
	}
	if retryErr := second.Run(t.Context()); !errors.Is(retryErr, ErrPluginRuntimeCoordinatorRetired) {
		t.Fatalf("same boot identity retry error=%v", retryErr)
	}
	if snapshot := firstRepository.snapshot(); snapshot.registerCalls != 1 {
		t.Fatalf("retired boot registered again: %#v", snapshot)
	}
}

func TestPluginRuntimeCoordinatorHeartbeatDeadlineCancelsApplyAndRetiresBoot(t *testing.T) {
	identity := uniquePluginRuntimeCoordinatorTestIdentity(
		"heartbeat-deadline", PluginRuntimeProcessWorker,
	)
	repository := &pluginRuntimeCoordinatorCancellationHeartbeatRepository{
		pluginRuntimeCoordinatorTestRepository: newPluginRuntimeCoordinatorTestRepository(),
		entered:                                make(chan struct{}),
	}
	publication := pluginRuntimeCoordinatorPublicationFixture(1, []PluginRuntimeMember{
		pluginRuntimeCoordinatorMemberFixture("heartbeat-deadline.plugin", 1, "7"),
	})
	repository.addPublication(publication)
	applierCancelled := make(chan struct{})
	applier := &pluginRuntimeCoordinatorTestApplier{apply: func(
		ctx context.Context,
		_ PluginRuntimePublication,
		_ int,
	) ([]PluginRuntimeAppliedMember, error) {
		<-ctx.Done()
		close(applierCancelled)
		return nil, ctx.Err()
	}}
	config := pluginRuntimeCoordinatorConfigFixture(identity)
	config.HeartbeatInterval = 10 * time.Millisecond
	config.PollInterval = time.Hour
	coordinator, err := NewPluginRuntimeCoordinator(repository, applier, nil, config)
	if err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() { result <- coordinator.Run(t.Context()) }()
	waitCtx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	runErr := assertPluginRuntimeCoordinatorRunStopped(waitCtx, result)
	if !errors.Is(runErr, context.DeadlineExceeded) {
		t.Fatalf("Run error=%v, want heartbeat deadline", runErr)
	}
	select {
	case <-applierCancelled:
	default:
		t.Fatal("heartbeat deadline did not cancel the in-flight apply")
	}
	snapshot := repository.snapshot()
	if snapshot.completeCalls != 0 || snapshot.failCalls != 0 || snapshot.acks[1].Status != PluginRuntimeAckApplying {
		t.Fatalf("snapshot=%#v", snapshot)
	}
	if retryErr := coordinator.Run(t.Context()); !errors.Is(retryErr, context.DeadlineExceeded) {
		t.Fatalf("terminal boot retry error=%v", retryErr)
	}
}

type pluginRuntimeCoordinatorCancellationHeartbeatRepository struct {
	*pluginRuntimeCoordinatorTestRepository
	entered chan struct{}
}

func (r *pluginRuntimeCoordinatorCancellationHeartbeatRepository) HeartbeatPluginRuntimeNode(
	ctx context.Context,
	_ PluginRuntimeNodeIdentity,
	_ time.Duration,
) (PluginRuntimeNode, error) {
	close(r.entered)
	<-ctx.Done()
	return PluginRuntimeNode{}, ctx.Err()
}

func TestPluginRuntimeCoordinatorHeartbeatFailureCancelsApplyAndIsTerminal(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
	}{
		{name: "lease lost", err: ErrPluginRuntimeNodeLeaseLost},
		{name: "database unavailable", err: errors.New("heartbeat database unavailable")},
	} {
		t.Run(test.name, func(t *testing.T) {
			identity := uniquePluginRuntimeCoordinatorTestIdentity(
				"heartbeat-"+test.name, PluginRuntimeProcessWorker,
			)
			repository := newPluginRuntimeCoordinatorTestRepository()
			repository.heartbeatErrorAfter = 1
			repository.heartbeatError = test.err
			repository.addPublication(pluginRuntimeCoordinatorPublicationFixture(1, []PluginRuntimeMember{
				pluginRuntimeCoordinatorMemberFixture("heartbeat.plugin", 1, "3"),
			}))
			entered := make(chan struct{})
			cancelled := make(chan struct{})
			applier := &pluginRuntimeCoordinatorTestApplier{apply: func(
				ctx context.Context,
				_ PluginRuntimePublication,
				_ int,
			) ([]PluginRuntimeAppliedMember, error) {
				close(entered)
				<-ctx.Done()
				close(cancelled)
				return nil, ctx.Err()
			}}
			config := pluginRuntimeCoordinatorConfigFixture(identity)
			config.PollInterval = time.Hour
			coordinator, err := NewPluginRuntimeCoordinator(repository, applier, nil, config)
			if err != nil {
				t.Fatal(err)
			}
			result := make(chan error, 1)
			go func() { result <- coordinator.Run(t.Context()) }()
			waitCtx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
			defer cancel()
			select {
			case <-entered:
			case <-waitCtx.Done():
				t.Fatal(waitCtx.Err())
			}
			runErr := assertPluginRuntimeCoordinatorRunStopped(waitCtx, result)
			if !errors.Is(runErr, test.err) {
				t.Fatalf("Run error=%v want=%v", runErr, test.err)
			}
			select {
			case <-cancelled:
			default:
				t.Fatal("heartbeat failure did not cancel applier")
			}
			snapshot := repository.snapshot()
			if snapshot.completeCalls != 0 || snapshot.failCalls != 0 || snapshot.acks[1].Status != PluginRuntimeAckApplying {
				t.Fatalf("snapshot=%#v", snapshot)
			}
			if retryErr := coordinator.Run(t.Context()); !errors.Is(retryErr, test.err) {
				t.Fatalf("terminal boot retry error=%v want=%v", retryErr, test.err)
			}
			if after := repository.snapshot(); after.registerCalls != snapshot.registerCalls {
				t.Fatalf("terminal boot registered again: before=%d after=%d", snapshot.registerCalls, after.registerCalls)
			}
		})
	}
}

func TestPluginRuntimeCoordinatorBootIdentityHasOneRunOwner(t *testing.T) {
	identity := uniquePluginRuntimeCoordinatorTestIdentity("one-owner", PluginRuntimeProcessAPI)
	repository := newPluginRuntimeCoordinatorTestRepository()
	repository.addPublication(pluginRuntimeCoordinatorPublicationFixture(1, []PluginRuntimeMember{
		pluginRuntimeCoordinatorMemberFixture("owner.plugin", 1, "4"),
	}))
	entered := make(chan struct{})
	applier := &pluginRuntimeCoordinatorTestApplier{apply: func(
		ctx context.Context,
		_ PluginRuntimePublication,
		_ int,
	) ([]PluginRuntimeAppliedMember, error) {
		select {
		case <-entered:
		default:
			close(entered)
		}
		<-ctx.Done()
		return nil, ctx.Err()
	}}
	config := pluginRuntimeCoordinatorConfigFixture(identity)
	config.PollInterval = time.Hour
	first, err := NewPluginRuntimeCoordinator(repository, applier, nil, config)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewPluginRuntimeCoordinator(repository, applier, nil, config)
	if err != nil {
		t.Fatal(err)
	}
	firstCtx, stop := context.WithCancel(t.Context())
	firstResult := make(chan error, 1)
	go func() { firstResult <- first.Run(firstCtx) }()
	waitCtx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	select {
	case <-entered:
	case <-waitCtx.Done():
		t.Fatal(waitCtx.Err())
	}
	if err := second.Run(t.Context()); !errors.Is(err, ErrPluginRuntimeCoordinatorRunning) {
		t.Fatalf("second Run error=%v", err)
	}
	if err := first.Run(t.Context()); !errors.Is(err, ErrPluginRuntimeCoordinatorRunning) {
		t.Fatalf("duplicate Run error=%v", err)
	}
	stop()
	if err := assertPluginRuntimeCoordinatorRunStopped(waitCtx, firstResult); err != nil {
		t.Fatalf("first Run returned %v", err)
	}
	calls, maxActive := applier.snapshot()
	if !reflect.DeepEqual(calls, []int64{1}) || maxActive != 1 {
		t.Fatalf("calls=%v maxActive=%d", calls, maxActive)
	}
}

func TestPluginRuntimeCoordinatorSkipsStaleAndNonNewerPublications(t *testing.T) {
	for _, test := range []struct {
		name        string
		latest      int64
		lastApplied int64
	}{
		{name: "same revision", latest: 5, lastApplied: 5},
		{name: "stale durable read", latest: 4, lastApplied: 5},
	} {
		t.Run(test.name, func(t *testing.T) {
			identity := uniquePluginRuntimeCoordinatorTestIdentity(
				"stale-"+test.name, PluginRuntimeProcessAPI,
			)
			repository := newPluginRuntimeCoordinatorTestRepository()
			repository.addPublication(pluginRuntimeCoordinatorPublicationFixture(test.latest, []PluginRuntimeMember{
				pluginRuntimeCoordinatorMemberFixture("stale.plugin", 1, "5"),
			}))
			repository.seedNode(identity, test.lastApplied, time.Second)
			applier := &pluginRuntimeCoordinatorTestApplier{}
			ready := make(chan struct{})
			config := pluginRuntimeCoordinatorConfigFixture(identity)
			config.PollInterval = time.Hour
			config.OnReady = func() { close(ready) }
			coordinator, err := NewPluginRuntimeCoordinator(repository, applier, nil, config)
			if err != nil {
				t.Fatal(err)
			}
			runCtx, stop := context.WithCancel(t.Context())
			result := make(chan error, 1)
			go func() { result <- coordinator.Run(runCtx) }()
			waitCtx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
			defer cancel()
			select {
			case <-ready:
			case <-waitCtx.Done():
				t.Fatal(waitCtx.Err())
			}
			stop()
			if err := assertPluginRuntimeCoordinatorRunStopped(waitCtx, result); err != nil {
				t.Fatal(err)
			}
			calls, _ := applier.snapshot()
			snapshot := repository.snapshot()
			if len(calls) != 0 || snapshot.beginCalls != 0 || snapshot.completeCalls != 0 || snapshot.failCalls != 0 {
				t.Fatalf("calls=%v snapshot=%#v", calls, snapshot)
			}
		})
	}
}

func TestPluginRuntimeCoordinatorRejectsCorruptDesiredBeforeBegin(t *testing.T) {
	identity := uniquePluginRuntimeCoordinatorTestIdentity("corrupt", PluginRuntimeProcessAPI)
	repository := newPluginRuntimeCoordinatorTestRepository()
	publication := pluginRuntimeCoordinatorPublicationFixture(1, []PluginRuntimeMember{
		pluginRuntimeCoordinatorMemberFixture("corrupt.plugin", 1, "6"),
	})
	publication.MemberCount++
	repository.addPublication(publication)
	applier := &pluginRuntimeCoordinatorTestApplier{}
	coordinator, err := NewPluginRuntimeCoordinator(
		repository, applier, nil, pluginRuntimeCoordinatorConfigFixture(identity),
	)
	if err != nil {
		t.Fatal(err)
	}
	runErr := coordinator.Run(t.Context())
	if !errors.Is(runErr, ErrPluginRuntimeCoordinatorInvalid) {
		t.Fatalf("Run error=%v", runErr)
	}
	snapshot := repository.snapshot()
	calls, _ := applier.snapshot()
	if snapshot.beginCalls != 0 || snapshot.completeCalls != 0 || snapshot.failCalls != 0 || len(calls) != 0 {
		t.Fatalf("calls=%v snapshot=%#v", calls, snapshot)
	}
}

func TestPluginRuntimeCoordinatorRejectsInvalidAppliedEvidenceAsFailure(t *testing.T) {
	identity := uniquePluginRuntimeCoordinatorTestIdentity("bad-evidence", PluginRuntimeProcessWorker)
	repository := newPluginRuntimeCoordinatorTestRepository()
	repository.addPublication(pluginRuntimeCoordinatorPublicationFixture(1, []PluginRuntimeMember{
		pluginRuntimeCoordinatorMemberFixture("evidence.plugin", 1, "7"),
	}))
	applier := &pluginRuntimeCoordinatorTestApplier{apply: func(
		context.Context,
		PluginRuntimePublication,
		int,
	) ([]PluginRuntimeAppliedMember, error) {
		return []PluginRuntimeAppliedMember{{
			PluginRuntimeMember: pluginRuntimeCoordinatorMemberFixture("evidence.plugin", 1, "7"),
			RuntimeInstanceID:   "",
		}}, nil
	}}
	config := pluginRuntimeCoordinatorConfigFixture(identity)
	config.PollInterval = time.Hour
	coordinator, err := NewPluginRuntimeCoordinator(repository, applier, nil, config)
	if err != nil {
		t.Fatal(err)
	}
	runCtx, stop := context.WithCancel(t.Context())
	result := make(chan error, 1)
	go func() { result <- coordinator.Run(runCtx) }()
	waitCtx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	if err := waitPluginRuntimeCoordinatorRevision(waitCtx, repository.failedSignal, 1); err != nil {
		t.Fatal(err)
	}
	stop()
	if err := assertPluginRuntimeCoordinatorRunStopped(waitCtx, result); err != nil {
		t.Fatalf("Run returned %v", err)
	}
	snapshot := repository.snapshot()
	if snapshot.failCalls != 1 || snapshot.completeCalls != 0 || snapshot.acks[1].Status != PluginRuntimeAckFailed {
		t.Fatalf("snapshot=%#v", snapshot)
	}
}

func TestPluginRuntimeCoordinatorRecoversAmbiguousComplete(t *testing.T) {
	for _, test := range []struct {
		name          string
		commit        bool
		wantApply     []int64
		wantCompletes int
	}{
		{name: "commit happened", commit: true, wantApply: []int64{1}, wantCompletes: 1},
		{name: "commit did not happen", commit: false, wantApply: []int64{1, 1}, wantCompletes: 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			identity := uniquePluginRuntimeCoordinatorTestIdentity(
				"ambiguous-"+test.name, PluginRuntimeProcessAPI,
			)
			repository := newPluginRuntimeCoordinatorTestRepository()
			repository.completeErrors = 1
			repository.completeCommits = test.commit
			repository.completeError = errors.New("ambiguous completion transport")
			repository.addPublication(pluginRuntimeCoordinatorPublicationFixture(1, []PluginRuntimeMember{
				pluginRuntimeCoordinatorMemberFixture("ambiguous.plugin", 1, "8"),
			}))
			applier := &pluginRuntimeCoordinatorTestApplier{}
			ready := make(chan struct{})
			errorsCh := make(chan error, 8)
			config := pluginRuntimeCoordinatorConfigFixture(identity)
			config.OnReady = func() { close(ready) }
			config.OnError = func(err error) { errorsCh <- err }
			coordinator, err := NewPluginRuntimeCoordinator(repository, applier, nil, config)
			if err != nil {
				t.Fatal(err)
			}
			runCtx, stop := context.WithCancel(t.Context())
			result := make(chan error, 1)
			go func() { result <- coordinator.Run(runCtx) }()
			waitCtx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
			defer cancel()
			select {
			case <-ready:
			case <-waitCtx.Done():
				t.Fatal(waitCtx.Err())
			}
			stop()
			if err := assertPluginRuntimeCoordinatorRunStopped(waitCtx, result); err != nil {
				t.Fatal(err)
			}
			calls, maxActive := applier.snapshot()
			snapshot := repository.snapshot()
			if !reflect.DeepEqual(calls, test.wantApply) || maxActive != 1 ||
				snapshot.completeCalls != test.wantCompletes || snapshot.failCalls != 0 ||
				snapshot.node.LastAppliedRevision != 1 {
				t.Fatalf("calls=%v max=%d snapshot=%#v", calls, maxActive, snapshot)
			}
			if test.commit {
				select {
				case err := <-errorsCh:
					t.Fatalf("commit proof should suppress transient error: %v", err)
				default:
				}
			} else {
				select {
				case err := <-errorsCh:
					if !strings.Contains(err.Error(), "ambiguous completion transport") {
						t.Fatalf("OnError=%v", err)
					}
				default:
					t.Fatal("missing transient completion error")
				}
			}
		})
	}
}

func TestPluginRuntimeCoordinatorPassesApplierAnIsolatedPublicationClone(t *testing.T) {
	identity := uniquePluginRuntimeCoordinatorTestIdentity("clone", PluginRuntimeProcessAPI)
	repository := newPluginRuntimeCoordinatorTestRepository()
	publication := pluginRuntimeCoordinatorPublicationFixture(1, []PluginRuntimeMember{
		pluginRuntimeCoordinatorMemberFixture("clone.plugin", 1, "9"),
	})
	repository.addPublication(publication)
	applier := &pluginRuntimeCoordinatorTestApplier{apply: func(
		_ context.Context,
		got PluginRuntimePublication,
		_ int,
	) ([]PluginRuntimeAppliedMember, error) {
		got.Members[0].ExtensionID = "mutated.plugin"
		return pluginRuntimeCoordinatorAppliedFixture(publication), nil
	}}
	config := pluginRuntimeCoordinatorConfigFixture(identity)
	config.PollInterval = time.Hour
	coordinator, err := NewPluginRuntimeCoordinator(repository, applier, nil, config)
	if err != nil {
		t.Fatal(err)
	}
	runCtx, stop := context.WithCancel(t.Context())
	result := make(chan error, 1)
	go func() { result <- coordinator.Run(runCtx) }()
	waitCtx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	if err := waitPluginRuntimeCoordinatorRevision(waitCtx, repository.appliedSignal, 1); err != nil {
		t.Fatal(err)
	}
	stop()
	if err := assertPluginRuntimeCoordinatorRunStopped(waitCtx, result); err != nil {
		t.Fatal(err)
	}
	if got := repository.snapshot().applied[1]; !reflect.DeepEqual(got, pluginRuntimeCoordinatorAppliedFixture(publication)) {
		t.Fatalf("applied=%#v", got)
	}
}

func TestPluginRuntimeCoordinatorFailureReasonIsBoundedValidUTF8(t *testing.T) {
	reason := pluginRuntimeCoordinatorFailureReason(errors.New(
		strings.Repeat("\u4e3b\u9898\u5931\u8d25", 1000) + string([]byte{0xff}),
	))
	if reason == "" || len([]byte(reason)) > 2048 || !utf8.ValidString(reason) {
		t.Fatalf("reason bytes=%d valid=%t", len([]byte(reason)), utf8.ValidString(reason))
	}
}
