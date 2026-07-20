package marketplace

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Service verifies and queries a Host marketplace index.
type Service struct {
	mu       sync.Mutex
	index    *Index
	policy   OperatorPolicy
	hmacKey  []byte
	// now injectable for stale tests.
	now func() time.Time
}

// New builds a marketplace service. hmacKey signs/verifies indexes (dev may be empty if AllowUnsigned).
func New(hmacKey []byte, policy OperatorPolicy) *Service {
	if len(policy.AllowedChannels) == 0 {
		policy.AllowedChannels = []string{ChannelStable}
	}
	policy.DirectUploadFallback = true // recommended default: offline upload remains
	return &Service{
		policy:  policy,
		hmacKey: append([]byte(nil), hmacKey...),
		now:     func() time.Time { return time.Now().UTC() },
	}
}

// LoadIndex verifies signature (unless AllowUnsigned) and replaces the active index.
func (s *Service) LoadIndex(index Index) error {
	if s == nil {
		return ErrInvalid
	}
	index.SchemaVersion = strings.TrimSpace(index.SchemaVersion)
	if index.SchemaVersion == "" {
		index.SchemaVersion = SchemaVersion
	}
	if index.SchemaVersion != SchemaVersion {
		return ErrInvalid
	}
	if !s.policy.AllowUnsigned {
		if len(s.hmacKey) == 0 || strings.TrimSpace(index.Signature) == "" {
			return ErrSignature
		}
		expected, err := SignIndex(s.hmacKey, index)
		if err != nil {
			return err
		}
		if !hmac.Equal([]byte(expected), []byte(index.Signature)) {
			return ErrSignature
		}
	}
	// Normalize digests.
	for i := range index.Entries {
		index.Entries[i].ExtensionID = strings.ToLower(strings.TrimSpace(index.Entries[i].ExtensionID))
		index.Entries[i].PackageDigest = strings.ToLower(strings.TrimSpace(index.Entries[i].PackageDigest))
		index.Entries[i].Channel = strings.ToLower(strings.TrimSpace(index.Entries[i].Channel))
		if index.Entries[i].Channel == "" {
			index.Entries[i].Channel = ChannelStable
		}
	}
	s.mu.Lock()
	clone := index
	clone.Entries = append([]Entry(nil), index.Entries...)
	s.index = &clone
	s.mu.Unlock()
	return nil
}

// SignIndex computes HMAC-SHA256 over canonical JSON body without Signature field.
func SignIndex(key []byte, index Index) (string, error) {
	if len(key) == 0 {
		return "", ErrSignature
	}
	body := index
	body.Signature = ""
	raw, err := json.Marshal(body)
	if err != nil {
		return "", ErrInvalid
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(raw)
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// Resolve finds an installable release and dependency order under operator policy.
func (s *Service) Resolve(extensionID, channel string) (ResolveResult, error) {
	if s == nil {
		return ResolveResult{}, ErrInvalid
	}
	extensionID = strings.ToLower(strings.TrimSpace(extensionID))
	channel = strings.ToLower(strings.TrimSpace(channel))
	if channel == "" {
		channel = ChannelStable
	}
	if !channelAllowed(s.policy.AllowedChannels, channel) {
		return ResolveResult{}, ErrPolicy
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.index == nil {
		return ResolveResult{}, ErrNotFound
	}
	if !s.index.ExpiresAt.IsZero() && s.now().After(s.index.ExpiresAt) {
		return ResolveResult{}, ErrStale
	}
	var match *Entry
	for i := range s.index.Entries {
		entry := &s.index.Entries[i]
		if entry.ExtensionID == extensionID && entry.Channel == channel {
			match = entry
			break
		}
	}
	if match == nil {
		return ResolveResult{}, ErrNotFound
	}
	if match.Withdrawn {
		return ResolveResult{}, ErrWithdrawn
	}
	if blockedByNotice(s.policy, match.Notices) {
		return ResolveResult{}, ErrPolicy
	}
	order := append([]string{}, match.Dependencies...)
	order = append(order, match.ExtensionID)
	return ResolveResult{
		ExtensionID: match.ExtensionID, Version: match.Version,
		PackageDigest: match.PackageDigest, Channel: match.Channel, Order: order,
	}, nil
}

// List returns non-withdrawn entries (or all if includeWithdrawn).
func (s *Service) List(includeWithdrawn bool) []Entry {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.index == nil {
		return nil
	}
	out := make([]Entry, 0, len(s.index.Entries))
	for _, entry := range s.index.Entries {
		if entry.Withdrawn && !includeWithdrawn {
			continue
		}
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ExtensionID != out[j].ExtensionID {
			return out[i].ExtensionID < out[j].ExtensionID
		}
		return out[i].Version < out[j].Version
	})
	return out
}

// Policy returns a copy of operator policy.
func (s *Service) Policy() OperatorPolicy {
	if s == nil {
		return OperatorPolicy{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.policy
}

// DirectUploadAvailable reports offline install fallback.
func (s *Service) DirectUploadAvailable() bool {
	if s == nil {
		return false
	}
	return s.policy.DirectUploadFallback
}

func channelAllowed(allowed []string, channel string) bool {
	for _, item := range allowed {
		if strings.EqualFold(item, channel) {
			return true
		}
	}
	return false
}

func blockedByNotice(policy OperatorPolicy, notices []Notice) bool {
	max := strings.ToLower(strings.TrimSpace(policy.MaxVulnerabilitySeverity))
	if max == "" {
		// Still block critical revocations.
		for _, n := range notices {
			if n.Kind == NoticeRevocation {
				return true
			}
		}
		return false
	}
	rank := severityRank(max)
	for _, n := range notices {
		if n.Kind == NoticeRevocation {
			return true
		}
		if n.Kind == NoticeVulnerability && severityRank(n.Severity) > rank {
			return true
		}
	}
	return false
}

func severityRank(value string) int {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "low":
		return 1
	case "medium":
		return 2
	case "high":
		return 3
	case "critical":
		return 4
	default:
		return 0
	}
}

// FormatNotice is a helper for tests/audit messages.
func FormatNotice(n Notice) string {
	return fmt.Sprintf("%s:%s", n.Kind, n.Summary)
}
