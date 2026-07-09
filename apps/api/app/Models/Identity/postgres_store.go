package identity

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	avatar "github.com/zhuchunshu/sforum/apps/api/app/Support/Avatar"
	store "github.com/zhuchunshu/sforum/apps/api/database/sqlc"
)

type PostgresStore struct {
	pool          *pgxpool.Pool
	queries       *store.Queries
	avatarBuilder *avatar.ViewBuilder
}

type postgresTxStore struct {
	queries       *store.Queries
	avatarBuilder *avatar.ViewBuilder
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return NewPostgresStoreWithAvatar(pool, nil)
}

func NewPostgresStoreWithAvatar(pool *pgxpool.Pool, avatarOptions avatar.OptionResolver) *PostgresStore {
	return &PostgresStore{
		pool:          pool,
		queries:       store.New(pool),
		avatarBuilder: avatar.NewViewBuilder(avatarOptions),
	}
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

	txStore := &postgresTxStore{queries: s.queries.WithTx(tx), avatarBuilder: s.avatarBuilder}
	if err := fn(ctx, txStore); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit identity tx: %w", err)
	}
	return nil
}

func (s *PostgresStore) AnyUserExists(ctx context.Context) (bool, error) {
	return s.queries.AnyUserExists(ctx)
}

func (s *PostgresStore) FindRegistrationConflicts(ctx context.Context, username string, email string) (RegistrationConflicts, error) {
	row, err := s.queries.FindRegistrationConflicts(ctx, store.FindRegistrationConflictsParams{
		Username: username,
		Email:    email,
	})
	if err != nil {
		return RegistrationConflicts{}, fmt.Errorf("find registration conflicts: %w", err)
	}
	return RegistrationConflicts{UsernameTaken: row.UsernameTaken, EmailTaken: row.EmailTaken}, nil
}

func (s *PostgresStore) GetCurrentUser(ctx context.Context, userID int64) (CurrentUser, error) {
	current, err := scanCurrentUserWithAvatar(ctx, s.avatarBuilder, s.pool.QueryRow(ctx, `
		SELECT users.id, users.username, users.display_name, users.email, users.locale,
		       users.status, users.is_initial_super_admin,
		       user_profiles.avatar_attachment_id,
		       attachments.id, attachments.public_id, attachments.owner_user_id,
		       attachments.content_type, attachments.status
		FROM users
		LEFT JOIN user_profiles ON user_profiles.user_id = users.id
		LEFT JOIN attachments ON attachments.id = user_profiles.avatar_attachment_id
		WHERE users.id = $1
	`, userID))
	if err != nil {
		return CurrentUser{}, fmt.Errorf("get current user: %w", err)
	}
	if err := s.loadCurrentUserAccess(ctx, &current); err != nil {
		return CurrentUser{}, err
	}
	return current, nil
}

func (s *PostgresStore) GetCredentialByLogin(ctx context.Context, login string) (CredentialUser, error) {
	current, passwordHash, err := scanCredentialUserWithAvatar(ctx, s.avatarBuilder, s.pool.QueryRow(ctx, `
		SELECT users.id, users.username, users.display_name, users.email, users.locale,
		       users.status, users.is_initial_super_admin, user_credentials.password_hash,
		       user_profiles.avatar_attachment_id,
		       attachments.id, attachments.public_id, attachments.owner_user_id,
		       attachments.content_type, attachments.status
		FROM users
		JOIN user_credentials ON user_credentials.user_id = users.id
		LEFT JOIN user_profiles ON user_profiles.user_id = users.id
		LEFT JOIN attachments ON attachments.id = user_profiles.avatar_attachment_id
		WHERE users.username_lower = lower($1) OR users.email_lower = lower($1)
	`, login))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CredentialUser{}, ErrCredentialNotFound
		}
		return CredentialUser{}, fmt.Errorf("get user credential: %w", err)
	}
	if err := s.loadCurrentUserAccess(ctx, &current); err != nil {
		return CredentialUser{}, err
	}
	return CredentialUser{CurrentUser: current, PasswordHash: passwordHash}, nil
}

