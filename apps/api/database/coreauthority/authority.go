package coreauthority

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

const (
	PublicSchema                    = "public"
	StableCoreViewSchema            = "sforum_core_v1"
	PhysicalAuthorityLockName       = "sforum.extension-database.physical-authority.v1"
	KernelCleanupPendingRevokeCode  = "lease_cleanup_pending.revoke"
	KernelCleanupPendingExpiredCode = "lease_cleanup_pending.expired"
	ownerRolePrefix                 = "sforum_core_o_"
	roleHashBytes                   = 20
	postgresNameMaxBytes            = 63
)

var ErrInvalidDatabaseIdentity = errors.New("core database identity is invalid")

// OwnerRoleName binds Core ownership to one PostgreSQL database name while
// keeping the cluster-scoped role identifier deterministic and opaque.
func OwnerRoleName(databaseName string) (string, error) {
	if databaseName == "" || len(databaseName) > postgresNameMaxBytes || strings.ContainsRune(databaseName, '\x00') {
		return "", ErrInvalidDatabaseIdentity
	}
	digest := sha256.Sum256([]byte("sforum:core-owner:" + databaseName))
	return ownerRolePrefix + hex.EncodeToString(digest[:])[:roleHashBytes], nil
}

func IsCoreSchema(schemaName string) bool {
	return schemaName == PublicSchema || schemaName == StableCoreViewSchema
}

// River owns its own migration history and relations even though River v5
// currently installs them in public.
func IsRiverObjectName(name string) bool {
	return strings.HasPrefix(name, "river_") || strings.HasPrefix(name, "_river_")
}
