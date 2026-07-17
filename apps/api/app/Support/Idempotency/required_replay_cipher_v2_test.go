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
			if _, err := cipher.EncryptReplay(
				"storage", strings.Repeat("a", 64), strings.Repeat("b", 64), []byte("secret"),
			); !errors.Is(err, ErrRequiredReplayCipherInvalid) {
				t.Fatalf("encrypt replay error = %v", err)
			}
			if _, err := cipher.DecryptReplay(
				"storage", strings.Repeat("a", 64), strings.Repeat("b", 64), "ciphertext",
			); !errors.Is(err, ErrRequiredReplayCipherInvalid) {
				t.Fatalf("decrypt replay error = %v", err)
			}
		})
	}
	for _, key := range []string{"not-hex", strings.Repeat("01", 31), strings.Repeat("01", 33)} {
		if cipher, err := NewRequiredReplayCipher(key); cipher != nil || !errors.Is(err, ErrRequiredReplayCipherInvalid) {
			t.Fatalf("key %q: cipher=%#v error=%v", key, cipher, err)
		}
	}
}

func TestRequiredReplayV3EncryptsCompleteResponseAndDetachesReplay(t *testing.T) {
	backend := NewMemoryBackend()
	store := NewStore(backend, DefaultTTL).WithRequiredReplayCipher(requiredReplayV2TestCipher(t, "03"))
	scope := requiredReplayTestScope("actor:72:bearer")
	fingerprint := strings.Repeat("a", 64)
	secret := "mutation-secret-must-not-reach-redis"
	body := `{"created":"response-body-secret-must-not-reach-redis"}`
	headerValue := "response-header-secret-must-not-reach-redis"
	canonicalPath := "/created/canonical-secret-must-not-reach-redis"
	lease, replay, err := store.BeginRequiredReplayBound(
		t.Context(), scope, "encrypted-transcript", requiredReplayV3TestBinding(fingerprint),
	)
	if err != nil || replay != nil {
		t.Fatalf("begin: lease=%#v replay=%#v error=%v", lease, replay, err)
	}
	authorization := requiredReplayV2TestAuthorization(secret)
	response := RequiredReplayResponse{
		Status:        http.StatusCreated,
		Headers:       http.Header{"Content-Type": {"application/json"}, "X-Private-Result": {headerValue}},
		Body:          []byte(body),
		CanonicalPath: canonicalPath,
		Authorization: authorization,
	}
	if err := store.CompleteRequiredReplay(t.Context(), lease, response); err != nil {
		t.Fatal(err)
	}

	raw := requiredReplayV2StoredBytes(t, store, backend, lease.storageKey)
	for _, forbidden := range []string{
		secret, body, headerValue, canonicalPath, "requestMutations", "beforeDigest", "afterDigest",
		"sforum.route-replay-authorization@1", `"response"`, `"authorizationCiphertext"`,
	} {
		if bytes.Contains(raw, []byte(forbidden)) {
			t.Fatalf("raw replay record contains payload plaintext %q: %s", forbidden, raw)
		}
	}
	if !bytes.Contains(raw, []byte(`"payloadCiphertext"`)) {
		t.Fatalf("raw replay record has no encrypted payload: %s", raw)
	}
	var record requiredReplayRecord
	if err := json.Unmarshal(raw, &record); err != nil || record.Schema != requiredReplaySchemaV3 ||
		record.PayloadCiphertext == "" || record.AuthorizationCiphertext != "" ||
		record.PlanDigest != authorization.PlanDigest ||
		record.Response != nil {
		t.Fatalf("stored record = %#v, %v", record, err)
	}

	// Mutating caller-owned values after completion must not alter the stored record.
	response.Body[0] = '!'
	authorization.RequestMutations[0].Operations[0].Value[1] = 'X'
	_, first, err := store.BeginRequiredReplayBound(
		t.Context(), scope, "encrypted-transcript", requiredReplayV3TestBinding(fingerprint),
	)
	if err != nil || first == nil || first.Authorization == nil || string(first.Body) != body ||
		first.Headers.Get("X-Private-Result") != headerValue || first.CanonicalPath != canonicalPath ||
		string(first.Authorization.RequestMutations[0].Operations[0].Value) != `"`+secret+`"` {
		t.Fatalf("first replay = %#v, %v", first, err)
	}
	first.Body[0] = '?'
	first.Headers.Set("Content-Type", "mutated")
	first.Authorization.RequestMutations[0].Operations[0].Value[1] = 'Y'
	_, detached, err := store.BeginRequiredReplayBound(
		t.Context(), scope, "encrypted-transcript", requiredReplayV3TestBinding(fingerprint),
	)
	if err != nil || detached == nil || detached.Authorization == nil || string(detached.Body) != body ||
		detached.Headers.Get("Content-Type") != "application/json" ||
		detached.Headers.Get("X-Private-Result") != headerValue || detached.CanonicalPath != canonicalPath ||
		string(detached.Authorization.RequestMutations[0].Operations[0].Value) != `"`+secret+`"` {
		t.Fatalf("detached replay = %#v, %v", detached, err)
	}
}

