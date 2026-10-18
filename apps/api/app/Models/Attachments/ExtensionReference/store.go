package extensionreference

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store 是扩展附件引用的唯一写入口。
//
// 所有方法都要求调用方提供事务：Host Command 需要把引用写入、命令回执与审计
// 证据放在同一个 PostgreSQL 事务里。
type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// PrepareClaim 完成认领的全部权威读取与校验，返回不透明计划。
//
// 步骤 0 的扩展级 advisory 锁把本扩展的所有 claim/release 变更串行化，之后的
// 「读旧引用 → 删旧 → 插新 → 重算计数」才是一个不会产生幽灵计数的临界区。
func (s *Store) PrepareClaim(ctx context.Context, tx pgx.Tx, input ClaimInput) (*ClaimPlan, error) {
	if s == nil || tx == nil {
		return nil, ErrInvalid
	}
	extensionID, scope, err := Scope(input.ExtensionID, input.Context)
	if err != nil {
		return nil, err
	}
	publicID := strings.TrimSpace(input.PublicID)
	if publicID == "" || input.ActorUserID <= 0 || input.ExpectedRevision.IsZero() {
		return nil, ErrInvalid
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, lockKey(extensionID)); err != nil {
		return nil, fmt.Errorf("lock extension attachment references: %w", err)
	}

	var (
		attachmentID        int64
		contentType, sha256 string
		visibility          string
		ownerUserID         sql.NullInt64
		sizeBytes           int64
		width, height       sql.NullInt64
		referenceCount      int
		revision            time.Time
	)
	err = tx.QueryRow(ctx, `
		SELECT id, content_type, size_bytes, sha256, image_width, image_height,
		  visibility, owner_user_id, reference_count, updated_at
		FROM attachments
		WHERE public_id = $1 AND status = 'active'
		FOR UPDATE
	`, publicID).Scan(
		&attachmentID, &contentType, &sizeBytes, &sha256, &width, &height,
		&visibility, &ownerUserID, &referenceCount, &revision,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock extension claim attachment: %w", err)
	}
	// 授权：附件所有者本人，或持 attachment.manage 的用户。插件前台允许持
	// create_clients 的普通用户自助建应用，那些用户通常没有 attachment.manage，
	// 所以按归属判定而不是统一要求管理权限。
	if !input.ManageAny && (!ownerUserID.Valid || ownerUserID.Int64 != input.ActorUserID) {
		return nil, ErrForbidden
	}
	if !IsContentType(contentType) {
		return nil, ErrInvalid
	}
	if input.Policy.MaxBytes > 0 && sizeBytes > input.Policy.MaxBytes {
		return nil, ErrTooLarge
	}
	if !revision.Equal(input.ExpectedRevision.UTC()) {
		return nil, ErrRevisionConflict
	}
	// 拒绝二次归属变更：正在被论坛/头像/SEO/站点图使用的文件不能被改成公开资产。
	var foreignReference bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1 FROM attachment_references
		  WHERE attachment_id = $1 AND resource_type <> $2
		)
	`, attachmentID, ResourceType).Scan(&foreignReference); err != nil {
		return nil, fmt.Errorf("check extension claim foreign references: %w", err)
	}
	if foreignReference {
		return nil, ErrForbidden
	}

	var oldAttachmentID int64
	err = tx.QueryRow(ctx, `
		SELECT attachment_id FROM attachment_references
		WHERE resource_type = $1 AND resource_id = 0 AND context = $2
		FOR UPDATE
	`, ResourceType, scope).Scan(&oldAttachmentID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("lock extension claim context: %w", err)
	}
	// 容量上限按「存活引用数」计。同一 context 换图不会改变引用行数，
	// 因此只在新增 context 时判定。
	if oldAttachmentID == 0 && input.Policy.ReferenceLimit > 0 {
		var live int64
		if err := tx.QueryRow(ctx, `
			SELECT count(*) FROM attachment_references
			WHERE resource_type = $1 AND resource_id = 0
			  AND LEFT(context, length($2) + 1) = $2 || ':'
		`, ResourceType, extensionID).Scan(&live); err != nil {
			return nil, fmt.Errorf("count extension attachment references: %w", err)
		}
		if live >= int64(input.Policy.ReferenceLimit) {
			return nil, ErrLimitReached
		}
	}

	return &ClaimPlan{
		ExtensionID: extensionID, Scope: scope, AttachmentID: attachmentID,
		PublicID: publicID, ContentType: contentType, SHA256: sha256, SizeBytes: sizeBytes,
		Width: int(width.Int64), Height: int(height.Int64), Visibility: visibility,
		OldAttachmentID: oldAttachmentID, ActorUserID: input.ActorUserID,
		ReferenceCount: referenceCount, UpdatedAt: revision,
	}, nil
}

// ApplyClaim 执行认领的写步骤：删旧引用、强制公开、插新引用、按引用行重算计数。
func (s *Store) ApplyClaim(ctx context.Context, tx pgx.Tx, plan *ClaimPlan) (Claim, error) {
	if s == nil || tx == nil || plan == nil || plan.AttachmentID <= 0 {
		return Claim{}, ErrInvalid
	}
	affected := []int64{plan.AttachmentID}
	if plan.OldAttachmentID != 0 && plan.OldAttachmentID != plan.AttachmentID {
		// 同一 context 全局至多一行引用，「换图」必须删旧插新。
		if _, err := tx.Exec(ctx, `
			DELETE FROM attachment_references
			WHERE resource_type = $1 AND resource_id = 0 AND context = $2
		`, ResourceType, plan.Scope); err != nil {
			return Claim{}, fmt.Errorf("delete replaced extension reference: %w", err)
		}
		affected = append(affected, plan.OldAttachmentID)
	}
	// 认领的附件必须是公开可读的站点资产：上传默认可见性由
	// attachment.default_visibility 决定，运营改私有时这一步真实生效。
	if _, err := tx.Exec(ctx, `
		UPDATE attachments
		SET visibility = $2, updated_at = now()
		WHERE id = $1 AND visibility <> $2
	`, plan.AttachmentID, "public"); err != nil {
		return Claim{}, fmt.Errorf("publish extension claim attachment: %w", err)
	}
	tag, err := tx.Exec(ctx, `
		INSERT INTO attachment_references (attachment_id, resource_type, resource_id, context, created_by_user_id)
		VALUES ($1, $2, 0, $3, $4)
		ON CONFLICT DO NOTHING
	`, plan.AttachmentID, ResourceType, plan.Scope, plan.ActorUserID)
	if err != nil {
		return Claim{}, fmt.Errorf("insert extension reference: %w", err)
	}
	if tag.RowsAffected() != 1 {
		// 扩展级锁就位后这里不应冲突。真发生了说明锁失效或调用方越过了
		// Prepare，按冲突拒绝并让审计看到，绝不静默继续。
		var existing int64
		loadErr := tx.QueryRow(ctx, `
			SELECT attachment_id FROM attachment_references
			WHERE resource_type = $1 AND resource_id = 0 AND context = $2
		`, ResourceType, plan.Scope).Scan(&existing)
		if loadErr != nil && !errors.Is(loadErr, pgx.ErrNoRows) {
			return Claim{}, fmt.Errorf("reload extension reference after conflict: %w", loadErr)
		}
		if existing != plan.AttachmentID {
			return Claim{}, ErrConflict
		}
	}
	counts, err := recount(ctx, tx, affected...)
	if err != nil {
		return Claim{}, err
	}
	count := counts[plan.AttachmentID]
	return Claim{
		AttachmentID: plan.AttachmentID, PublicID: plan.PublicID, URL: URLPath(plan.PublicID),
		ContentType: plan.ContentType, Width: plan.Width, Height: plan.Height,
		SizeBytes: plan.SizeBytes, SHA256: plan.SHA256,
		ReferenceCount: count.Count, UpdatedAt: count.UpdatedAt,
	}, nil
}

// Release 删除本扩展的引用行并按引用行重算计数。
// 幂等：没有匹配行时返回 Released = 0 且不报错。
func (s *Store) Release(ctx context.Context, tx pgx.Tx, input ReleaseInput) (Release, error) {
	if s == nil || tx == nil {
		return Release{}, ErrInvalid
	}
	extensionID, err := NormalizeExtensionID(input.ExtensionID)
	if err != nil {
		return Release{}, err
	}
	where, args, matchable, err := releasePredicate(ctx, tx, extensionID, input)
	if err != nil {
		return Release{}, err
	}
	if !matchable {
		return Release{}, nil
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, lockKey(extensionID)); err != nil {
		return Release{}, fmt.Errorf("lock extension attachment references: %w", err)
	}
	affected, released, err := deleteReferences(ctx, tx, where, args)
	if err != nil {
		return Release{}, err
	}
	if released == 0 {
		return Release{}, nil
	}
	counts, err := recount(ctx, tx, affected...)
	if err != nil {
		return Release{}, err
	}
	return Release{Released: released, Revision: latestRevision(counts)}, nil
}

// Count 是 release 的只读投影，供 Host Command 的 Plan/Prepare 阶段使用。
func (s *Store) Count(ctx context.Context, tx pgx.Tx, input ReleaseInput) (int, error) {
	if s == nil || tx == nil {
		return 0, ErrInvalid
	}
	extensionID, err := NormalizeExtensionID(input.ExtensionID)
	if err != nil {
		return 0, err
	}
	where, args, matchable, err := releasePredicate(ctx, tx, extensionID, input)
	if err != nil {
		return 0, err
	}
	if !matchable {
		return 0, nil
	}
	var total int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM attachment_references r WHERE `+where, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("count extension attachment references: %w", err)
	}
	return total, nil
}

