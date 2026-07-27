package extensions

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const (
	DefaultPluginRuntimeNodeLease         = 45 * time.Second
	DefaultPluginRuntimeHeartbeatInterval = 10 * time.Second
	DefaultPluginRuntimePollInterval      = 5 * time.Second
)

var (
	ErrPluginRuntimeCoordinatorInvalid = errors.New("extensions: plugin runtime coordinator is invalid")
	ErrPluginRuntimeCoordinatorRunning = errors.New("extensions: plugin runtime coordinator is already running")
	ErrPluginRuntimeCoordinatorRetired = errors.New("extensions: plugin runtime coordinator boot is retired")
	errPluginRuntimeDesiredNotSeeded   = errors.New("extensions: plugin runtime desired publication is not seeded")
	// errPluginRuntimeLifecyclePending keeps a coordinator from applying an
	// older full-set snapshot while a lifecycle operation is directly moving an
	// exact runtime through its staged/published/drained boundary. The pending
	// operation will publish its own durable revision before it completes.
	errPluginRuntimeLifecyclePending = errors.New("extensions: plugin runtime lifecycle operation is pending")
)

// PluginRuntimeFullSetApplier owns the process-local atomic switch from the
// current runtime set to one exact durable publication. Implementations must be
// idempotent because an ambiguous acknowledgement commit is retried. The result
// must contain the complete active set and each node-local runtime instance id.
// Every wait and transition must return promptly after ctx cancellation so a
// lost heartbeat can fence further process-local changes before the lease ends.
type PluginRuntimeFullSetApplier interface {
	ApplyPluginRuntimeFullSet(
		context.Context,
		PluginRuntimePublication,
	) ([]PluginRuntimeAppliedMember, error)
}

type pluginRuntimeCoordinatorRepository interface {
	PluginRuntimeNodeRepository
	LatestPluginRuntimePublication(context.Context) (PluginRuntimePublication, error)
}

// pluginRuntimeLifecycleFenceRepository is deliberately optional for narrow
// coordinator unit doubles. Production PostgresStore implements it, making an
// open lifecycle operation a fail-closed full-set publication fence.
type pluginRuntimeLifecycleFenceRepository interface {
	ListOpenLifecycleOperations(context.Context, int) ([]LifecycleOperation, error)
}

type PluginRuntimeCoordinatorConfig struct {
	Identity          PluginRuntimeNodeIdentity
	LeaseDuration     time.Duration
	HeartbeatInterval time.Duration
	PollInterval      time.Duration
	OnError           func(error)
	OnReady           func()
}

// PluginRuntimeCoordinator converges one API or worker boot to PostgreSQL's
// latest immutable full-set publication. LISTEN is only a wake hint; startup
// and periodic durable reads provide correctness after disconnects.
type PluginRuntimeCoordinator struct {
	repository    pluginRuntimeCoordinatorRepository
	applier       PluginRuntimeFullSetApplier
	notifications PluginRuntimePublicationNotificationSource
	config        PluginRuntimeCoordinatorConfig
	runToken      chan struct{}
}

func NewPluginRuntimeCoordinator(
	repository pluginRuntimeCoordinatorRepository,
	applier PluginRuntimeFullSetApplier,
	notifications PluginRuntimePublicationNotificationSource,
	config PluginRuntimeCoordinatorConfig,
) (*PluginRuntimeCoordinator, error) {
	if config.LeaseDuration == 0 {
		config.LeaseDuration = DefaultPluginRuntimeNodeLease
	}
	if config.HeartbeatInterval == 0 {
		config.HeartbeatInterval = DefaultPluginRuntimeHeartbeatInterval
	}
	if config.PollInterval == 0 {
		config.PollInterval = DefaultPluginRuntimePollInterval
	}
	if repository == nil || applier == nil || !validPluginRuntimeNodeIdentity(config.Identity) ||
		!validPluginRuntimeNodeLease(config.LeaseDuration) || config.HeartbeatInterval <= 0 ||
		config.HeartbeatInterval > config.LeaseDuration/3 || config.PollInterval <= 0 {
		return nil, ErrPluginRuntimeCoordinatorInvalid
	}
	coordinator := &PluginRuntimeCoordinator{
		repository: repository, applier: applier, notifications: notifications, config: config,
		runToken: make(chan struct{}, 1),
	}
	coordinator.runToken <- struct{}{}
	return coordinator, nil
}

