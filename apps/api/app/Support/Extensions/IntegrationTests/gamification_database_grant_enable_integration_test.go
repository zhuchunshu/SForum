package extensionsruntime_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	audit "github.com/zhuchunshu/sforum/apps/api/app/Support/Audit"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
)

// 这一组测试补上 M1–M5 集成测试的空白：既有测试手动构造 Actor 权威预置授权行，
// 因此从未覆盖真实启用路径。真实启用的顺序是
// service → ActivationCoordinator(actor, enable) → runtime.Start(ctx) → 数据库租约，
// 其中 only-actor 分支才是允许创建 own_schema 精确授权的入口。

// TestGamificationPluginProvisionsOwnSchemaGrantOnOperatorBoundStart 证明操作者
// 发起的启动（上下文携带 actor 与审计事件 id）可以在没有任何预置授权行的情况下
// 创建精确授权并真正把插件跑起来。
func TestGamificationPluginProvisionsOwnSchemaGrantOnOperatorBoundStart(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping gamification plugin subprocess build in short mode")
	}
	extension, ctx, pool, gamificationSchema := prepareGamificationDatabaseGrantFixture(t, "operator")
	dbRegistry := extensionsruntime.NewPostgresExtensionDatabaseRegistry(pool, nil)
	starter := gamificationGrantStarter(extension, dbRegistry)
	manager := extensionsruntime.NewManager(extensionsruntime.ManagerConfig{Starter: starter})

	actorUserID, auditEventID := seedDatabaseGrantActorAudit(t, ctx, pool)
	startCtx := extensions.WithDatabaseGrantAuthority(ctx, extensions.DatabaseGrantAuthority{
		ActorUserID: actorUserID, AuditEventID: auditEventID,
	})
	if err := manager.Start(startCtx, extension); err != nil {
		t.Fatalf("operator-bound start without a pre-seeded grant: %v", err)
	}
	stopped := false
	t.Cleanup(func() {
		if !stopped {
			_ = manager.Stop(context.Background(), extension)
		}
	})

	assertRowCount(t, ctx, pool, `
SELECT count(*) FROM extension_database_grant_powers AS powers
JOIN extension_database_grants AS grants ON grants.id = powers.grant_id
WHERE grants.extension_id = $1 AND grants.status = 'active' AND powers.power = 'own_schema'`,
		[]any{extension.ID}, 1, "operator-bound exact own_schema power")
	assertGamificationGrantAuditActor(t, ctx, pool, extension.ID, actorUserID)
	// 租约真的可用：插件在自己的 schema 里完成了 DDL。
	waitGamificationTable(t, ctx, pool, gamificationSchema)

	stopped = true
	if err := manager.Stop(ctx, extension); err != nil {
		t.Fatalf("stop: %v", err)
	}
}

// TestGamificationPluginHostStartWithoutGrantFailsClosed 证明 Host 权威启动（启动期
// 没有操作者）不会静默创建插件数据库授权：缺少授权行时启动失败关闭。
func TestGamificationPluginHostStartWithoutGrantFailsClosed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping gamification plugin subprocess build in short mode")
	}
	extension, ctx, pool, gamificationSchema := prepareGamificationDatabaseGrantFixture(t, "host")
	dbRegistry := extensionsruntime.NewPostgresExtensionDatabaseRegistry(pool, nil)
	manager := extensionsruntime.NewManager(extensionsruntime.ManagerConfig{
		Starter: gamificationGrantStarter(extension, dbRegistry),
	})

	err := manager.Start(ctx, extension)
	if err == nil {
		_ = manager.Stop(context.Background(), extension)
		t.Fatal("Host-authority start created a plugin database grant")
	}
	if !errors.Is(err, extensionsruntime.ErrExtensionDatabaseGrantNotFound) {
		t.Fatalf("Host start without a grant must fail closed with the missing exact grant: %v", err)
	}
	assertRowCount(t, ctx, pool, `
SELECT count(*) FROM extension_database_grants WHERE extension_id = $1`,
		[]any{extension.ID}, 0, "no grant created by Host start")
	var initialized bool
	if err := pool.QueryRow(ctx, `
SELECT EXISTS (SELECT 1 FROM information_schema.tables
WHERE table_schema = $1 AND table_name = 'badge_definitions')`, gamificationSchema).Scan(&initialized); err != nil {
		t.Fatalf("inspect plugin schema: %v", err)
	}
	if initialized {
		t.Fatalf("plugin schema %s was initialized without an operator-bound grant", gamificationSchema)
	}
}

