package extensionsruntime_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	capabilities "github.com/zhuchunshu/sforum/apps/api/app/Support/Capabilities"
	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	extensionpackage "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionPackage"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
)

// TestGamificationPluginGrantsBadgesThroughObserveDelivery proves M3: the
// protected sforum.gamification plugin receives real observe envelopes
// (user.registered / topic.created / comment.created) through the Host
// delivery path, snapshots registration time, credits growth value once
// under River-style replay, and grants threshold badges into its own schema.
func TestGamificationPluginGrantsBadgesThroughObserveDelivery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping gamification plugin subprocess build in short mode")
	}
	extension := buildGamificationExtension(t)
	if extension.ID != "sforum.gamification" || len(extension.Manifest.Events) != 3 {
		t.Fatalf("gamification manifest incomplete: id=%q events=%d", extension.ID, len(extension.Manifest.Events))
	}
	for _, event := range extension.Manifest.Events {
		if event.Kind != "observe" {
			t.Fatalf("gamification event %s must be observe, got %s", event.Name, event.Kind)
		}
	}

	databaseURL := commerceTestDatabaseURL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	t.Cleanup(pool.Close)
	schemaID, err := extensionsruntime.ExtensionDatabaseIdentifiersFor(extension.ID)
	if err != nil {
		t.Fatalf("resolve gamification schema: %v", err)
	}
	gamificationSchema := schemaID.Schema // 宿主 own_schema 租约分配的物理 schema
	dropGamificationSchema(t, ctx, pool, gamificationSchema)
	t.Cleanup(func() { dropGamificationSchema(t, ctx, pool, gamificationSchema) })
	versionID, err := seedCommerceOwnSchemaGrant(t, ctx, pool, extension)
	if err != nil {
		t.Fatalf("seed own_schema grant: %v", err)
	}
	extension.ActiveVersionID = versionID
	artifact := extensionsruntime.ExtensionDatabaseArtifact{
		ExtensionID: extension.ID, Version: extension.Version,
		VersionID: extension.ActiveVersionID, PackageDigest: extension.PackageDigest,
	}
	t.Cleanup(func() { cleanupCommerceOwnSchemaGrant(t, pool, extension.ID) })

	dbRegistry := extensionsruntime.NewPostgresExtensionDatabaseRegistry(pool, nil)
	seed, err := dbRegistry.IssueRuntimeLease(ctx, extensionsruntime.ExtensionDatabaseRuntimeLeaseIssue{
		Artifact: artifact, RuntimeInstanceID: "gamification-grant-seed",
		Authority: extensionsruntime.ExtensionDatabaseLeaseAuthority{
			Kind: extensionsruntime.ExtensionDatabaseLeaseIssuerActor, ActorUserID: 903, AuditEventID: 1903,
		},
	})
	if err != nil {
		t.Fatalf("seed lease: %v", err)
	}
	if _, err := dbRegistry.RevokeRuntimeLease(ctx, extensionsruntime.ExtensionDatabaseRuntimeLeaseRef{
		Artifact: artifact, RuntimeInstanceID: seed.RuntimeInstanceID, LeaseID: seed.LeaseID,
	}, extensionsruntime.ExtensionDatabaseLeaseAuthority{Kind: extensionsruntime.ExtensionDatabaseLeaseIssuerHost}); err != nil {
		t.Fatalf("revoke seed lease: %v", err)
	}

	starter := extensionsruntime.NewProtocolStarter(extensionsruntime.ProtocolStarterConfig{
		Trust: staticRuntimeTrust{identity: extensions.RuntimeTrustIdentity{
			TrustGrantID: "gamification-reference", ImpactDigest: extension.PackageDigest,
		}},
		DatabaseLeases:                 dbRegistry,
		DatabaseLeaseHeartbeatInterval: 100 * time.Millisecond,
		DatabaseLeaseOperationTimeout:  5 * time.Second,
	})
	manager := extensionsruntime.NewManager(extensionsruntime.ManagerConfig{Starter: starter})
	if err := manager.Start(ctx, extension); err != nil {
		t.Fatalf("start gamification plugin: %v", err)
	}
	stopped := false
	t.Cleanup(func() {
		if !stopped {
			_ = manager.Stop(context.Background(), extension)
		}
	})
	active, err := manager.ActiveRuntimeInstance(extension.ID)
	if err != nil || active.Identity.InstanceID == "" {
		t.Fatalf("expected protocol v2 subprocess: %#v err=%v", active, err)
	}

	// 默认徽章/等级种子经插件 API 层在 M4 接入；此处直接 SQL 种一条阈值徽章
	// （集成测试关注投递链路而非种子）。种子前先等插件完成 DDL。
	waitGamificationTable(t, ctx, pool, gamificationSchema)
	if _, err := pool.Exec(ctx, `
INSERT INTO `+gamificationSchema+`.badge_definitions (slug, name, rarity, rule, grants_points)
VALUES ('first-topic', '初露锋芒', 'common', '{"kind":"threshold","metric":"topic_count","threshold":1}', 5)
ON CONFLICT (slug) DO NOTHING`); err != nil {
		t.Fatalf("seed badge: %v", err)
	}

	// --- 注册事件：快照 + 注册成长值 ---
	register := appevents.Envelope{Name: "user.registered", Kind: appevents.KindObserve,
		CorrelationID: "gamification-it-register", Payload: map[string]any{
			"userId": 4242, "username": "tester", "email": "t@example.com", "locale": "zh-CN",
		}}
	if result := manager.Deliver(ctx, extension.ID, 1, register); !result.OK {
		t.Fatalf("register delivery: %+v", result)
	}
	assertGamificationRow(t, ctx, pool,
		`SELECT count(*) FROM `+gamificationSchema+`.user_registrations WHERE user_id = 4242`, 1,
		"registration snapshot")
	assertGamificationRow(t, ctx, pool,
		`SELECT COALESCE(SUM(amount),0) FROM `+gamificationSchema+`.user_points_ledger WHERE user_id = 4242`, 10,
		"registration growth credit")

	// --- 首帖事件：成长值 + 阈值徽章 ---
	topic := appevents.Envelope{Name: "topic.created", Kind: appevents.KindObserve,
		CorrelationID: "gamification-it-topic", Payload: map[string]any{
			"topicId": 9001, "authorUserId": 4242, "categorySlug": "tech", "title": "Hello",
		}}
	if result := manager.Deliver(ctx, extension.ID, 2, topic); !result.OK {
		t.Fatalf("topic delivery: %+v", result)
	}
	assertGamificationRow(t, ctx, pool,
		`SELECT count(*) FROM `+gamificationSchema+`.user_badges WHERE user_id = 4242 AND badge_slug = 'first-topic'`, 1,
		"threshold badge grant")
	assertGamificationRow(t, ctx, pool,
		`SELECT COALESCE(SUM(amount),0) FROM `+gamificationSchema+`.user_points_ledger WHERE user_id = 4242`, 20,
		"growth + badge points")

	// --- River 风格重放（不同 delivery id / correlation）：不双计不双授 ---
	replay := appevents.Envelope{Name: "topic.created", Kind: appevents.KindObserve,
		CorrelationID: "gamification-it-topic-replayed", Payload: map[string]any{
			"topicId": 9001, "authorUserId": 4242, "categorySlug": "tech", "title": "Hello",
		}}
	if result := manager.Deliver(ctx, extension.ID, 3, replay); !result.OK {
		t.Fatalf("replay delivery: %+v", result)
	}
	assertGamificationRow(t, ctx, pool,
		`SELECT count(*) FROM `+gamificationSchema+`.user_badges WHERE user_id = 4242 AND badge_slug = 'first-topic'`, 1,
		"replay badge idempotency")
	assertGamificationRow(t, ctx, pool,
		`SELECT COALESCE(SUM(amount),0) FROM `+gamificationSchema+`.user_points_ledger WHERE user_id = 4242`, 20,
		"replay ledger idempotency")
	assertGamificationRow(t, ctx, pool,
		`SELECT value FROM `+gamificationSchema+`.user_metric_counters WHERE user_id = 4242 AND metric = 'topic_count'`, 1,
		"replay counter idempotency")

	// --- 评论事件 ---
	comment := appevents.Envelope{Name: "comment.created", Kind: appevents.KindObserve,
		CorrelationID: "gamification-it-comment", Payload: map[string]any{
			"commentId": 5001, "topicId": 9001, "authorUserId": 4242, "parentId": nil,
		}}
	if result := manager.Deliver(ctx, extension.ID, 4, comment); !result.OK {
		t.Fatalf("comment delivery: %+v", result)
	}
	assertGamificationRow(t, ctx, pool,
		`SELECT value FROM `+gamificationSchema+`.user_metric_counters WHERE user_id = 4242 AND metric = 'comment_count'`, 1,
		"comment counter")

	// --- 未订阅事件拒绝（fail-fast 到插件层）---
	unknown := appevents.Envelope{Name: "topic.updated", Kind: appevents.KindObserve,
		CorrelationID: "gamification-it-unknown", Payload: map[string]any{
			"topicId": 9001, "actorUserId": 4242,
		}}
	result := manager.Deliver(ctx, extension.ID, 5, unknown)
	if result.OK {
		t.Fatalf("unknown event delivery must not be OK: %+v", result)
	}

	stopped = true
	if err := manager.Stop(ctx, extension); err != nil {
		t.Fatalf("stop: %v", err)
	}
}

