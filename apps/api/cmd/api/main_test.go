package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
)

var (
	errAPILifecycleRuntimeTest  = errors.New("api runtime lease lost")
	errAPILifecycleListenTest   = errors.New("api listen failed")
	errAPILifecycleShutdownTest = errors.New("api shutdown failed")
)

func TestRunAPILifecycleRuntimeFailureDrainsHTTPBeforeClose(t *testing.T) {
	api := newAPILifecycleTestDouble()
	go func() {
		<-api.listenStarted
		api.failures <- errAPILifecycleRuntimeTest
	}()

	err := runAPILifecycle(context.Background(), apiLifecycleTestLogger(), api)
	if !errors.Is(err, errAPILifecycleRuntimeTest) {
		t.Fatalf("terminal result = %v", err)
	}
	if got := api.snapshot(); len(got) != 3 || got[0] != "listen" || got[1] != "shutdown" || got[2] != "close" {
		t.Fatalf("lifecycle order = %#v", got)
	}
}

func TestRunAPILifecycleSignalShutdownWithNilFailureSource(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	api := newAPILifecycleTestDouble()
	go func() {
		<-api.listenStarted
		cancel()
	}()

	if err := runAPILifecycle(ctx, apiLifecycleTestLogger(), api); err != nil {
		t.Fatal(err)
	}
	if got := api.snapshot(); len(got) != 3 || got[0] != "listen" || got[1] != "shutdown" || got[2] != "close" {
		t.Fatalf("lifecycle order = %#v", got)
	}
}

func TestRunAPILifecycleJoinsListenAndShutdownFailures(t *testing.T) {
	api := newAPILifecycleTestDouble()
	api.listenErr = errAPILifecycleListenTest
	api.shutdownErr = errAPILifecycleShutdownTest

	err := runAPILifecycle(context.Background(), apiLifecycleTestLogger(), api)
	if !errors.Is(err, errAPILifecycleListenTest) || !errors.Is(err, errAPILifecycleShutdownTest) {
		t.Fatalf("joined result = %v", err)
	}
	if api.closeCalls != 1 {
		t.Fatalf("close calls = %d", api.closeCalls)
	}
}

func apiLifecycleTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type apiLifecycleTestDouble struct {
	mu            sync.Mutex
	events        []string
	failures      chan error
	listenStarted chan struct{}
	shutdown      chan struct{}
	shutdownOnce  sync.Once
	closeOnce     sync.Once
	listenErr     error
	shutdownErr   error
	closeCalls    int
}

func newAPILifecycleTestDouble() *apiLifecycleTestDouble {
	return &apiLifecycleTestDouble{
		failures: make(chan error, 1), listenStarted: make(chan struct{}), shutdown: make(chan struct{}),
	}
}

func (api *apiLifecycleTestDouble) Listen() error {
	api.add("listen")
	close(api.listenStarted)
	if api.listenErr != nil {
		return api.listenErr
	}
	<-api.shutdown
	return nil
}

func (api *apiLifecycleTestDouble) Shutdown(context.Context) error {
	api.add("shutdown")
	api.shutdownOnce.Do(func() { close(api.shutdown) })
	return api.shutdownErr
}

func (api *apiLifecycleTestDouble) Failures() <-chan error {
	return api.failures
}

func (api *apiLifecycleTestDouble) Close() {
	api.closeOnce.Do(func() {
		api.mu.Lock()
		defer api.mu.Unlock()
		api.closeCalls++
		api.events = append(api.events, "close")
	})
}

func (api *apiLifecycleTestDouble) add(event string) {
	api.mu.Lock()
	api.events = append(api.events, event)
	api.mu.Unlock()
}

func (api *apiLifecycleTestDouble) snapshot() []string {
	api.mu.Lock()
	defer api.mu.Unlock()
	return append([]string(nil), api.events...)
}
