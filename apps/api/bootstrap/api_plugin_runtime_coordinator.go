package bootstrap

import (
	"context"
	"sync"
)

// superviseAPIPluginRuntimeCoordinator bridges the coordinator's terminal
// lease failure into the command-owned process lifecycle. closeRuntime must
// revoke process-local plugin admission before the failure becomes observable.
func superviseAPIPluginRuntimeCoordinator(
	coordinator *pluginRuntimeCoordinatorRuntime,
	closeRuntime func(),
) (<-chan error, <-chan struct{}) {
	if coordinator == nil || !coordinator.Active() {
		return nil, nil
	}
	failures := make(chan error, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer close(failures)
		runtimeErr, ok := <-coordinator.Failures()
		if !ok || runtimeErr == nil {
			return
		}
		if closeRuntime != nil {
			closeRuntime()
		}
		failures <- runtimeErr
	}()
	return failures, done
}

func mergeAPIRuntimeFailureSources(sources ...<-chan error) (<-chan error, func()) {
	active := make([]<-chan error, 0, len(sources))
	for _, source := range sources {
		if source != nil {
			active = append(active, source)
		}
	}
	if len(active) == 0 {
		return nil, func() {}
	}
	ctx, cancel := context.WithCancel(context.Background())
	failures := make(chan error, 1)
	var wait sync.WaitGroup
	wait.Add(len(active))
	for _, source := range active {
		go func(source <-chan error) {
			defer wait.Done()
			select {
			case runtimeErr, ok := <-source:
				if !ok || runtimeErr == nil {
					return
				}
				select {
				case failures <- runtimeErr:
					cancel()
				case <-ctx.Done():
				}
			case <-ctx.Done():
			}
		}(source)
	}
	go func() {
		wait.Wait()
		close(failures)
	}()
	return failures, cancel
}
