package identity

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	avatar "github.com/zhuchunshu/sforum/apps/api/app/Support/Avatar"
	store "github.com/zhuchunshu/sforum/apps/api/database/sqlc"
)

type PostgresStore struct {
	pool                  *pgxpool.Pool
	queries               *store.Queries
	avatarBuilder         *avatar.ViewBuilder
	authorityMutationGate IdentityAuthorityMutationGate
}

// IdentityAuthorityMutationGate keeps user-row authority writers outside the
// main pool while an accepted session issue/renew effect holds compatible row
// locks on a reserved connection.
type IdentityAuthorityMutationGate interface {
	RunSessionPolicyMutation(context.Context, func() error) error
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

// WithAuthorityMutationGate binds the production Session Policy mutation gate.
// Bootstrap calls this before the HTTP server starts; tests and legacy seams may
// leave it nil.
func (s *PostgresStore) WithAuthorityMutationGate(gate IdentityAuthorityMutationGate) *PostgresStore {
	if s != nil {
		s.authorityMutationGate = gate
	}
	return s
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
		       users.status, users.is_initial_super_admin, users.current_token_version,
		       user_appearance_preferences.theme, user_appearance_preferences.light_background,
		       user_profiles.avatar_attachment_id,
		       attachments.id, attachments.public_id, attachments.owner_user_id,
		       attachments.content_type, attachments.status
		FROM users
		LEFT JOIN user_appearance_preferences ON user_appearance_preferences.user_id = users.id
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

// UpdateCurrentUserLocale persists a private account preference and returns the
// same current-user projection used by the authenticated session endpoint.
func (s *PostgresStore) UpdateCurrentUserLocale(ctx context.Context, userID int64, locale string) (CurrentUser, error) {
	result, err := s.pool.Exec(ctx, `
		UPDATE users
		SET locale = $2, updated_at = now()
		WHERE id = $1
	`, userID, locale)
	if err != nil {
		return CurrentUser{}, fmt.Errorf("update user locale: %w", err)
	}
	if result.RowsAffected() == 0 {
		return CurrentUser{}, ErrUserNotFound
	}
	return s.GetCurrentUser(ctx, userID)
}

func (s *PostgresStore) UpdateCurrentUserAppearance(ctx context.Context, userID int64, preference AppearancePreference) (CurrentUser, error) {
	result, err := s.pool.Exec(ctx, `
		INSERT INTO user_appearance_preferences (user_id, theme, light_background, created_at, updated_at)
		SELECT id, $2, $3, now(), now() FROM users WHERE id = $1
		ON CONFLICT (user_id) DO UPDATE
		SET theme = EXCLUDED.theme, light_background = EXCLUDED.light_background, updated_at = now()
	`, userID, preference.Theme, preference.LightBackground)
	if err != nil {
		return CurrentUser{}, fmt.Errorf("update user appearance: %w", err)
	}
	if result.RowsAffected() == 0 {
		return CurrentUser{}, ErrUserNotFound
	}
	return s.GetCurrentUser(ctx, userID)
}

func (s *PostgresStore) ClearCurrentUserAppearance(ctx context.Context, userID int64) (CurrentUser, error) {
	var exists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, userID).Scan(&exists); err != nil {
		return CurrentUser{}, fmt.Errorf("check user before clearing appearance: %w", err)
	}
	if !exists {
		return CurrentUser{}, ErrUserNotFound
	}
	if _, err := s.pool.Exec(ctx, `DELETE FROM user_appearance_preferences WHERE user_id = $1`, userID); err != nil {
		return CurrentUser{}, fmt.Errorf("clear user appearance: %w", err)
	}
	return s.GetCurrentUser(ctx, userID)
}

// GetCurrentUserByEmail 按邮箱加载完整 CurrentUser（不要求 password credential）。
func (s *PostgresStore) GetCurrentUserByEmail(ctx context.Context, email string) (CurrentUser, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return CurrentUser{}, ErrUserNotFound
	}
	current, err := scanCurrentUserWithAvatar(ctx, s.avatarBuilder, s.pool.QueryRow(ctx, `
		SELECT users.id, users.username, users.display_name, users.email, users.locale,
		       users.status, users.is_initial_super_admin, users.current_token_version,
		       user_appearance_preferences.theme, user_appearance_preferences.light_background,
		       user_profiles.avatar_attachment_id,
		       attachments.id, attachments.public_id, attachments.owner_user_id,
		       attachments.content_type, attachments.status
		FROM users
		LEFT JOIN user_appearance_preferences ON user_appearance_preferences.user_id = users.id
		LEFT JOIN user_profiles ON user_profiles.user_id = users.id
		LEFT JOIN attachments ON attachments.id = user_profiles.avatar_attachment_id
		WHERE users.email_lower = lower($1)
	`, email))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CurrentUser{}, ErrUserNotFound
		}
		return CurrentUser{}, fmt.Errorf("get current user by email: %w", err)
	}
	if err := s.loadCurrentUserAccess(ctx, &current); err != nil {
		return CurrentUser{}, err
	}
	return current, nil
}

