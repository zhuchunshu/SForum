package idempotency

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestRequiredReplayAcquiresConflictsAndReplaysDetachedResponse(t *testing.T) {
	backend := NewMemoryBackend()
	store := newRequiredReplayTestStore(t, backend)
	scope := requiredReplayTestScope("actor:42:bearer")
	fingerprint := strings.Repeat("a", 64)
	lease, replay, err := store.BeginRequiredReplayBound(
		t.Context(), scope, "request-42", requiredReplayV3TestBinding(fingerprint),
	)
	if err != nil || replay != nil || lease.storageKey == "" {
		t.Fatalf("first begin: lease=%#v replay=%#v err=%v", lease, replay, err)
	}
	if _, _, err := store.BeginRequiredReplayBound(
		t.Context(), scope, "request-42", requiredReplayV3TestBinding(fingerprint),
	); !errors.Is(err, ErrRequiredReplayInProgress) {
		t.Fatalf("in-flight error = %v", err)
	}
	if _, _, err := store.BeginRequiredReplayBound(
		t.Context(), scope, "request-42", requiredReplayV3TestBinding(strings.Repeat("b", 64)),
	); !errors.Is(err, ErrRequiredReplayFingerprintConflict) {
		t.Fatalf("fingerprint error = %v", err)
	}
	want := RequiredReplayResponse{
		Status: http.StatusCreated, Headers: http.Header{"Content-Type": {"application/json"}}, Body: []byte(`{"id":42}`),
		ResponseContractKnown: true,
		ResponseContract: &RequiredReplayResponseContract{
			StepIndex: 2, InvocationStage: "handler", RouteID: "demo.plugin.create",
			ContractVersion: "demo.plugin.create@1", ResponseSchema: "demo.plugin.create.response@1",
		},
	}
	if err := store.CompleteRequiredReplay(t.Context(), lease, want); err != nil {
		t.Fatal(err)
	}
	_, replay, err = store.BeginRequiredReplayBound(
		t.Context(), scope, "request-42", requiredReplayV3TestBinding(fingerprint),
	)
	if err != nil || replay == nil || replay.Status != want.Status || string(replay.Body) != string(want.Body) ||
		!replay.ResponseContractKnown || replay.ResponseContract == nil ||
		replay.ResponseContract.RouteID != want.ResponseContract.RouteID {
		t.Fatalf("replay=%#v err=%v", replay, err)
	}
	replay.Body[0] = '!'
	replay.Headers.Set("Content-Type", "mutated")
	replay.ResponseContract.RouteID = "mutated"
	_, detached, err := store.BeginRequiredReplayBound(
		t.Context(), scope, "request-42", requiredReplayV3TestBinding(fingerprint),
	)
	if err != nil || string(detached.Body) != string(want.Body) || detached.Headers.Get("Content-Type") != "application/json" ||
		detached.ResponseContract == nil || detached.ResponseContract.RouteID != want.ResponseContract.RouteID {
		t.Fatal("replay response escaped by reference")
	}
}

