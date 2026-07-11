package sitechrome

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) ListNavItems(ctx context.Context, enabledOnly bool) ([]NavItem, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, label_zh_cn, label_en_us, href, open_in_new_tab, position, enabled, created_at, updated_at
		FROM site_nav_items
		WHERE ($1 = FALSE OR enabled = TRUE)
		ORDER BY position ASC, id ASC
	`, enabledOnly)
	if err != nil {
		return nil, fmt.Errorf("list nav items: %w", err)
	}
	defer rows.Close()

	items := make([]NavItem, 0)
	for rows.Next() {
		item, err := scanNavItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) CreateNavItem(ctx context.Context, input CreateNavItemInput) (NavItem, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO site_nav_items (label_zh_cn, label_en_us, href, open_in_new_tab, position, enabled)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, label_zh_cn, label_en_us, href, open_in_new_tab, position, enabled, created_at, updated_at
	`, input.LabelZhCN, input.LabelEnUS, input.Href, input.OpenInNewTab, input.Position, input.Enabled)
	return scanNavItem(row)
}

func (s *PostgresStore) UpdateNavItem(ctx context.Context, input UpdateNavItemInput) (NavItem, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE site_nav_items
		SET label_zh_cn = COALESCE($2::text, label_zh_cn),
		    label_en_us = COALESCE($3::text, label_en_us),
		    href = COALESCE($4::text, href),
		    open_in_new_tab = COALESCE($5::boolean, open_in_new_tab),
		    position = COALESCE($6::integer, position),
		    enabled = COALESCE($7::boolean, enabled),
		    updated_at = now()
		WHERE id = $1
		RETURNING id, label_zh_cn, label_en_us, href, open_in_new_tab, position, enabled, created_at, updated_at
	`, input.ID, input.LabelZhCN, input.LabelEnUS, input.Href, input.OpenInNewTab, input.Position, input.Enabled)
	item, err := scanNavItem(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return NavItem{}, ErrNotFound
	}
	return item, err
}

func (s *PostgresStore) DeleteNavItem(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM site_nav_items WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) ListFriendLinks(ctx context.Context, enabledOnly bool) ([]FriendLink, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, url, description, logo_url, position, enabled, created_at, updated_at
		FROM site_friend_links
		WHERE ($1 = FALSE OR enabled = TRUE)
		ORDER BY position ASC, id ASC
	`, enabledOnly)
	if err != nil {
		return nil, fmt.Errorf("list friend links: %w", err)
	}
	defer rows.Close()

	items := make([]FriendLink, 0)
	for rows.Next() {
		item, err := scanFriendLink(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) CreateFriendLink(ctx context.Context, input CreateFriendLinkInput) (FriendLink, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO site_friend_links (name, url, description, logo_url, position, enabled)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, url, description, logo_url, position, enabled, created_at, updated_at
	`, input.Name, input.URL, input.Description, input.LogoURL, input.Position, input.Enabled)
	return scanFriendLink(row)
}

func (s *PostgresStore) UpdateFriendLink(ctx context.Context, input UpdateFriendLinkInput) (FriendLink, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE site_friend_links
		SET name = COALESCE($2::text, name),
		    url = COALESCE($3::text, url),
		    description = COALESCE($4::text, description),
		    logo_url = COALESCE($5::text, logo_url),
		    position = COALESCE($6::integer, position),
		    enabled = COALESCE($7::boolean, enabled),
		    updated_at = now()
		WHERE id = $1
		RETURNING id, name, url, description, logo_url, position, enabled, created_at, updated_at
	`, input.ID, input.Name, input.URL, input.Description, input.LogoURL, input.Position, input.Enabled)
	item, err := scanFriendLink(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return FriendLink{}, ErrNotFound
	}
	return item, err
}

