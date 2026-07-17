package idempotency

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestRequiredReplayCipherV2BindsAADAndUsesUniqueNonces(t *testing.T) {
	cipherA := requiredReplayV2TestCipher(t, "01")
	cipherB := requiredReplayV2TestCipher(t, "02")
	storageKey := "plugin-route:required.24h@1:storage-a"
	fingerprint := strings.Repeat("a", 64)
	planDigest := strings.Repeat("b", 64)
	plaintext := []byte(`{"requestMutations":[{"value":"private-patch-value"}]}`)

	first, err := cipherA.Encrypt(storageKey, fingerprint, planDigest, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	second, err := cipherA.Encrypt(storageKey, fingerprint, planDigest, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("AES-GCM encryption reused a nonce")
	}
	firstRaw, err := base64.RawURLEncoding.DecodeString(first)
	if err != nil {
		t.Fatal(err)
	}
	secondRaw, err := base64.RawURLEncoding.DecodeString(second)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(firstRaw[:cipherA.aead.NonceSize()], secondRaw[:cipherA.aead.NonceSize()]) {
		t.Fatal("AES-GCM ciphertexts contain the same nonce")
	}
	if bytes.Contains(firstRaw, plaintext) || strings.Contains(first, "private-patch-value") {
		t.Fatal("ciphertext contains the mutation transcript plaintext")
	}
	decrypted, err := cipherA.Decrypt(storageKey, fingerprint, planDigest, first)
	if err != nil || !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("round trip = %q, %v", decrypted, err)
	}

	tamperedRaw := append([]byte(nil), firstRaw...)
	tamperedRaw[len(tamperedRaw)-1] ^= 0x01
	tampered := base64.RawURLEncoding.EncodeToString(tamperedRaw)
	checks := []struct {
		name        string
		cipher      *RequiredReplayCipher
		storageKey  string
		fingerprint string
		planDigest  string
		ciphertext  string
	}{
		{name: "wrong key", cipher: cipherB, storageKey: storageKey, fingerprint: fingerprint, planDigest: planDigest, ciphertext: first},
		{name: "storage key AAD", cipher: cipherA, storageKey: storageKey + "-other", fingerprint: fingerprint, planDigest: planDigest, ciphertext: first},
		{name: "fingerprint AAD", cipher: cipherA, storageKey: storageKey, fingerprint: strings.Repeat("c", 64), planDigest: planDigest, ciphertext: first},
		{name: "plan digest AAD", cipher: cipherA, storageKey: storageKey, fingerprint: fingerprint, planDigest: strings.Repeat("d", 64), ciphertext: first},
		{name: "tampered ciphertext", cipher: cipherA, storageKey: storageKey, fingerprint: fingerprint, planDigest: planDigest, ciphertext: tampered},
		{name: "malformed base64", cipher: cipherA, storageKey: storageKey, fingerprint: fingerprint, planDigest: planDigest, ciphertext: "not!base64"},
		{name: "truncated ciphertext", cipher: cipherA, storageKey: storageKey, fingerprint: fingerprint, planDigest: planDigest, ciphertext: base64.RawURLEncoding.EncodeToString(firstRaw[:cipherA.aead.NonceSize()])},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			if _, err := check.cipher.Decrypt(
				check.storageKey, check.fingerprint, check.planDigest, check.ciphertext,
			); !errors.Is(err, ErrRequiredReplayCipherInvalid) {
				t.Fatalf("decrypt error = %v", err)
			}
		})
	}
}