func TestRequiredReplayScopesDoNotCrossActorsCredentialsOrAnonymousClients(t *testing.T) {
	backend := NewMemoryBackend()
	store := newRequiredReplayTestStore(t, backend)
	fingerprint := strings.Repeat("c", 64)
	scopes := []RequiredReplayScope{
		requiredReplayTestScope("actor:42:cookie"),
		requiredReplayTestScope("actor:42:bearer"),
		requiredReplayTestScope("anonymous:" + strings.Repeat("1", 64)),
		requiredReplayTestScope("anonymous:" + strings.Repeat("2", 64)),
	}
	keys := make(map[string]bool, len(scopes))
	for _, scope := range scopes {
		lease, replay, err := store.BeginRequiredReplayBound(
			t.Context(), scope, "private-key", requiredReplayV3TestBinding(fingerprint),
		)
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
	store := newRequiredReplayTestStore(t, NewMemoryBackend())
	scope := requiredReplayTestScope("actor:7:cookie")
	fingerprint := strings.Repeat("d", 64)
	lease, _, err := store.BeginRequiredReplayBound(
		t.Context(), scope, "retry", requiredReplayV3TestBinding(fingerprint),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AbortRequiredReplay(t.Context(), lease); err != nil {
		t.Fatal(err)
	}
	replacement, _, err := store.BeginRequiredReplayBound(
		t.Context(), scope, "retry", requiredReplayV3TestBinding(fingerprint),
	)
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
	store := newRequiredReplayTestStore(t, NewMemoryBackend())
	scope := requiredReplayTestScope("actor:9:cookie")
	for _, key := range []string{"", "has space", strings.Repeat("x", MaxKeyLength+1)} {
		if _, _, err := store.BeginRequiredReplayBound(
			t.Context(), scope, key, requiredReplayV3TestBinding(strings.Repeat("e", 64)),
		); !errors.Is(err, ErrRequiredReplayInvalid) {
			t.Fatalf("key %q error = %v", key, err)
		}
	}
	lease, _, err := store.BeginRequiredReplayBound(
		t.Context(), scope, "large", requiredReplayV3TestBinding(strings.Repeat("f", 64)),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CompleteRequiredReplay(t.Context(), lease, RequiredReplayResponse{
		Status: http.StatusOK, Body: make([]byte, MaxRequiredReplayBody+1),
	}); !errors.Is(err, ErrRequiredReplayInvalid) {
		t.Fatalf("oversized response error = %v", err)
	}
	if _, _, err := store.BeginRequiredReplayBound(
		t.Context(), scope, "large", requiredReplayV3TestBinding(strings.Repeat("f", 64)),
	); !errors.Is(err, ErrRequiredReplayInProgress) {
		t.Fatalf("oversized completion released lease: %v", err)
	}
}

func TestRequiredReplayRejectsInvalidResponseContractEvidence(t *testing.T) {
	valid := func() RequiredReplayResponse {
		return RequiredReplayResponse{
			Status: http.StatusOK, ResponseContractKnown: true,
			ResponseContract: &RequiredReplayResponseContract{
				StepIndex: 1, InvocationStage: "response", RouteID: "demo.route.after",
				ContractVersion: "demo.route.after@1", ResponseSchema: "demo.route.after.response@1",
			},
		}
	}
	if err := validateRequiredReplayResponse(valid()); err != nil {
		t.Fatalf("valid response contract rejected: %v", err)
	}
	explicitNone := valid()
	explicitNone.ResponseContract = nil
	if err := validateRequiredReplayResponse(explicitNone); err != nil {
		t.Fatalf("explicit no-contract evidence rejected: %v", err)
	}

	for _, test := range []struct {
		name   string
		mutate func(*RequiredReplayResponse)
	}{
		{name: "unknown with evidence", mutate: func(value *RequiredReplayResponse) { value.ResponseContractKnown = false }},
		{name: "negative index", mutate: func(value *RequiredReplayResponse) { value.ResponseContract.StepIndex = -1 }},
		{name: "invalid stage", mutate: func(value *RequiredReplayResponse) { value.ResponseContract.InvocationStage = "request" }},
		{name: "empty route", mutate: func(value *RequiredReplayResponse) { value.ResponseContract.RouteID = "" }},
		{name: "padded schema", mutate: func(value *RequiredReplayResponse) { value.ResponseContract.ResponseSchema += " " }},
	} {
		t.Run(test.name, func(t *testing.T) {
			value := valid()
			test.mutate(&value)
			if err := validateRequiredReplayResponse(value); !errors.Is(err, ErrRequiredReplayInvalid) {
				t.Fatalf("error=%v response=%#v", err, value)
			}
		})
	}
}

func TestRequiredReplayFailsClosedWithoutRedis(t *testing.T) {
	store := NewStore(NewRedisBackend(nil), DefaultTTL)
	_, _, err := store.BeginRequiredReplayBound(
		t.Context(), requiredReplayTestScope("actor:11:bearer"), "required",
		requiredReplayV3TestBinding(strings.Repeat("a", 64)),
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

func newRequiredReplayTestStore(t *testing.T, backend Backend) *Store {
	t.Helper()
	return NewStore(backend, DefaultTTL).WithRequiredReplayCipher(requiredReplayV2TestCipher(t, "0c"))
}