func (s *PostgresStore) DeleteFriendLink(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM site_friend_links WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) ListAnnouncements(ctx context.Context, enabledOnly bool, activeNow bool) ([]Announcement, error) {
	now := time.Now().UTC()
	rows, err := s.pool.Query(ctx, `
		SELECT id, title_zh_cn, title_en_us, body_zh_cn, body_en_us, style, href,
		       dismissible, position, enabled, starts_at, ends_at, created_at, updated_at
		FROM site_announcements
		WHERE ($1 = FALSE OR enabled = TRUE)
		  AND (
		    $2 = FALSE
		    OR (
		      (starts_at IS NULL OR starts_at <= $3)
		      AND (ends_at IS NULL OR ends_at >= $3)
		    )
		  )
		ORDER BY position ASC, id ASC
	`, enabledOnly, activeNow, now)
	if err != nil {
		return nil, fmt.Errorf("list announcements: %w", err)
	}
	defer rows.Close()

	items := make([]Announcement, 0)
	for rows.Next() {
		item, err := scanAnnouncement(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) CreateAnnouncement(ctx context.Context, input CreateAnnouncementInput) (Announcement, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO site_announcements (
		  title_zh_cn, title_en_us, body_zh_cn, body_en_us, style, href,
		  dismissible, position, enabled, starts_at, ends_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, title_zh_cn, title_en_us, body_zh_cn, body_en_us, style, href,
		          dismissible, position, enabled, starts_at, ends_at, created_at, updated_at
	`, input.TitleZhCN, input.TitleEnUS, input.BodyZhCN, input.BodyEnUS, input.Style, input.Href,
		input.Dismissible, input.Position, input.Enabled, input.StartsAt, input.EndsAt)
	return scanAnnouncement(row)
}

func (s *PostgresStore) UpdateAnnouncement(ctx context.Context, input UpdateAnnouncementInput) (Announcement, error) {
	// starts_at / ends_at：Clear* 清空；指针非 nil 则写入；否则保持原值。
	row := s.pool.QueryRow(ctx, `
		UPDATE site_announcements
		SET title_zh_cn = COALESCE($2::text, title_zh_cn),
		    title_en_us = COALESCE($3::text, title_en_us),
		    body_zh_cn = COALESCE($4::text, body_zh_cn),
		    body_en_us = COALESCE($5::text, body_en_us),
		    style = COALESCE($6::text, style),
		    href = COALESCE($7::text, href),
		    dismissible = COALESCE($8::boolean, dismissible),
		    position = COALESCE($9::integer, position),
		    enabled = COALESCE($10::boolean, enabled),
		    starts_at = CASE
		      WHEN $11::boolean THEN NULL
		      WHEN $12::boolean THEN $13::timestamptz
		      ELSE starts_at
		    END,
		    ends_at = CASE
		      WHEN $14::boolean THEN NULL
		      WHEN $15::boolean THEN $16::timestamptz
		      ELSE ends_at
		    END,
		    updated_at = now()
		WHERE id = $1
		RETURNING id, title_zh_cn, title_en_us, body_zh_cn, body_en_us, style, href,
		          dismissible, position, enabled, starts_at, ends_at, created_at, updated_at
	`, input.ID,
		input.TitleZhCN, input.TitleEnUS, input.BodyZhCN, input.BodyEnUS, input.Style, input.Href,
		input.Dismissible, input.Position, input.Enabled,
		input.ClearStartsAt, input.StartsAt != nil, input.StartsAt,
		input.ClearEndsAt, input.EndsAt != nil, input.EndsAt,
	)
	item, err := scanAnnouncement(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Announcement{}, ErrNotFound
	}
	return item, err
}

func (s *PostgresStore) DeleteAnnouncement(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM site_announcements WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanNavItem(row scannable) (NavItem, error) {
	var item NavItem
	err := row.Scan(
		&item.ID, &item.LabelZhCN, &item.LabelEnUS, &item.Href, &item.OpenInNewTab,
		&item.Position, &item.Enabled, &item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}

func scanFriendLink(row scannable) (FriendLink, error) {
	var item FriendLink
	err := row.Scan(
		&item.ID, &item.Name, &item.URL, &item.Description, &item.LogoURL,
		&item.Position, &item.Enabled, &item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}

func scanAnnouncement(row scannable) (Announcement, error) {
	var item Announcement
	err := row.Scan(
		&item.ID, &item.TitleZhCN, &item.TitleEnUS, &item.BodyZhCN, &item.BodyEnUS,
		&item.Style, &item.Href, &item.Dismissible, &item.Position, &item.Enabled,
		&item.StartsAt, &item.EndsAt, &item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}