type currentUserAvatarScanner interface {
	Scan(dest ...any) error
}

func scanCurrentUserWithAvatar(ctx context.Context, builder *avatar.ViewBuilder, row currentUserAvatarScanner) (CurrentUser, error) {
	var current CurrentUser
	var email string
	var status string
	var avatarAttachmentID sql.NullInt64
	var attachmentID sql.NullInt64
	var attachmentPublicID sql.NullString
	var attachmentOwnerID sql.NullInt64
	var attachmentContentType sql.NullString
	var attachmentStatus sql.NullString
	if err := row.Scan(
		&current.ID,
		&current.Username,
		&current.DisplayName,
		&email,
		&current.Locale,
		&status,
		&current.IsInitialSuperAdmin,
		&avatarAttachmentID,
		&attachmentID,
		&attachmentPublicID,
		&attachmentOwnerID,
		&attachmentContentType,
		&attachmentStatus,
	); err != nil {
		return CurrentUser{}, err
	}
	current.Status = UserStatus(status)
	current.Avatar = currentUserAvatar(ctx, builder, current, email, avatarAttachmentID, attachmentID, attachmentPublicID, attachmentOwnerID, attachmentContentType, attachmentStatus)
	return current, nil
}

func scanCredentialUserWithAvatar(ctx context.Context, builder *avatar.ViewBuilder, row currentUserAvatarScanner) (CurrentUser, string, error) {
	var current CurrentUser
	var email string
	var status string
	var passwordHash string
	var avatarAttachmentID sql.NullInt64
	var attachmentID sql.NullInt64
	var attachmentPublicID sql.NullString
	var attachmentOwnerID sql.NullInt64
	var attachmentContentType sql.NullString
	var attachmentStatus sql.NullString
	if err := row.Scan(
		&current.ID,
		&current.Username,
		&current.DisplayName,
		&email,
		&current.Locale,
		&status,
		&current.IsInitialSuperAdmin,
		&passwordHash,
		&avatarAttachmentID,
		&attachmentID,
		&attachmentPublicID,
		&attachmentOwnerID,
		&attachmentContentType,
		&attachmentStatus,
	); err != nil {
		return CurrentUser{}, "", err
	}
	current.Status = UserStatus(status)
	current.Avatar = currentUserAvatar(ctx, builder, current, email, avatarAttachmentID, attachmentID, attachmentPublicID, attachmentOwnerID, attachmentContentType, attachmentStatus)
	return current, passwordHash, nil
}

func currentUserAvatar(ctx context.Context, builder *avatar.ViewBuilder, current CurrentUser, email string, avatarAttachmentID sql.NullInt64, attachmentID sql.NullInt64, attachmentPublicID sql.NullString, attachmentOwnerID sql.NullInt64, attachmentContentType sql.NullString, attachmentStatus sql.NullString) avatar.View {
	if builder == nil {
		builder = avatar.NewViewBuilder(nil)
	}
	source := avatar.Source{}
	if avatarAttachmentID.Valid && avatarAttachmentID.Int64 > 0 {
		id := avatarAttachmentID.Int64
		source.AttachmentID = &id
	}
	if attachmentID.Valid && attachmentID.Int64 > 0 {
		source.Attachment = &avatar.Attachment{
			ID:          attachmentID.Int64,
			PublicID:    attachmentPublicID.String,
			OwnerUserID: nullableSQLInt64Value(attachmentOwnerID),
			ContentType: attachmentContentType.String,
			Status:      attachmentStatus.String,
		}
	}
	return builder.AvatarView(ctx, avatar.User{
		UserID:      current.ID,
		Username:    current.Username,
		DisplayName: current.DisplayName,
		Email:       email,
	}, source)
}