// Run owns the boot lease until cancellation or a heartbeat failure. A
// heartbeat failure is terminal: continuing an apply without durable ownership
// could publish runtime state that this boot is no longer allowed to attest.
func (c *PluginRuntimeCoordinator) Run(ctx context.Context) error {
	if c == nil || ctx == nil {
		return ErrPluginRuntimeCoordinatorInvalid
	}
	if ctx.Err() != nil {
		return nil
	}
	select {
	case <-c.runToken:
		defer func() { c.runToken <- struct{}{} }()
	default:
		return ErrPluginRuntimeCoordinatorRunning
	}
	releaseBoot, err := acquirePluginRuntimeCoordinatorBoot(c.config.Identity)
	if err != nil {
		return err
	}
	defer releaseBoot()

	node, err := c.repository.RegisterPluginRuntimeNode(ctx, c.config.Identity, c.config.LeaseDuration)
	if err != nil {
		if ctx.Err() != nil {
			return nil
		}
		if errors.Is(err, ErrPluginRuntimeNodeLeaseLost) {
			retirePluginRuntimeCoordinatorBoot(c.config.Identity, err)
		}
		return err
	}
	if !validPluginRuntimeCoordinatorNode(node, c.config.Identity) {
		return ErrPluginRuntimeCoordinatorInvalid
	}
	// Durable node progress refers to this boot's process-local Manager and
	// runtime instance ids. Once Run exits, the identity is one-shot: a rebuilt
	// Manager must register a new BootID and reapply the latest full set.
	defer retirePluginRuntimeCoordinatorBoot(c.config.Identity, ErrPluginRuntimeCoordinatorRetired)

	runCtx, cancel := context.WithCancel(ctx)
	wake := make(chan struct{}, 1)
	heartbeatErrors := make(chan error, 1)
	workersDone := make(chan struct{}, 2)
	workerCount := 1
	go c.heartbeat(runCtx, cancel, heartbeatErrors, workersDone)
	if c.notifications != nil {
		workerCount++
		go func() {
			defer func() { workersDone <- struct{}{} }()
			c.notifications.WatchPluginRuntimePublications(runCtx, func() {
				select {
				case wake <- struct{}{}:
				default:
				}
			})
		}()
	}
	defer func() {
		cancel()
		for range workerCount {
			<-workersDone
		}
	}()

	poll := time.NewTicker(c.config.PollInterval)
	defer poll.Stop()
	converged, err := c.reconcileAndReport(runCtx)
	if err != nil {
		if heartbeatErr := receivePluginRuntimeHeartbeatError(heartbeatErrors); heartbeatErr != nil {
			return heartbeatErr
		}
		if ctx.Err() != nil {
			return nil
		}
		if errors.Is(err, ErrPluginRuntimeNodeLeaseLost) {
			retirePluginRuntimeCoordinatorBoot(c.config.Identity, err)
		}
		return err
	}
	if heartbeatErr := receivePluginRuntimeHeartbeatError(heartbeatErrors); heartbeatErr != nil {
		return heartbeatErr
	}
	if ctx.Err() != nil {
		return nil
	}
	ready := converged
	if ready && c.config.OnReady != nil {
		c.config.OnReady()
	}

	for {
		select {
		case heartbeatErr := <-heartbeatErrors:
			return heartbeatErr
		case <-ctx.Done():
			if heartbeatErr := receivePluginRuntimeHeartbeatError(heartbeatErrors); heartbeatErr != nil {
				return heartbeatErr
			}
			return nil
		case <-poll.C:
			converged, err := c.reconcileAndReport(runCtx)
			if err != nil {
				if heartbeatErr := receivePluginRuntimeHeartbeatError(heartbeatErrors); heartbeatErr != nil {
					return heartbeatErr
				}
				if ctx.Err() != nil {
					return nil
				}
				if errors.Is(err, ErrPluginRuntimeNodeLeaseLost) {
					retirePluginRuntimeCoordinatorBoot(c.config.Identity, err)
				}
				return err
			}
			if converged && !ready {
				ready = true
				if c.config.OnReady != nil {
					c.config.OnReady()
				}
			}
		case <-wake:
			converged, err := c.reconcileAndReport(runCtx)
			if err != nil {
				if heartbeatErr := receivePluginRuntimeHeartbeatError(heartbeatErrors); heartbeatErr != nil {
					return heartbeatErr
				}
				if ctx.Err() != nil {
					return nil
				}
				if errors.Is(err, ErrPluginRuntimeNodeLeaseLost) {
					retirePluginRuntimeCoordinatorBoot(c.config.Identity, err)
				}
				return err
			}
			if converged && !ready {
				ready = true
				if c.config.OnReady != nil {
					c.config.OnReady()
				}
			}
		}
	}
}

