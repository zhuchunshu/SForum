package humanverify

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	altcha "github.com/altcha-org/altcha-lib-go/v2"
)

func TestAltchaProviderVerifiesSolvedChallenge(t *testing.T) {
	provider := NewAltchaProvider(AltchaConfig{
		Secret:       "test-secret",
		Cost:         1,
		ChallengeTTL: time.Minute,
	})

	challenge, err := provider.Challenge(context.Background(), PurposeRegister, Subject{IP: "127.0.0.1"})
	if err != nil {
		t.Fatalf("Challenge returned error: %v", err)
	}
	rawChallenge, ok := challenge.Payload.(altcha.Challenge)
	if !ok {
		t.Fatalf("expected altcha challenge payload, got %T", challenge.Payload)
	}

	solution, err := altcha.SolveChallenge(altcha.SolveChallengeOptions{
		Challenge: rawChallenge,
		DeriveKey: altcha.DeriveKeyPBKDF2(),
	})
	if err != nil {
		t.Fatalf("SolveChallenge returned error: %v", err)
	}
	payload, err := encodeAltchaPayload(rawChallenge, *solution)
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}

	result, err := provider.Verify(context.Background(), VerifyRequest{
		Purpose:  PurposeRegister,
		Provider: ProviderAltcha,
		Token:    payload,
	})
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if !result.Verified || result.Code != CodeOK {
		t.Fatalf("expected verified result, got %#v", result)
	}
}

func TestAltchaProviderReportsExpiredChallenge(t *testing.T) {
	provider := NewAltchaProvider(AltchaConfig{
		Secret:       "test-secret",
		Cost:         1,
		ChallengeTTL: -time.Minute,
	})

	challenge, err := provider.Challenge(context.Background(), PurposeRegister, Subject{})
	if err != nil {
		t.Fatalf("Challenge returned error: %v", err)
	}
	rawChallenge := challenge.Payload.(altcha.Challenge)
	payload, err := encodeAltchaPayload(rawChallenge, altcha.Solution{Counter: 0, DerivedKey: "00"})
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}

	result, err := provider.Verify(context.Background(), VerifyRequest{
		Purpose:  PurposeRegister,
		Provider: ProviderAltcha,
		Token:    payload,
	})
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if result.Code != CodeExpired {
		t.Fatalf("expected expired result, got %#v", result)
	}
}

func encodeAltchaPayload(challenge altcha.Challenge, solution altcha.Solution) (string, error) {
	body, err := json.Marshal(altcha.Payload{Challenge: challenge, Solution: solution})
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(body), nil
}
