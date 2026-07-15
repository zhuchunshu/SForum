package coreauthority

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"
)

const (
	PublicSchema                    = "public"
	StableCoreViewSchema            = "sforum_core_v1"
	PhysicalAuthorityLockName       = "sforum.extension-database.physical-authority.v1"
	RuntimeStateTable               = "sforum_core_runtime_state"
	KernelCleanupPendingRevokeCode  = "lease_cleanup_pending.revoke"
	KernelCleanupPendingExpiredCode = "lease_cleanup_pending.expired"
	ownerRolePrefix                 = "sforum_core_o_"
	roleHashBytes                   = 20
	postgresNameMaxBytes            = 63
	extensionDatabaseNamespace      = "sforum_ext"
	extensionDatabaseSlugBytes      = 24
	extensionDatabaseHashHexBytes   = 20
	extensionDatabaseLockKeyDomain  = "sforum:extension-database:"
	extensionDatabasePlanRoleDomain = "sforum:extension-database:migration-role:"
	extensionDatabaseLeaseDomain    = "sforum:extension-database:runtime-lease-role:"
)

var (
	ErrInvalidDatabaseIdentity  = errors.New("core database identity is invalid")
	ErrInvalidExtensionIdentity = errors.New("extension database identity is invalid")
	extensionIDPattern          = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{1,80}$`)
	postgresIdentifierPattern   = regexp.MustCompile(`^[a-z_][a-z0-9_]{0,62}$`)
)

type ExtensionDatabaseIdentifiers struct {
	Schema      string
	OwnerRole   string
	RuntimeRole string
	LockKey     int64
}

// OwnerRoleName binds Core ownership to one PostgreSQL database name while
// keeping the cluster-scoped role identifier deterministic and opaque.
func OwnerRoleName(databaseName string) (string, error) {
	if databaseName == "" || len(databaseName) > postgresNameMaxBytes || strings.ContainsRune(databaseName, '\x00') {
		return "", ErrInvalidDatabaseIdentity
	}
	digest := sha256.Sum256([]byte("sforum:core-owner:" + databaseName))
	return ownerRolePrefix + hex.EncodeToString(digest[:])[:roleHashBytes], nil
}

// ExtensionDatabaseIdentifiersFor owns the physical naming contract shared by
// runtime provisioning and pre-migration authority reconciliation.
func ExtensionDatabaseIdentifiersFor(extensionID string) (ExtensionDatabaseIdentifiers, error) {
	if !extensionIDPattern.MatchString(extensionID) {
		return ExtensionDatabaseIdentifiers{}, ErrInvalidExtensionIdentity
	}
	slug := extensionDatabaseSlug(extensionID, extensionDatabaseSlugBytes)
	hash := extensionDatabaseHash(extensionID)
	identifiers := ExtensionDatabaseIdentifiers{
		Schema:      extensionDatabasePhysicalName("s", slug, hash),
		OwnerRole:   extensionDatabasePhysicalName("o", slug, hash),
		RuntimeRole: extensionDatabasePhysicalName("r", slug, hash),
		LockKey:     extensionDatabaseAdvisoryKey(extensionID),
	}
	if identifiers.Schema == identifiers.OwnerRole || identifiers.Schema == identifiers.RuntimeRole ||
		identifiers.OwnerRole == identifiers.RuntimeRole || !validPostgresIdentifier(identifiers.Schema) ||
		!validPostgresIdentifier(identifiers.OwnerRole) || !validPostgresIdentifier(identifiers.RuntimeRole) {
		return ExtensionDatabaseIdentifiers{}, ErrInvalidExtensionIdentity
	}
	return identifiers, nil
}

func ExtensionDatabaseMigrationRoleFor(extensionID string, planDigest string) (string, error) {
	identifiers, err := ExtensionDatabaseIdentifiersFor(extensionID)
	if err != nil || !validDigest(planDigest) {
		return "", ErrInvalidExtensionIdentity
	}
	slug := extensionDatabaseSlug(extensionID, 16)
	hash := extensionDatabaseHash(extensionDatabasePlanRoleDomain + extensionID + ":" + planDigest)
	name := extensionDatabasePhysicalName("m", slug, hash)
	if name == identifiers.OwnerRole || name == identifiers.RuntimeRole || !validPostgresIdentifier(name) {
		return "", ErrInvalidExtensionIdentity
	}
	return name, nil
}

func ExtensionDatabaseRuntimeLeaseRoleFor(
	extensionID string,
	runtimeInstanceID string,
	leaseID string,
) (string, error) {
	identifiers, err := ExtensionDatabaseIdentifiersFor(extensionID)
	if err != nil || runtimeInstanceID == "" || runtimeInstanceID != strings.TrimSpace(runtimeInstanceID) ||
		len(runtimeInstanceID) > 512 || !validDigest(leaseID) {
		return "", ErrInvalidExtensionIdentity
	}
	slug := extensionDatabaseSlug(extensionID, 16)
	hash := extensionDatabaseHash(extensionDatabaseLeaseDomain + extensionID + ":" + runtimeInstanceID + ":" + leaseID)
	name := extensionDatabasePhysicalName("l", slug, hash)
	if name == identifiers.OwnerRole || name == identifiers.RuntimeRole || !validPostgresIdentifier(name) {
		return "", ErrInvalidExtensionIdentity
	}
	return name, nil
}

func IsCoreSchema(schemaName string) bool {
	return schemaName == PublicSchema || schemaName == StableCoreViewSchema
}

// River owns its own migration history and relations even though River v5
// currently installs them in public.
func IsRiverObjectName(name string) bool {
	return strings.HasPrefix(name, "river_") || strings.HasPrefix(name, "_river_")
}

func extensionDatabasePhysicalName(kind string, slug string, hash string) string {
	return extensionDatabaseNamespace + "_" + kind + "_" + slug + "_" + hash[:extensionDatabaseHashHexBytes]
}

func extensionDatabaseSlug(extensionID string, maximum int) string {
	var builder strings.Builder
	builder.Grow(len(extensionID))
	previousSeparator := false
	for _, char := range extensionID {
		separator := char == '.' || char == '-' || char == '_'
		if separator {
			if builder.Len() > 0 && !previousSeparator {
				builder.WriteByte('_')
			}
			previousSeparator = true
			continue
		}
		builder.WriteRune(char)
		previousSeparator = false
	}
	value := strings.Trim(builder.String(), "_")
	if len(value) > maximum {
		value = strings.TrimRight(value[:maximum], "_")
	}
	if value == "" {
		return "extension"
	}
	return value
}

func extensionDatabaseHash(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func extensionDatabaseAdvisoryKey(extensionID string) int64 {
	digest := sha256.Sum256([]byte(extensionDatabaseLockKeyDomain + extensionID))
	return int64(binary.BigEndian.Uint64(digest[:8]))
}

func validPostgresIdentifier(value string) bool {
	return len(value) <= postgresNameMaxBytes && postgresIdentifierPattern.MatchString(value)
}

func validDigest(value string) bool {
	if len(value) != 64 || value != strings.ToLower(value) || value != strings.TrimSpace(value) {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size
}
