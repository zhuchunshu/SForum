package identitydelegation

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"strings"
	"time"
)

type Service struct {
	store SubjectStore
	clock Clock
	ttl   time.Duration
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

func NewService(store SubjectStore) *Service {
	return &Service{store: store, clock: systemClock{}, ttl: SubjectTTL}
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

// Issue creates a fresh subject for one exact plugin artifact. Subject values
// are random opaque bytes; deterministic encodings of Core user IDs are not
// permitted because they become cross-plugin correlation identifiers.
func (s *Service) Issue(ctx context.Context, binding ArtifactBinding, actor ActorProjectionInput, authTime time.Time) (DelegatedIdentity, error) {
	if s == nil || s.store == nil || ctx == nil || !validBinding(binding) || actor.UserID <= 0 || actor.Status != ActorStatusActive {
		return DelegatedIdentity{}, ErrActorUnavailable
	}
	now := s.clock.Now().UTC()
	if authTime.IsZero() || authTime.After(now.Add(time.Second)) {
		return DelegatedIdentity{}, ErrInvalid
	}
	subjectBytes := make([]byte, 32)
	if _, err := rand.Read(subjectBytes); err != nil {
		return DelegatedIdentity{}, err
	}
	identity := DelegatedIdentity{
		Subject:  base64.RawURLEncoding.EncodeToString(subjectBytes),
		Username: trimClaim(actor.Username, 128), DisplayName: trimClaim(actor.DisplayName, 256),
		Locale: trimClaim(actor.Locale, 32), AuthTime: authTime.UTC(), ExpiresAt: now.Add(s.ttl),
	}
	if actor.EmailVerified {
		identity.Email = trimClaim(actor.Email, 320)
		identity.EmailVerified = identity.Email != ""
	}
	if identity.Username == "" && identity.DisplayName == "" && identity.Email == "" {
		return DelegatedIdentity{}, ErrInvalid
	}
	if err := s.store.Put(ctx, SubjectRecord{
		Binding: binding, Subject: identity.Subject, ActorUserID: actor.UserID,
		Identity: identity, CreatedAt: now, ExpiresAt: identity.ExpiresAt,
	}); err != nil {
		return DelegatedIdentity{}, err
	}
	return identity, nil
}

func (s *Service) Resolve(ctx context.Context, binding ArtifactBinding, subject string) (DelegatedIdentity, error) {
	if s == nil || s.store == nil || ctx == nil || !validBinding(binding) || !validSubject(subject) {
		return DelegatedIdentity{}, ErrInvalid
	}
	record, err := s.store.Get(ctx, binding, subject)
	if err != nil {
		return DelegatedIdentity{}, err
	}
	now := s.clock.Now().UTC()
	if record.Binding != binding || record.Subject != subject || record.RevokedAt != nil || record.ExpiresAt.Before(now) {
		return DelegatedIdentity{}, ErrSubjectNotFound
	}
	identity := record.Identity
	identity.Email = strings.TrimSpace(identity.Email)
	if !identity.EmailVerified {
		identity.Email = ""
	}
	return identity, nil
}

func validBinding(binding ArtifactBinding) bool {
	return validToken(binding.ExtensionID, 120) && validToken(binding.ExtensionVersion, 32) &&
		validDigest(binding.PackageDigest) && validDigest(binding.ImpactDigest)
}

func validSubject(subject string) bool {
	return len(subject) >= 40 && len(subject) <= 128 && !strings.ContainsAny(subject, "=+/\\")
}

func validToken(value string, max int) bool {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > max {
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
	value = strings.TrimSpace(value)
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

func trimClaim(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) > max {
		return value[:max]
	}
	return value
}