func (s *PostgresStore) GetCredentialByLogin(ctx context.Context, login string) (CredentialUser, error) {
	current, passwordHash, err := scanCredentialUserWithAvatar(ctx, s.avatarBuilder, s.pool.QueryRow(ctx, `
		SELECT users.id, users.username, users.display_name, users.email, users.locale,
		       users.status, users.is_initial_super_admin, users.current_token_version,
		       user_credentials.password_hash,
		       user_appearance_preferences.theme, user_appearance_preferences.light_background,
		       user_profiles.avatar_attachment_id,
		       attachments.id, attachments.public_id, attachments.owner_user_id,
		       attachments.content_type, attachments.status
		FROM users
		JOIN user_credentials ON user_credentials.user_id = users.id
		LEFT JOIN user_appearance_preferences ON user_appearance_preferences.user_id = users.id
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
	var appearanceTheme sql.NullString
	var appearanceLightBackground sql.NullString
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
		&current.CurrentTokenVersion,
		&appearanceTheme,
		&appearanceLightBackground,
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
	current.Appearance = scanAppearancePreference(appearanceTheme, appearanceLightBackground)
	current.Avatar = currentUserAvatar(ctx, builder, current, email, avatarAttachmentID, attachmentID, attachmentPublicID, attachmentOwnerID, attachmentContentType, attachmentStatus)
	return current, nil
}

func scanCredentialUserWithAvatar(ctx context.Context, builder *avatar.ViewBuilder, row currentUserAvatarScanner) (CurrentUser, string, error) {
	var current CurrentUser
	var email string
	var status string
	var passwordHash string
	var appearanceTheme sql.NullString
	var appearanceLightBackground sql.NullString
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
		&current.CurrentTokenVersion,
		&passwordHash,
		&appearanceTheme,
		&appearanceLightBackground,
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
	current.Appearance = scanAppearancePreference(appearanceTheme, appearanceLightBackground)
	current.Avatar = currentUserAvatar(ctx, builder, current, email, avatarAttachmentID, attachmentID, attachmentPublicID, attachmentOwnerID, attachmentContentType, attachmentStatus)
	return current, passwordHash, nil
}

func scanAppearancePreference(theme sql.NullString, lightBackground sql.NullString) *AppearancePreference {
	if !theme.Valid || !lightBackground.Valid {
		return nil
	}
	return &AppearancePreference{Theme: theme.String, LightBackground: lightBackground.String}
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
	// 注册时间单独读取，供新人信任阶梯；失败时保留零值（跳过新人限制）。
	var createdAt time.Time
	_ = s.pool.QueryRow(ctx, `SELECT created_at FROM users WHERE id = $1`, userID).Scan(&createdAt)
	return Actor{
		ID:          userID,
		Status:      current.Status,
		RoleKeys:    current.RoleKeys,
		Permissions: permissions,
		CreatedAt:   createdAt,
	}, nil
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
