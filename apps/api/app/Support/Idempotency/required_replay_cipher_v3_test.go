package idempotency

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestRequiredReplayCipherV3BindsCompleteContextAndUsesUniqueNonces(t *testing.T) {
	cipherA := requiredReplayV2TestCipher(t, "0e")
	cipherB := requiredReplayV2TestCipher(t, "0f")
	storageKey := "plugin-route:required.24h@1:v3-storage-a"
	fingerprint := strings.Repeat("1", 64)
	planDigest := strings.Repeat("2", 64)
	plaintext := []byte(`{"response":{"status":201,"body":"complete-response-secret"}}`)

	first, err := cipherA.EncryptReplay(storageKey, fingerprint, planDigest, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	second, err := cipherA.EncryptReplay(storageKey, fingerprint, planDigest, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("V3 AES-GCM encryption reused a nonce")
	}
	firstRaw, err := base64.RawURLEncoding.DecodeString(first)
	if err != nil {
		t.Fatal(err)
	}
	secondRaw, err := base64.RawURLEncoding.DecodeString(second)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(
		firstRaw[:cipherA.payloadAEAD.NonceSize()], secondRaw[:cipherA.payloadAEAD.NonceSize()],
	) {
		t.Fatal("V3 ciphertexts contain the same nonce")
	}
	if bytes.Contains(firstRaw, plaintext) || strings.Contains(first, "complete-response-secret") {
		t.Fatal("V3 ciphertext contains response plaintext")
	}
	decrypted, err := cipherA.DecryptReplay(storageKey, fingerprint, planDigest, first)
	if err != nil || !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("V3 round trip = %q, %v", decrypted, err)
	}

	tamperedRaw := append([]byte(nil), firstRaw...)
	tamperedRaw[len(tamperedRaw)-1] ^= 0x01
	legacyCiphertext, err := cipherA.Encrypt(storageKey, fingerprint, planDigest, plaintext)
	if err != nil {
		t.Fatal(err)
	}
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
		{name: "fingerprint AAD", cipher: cipherA, storageKey: storageKey, fingerprint: strings.Repeat("3", 64), planDigest: planDigest, ciphertext: first},
		{name: "plan digest AAD", cipher: cipherA, storageKey: storageKey, fingerprint: fingerprint, planDigest: strings.Repeat("4", 64), ciphertext: first},
		{name: "tampered ciphertext", cipher: cipherA, storageKey: storageKey, fingerprint: fingerprint, planDigest: planDigest, ciphertext: base64.RawURLEncoding.EncodeToString(tamperedRaw)},
		{name: "malformed base64", cipher: cipherA, storageKey: storageKey, fingerprint: fingerprint, planDigest: planDigest, ciphertext: "not!base64"},
		{name: "truncated ciphertext", cipher: cipherA, storageKey: storageKey, fingerprint: fingerprint, planDigest: planDigest, ciphertext: base64.RawURLEncoding.EncodeToString(firstRaw[:cipherA.payloadAEAD.NonceSize()])},
		{name: "V2 domain ciphertext", cipher: cipherA, storageKey: storageKey, fingerprint: fingerprint, planDigest: planDigest, ciphertext: legacyCiphertext},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			if _, err := check.cipher.DecryptReplay(
				check.storageKey, check.fingerprint, check.planDigest, check.ciphertext,
			); !errors.Is(err, ErrRequiredReplayCipherInvalid) {
				t.Fatalf("decrypt error = %v", err)
			}
		})
	}
	if _, err := cipherA.Decrypt(storageKey, fingerprint, planDigest, first); !errors.Is(err, ErrRequiredReplayCipherInvalid) {
		t.Fatalf("V3 ciphertext opened under V2 domain: %v", err)
	}
	if _, err := cipherA.EncryptReplay(
		storageKey, fingerprint, planDigest, make([]byte, MaxRequiredReplayPayload+1),
	); !errors.Is(err, ErrRequiredReplayCipherInvalid) {
		t.Fatalf("oversized V3 plaintext error = %v", err)
	}
	encodedLimit := base64.RawURLEncoding.EncodedLen(
		MaxRequiredReplayPayload + cipherA.payloadAEAD.NonceSize() + cipherA.payloadAEAD.Overhead(),
	)
	if _, err := cipherA.DecryptReplay(
		storageKey, fingerprint, planDigest, strings.Repeat("A", encodedLimit+1),
	); !errors.Is(err, ErrRequiredReplayCipherInvalid) {
		t.Fatalf("oversized V3 ciphertext error = %v", err)
	}
}