func nullableSQLInt64Value(value sql.NullInt64) int64 {
	if value.Valid {
		return value.Int64
	}
	return 0
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

func (s *PostgresStore) ListPermissions(ctx context.Context) ([]Permission, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT key, module, description
		FROM permissions
		ORDER BY module ASC, key ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list permissions: %w", err)
	}
	defer rows.Close()

	permissions := []Permission{}
	for rows.Next() {
		var permission Permission
		if err := rows.Scan(&permission.Key, &permission.Module, &permission.Description); err != nil {
			return nil, fmt.Errorf("scan permission: %w", err)
		}
		permissions = append(permissions, permission)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate permissions: %w", err)
	}
	return permissions, nil
}

func (s *PostgresStore) ListPermissionMatrix(ctx context.Context) (PermissionMatrix, error) {
	permissions, err := s.ListPermissions(ctx)
	if err != nil {
		return PermissionMatrix{}, err
	}
	roles, err := s.ListRoles(ctx)
	if err != nil {
		return PermissionMatrix{}, err
	}

	matrix := PermissionMatrix{
		Permissions: permissions,
		Roles:       make([]RolePermissionSet, 0, len(roles)),
	}
	for _, role := range roles {
		matrix.Roles = append(matrix.Roles, RolePermissionSet{
			RoleKey:        role.Key,
			PermissionKeys: role.PermissionKeys,
		})
	}
	return matrix, nil
}

func (s *PostgresStore) ListUsers(ctx context.Context, input UserListInput) (AdminUserList, error) {
	const countSQL = `
		SELECT count(*)
		FROM users
		WHERE ($1 = '' OR username_lower LIKE '%' || lower($1) || '%' ESCAPE '\' OR email_lower LIKE '%' || lower($1) || '%' ESCAPE '\' OR lower(display_name) LIKE '%' || lower($1) || '%' ESCAPE '\')
		  AND ($2 = '' OR status = $2)
		  AND ($3 = '' OR EXISTS (
		    SELECT 1
		    FROM user_roles
		    JOIN roles ON roles.id = user_roles.role_id
		    WHERE user_roles.user_id = users.id AND roles.key = $3
		  ))
	`
	const listSQL = `
		SELECT id, username, email, display_name, locale, status, is_initial_super_admin
		FROM users
		WHERE ($1 = '' OR username_lower LIKE '%' || lower($1) || '%' ESCAPE '\' OR email_lower LIKE '%' || lower($1) || '%' ESCAPE '\' OR lower(display_name) LIKE '%' || lower($1) || '%' ESCAPE '\')
		  AND ($2 = '' OR status = $2)
		  AND ($3 = '' OR EXISTS (
		    SELECT 1
		    FROM user_roles
		    JOIN roles ON roles.id = user_roles.role_id
		    WHERE user_roles.user_id = users.id AND roles.key = $3
		  ))
		ORDER BY id ASC
		LIMIT $4 OFFSET $5
	`

	var total int64
	if err := s.pool.QueryRow(ctx, countSQL, input.Query, input.Status, input.RoleKey).Scan(&total); err != nil {
		return AdminUserList{}, fmt.Errorf("count admin users: %w", err)
	}

	offset := (input.Page - 1) * input.PerPage
	rows, err := s.pool.Query(ctx, listSQL, input.Query, input.Status, input.RoleKey, input.PerPage, offset)
	if err != nil {
		return AdminUserList{}, fmt.Errorf("list admin users: %w", err)
	}
	defer rows.Close()

	items := []AdminUserSummary{}
	for rows.Next() {
		var user AdminUserSummary
		if err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.DisplayName, &user.Locale, &user.Status, &user.IsInitialSuperAdmin); err != nil {
			return AdminUserList{}, fmt.Errorf("scan admin user: %w", err)
		}
		roleKeys, err := s.listAssignedUserRoleKeys(ctx, user.ID)
		if err != nil {
			return AdminUserList{}, err
		}
		user.RoleKeys = roleKeys
		items = append(items, user)
	}
	if err := rows.Err(); err != nil {
		return AdminUserList{}, fmt.Errorf("iterate admin users: %w", err)
	}

	return AdminUserList{Items: items, Total: total, Page: input.Page, PerPage: input.PerPage}, nil
}

