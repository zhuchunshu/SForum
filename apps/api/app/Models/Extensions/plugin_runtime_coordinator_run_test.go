package extensions

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPluginRuntimeCoordinatorStartupPollConvergesAPIAndWorker(t *testing.T) {
	for _, role := range []PluginRuntimeProcessRole{PluginRuntimeProcessAPI, PluginRuntimeProcessWorker} {
		t.Run(string(role), func(t *testing.T) {
			identity := uniquePluginRuntimeCoordinatorTestIdentity("startup-"+string(role), role)
			repository := newPluginRuntimeCoordinatorTestRepository()
			publication := pluginRuntimeCoordinatorPublicationFixture(1, []PluginRuntimeMember{
				pluginRuntimeCoordinatorMemberFixture("startup.plugin", 1, "a"),
			})
			repository.addPublication(publication)
			applier := &pluginRuntimeCoordinatorTestApplier{}
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
			if err := waitPluginRuntimeCoordinatorRevision(waitCtx, repository.appliedSignal, 1); err != nil {
				t.Fatal(err)
			}
			select {
			case <-ready:
			case <-waitCtx.Done():
				t.Fatal(waitCtx.Err())
			}
			stop()
			if err := assertPluginRuntimeCoordinatorRunStopped(waitCtx, result); err != nil {
				t.Fatalf("Run returned %v", err)
			}

			snapshot := repository.snapshot()
			if snapshot.registerCalls != 1 || snapshot.node.ProcessRole != role ||
				snapshot.node.LastAppliedRevision != 1 || snapshot.beginCalls != 1 ||
				snapshot.completeCalls != 1 || snapshot.failCalls != 0 || readyCount.Load() != 1 {
				t.Fatalf("repository=%#v ready=%d", snapshot, readyCount.Load())
			}
			want := pluginRuntimeCoordinatorAppliedFixture(publication)
			if !reflect.DeepEqual(snapshot.applied[1], want) {
				t.Fatalf("applied=%#v want=%#v", snapshot.applied[1], want)
			}
			calls, maxActive := applier.snapshot()
			if !reflect.DeepEqual(calls, []int64{1}) || maxActive != 1 {
				t.Fatalf("calls=%v maxActive=%d", calls, maxActive)
			}
		})
	}
}

