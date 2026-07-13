package extensions

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

const (
	lifecycleCoordinatorLeaseDuration          = 30 * time.Second
	lifecycleCoordinatorLeaseHeartbeatInterval = 10 * time.Second
)

type lifecycleCoordinatorLeaseSession struct {
	repository LifecycleCoordinatorRepository
	attemptID  int64
	ownerToken string
	durationMS int64
	cancelRun  context.CancelFunc
	cancelBeat context.CancelFunc

	mu       sync.Mutex
	revision int64
	err      error
	stopOnce sync.Once
	stop     chan struct{}
	done     chan struct{}
}

func (c *LifecycleCoordinator) claimStepLease(
	ctx context.Context,
	attempt LifecycleStepAttempt,
	cancelRun context.CancelFunc,
) (*lifecycleCoordinatorLeaseSession, error) {
	ownerToken, err := lifecycleCoordinatorLeaseOwnerToken(attempt.OperationID, attempt.ID)
	if err != nil {
		return nil, err
	}
	durationMS := c.leaseDuration.Milliseconds()
	if durationMS <= 0 || c.leaseHeartbeatInterval <= 0 || c.leaseHeartbeatInterval >= c.leaseDuration {
		return nil, fmt.Errorf("%w: invalid lifecycle lease policy", ErrLifecycleCoordinatorInvalid)
	}
	claimed, err := c.repository.ClaimStepLease(ctx, ClaimLifecycleStepLeaseInput{
		AttemptID: attempt.ID, ExpectedRevision: attempt.LeaseRevision,
		OwnerToken: ownerToken, DurationMS: durationMS,
	})
	if err != nil {
		return nil, err
	}
	heartbeatCtx, cancelHeartbeat := context.WithCancel(ctx)
	session := &lifecycleCoordinatorLeaseSession{
		repository: c.repository, attemptID: attempt.ID, ownerToken: ownerToken,
		durationMS: durationMS, revision: claimed.LeaseRevision, cancelRun: cancelRun,
		cancelBeat: cancelHeartbeat,
		stop:       make(chan struct{}), done: make(chan struct{}),
	}
	go session.heartbeat(heartbeatCtx, c.leaseHeartbeatInterval)
	return session, nil
}

func (s *lifecycleCoordinatorLeaseSession) heartbeat(ctx context.Context, interval time.Duration) {
	defer close(s.done)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.mu.Lock()
			if s.err == nil {
				updated, err := s.repository.HeartbeatStepLease(ctx, HeartbeatLifecycleStepLeaseInput{
					AttemptID: s.attemptID, OwnerToken: s.ownerToken,
					Revision: s.revision, DurationMS: s.durationMS,
				})
				if err != nil {
					if ctx.Err() == nil {
						s.err = err
						s.cancelRun()
					}
				} else {
					s.revision = updated.LeaseRevision
				}
			}
			s.mu.Unlock()
		}
	}
}

func (s *lifecycleCoordinatorLeaseSession) stopHeartbeat() {
	s.stopOnce.Do(func() {
		close(s.stop)
		s.cancelBeat()
	})
	<-s.done
}

func (s *lifecycleCoordinatorLeaseSession) failure() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

func (s *lifecycleCoordinatorLeaseSession) updateProgress(
	ctx context.Context,
	input UpdateLifecycleStepProgressInput,
) (LifecycleStepAttempt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return LifecycleStepAttempt{}, s.err
	}
	input.LeaseOwnerToken = s.ownerToken
	input.LeaseRevision = s.revision
	return s.repository.UpdateStepProgress(ctx, input)
}

func (s *lifecycleCoordinatorLeaseSession) complete(
	ctx context.Context,
	input CompleteLifecycleStepAttemptInput,
) (LifecycleStepAttempt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return LifecycleStepAttempt{}, s.err
	}
	input.LeaseOwnerToken = s.ownerToken
	input.LeaseRevision = s.revision
	return s.repository.CompleteStepAttempt(ctx, input)
}

func lifecycleCoordinatorLeaseOwnerToken(operationID, attemptID int64) (string, error) {
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", fmt.Errorf("create lifecycle step lease owner: %w", err)
	}
	return fmt.Sprintf("lifecycle-coordinator:%d:%d:%s", operationID, attemptID, hex.EncodeToString(nonce[:])), nil
}
