package identityregistry

import (
	"context"
	"sync"
)

// runSessionPolicyMutationGate enforces the callback lifetime promised by the
// additive lifecycle gate. Only the first callback admitted before gate return
// may execute the durable reconciliation.
func runSessionPolicyMutationGate(
	ctx context.Context,
	gate SessionPolicyLifecycleMutationGate,
	run func() (DurableState, error),
) (DurableState, error) {
	if gate == nil {
		return run()
	}

	var mu sync.Mutex
	open := true
	started := false
	var state DurableState
	var callbackErr error
	var callbackPanic any
	done := make(chan struct{})

	callback := func() (result error) {
		mu.Lock()
		if !open {
			mu.Unlock()
			return errIdentityRegistryStore
		}
		if err := ctx.Err(); err != nil {
			mu.Unlock()
			return err
		}
		if started {
			mu.Unlock()
			return errIdentityRegistryStore
		}
		started = true
		mu.Unlock()

		defer func() {
			panicValue := recover()
			if panicValue != nil {
				result = errIdentityRegistryStore
			}
			mu.Lock()
			callbackErr = result
			callbackPanic = panicValue
			close(done)
			mu.Unlock()
		}()
		state, result = run()
		return result
	}

	var gateErr error
	var gatePanic any
	func() {
		defer func() { gatePanic = recover() }()
		gateErr = gate.RunSessionPolicyMutation(ctx, callback)
	}()

	mu.Lock()
	open = false
	wasStarted := started
	mu.Unlock()
	if wasStarted {
		<-done
	}
	mu.Lock()
	resultState := state
	resultErr := callbackErr
	panicValue := callbackPanic
	mu.Unlock()
	if panicValue != nil {
		panic(panicValue)
	}
	if gatePanic != nil {
		panic(gatePanic)
	}
	if resultErr != nil {
		return resultState, resultErr
	}
	if !wasStarted {
		if gateErr != nil {
			return DurableState{}, mapStoreError(gateErr)
		}
		return DurableState{}, errIdentityRegistryStore
	}
	// The reconciliation callback may already have committed. Once it succeeds,
	// its durable state is terminal even if a malformed gate later reports an
	// error or attempts a duplicate callback.
	return resultState, nil
}
