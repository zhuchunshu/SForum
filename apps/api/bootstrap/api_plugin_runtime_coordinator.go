package bootstrap

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
