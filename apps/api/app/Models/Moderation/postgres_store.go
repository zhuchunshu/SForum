package moderation

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) GetSettings(ctx context.Context) (Settings, error) {
	return scanSettings(s.pool.QueryRow(ctx, `
		SELECT mode, review_new_users, new_user_max_age_days, review_external_links,
		  updated_by_user_id, updated_at
		FROM moderation_settings
		WHERE singleton = TRUE
	`))
}

func (s *PostgresStore) SaveSettings(ctx context.Context, settings Settings, actorUserID int64) (Settings, error) {
	return s.writeSettings(ctx, settings, actorUserID)
}

func (s *PostgresStore) ResetSettings(ctx context.Context, settings Settings, actorUserID int64) (Settings, error) {
	return s.writeSettings(ctx, settings, actorUserID)
}

func (s *PostgresStore) writeSettings(ctx context.Context, settings Settings, actorUserID int64) (Settings, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO moderation_settings (
		  singleton, mode, review_new_users, new_user_max_age_days,
		  review_external_links, updated_by_user_id, updated_at
		) VALUES (TRUE, $1, $2, $3, $4, $5, now())
		ON CONFLICT (singleton) DO UPDATE SET
		  mode = EXCLUDED.mode,
		  review_new_users = EXCLUDED.review_new_users,
		  new_user_max_age_days = EXCLUDED.new_user_max_age_days,
		  review_external_links = EXCLUDED.review_external_links,
		  updated_by_user_id = EXCLUDED.updated_by_user_id,
		  updated_at = now()
		RETURNING mode, review_new_users, new_user_max_age_days, review_external_links,
		  updated_by_user_id, updated_at
	`, settings.Mode, settings.ReviewNewUsers, settings.NewUserMaxAgeDays,
		settings.ReviewExternalLinks, nullableReporter(actorUserID))
	return scanSettings(row)
}

func scanSettings(row reportScanner) (Settings, error) {
	var settings Settings
	var updatedBy sql.NullInt64
	if err := row.Scan(
		&settings.Mode,
		&settings.ReviewNewUsers,
		&settings.NewUserMaxAgeDays,
		&settings.ReviewExternalLinks,
		&updatedBy,
		&settings.UpdatedAt,
	); err != nil {
		return Settings{}, fmt.Errorf("scan moderation settings: %w", err)
	}
	if updatedBy.Valid {
		value := updatedBy.Int64
		settings.UpdatedByUserID = &value
	}
	return settings, nil
}

func (s *PostgresStore) CreateReport(ctx context.Context, input CreateReportInput) (Report, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO moderation_reports (reporter_user_id, target_type, target_id, reason_code, body)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, reporter_user_id, target_type, target_id, reason_code, body, status,
		  reviewer_user_id, review_note, created_at, updated_at, resolved_at
	`, nullableReporter(input.ReporterUserID), input.TargetType, input.TargetID, input.ReasonCode, input.Body)
	report, err := scanReport(row)
	if err != nil {
		// 唯一索引冲突：同一举报者对同一目标已有 open 举报。
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Report{}, ErrReportDuplicate
		}
		return Report{}, fmt.Errorf("create report: %w", err)
	}
	return report, nil
}

