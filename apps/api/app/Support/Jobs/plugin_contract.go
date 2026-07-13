package jobs

import (
	"errors"
	"strings"
	"time"
)

const PluginJobEnvelopeVersion = 1

var ErrPluginJobRuntimeStale = errors.New("plugin job runtime contract is stale")

// PluginJobContract identifies the immutable code and data contract that may
// consume one queued plugin job. Runtime epochs and process instances are
// intentionally excluded because a durable job may outlive either value.
type PluginJobContract struct {
	ExtensionID          string `json:"extensionId"`
	ExtensionVersion     string `json:"extensionVersion"`
	ArtifactDigest       string `json:"artifactDigest"`
	JobName              string `json:"jobName"`
	JobContract          string `json:"jobContractVersion"`
	PayloadSchemaID      string `json:"payloadSchemaId"`
	PayloadSchemaVersion string `json:"payloadSchemaVersion"`
}

func (c PluginJobContract) Valid() bool {
	return strings.TrimSpace(c.ExtensionID) != "" &&
		strings.TrimSpace(c.ExtensionVersion) != "" &&
		strings.TrimSpace(c.ArtifactDigest) != "" &&
		strings.TrimSpace(c.JobName) != "" &&
		strings.TrimSpace(c.JobContract) != "" &&
		strings.TrimSpace(c.PayloadSchemaID) != "" &&
		strings.TrimSpace(c.PayloadSchemaVersion) != ""
}

func (c PluginJobContract) Equal(other PluginJobContract) bool {
	return c == other
}

// SplitVersionedSchema separates a manifest reference such as demo.payload@1.
func SplitVersionedSchema(reference string) (schemaID, version string, ok bool) {
	reference = strings.TrimSpace(reference)
	index := strings.LastIndexByte(reference, '@')
	if index <= 0 || index == len(reference)-1 {
		return "", "", false
	}
	return reference[:index], reference[index+1:], true
}

type PluginJobAction string

const (
	PluginJobExecute PluginJobAction = "execute"
	PluginJobDrain   PluginJobAction = "drain"
	PluginJobMigrate PluginJobAction = "migrate"
	PluginJobCancel  PluginJobAction = "cancel"
)

const (
	PluginJobReasonExactMatch        = "plugin_job.contract_exact"
	PluginJobReasonSourceCompatible  = "plugin_job.source_runtime_compatible"
	PluginJobReasonMigrationDeclared = "plugin_job.migration_declared"
	PluginJobReasonIncompatible      = "plugin_job.contract_incompatible"
	PluginJobReasonEnvelopeInvalid   = "plugin_job.envelope_invalid"
	PluginJobReasonTrustGrantStale   = "plugin_job.trust_grant_stale"
	PluginJobReasonRuntimeChanged    = "plugin_job.runtime_changed"
)

type PluginJobMigration struct {
	ID   string
	From PluginJobContract
	To   PluginJobContract
}

type PluginJobDecision struct {
	Action      PluginJobAction
	Reason      string
	MigrationID string
}

// DecidePluginJobExecution allows execution only for an exact immutable
// contract. Payload migration is a separately declared lifecycle operation.
func DecidePluginJobExecution(queued, target PluginJobContract, migrations []PluginJobMigration) PluginJobDecision {
	if queued.Valid() && queued.Equal(target) {
		return PluginJobDecision{Action: PluginJobExecute, Reason: PluginJobReasonExactMatch}
	}
	if migrationID := exactPluginJobMigration(queued, target, migrations); migrationID != "" {
		return PluginJobDecision{Action: PluginJobMigrate, Reason: PluginJobReasonMigrationDeclared, MigrationID: migrationID}
	}
	return PluginJobDecision{Action: PluginJobCancel, Reason: PluginJobReasonIncompatible}
}

type PluginJobUpgrade struct {
	Queued                 PluginJobContract
	Source                 PluginJobContract
	Target                 PluginJobContract
	SourceRuntimeAvailable bool
	Migrations             []PluginJobMigration
}

// DecidePluginJobUpgrade is the inspectable policy used before switching an
// upgraded runtime: target-compatible jobs execute, source-compatible jobs
// drain, exact declared migrations migrate, and everything else is cancelled.
func DecidePluginJobUpgrade(input PluginJobUpgrade) PluginJobDecision {
	if input.Queued.Valid() && input.Queued.Equal(input.Target) {
		return PluginJobDecision{Action: PluginJobExecute, Reason: PluginJobReasonExactMatch}
	}
	if input.SourceRuntimeAvailable && input.Queued.Valid() && input.Queued.Equal(input.Source) {
		return PluginJobDecision{Action: PluginJobDrain, Reason: PluginJobReasonSourceCompatible}
	}
	if migrationID := exactPluginJobMigration(input.Queued, input.Target, input.Migrations); migrationID != "" {
		return PluginJobDecision{Action: PluginJobMigrate, Reason: PluginJobReasonMigrationDeclared, MigrationID: migrationID}
	}
	return PluginJobDecision{Action: PluginJobCancel, Reason: PluginJobReasonIncompatible}
}

func exactPluginJobMigration(from, to PluginJobContract, migrations []PluginJobMigration) string {
	if !from.Valid() || !to.Valid() {
		return ""
	}
	selected := ""
	for _, migration := range migrations {
		id := strings.TrimSpace(migration.ID)
		if id == "" || !migration.From.Equal(from) || !migration.To.Equal(to) {
			continue
		}
		if selected == "" || id < selected {
			selected = id
		}
	}
	return selected
}

// PluginJobInvocation is the runtime-facing value after River compatibility
// checks have succeeded.
type PluginJobInvocation struct {
	JobID        int64
	Attempt      int
	Contract     PluginJobContract
	TrustGrantID string
	Payload      map[string]any
	EnqueuedAt   time.Time
}