func TestRequiredReplayCipherV2RejectsMissingAndInvalidKeys(t *testing.T) {
	disabled, err := NewRequiredReplayCipher("")
	if err != nil || disabled.Enabled() {
		t.Fatalf("disabled cipher = %#v, %v", disabled, err)
	}
	var nilCipher *RequiredReplayCipher
	for name, cipher := range map[string]*RequiredReplayCipher{"nil": nilCipher, "disabled": disabled} {
		t.Run(name, func(t *testing.T) {
			if _, err := cipher.Encrypt(
				"storage", strings.Repeat("a", 64), strings.Repeat("b", 64), []byte("secret"),
			); !errors.Is(err, ErrRequiredReplayCipherInvalid) {
				t.Fatalf("encrypt error = %v", err)
			}
			if _, err := cipher.Decrypt(
				"storage", strings.Repeat("a", 64), strings.Repeat("b", 64), "ciphertext",
			); !errors.Is(err, ErrRequiredReplayCipherInvalid) {
				t.Fatalf("decrypt error = %v", err)
			}
		})
	}
	for _, key := range []string{"not-hex", strings.Repeat("01", 31), strings.Repeat("01", 33)} {
		if cipher, err := NewRequiredReplayCipher(key); cipher != nil || !errors.Is(err, ErrRequiredReplayCipherInvalid) {
			t.Fatalf("key %q: cipher=%#v error=%v", key, cipher, err)
		}
	}
}

func TestRequiredReplayV2EncryptsMutationTranscriptAndDetachesReplay(t *testing.T) {
	backend := NewMemoryBackend()
	store := NewStore(backend, DefaultTTL).WithRequiredReplayCipher(requiredReplayV2TestCipher(t, "03"))
	scope := requiredReplayTestScope("actor:72:bearer")
	fingerprint := strings.Repeat("a", 64)
	secret := "mutation-secret-must-not-reach-redis"
	lease, replay, err := store.BeginRequiredReplay(t.Context(), scope, "encrypted-transcript", fingerprint)
	if err != nil || replay != nil {
		t.Fatalf("begin: lease=%#v replay=%#v error=%v", lease, replay, err)
	}
	authorization := requiredReplayV2TestAuthorization(secret)
	response := RequiredReplayResponse{
		Status:        http.StatusCreated,
		Headers:       http.Header{"Content-Type": {"application/json"}},
		Body:          []byte(`{"created":true}`),
		CanonicalPath: "/created/72",
		Authorization: authorization,
	}
	if err := store.CompleteRequiredReplay(t.Context(), lease, response); err != nil {
		t.Fatal(err)
	}

	raw := requiredReplayV2StoredBytes(t, store, backend, lease.storageKey)
	for _, forbidden := range []string{secret, "requestMutations", "beforeDigest", "afterDigest", "sforum.route-replay-authorization@1"} {
		if bytes.Contains(raw, []byte(forbidden)) {
			t.Fatalf("raw replay record contains transcript plaintext %q: %s", forbidden, raw)
		}
	}
	if !bytes.Contains(raw, []byte(`"authorizationCiphertext"`)) {
		t.Fatalf("raw replay record has no encrypted transcript: %s", raw)
	}
	var record requiredReplayRecord
	if err := json.Unmarshal(raw, &record); err != nil || record.Schema != requiredReplaySchemaV2 ||
		record.AuthorizationCiphertext == "" || record.PlanDigest != authorization.PlanDigest ||
		record.Response == nil || record.Response.Authorization != nil {
		t.Fatalf("stored record = %#v, %v", record, err)
	}

	// Mutating caller-owned values after completion must not alter the stored record.
	response.Body[0] = '!'
	authorization.RequestMutations[0].Operations[0].Value[1] = 'X'
	_, first, err := store.BeginRequiredReplay(t.Context(), scope, "encrypted-transcript", fingerprint)
	if err != nil || first == nil || first.Authorization == nil || string(first.Body) != `{"created":true}` ||
		string(first.Authorization.RequestMutations[0].Operations[0].Value) != `"`+secret+`"` {
		t.Fatalf("first replay = %#v, %v", first, err)
	}
	first.Body[0] = '?'
	first.Headers.Set("Content-Type", "mutated")
	first.Authorization.RequestMutations[0].Operations[0].Value[1] = 'Y'
	_, detached, err := store.BeginRequiredReplay(t.Context(), scope, "encrypted-transcript", fingerprint)
	if err != nil || detached == nil || detached.Authorization == nil || string(detached.Body) != `{"created":true}` ||
		detached.Headers.Get("Content-Type") != "application/json" ||
		string(detached.Authorization.RequestMutations[0].Operations[0].Value) != `"`+secret+`"` {
		t.Fatalf("detached replay = %#v, %v", detached, err)
	}
}

