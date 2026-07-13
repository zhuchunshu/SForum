package jobs

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestPluginScheduleAdmissionAllowsActiveExactRuntimeAndWaitsForInflight(t *testing.T) {
	registry := NewPluginScheduleAdmissionRegistry()
	runtime := pluginScheduleRuntimeFixture("instance-a", "1.0.0", "sha256:a")
	published, err := registry.PublishActive(runtime)
	if err != nil || !published.Active || published.Draining || len(published.Schedules) != 1 {
		t.Fatalf("publish = %#v, %v", published, err)
	}

	declaration, lease, err := registry.AcquireTrigger(context.Background(), runtime.Identity, "daily-sync")
	if err != nil || declaration.JobContract != "demo.plugin.job.sync@1" || lease == nil {
		t.Fatalf("acquire = %#v, %#v, %v", declaration, lease, err)
	}
	snapshot, err := registry.BeginDrain(runtime.Identity)
	if err != nil || !snapshot.Draining || snapshot.ActiveTriggers != 1 {
		t.Fatalf("drain = %#v, %v", snapshot, err)
	}
	if _, _, err := registry.AcquireTrigger(context.Background(), runtime.Identity, "daily-sync"); !errors.Is(err, ErrPluginScheduleDraining) {
		t.Fatalf("post-drain acquire = %v", err)
	}

	timeout, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := registry.WaitDrain(timeout, runtime.Identity); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("wait before release = %v", err)
	}
	lease.Release()
	if err := registry.WaitDrain(context.Background(), runtime.Identity); err != nil {
		t.Fatalf("wait after release = %v", err)
	}
	final, err := registry.Snapshot(runtime.Identity)
	if err != nil || final.ActiveTriggers != 0 || !final.Draining {
		t.Fatalf("snapshot = %#v, %v", final, err)
	}
}

func TestPluginScheduleAdmissionRejectsStaleInstanceAfterPublish(t *testing.T) {
	registry := NewPluginScheduleAdmissionRegistry()
	oldRuntime := pluginScheduleRuntimeFixture("instance-a", "1.0.0", "sha256:a")
	newRuntime := pluginScheduleRuntimeFixture("instance-b", "2.0.0", "sha256:b")
	if _, err := registry.PublishActive(oldRuntime); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.PublishActive(newRuntime); err != nil {
		t.Fatal(err)
	}
	if _, _, err := registry.AcquireTrigger(context.Background(), oldRuntime.Identity, "daily-sync"); !errors.Is(err, ErrPluginScheduleRuntimeStale) {
		t.Fatalf("old instance acquire = %v", err)
	}
	oldSnapshot, err := registry.Snapshot(oldRuntime.Identity)
	if err != nil || oldSnapshot.Active || !oldSnapshot.Draining {
		t.Fatalf("old snapshot = %#v, %v", oldSnapshot, err)
	}
	rolledBack, err := registry.PublishActive(oldRuntime)
	if err != nil || !rolledBack.Active || rolledBack.Draining {
		t.Fatalf("republish retained instance = %#v, %v", rolledBack, err)
	}
	newSnapshot, err := registry.Snapshot(newRuntime.Identity)
	if err != nil || newSnapshot.Active || !newSnapshot.Draining {
		t.Fatalf("rollback did not drain replacement: %#v, %v", newSnapshot, err)
	}
	if _, _, err := registry.AcquireTrigger(context.Background(), newRuntime.Identity, "daily-sync"); !errors.Is(err, ErrPluginScheduleRuntimeStale) {
		t.Fatalf("rolled-back instance acquire = %v", err)
	}
	if _, lease, err := registry.AcquireTrigger(context.Background(), oldRuntime.Identity, "daily-sync"); err != nil {
		t.Fatalf("restored instance acquire = %v", err)
	} else {
		lease.Release()
	}
}

func TestPluginScheduleAcquireAndDrainHaveOneLinearizationBoundary(t *testing.T) {
	registry := NewPluginScheduleAdmissionRegistry()
	runtime := pluginScheduleRuntimeFixture("instance-race", "1.0.0", "sha256:race")
	if _, err := registry.PublishActive(runtime); err != nil {
		t.Fatal(err)
	}

	const callers = 64
	start := make(chan struct{})
	var wg sync.WaitGroup
	var mu sync.Mutex
	var leases []*PluginScheduleTriggerLease
	errorsSeen := make([]error, 0, callers)
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, lease, err := registry.AcquireTrigger(context.Background(), runtime.Identity, "daily-sync")
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errorsSeen = append(errorsSeen, err)
				return
			}
			leases = append(leases, lease)
		}()
	}
	drainDone := make(chan error, 1)
	go func() {
		<-start
		_, err := registry.BeginDrain(runtime.Identity)
		drainDone <- err
	}()
	close(start)
	wg.Wait()
	if err := <-drainDone; err != nil {
		t.Fatal(err)
	}
	for _, err := range errorsSeen {
		if !errors.Is(err, ErrPluginScheduleDraining) {
			t.Fatalf("race acquire error = %v", err)
		}
	}
	if len(leases)+len(errorsSeen) != callers {
		t.Fatalf("outcomes = leases %d + errors %d", len(leases), len(errorsSeen))
	}
	if _, _, err := registry.AcquireTrigger(context.Background(), runtime.Identity, "daily-sync"); !errors.Is(err, ErrPluginScheduleDraining) {
		t.Fatalf("acquire after drain = %v", err)
	}
	for _, lease := range leases {
		lease.Release()
	}
	if err := registry.WaitDrain(context.Background(), runtime.Identity); err != nil {
		t.Fatal(err)
	}
}

func TestPluginScheduleAdmissionValidatesCompleteHostOwnedDeclaration(t *testing.T) {
	registry := NewPluginScheduleAdmissionRegistry()
	runtime := pluginScheduleRuntimeFixture("instance-a", "1.0.0", "sha256:a")
	runtime.Schedules[0].Timezone = ""
	if _, err := registry.PublishActive(runtime); !errors.Is(err, ErrPluginScheduleInvalid) {
		t.Fatalf("publish incomplete declaration = %v", err)
	}

	runtime = pluginScheduleRuntimeFixture("instance-a", "1.0.0", "sha256:a")
	if _, err := registry.PublishActive(runtime); err != nil {
		t.Fatal(err)
	}
	if _, _, err := registry.AcquireTrigger(context.Background(), runtime.Identity, "undeclared"); !errors.Is(err, ErrPluginScheduleNotDeclared) {
		t.Fatalf("undeclared schedule = %v", err)
	}
}

func pluginScheduleRuntimeFixture(instanceID, version, digest string) PluginScheduleRuntime {
	return PluginScheduleRuntime{
		Identity: PluginScheduleRuntimeIdentity{
			ExtensionID: "demo.plugin", ExtensionVersion: version, ArtifactDigest: digest, InstanceID: instanceID,
		},
		Schedules: []PluginScheduleDeclaration{{
			ScheduleID: "daily-sync", JobName: "demo.sync", JobContract: "demo.plugin.job.sync@1",
			Cron: "0 3 * * *", Timezone: "Asia/Shanghai",
		}},
	}
}
