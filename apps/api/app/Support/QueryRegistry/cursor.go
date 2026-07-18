package queryregistry

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
)

const (
	cursorSchemaVersion  = "sforum.query-cursor@2"
	maximumCursorLength  = 2048
	minimumCursorKeySize = 32
)

var ErrCursorInvalid = errors.New("query registry cursor is invalid")

// CursorClaims is the complete Host-authenticated continuation state. Raw actor
// and locale values are represented only by IsolationDigest so cursors do not
// disclose the caller projection.
type CursorClaims struct {
	SchemaVersion    string `json:"schemaVersion"`
	QueryID          string `json:"queryId"`
	ContractVersion  string `json:"contractVersion"`
	PlanVersion      string `json:"planVersion"`
	ResultSchema     string `json:"resultSchema"`
	ShapeDigest      string `json:"shapeDigest"`
	RegistryRevision uint64 `json:"registryRevision"`
	RegistryDigest   string `json:"registryDigest"`
	ArtifactDigest   string `json:"artifactDigest"`
	IsolationDigest  string `json:"isolationDigest"`
	ExecutionDigest  string `json:"executionDigest"`
	Offset           int    `json:"offset"`
	Limit            int    `json:"limit"`
}

type CursorCodec interface {
	EncodeQueryCursor(CursorClaims) (string, error)
	DecodeQueryCursor(string) (CursorClaims, error)
}

// HMACCursorCodec uses a Host secret to make continuation offsets
// non-malleable. The payload is signed, not encrypted, and therefore contains
// only digests for actor-scoped material.
type HMACCursorCodec struct {
	key []byte
}

func NewHMACCursorCodec(key []byte) (*HMACCursorCodec, error) {
	if len(key) < minimumCursorKeySize {
		return nil, ErrInvalid
	}
	return &HMACCursorCodec{key: append([]byte(nil), key...)}, nil
}

func (c *HMACCursorCodec) EncodeQueryCursor(claims CursorClaims) (string, error) {
	if c == nil || len(c.key) < minimumCursorKeySize || !validCursorClaims(claims) {
		return "", ErrCursorInvalid
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", ErrCursorInvalid
	}
	mac := hmac.New(sha256.New, c.key)
	_, _ = mac.Write(payload)
	value := base64.RawURLEncoding.EncodeToString(payload) + "." +
		base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if len(value) > maximumCursorLength {
		return "", ErrCursorInvalid
	}
	return value, nil
}

func (c *HMACCursorCodec) DecodeQueryCursor(value string) (CursorClaims, error) {
	if c == nil || len(c.key) < minimumCursorKeySize || value == "" || len(value) > maximumCursorLength {
		return CursorClaims{}, ErrCursorInvalid
	}
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return CursorClaims{}, ErrCursorInvalid
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || base64.RawURLEncoding.EncodeToString(payload) != parts[0] {
		return CursorClaims{}, ErrCursorInvalid
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(signature) != sha256.Size || base64.RawURLEncoding.EncodeToString(signature) != parts[1] {
		return CursorClaims{}, ErrCursorInvalid
	}
	mac := hmac.New(sha256.New, c.key)
	_, _ = mac.Write(payload)
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return CursorClaims{}, ErrCursorInvalid
	}
	var claims CursorClaims
	if err := json.Unmarshal(payload, &claims); err != nil || !validCursorClaims(claims) {
		return CursorClaims{}, ErrCursorInvalid
	}
	return claims, nil
}

func validCursorClaims(claims CursorClaims) bool {
	return claims.SchemaVersion == cursorSchemaVersion &&
		idPattern.MatchString(claims.QueryID) && contractPattern.MatchString(claims.ContractVersion) &&
		contractPattern.MatchString(claims.PlanVersion) && validSchemaRef(claims.ResultSchema) &&
		digestPattern.MatchString(claims.ShapeDigest) && digestPattern.MatchString(claims.RegistryDigest) &&
		digestPattern.MatchString(claims.ArtifactDigest) && digestPattern.MatchString(claims.IsolationDigest) &&
		digestPattern.MatchString(claims.ExecutionDigest) &&
		claims.RegistryRevision > 0 && claims.Offset > 0 && claims.Offset <= maximumOffset &&
		claims.Limit > 0 && claims.Limit <= maximumPageLimit
}

func (r *Registry) decodeCursorForPlan(
	state *registryState,
	query QueryContribution,
	pagination PaginationPlan,
	actorFingerprint, policyFingerprint, locale, scope string,
) (CursorClaims, error) {
	if r == nil || r.cursorCodec == nil {
		return CursorClaims{}, fmtContractCursorCodec()
	}
	claims, err := r.cursorCodec.DecodeQueryCursor(pagination.Cursor)
	if err != nil || !validCursorClaims(claims) || claims.QueryID != query.ID || claims.ContractVersion != query.ContractVersion ||
		claims.PlanVersion != query.PlanVersion || claims.ResultSchema != query.ResultSchema ||
		// Revision is local ABA evidence and may differ across nodes that converged
		// on the same immutable graph. Portable cursor authority is the graph digest
		// plus exact artifact, isolation, shape, and execution digests below.
		claims.RegistryDigest != state.digest ||
		claims.ArtifactDigest != cursorArtifactDigest(query.Artifact) ||
		claims.IsolationDigest != cursorIsolationDigest(query.Artifact, actorFingerprint, policyFingerprint, locale, scope) ||
		(pagination.Limit != 0 && pagination.Limit != claims.Limit) {
		return CursorClaims{}, ErrCursorInvalid
	}
	return claims, nil
}