func TestRequiredReplayV3RejectsCiphertextSubstitutionAndMalformedPayload(t *testing.T) {
	backend := NewMemoryBackend()
	cipher := requiredReplayV2TestCipher(t, "10")
	store := NewStore(backend, DefaultTTL).WithRequiredReplayCipher(cipher)
	binding := RequiredReplayBinding{
		Fingerprint: strings.Repeat("5", 64), PlanDigest: strings.Repeat("6", 64),
	}
	scopeA := requiredReplayTestScope("actor:80:bearer")
	scopeB := requiredReplayTestScope("actor:81:bearer")
	leaseA, _, err := store.BeginRequiredReplayBound(t.Context(), scopeA, "substitution-a", binding)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CompleteRequiredReplay(t.Context(), leaseA, RequiredReplayResponse{
		Status: http.StatusCreated, Body: []byte("response-a"),
	}); err != nil {
		t.Fatal(err)
	}
	leaseB, _, err := store.BeginRequiredReplayBound(t.Context(), scopeB, "substitution-b", binding)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CompleteRequiredReplay(t.Context(), leaseB, RequiredReplayResponse{
		Status: http.StatusCreated, Body: []byte("response-b"),
	}); err != nil {
		t.Fatal(err)
	}

	rawA := requiredReplayV2StoredBytes(t, store, backend, leaseA.storageKey)
	rawB := requiredReplayV2StoredBytes(t, store, backend, leaseB.storageKey)
	var recordA, recordB requiredReplayRecord
	if json.Unmarshal(rawA, &recordA) != nil || json.Unmarshal(rawB, &recordB) != nil {
		t.Fatal("decode stored V3 records")
	}
	recordB.PayloadCiphertext = recordA.PayloadCiphertext
	substituted, err := json.Marshal(recordB)
	if err != nil {
		t.Fatal(err)
	}
	if err := backend.Set(t.Context(), store.fullKey(leaseB.storageKey), substituted, DefaultTTL); err != nil {
		t.Fatal(err)
	}
	if _, replay, err := store.BeginRequiredReplayBound(
		t.Context(), scopeB, "substitution-b", binding,
	); replay != nil || !errors.Is(err, ErrRequiredReplayUnavailable) ||
		!errors.Is(err, ErrRequiredReplayCipherInvalid) {
		t.Fatalf("substituted replay = %#v, %v", replay, err)
	}

	malformed, err := cipher.EncryptReplay(
		leaseB.storageKey, binding.Fingerprint, binding.PlanDigest, []byte(`{"schema":"wrong"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	recordB.PayloadCiphertext = malformed
	malformedRecord, err := json.Marshal(recordB)
	if err != nil {
		t.Fatal(err)
	}
	if err := backend.Set(t.Context(), store.fullKey(leaseB.storageKey), malformedRecord, DefaultTTL); err != nil {
		t.Fatal(err)
	}
	if _, replay, err := store.BeginRequiredReplayBound(
		t.Context(), scopeB, "substitution-b", binding,
	); replay != nil || !errors.Is(err, ErrRequiredReplayUnavailable) {
		t.Fatalf("malformed payload replay = %#v, %v", replay, err)
	}

	wrongPlan := binding
	wrongPlan.PlanDigest = strings.Repeat("7", 64)
	if err := json.Unmarshal(rawB, &recordB); err != nil {
		t.Fatal(err)
	}
	recordB.PlanDigest = wrongPlan.PlanDigest
	wrongPlanRecord, err := json.Marshal(recordB)
	if err != nil {
		t.Fatal(err)
	}
	if err := backend.Set(t.Context(), store.fullKey(leaseB.storageKey), wrongPlanRecord, DefaultTTL); err != nil {
		t.Fatal(err)
	}
	if _, replay, err := store.BeginRequiredReplayBound(
		t.Context(), scopeB, "substitution-b", wrongPlan,
	); replay != nil || !errors.Is(err, ErrRequiredReplayUnavailable) ||
		!errors.Is(err, ErrRequiredReplayCipherInvalid) {
		t.Fatalf("wrong-plan replay = %#v, %v", replay, err)
	}

}
