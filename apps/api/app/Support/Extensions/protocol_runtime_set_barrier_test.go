package extensionsruntime

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestProtocolRuntimeSetTransitionWaitHonorsCallerContext(t *testing.T) {
	starter := NewProtocolStarter(ProtocolStarterConfig{})
	release, err := starter.lockRuntimeSetTransition(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, lease, err := starter.PublishInstanceSet(ctx, nil); !errors.Is(err, context.DeadlineExceeded) || lease != nil {
		t.Fatalf("queued complete-set publish = lease:%v err:%v", lease != nil, err)
	}
	release()

	reacquired, err := starter.lockRuntimeSetTransition(context.Background())
	if err != nil {
		t.Fatalf("reacquire set transition: %v", err)
	}
	reacquired()
}

func TestProtocolExtensionLifecycleWaitHonorsCallerContext(t *testing.T) {
	starter := NewProtocolStarter(ProtocolStarterConfig{})
	release, err := starter.lockExtensionLifecycleContext(context.Background(), "queued.plugin")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := starter.lockExtensionLifecycleContext(ctx, "queued.plugin"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("queued lifecycle wait error = %v", err)
	}
	release()

	reacquired, err := starter.lockExtensionLifecycleContext(context.Background(), "queued.plugin")
	if err != nil {
		t.Fatalf("reacquire lifecycle: %v", err)
	}
	reacquired()
}

func TestProtocolRuntimeSetLeaseFinalValidationIsFencedAndCancellable(t *testing.T) {
	want := []RuntimeInstanceIdentity{{ExtensionID: "lease.plugin", InstanceID: "runtime-1"}}
	called := 0
	lease := &ProtocolRuntimeSetLease{
		release: func() {},
		validate: func(ctx context.Context, identities []RuntimeInstanceIdentity) error {
			called++
			if err := ctx.Err(); err != nil {
				return err
			}
			if len(identities) != 1 || identities[0] != want[0] {
				return ErrProtocolInstanceNotReady
			}
			return nil
		},
	}
	if err := lease.Validate(context.Background(), want); err != nil || called != 1 {
		t.Fatalf("validate live lease error=%v called=%d", err, called)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := lease.Validate(canceled, want); !errors.Is(err, context.Canceled) || called != 2 {
		t.Fatalf("validate canceled lease error=%v called=%d", err, called)
	}
	lease.Release()
	if err := lease.Validate(context.Background(), want); !errors.Is(err, ErrProtocolInstanceNotReady) || called != 2 {
		t.Fatalf("validate released lease error=%v called=%d", err, called)
	}
}
