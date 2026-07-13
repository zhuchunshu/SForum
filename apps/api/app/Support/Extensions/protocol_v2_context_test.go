package extensionsruntime

import (
	"context"
	"testing"
	"time"
)

func TestProtocolV2DeadlineCapsLaterParentDeadline(t *testing.T) {
	parent, cancelParent := context.WithTimeout(context.Background(), time.Minute)
	defer cancelParent()
	started := time.Now()
	ctx, cancel := protocolV2Deadline(parent, 50*time.Millisecond)
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("protocol v2 context has no deadline")
	}
	if deadline.After(started.Add(200 * time.Millisecond)) {
		t.Fatalf("protocol v2 timeout did not cap parent deadline: %s", deadline.Sub(started))
	}
}