func (s *PostgresStore) ListReports(ctx context.Context, input ReportListInput) (ReportList, error) {
	where := "WHERE 1=1"
	args := []any{}
	argIndex := 1
	if input.Status != "" {
		where += fmt.Sprintf(" AND moderation_reports.status = $%d", argIndex)
		args = append(args, input.Status)
		argIndex++
	}
	if input.TargetType != "" {
		where += fmt.Sprintf(" AND moderation_reports.target_type = $%d", argIndex)
		args = append(args, input.TargetType)
		argIndex++
	}
	if input.ReporterID > 0 {
		where += fmt.Sprintf(" AND moderation_reports.reporter_user_id = $%d", argIndex)
		args = append(args, input.ReporterID)
		argIndex++
	}

	var total int64
	countArgs := append([]any{}, args...)
	if err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM moderation_reports
	`+where, countArgs...).Scan(&total); err != nil {
		return ReportList{}, fmt.Errorf("count reports: %w", err)
	}

	args = append(args, input.PerPage, (input.Page-1)*input.PerPage)
	rows, err := s.pool.Query(ctx, reportSelectSQL()+where+`
		ORDER BY moderation_reports.created_at DESC, moderation_reports.id DESC
		LIMIT $`+itoa(argIndex)+` OFFSET $`+itoa(argIndex+1)+`
	`, args...)
	if err != nil {
		return ReportList{}, fmt.Errorf("list reports: %w", err)
	}
	defer rows.Close()

	items := []Report{}
	for rows.Next() {
		report, err := scanReport(rows)
		if err != nil {
			return ReportList{}, err
		}
		items = append(items, report)
	}
	if err := rows.Err(); err != nil {
		return ReportList{}, fmt.Errorf("iterate reports: %w", err)
	}
	return ReportList{Items: items, Total: total, Page: input.Page, PerPage: input.PerPage}, nil
}

func (s *PostgresStore) GetReport(ctx context.Context, reportID int64) (Report, error) {
	row := s.pool.QueryRow(ctx, reportSelectSQL()+`
		WHERE moderation_reports.id = $1
	`, reportID)
	report, err := scanReport(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Report{}, ErrReportNotFound
	}
	if err != nil {
		return Report{}, fmt.Errorf("get report: %w", err)
	}
	return report, nil
}

func (s *PostgresStore) UpdateReport(ctx context.Context, input UpdateReportInput) (Report, error) {
	// resolved/rejected 时记录 resolved_at。
	resolvedExpr := "resolved_at"
	switch input.Status {
	case StatusResolved, StatusRejected:
		resolvedExpr = "now()"
	case StatusOpen, StatusReviewing:
		resolvedExpr = "NULL"
	}
	row := s.pool.QueryRow(ctx, `
		UPDATE moderation_reports
		SET status = $2,
		    reviewer_user_id = $3,
		    review_note = $4,
		    resolved_at = `+resolvedExpr+`,
		    updated_at = now()
		WHERE id = $1
		RETURNING id, reporter_user_id, target_type, target_id, reason_code, body, status,
		  reviewer_user_id, review_note, created_at, updated_at, resolved_at
	`, input.ReportID, input.Status, nullableReporter(input.ReviewerUserID), input.ReviewNote)
	report, err := scanReport(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Report{}, ErrReportNotFound
	}
	if err != nil {
		return Report{}, fmt.Errorf("update report: %w", err)
	}
	return report, nil
}

func reportSelectSQL() string {
	return `
		SELECT moderation_reports.id, moderation_reports.reporter_user_id,
		  reporter.username, reporter.display_name,
		  moderation_reports.target_type, moderation_reports.target_id,
		  moderation_reports.reason_code, moderation_reports.body, moderation_reports.status,
		  moderation_reports.reviewer_user_id,
		  reviewer.username, reviewer.display_name,
		  moderation_reports.review_note,
		  moderation_reports.created_at, moderation_reports.updated_at, moderation_reports.resolved_at
		FROM moderation_reports
		LEFT JOIN users reporter ON reporter.id = moderation_reports.reporter_user_id
		LEFT JOIN users reviewer ON reviewer.id = moderation_reports.reviewer_user_id
	`
}

type reportScanner interface {
	Scan(dest ...any) error
}

func scanReport(row reportScanner) (Report, error) {
	var report Report
	var reporterID sql.NullInt64
	var reporterUsername sql.NullString
	var reporterDisplay sql.NullString
	var reviewerID sql.NullInt64
	var reviewerUsername sql.NullString
	var reviewerDisplay sql.NullString
	var resolvedAt sql.NullTime
	if err := row.Scan(
		&report.ID,
		&reporterID,
		&reporterUsername,
		&reporterDisplay,
		&report.TargetType,
		&report.TargetID,
		&report.ReasonCode,
		&report.Body,
		&report.Status,
		&reviewerID,
		&reviewerUsername,
		&reviewerDisplay,
		&report.ReviewNote,
		&report.CreatedAt,
		&report.UpdatedAt,
		&resolvedAt,
	); err != nil {
		return Report{}, err
	}
	if reporterID.Valid {
		report.ReporterUserID = reporterID.Int64
		report.ReporterName = displayName(reporterUsername, reporterDisplay)
	}
	if reviewerID.Valid {
		id := reviewerID.Int64
		report.ReviewerUserID = &id
		report.ReviewerName = displayName(reviewerUsername, reviewerDisplay)
	}
	if resolvedAt.Valid {
		t := resolvedAt.Time
		report.ResolvedAt = &t
	}
	return report, nil
}

func displayName(username, display sql.NullString) string {
	if display.Valid && display.String != "" {
		return display.String
	}
	if username.Valid && username.String != "" {
		return username.String
	}
	return ""
}

func nullableReporter(userID int64) any {
	if userID <= 0 {
		return nil
	}
	return userID
}

func itoa(value int) string {
	const digits = "0123456789"
	if value == 0 {
		return "0"
	}
	// 简单整数转字符串，避免引入 strconv 仅为单处使用。
	result := ""
	n := value
	for n > 0 {
		result = string(digits[n%10]) + result
		n /= 10
	}
	return result
}

// 保留 strings 引用（body/note 已在 service 层 trim，store 层不再重复处理）。
var _ = strings.TrimSpace
