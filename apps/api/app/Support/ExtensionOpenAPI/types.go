package extensionopenapi

import (
	"encoding/json"
	"errors"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

var (
	ErrInvalidArtifact  = errors.New("extension openapi: invalid exact artifact")
	ErrInvalidDocument  = errors.New("extension openapi: invalid OpenAPI document")
	ErrUnsafeReference  = errors.New("extension openapi: unsafe package reference")
	ErrCollision        = errors.New("extension openapi: aggregate collision")
	ErrContractMismatch = errors.New("extension openapi: route contract mismatch")
	ErrResourceBudget   = errors.New("extension openapi: static resource budget exceeded")
)

const (
	SecurityPublic        = "public"
	SecurityAuthenticated = "authenticated"
	SecurityHostInherited = "host_inherited"
	SecurityPluginOwned   = "plugin_owned"
	PolicyDisabled        = "disabled"
)

// Artifact is an immutable package snapshot plus Host-owned route policy facts.
// Policies are deliberately outside the plugin manifest: the runtime policy
// registry, rather than plugin prose, remains authoritative for enforcement.
type Artifact struct {
	Root          string
	ExtensionID   string
	Version       string
	PackageDigest string
	Manifest      extensionmanifest.Manifest
	Policies      []RoutePolicy
}

type RoutePolicy struct {
	RouteID     string
	Method      string
	RateLimit   string
	Idempotency string
	Security    string
}

// CoreOperation reserves a Host path/method and operation id during aggregation.
// A plugin may occupy it only through an explicit replace declaration targeting
// the same stable route id.
type CoreOperation struct {
	RouteID     string
	Path        string
	Method      string
	OperationID string
}

type BuildInput struct {
	Core      []CoreOperation
	Artifacts []Artifact
}

type SourceIdentity struct {
	ExtensionID      string `json:"extensionId"`
	ExtensionVersion string `json:"extensionVersion"`
	PackageDigest    string `json:"packageDigest"`
	FragmentID       string `json:"fragmentId"`
	ContractVersion  string `json:"contractVersion"`
	Path             string `json:"path"`
	Digest           string `json:"digest"`
	Namespace        string `json:"namespace"`
}

type GeneratedOperation struct {
	OperationID      string `json:"operationId"`
	RouteID          string `json:"routeId"`
	ContractVersion  string `json:"contractVersion"`
	Path             string `json:"path"`
	Method           string `json:"method"`
	Guard            string `json:"guard"`
	Permission       string `json:"permission,omitempty"`
	RequestSchema    string `json:"requestSchema,omitempty"`
	ResponseSchema   string `json:"responseSchema,omitempty"`
	RateLimit        string `json:"rateLimit"`
	Idempotency      string `json:"idempotency"`
	Security         string `json:"security"`
	ExtensionID      string `json:"extensionId"`
	ExtensionVersion string `json:"extensionVersion"`
	PackageDigest    string `json:"packageDigest"`
	FragmentID       string `json:"fragmentId"`
	Namespace        string `json:"namespace"`
}

// Snapshot exposes copies only. Callers cannot mutate the canonical aggregate
// or the metadata used to derive its revision.
type Snapshot struct {
	revision   string
	document   []byte
	sources    []SourceIdentity
	operations []GeneratedOperation
}

func (s Snapshot) Revision() string { return s.revision }

func (s Snapshot) Document() json.RawMessage {
	return append(json.RawMessage(nil), s.document...)
}

func (s Snapshot) Sources() []SourceIdentity {
	return append([]SourceIdentity(nil), s.sources...)
}

func (s Snapshot) GeneratedClientOperations() []GeneratedOperation {
	return append([]GeneratedOperation(nil), s.operations...)
}
