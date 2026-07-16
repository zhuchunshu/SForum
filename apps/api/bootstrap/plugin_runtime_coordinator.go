package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
)

const defaultPluginRuntimeCoordinatorStopTimeout = 10 * time.Second

var (
	errPluginRuntimeCoordinatorBootstrapInvalid   = errors.New("bootstrap: plugin runtime coordinator is invalid")
	errPluginRuntimeCoordinatorStoppedBeforeReady = errors.New("bootstrap: plugin runtime coordinator stopped before readiness")
)

// pluginRuntimeCoordinatorRunner keeps the bootstrap lifetime wrapper testable
// without weakening the production constructor's exact concrete dependencies.
type pluginRuntimeCoordinatorRunner interface {
	Run(context.Context) error
}

type pluginRuntimeCoordinatorBuilder func(
	extensions.PluginRuntimeNodeIdentity,
	func(),
	func(error),
) (pluginRuntimeCoordinatorRunner, error)

type pluginRuntimeCoordinatorBootstrapConfig struct {
	SafeMode    bool
	ProcessRole extensions.PluginRuntimeProcessRole
	Store       *extensions.PostgresStore
	Manager     *extensionsruntime.Manager
	Logger      *slog.Logger

	LeaseDuration     time.Duration
	HeartbeatInterval time.Duration
	PollInterval      time.Duration
	StopTimeout       time.Duration
}

type pluginRuntimeCoordinatorLaunchConfig struct {
	SafeMode    bool
	Identity    extensions.PluginRuntimeNodeIdentity
	Ensurer     extensions.InitialPluginRuntimePublicationEnsurer
	Build       pluginRuntimeCoordinatorBuilder
	StopTimeout time.Duration
}

// pluginRuntimeCoordinatorRuntime owns the coordinator goroutine. Failures is
// the fail-closed process signal: a non-nil value means the durable boot lease
// can no longer authorize this process-local Manager.
type pluginRuntimeCoordinatorRuntime struct {
	identity    extensions.PluginRuntimeNodeIdentity
	active      bool
	stopTimeout time.Duration

	cancel   context.CancelFunc
	done     chan struct{}
	failures chan error
	stopOnce sync.Once
	stopping atomic.Bool
	ready    atomic.Bool

	errMu sync.RWMutex
	err   error
}

// startPluginRuntimeCoordinator constructs the production exact full-set
// coordinator. Safe Mode returns before hostname lookup, genesis import, node
// registration, LISTEN, or any plugin process start.
func startPluginRuntimeCoordinator(
	ctx context.Context,
	config pluginRuntimeCoordinatorBootstrapConfig,
) (*pluginRuntimeCoordinatorRuntime, error) {
	if ctx == nil {
		return nil, errPluginRuntimeCoordinatorBootstrapInvalid
	}
	if config.SafeMode {
		return newInactivePluginRuntimeCoordinatorRuntime(), nil
	}
	if config.Store == nil || config.Manager == nil ||
		(config.ProcessRole != extensions.PluginRuntimeProcessAPI &&
			config.ProcessRole != extensions.PluginRuntimeProcessWorker) {
		return nil, errPluginRuntimeCoordinatorBootstrapInvalid
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("resolve plugin runtime node hostname: %w", err)
	}
	identity, err := newPluginRuntimeCoordinatorIdentity(
		hostname, config.ProcessRole, extensions.NewActivationBootID(),
	)
	if err != nil {
		return nil, err
	}

	report := func(message string, err error) {
		if err != nil && config.Logger != nil {
			config.Logger.Warn(message, "error", err)
		}
	}
	return launchPluginRuntimeCoordinator(ctx, pluginRuntimeCoordinatorLaunchConfig{
		Identity:    identity,
		Ensurer:     config.Store,
		StopTimeout: config.StopTimeout,
		Build: func(
			identity extensions.PluginRuntimeNodeIdentity,
			onReady func(),
			onError func(error),
		) (pluginRuntimeCoordinatorRunner, error) {
			// API 与 standalone worker 共用此路径：bootstrap 首轮 full-set
			// 允许在单 barrier 内 cold-start exact Protocol V1；成功后窗口关闭。
			applier, err := newProductionPluginRuntimeFullSetApplier(
				config.Manager, config.Store,
			)
			if err != nil {
				return nil, fmt.Errorf("construct exact plugin runtime full-set applier: %w", err)
			}
			notifications := extensions.NewPostgresPluginRuntimeNotifications(
				config.Store,
				extensions.DefaultPluginRuntimeReconnectDelay,
				func(err error) { report("plugin runtime LISTEN degraded; durable polling remains active", err) },
			)
			return extensions.NewPluginRuntimeCoordinator(
				config.Store,
				applier,
				notifications,
				extensions.PluginRuntimeCoordinatorConfig{
					Identity:          identity,
					LeaseDuration:     config.LeaseDuration,
					HeartbeatInterval: config.HeartbeatInterval,
					PollInterval:      config.PollInterval,
					OnReady:           onReady,
					OnError: func(err error) {
						report("plugin runtime convergence degraded", err)
						onError(err)
					},
				},
			)
		},
	})
}

