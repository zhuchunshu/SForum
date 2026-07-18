package queryregistry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

// SemanticCacheInvalidator is the narrow Host-owned boundary used by durable
// mutation workers. Callers provide stable logical tags, never physical Redis
// keys or exact-artifact identity.
type SemanticCacheInvalidator interface {
	InvalidateOwnerTags(context.Context, string, []string) (uint64, error)
}

// CanonicalSemanticCacheTags validates one owner's logical invalidation set and
// returns a sorted clone suitable for a durable job envelope.
func CanonicalSemanticCacheTags(owner string, tags []string) ([]string, error) {
	owner = strings.ToLower(strings.TrimSpace(owner))
	if !idPattern.MatchString(owner) || owner == "core" || len(tags) == 0 {
		return nil, ErrInvalid
	}
	canonical, err := normalizeOwnedCacheTags(owner, tags)
	if err != nil {
		return nil, err
	}
	sort.Strings(canonical)
	return canonical, nil
}

func sharedSemanticCacheTag(owner, logicalTag string) string {
	digest := sha256.Sum256([]byte(
		resultCacheSchemaVersion + "\x00invalidation\x00" + owner + "\x00" + logicalTag,
	))
	return "query:shared:" + hex.EncodeToString(digest[:16])
}

func sharedSemanticCacheTags(owner string, logicalTags []string) ([]string, error) {
	canonical, err := CanonicalSemanticCacheTags(owner, logicalTags)
	if err != nil {
		return nil, err
	}
	shared := make([]string, len(canonical))
	for index, tag := range canonical {
		shared[index] = sharedSemanticCacheTag(owner, tag)
	}
	return shared, nil
}

// InvalidateOwnerTags maps canonical logical tags to the same shared tags used
// by every actor/locale/version-specific cached result before rotating Redis.
func (c *RedisQueryResultCache) InvalidateOwnerTags(
	ctx context.Context,
	owner string,
	logicalTags []string,
) (uint64, error) {
	shared, err := sharedSemanticCacheTags(strings.ToLower(strings.TrimSpace(owner)), logicalTags)
	if err != nil {
		return 0, err
	}
	return c.InvalidateTags(ctx, shared)
}
