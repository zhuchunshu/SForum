package consentbridge

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net/url"
	"sort"
	"strings"
	"time"
)

type Service struct {
	store Store
	clock Clock
	ttl   time.Duration
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

func NewService(store Store) *Service {
	return &Service{store: store, clock: systemClock{}, ttl: TransactionTTL}
}

func (s *Service) WithClock(clock Clock) *Service {
	if s != nil && clock != nil {
		s.clock = clock
	}
	return s
}

func (s *Service) WithTTL(ttl time.Duration) *Service {
	if s != nil && ttl > 0 && ttl <= time.Hour {
		s.ttl = ttl
	}
	return s
}

func (s *Service) Begin(ctx context.Context, input BeginInput) (ConsentFrame, error) {
	if s == nil || s.store == nil || ctx == nil || input.ActorUserID <= 0 || !validFingerprint(input.SessionFingerprint) ||
		!validBinding(input.Binding) || !validClient(input.Client) || !safeRedirect(input.RedirectURI) || input.RequireRecentAuth && !input.RecentAuth {
		if input.RequireRecentAuth && !input.RecentAuth {
			return ConsentFrame{}, ErrRecentAuth
		}
		return ConsentFrame{}, ErrInvalid
	}
	scopes := normalizeScopes(input.Scopes)
	if len(scopes) == 0 {
		return ConsentFrame{}, ErrInvalid
	}
	id, err := randomToken(24)
	if err != nil {
		return ConsentFrame{}, err
	}
	csrf, err := randomToken(32)
	if err != nil {
		return ConsentFrame{}, err
	}
	now := s.clock.Now().UTC()
	frame := ConsentFrame{TransactionID: id, CSRFToken: csrf, Client: input.Client, RedirectURI: input.RedirectURI, Scopes: scopes, ExpiresAt: now.Add(s.ttl)}
	if err := s.store.Put(ctx, TransactionRecord{Frame: frame, ActorUserID: input.ActorUserID, SessionFingerprint: input.SessionFingerprint, Binding: input.Binding, RequireRecentAuth: input.RequireRecentAuth}); err != nil {
		return ConsentFrame{}, err
	}
	return frame, nil
}

// Decide atomically consumes the transaction before returning the decision.
// The caller must persist the OAuth-specific result separately; replaying this
// method can never mint a second decision.
func (s *Service) Decide(ctx context.Context, actorUserID int64, sessionFingerprint string, binding ArtifactBinding, transactionID, csrfToken string, decision Decision) error {
	if s == nil || s.store == nil || ctx == nil || actorUserID <= 0 || !validFingerprint(sessionFingerprint) || !validBinding(binding) || !validToken(transactionID) || !validToken(csrfToken) {
		return ErrInvalid
	}
	record, err := s.store.Get(ctx, transactionID)
	if err != nil {
		return err
	}
	now := s.clock.Now().UTC()
	if record.UsedAt != nil {
		return ErrReplay
	}
	if !record.Frame.ExpiresAt.After(now) {
		return ErrExpired
	}
	if record.ActorUserID != actorUserID {
		return ErrActorMismatch
	}
	if record.SessionFingerprint != sessionFingerprint {
		return ErrSessionMismatch
	}
	if record.Binding != binding {
		return ErrArtifactMismatch
	}
	if !secureEqual(record.Frame.CSRFToken, csrfToken) {
		return ErrCSRF
	}
	if decision.Approved && !sameScopes(record.Frame.Scopes, decision.Scopes) {
		return ErrInvalid
	}
	if err := s.store.Consume(ctx, transactionID, now); err != nil {
		return err
	}
	return nil
}

func validBinding(binding ArtifactBinding) bool {
	return validToken(binding.ExtensionID) && validToken(binding.ExtensionVersion) && validDigest(binding.PackageDigest) && validDigest(binding.ImpactDigest)
}

func validClient(client ClientDescriptor) bool {
	return validToken(client.ClientID) && strings.TrimSpace(client.Name) != "" && len(client.Name) <= 160
}

func validFingerprint(value string) bool {
	return len(value) == 64 && validDigest(value)
}

func validToken(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 256 {
		return false
	}
	for _, r := range value {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

func validDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

func safeRedirect(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	return err == nil && parsed.IsAbs() && parsed.Scheme == "https" && parsed.Host != "" && parsed.User == nil
}

func normalizeScopes(scopes []string) []string {
	set := map[string]bool{}
	for _, scope := range scopes {
		scope = strings.ToLower(strings.TrimSpace(scope))
		if scope == "openid" || scope == "profile" || scope == "email" {
			set[scope] = true
		}
	}
	result := make([]string, 0, len(set))
	for scope := range set {
		result = append(result, scope)
	}
	sort.Strings(result)
	return result
}

func sameScopes(left, right []string) bool {
	left = normalizeScopes(left)
	right = normalizeScopes(right)
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func randomToken(size int) (string, error) {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func secureEqual(left, right string) bool {
	leftDigest := sha256.Sum256([]byte(left))
	rightDigest := sha256.Sum256([]byte(right))
	return subtle.ConstantTimeCompare(leftDigest[:], rightDigest[:]) == 1
}