func launchPluginRuntimeCoordinator(
	ctx context.Context,
	config pluginRuntimeCoordinatorLaunchConfig,
) (*pluginRuntimeCoordinatorRuntime, error) {
	if ctx == nil {
		return nil, errPluginRuntimeCoordinatorBootstrapInvalid
	}
	if config.SafeMode {
		return newInactivePluginRuntimeCoordinatorRuntime(), nil
	}
	if config.Ensurer == nil || config.Build == nil ||
		strings.TrimSpace(config.Identity.NodeID) == "" ||
		strings.TrimSpace(config.Identity.BootID) == "" ||
		(config.Identity.ProcessRole != extensions.PluginRuntimeProcessAPI &&
			config.Identity.ProcessRole != extensions.PluginRuntimeProcessWorker) {
		return nil, errPluginRuntimeCoordinatorBootstrapInvalid
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	publication, err := config.Ensurer.EnsureInitialPluginRuntimePublication(ctx)
	if err != nil {
		return nil, fmt.Errorf("ensure initial plugin runtime publication: %w", err)
	}
	if !validPluginRuntimeCoordinatorGenesis(publication) {
		return nil, fmt.Errorf("%w: genesis publication is not exact", errPluginRuntimeCoordinatorBootstrapInvalid)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	ready := make(chan struct{})
	startupErrors := make(chan error, 1)
	var readyOnce sync.Once
	runtime := &pluginRuntimeCoordinatorRuntime{
		identity:    config.Identity,
		active:      true,
		stopTimeout: normalizedPluginRuntimeCoordinatorStopTimeout(config.StopTimeout),
		done:        make(chan struct{}),
		failures:    make(chan error, 1),
	}
	runner, err := config.Build(
		config.Identity,
		func() {
			if runtime.stopping.Load() {
				return
			}
			readyOnce.Do(func() {
				runtime.ready.Store(true)
				close(ready)
			})
		},
		func(err error) {
			if err == nil || runtime.ready.Load() || runtime.stopping.Load() {
				return
			}
			select {
			case startupErrors <- err:
			default:
			}
		},
	)
	if err != nil {
		return nil, fmt.Errorf("construct plugin runtime coordinator: %w", err)
	}
	if runner == nil {
		return nil, errPluginRuntimeCoordinatorBootstrapInvalid
	}

	runCtx, cancel := context.WithCancel(context.Background())
	runtime.cancel = cancel
	go runtime.run(runCtx, runner)

	abort := func(cause error) (*pluginRuntimeCoordinatorRuntime, error) {
		stopCtx, stopCancel := context.WithTimeout(
			context.Background(), runtime.stopTimeout,
		)
		defer stopCancel()
		if stopErr := runtime.Stop(stopCtx); stopErr != nil {
			return nil, errors.Join(cause, fmt.Errorf("stop plugin runtime coordinator after startup failure: %w", stopErr))
		}
		return nil, cause
	}

	select {
	case <-ready:
		// 首轮失败可能已入队，而 coordinator 随即重试成功；启动边界仍应
		// 返回第一次错误，让进程重新走一遍干净的 genesis/boot lease。
		select {
		case startupErr := <-startupErrors:
			return abort(fmt.Errorf("initial plugin runtime convergence: %w", startupErr))
		default:
		}
		if err := ctx.Err(); err != nil {
			return abort(err)
		}
		// Readiness and terminal failure may race. Never return a handle that has
		// already lost its lease or stopped before the caller can install a monitor.
		select {
		case <-runtime.done:
			if runErr := runtime.Err(); runErr != nil {
				return nil, runErr
			}
			return nil, errPluginRuntimeCoordinatorStoppedBeforeReady
		default:
			return runtime, nil
		}
	case startupErr := <-startupErrors:
		return abort(fmt.Errorf("initial plugin runtime convergence: %w", startupErr))
	case <-runtime.done:
		if runErr := runtime.Err(); runErr != nil {
			return nil, runErr
		}
		return nil, errPluginRuntimeCoordinatorStoppedBeforeReady
	case <-ctx.Done():
		return abort(ctx.Err())
	}
}

func (runtime *pluginRuntimeCoordinatorRuntime) run(
	ctx context.Context,
	runner pluginRuntimeCoordinatorRunner,
) {
	err := runner.Run(ctx)
	if err == nil && !runtime.stopping.Load() {
		err = errPluginRuntimeCoordinatorStoppedBeforeReady
	}
	runtime.errMu.Lock()
	runtime.err = err
	runtime.errMu.Unlock()
	if err != nil && !runtime.stopping.Load() {
		runtime.failures <- err
	}
	close(runtime.failures)
	close(runtime.done)
}

func (runtime *pluginRuntimeCoordinatorRuntime) Stop(ctx context.Context) error {
	if runtime == nil || !runtime.active {
		return nil
	}
	if ctx == nil {
		return errPluginRuntimeCoordinatorBootstrapInvalid
	}
	runtime.stopOnce.Do(func() {
		runtime.stopping.Store(true)
		if runtime.cancel != nil {
			runtime.cancel()
		}
	})
	waitCtx, cancel := context.WithTimeout(ctx, runtime.stopTimeout)
	defer cancel()
	select {
	case <-runtime.done:
		return nil
	case <-waitCtx.Done():
		return waitCtx.Err()
	}
}

func (runtime *pluginRuntimeCoordinatorRuntime) Failures() <-chan error {
	if runtime == nil {
		return nil
	}
	return runtime.failures
}

func (runtime *pluginRuntimeCoordinatorRuntime) Done() <-chan struct{} {
	if runtime == nil {
		return nil
	}
	return runtime.done
}

func (runtime *pluginRuntimeCoordinatorRuntime) Err() error {
	if runtime == nil {
		return nil
	}
	runtime.errMu.RLock()
	defer runtime.errMu.RUnlock()
	return runtime.err
}

func (runtime *pluginRuntimeCoordinatorRuntime) Identity() extensions.PluginRuntimeNodeIdentity {
	if runtime == nil {
		return extensions.PluginRuntimeNodeIdentity{}
	}
	return runtime.identity
}

func (runtime *pluginRuntimeCoordinatorRuntime) Active() bool {
	return runtime != nil && runtime.active
}

func newInactivePluginRuntimeCoordinatorRuntime() *pluginRuntimeCoordinatorRuntime {
	done := make(chan struct{})
	failures := make(chan error)
	close(done)
	close(failures)
	return &pluginRuntimeCoordinatorRuntime{done: done, failures: failures}
}

func newPluginRuntimeCoordinatorIdentity(
	hostname string,
	role extensions.PluginRuntimeProcessRole,
	bootID string,
) (extensions.PluginRuntimeNodeIdentity, error) {
	nodeID := strings.TrimSpace(hostname)
	bootID = strings.TrimSpace(bootID)
	if nodeID == "" || bootID == "" || len([]byte(bootID)) > 128 ||
		(role != extensions.PluginRuntimeProcessAPI && role != extensions.PluginRuntimeProcessWorker) {
		return extensions.PluginRuntimeNodeIdentity{}, errPluginRuntimeCoordinatorBootstrapInvalid
	}
	if len([]byte(nodeID)) > 128 {
		digest := sha256.Sum256([]byte(nodeID))
		nodeID = "host-" + hex.EncodeToString(digest[:])
	}
	return extensions.PluginRuntimeNodeIdentity{
		NodeID: nodeID, ProcessRole: role, BootID: bootID,
	}, nil
}

func normalizedPluginRuntimeCoordinatorStopTimeout(value time.Duration) time.Duration {
	if value <= 0 {
		return defaultPluginRuntimeCoordinatorStopTimeout
	}
	return value
}

func validPluginRuntimeCoordinatorGenesis(publication extensions.PluginRuntimePublication) bool {
	if publication.Revision <= 0 || publication.MemberCount != len(publication.Members) ||
		publication.CreatedAt.IsZero() || publication.ActorUserID < 0 {
		return false
	}
	digest, err := extensions.PluginRuntimeMembersDigest(publication.Members)
	return err == nil && digest == publication.MembersDigest
}

// newProductionPluginRuntimeFullSetApplier 是 API 与 worker 共用的 production
// full-set 适配器构造点。initial-bootstrap Protocol V1 兼容窗口只在此开启。
func newProductionPluginRuntimeFullSetApplier(
	manager *extensionsruntime.Manager,
	inventory extensionsruntime.PluginRuntimeFullSetInventory,
) (*extensionsruntime.ManagerPluginRuntimeFullSetApplier, error) {
	return extensionsruntime.NewInitialBootstrapManagerPluginRuntimeFullSetApplier(manager, inventory)
}