func TestPluginRuntimeCoordinatorSupersededApplyContinuesToLatest(t *testing.T) {
	identity := uniquePluginRuntimeCoordinatorTestIdentity("superseded", PluginRuntimeProcessAPI)
	repository := newPluginRuntimeCoordinatorTestRepository()
	first := pluginRuntimeCoordinatorPublicationFixture(1, []PluginRuntimeMember{
		pluginRuntimeCoordinatorMemberFixture("superseded.plugin", 1, "a"),
	})
	latest := pluginRuntimeCoordinatorPublicationFixture(2, nil)
	repository.addPublication(first)
	applier := &pluginRuntimeCoordinatorTestApplier{apply: func(
		_ context.Context,
		publication PluginRuntimePublication,
		call int,
	) ([]PluginRuntimeAppliedMember, error) {
		if call == 1 {
			repository.addPublication(latest)
			return nil, ErrPluginRuntimePublicationSuperseded
		}
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
	if err := waitPluginRuntimeCoordinatorRevision(waitCtx, repository.appliedSignal, latest.Revision); err != nil {
		t.Fatal(err)
	}
	stop()
	if err := assertPluginRuntimeCoordinatorRunStopped(waitCtx, result); err != nil {
		t.Fatal(err)
	}
	snapshot := repository.snapshot()
	if snapshot.node.LastAppliedRevision != latest.Revision || snapshot.failCalls != 1 ||
		snapshot.completeCalls != 1 || snapshot.beginCalls != 2 {
		t.Fatalf("repository=%#v", snapshot)
	}
	calls, maxActive := applier.snapshot()
	if !reflect.DeepEqual(calls, []int64{first.Revision, latest.Revision}) || maxActive != 1 {
		t.Fatalf("calls=%v maxActive=%d", calls, maxActive)
	}
}

func TestPluginRuntimeCoordinatorSupersededApplyDoesNotHideFailureAckError(t *testing.T) {
	identity := uniquePluginRuntimeCoordinatorTestIdentity("superseded-fail-ack", PluginRuntimeProcessAPI)
	repository := newPluginRuntimeCoordinatorTestRepository()
	first := pluginRuntimeCoordinatorPublicationFixture(1, []PluginRuntimeMember{
		pluginRuntimeCoordinatorMemberFixture("superseded-fail.plugin", 1, "a"),
	})
	latest := pluginRuntimeCoordinatorPublicationFixture(2, nil)
	repository.addPublication(first)
	failErr := errors.New("persist failed acknowledgement")
	repository.failError = failErr
	applier := &pluginRuntimeCoordinatorTestApplier{apply: func(
		_ context.Context,
		_ PluginRuntimePublication,
		call int,
	) ([]PluginRuntimeAppliedMember, error) {
		if call == 1 {
			repository.addPublication(latest)
			return nil, ErrPluginRuntimePublicationSuperseded
		}
		t.Fatalf("latest publication applied after failure ack error")
		return nil, nil
	}}
	repository.seedNode(identity, 0, time.Minute)
	coordinator, err := NewPluginRuntimeCoordinator(
		repository, applier, nil, pluginRuntimeCoordinatorConfigFixture(identity),
	)
	if err != nil {
		t.Fatal(err)
	}
	err = coordinator.reconcileOnce(t.Context())
	if !errors.Is(err, failErr) || !errors.Is(err, ErrPluginRuntimePublicationSuperseded) {
		t.Fatalf("reconcile error=%v", err)
	}
	snapshot := repository.snapshot()
	if snapshot.failCalls != 1 || snapshot.completeCalls != 0 || snapshot.node.LastAppliedRevision != 0 {
		t.Fatalf("repository=%#v", snapshot)
	}
	calls, _ := applier.snapshot()
	if !reflect.DeepEqual(calls, []int64{first.Revision}) {
		t.Fatalf("applier calls=%v", calls)
	}
}

func TestPluginRuntimeCoordinatorProcessAheadDoesNotRetrySameDurableRevision(t *testing.T) {
	identity := uniquePluginRuntimeCoordinatorTestIdentity("process-ahead", PluginRuntimeProcessAPI)
	repository := newPluginRuntimeCoordinatorTestRepository()
	publication := pluginRuntimeCoordinatorPublicationFixture(1, []PluginRuntimeMember{
		pluginRuntimeCoordinatorMemberFixture("process-ahead.plugin", 1, "a"),
	})
	repository.addPublication(publication)
	unexpectedRetry := errors.New("process-ahead publication retried immediately")
	applier := &pluginRuntimeCoordinatorTestApplier{apply: func(
		_ context.Context,
		_ PluginRuntimePublication,
		call int,
	) ([]PluginRuntimeAppliedMember, error) {
		if call == 1 {
			return nil, ErrPluginRuntimePublicationSuperseded
		}
		return nil, unexpectedRetry
	}}
	repository.seedNode(identity, 0, time.Minute)
	coordinator, err := NewPluginRuntimeCoordinator(
		repository, applier, nil, pluginRuntimeCoordinatorConfigFixture(identity),
	)
	if err != nil {
		t.Fatal(err)
	}
	err = coordinator.reconcileOnce(t.Context())
	if !errors.Is(err, ErrPluginRuntimePublicationSuperseded) || errors.Is(err, unexpectedRetry) {
		t.Fatalf("reconcile error=%v", err)
	}
	snapshot := repository.snapshot()
	if snapshot.beginCalls != 1 || snapshot.failCalls != 1 || snapshot.completeCalls != 0 ||
		snapshot.node.LastAppliedRevision != 0 {
		t.Fatalf("repository=%#v", snapshot)
	}
	calls, _ := applier.snapshot()
	if !reflect.DeepEqual(calls, []int64{publication.Revision}) {
		t.Fatalf("applier calls=%v", calls)
	}
}

func TestPluginRuntimeCoordinatorReadyWaitsForSuccessfulCatchUp(t *testing.T) {
	identity := uniquePluginRuntimeCoordinatorTestIdentity("ready-retry", PluginRuntimeProcessAPI)
	repository := newPluginRuntimeCoordinatorTestRepository()
	repository.addPublication(pluginRuntimeCoordinatorPublicationFixture(1, []PluginRuntimeMember{
		pluginRuntimeCoordinatorMemberFixture("ready.plugin", 1, "b"),
	}))
	secondEntered := make(chan struct{})
	releaseSecond := make(chan struct{})
	applier := &pluginRuntimeCoordinatorTestApplier{}
	applier.apply = func(ctx context.Context, publication PluginRuntimePublication, call int) ([]PluginRuntimeAppliedMember, error) {
		if call == 1 {
			return nil, errPluginRuntimeCoordinatorTestApply
		}
		if call == 2 {
			close(secondEntered)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-releaseSecond:
			}
		}
		return pluginRuntimeCoordinatorAppliedFixture(publication), nil
	}
	var readyCount atomic.Int32
	ready := make(chan struct{})
	errorsCh := make(chan error, 8)
	config := pluginRuntimeCoordinatorConfigFixture(identity)
	config.OnReady = func() {
		if readyCount.Add(1) == 1 {
			close(ready)
		}
	}
	config.OnError = func(err error) { errorsCh <- err }
	coordinator, err := NewPluginRuntimeCoordinator(repository, applier, nil, config)
	if err != nil {
		t.Fatal(err)
	}
	runCtx, stop := context.WithCancel(t.Context())
	defer stop()
	result := make(chan error, 1)
	go func() { result <- coordinator.Run(runCtx) }()
	waitCtx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	if err := waitPluginRuntimeCoordinatorRevision(waitCtx, repository.failedSignal, 1); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-errorsCh:
		if !errors.Is(got, errPluginRuntimeCoordinatorTestApply) {
			t.Fatalf("OnError=%v", got)
		}
	case <-waitCtx.Done():
		t.Fatal(waitCtx.Err())
	}
	select {
	case <-secondEntered:
	case <-waitCtx.Done():
		t.Fatal(waitCtx.Err())
	}
	if readyCount.Load() != 0 {
		t.Fatalf("ready before catch-up=%d", readyCount.Load())
	}
	close(releaseSecond)
	select {
	case <-ready:
	case <-waitCtx.Done():
		t.Fatal(waitCtx.Err())
	}
	stop()
	if err := assertPluginRuntimeCoordinatorRunStopped(waitCtx, result); err != nil {
		t.Fatalf("Run returned %v", err)
	}
	ack := repository.snapshot().acks[1]
	if ack.Status != PluginRuntimeAckApplied || ack.AttemptCount != 2 || readyCount.Load() != 1 {
		t.Fatalf("ack=%#v ready=%d", ack, readyCount.Load())
	}
}

