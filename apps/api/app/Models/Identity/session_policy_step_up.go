package identity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	sessionPolicyStepUpTokenBytes = 32
	sessionPolicyStepUpTTL        = 10 * time.Minute
)

var (
	ErrSessionPolicyStepUpInvalid  = errors.New("identity: session policy step-up evidence is invalid")
	ErrSessionPolicyStepUpExpired  = errors.New("identity: session policy step-up evidence expired")
	ErrSessionPolicyStepUpReplayed = errors.New("identity: session policy step-up evidence was already used")
	ErrSessionPolicyStepUpStale    = errors.New("identity: session policy step-up evidence became stale")
	ErrSessionPolicyStepUpStore    = errors.New("identity: session policy step-up store is unavailable")
)

// SessionPolicyStepUpClaim binds one-use evidence to an exact evaluation tip.
// Plaintext tokens are never durable.
type SessionPolicyStepUpClaim struct {
	UserID            int64
	TokenVersion      int64
	Purpose           string
	PolicyID          string
	SelectionRevision int64
	RegistryRevision  uint64
	RegistryDigest    string
	PackageDigest     string
	OwnerExtensionID  string
	CorrelationID     string
	DeviceFingerprint string
}

// SessionPolicyStepUpIssueResult is returned once when a step_up disposition is
// first accepted. Token is plaintext; only TokenHash is stored.
type SessionPolicyStepUpIssueResult struct {
	Token     string
	TokenHash string
	ExpiresAt time.Time
	Claim     SessionPolicyStepUpClaim
}

// SessionPolicyStepUpStore issues and consumes Host-owned one-use evidence.
type SessionPolicyStepUpStore interface {
	Issue(ctx context.Context, claim SessionPolicyStepUpClaim, expiresAt time.Time, tokenHash string) error
	ConsumeForEffect(ctx context.Context, tokenHash string, expected SessionPolicyStepUpClaim) error
}

func hashSessionPolicyStepUpToken(token string) (string, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", ErrSessionPolicyStepUpInvalid
	}
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:]), nil
}

func newSessionPolicyStepUpToken() (plaintext string, tokenHash string, err error) {
	raw := make([]byte, sessionPolicyStepUpTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("generate session policy step-up token: %w", err)
	}
	plaintext = hex.EncodeToString(raw)
	tokenHash, err = hashSessionPolicyStepUpToken(plaintext)
	if err != nil {
		return "", "", err
	}
	return plaintext, tokenHash, nil
}

func normalizeSessionPolicyStepUpClaim(claim SessionPolicyStepUpClaim) (SessionPolicyStepUpClaim, error) {
	claim.Purpose = strings.TrimSpace(claim.Purpose)
	claim.PolicyID = strings.TrimSpace(claim.PolicyID)
	claim.RegistryDigest = strings.TrimSpace(claim.RegistryDigest)
	claim.PackageDigest = strings.TrimSpace(strings.ToLower(claim.PackageDigest))
	claim.OwnerExtensionID = strings.TrimSpace(claim.OwnerExtensionID)
	claim.CorrelationID = strings.TrimSpace(claim.CorrelationID)
	claim.DeviceFingerprint = strings.TrimSpace(claim.DeviceFingerprint)
	if claim.UserID <= 0 || claim.TokenVersion < 0 || claim.PolicyID == "" || claim.RegistryDigest == "" {
		return SessionPolicyStepUpClaim{}, ErrSessionPolicyStepUpInvalid
	}
	if claim.Purpose != SessionEvaluationPurposeIssue && claim.Purpose != SessionEvaluationPurposeRenew {
		return SessionPolicyStepUpClaim{}, ErrSessionPolicyStepUpInvalid
	}
	if claim.SelectionRevision < 0 {
		return SessionPolicyStepUpClaim{}, ErrSessionPolicyStepUpInvalid
	}
	return claim, nil
}