// reconcileOnce runs only under Run's boot ownership and lease heartbeat.
// Each publication is a full set, so obsolete intermediate revisions may be
// skipped while durable node progress remains strictly monotonic.
func (c *PluginRuntimeCoordinator) reconcileOnce(ctx context.Context) error {
	for {
		publication, err := c.repository.LatestPluginRuntimePublication(ctx)
		if errors.Is(err, ErrPluginRuntimePublicationNotFound) {
			return errPluginRuntimeDesiredNotSeeded
		}
		if err != nil {
			return err
		}
		publication, err = normalizedPluginRuntimePublication(publication)
		if err != nil {
			return fmt.Errorf("%w: invalid durable publication", ErrPluginRuntimeCoordinatorInvalid)
		}
		if fence, ok := c.repository.(pluginRuntimeLifecycleFenceRepository); ok {
			open, fenceErr := fence.ListOpenLifecycleOperations(ctx, 1)
			if fenceErr != nil {
				return fmt.Errorf("inspect open lifecycle operations before runtime apply: %w", fenceErr)
			}
			if len(open) != 0 {
				return errPluginRuntimeLifecyclePending
			}
		}

		node, err := c.repository.GetPluginRuntimeNode(ctx, c.config.Identity)
		if err != nil {
			return err
		}
		if !validPluginRuntimeCoordinatorNode(node, c.config.Identity) {
			return ErrPluginRuntimeCoordinatorInvalid
		}
		if node.LastAppliedRevision >= publication.Revision {
			return nil
		}

		ack, err := c.repository.BeginPluginRuntimePublicationApply(
			ctx, c.config.Identity, publication.Revision,
		)
		if err != nil {
			return err
		}
		if !validPluginRuntimeApplyingAck(ack, c.config.Identity, publication.Revision) {
			return ErrPluginRuntimeCoordinatorInvalid
		}

		applied, applyErr := c.applier.ApplyPluginRuntimeFullSet(
			ctx, clonePluginRuntimePublication(publication),
		)
		if applyErr == nil {
			var appliedDigest string
			applied, appliedDigest, applyErr = canonicalPluginRuntimeAppliedMembers(publication.Members, applied)
			if applyErr == nil && appliedDigest != publication.MembersDigest {
				applyErr = ErrPluginRuntimeAckConflict
			}
		}
		if applyErr != nil {
			failureRecorded, recordedErr := c.recordPluginRuntimeApplyFailure(ctx, publication, ack, applyErr)
			if failureRecorded && errors.Is(applyErr, ErrPluginRuntimePublicationSuperseded) {
				// Superseded also covers a process whose applied revision is ahead of
				// durable state. Only a genuinely newer durable full set may bypass the
				// normal poll/backoff path; otherwise an immediate retry would spin on
				// the same revision and write unbounded failed acknowledgements.
				latest, latestErr := c.repository.LatestPluginRuntimePublication(ctx)
				if latestErr != nil {
					return errors.Join(recordedErr, fmt.Errorf("verify superseding plugin runtime publication: %w", latestErr))
				}
				latest, latestErr = normalizedPluginRuntimePublication(latest)
				if latestErr != nil {
					return errors.Join(recordedErr, fmt.Errorf("%w: invalid superseding durable publication", ErrPluginRuntimeCoordinatorInvalid))
				}
				if latest.Revision > publication.Revision {
					continue
				}
			}
			return recordedErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}

		completed, err := c.repository.CompletePluginRuntimePublicationApply(
			ctx, c.config.Identity, publication, ack.Revision, applied,
		)
		if err != nil {
			// A connection can fail after PostgreSQL committed the exact ack,
			// evidence, and node advance. The monotonic node row is sufficient
			// durable proof; otherwise the idempotent applier must retry later.
			advanced, verifyErr := c.repository.GetPluginRuntimeNode(ctx, c.config.Identity)
			if verifyErr == nil && validPluginRuntimeCoordinatorNode(advanced, c.config.Identity) &&
				advanced.LastAppliedRevision >= publication.Revision {
				continue
			}
			if verifyErr != nil {
				return errors.Join(err, fmt.Errorf("verify plugin runtime completion: %w", verifyErr))
			}
			if !validPluginRuntimeCoordinatorNode(advanced, c.config.Identity) {
				return errors.Join(err, ErrPluginRuntimeCoordinatorInvalid)
			}
			return err
		}
		if !validPluginRuntimeCompletedAck(completed, ack, publication) {
			return ErrPluginRuntimeCoordinatorInvalid
		}
		// Reload Latest after every successful switch. A publication committed
		// during apply is therefore observed even when its NOTIFY was coalesced.
	}
}

