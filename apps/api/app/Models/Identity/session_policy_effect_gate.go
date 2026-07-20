package identity

import (
	"context"
	"sync"
	"sync/atomic"
)

// identitySessionPolicyEffectGate is a context-aware, writer-preferred RW gate.
// Waiting lifecycle/selection writers stop new effects from entering, while an
// already accepted effect keeps its admission until the Host callback returns.
type identitySessionPolicyEffectGate struct {
	mu             sync.Mutex
	changed        chan struct{}
	readers        int
	writer         bool
	waitingWriters int
}

func (g *identitySessionPolicyEffectGate) lockRead(ctx context.Context) error {
	for {
		g.mu.Lock()
		if err := ctx.Err(); err != nil {
			g.mu.Unlock()
			return err
		}
		if !g.writer && g.waitingWriters == 0 {
			g.readers++
			g.mu.Unlock()
			return nil
		}
		changed := g.changedLocked()
		g.mu.Unlock()
		select {
		case <-changed:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (g *identitySessionPolicyEffectGate) unlockRead() {
	g.mu.Lock()
	if g.readers <= 0 {
		g.mu.Unlock()
		panic("identity: unlock of unlocked session policy effect read gate")
	}
	g.readers--
	if g.readers == 0 {
		g.notifyLocked()
	}
	g.mu.Unlock()
}

func (g *identitySessionPolicyEffectGate) lockWrite(ctx context.Context) error {
	g.mu.Lock()
	g.waitingWriters++
	for {
		if err := ctx.Err(); err != nil {
			g.waitingWriters--
			g.notifyLocked()
			g.mu.Unlock()
			return err
		}
		if !g.writer && g.readers == 0 {
			g.waitingWriters--
			g.writer = true
			g.mu.Unlock()
			return nil
		}
		changed := g.changedLocked()
		g.mu.Unlock()
		select {
		case <-changed:
		case <-ctx.Done():
		}
		g.mu.Lock()
	}
}

func (g *identitySessionPolicyEffectGate) unlockWrite() {
	g.mu.Lock()
	if !g.writer {
		g.mu.Unlock()
		panic("identity: unlock of unlocked session policy effect write gate")
	}
	g.writer = false
	g.notifyLocked()
	g.mu.Unlock()
}

func (g *identitySessionPolicyEffectGate) tryLockRead() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.writer || g.waitingWriters > 0 {
		return false
	}
	g.readers++
	return true
}

func (g *identitySessionPolicyEffectGate) tryLockWrite() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.writer || g.readers > 0 || g.waitingWriters > 0 {
		return false
	}
	g.writer = true
	return true
}

func (g *identitySessionPolicyEffectGate) changedLocked() <-chan struct{} {
	if g.changed == nil {
		g.changed = make(chan struct{})
	}
	return g.changed
}

func (g *identitySessionPolicyEffectGate) notifyLocked() {
	if g.changed != nil {
		close(g.changed)
		g.changed = nil
	}
}

// TryRLock/RUnlock and TryLock/Unlock keep deterministic package tests concise;
// production callers use the context-aware methods above.
func (g *identitySessionPolicyEffectGate) TryRLock() bool { return g.tryLockRead() }
func (g *identitySessionPolicyEffectGate) RUnlock()       { g.unlockRead() }
func (g *identitySessionPolicyEffectGate) TryLock() bool  { return g.tryLockWrite() }
func (g *identitySessionPolicyEffectGate) Unlock()        { g.unlockWrite() }

type identitySessionPolicyEffectContextKey struct{}

type identitySessionPolicyEffectContext struct {
	owner  *PostgresIdentitySessionPolicyStore
	active atomic.Bool
}

func beginSessionPolicyEffectContext(
	ctx context.Context,
	owner *PostgresIdentitySessionPolicyStore,
) (context.Context, func()) {
	marker := &identitySessionPolicyEffectContext{owner: owner}
	marker.active.Store(true)
	return context.WithValue(ctx, identitySessionPolicyEffectContextKey{}, marker), func() {
		marker.active.Store(false)
	}
}

func isSessionPolicyEffectContext(ctx context.Context, owner *PostgresIdentitySessionPolicyStore) bool {
	marker, _ := ctx.Value(identitySessionPolicyEffectContextKey{}).(*identitySessionPolicyEffectContext)
	return marker != nil && marker.owner == owner && marker.active.Load()
}