func TestPluginRuntimeCoordinatorNotificationStormCoalesces(t *testing.T) {
	identity := uniquePluginRuntimeCoordinatorTestIdentity("notify", PluginRuntimeProcessAPI)
	repository := newPluginRuntimeCoordinatorTestRepository()
	notifications := newPluginRuntimeCoordinatorTestNotifications()
	entered := make(chan struct{})
	release := make(chan struct{})
	var enteredOnce sync.Once
	applier := &pluginRuntimeCoordinatorTestApplier{apply: func(
		ctx context.Context,
		publication PluginRuntimePublication,
		_ int,
	) ([]PluginRuntimeAppliedMember, error) {
		enteredOnce.Do(func() { close(entered) })
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-release:
			return pluginRuntimeCoordinatorAppliedFixture(publication), nil
		}
	}}
	config := pluginRuntimeCoordinatorConfigFixture(identity)
	config.PollInterval = time.Hour
	coordinator, err := NewPluginRuntimeCoordinator(repository, applier, notifications, config)
	if err != nil {
		t.Fatal(err)
	}
	runCtx, stop := context.WithCancel(t.Context())
	result := make(chan error, 1)
	go func() { result <- coordinator.Run(runCtx) }()
	waitCtx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	if err := waitPluginRuntimeCoordinatorSignal(waitCtx, repository.latestSignal); err != nil {
		t.Fatal(err)
	}
	select {
	case <-notifications.ready:
	case <-waitCtx.Done():
		t.Fatal(waitCtx.Err())
	}
	repository.addPublication(pluginRuntimeCoordinatorPublicationFixture(1, []PluginRuntimeMember{
		pluginRuntimeCoordinatorMemberFixture("notify.plugin", 1, "c"),
	}))
	notifications.notify(256)
	select {
	case <-entered:
	case <-waitCtx.Done():
		t.Fatal(waitCtx.Err())
	}
	notifications.notify(256)
	close(release)
	if err := waitPluginRuntimeCoordinatorRevision(waitCtx, repository.appliedSignal, 1); err != nil {
		t.Fatal(err)
	}
	stop()
	if err := assertPluginRuntimeCoordinatorRunStopped(waitCtx, result); err != nil {
		t.Fatalf("Run returned %v", err)
	}
	select {
	case <-notifications.done:
	case <-waitCtx.Done():
		t.Fatal("notification watcher did not stop")
	}
	calls, maxActive := applier.snapshot()
	if !reflect.DeepEqual(calls, []int64{1}) || maxActive != 1 {
		t.Fatalf("calls=%v maxActive=%d", calls, maxActive)
	}
}