func (s *PostgresStore) GetAdminUser(ctx context.Context, userID int64) (AdminUserDetail, error) {
	var detail AdminUserDetail
	err := s.pool.QueryRow(ctx, `
		SELECT id, username, email, display_name, locale, status, is_initial_super_admin
		FROM users
		WHERE id = $1
	`, userID).Scan(
		&detail.ID,
		&detail.Username,
		&detail.Email,
		&detail.DisplayName,
		&detail.Locale,
		&detail.Status,
		&detail.IsInitialSuperAdmin,
	)
	if err != nil {
		return AdminUserDetail{}, fmt.Errorf("get admin user: %w", err)
	}

	if err := s.loadAdminUserAccess(ctx, &detail); err != nil {
		return AdminUserDetail{}, err
	}
	return detail, nil
}

func (s *PostgresStore) ListRoles(ctx context.Context) ([]Role, error) {
	rows, err := s.queries.ListRoles(ctx)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}

	roles := make([]Role, 0, len(rows))
	for _, row := range rows {
		role := mapRole(row.ID, row.Key, row.Alias, row.Description, row.IsSystem, row.IsDefault, row.IsDeletable, row.IsEnabled)
		permissionKeys, err := s.listRolePermissionKeys(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		role.PermissionKeys = permissionKeys
		roles = append(roles, role)
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

func (s *PostgresStore) ReplaceUserRoles(ctx context.Context, actorUserID int64, targetUserID int64, roleKeys []string) (AdminUserDetail, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return AdminUserDetail{}, fmt.Errorf("begin user roles tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	queries := s.queries.WithTx(tx)
	if _, err := tx.Exec(ctx, "DELETE FROM user_roles WHERE user_id = $1", targetUserID); err != nil {
		return AdminUserDetail{}, fmt.Errorf("delete user roles: %w", err)
	}
	for _, roleKey := range roleKeys {
		role, err := queries.GetRoleByKey(ctx, roleKey)
		if err != nil {
			return AdminUserDetail{}, fmt.Errorf("get role for user assignment: %w", err)
		}
		if err := queries.AssignRoleToUser(ctx, store.AssignRoleToUserParams{
			UserID: targetUserID,
			RoleID: role.ID,
		}); err != nil {
			return AdminUserDetail{}, fmt.Errorf("assign user role %s: %w", roleKey, err)
		}
	}

	metadata := auditMetadata(map[string]any{"roleKeys": roleKeys})
	if err := queries.CreateAuditEvent(ctx, store.CreateAuditEventParams{
		ActorUserID:  nullableInt8(actorUserID),
		TargetUserID: nullableInt8(targetUserID),
		Action:       "user.roles.replace",
		Metadata:     metadata,
	}); err != nil {
		return AdminUserDetail{}, fmt.Errorf("audit user roles: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return AdminUserDetail{}, fmt.Errorf("commit user roles tx: %w", err)
	}
	return s.GetAdminUser(ctx, targetUserID)
}

func (s *PostgresStore) ReplaceUserPermissionOverrides(ctx context.Context, actorUserID int64, targetUserID int64, overrides PermissionOverrides) (AdminUserDetail, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return AdminUserDetail{}, fmt.Errorf("begin user permission overrides tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	queries := s.queries.WithTx(tx)
	if _, err := tx.Exec(ctx, "DELETE FROM user_permission_overrides WHERE user_id = $1", targetUserID); err != nil {
		return AdminUserDetail{}, fmt.Errorf("delete user permission overrides: %w", err)
	}

	for _, permission := range overrides.Allow {
		if err := insertUserPermissionOverride(ctx, tx, actorUserID, targetUserID, permission, "allow"); err != nil {
			return AdminUserDetail{}, err
		}
	}
	for _, permission := range overrides.Deny {
		if err := insertUserPermissionOverride(ctx, tx, actorUserID, targetUserID, permission, "deny"); err != nil {
			return AdminUserDetail{}, err
		}
	}

	metadata := auditMetadata(map[string]any{"allow": overrides.Allow, "deny": overrides.Deny})
	if err := queries.CreateAuditEvent(ctx, store.CreateAuditEventParams{
		ActorUserID:  nullableInt8(actorUserID),
		TargetUserID: nullableInt8(targetUserID),
		Action:       "user.permissions.replace",
		Metadata:     metadata,
	}); err != nil {
		return AdminUserDetail{}, fmt.Errorf("audit user permissions: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return AdminUserDetail{}, fmt.Errorf("commit user permission overrides tx: %w", err)
	}
	return s.GetAdminUser(ctx, targetUserID)
}

func (s *PostgresStore) RecordLoginAudit(ctx context.Context, input LoginAudit) error {
	action := input.Action
	if action == "" {
		action = AuditActionLogin
	}

	metadata := auditMetadata(map[string]any{
		"ipAddress":   input.IPAddress,
		"userAgent":   input.UserAgent,
		"sessionHash": input.SessionHash,
	})
	if err := s.queries.CreateAuditEvent(ctx, store.CreateAuditEventParams{
		ActorUserID:  nullableInt8(input.UserID),
		TargetUserID: nullableInt8(input.UserID),
		Action:       action,
		Metadata:     metadata,
	}); err != nil {
		return fmt.Errorf("audit login: %w", err)
	}
	return nil
}

func (s *PostgresStore) loadCurrentUserAccess(ctx context.Context, current *CurrentUser) error {
	return loadCurrentUserAccess(ctx, s.queries, current)
}

func (s *postgresTxStore) LoadCurrentUserAccess(ctx context.Context, current *CurrentUser) error {
	return loadCurrentUserAccess(ctx, s.queries, current)
}

func (s *PostgresStore) loadAdminUserAccess(ctx context.Context, detail *AdminUserDetail) error {
	roleKeys, err := s.listAssignedUserRoleKeys(ctx, detail.ID)
	if err != nil {
		return err
	}
	permissions, err := s.queries.ListUserPermissions(ctx, detail.ID)
	if err != nil {
		return fmt.Errorf("list admin user permissions: %w", err)
	}
	overrides, err := s.listPermissionOverrides(ctx, detail.ID)
	if err != nil {
		return err
	}
	detail.RoleKeys = roleKeys
	if permissions == nil {
		permissions = []string{}
	}
	detail.Permissions = permissions
	detail.PermissionOverrides = overrides
	return nil
}

func loadCurrentUserAccess(ctx context.Context, queries *store.Queries, current *CurrentUser) error {
	roleKeys, err := queries.ListUserRoleKeys(ctx, current.ID)
	if err != nil {
		return fmt.Errorf("list current user roles: %w", err)
	}
	permissions, err := queries.ListUserPermissions(ctx, current.ID)
	if err != nil {
		return fmt.Errorf("list current user permissions: %w", err)
	}
	if roleKeys == nil {
		roleKeys = []string{}
	}
	if permissions == nil {
		permissions = []string{}
	}
	current.RoleKeys = roleKeys
	current.Permissions = permissions
	return nil
}

func (s *PostgresStore) listAssignedUserRoleKeys(ctx context.Context, userID int64) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT roles.key
		FROM user_roles
		JOIN roles ON roles.id = user_roles.role_id
		WHERE user_roles.user_id = $1
		ORDER BY roles.key
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list assigned user roles: %w", err)
	}
	defer rows.Close()

	roleKeys := []string{}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("scan assigned user role: %w", err)
		}
		roleKeys = append(roleKeys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate assigned user roles: %w", err)
	}
	return roleKeys, nil
}

func (s *PostgresStore) listRolePermissionKeys(ctx context.Context, roleID int64) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT permission_key
		FROM role_permissions
		WHERE role_id = $1
		ORDER BY permission_key
	`, roleID)
	if err != nil {
		return nil, fmt.Errorf("list role permissions: %w", err)
	}
	defer rows.Close()

	permissions := []string{}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("scan role permission: %w", err)
		}
		permissions = append(permissions, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate role permissions: %w", err)
	}
	return permissions, nil
}

func (s *PostgresStore) listPermissionOverrides(ctx context.Context, userID int64) (PermissionOverrides, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT permission_key, effect
		FROM user_permission_overrides
		WHERE user_id = $1
		ORDER BY permission_key
	`, userID)
	if err != nil {
		return PermissionOverrides{}, fmt.Errorf("list user permission overrides: %w", err)
	}
	defer rows.Close()

	overrides := PermissionOverrides{Allow: []string{}, Deny: []string{}}
	for rows.Next() {
		var permission string
		var effect string
		if err := rows.Scan(&permission, &effect); err != nil {
			return PermissionOverrides{}, fmt.Errorf("scan user permission override: %w", err)
		}
		switch effect {
		case "allow":
			overrides.Allow = append(overrides.Allow, permission)
		case "deny":
			overrides.Deny = append(overrides.Deny, permission)
		}
	}
	if err := rows.Err(); err != nil {
		return PermissionOverrides{}, fmt.Errorf("iterate user permission overrides: %w", err)
	}
	return overrides, nil
}

func insertUserPermissionOverride(ctx context.Context, tx pgx.Tx, actorUserID int64, targetUserID int64, permission string, effect string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO user_permission_overrides (user_id, permission_key, effect, updated_by_user_id)
		VALUES ($1, $2, $3, $4)
	`, targetUserID, permission, effect, nullableInt8(actorUserID))
	if err != nil {
		return fmt.Errorf("insert user permission override %s:%s: %w", permission, effect, err)
	}
	return nil
}

func (s *postgresTxStore) AnyUserExists(ctx context.Context) (bool, error) {
	return s.queries.AnyUserExists(ctx)
}

func (s *postgresTxStore) FindRegistrationConflicts(ctx context.Context, username string, email string) (RegistrationConflicts, error) {
	row, err := s.queries.FindRegistrationConflicts(ctx, store.FindRegistrationConflictsParams{
		Username: username,
		Email:    email,
	})
	if err != nil {
		return RegistrationConflicts{}, fmt.Errorf("find registration conflicts: %w", err)
	}
	return RegistrationConflicts{UsernameTaken: row.UsernameTaken, EmailTaken: row.EmailTaken}, nil
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
		if fields := uniqueRegistrationFields(err); len(fields) > 0 {
			return CurrentUser{}, NewRegisterInvalid(fields)
		}
		return CurrentUser{}, fmt.Errorf("create user: %w", err)
	}
	current := CurrentUser{
		ID:                  row.ID,
		Username:            row.Username,
		DisplayName:         row.DisplayName,
		Locale:              row.Locale,
		Status:              UserStatus(row.Status),
		IsInitialSuperAdmin: row.IsInitialSuperAdmin,
	}
	current.Avatar = currentUserAvatar(ctx, s.avatarBuilder, current, input.Email, sql.NullInt64{}, sql.NullInt64{}, sql.NullString{}, sql.NullInt64{}, sql.NullString{}, sql.NullString{})
	return current, nil
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
		ID:             id,
		Key:            key,
		Alias:          alias,
		Description:    description,
		IsSystem:       isSystem,
		IsDefault:      isDefault,
		IsDeletable:    isDeletable,
		IsEnabled:      isEnabled,
		PermissionKeys: []string{},
	}
}

func uniqueRegistrationFields(err error) FieldMessages {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return nil
	}

	fields := FieldMessages{}
	switch pgErr.ConstraintName {
	case "users_username_lower_key":
		addFieldMessage(fields, FieldUsername, MessageUsernameTaken)
	case "users_email_lower_key":
		addFieldMessage(fields, FieldEmail, MessageEmailTaken)
	default:
		return nil
	}
	return fields
}
