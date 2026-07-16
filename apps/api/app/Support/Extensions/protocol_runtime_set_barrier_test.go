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
