package queryregistry

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
)

func (c *RedisQueryResultCache) decodeEnvelope(
	encoded []byte,
	key string,
) (redisQueryCacheEnvelope, CachedQueryResult, error) {
	if len(encoded) == 0 || len(encoded) > redisQueryCacheMaximumEnvelope || validateRedisQueryCacheJSON(encoded) != nil {
		return redisQueryCacheEnvelope{}, CachedQueryResult{}, ErrCachePoisoned
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	var envelope redisQueryCacheEnvelope
	if err := decoder.Decode(&envelope); err != nil {
		return redisQueryCacheEnvelope{}, CachedQueryResult{}, ErrCachePoisoned
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return redisQueryCacheEnvelope{}, CachedQueryResult{}, ErrCachePoisoned
	}
	if envelope.Version != redisQueryCacheEnvelopeVersion || !validRedisQueryCacheDigest(envelope.TagDigest) {
		return redisQueryCacheEnvelope{}, CachedQueryResult{}, ErrCachePoisoned
	}
	value, err := validateRedisCachedQueryResult(envelope.Result.value(), key)
	if err != nil {
		return redisQueryCacheEnvelope{}, CachedQueryResult{}, ErrCachePoisoned
	}
	return envelope, value, nil
}

func validateRedisCachedQueryResult(value CachedQueryResult, key string) (CachedQueryResult, error) {
	if value.SchemaVersion != resultCacheSchemaVersion || value.CacheKey != key ||
		value.RegistryRevision == 0 || !validRedisQueryCacheDigest(value.RegistryDigest) ||
		!validRedisQueryCacheDigest(value.ShapeDigest) || !validRedisQueryCacheDigest(value.FilterPlan) ||
		len(value.QueryID) > maxIDLength || !idPattern.MatchString(value.QueryID) ||
		len(value.ContractVersion) > maxSchemaRefLength || !contractPattern.MatchString(value.ContractVersion) ||
		len(value.PlanVersion) > maxSchemaRefLength || !contractPattern.MatchString(value.PlanVersion) ||
		!validSchemaRef(value.ResultSchema) || !validRedisQueryCacheDigest(value.ProviderDigest) ||
		len(value.CacheTags) == 0 || len(value.CacheTags) > redisQueryCacheMaximumTags ||
		!validRedisQueryCachePage(value.Page) {
		return CachedQueryResult{}, ErrExecutionInvalid
	}
	seenTags := make(map[string]struct{}, len(value.CacheTags))
	for _, tag := range value.CacheTags {
		if !validLogicalQueryCacheTag(tag) {
			return CachedQueryResult{}, ErrExecutionInvalid
		}
		if _, duplicate := seenTags[tag]; duplicate {
			return CachedQueryResult{}, ErrExecutionInvalid
		}
		seenTags[tag] = struct{}{}
	}
	artifact, err := rehydrateRedisCachedArtifact(value.Artifact)
	if err != nil {
		return CachedQueryResult{}, ErrExecutionInvalid
	}
	rows, _, err := cloneRowsBounded(value.Rows, maximumResultBytes)
	if err != nil {
		return CachedQueryResult{}, err
	}
	value.Artifact = artifact
	value.Rows = rows
	value.CacheTags = slices.Clone(value.CacheTags)
	return value, nil
}

func validRedisQueryCachePage(page QueryResultPage) bool {
	return validPagination(page.Mode) && page.Limit >= 1 && page.Limit <= maximumPageLimit &&
		page.Offset >= 0 && page.Offset <= maximumOffset && page.NextOffset >= 0 &&
		page.NextOffset <= maximumOffset && len(page.NextCursor) <= 8<<10
}

func rehydrateRedisCachedArtifact(input Artifact) (Artifact, error) {
	var (
		artifact Artifact
		err      error
	)
	if input.Core {
		artifact, err = NewCoreArtifact(input.ExtensionID, input.ExtensionVersion, input.PackageDigest)
	} else {
		artifact, err = normalizeArtifact(input)
	}
	if err != nil {
		return Artifact{}, err
	}
	input.coreSeal = artifact.coreSeal
	if input != artifact {
		return Artifact{}, ErrInvalid
	}
	return artifact, nil
}

func redisCachedQueryResultFromValue(value CachedQueryResult) redisCachedQueryResult {
	return redisCachedQueryResult{
		SchemaVersion: value.SchemaVersion, CacheKey: value.CacheKey,
		RegistryRevision: value.RegistryRevision, RegistryDigest: value.RegistryDigest,
		ShapeDigest: value.ShapeDigest, FilterPlan: value.FilterPlan, QueryID: value.QueryID,
		ContractVersion: value.ContractVersion, PlanVersion: value.PlanVersion,
		ResultSchema: value.ResultSchema, Artifact: value.Artifact, ProviderDigest: value.ProviderDigest,
		Page: value.Page, Rows: value.Rows, CacheTags: slices.Clone(value.CacheTags),
	}
}

func (value redisCachedQueryResult) value() CachedQueryResult {
	return CachedQueryResult{
		SchemaVersion: value.SchemaVersion, CacheKey: value.CacheKey,
		RegistryRevision: value.RegistryRevision, RegistryDigest: value.RegistryDigest,
		ShapeDigest: value.ShapeDigest, FilterPlan: value.FilterPlan, QueryID: value.QueryID,
		ContractVersion: value.ContractVersion, PlanVersion: value.PlanVersion,
		ResultSchema: value.ResultSchema, Artifact: value.Artifact, ProviderDigest: value.ProviderDigest,
		Page: value.Page, Rows: value.Rows, CacheTags: slices.Clone(value.CacheTags),
	}
}

func validateRedisQueryCacheJSON(encoded []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	if err := consumeRedisQueryCacheJSONValue(decoder, 1); err != nil {
		return err
	}
	if token, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err != nil {
			return err
		}
		return fmt.Errorf("trailing JSON token %v", token)
	}
	return nil
}

func consumeRedisQueryCacheJSONValue(decoder *json.Decoder, depth int) error {
	if depth > redisQueryCacheMaximumJSONDepth {
		return errors.New("query result cache JSON nesting exceeds Host bounds")
	}
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, structured := token.(json.Delim)
	if !structured {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("query result cache object key is not a string")
			}
			if _, duplicate := seen[key]; duplicate {
				return fmt.Errorf("duplicate JSON object key %q", key)
			}
			seen[key] = struct{}{}
			if err := consumeRedisQueryCacheJSONValue(decoder, depth+1); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil || closing != json.Delim('}') {
			return errors.New("query result cache object is not terminated")
		}
	case '[':
		for decoder.More() {
			if err := consumeRedisQueryCacheJSONValue(decoder, depth+1); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil || closing != json.Delim(']') {
			return errors.New("query result cache array is not terminated")
		}
	default:
		return errors.New("query result cache JSON has an invalid delimiter")
	}
	return nil
}