// List 列出本扩展全部存活引用，供插件对账与前端回显。
func (s *Store) List(ctx context.Context, tx pgx.Tx, input ListInput) (List, error) {
	if s == nil || tx == nil {
		return List{}, ErrInvalid
	}
	extensionID, err := NormalizeExtensionID(input.ExtensionID)
	if err != nil {
		return List{}, err
	}
	page, perPage := normalizePage(input.Page, input.PerPage)
	var total int64
	if err := tx.QueryRow(ctx, `
		SELECT count(*)
		FROM attachment_references r
		JOIN attachments a ON a.id = r.attachment_id AND a.status = 'active'
		WHERE r.resource_type = $1 AND r.resource_id = 0
		  AND LEFT(r.context, length($2) + 1) = $2 || ':'
	`, ResourceType, extensionID).Scan(&total); err != nil {
		return List{}, fmt.Errorf("count extension attachment reference list: %w", err)
	}
	rows, err := tx.Query(ctx, `
		SELECT r.context, a.public_id, a.content_type, a.image_width, a.image_height,
		  a.size_bytes, a.sha256, r.created_at
		FROM attachment_references r
		JOIN attachments a ON a.id = r.attachment_id AND a.status = 'active'
		WHERE r.resource_type = $1 AND r.resource_id = 0
		  AND LEFT(r.context, length($2) + 1) = $2 || ':'
		ORDER BY r.created_at DESC, r.id DESC
		LIMIT $3 OFFSET $4
	`, ResourceType, extensionID, perPage, (page-1)*perPage)
	if err != nil {
		return List{}, fmt.Errorf("list extension attachment references: %w", err)
	}
	defer rows.Close()
	items := []Item{}
	for rows.Next() {
		var (
			item          Item
			context       string
			width, height sql.NullInt64
		)
		if err := rows.Scan(
			&context, &item.PublicID, &item.ContentType, &width, &height,
			&item.SizeBytes, &item.SHA256, &item.CreatedAt,
		); err != nil {
			return List{}, fmt.Errorf("scan extension attachment reference: %w", err)
		}
		// 存储键是 "<extID>:<context>"，对插件只暴露它自己的那一段。
		item.Context = strings.TrimPrefix(context, extensionID+":")
		item.URL = URLPath(item.PublicID)
		item.Width = int(width.Int64)
		item.Height = int(height.Int64)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return List{}, fmt.Errorf("iterate extension attachment references: %w", err)
	}
	return List{Items: items, Total: total, Page: page, PerPage: perPage}, nil
}

