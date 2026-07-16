package bootstrap

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

var errAPIThemeRuntimeWatcherTest = errors.New("API theme runtime watcher test failure")

type apiThemeRuntimeWatcherTestRunner func(context.Context) error

func (runner apiThemeRuntimeWatcherTestRunner) Run(ctx context.Context) error {
	return runner(ctx)
}

type apiThemeRuntimeWatcherTestStartResult struct {
	runtime *apiThemeRuntimeWatcherRuntime
	err     error
}

func TestNormalizeThemeRuntimeNodeIDKeepsStableBoundedIdentity(t *testing.T) {
	if nodeID, err := normalizeThemeRuntimeNodeID("  api-1  "); err != nil || nodeID != "api-1" {
		t.Fatalf("nodeID=%q err=%v", nodeID, err)
	}
	long := strings.Repeat("node", 80)
	first, err := normalizeThemeRuntimeNodeID(long)
	if err != nil || len([]byte(first)) > 128 || !strings.HasPrefix(first, "host-") {
		t.Fatalf("hashed nodeID=%q err=%v", first, err)
	}
	second, err := normalizeThemeRuntimeNodeID(long)
	if err != nil || second != first {
		t.Fatalf("unstable nodeID first=%q second=%q err=%v", first, second, err)
	}
	if _, err := normalizeThemeRuntimeNodeID("   "); err == nil {
		t.Fatal("blank hostname was accepted")
	}
}

func TestAPIThemeRuntimeWatcherLaunchWaitsForInitialConvergenceAndStopsCleanly(t *testing.T) {
	started := make(chan struct{})
	converge := make(chan struct{})
	result := make(chan apiThemeRuntimeWatcherTestStartResult, 1)
	go func() {
		runtime, err := launchAPIThemeRuntimeWatcher(t.Context(), apiThemeRuntimeWatcherLaunchConfig{
			Build: func(onReady func()) (apiThemeRuntimeWatcherRunner, error) {
				return apiThemeRuntimeWatcherTestRunner(func(ctx context.Context) error {
					close(started)
					select {
					case <-converge:
						onReady()
					case <-ctx.Done():
						return nil
					}
					<-ctx.Done()
					return nil
				}), nil
			},
			StopTimeout: time.Second,
		})
		result <- apiThemeRuntimeWatcherTestStartResult{runtime: runtime, err: err}
	}()
	waitAPIThemeRuntimeWatcherSignal(t, started)
	select {
	case premature := <-result:
		t.Fatalf("bootstrap returned before initial convergence: %#v", premature)
	default:
	}
	close(converge)
	launched := waitAPIThemeRuntimeWatcherStart(t, result)
	if launched.err != nil || launched.runtime == nil {
		t.Fatalf("runtime=%#v err=%v", launched.runtime, launched.err)
	}
	if err := launched.runtime.Stop(t.Context()); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if failure, ok := <-launched.runtime.Failures(); ok || failure != nil {
		t.Fatalf("normal stop failure=%v open=%t", failure, ok)
	}
}

func TestAPIThemeRuntimeWatcherFailureRestoresFallbackBeforeReporting(t *testing.T) {
	release := make(chan struct{})
	var restored atomic.Bool
	runtime, err := launchAPIThemeRuntimeWatcher(t.Context(), apiThemeRuntimeWatcherLaunchConfig{
		Build: func(onReady func()) (apiThemeRuntimeWatcherRunner, error) {
			return apiThemeRuntimeWatcherTestRunner(func(context.Context) error {
				onReady()
				<-release
				return errAPIThemeRuntimeWatcherTest
			}), nil
		},
		Fallback: func(context.Context) error {
			restored.Store(true)
			return nil
		},
		StopTimeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	close(release)
	select {
	case failure, ok := <-runtime.Failures():
		if !ok || !errors.Is(failure, errAPIThemeRuntimeWatcherTest) {
			t.Fatalf("failure=%v open=%t", failure, ok)
		}
		if !restored.Load() {
			t.Fatal("terminal failure became observable before safe theme fallback")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for terminal theme watcher failure")
	}
	waitAPIThemeRuntimeWatcherSignal(t, runtime.Done())
}

func TestAPIThemeRuntimeWatcherStopIsBoundedAndIdempotent(t *testing.T) {
	cancelled := make(chan struct{})
	release := make(chan struct{})
	runtime, err := launchAPIThemeRuntimeWatcher(t.Context(), apiThemeRuntimeWatcherLaunchConfig{
		Build: func(onReady func()) (apiThemeRuntimeWatcherRunner, error) {
			return apiThemeRuntimeWatcherTestRunner(func(ctx context.Context) error {
				onReady()
				<-ctx.Done()
				close(cancelled)
				<-release
				return nil
			}), nil
		},
		StopTimeout: 10 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	if stopErr := runtime.Stop(t.Context()); !errors.Is(stopErr, context.DeadlineExceeded) {
		t.Fatalf("bounded stop error=%v", stopErr)
	}
	waitAPIThemeRuntimeWatcherSignal(t, cancelled)
	close(release)
	waitAPIThemeRuntimeWatcherSignal(t, runtime.Done())
	for range 4 {
		if stopErr := runtime.Stop(t.Context()); stopErr != nil {
			t.Fatalf("idempotent stop error=%v", stopErr)
		}
	}
}

func waitAPIThemeRuntimeWatcherStart(
	t *testing.T,
	result <-chan apiThemeRuntimeWatcherTestStartResult,
) apiThemeRuntimeWatcherTestStartResult {
	t.Helper()
	select {
	case launched := <-result:
		return launched
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for API theme runtime watcher startup")
		return apiThemeRuntimeWatcherTestStartResult{}
	}
}

func waitAPIThemeRuntimeWatcherSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for API theme runtime watcher signal")
	}
}
