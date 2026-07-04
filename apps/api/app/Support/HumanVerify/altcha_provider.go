package humanverify

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"time"

	altcha "github.com/altcha-org/altcha-lib-go/v2"
)

type AltchaConfig struct {
	Secret       string
	Cost         int
	ChallengeTTL time.Duration
}

type AltchaProvider struct {
	secret       string
	cost         int
	challengeTTL time.Duration
}

func NewAltchaProvider(cfg AltchaConfig) *AltchaProvider {
	if cfg.Cost <= 0 {
		cfg.Cost = 1000
	}
	if cfg.ChallengeTTL == 0 {
		cfg.ChallengeTTL = 10 * time.Minute
	}
	return &AltchaProvider{
		secret:       cfg.Secret,
		cost:         cfg.Cost,
		challengeTTL: cfg.ChallengeTTL,
	}
}

func (p *AltchaProvider) Challenge(_ context.Context, purpose Purpose, _ Subject) (Challenge, error) {
	expiresAt := time.Now().Add(p.challengeTTL)
	challenge, err := altcha.CreateChallenge(altcha.CreateChallengeOptions{
		Algorithm:           "PBKDF2/SHA-256",
		Cost:                p.cost,
		ExpiresAt:           &expiresAt,
		HMACSignatureSecret: p.secret,
	})
	if err != nil {
		return Challenge{}, err
	}
	return Challenge{Provider: ProviderAltcha, Purpose: purpose, Payload: challenge}, nil
}

func (p *AltchaProvider) Verify(_ context.Context, req VerifyRequest) (VerifyResult, error) {
	payload, err := decodePayload(req.Token)
	if err != nil {
		return VerifyResult{Verified: false, Code: CodeInvalid}, nil
	}

	result, err := altcha.VerifySolution(altcha.VerifySolutionOptions{
		Challenge:           payload.Challenge,
		Solution:            payload.Solution,
		DeriveKey:           altcha.DeriveKeyPBKDF2(),
		HMACSignatureSecret: p.secret,
	})
	if err != nil {
		return VerifyResult{}, err
	}
	if result.Expired {
		return VerifyResult{Verified: false, Code: CodeExpired}, nil
	}
	if result.InvalidSignature != nil && *result.InvalidSignature {
		return VerifyResult{Verified: false, Code: CodeInvalid}, nil
	}
	if result.InvalidSolution != nil && *result.InvalidSolution {
		return VerifyResult{Verified: false, Code: CodeInvalid}, nil
	}
	if !result.Verified {
		return VerifyResult{Verified: false, Code: CodeInvalid}, nil
	}
	return VerifyResult{Verified: true, Code: CodeOK}, nil
}

func decodePayload(token string) (altcha.Payload, error) {
	body, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return altcha.Payload{}, err
	}

	var payload altcha.Payload
	if err := json.Unmarshal(body, &payload); err != nil {
		return altcha.Payload{}, err
	}
	return payload, nil
}