// ReleaseExtensionAttachmentReferences 释放某个扩展的全部附件引用。
//
// 卸载收口调用：插件被删掉后若引用悬空，附件永远不会被孤儿回收。禁用不释放，
// 禁用期间图标保持可读是预期行为。返回释放的引用行数，便于审计与日志。
func (s *Store) ReleaseExtensionAttachmentReferences(ctx context.Context, extensionID string) (int, error) {
	if s == nil || s.pool == nil {
		return 0, ErrInvalid
	}
	normalized, err := NormalizeExtensionID(extensionID)
	if err != nil {
		return 0, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin extension attachment reference release: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, lockKey(normalized)); err != nil {
		return 0, fmt.Errorf("lock extension attachment references: %w", err)
	}
	where := `r.resource_type = $1 AND r.resource_id = 0 AND LEFT(r.context, length($2) + 1) = $2 || ':'`
	affected, released, err := deleteReferences(ctx, tx, where, []any{ResourceType, normalized})
	if err != nil {
		return 0, err
	}
	if released == 0 {
		return 0, tx.Commit(ctx)
	}
	if _, err := recount(ctx, tx, affected...); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit extension attachment reference release: %w", err)
	}
	return released, nil
}

// releasePredicate 生成与 release 作用域一致的 WHERE 片段。
//
// 作用域必须同时限定 resource_type='extension' 与本扩展 context 前缀；不限定的话
// A 扩展就能删掉 B 扩展的引用 —— 这是安全边界，不是实现细节。
func releasePredicate(
	ctx context.Context,
	tx pgx.Tx,
	extensionID string,
	input ReleaseInput,
) (string, []any, bool, error) {
	contextValue := strings.TrimSpace(input.Context)
	publicID := strings.TrimSpace(input.PublicID)
	if (contextValue == "") == (publicID == "") {
		return "", nil, false, ErrInvalid
	}
	if contextValue != "" {
		_, scope, err := Scope(extensionID, contextValue)
		if err != nil {
			return "", nil, false, err
		}
		return `r.resource_type = $1 AND r.resource_id = 0 AND r.context = $2`,
			[]any{ResourceType, scope}, true, nil
	}
	var attachmentID int64
	err := tx.QueryRow(ctx, `SELECT id FROM attachments WHERE public_id = $1`, publicID).Scan(&attachmentID)
	if errors.Is(err, pgx.ErrNoRows) {
		// 附件已被回收：release 幂等成功，不报错。
		return "", nil, false, nil
	}
	if err != nil {
		return "", nil, false, fmt.Errorf("resolve extension release attachment: %w", err)
	}
	return `r.resource_type = $1 AND r.resource_id = 0 AND r.attachment_id = $2` +
			` AND LEFT(r.context, length($3) + 1) = $3 || ':'`,
		[]any{ResourceType, attachmentID, extensionID}, true, nil
}

