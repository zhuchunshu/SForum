package idempotency

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestRequiredReplayAcquiresConflictsAndReplaysDetachedResponse(t *testing.T) {
	backend := NewMemoryBackend()
	store := NewStore(backend, DefaultTTL)
	scope := requiredReplayTestScope("actor:42:bearer")
	fingerprint := strings.Repeat("a", 64)
	lease, replay, err := store.BeginRequiredReplay(t.Context(), scope, "request-42", fingerprint)
	if err != nil || replay != nil || lease.storageKey == "" {
		t.Fatalf("first begin: lease=%#v replay=%#v err=%v", lease, replay, err)
	}
	if _, _, err := store.BeginRequiredReplay(t.Context(), scope, "request-42", fingerprint); !errors.Is(err, ErrRequiredReplayInProgress) {
		t.Fatalf("in-flight error = %v", err)
	}
	if _, _, err := store.BeginRequiredReplay(t.Context(), scope, "request-42", strings.Repeat("b", 64)); !errors.Is(err, ErrRequiredReplayFingerprintConflict) {
		t.Fatalf("fingerprint error = %v", err)
	}
	want := RequiredReplayResponse{
		Status: http.StatusCreated, Headers: http.Header{"Content-Type": {"application/json"}}, Body: []byte(`{"id":42}`),
	}
	if err := store.CompleteRequiredReplay(t.Context(), lease, want); err != nil {
		t.Fatal(err)
	}
	_, replay, err = store.BeginRequiredReplay(t.Context(), scope, "request-42", fingerprint)
	if err != nil || replay == nil || replay.Status != want.Status || string(replay.Body) != string(want.Body) {
		t.Fatalf("replay=%#v err=%v", replay, err)
	}
	replay.Body[0] = '!'
	replay.Headers.Set("Content-Type", "mutated")
	_, detached, err := store.BeginRequiredReplay(t.Context(), scope, "request-42", fingerprint)
	if err != nil || string(detached.Body) != string(want.Body) || detached.Headers.Get("Content-Type") != "application/json" {
		t.Fatal("replay response escaped by reference")
	}
}

func TestRequiredReplayScopesDoNotCrossActorsCredentialsOrAnonymousClients(t *testing.T) {
	backend := NewMemoryBackend()
	store := NewStore(backend, DefaultTTL)
	fingerprint := strings.Repeat("c", 64)
	scopes := []RequiredReplayScope{
		requiredReplayTestScope("actor:42:cookie"),
		requiredReplayTestScope("actor:42:bearer"),
		requiredReplayTestScope("anonymous:" + strings.Repeat("1", 64)),
		requiredReplayTestScope("anonymous:" + strings.Repeat("2", 64)),
	}
	keys := make(map[string]bool, len(scopes))
	for _, scope := range scopes {
		lease, replay, err := store.BeginRequiredReplay(t.Context(), scope, "private-key", fingerprint)
		if err != nil || replay != nil || keys[lease.storageKey] {
			t.Fatalf("scope %q: lease=%#v replay=%#v err=%v", scope.ActorScope, lease, replay, err)
		}
		keys[lease.storageKey] = true
	}
	for storageKey, entry := range backend.items {
		if strings.Contains(storageKey, "private-key") || strings.Contains(string(entry.value), "private-key") ||
			strings.Contains(string(entry.value), "actor:42") || strings.Contains(string(entry.value), "anonymous:") {
			t.Fatalf("replay storage exposed scope or key: %s %s", storageKey, entry.value)
		}
	}
}

func TestRequiredReplayAbortAndCompletionAreLeaseFenced(t *testing.T) {
	store := NewStore(NewMemoryBackend(), DefaultTTL)
	scope := requiredReplayTestScope("actor:7:cookie")
	fingerprint := strings.Repeat("d", 64)
	lease, _, err := store.BeginRequiredReplay(t.Context(), scope, "retry", fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AbortRequiredReplay(t.Context(), lease); err != nil {
		t.Fatal(err)
	}
	replacement, _, err := store.BeginRequiredReplay(t.Context(), scope, "retry", fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CompleteRequiredReplay(t.Context(), lease, RequiredReplayResponse{Status: http.StatusOK}); !errors.Is(err, ErrRequiredReplayLeaseLost) {
		t.Fatalf("stale completion error = %v", err)
	}
	if err := store.CompleteRequiredReplay(t.Context(), replacement, RequiredReplayResponse{Status: http.StatusOK}); err != nil {
		t.Fatal(err)
	}
}

func TestRequiredReplayRejectsInvalidKeysAndOversizedResponses(t *testing.T) {
	store := NewStore(NewMemoryBackend(), DefaultTTL)
	scope := requiredReplayTestScope("actor:9:cookie")
	for _, key := range []string{"", "has space", strings.Repeat("x", MaxKeyLength+1)} {
		if _, _, err := store.BeginRequiredReplay(t.Context(), scope, key, strings.Repeat("e", 64)); !errors.Is(err, ErrRequiredReplayInvalid) {
			t.Fatalf("key %q error = %v", key, err)
		}
	}
	lease, _, err := store.BeginRequiredReplay(t.Context(), scope, "large", strings.Repeat("f", 64))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CompleteRequiredReplay(t.Context(), lease, RequiredReplayResponse{
		Status: http.StatusOK, Body: make([]byte, MaxRequiredReplayBody+1),
	}); !errors.Is(err, ErrRequiredReplayInvalid) {
		t.Fatalf("oversized response error = %v", err)
	}
	if _, _, err := store.BeginRequiredReplay(t.Context(), scope, "large", strings.Repeat("f", 64)); !errors.Is(err, ErrRequiredReplayInProgress) {
		t.Fatalf("oversized completion released lease: %v", err)
	}
}

func TestRequiredReplayFailsClosedWithoutRedis(t *testing.T) {
	store := NewStore(NewRedisBackend(nil), DefaultTTL)
	_, _, err := store.BeginRequiredReplay(
		t.Context(), requiredReplayTestScope("actor:11:bearer"), "required", strings.Repeat("a", 64),
	)
	if !errors.Is(err, ErrRequiredReplayUnavailable) {
		t.Fatalf("unavailable Redis error = %v", err)
	}
}

func requiredReplayTestScope(actorScope string) RequiredReplayScope {
	return RequiredReplayScope{
		ActorScope: actorScope, ExtensionID: "demo.plugin", ExtensionVersion: "1.0.0",
		PackageDigest: strings.Repeat("a", 64), RouteID: "demo.plugin.create",
		ContractVersion: "demo.plugin.create@1", Method: http.MethodPost,
	}
}