func (c *PluginRuntimeCoordinator) heartbeat(
	ctx context.Context,
	cancel context.CancelFunc,
	errorsCh chan<- error,
	done chan<- struct{},
) {
	defer func() { done <- struct{}{} }()
	ticker := time.NewTicker(c.config.HeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			heartbeatCtx, stopHeartbeat := context.WithTimeout(ctx, c.config.HeartbeatInterval)
			node, err := c.repository.HeartbeatPluginRuntimeNode(
				heartbeatCtx, c.config.Identity, c.config.LeaseDuration,
			)
			heartbeatContextErr := heartbeatCtx.Err()
			stopHeartbeat()
			// Parent shutdown may race the ticker after this branch was selected.
			// A canceled repository call is not lease loss and must not retire the
			// otherwise reusable boot identity.
			if ctx.Err() != nil {
				return
			}
			// A heartbeat whose outcome is unknown past its own deadline cannot
			// authorize more process-local changes, even if the repository returned
			// a late nil result.
			if heartbeatContextErr != nil {
				err = heartbeatContextErr
			}
			if err == nil && !validPluginRuntimeCoordinatorNode(node, c.config.Identity) {
				err = ErrPluginRuntimeCoordinatorInvalid
			}
			if err != nil {
				retirePluginRuntimeCoordinatorBoot(c.config.Identity, err)
				errorsCh <- err
				cancel()
				return
			}
		}
	}
}

func (c *PluginRuntimeCoordinator) reconcileAndReport(ctx context.Context) (bool, error) {
	err := c.reconcileOnce(ctx)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, errPluginRuntimeDesiredNotSeeded) {
		return false, nil
	}
	if errors.Is(err, errPluginRuntimeLifecyclePending) {
		return false, nil
	}
	if ctx.Err() != nil {
		return false, nil
	}
	if errors.Is(err, ErrPluginRuntimeNodeLeaseLost) ||
		errors.Is(err, ErrPluginRuntimeCoordinatorInvalid) {
		return false, err
	}
	c.report(err)
	return false, nil
}

func (c *PluginRuntimeCoordinator) recordPluginRuntimeApplyFailure(
	ctx context.Context,
	publication PluginRuntimePublication,
	ack PluginRuntimePublicationAck,
	cause error,
) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	failed, failErr := c.repository.FailPluginRuntimePublicationApply(
		ctx,
		c.config.Identity,
		publication.Revision,
		ack.Revision,
		pluginRuntimeCoordinatorFailureReason(cause),
	)
	applyErr := fmt.Errorf(
		"apply plugin runtime revision %d attempt %d: %w",
		publication.Revision, ack.AttemptCount, cause,
	)
	if failErr != nil {
		return false, errors.Join(
			applyErr,
			fmt.Errorf("record plugin runtime revision %d failure: %w", publication.Revision, failErr),
		)
	}
	if !validPluginRuntimeFailedAck(failed, ack, publication.Revision) {
		return false, errors.Join(applyErr, ErrPluginRuntimeCoordinatorInvalid)
	}
	return true, applyErr
}

func (c *PluginRuntimeCoordinator) report(err error) {
	if err != nil && c.config.OnError != nil {
		c.config.OnError(err)
	}
}