// deleteReferences 返回受影响的附件 id（去重，供重算计数）与实际删除的引用行数。
//
// 两者必须分开：同一附件允许被同扩展多个 context 共享，所以「删了 2 行」和
// 「影响了 1 个附件」是不同的数字，插件侧的 released 语义是后者之外的行数。
func deleteReferences(ctx context.Context, tx pgx.Tx, where string, args []any) ([]int64, int, error) {
	rows, err := tx.Query(ctx, `DELETE FROM attachment_references AS r WHERE `+where+` RETURNING r.attachment_id`, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("delete extension attachment references: %w", err)
	}
	defer rows.Close()
	seen := map[int64]bool{}
	affected := []int64{}
	released := 0
	for rows.Next() {
		var attachmentID int64
		if err := rows.Scan(&attachmentID); err != nil {
			return nil, 0, fmt.Errorf("scan released extension attachment reference: %w", err)
		}
		released++
		if !seen[attachmentID] {
			seen[attachmentID] = true
			affected = append(affected, attachmentID)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate released extension attachment references: %w", err)
	}
	return affected, released, nil
}

type referenceCount struct {
	Count     int
	UpdatedAt time.Time
}

// recount 按引用行重算 reference_count，而不是盲目 ±1。
//
// 重算是自愈的：即使历史数据已有漂移，下一次 claim/release 也会把它拉回
// reference_count == 引用行数 的不变式。只有计数真正变化时才推进 updated_at，
// 这样「重复认领同一附件同一 context」不会无谓地让插件手里的修订号失效。
func recount(ctx context.Context, tx pgx.Tx, attachmentIDs ...int64) (map[int64]referenceCount, error) {
	counts := make(map[int64]referenceCount, len(attachmentIDs))
	if len(attachmentIDs) == 0 {
		return counts, nil
	}
	rows, err := tx.Query(ctx, `
		UPDATE attachments a
		SET reference_count = live.count,
		    updated_at = CASE WHEN a.reference_count <> live.count THEN now() ELSE a.updated_at END
		FROM (
		  SELECT target.id,
		    (SELECT count(*) FROM attachment_references r WHERE r.attachment_id = target.id) AS count
		  FROM unnest($1::bigint[]) AS target(id)
		) AS live
		WHERE a.id = live.id
		RETURNING a.id, a.reference_count, a.updated_at
	`, attachmentIDs)
	if err != nil {
		return nil, fmt.Errorf("recount attachment references: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			id    int64
			count referenceCount
		)
		if err := rows.Scan(&id, &count.Count, &count.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan attachment reference count: %w", err)
		}
		counts[id] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate attachment reference counts: %w", err)
	}
	return counts, nil
}

func latestRevision(counts map[int64]referenceCount) time.Time {
	var latest time.Time
	for _, count := range counts {
		if count.UpdatedAt.After(latest) {
			latest = count.UpdatedAt
		}
	}
	return latest
}

func normalizePage(page int, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > maxListPerPage {
		perPage = maxListPerPage
	}
	return page, perPage
}