func TestPluginRuntimeCoordinatorPeriodicPollRecoversMissedWake(t *testing.T) {
	identity := uniquePluginRuntimeCoordinatorTestIdentity("poll", PluginRuntimeProcessWorker)
	repository := newPluginRuntimeCoordinatorTestRepository()
	applier := &pluginRuntimeCoordinatorTestApplier{}
	coordinator, err := NewPluginRuntimeCoordinator(
		repository, applier, nil, pluginRuntimeCoordinatorConfigFixture(identity),
	)
	if err != nil {
		t.Fatal(err)
	}
	runCtx, stop := context.WithCancel(t.Context())
	result := make(chan error, 1)
	go func() { result <- coordinator.Run(runCtx) }()
	waitCtx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	if err := waitPluginRuntimeCoordinatorSignal(waitCtx, repository.latestSignal); err != nil {
		t.Fatal(err)
	}
	repository.addPublication(pluginRuntimeCoordinatorPublicationFixture(1, []PluginRuntimeMember{
		pluginRuntimeCoordinatorMemberFixture("poll.plugin", 1, "d"),
	}))
	if err := waitPluginRuntimeCoordinatorRevision(waitCtx, repository.appliedSignal, 1); err != nil {
		t.Fatal(err)
	}
	stop()
	if err := assertPluginRuntimeCoordinatorRunStopped(waitCtx, result); err != nil {
		t.Fatalf("Run returned %v", err)
	}
	if snapshot := repository.snapshot(); snapshot.latestCalls < 2 || snapshot.completeCalls != 1 {
		t.Fatalf("snapshot=%#v", snapshot)
	}
}

