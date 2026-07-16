package extensions

import "sync"

// A database acknowledgement is a CAS fence for evidence, not ownership of
// external process side effects. Run therefore owns one process-local boot
// identity until shutdown, and a heartbeat-failed identity cannot be reused.
var pluginRuntimeCoordinatorBootOwners = struct {
	sync.Mutex
	owners   map[PluginRuntimeNodeIdentity]struct{}
	terminal map[PluginRuntimeNodeIdentity]error
}{
	owners:   make(map[PluginRuntimeNodeIdentity]struct{}),
	terminal: make(map[PluginRuntimeNodeIdentity]error),
}

func acquirePluginRuntimeCoordinatorBoot(identity PluginRuntimeNodeIdentity) (func(), error) {
	pluginRuntimeCoordinatorBootOwners.Lock()
	defer pluginRuntimeCoordinatorBootOwners.Unlock()
	if err := pluginRuntimeCoordinatorBootOwners.terminal[identity]; err != nil {
		return nil, err
	}
	if _, exists := pluginRuntimeCoordinatorBootOwners.owners[identity]; exists {
		return nil, ErrPluginRuntimeCoordinatorRunning
	}
	pluginRuntimeCoordinatorBootOwners.owners[identity] = struct{}{}
	return func() {
		pluginRuntimeCoordinatorBootOwners.Lock()
		delete(pluginRuntimeCoordinatorBootOwners.owners, identity)
		pluginRuntimeCoordinatorBootOwners.Unlock()
	}, nil
}

func retirePluginRuntimeCoordinatorBoot(identity PluginRuntimeNodeIdentity, err error) {
	if err == nil {
		return
	}
	pluginRuntimeCoordinatorBootOwners.Lock()
	if pluginRuntimeCoordinatorBootOwners.terminal[identity] == nil {
		pluginRuntimeCoordinatorBootOwners.terminal[identity] = err
	}
	pluginRuntimeCoordinatorBootOwners.Unlock()
}