func fmtContractCursorCodec() error {
	return errors.Join(ErrContractInsufficient, ErrCursorInvalid)
}

func (r *Registry) validateCursorExecution(plan QueryPlan, providerDigest, filterPlan string) error {
	if plan.Pagination.Mode != PaginationCursor || plan.Pagination.Cursor == "" {
		return nil
	}
	state := r.load()
	query, ok := state.queries[plan.Query.ID]
	if !ok || query.Artifact != plan.Query.Artifact {
		return ErrCursorInvalid
	}
	claims, err := r.decodeCursorForPlan(
		state, query, plan.Pagination, plan.ActorFingerprint, plan.PolicyFingerprint, plan.Locale, plan.Scope,
	)
	if err != nil || claims.ShapeDigest != plan.ShapeDigest ||
		claims.ExecutionDigest != cursorExecutionDigest(providerDigest, filterPlan) {
		return ErrCursorInvalid
	}
	return nil
}

// EncodeNextCursor creates the only accepted cursor continuation for a released
// execution mapping. Callers cannot choose an arbitrary offset or continue a
// cursor after the provider/result-filter plan changes.
func (r *Registry) EncodeNextCursor(plan QueryPlan, providerDigest, filterPlan string) (string, error) {
	if r == nil || r.cursorCodec == nil {
		return "", fmtContractCursorCodec()
	}
	executionDigest := cursorExecutionDigest(providerDigest, filterPlan)
	if !digestPattern.MatchString(executionDigest) {
		return "", ErrCursorInvalid
	}
	state := r.load()
	query, ok := state.queries[plan.Query.ID]
	if !ok || query.Artifact != plan.Query.Artifact || plan.Pagination.Mode != PaginationCursor ||
		plan.Pagination.Limit <= 0 || plan.Pagination.Offset > maximumOffset-plan.Pagination.Limit {
		return "", ErrCursorInvalid
	}
	if err := r.validatePlanForRelease(state, query, plan); err != nil {
		return "", ErrCursorInvalid
	}
	claims := CursorClaims{
		SchemaVersion: cursorSchemaVersion, QueryID: query.ID, ContractVersion: query.ContractVersion,
		PlanVersion: query.PlanVersion, ResultSchema: query.ResultSchema, ShapeDigest: plan.ShapeDigest,
		RegistryRevision: state.revision, RegistryDigest: state.digest,
		ArtifactDigest: cursorArtifactDigest(query.Artifact),
		IsolationDigest: cursorIsolationDigest(
			query.Artifact, plan.ActorFingerprint, plan.PolicyFingerprint, plan.Locale, plan.Scope,
		),
		ExecutionDigest: executionDigest,
		Offset:          plan.Pagination.Offset + plan.Pagination.Limit, Limit: plan.Pagination.Limit,
	}
	value, err := r.cursorCodec.EncodeQueryCursor(claims)
	if err != nil || value == "" || len(value) > maximumCursorLength {
		return "", ErrCursorInvalid
	}
	decoded, err := r.cursorCodec.DecodeQueryCursor(value)
	if err != nil || decoded != claims {
		return "", ErrCursorInvalid
	}
	return value, nil
}

func cursorArtifactDigest(artifact Artifact) string {
	material := artifact.ExtensionID + "\x00" + artifact.ExtensionVersion + "\x00" +
		artifact.PackageDigest + "\x00" + strconv.FormatInt(artifact.VersionID, 10) + "\x00" +
		artifact.RuntimeInstanceID + "\x00" + strconv.FormatBool(artifact.Core)
	digest := sha256.Sum256([]byte(material))
	return hex.EncodeToString(digest[:])
}

func cursorIsolationDigest(artifact Artifact, actor, policy, locale, scope string) string {
	material := cursorArtifactDigest(artifact) + "\x00" + actor + "\x00" + policy + "\x00" + locale + "\x00" + scope
	digest := sha256.Sum256([]byte(material))
	return hex.EncodeToString(digest[:])
}

func cursorExecutionDigest(providerDigest, filterPlan string) string {
	if !digestPattern.MatchString(providerDigest) || !digestPattern.MatchString(filterPlan) {
		return ""
	}
	digest := sha256.Sum256([]byte(cursorSchemaVersion + "\x00execution\x00" + providerDigest + "\x00" + filterPlan))
	return hex.EncodeToString(digest[:])
}