func TestPluginRuntimeCoordinatorAppliesNewLatestWithoutWaitingForPoll(t *testing.T) {
	identity := uniquePluginRuntimeCoordinatorTestIdentity("new-latest", PluginRuntimeProcessAPI)
	repository := newPluginRuntimeCoordinatorTestRepository()
	first := pluginRuntimeCoordinatorPublicationFixture(1, []PluginRuntimeMember{
		pluginRuntimeCoordinatorMemberFixture("latest.plugin", 1, "e"),
	})
	second := pluginRuntimeCoordinatorPublicationFixture(2, []PluginRuntimeMember{
		pluginRuntimeCoordinatorMemberFixture("latest.plugin", 2, "f"),
	})
	repository.addPublication(first)
	applier := &pluginRuntimeCoordinatorTestApplier{}
	applier.apply = func(ctx context.Context, publication PluginRuntimePublication, call int) ([]PluginRuntimeAppliedMember, error) {
		if call == 1 {
			repository.addPublication(second)
		}
		return pluginRuntimeCoordinatorAppliedFixture(publication), nil
	}
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
	if err := waitPluginRuntimeCoordinatorRevision(waitCtx, repository.appliedSignal, 2); err != nil {
		t.Fatal(err)
	}
	stop()
	if err := assertPluginRuntimeCoordinatorRunStopped(waitCtx, result); err != nil {
		t.Fatalf("Run returned %v", err)
	}
	calls, maxActive := applier.snapshot()
	if !reflect.DeepEqual(calls, []int64{1, 2}) || maxActive != 1 {
		t.Fatalf("calls=%v maxActive=%d", calls, maxActive)
	}
}

func TestPluginRuntimeCoordinatorAcknowledgesNewRevisionWithSameFullSet(t *testing.T) {
	identity := uniquePluginRuntimeCoordinatorTestIdentity("same-set", PluginRuntimeProcessWorker)
	repository := newPluginRuntimeCoordinatorTestRepository()
	members := []PluginRuntimeMember{pluginRuntimeCoordinatorMemberFixture("same.plugin", 1, "0")}
	first := pluginRuntimeCoordinatorPublicationFixture(1, members)
	second := pluginRuntimeCoordinatorPublicationFixture(2, members)
	repository.addPublication(first)
	applier := &pluginRuntimeCoordinatorTestApplier{}
	applier.apply = func(_ context.Context, publication PluginRuntimePublication, call int) ([]PluginRuntimeAppliedMember, error) {
		if call == 1 {
			repository.addPublication(second)
		}
		return []PluginRuntimeAppliedMember{{
			PluginRuntimeMember: publication.Members[0], RuntimeInstanceID: "stable-runtime-instance",
		}}, nil
	}
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
	if err := waitPluginRuntimeCoordinatorRevision(waitCtx, repository.appliedSignal, 2); err != nil {
		t.Fatal(err)
	}
	stop()
	if err := assertPluginRuntimeCoordinatorRunStopped(waitCtx, result); err != nil {
		t.Fatal(err)
	}
	snapshot := repository.snapshot()
	if snapshot.node.LastAppliedRevision != 2 || snapshot.acks[1].Status != PluginRuntimeAckApplied ||
		snapshot.acks[2].Status != PluginRuntimeAckApplied ||
		snapshot.applied[1][0].RuntimeInstanceID != snapshot.applied[2][0].RuntimeInstanceID {
		t.Fatalf("snapshot=%#v", snapshot)
	}
	calls, maxActive := applier.snapshot()
	if !reflect.DeepEqual(calls, []int64{1, 2}) || maxActive != 1 {
		t.Fatalf("calls=%v maxActive=%d", calls, maxActive)
	}
}

func TestPluginRuntimeCoordinatorEmptyPublicationIsAppliedAndAcknowledged(t *testing.T) {
	identity := uniquePluginRuntimeCoordinatorTestIdentity("safe-mode", PluginRuntimeProcessAPI)
	repository := newPluginRuntimeCoordinatorTestRepository()
	publication := pluginRuntimeCoordinatorPublicationFixture(1, nil)
	repository.addPublication(publication)
	applier := &pluginRuntimeCoordinatorTestApplier{}
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
	snapshot := repository.snapshot()
	ack := snapshot.acks[1]
	if ack.AppliedMemberCount == nil || *ack.AppliedMemberCount != 0 ||
		ack.AppliedMembersDigest != publication.MembersDigest || len(snapshot.applied[1]) != 0 {
		t.Fatalf("ack=%#v applied=%#v", ack, snapshot.applied[1])
	}
	calls, _ := applier.snapshot()
	if !reflect.DeepEqual(calls, []int64{1}) {
		t.Fatalf("empty full set was not applied: %v", calls)
	}
}