func TestRequiredReplayV3WrongKeyFailsClosedWithoutReplacingRecord(t *testing.T) {
	backend := NewMemoryBackend()
	right := NewStore(backend, DefaultTTL).WithRequiredReplayCipher(requiredReplayV2TestCipher(t, "04"))
	scope := requiredReplayTestScope("actor:73:cookie")
	fingerprint := strings.Repeat("b", 64)
	lease, _, err := right.BeginRequiredReplayBound(
		t.Context(), scope, "rotation", requiredReplayV3TestBinding(fingerprint),
	)
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
	if _, replay, err := wrong.BeginRequiredReplayBound(
		t.Context(), scope, "rotation", requiredReplayV3TestBinding(fingerprint),
	); replay != nil || !errors.Is(err, ErrRequiredReplayUnavailable) || !errors.Is(err, ErrRequiredReplayCipherInvalid) {
		t.Fatalf("wrong-key replay = %#v, %v", replay, err)
	}
	entry := backend.items[fullKey]
	if !bytes.Equal(entry.value, before) || !entry.expiresAt.Equal(beforeExpiry) {
		t.Fatal("wrong-key replay replaced or refreshed the durable record")
	}
	if _, replay, err := right.BeginRequiredReplayBound(
		t.Context(), scope, "rotation", requiredReplayV3TestBinding(fingerprint),
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
		if _, _, err := store.BeginRequiredReplayBound(
			t.Context(), scope, "v1-pending", requiredReplayV3TestBinding(currentFingerprint, legacyFingerprint),
		); !errors.Is(err, ErrRequiredReplayInProgress) {
			t.Fatalf("V1 pending error = %v", err)
		}
	})

	t.Run("V2 pending retains rolling fingerprint compatibility", func(t *testing.T) {
		_, store := requiredReplayV2SeedRecord(t, scope, "v2-pending", requiredReplayRecord{
			Schema: requiredReplaySchemaV2, State: requiredReplayPending,
			Fingerprint: legacyFingerprint, LeaseToken: strings.Repeat("2", 32),
		})
		if _, _, err := store.BeginRequiredReplayBound(
			t.Context(), scope, "v2-pending", requiredReplayV3TestBinding(currentFingerprint, legacyFingerprint),
		); !errors.Is(err, ErrRequiredReplayInProgress) {
			t.Fatalf("V2 pending error = %v", err)
		}
	})

	t.Run("V1 completed response-only replay remains compatible", func(t *testing.T) {
		_, store := requiredReplayV2SeedRecord(t, scope, "v1-completed", requiredReplayRecord{
			Schema: requiredReplaySchemaV1, State: requiredReplayCompleted, Fingerprint: legacyFingerprint,
			Response: &RequiredReplayResponse{Status: http.StatusCreated, Body: []byte("legacy")},
		})
		_, replay, err := store.BeginRequiredReplayBound(
			t.Context(), scope, "v1-completed", requiredReplayV3TestBinding(currentFingerprint, legacyFingerprint),
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
		_, replay, err := store.BeginRequiredReplayBound(
			t.Context(), scope, "v2-completed", requiredReplayV3TestBinding(currentFingerprint),
		)
		if err != nil || replay == nil || string(replay.Body) != "current" || replay.Authorization != nil {
			t.Fatalf("V2 replay = %#v, %v", replay, err)
		}
	})

	t.Run("V2 completed encrypted transcript remains compatible", func(t *testing.T) {
		cipher := requiredReplayV2TestCipher(t, "0b")
		key := "v2-encrypted"
		storageKey, err := requiredReplayStorageKey(scope, key)
		if err != nil {
			t.Fatal(err)
		}
		authorization := requiredReplayV2TestAuthorization("v2-compatibility-secret")
		plaintext, err := json.Marshal(authorization)
		if err != nil {
			t.Fatal(err)
		}
		ciphertext, err := cipher.Encrypt(storageKey, currentFingerprint, authorization.PlanDigest, plaintext)
		if err != nil {
			t.Fatal(err)
		}
		_, store := requiredReplayV2SeedRecord(t, scope, key, requiredReplayRecord{
			Schema: requiredReplaySchemaV2, State: requiredReplayCompleted, Fingerprint: currentFingerprint,
			Response:   &RequiredReplayResponse{Status: http.StatusOK, Body: []byte("v2-response")},
			PlanDigest: authorization.PlanDigest, AuthorizationCiphertext: ciphertext,
		})
		store.WithRequiredReplayCipher(cipher)
		_, replay, err := store.BeginRequiredReplayBound(
			t.Context(), scope, key, requiredReplayV3TestBinding(currentFingerprint),
		)
		if err != nil || replay == nil || replay.Authorization == nil || string(replay.Body) != "v2-response" ||
			string(replay.Authorization.RequestMutations[0].Operations[0].Value) != `"v2-compatibility-secret"` {
			t.Fatalf("V2 encrypted replay = %#v, %v", replay, err)
		}
	})

	t.Run("V2 response-only records retain rolling fingerprint compatibility", func(t *testing.T) {
		_, store := requiredReplayV2SeedRecord(t, scope, "v2-legacy-fingerprint", requiredReplayRecord{
			Schema: requiredReplaySchemaV2, State: requiredReplayCompleted, Fingerprint: legacyFingerprint,
			Response: &RequiredReplayResponse{Status: http.StatusOK, Body: []byte("rolling-v2")},
		})
		_, replay, err := store.BeginRequiredReplayBound(
			t.Context(), scope, "v2-legacy-fingerprint", requiredReplayV3TestBinding(currentFingerprint, legacyFingerprint),
		)
		if err != nil || replay == nil || string(replay.Body) != "rolling-v2" {
			t.Fatalf("V2 rolling replay=%#v error=%v", replay, err)
		}
	})

	t.Run("V2 encrypted transcript cannot borrow the legacy fingerprint", func(t *testing.T) {
		cipher := requiredReplayV2TestCipher(t, "11")
		key := "v2-encrypted-legacy-fingerprint"
		storageKey, err := requiredReplayStorageKey(scope, key)
		if err != nil {
			t.Fatal(err)
		}
		authorization := requiredReplayV2TestAuthorization("must-not-borrow")
		plaintext, err := json.Marshal(authorization)
		if err != nil {
			t.Fatal(err)
		}
		ciphertext, err := cipher.Encrypt(storageKey, legacyFingerprint, authorization.PlanDigest, plaintext)
		if err != nil {
			t.Fatal(err)
		}
		_, store := requiredReplayV2SeedRecord(t, scope, key, requiredReplayRecord{
			Schema: requiredReplaySchemaV2, State: requiredReplayCompleted, Fingerprint: legacyFingerprint,
			Response: &RequiredReplayResponse{Status: http.StatusOK}, PlanDigest: authorization.PlanDigest,
			AuthorizationCiphertext: ciphertext,
		})
		store.WithRequiredReplayCipher(cipher)
		if _, replay, err := store.BeginRequiredReplayBound(
			t.Context(), scope, key, requiredReplayV3TestBinding(currentFingerprint, legacyFingerprint),
		); replay != nil || !errors.Is(err, ErrRequiredReplayFingerprintConflict) {
			t.Fatalf("V2 encrypted legacy replay=%#v error=%v", replay, err)
		}
	})

	t.Run("V3 records cannot borrow the legacy fingerprint", func(t *testing.T) {
		_, store := requiredReplayV2SeedRecord(t, scope, "v3-legacy-fingerprint", requiredReplayRecord{
			Schema: requiredReplaySchemaV3, State: requiredReplayCompleted, Fingerprint: legacyFingerprint,
			PlanDigest: strings.Repeat("a", 64), PayloadCiphertext: "ciphertext",
		})
		if _, replay, err := store.BeginRequiredReplayBound(
			t.Context(), scope, "v3-legacy-fingerprint", requiredReplayV3TestBinding(currentFingerprint, legacyFingerprint),
		); replay != nil || !errors.Is(err, ErrRequiredReplayFingerprintConflict) {
			t.Fatalf("V3 legacy replay=%#v error=%v", replay, err)
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
		{name: "V2 payload ciphertext", record: requiredReplayRecord{
			Schema: requiredReplaySchemaV2, State: requiredReplayCompleted, Fingerprint: currentFingerprint,
			Response: &RequiredReplayResponse{Status: http.StatusOK}, PayloadCiphertext: "ciphertext",
		}},
		{name: "V3 plaintext response", record: requiredReplayRecord{
			Schema: requiredReplaySchemaV3, State: requiredReplayCompleted, Fingerprint: currentFingerprint,
			PlanDigest: strings.Repeat("e", 64), Response: &RequiredReplayResponse{Status: http.StatusOK},
			PayloadCiphertext: "ciphertext",
		}},
		{name: "V3 legacy ciphertext fields", record: requiredReplayRecord{
			Schema: requiredReplaySchemaV3, State: requiredReplayCompleted, Fingerprint: currentFingerprint,
			PlanDigest: strings.Repeat("e", 64), AuthorizationCiphertext: "legacy", PayloadCiphertext: "ciphertext",
		}},
		{name: "V3 missing payload", record: requiredReplayRecord{
			Schema: requiredReplaySchemaV3, State: requiredReplayCompleted, Fingerprint: currentFingerprint,
			PlanDigest: strings.Repeat("e", 64),
		}},
		{name: "V3 pending with payload", record: requiredReplayRecord{
			Schema: requiredReplaySchemaV3, State: requiredReplayPending, Fingerprint: currentFingerprint,
			PlanDigest: strings.Repeat("e", 64), LeaseToken: strings.Repeat("3", 32), PayloadCiphertext: "ciphertext",
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

func TestRequiredReplayV2LegacyWriterRemainsAvailableUntilBoundCallerCutover(t *testing.T) {
	backend := NewMemoryBackend()
	store := NewStore(backend, DefaultTTL).WithRequiredReplayCipher(requiredReplayV2TestCipher(t, "12"))
	scope := requiredReplayTestScope("actor:76:bearer")
	fingerprint := strings.Repeat("8", 64)
	lease, replay, err := store.BeginRequiredReplay(t.Context(), scope, "legacy-writer", fingerprint)
	if err != nil || replay != nil || lease.storageKey == "" {
		t.Fatalf("legacy begin: lease=%#v replay=%#v error=%v", lease, replay, err)
	}
	if err := store.CompleteRequiredReplay(t.Context(), lease, RequiredReplayResponse{
		Status: http.StatusCreated, Body: []byte("legacy-v2-response"),
	}); err != nil {
		t.Fatal(err)
	}
	raw := requiredReplayV2StoredBytes(t, store, backend, lease.storageKey)
	var record requiredReplayRecord
	if json.Unmarshal(raw, &record) != nil || record.Schema != requiredReplaySchemaV2 ||
		record.Response == nil || string(record.Response.Body) != "legacy-v2-response" || record.PayloadCiphertext != "" {
		t.Fatalf("legacy V2 record = %#v", record)
	}
	if _, replay, err := store.BeginRequiredReplay(
		t.Context(), scope, "legacy-writer", fingerprint,
	); err != nil || replay == nil || string(replay.Body) != "legacy-v2-response" {
		t.Fatalf("legacy V2 replay = %#v, %v", replay, err)
	}

	_, legacyReader := requiredReplayV2SeedRecord(t, scope, "legacy-reader-v3", requiredReplayRecord{
		Schema: requiredReplaySchemaV3, State: requiredReplayCompleted, Fingerprint: fingerprint,
		PlanDigest: strings.Repeat("9", 64), PayloadCiphertext: "ciphertext",
	})
	if _, replay, err := legacyReader.BeginRequiredReplay(
		t.Context(), scope, "legacy-reader-v3", fingerprint,
	); replay != nil || !errors.Is(err, ErrRequiredReplayUnavailable) {
		t.Fatalf("legacy API accepted V3 replay = %#v, %v", replay, err)
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
	v3Record := requiredReplayRecord{
		Schema: requiredReplaySchemaV3, State: requiredReplayCompleted,
		Fingerprint: strings.Repeat("e", 64), PlanDigest: strings.Repeat("f", 64),
		PayloadCiphertext: "ciphertext",
	}
	v3Raw, err := json.Marshal(v3Record)
	if err != nil {
		t.Fatal(err)
	}
	v3Exact := append(v3Raw, bytes.Repeat([]byte(" "), MaxRequiredReplayEncryptedRecord-len(v3Raw))...)
	if _, err := decodeRequiredReplayRecord(v3Exact); err != nil {
		t.Fatalf("exact V3 record cap rejected: %v", err)
	}
	if _, err := decodeRequiredReplayRecord(append(v3Exact, ' ')); !errors.Is(err, ErrRequiredReplayUnavailable) {
		t.Fatalf("V3 record above cap error = %v", err)
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

	t.Run("missing cipher never acquires a new record", func(t *testing.T) {
		backend := &requiredReplayV2CASBackend{MemoryBackend: NewMemoryBackend()}
		store := NewStore(backend, DefaultTTL)
		lease, replay, err := store.BeginRequiredReplayBound(
			t.Context(), scope, "missing-cipher", requiredReplayV3TestBinding(fingerprint),
		)
		if lease.storageKey != "" || replay != nil || !errors.Is(err, ErrRequiredReplayUnavailable) ||
			!errors.Is(err, ErrRequiredReplayCipherInvalid) {
			t.Fatalf("begin result: lease=%#v replay=%#v error=%v", lease, replay, err)
		}
		if backend.casCalls != 0 || len(backend.items) != 0 {
			t.Fatalf("missing cipher mutated storage: CAS=%d items=%d", backend.casCalls, len(backend.items))
		}
	})

	t.Run("CAS error preserves pending", func(t *testing.T) {
		backend := &requiredReplayV2CASBackend{MemoryBackend: NewMemoryBackend(), casErr: errors.New("redis unavailable")}
		store := NewStore(backend, DefaultTTL).WithRequiredReplayCipher(requiredReplayV2TestCipher(t, "06"))
		lease, _, err := store.BeginRequiredReplayBound(
			t.Context(), scope, "cas-error", requiredReplayV3TestBinding(fingerprint),
		)
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
		if _, _, err := store.BeginRequiredReplayBound(
			t.Context(), scope, "cas-error", requiredReplayV3TestBinding(fingerprint),
		); !errors.Is(err, ErrRequiredReplayInProgress) {
			t.Fatalf("CAS error released pending: %v", err)
		}
	})

	t.Run("CAS loss preserves the replacement owner", func(t *testing.T) {
		backend := &requiredReplayV2CASBackend{MemoryBackend: NewMemoryBackend(), casLost: true}
		store := NewStore(backend, DefaultTTL).WithRequiredReplayCipher(requiredReplayV2TestCipher(t, "07"))
		lease, _, err := store.BeginRequiredReplayBound(
			t.Context(), scope, "cas-lost", requiredReplayV3TestBinding(fingerprint),
		)
		if err != nil {
			t.Fatal(err)
		}
		if err := store.CompleteRequiredReplay(t.Context(), lease, RequiredReplayResponse{
			Status: http.StatusOK, Authorization: requiredReplayV2TestAuthorization("cas-lost-secret"),
		}); !errors.Is(err, ErrRequiredReplayLeaseLost) {
			t.Fatalf("completion error = %v", err)
		}
		if _, _, err := store.BeginRequiredReplayBound(
			t.Context(), scope, "cas-lost", requiredReplayV3TestBinding(fingerprint),
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

func requiredReplayV3TestBinding(fingerprint string, compatible ...string) RequiredReplayBinding {
	return RequiredReplayBinding{
		Fingerprint: fingerprint, PlanDigest: strings.Repeat("a", 64),
		CompatibleFingerprints: append([]string(nil), compatible...),
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