func TestRequiredReplayV2WrongKeyFailsClosedWithoutReplacingRecord(t *testing.T) {
	backend := NewMemoryBackend()
	right := NewStore(backend, DefaultTTL).WithRequiredReplayCipher(requiredReplayV2TestCipher(t, "04"))
	scope := requiredReplayTestScope("actor:73:cookie")
	fingerprint := strings.Repeat("b", 64)
	lease, _, err := right.BeginRequiredReplay(t.Context(), scope, "rotation", fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	if err := right.CompleteRequiredReplay(t.Context(), lease, RequiredReplayResponse{
		Status: http.StatusOK, Authorization: requiredReplayV2TestAuthorization("rotation-secret"),
	}); err != nil {
		t.Fatal(err)
	}
	fullKey := right.fullKey(lease.storageKey)
	before := append([]byte(nil), backend.items[fullKey].value...)
	beforeExpiry := backend.items[fullKey].expiresAt

	wrong := NewStore(backend, DefaultTTL).WithRequiredReplayCipher(requiredReplayV2TestCipher(t, "05"))
	if _, replay, err := wrong.BeginRequiredReplay(
		t.Context(), scope, "rotation", fingerprint,
	); replay != nil || !errors.Is(err, ErrRequiredReplayUnavailable) || !errors.Is(err, ErrRequiredReplayCipherInvalid) {
		t.Fatalf("wrong-key replay = %#v, %v", replay, err)
	}
	entry := backend.items[fullKey]
	if !bytes.Equal(entry.value, before) || !entry.expiresAt.Equal(beforeExpiry) {
		t.Fatal("wrong-key replay replaced or refreshed the durable record")
	}
	if _, replay, err := right.BeginRequiredReplay(
		t.Context(), scope, "rotation", fingerprint,
	); err != nil || replay == nil || replay.Authorization == nil {
		t.Fatalf("right-key replay after failed rotation = %#v, %v", replay, err)
	}
}

func TestRequiredReplayV1V2PendingAndCompletedCompatibility(t *testing.T) {
	scope := requiredReplayTestScope("actor:74:bearer")
	currentFingerprint := strings.Repeat("c", 64)
	legacyFingerprint := strings.Repeat("d", 64)

	t.Run("V1 pending remains in progress through the legacy fingerprint", func(t *testing.T) {
		backend, store := requiredReplayV2SeedRecord(t, scope, "v1-pending", requiredReplayRecord{
			Schema: requiredReplaySchemaV1, State: requiredReplayPending,
			Fingerprint: legacyFingerprint, LeaseToken: strings.Repeat("1", 32),
		})
		_ = backend
		if _, _, err := store.BeginRequiredReplay(
			t.Context(), scope, "v1-pending", currentFingerprint, legacyFingerprint,
		); !errors.Is(err, ErrRequiredReplayInProgress) {
			t.Fatalf("V1 pending error = %v", err)
		}
	})

	t.Run("V2 pending uses the current fingerprint", func(t *testing.T) {
		_, store := requiredReplayV2SeedRecord(t, scope, "v2-pending", requiredReplayRecord{
			Schema: requiredReplaySchemaV2, State: requiredReplayPending,
			Fingerprint: currentFingerprint, LeaseToken: strings.Repeat("2", 32),
		})
		if _, _, err := store.BeginRequiredReplay(
			t.Context(), scope, "v2-pending", currentFingerprint,
		); !errors.Is(err, ErrRequiredReplayInProgress) {
			t.Fatalf("V2 pending error = %v", err)
		}
	})

	t.Run("V1 completed response-only replay remains compatible", func(t *testing.T) {
		_, store := requiredReplayV2SeedRecord(t, scope, "v1-completed", requiredReplayRecord{
			Schema: requiredReplaySchemaV1, State: requiredReplayCompleted, Fingerprint: legacyFingerprint,
			Response: &RequiredReplayResponse{Status: http.StatusCreated, Body: []byte("legacy")},
		})
		_, replay, err := store.BeginRequiredReplay(
			t.Context(), scope, "v1-completed", currentFingerprint, legacyFingerprint,
		)
		if err != nil || replay == nil || string(replay.Body) != "legacy" || replay.Authorization != nil {
			t.Fatalf("V1 replay = %#v, %v", replay, err)
		}
	})

	t.Run("V2 completed response-only replay needs no cipher", func(t *testing.T) {
		_, store := requiredReplayV2SeedRecord(t, scope, "v2-completed", requiredReplayRecord{
			Schema: requiredReplaySchemaV2, State: requiredReplayCompleted, Fingerprint: currentFingerprint,
			Response: &RequiredReplayResponse{Status: http.StatusOK, Body: []byte("current")},
		})
		_, replay, err := store.BeginRequiredReplay(t.Context(), scope, "v2-completed", currentFingerprint)
		if err != nil || replay == nil || string(replay.Body) != "current" || replay.Authorization != nil {
			t.Fatalf("V2 replay = %#v, %v", replay, err)
		}
	})

	t.Run("V2 records cannot claim V1 fingerprint compatibility", func(t *testing.T) {
		_, store := requiredReplayV2SeedRecord(t, scope, "v2-legacy-fingerprint", requiredReplayRecord{
			Schema: requiredReplaySchemaV2, State: requiredReplayCompleted, Fingerprint: legacyFingerprint,
			Response: &RequiredReplayResponse{Status: http.StatusOK, Body: []byte("wrong-version")},
		})
		_, replay, err := store.BeginRequiredReplay(
			t.Context(), scope, "v2-legacy-fingerprint", currentFingerprint, legacyFingerprint,
		)
		if replay != nil || !errors.Is(err, ErrRequiredReplayFingerprintConflict) {
			t.Fatalf("V2 record used V1 compatibility: replay=%#v error=%v", replay, err)
		}
	})

	invalid := []struct {
		name   string
		record requiredReplayRecord
	}{
		{name: "V1 encrypted transcript", record: requiredReplayRecord{
			Schema: requiredReplaySchemaV1, State: requiredReplayCompleted, Fingerprint: legacyFingerprint,
			Response: &RequiredReplayResponse{Status: http.StatusOK}, PlanDigest: strings.Repeat("e", 64),
			AuthorizationCiphertext: "ciphertext",
		}},
		{name: "V2 plan without ciphertext", record: requiredReplayRecord{
			Schema: requiredReplaySchemaV2, State: requiredReplayCompleted, Fingerprint: currentFingerprint,
			Response: &RequiredReplayResponse{Status: http.StatusOK}, PlanDigest: strings.Repeat("e", 64),
		}},
		{name: "V2 ciphertext without plan", record: requiredReplayRecord{
			Schema: requiredReplaySchemaV2, State: requiredReplayCompleted, Fingerprint: currentFingerprint,
			Response: &RequiredReplayResponse{Status: http.StatusOK}, AuthorizationCiphertext: "ciphertext",
		}},
	}
	for _, test := range invalid {
		t.Run(test.name, func(t *testing.T) {
			raw, err := json.Marshal(test.record)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := decodeRequiredReplayRecord(raw); !errors.Is(err, ErrRequiredReplayUnavailable) {
				t.Fatalf("decode error = %v", err)
			}
		})
	}
}

func TestRequiredReplayV2RecordAndCanonicalPathCaps(t *testing.T) {
	record := requiredReplayRecord{
		Schema: requiredReplaySchemaV2, State: requiredReplayCompleted,
		Fingerprint: strings.Repeat("e", 64), Response: &RequiredReplayResponse{Status: http.StatusOK},
	}
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	exact := append(raw, bytes.Repeat([]byte(" "), MaxRequiredReplayRecord-len(raw))...)
	if _, err := decodeRequiredReplayRecord(exact); err != nil {
		t.Fatalf("exact record cap rejected: %v", err)
	}
	if _, err := decodeRequiredReplayRecord(append(exact, ' ')); !errors.Is(err, ErrRequiredReplayUnavailable) {
		t.Fatalf("record above cap error = %v", err)
	}

	exactPath := "/" + strings.Repeat("a", MaxRequiredReplayCanonicalPath-1)
	if err := validateRequiredReplayResponse(RequiredReplayResponse{
		Status: http.StatusOK, CanonicalPath: exactPath,
	}); err != nil {
		t.Fatalf("exact canonical path cap rejected: %v", err)
	}
	if err := validateRequiredReplayResponse(RequiredReplayResponse{
		Status: http.StatusOK, CanonicalPath: exactPath + "a",
	}); !errors.Is(err, ErrRequiredReplayInvalid) {
		t.Fatalf("canonical path above cap error = %v", err)
	}
}

func TestRequiredReplayV2CompletionFailuresPreservePending(t *testing.T) {
	scope := requiredReplayTestScope("actor:75:cookie")
	fingerprint := strings.Repeat("f", 64)

	t.Run("missing cipher fails before CAS", func(t *testing.T) {
		backend := &requiredReplayV2CASBackend{MemoryBackend: NewMemoryBackend()}
		store := NewStore(backend, DefaultTTL)
		lease, _, err := store.BeginRequiredReplay(t.Context(), scope, "missing-cipher", fingerprint)
		if err != nil {
			t.Fatal(err)
		}
		if err := store.CompleteRequiredReplay(t.Context(), lease, RequiredReplayResponse{
			Status: http.StatusOK, Authorization: requiredReplayV2TestAuthorization("missing-cipher-secret"),
		}); !errors.Is(err, ErrRequiredReplayUnavailable) || !errors.Is(err, ErrRequiredReplayCipherInvalid) {
			t.Fatalf("completion error = %v", err)
		}
		if backend.casCalls != 0 {
			t.Fatalf("missing cipher reached CAS %d times", backend.casCalls)
		}
		if _, _, err := store.BeginRequiredReplay(
			t.Context(), scope, "missing-cipher", fingerprint,
		); !errors.Is(err, ErrRequiredReplayInProgress) {
			t.Fatalf("missing-cipher completion released pending: %v", err)
		}
	})

	t.Run("CAS error preserves pending", func(t *testing.T) {
		backend := &requiredReplayV2CASBackend{MemoryBackend: NewMemoryBackend(), casErr: errors.New("redis unavailable")}
		store := NewStore(backend, DefaultTTL).WithRequiredReplayCipher(requiredReplayV2TestCipher(t, "06"))
		lease, _, err := store.BeginRequiredReplay(t.Context(), scope, "cas-error", fingerprint)
		if err != nil {
			t.Fatal(err)
		}
		if err := store.CompleteRequiredReplay(t.Context(), lease, RequiredReplayResponse{
			Status: http.StatusOK, Authorization: requiredReplayV2TestAuthorization("cas-error-secret"),
		}); !errors.Is(err, ErrRequiredReplayUnavailable) || !errors.Is(err, backend.casErr) {
			t.Fatalf("completion error = %v", err)
		}
		if backend.casCalls != 1 {
			t.Fatalf("CAS calls = %d", backend.casCalls)
		}
		if _, _, err := store.BeginRequiredReplay(
			t.Context(), scope, "cas-error", fingerprint,
		); !errors.Is(err, ErrRequiredReplayInProgress) {
			t.Fatalf("CAS error released pending: %v", err)
		}
	})

	t.Run("CAS loss preserves the replacement owner", func(t *testing.T) {
		backend := &requiredReplayV2CASBackend{MemoryBackend: NewMemoryBackend(), casLost: true}
		store := NewStore(backend, DefaultTTL).WithRequiredReplayCipher(requiredReplayV2TestCipher(t, "07"))
		lease, _, err := store.BeginRequiredReplay(t.Context(), scope, "cas-lost", fingerprint)
		if err != nil {
			t.Fatal(err)
		}
		if err := store.CompleteRequiredReplay(t.Context(), lease, RequiredReplayResponse{
			Status: http.StatusOK, Authorization: requiredReplayV2TestAuthorization("cas-lost-secret"),
		}); !errors.Is(err, ErrRequiredReplayLeaseLost) {
			t.Fatalf("completion error = %v", err)
		}
		if _, _, err := store.BeginRequiredReplay(
			t.Context(), scope, "cas-lost", fingerprint,
		); !errors.Is(err, ErrRequiredReplayInProgress) {
			t.Fatalf("CAS loss released pending: %v", err)
		}
	})
}

type requiredReplayV2CASBackend struct {
	*MemoryBackend
	casCalls int
	casErr   error
	casLost  bool
}

func (b *requiredReplayV2CASBackend) CompareAndSwap(
	ctx context.Context,
	key string,
	expected, replacement []byte,
	ttl time.Duration,
) (bool, error) {
	b.casCalls++
	if b.casErr != nil {
		return false, b.casErr
	}
	if b.casLost {
		return false, nil
	}
	return b.MemoryBackend.CompareAndSwap(ctx, key, expected, replacement, ttl)
}

func requiredReplayV2TestCipher(t *testing.T, pair string) *RequiredReplayCipher {
	t.Helper()
	cipher, err := NewRequiredReplayCipher(strings.Repeat(pair, 32))
	if err != nil || !cipher.Enabled() {
		t.Fatalf("cipher = %#v, %v", cipher, err)
	}
	return cipher
}

func requiredReplayV2TestAuthorization(secret string) *RequiredReplayAuthorization {
	return &RequiredReplayAuthorization{
		Schema:     "sforum.route-replay-authorization@1",
		PlanDigest: strings.Repeat("a", 64),
		BaseDigest: strings.Repeat("b", 64),
		RequestMutations: []RequiredReplayRequestMutation{{
			StepIndex: 0, BeforeDigest: strings.Repeat("c", 64), AfterDigest: strings.Repeat("d", 64),
			Operations: []RequiredReplayPatchOperation{{
				Kind: "replace", Path: "/body/private", Value: json.RawMessage(`"` + secret + `"`),
			}},
		}},
	}
}

func requiredReplayV2StoredBytes(
	t *testing.T,
	store *Store,
	backend *MemoryBackend,
	storageKey string,
) []byte {
	t.Helper()
	raw, found, err := backend.Get(t.Context(), store.fullKey(storageKey))
	if err != nil || !found {
		t.Fatalf("stored replay record: found=%t error=%v", found, err)
	}
	return raw
}

func requiredReplayV2SeedRecord(
	t *testing.T,
	scope RequiredReplayScope,
	key string,
	record requiredReplayRecord,
) (*MemoryBackend, *Store) {
	t.Helper()
	backend := NewMemoryBackend()
	store := NewStore(backend, DefaultTTL)
	storageKey, err := requiredReplayStorageKey(scope, key)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := backend.Set(t.Context(), store.fullKey(storageKey), raw, DefaultTTL); err != nil {
		t.Fatal(err)
	}
	return backend, store
}
