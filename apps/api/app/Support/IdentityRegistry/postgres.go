package identityregistry

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore persists Identity Registry ownership tips and Host role
// suggestion decisions. It never writes permissions, role_permissions,
// user_permission_overrides, or user_roles.
type PostgresStore struct {
	pool *pgxpool.Pool
}

const maxRoleSuggestionDecisionAttempts = 3

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) LoadDurableState(ctx context.Context) (DurableState, error) {
	if s == nil || s.pool == nil || ctx == nil {
		return DurableState{}, ErrInvalid
	}

	// Owners and declaration tips are one restart unit. A shared snapshot avoids
	// observing a lifecycle commit between the two queries and restoring only
	// half of its permanent ownership graph.
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.RepeatableRead,
		AccessMode: pgx.ReadOnly,
	})
	if err != nil {
		return DurableState{}, mapStoreError(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	ownerRows, err := tx.Query(ctx, `
			SELECT identity_kind, stable_id, owner_extension_id, claimed_at
			FROM extension_identity_registry_owners
			ORDER BY identity_kind ASC, stable_id ASC
		`)
	if err != nil {
		return DurableState{}, mapStoreError(err)
	}
	defer ownerRows.Close()

	owners := make([]DurableOwner, 0)
	for ownerRows.Next() {
		var owner DurableOwner
		if scanErr := ownerRows.Scan(
			&owner.IdentityKind, &owner.StableID, &owner.OwnerExtensionID, &owner.ClaimedAt,
		); scanErr != nil {
			return DurableState{}, mapStoreError(scanErr)
		}
		owners = append(owners, owner)
	}
	if err := ownerRows.Err(); err != nil {
		ownerRows.Close()
		return DurableState{}, mapStoreError(err)
	}
	ownerRows.Close()

	tipRows, err := tx.Query(ctx, `
			SELECT DISTINCT ON (identity_kind, stable_id)
			identity_kind, stable_id, owner_extension_id, revision, registry_state,
			extension_version_id, extension_version, package_digest, contract_version,
			declaration_digest, actor_user_id, audit_event_id, created_at
		FROM extension_identity_registry_declarations
		ORDER BY identity_kind ASC, stable_id ASC, revision DESC
	`)
	if err != nil {
		return DurableState{}, mapStoreError(err)
	}
	defer tipRows.Close()

	tips := make([]DurableDeclarationTip, 0)
	for tipRows.Next() {
		var tip DurableDeclarationTip
		var actorUserID, auditEventID *int64
		if scanErr := tipRows.Scan(
			&tip.IdentityKind, &tip.StableID, &tip.OwnerExtensionID, &tip.Revision,
			&tip.RegistryState, &tip.ExtensionVersionID, &tip.ExtensionVersion,
			&tip.PackageDigest, &tip.ContractVersion, &tip.DeclarationDigest,
			&actorUserID, &auditEventID, &tip.CreatedAt,
		); scanErr != nil {
			return DurableState{}, mapStoreError(scanErr)
		}
		if actorUserID != nil {
			tip.ActorUserID = *actorUserID
		}
		if auditEventID != nil {
			tip.AuditEventID = *auditEventID
		}
		tips = append(tips, tip)
	}
	if err := tipRows.Err(); err != nil {
		tipRows.Close()
		return DurableState{}, mapStoreError(err)
	}
	tipRows.Close()

	if err := tx.Commit(ctx); err != nil {
		return DurableState{}, mapStoreError(err)
	}
	return DurableState{Owners: owners, Tips: tips}, nil
}

func (s *PostgresStore) ListRoleSuggestions(
	ctx context.Context,
	filter RoleSuggestionFilter,
) ([]RoleSuggestion, error) {
	if s == nil || s.pool == nil || ctx == nil {
		return nil, ErrInvalid
	}
	filter = normalizeRoleSuggestionFilter(filter)
	if filter.ApprovalState != "" &&
		filter.ApprovalState != RoleSuggestionPending &&
		filter.ApprovalState != RoleSuggestionApproved &&
		filter.ApprovalState != RoleSuggestionRejected {
		return nil, ErrInvalid
	}

	rows, err := s.pool.Query(ctx, roleSuggestionSelectSQL+`
		WHERE ($1 = '' OR approval_state = $1)
		  AND ($2 = '' OR role_key = $2)
		  AND ($3 = '' OR permission_key = $3)
		  AND ($4 = '' OR owner_extension_id = $4)
		ORDER BY approval_state ASC, role_key ASC, permission_key ASC, id ASC
		LIMIT $5
	`, filter.ApprovalState, filter.RoleKey, filter.PermissionKey, filter.OwnerExtensionID, filter.Limit)
	if err != nil {
		return nil, mapStoreError(err)
	}
	defer rows.Close()

	result := make([]RoleSuggestion, 0)
	for rows.Next() {
		suggestion, scanErr := scanRoleSuggestion(rows)
		if scanErr != nil {
			return nil, mapStoreError(scanErr)
		}
		result = append(result, suggestion)
	}
	if err := rows.Err(); err != nil {
		return nil, mapStoreError(err)
	}
	return result, nil
}

func (s *PostgresStore) DecideRoleSuggestion(
	ctx context.Context,
	input DecideRoleSuggestionInput,
) (RoleSuggestion, error) {
	if s == nil || s.pool == nil || ctx == nil || !validRoleSuggestionDecision(input) {
		return RoleSuggestion{}, ErrInvalid
	}
	approvalState := strings.ToLower(strings.TrimSpace(input.ApprovalState))
	action := roleSuggestionDecisionAction(approvalState)
	if action == "" {
		return RoleSuggestion{}, ErrInvalid
	}
	for attempt := 0; attempt < maxRoleSuggestionDecisionAttempts; attempt++ {
		updated, err := s.decideRoleSuggestionOnce(ctx, input, approvalState, action)
		if !errors.Is(err, errRetryableIdentityRegistryTransaction) {
			return updated, err
		}
		if ctx.Err() != nil {
			return RoleSuggestion{}, mapStoreError(ctx.Err())
		}
	}
	// Repeated PostgreSQL serialization/deadlock aborts are exposed as a stable
	// CAS conflict. The caller must refresh before attempting another decision.
	return RoleSuggestion{}, ErrRevisionConflict
}

func (s *PostgresStore) decideRoleSuggestionOnce(
	ctx context.Context,
	input DecideRoleSuggestionInput,
	approvalState string,
	action string,
) (RoleSuggestion, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return RoleSuggestion{}, mapStoreError(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	// 先锁 pending 行并校验 revision，再写 audit，保证 actor 与决策同一事务。
	current, err := lockPendingRoleSuggestion(ctx, tx, input.ID)
	if err != nil {
		return RoleSuggestion{}, err
	}
	if current.Revision != input.ExpectedRevision {
		return RoleSuggestion{}, ErrRevisionConflict
	}

	auditEventID, err := insertRoleSuggestionAuditEvent(ctx, tx, action, input.ActorUserID, current, approvalState)
	if err != nil {
		return RoleSuggestion{}, err
	}

	updated, err := casDecideRoleSuggestion(ctx, tx, input.ID, input.ExpectedRevision, approvalState, input.ActorUserID, auditEventID)
	if err != nil {
		return RoleSuggestion{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return RoleSuggestion{}, mapStoreError(err)
	}
	return updated, nil
}

const roleSuggestionSelectSQL = `
	SELECT id, permission_key, owner_extension_id, extension_version_id, extension_version,
	       package_digest, permission_contract_version, declaration_digest, role_key,
	       approval_state, revision, decided_by_user_id, decision_audit_event_id,
	       decided_at, created_at, updated_at
	FROM extension_permission_role_suggestions
`

type roleSuggestionScanner interface {
	Scan(dest ...any) error
}

func scanRoleSuggestion(scanner roleSuggestionScanner) (RoleSuggestion, error) {
	var suggestion RoleSuggestion
	var decidedBy, decisionAudit *int64
	var decidedAt *time.Time
	err := scanner.Scan(
		&suggestion.ID, &suggestion.PermissionKey, &suggestion.OwnerExtensionID,
		&suggestion.ExtensionVersionID, &suggestion.ExtensionVersion, &suggestion.PackageDigest,
		&suggestion.PermissionContractVersion, &suggestion.DeclarationDigest, &suggestion.RoleKey,
		&suggestion.ApprovalState, &suggestion.Revision, &decidedBy, &decisionAudit,
		&decidedAt, &suggestion.CreatedAt, &suggestion.UpdatedAt,
	)
	if err != nil {
		return RoleSuggestion{}, err
	}
	if decidedBy != nil {
		suggestion.DecidedByUserID = *decidedBy
	}
	if decisionAudit != nil {
		suggestion.DecisionAuditEventID = *decisionAudit
	}
	if decidedAt != nil {
		value := *decidedAt
		suggestion.DecidedAt = &value
	}
	return suggestion, nil
}

func lockPendingRoleSuggestion(ctx context.Context, tx pgx.Tx, id int64) (RoleSuggestion, error) {
	suggestion, err := scanRoleSuggestion(tx.QueryRow(ctx, roleSuggestionSelectSQL+`
		WHERE id = $1
		FOR UPDATE
	`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return RoleSuggestion{}, ErrNotFound
	}
	if err != nil {
		return RoleSuggestion{}, mapStoreError(err)
	}
	if suggestion.ApprovalState != RoleSuggestionPending {
		return RoleSuggestion{}, ErrRevisionConflict
	}
	return suggestion, nil
}

func insertRoleSuggestionAuditEvent(
	ctx context.Context,
	tx pgx.Tx,
	action string,
	actorUserID int64,
	suggestion RoleSuggestion,
	approvalState string,
) (int64, error) {
	metadata, err := json.Marshal(map[string]any{
		"suggestionId":              suggestion.ID,
		"permissionKey":             suggestion.PermissionKey,
		"ownerExtensionId":          suggestion.OwnerExtensionID,
		"extensionVersionId":        suggestion.ExtensionVersionID,
		"extensionVersion":          suggestion.ExtensionVersion,
		"packageDigest":             suggestion.PackageDigest,
		"permissionContractVersion": suggestion.PermissionContractVersion,
		"declarationDigest":         suggestion.DeclarationDigest,
		"roleKey":                   suggestion.RoleKey,
		"expectedRevision":          suggestion.Revision,
		"approvalState":             approvalState,
	})
	if err != nil {
		return 0, ErrInvalid
	}
	var auditEventID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO audit_events (actor_user_id, action, metadata)
		VALUES ($1, $2, $3::jsonb)
		RETURNING id
	`, actorUserID, action, string(metadata)).Scan(&auditEventID)
	if err != nil {
		return 0, mapStoreError(err)
	}
	if auditEventID <= 0 {
		return 0, ErrInvalid
	}
	return auditEventID, nil
}

func casDecideRoleSuggestion(
	ctx context.Context,
	tx pgx.Tx,
	id, expectedRevision int64,
	approvalState string,
	actorUserID, auditEventID int64,
) (RoleSuggestion, error) {
	// decided_at/updated_at 由迁移 trigger 统一赋值，避免应用时钟漂移。
	suggestion, err := scanRoleSuggestion(tx.QueryRow(ctx, `
		UPDATE extension_permission_role_suggestions
		SET approval_state = $2,
		    revision = revision + 1,
		    decided_by_user_id = $3,
		    decision_audit_event_id = $4
		WHERE id = $1
		  AND revision = $5
		  AND approval_state = 'pending'
		RETURNING id, permission_key, owner_extension_id, extension_version_id, extension_version,
		          package_digest, permission_contract_version, declaration_digest, role_key,
		          approval_state, revision, decided_by_user_id, decision_audit_event_id,
		          decided_at, created_at, updated_at
	`, id, approvalState, actorUserID, auditEventID, expectedRevision))
	if errors.Is(err, pgx.ErrNoRows) {
		return RoleSuggestion{}, ErrRevisionConflict
	}
	if err != nil {
		return RoleSuggestion{}, mapStoreError(err)
	}
	return suggestion, nil
}

// mapStoreError collapses PostgreSQL details into stable sentinels so callers
// never surface raw constraint/trigger text to HTTP or logs by default.
func mapStoreError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, context.Canceled):
		return context.Canceled
	case errors.Is(err, context.DeadlineExceeded):
		return context.DeadlineExceeded
	case errors.Is(err, ErrInvalid),
		errors.Is(err, ErrNotFound),
		errors.Is(err, ErrRevisionConflict),
		errors.Is(err, ErrStale),
		errors.Is(err, ErrTargetConflict):
		return err
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return errIdentityRegistryStore
	}
	switch pgErr.Code {
	case "40001", "40P01":
		return errRetryableIdentityRegistryTransaction
	case "23505":
		return ErrRevisionConflict
	case "23503":
		// The only repository write with a foreign key is audit.actor_user_id.
		// A concurrently removed actor is invalid authority, not an internal 500.
		return ErrInvalid
	case "P0001", "23514":
		message := strings.ToLower(pgErr.Message)
		switch {
		case strings.Contains(message, "decision is stale"):
			return ErrStale
		case strings.Contains(message, "target is unavailable"):
			return ErrTargetConflict
		case strings.Contains(message, "host cas evidence"):
			return ErrRevisionConflict
		case strings.Contains(message, "actor is not active"):
			return ErrInvalid
		default:
			return ErrInvalid
		}
	default:
		return errIdentityRegistryStore
	}
}

var errIdentityRegistryStore = errors.New("identity registry store operation failed")
var errRetryableIdentityRegistryTransaction = errors.New("identity registry transaction should retry")

var _ Store = (*PostgresStore)(nil)