func sameSessionPolicyStepUpClaim(left, right SessionPolicyStepUpClaim) bool {
	return left.UserID == right.UserID &&
		left.TokenVersion == right.TokenVersion &&
		left.Purpose == right.Purpose &&
		left.PolicyID == right.PolicyID &&
		left.SelectionRevision == right.SelectionRevision &&
		left.RegistryRevision == right.RegistryRevision &&
		left.RegistryDigest == right.RegistryDigest &&
		left.PackageDigest == right.PackageDigest &&
		left.OwnerExtensionID == right.OwnerExtensionID &&
		left.CorrelationID == right.CorrelationID &&
		left.DeviceFingerprint == right.DeviceFingerprint
}

func sessionPolicyStepUpClaimFromResult(
	result SessionEvaluationResult,
	input preparedSessionEvaluationInput,
) SessionPolicyStepUpClaim {
	claim := SessionPolicyStepUpClaim{
		UserID:            input.UserID,
		TokenVersion:      input.TokenVersion,
		Purpose:           input.Purpose,
		PolicyID:          result.PolicyID,
		SelectionRevision: result.SelectionRevision,
		RegistryRevision:  result.RegistryRevision,
		RegistryDigest:    result.RegistryDigest,
		CorrelationID:     input.CorrelationID,
		DeviceFingerprint: input.DeviceFingerprint,
	}
	if result.Provider != nil {
		claim.PackageDigest = strings.TrimSpace(result.Provider.Artifact.PackageDigest)
		claim.OwnerExtensionID = strings.TrimSpace(result.Provider.Artifact.ExtensionID)
	}
	return claim
}

// MemorySessionPolicyStepUpStore is a process-local store for unit tests.
type MemorySessionPolicyStepUpStore struct {
	mu    sync.Mutex
	rows  map[string]memorySessionPolicyStepUpRow
	nowFn func() time.Time
}

type memorySessionPolicyStepUpRow struct {
	claim     SessionPolicyStepUpClaim
	expiresAt time.Time
	consumed  bool
}

func NewMemorySessionPolicyStepUpStore() *MemorySessionPolicyStepUpStore {
	return &MemorySessionPolicyStepUpStore{
		rows:  make(map[string]memorySessionPolicyStepUpRow),
		nowFn: time.Now,
	}
}

func (s *MemorySessionPolicyStepUpStore) Issue(
	ctx context.Context,
	claim SessionPolicyStepUpClaim,
	expiresAt time.Time,
	tokenHash string,
) error {
	if s == nil {
		return ErrSessionPolicyStepUpStore
	}
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	claim, err := normalizeSessionPolicyStepUpClaim(claim)
	if err != nil {
		return err
	}
	tokenHash = strings.ToLower(strings.TrimSpace(tokenHash))
	if tokenHash == "" || !isHex64(tokenHash) {
		return ErrSessionPolicyStepUpInvalid
	}
	now := s.nowFn()
	if !expiresAt.After(now) {
		return ErrSessionPolicyStepUpInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.rows[tokenHash]; exists {
		return ErrSessionPolicyStepUpInvalid
	}
	s.rows[tokenHash] = memorySessionPolicyStepUpRow{claim: claim, expiresAt: expiresAt}
	return nil
}

func (s *MemorySessionPolicyStepUpStore) ConsumeForEffect(
	ctx context.Context,
	tokenHash string,
	expected SessionPolicyStepUpClaim,
) error {
	if s == nil {
		return ErrSessionPolicyStepUpStore
	}
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	expected, err := normalizeSessionPolicyStepUpClaim(expected)
	if err != nil {
		return err
	}
	tokenHash = strings.ToLower(strings.TrimSpace(tokenHash))
	if tokenHash == "" || !isHex64(tokenHash) {
		return ErrSessionPolicyStepUpInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	row, ok := s.rows[tokenHash]
	if !ok {
		return ErrSessionPolicyStepUpInvalid
	}
	if row.consumed {
		return ErrSessionPolicyStepUpReplayed
	}
	if !s.nowFn().Before(row.expiresAt) {
		return ErrSessionPolicyStepUpExpired
	}
	if !sameSessionPolicyStepUpClaim(row.claim, expected) {
		return ErrSessionPolicyStepUpStale
	}
	row.consumed = true
	s.rows[tokenHash] = row
	return nil
}

func isHex64(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}