// prepareGamificationDatabaseGrantFixture 构建一个只缺「精确授权行」的启用现场：
// 扩展与版本行已入库（授权声明来自 extension_versions.manifest），插件 schema 与
// 历史授权行被清空。
func prepareGamificationDatabaseGrantFixture(
	t *testing.T,
	label string,
) (extensions.Extension, context.Context, *pgxpool.Pool, string) {
	t.Helper()
	extension := buildGamificationExtension(t)
	databaseURL := commerceTestDatabaseURL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	t.Cleanup(cancel)
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	t.Cleanup(pool.Close)
	schemaID, err := extensionsruntime.ExtensionDatabaseIdentifiersFor(extension.ID)
	if err != nil {
		t.Fatalf("resolve %s schema: %v", label, err)
	}
	gamificationSchema := schemaID.Schema
	dropGamificationSchema(t, ctx, pool, gamificationSchema)
	resetExtensionDatabaseGrants(t, ctx, pool, extension.ID)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		dropGamificationSchema(t, cleanupCtx, pool, gamificationSchema)
		resetExtensionDatabaseGrants(t, cleanupCtx, pool, extension.ID)
		cleanupCommerceOwnSchemaGrant(t, pool, extension.ID)
	})
	versionID, err := seedCommerceOwnSchemaGrant(t, ctx, pool, extension)
	if err != nil {
		t.Fatalf("seed %s extension version: %v", label, err)
	}
	extension.ActiveVersionID = versionID
	return extension, ctx, pool, gamificationSchema
}

func gamificationGrantStarter(
	extension extensions.Extension,
	registry extensionsruntime.RuntimeDatabaseLeaseRegistry,
) *extensionsruntime.ProtocolStarter {
	return extensionsruntime.NewProtocolStarter(extensionsruntime.ProtocolStarterConfig{
		Trust: staticRuntimeTrust{identity: extensions.RuntimeTrustIdentity{
			TrustGrantID: "gamification-enable-database-grant", ImpactDigest: extension.PackageDigest,
		}},
		DatabaseLeases:                 registry,
		DatabaseLeaseHeartbeatInterval: 100 * time.Millisecond,
		DatabaseLeaseOperationTimeout:  5 * time.Second,
	})
}

// seedDatabaseGrantActorAudit 落一条操作者归属的授权审计事件，模拟
// ActivationCoordinator 在启用前写入的那一条。
func seedDatabaseGrantActorAudit(t *testing.T, ctx context.Context, pool *pgxpool.Pool) (int64, int64) {
	t.Helper()
	var actorUserID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM users ORDER BY id LIMIT 1`).Scan(&actorUserID); err != nil {
		actorUserID = 0
	}
	var auditEventID int64
	if err := pool.QueryRow(ctx, `
INSERT INTO audit_events (action, actor_user_id, metadata)
VALUES ($1, NULLIF($2::bigint, 0), jsonb_build_object('extensionId', 'sforum.gamification'))
RETURNING id`, audit.ActionExtensionDatabaseGrant, actorUserID).Scan(&auditEventID); err != nil {
		t.Fatalf("seed database grant audit event: %v", err)
	}
	if actorUserID <= 0 {
		actorUserID = 1
	}
	return actorUserID, auditEventID
}

func assertGamificationGrantAuditActor(t *testing.T, ctx context.Context, pool *pgxpool.Pool, extensionID string, actorUserID int64) {
	t.Helper()
	var grantedBy int64
	err := pool.QueryRow(ctx, `
SELECT granted_by_user_id FROM extension_database_grants
WHERE extension_id = $1 AND status = 'active'`, extensionID).Scan(&grantedBy)
	if err != nil {
		t.Fatalf("read grant actor: %v", err)
	}
	if grantedBy != actorUserID {
		t.Fatalf("grant actor=%d want %d", grantedBy, actorUserID)
	}
}

// resetExtensionDatabaseGrants 清空该扩展的运行时租约、凭据、能力位与授权行
// （按外键依赖顺序）。
func resetExtensionDatabaseGrants(t *testing.T, ctx context.Context, pool *pgxpool.Pool, extensionID string) {
	t.Helper()
	for _, statement := range []string{
		`DELETE FROM extension_database_runtime_leases WHERE extension_id = $1`,
		`DELETE FROM extension_database_credentials WHERE grant_id IN (
			SELECT id FROM extension_database_grants WHERE extension_id = $1)`,
		`DELETE FROM extension_database_grant_powers WHERE grant_id IN (
			SELECT id FROM extension_database_grants WHERE extension_id = $1)`,
		`DELETE FROM extension_database_grants WHERE extension_id = $1`,
	} {
		if _, err := pool.Exec(ctx, statement, extensionID); err != nil {
			t.Fatalf("reset extension database grants: %v", err)
		}
	}
}

func assertRowCount(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	query string,
	args []any,
	want int64,
	what string,
) {
	t.Helper()
	var got int64
	if err := pool.QueryRow(ctx, query, args...).Scan(&got); err != nil {
		t.Fatalf("%s: %v", what, err)
	}
	if got != want {
		t.Fatalf("%s: got %d want %d", what, got, want)
	}
}
