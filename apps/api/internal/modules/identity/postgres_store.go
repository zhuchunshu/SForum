package identity

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	store "github.com/inkedus/sforum/apps/api/internal/store/sqlc"
)

type PostgresStore struct {
	pool    *pgxpool.Pool
	queries *store.Queries
}

type postgresTxStore struct {
	queries *store.Queries
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool, queries: store.New(pool)}
}

func (s *PostgresStore) WithBootstrapTx(ctx context.Context, fn func(context.Context, TxStore) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin identity tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtext('sforum.identity.bootstrap'))"); err != nil {
		return fmt.Errorf("lock identity bootstrap: %w", err)
	}

	txStore := &postgresTxStore{queries: s.queries.WithTx(tx)}
	if err := fn(ctx, txStore); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit identity tx: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetCurrentUser(ctx context.Context, userID int64) (CurrentUser, error) {
	row, err := s.queries.GetCurrentUser(ctx, userID)
	if err != nil {
		return CurrentUser{}, fmt.Errorf("get current user: %w", err)
	}

	current := CurrentUser{
		ID:                  row.ID,
		Username:            row.Username,
		DisplayName:         row.DisplayName,
		Locale:              row.Locale,
		Status:              UserStatus(row.Status),
		IsInitialSuperAdmin: row.IsInitialSuperAdmin,
	}
	if err := s.loadCurrentUserAccess(ctx, &current); err != nil {
		return CurrentUser{}, err
	}
	return current, nil
}

func (s *PostgresStore) GetCredentialByLogin(ctx context.Context, login string) (CredentialUser, error) {
	row, err := s.queries.GetUserCredentialByLogin(ctx, login)
	if err != nil {
		return CredentialUser{}, fmt.Errorf("get user credential: %w", err)
	}

	current := CurrentUser{
		ID:                  row.ID,
		Username:            row.Username,
		DisplayName:         row.DisplayName,
		Locale:              row.Locale,
		Status:              UserStatus(row.Status),
		IsInitialSuperAdmin: row.IsInitialSuperAdmin,
	}
	if err := s.loadCurrentUserAccess(ctx, &current); err != nil {
		return CredentialUser{}, err
	}
	return CredentialUser{CurrentUser: current, PasswordHash: row.PasswordHash}, nil
}

func (s *PostgresStore) LoadActor(ctx context.Context, userID int64) (Actor, error) {
	current, err := s.GetCurrentUser(ctx, userID)
	if err != nil {
		return Actor{}, err
	}

	permissions := make(map[string]bool, len(current.Permissions))
	for _, key := range current.Permissions {
		permissions[key] = true
	}
	return Actor{
		ID:          userID,
		Status:      current.Status,
		RoleKeys:    current.RoleKeys,
		Permissions: permissions,
	}, nil
}

func (s *PostgresStore) ListRoles(ctx context.Context) ([]Role, error) {
	rows, err := s.queries.ListRoles(ctx)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}

	roles := make([]Role, 0, len(rows))
	for _, row := range rows {
		roles = append(roles, mapRole(row.ID, row.Key, row.Alias, row.Description, row.IsSystem, row.IsDefault, row.IsDeletable, row.IsEnabled))
	}
	return roles, nil
}

func (s *PostgresStore) CreateRole(ctx context.Context, input RoleInput) (Role, error) {
	row, err := s.queries.CreateRole(ctx, store.CreateRoleParams{
		Key:         input.Key,
		Alias:       input.Alias,
		Description: input.Description,
	})
	if err != nil {
		return Role{}, fmt.Errorf("create role: %w", err)
	}
	return mapRole(row.ID, row.Key, row.Alias, row.Description, row.IsSystem, row.IsDefault, row.IsDeletable, row.IsEnabled), nil
}

func (s *PostgresStore) UpdateRole(ctx context.Context, roleKey string, input RoleInput) (Role, error) {
	row, err := s.queries.UpdateRoleAlias(ctx, store.UpdateRoleAliasParams{
		Key:         roleKey,
		Alias:       input.Alias,
		Description: input.Description,
	})
	if err != nil {
		return Role{}, fmt.Errorf("update role: %w", err)
	}
	return mapRole(row.ID, row.Key, row.Alias, row.Description, row.IsSystem, row.IsDefault, row.IsDeletable, row.IsEnabled), nil
}

func (s *PostgresStore) DeleteRole(ctx context.Context, roleKey string) error {
	if err := s.queries.DeleteRoleByKey(ctx, roleKey); err != nil {
		return fmt.Errorf("delete role: %w", err)
	}
	return nil
}