func assertGamificationRow(t *testing.T, ctx context.Context, pool *pgxpool.Pool, query string, want int64, what string) {
	t.Helper()
	var got int64
	if err := pool.QueryRow(ctx, query).Scan(&got); err != nil {
		t.Fatalf("%s: %v", what, err)
	}
	if got != want {
		t.Fatalf("%s: got %d want %d", what, got, want)
	}
}

func dropGamificationSchema(t *testing.T, ctx context.Context, pool *pgxpool.Pool, schema string) {
	t.Helper()
	_, _ = pool.Exec(ctx, `DROP SCHEMA IF EXISTS `+schema+` CASCADE`)
}

// waitGamificationTable 等插件子进程完成 EnsureSchema（租约 DDL 异步于 Start）。
func waitGamificationTable(t *testing.T, ctx context.Context, pool *pgxpool.Pool, schema string) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for {
		var exists bool
		if err := pool.QueryRow(ctx, `
SELECT EXISTS (SELECT 1 FROM information_schema.tables
WHERE table_schema = $1 AND table_name = 'badge_definitions')`, schema).Scan(&exists); err == nil && exists {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("gamification schema %s was not initialized in time", schema)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func buildGamificationExtension(t *testing.T) extensions.Extension {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	fixtureRoot := filepath.Clean(filepath.Join(filepath.Dir(file),
		"../../../../../../extensions/builtin/plugins/sforum-gamification"))
	packageRoot := filepath.Join(t.TempDir(), "sforum.gamification")
	if err := os.CopyFS(packageRoot, os.DirFS(fixtureRoot)); err != nil {
		t.Fatalf("copy gamification plugin: %v", err)
	}
	repositoryRoot := filepath.Clean(filepath.Join(fixtureRoot, "../../../.."))
	goModPath := filepath.Join(packageRoot, "backend", "go.mod")
	goMod, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatal(err)
	}
	goMod = []byte(strings.ReplaceAll(string(goMod), "../../../../../apps/api", filepath.Join(repositoryRoot, "apps/api")))
	if err := os.WriteFile(goModPath, goMod, 0o600); err != nil {
		t.Fatal(err)
	}
	binaryPath := filepath.Join(packageRoot, "backend", "plugin")
	build := exec.Command("go", "build", "-mod=mod", "-trimpath", "-buildvcs=false", "-o", binaryPath, ".")
	build.Dir = filepath.Join(packageRoot, "backend")
	build.Env = append(os.Environ(), "CGO_ENABLED=0", "GOWORK=off")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build gamification plugin: %v\n%s", err, output)
	}
	// 重算 backend digest：源码树里的 digest 属于上次构建，与本测试二进制不同。
	manifestPath := filepath.Join(packageRoot, extensionmanifest.ManifestFileName)
	manifestBody, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	patchManifestBackendDigest(t, manifestPath, string(manifestBody), fileSHA256(t, binaryPath))
	manifest, err := extensionmanifest.LoadPackage(packageRoot)
	if err != nil {
		t.Fatalf("load gamification package: %v", err)
	}
	packageDigest, err := extensionpackage.DigestTree(packageRoot)
	if err != nil {
		t.Fatal(err)
	}
	return extensions.Extension{
		ID: manifest.ID, Name: manifest.Name, Version: manifest.Version, Type: extensions.TypePlugin,
		Status: extensions.StatusEnabled, Source: extensions.SourceBuiltin,
		PackagePath: packageRoot, PackageDigest: packageDigest, Manifest: manifest, ActiveVersionID: 701,
		CapabilityGrants: []extensions.CapabilityGrant{{Key: capabilities.HostAPI, Risk: capabilities.RiskLow}},
	}
}

// patchManifestBackendDigest 只替换 backend.digest 与对应 packageFiles 的
// executable digest（JSON 感知，避免文本替换误伤）。
func patchManifestBackendDigest(t *testing.T, path, body, digest string) {
	t.Helper()
	var manifest map[string]any
	if err := json.Unmarshal([]byte(body), &manifest); err != nil {
		t.Fatalf("manifest json: %v", err)
	}
	if backend, ok := manifest["backend"].(map[string]any); ok {
		backend["digest"] = digest
	}
	if files, ok := manifest["packageFiles"].([]any); ok {
		for _, item := range files {
			if file, ok := item.(map[string]any); ok && file["kind"] == "executable" {
				file["digest"] = digest
			}
		}
	}
	updated, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(path, append(updated, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}
