package extensionsruntime

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type identityProviderCommitFenceOutcome struct {
	result     error
	violation  error
	panicValue any
}

// identityProviderCommitFenceState owns the one-shot Host commit fence. Closing
// the acceptance window is ordered with fence completion under the same mutex,
// so an in-flight fence can never report success after Accept has returned.
type identityProviderCommitFenceState struct {
	mu         sync.Mutex
	open       bool
	started    bool
	result     error
	violation  error
	panicValue any
	done       chan struct{}
	validate   func() error
}

func newIdentityProviderCommitFenceState(validate func() error) *identityProviderCommitFenceState {
	return &identityProviderCommitFenceState{
		open: true, done: make(chan struct{}), validate: validate,
	}
}

func (s *identityProviderCommitFenceState) Run() (result error) {
	if s == nil {
		return ErrIdentityProviderInvocationInvalid
	}
	s.mu.Lock()
	if !s.open {
		s.mu.Unlock()
		return ErrIdentityProviderInvocationInvalid
	}
	if s.started {
		s.violation = errors.Join(s.violation, ErrIdentityProviderInvocationInvalid)
		s.mu.Unlock()
		return ErrIdentityProviderInvocationInvalid
	}
	s.started = true
	s.mu.Unlock()

	defer func() {
		panicValue := recover()
		if panicValue != nil {
			result = fmt.Errorf("%w: commit fence panicked", ErrIdentityProviderInvocationInvalid)
		}
		s.mu.Lock()
		if !s.open {
			late := fmt.Errorf(
				"%w: commit fence completed after Host acceptance returned",
				ErrIdentityProviderInvocationInvalid,
			)
			result = errors.Join(result, late)
			s.violation = errors.Join(s.violation, late)
		}
		s.result = result
		s.panicValue = panicValue
		close(s.done)
		s.mu.Unlock()
	}()
	if s.validate == nil {
		return ErrIdentityProviderInvocationInvalid
	}
	return s.validate()
}

func (s *identityProviderCommitFenceState) Finish() identityProviderCommitFenceOutcome {
	if s == nil {
		return identityProviderCommitFenceOutcome{violation: ErrIdentityProviderInvocationInvalid}
	}
	s.mu.Lock()
	started := s.started
	if s.open {
		s.open = false
		s.violation = errors.Join(s.violation, fmt.Errorf(
			"%w: Host acceptance window was not closed",
			ErrIdentityProviderInvocationInvalid,
		))
	}
	s.mu.Unlock()
	if started {
		<-s.done
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	violation := s.violation
	if !started {
		violation = errors.Join(violation, fmt.Errorf(
			"%w: commit fence must be called exactly once",
			ErrIdentityProviderInvocationInvalid,
		))
	}
	return identityProviderCommitFenceOutcome{
		result: s.result, violation: violation, panicValue: s.panicValue,
	}
}

func (s *identityProviderCommitFenceState) closeAcceptance() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.open = false
	return s.started
}

func runIdentityProviderAccept(
	accept IdentityProviderAccept,
	ctx context.Context,
	accepted IdentityProviderInvocationResult,
	fence *identityProviderCommitFenceState,
) (acceptErr error, panicValue any) {
	defer func() {
		panicValue = recover()
		fence.closeAcceptance()
	}()
	acceptErr = accept(ctx, accepted, fence.Run)
	return acceptErr, nil
}