func (s *PostgresStore) ReplaceRolePermissions(ctx context.Context, actorUserID int64, roleKey string, permissions []string) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin role permissions tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	queries := s.queries.WithTx(tx)
	role, err := queries.GetRoleByKey(ctx, roleKey)
	if err != nil {
		return fmt.Errorf("get role for permissions: %w", err)
	}
	if err := queries.DeleteRolePermissions(ctx, role.ID); err != nil {
		return fmt.Errorf("delete role permissions: %w", err)
	}
	for _, permission := range permissions {
		if err := queries.AddRolePermission(ctx, store.AddRolePermissionParams{
			RoleID:        role.ID,
			PermissionKey: permission,
		}); err != nil {
			return fmt.Errorf("add role permission %s: %w", permission, err)
		}
	}

	metadata := auditMetadata(map[string]any{"roleKey": roleKey, "permissions": permissions})
	if err := queries.CreateAuditEvent(ctx, store.CreateAuditEventParams{
		ActorUserID:  nullableInt8(actorUserID),
		TargetUserID: pgtype.Int8{},
		Action:       "role.permissions.replace",
		Metadata:     metadata,
	}); err != nil {
		return fmt.Errorf("audit role permissions: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit role permissions tx: %w", err)
	}
	return nil
}

func (s *PostgresStore) loadCurrentUserAccess(ctx context.Context, current *CurrentUser) error {
	roleKeys, err := s.queries.ListUserRoleKeys(ctx, current.ID)
	if err != nil {
		return fmt.Errorf("list current user roles: %w", err)
	}
	permissions, err := s.queries.ListUserPermissions(ctx, current.ID)
	if err != nil {
		return fmt.Errorf("list current user permissions: %w", err)
	}
	current.RoleKeys = roleKeys
	current.Permissions = permissions
	return nil
}

func (s *postgresTxStore) AnyUserExists(ctx context.Context) (bool, error) {
	return s.queries.AnyUserExists(ctx)
}

func (s *postgresTxStore) CreateUser(ctx context.Context, input CreateUserInput) (CurrentUser, error) {
	row, err := s.queries.CreateUser(ctx, store.CreateUserParams{
		Username:            input.Username,
		Email:               input.Email,
		DisplayName:         input.DisplayName,
		Locale:              input.Locale,
		IsInitialSuperAdmin: input.IsInitialSuperAdmin,
	})
	if err != nil {
		return CurrentUser{}, fmt.Errorf("create user: %w", err)
	}
	return CurrentUser{
		ID:                  row.ID,
		Username:            row.Username,
		DisplayName:         row.DisplayName,
		Locale:              row.Locale,
		Status:              UserStatus(row.Status),
		IsInitialSuperAdmin: row.IsInitialSuperAdmin,
	}, nil
}

func (s *postgresTxStore) CreateCredential(ctx context.Context, userID int64, passwordHash string) error {
	if err := s.queries.CreateUserCredential(ctx, store.CreateUserCredentialParams{
		UserID:       userID,
		PasswordHash: passwordHash,
	}); err != nil {
		return fmt.Errorf("create user credential: %w", err)
	}
	return nil
}

func (s *postgresTxStore) GetDefaultRole(ctx context.Context) (Role, error) {
	row, err := s.queries.GetDefaultRole(ctx)
	if err != nil {
		return Role{}, fmt.Errorf("get default role: %w", err)
	}
	return mapRole(row.ID, row.Key, row.Alias, row.Description, row.IsSystem, row.IsDefault, row.IsDeletable, row.IsEnabled), nil
}

func (s *postgresTxStore) GetRole(ctx context.Context, roleKey string) (Role, error) {
	row, err := s.queries.GetRoleByKey(ctx, roleKey)
	if err != nil {
		return Role{}, fmt.Errorf("get role: %w", err)
	}
	return mapRole(row.ID, row.Key, row.Alias, row.Description, row.IsSystem, row.IsDefault, row.IsDeletable, row.IsEnabled), nil
}

func (s *postgresTxStore) AssignRole(ctx context.Context, userID int64, roleID int64) error {
	if err := s.queries.AssignRoleToUser(ctx, store.AssignRoleToUserParams{
		UserID: userID,
		RoleID: roleID,
	}); err != nil {
		return fmt.Errorf("assign role to user: %w", err)
	}
	return nil
}

func auditMetadata(value any) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		return []byte(`{"error":"metadata_marshal_failed"}`)
	}
	return data
}

func nullableInt8(value int64) pgtype.Int8 {
	if value == 0 {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: value, Valid: true}
}

func mapRole(id int64, key, alias, description string, isSystem, isDefault, isDeletable, isEnabled bool) Role {
	return Role{
		ID:          id,
		Key:         key,
		Alias:       alias,
		Description: description,
		IsSystem:    isSystem,
		IsDefault:   isDefault,
		IsDeletable: isDeletable,
		IsEnabled:   isEnabled,
	}
}
