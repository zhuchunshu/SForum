package extensionsruntime_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	routes "github.com/zhuchunshu/sforum/apps/api/app/Support/Routes"
)

// TestGamificationAdminRoutesManageBadges proves M4 admin surface:
// seed defaults → manual grant → grant log → revoke → public badge wall,
// all through the declared admin routes with the plugin permission guard.
func TestGamificationAdminRoutesManageBadges(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping gamification plugin subprocess build in short mode")
	}
	extension := buildGamificationExtension(t)
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
		t.Fatalf("resolve schema: %v", err)
	}
	schema := schemaID.Schema
	dropGamificationSchema(t, ctx, pool, schema)
	t.Cleanup(func() { dropGamificationSchema(t, ctx, pool, schema) })
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
		Artifact: artifact, RuntimeInstanceID: "gamification-admin-seed",
		Authority: extensionsruntime.ExtensionDatabaseLeaseAuthority{
			Kind: extensionsruntime.ExtensionDatabaseLeaseIssuerActor, ActorUserID: 904, AuditEventID: 1904,
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
			TrustGrantID: "gamification-admin", ImpactDigest: extension.PackageDigest,
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
	waitGamificationTable(t, ctx, pool, schema)

	// 真实 routes.Registry：管理路由必须经完整 Route Plan，禁止仅 InvokeRouteInstance。
	routeReg := routes.NewRegistry()
	if _, err := routeReg.Publish(routes.Publication{
		Plugins: []routes.PluginRouteSet{{
			Artifact: routes.PluginArtifact{
				ExtensionID: extension.ID, ExtensionVersion: extension.Version,
				PackageDigest: extension.PackageDigest, RuntimeInstanceID: active.Identity.InstanceID,
			},
			Routes: extension.Manifest.Routes,
			Guards: extension.Manifest.Guards,
		}},
	}); err != nil {
		t.Fatalf("publish gamification routes: %v", err)
	}

	invoke := func(method, path string, routeID string, pathParams map[string]string, body map[string]any) map[string]any {
		t.Helper()
		declared := gamificationRouteSchemas(t, extension, routeID)
		response := invokeGamificationRoute(t, manager, active.Identity, extensionsruntime.ProtocolV2RouteRequest{
			RouteID: routeID, ContractVersion: routeID + "@1",
			RouteAction:     extensionmanifest.RouteActionAdd,
			InvocationStage: extensionsruntime.ProtocolV2RouteInvocationStageHandler,
			Method:          method, Path: path,
			PathParameters: pathParams,
			Body:           body, BodyPresent: body != nil,
			RequestSchema:  declared.request,
			ResponseSchema: declared.response,
			Authority:      commerceFilteredHostAuthority(),
			Actor:          extensionsruntime.NewProtocolV2RouteActor(7, true, map[string]bool{"*": true}),
			Timeout:        5 * time.Second,
		})
		if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
			t.Fatalf("%s %s status=%d body=%v", method, path, response.StatusCode, response.Body)
		}
		raw, _ := json.Marshal(response.Body)
		var decoded map[string]any
		_ = json.Unmarshal(raw, &decoded)
		return decoded
	}

	// --- 一键恢复推荐默认 ---
	seeded := invoke(http.MethodPost, "/api/v1/admin/gamification/seed-defaults",
		"sforum.gamification.route.admin.seed-defaults", nil, map[string]any{})
	if seeded["badges"] != float64(8) || seeded["levels"] != float64(5) {
		t.Fatalf("seed defaults: %#v", seeded)
	}

	// --- 手动授予 manual 徽章 ---
	granted := invoke(http.MethodPost, "/api/v1/admin/gamification/grants",
		"sforum.gamification.route.admin.grants", nil,
		map[string]any{"userId": 4242, "badgeSlug": "outstanding"})
	if granted["granted"] != true {
		t.Fatalf("manual grant: %#v", granted)
	}
	// 重复授予幂等。
	again := invoke(http.MethodPost, "/api/v1/admin/gamification/grants",
		"sforum.gamification.route.admin.grants", nil,
		map[string]any{"userId": 4242, "badgeSlug": "outstanding"})
	if again["granted"] != false || again["already"] != true {
		t.Fatalf("manual grant replay: %#v", again)
	}

	// --- 授予日志 ---
	logBody := invoke(http.MethodGet, "/api/v1/admin/gamification/users/4242/badges",
		"sforum.gamification.route.admin.grant-log", map[string]string{"userId": "4242"}, nil)
	grants := decodeIntegrationJSONArray(t, logBody["grants"])
	if len(grants) != 1 {
		t.Fatalf("grant log: %#v", logBody)
	}

	// --- 公开徽章墙（manual 徽章应可见）---
	wall := invoke(http.MethodGet, "/api/v1/gamification/users/4242/achievements",
		"sforum.gamification.route.profile.achievements", map[string]string{"userId": "4242"}, nil)
	badges := decodeIntegrationJSONArray(t, wall["badges"])
	if len(badges) != 1 {
		t.Fatalf("badge wall: %#v", wall)
	}

	// --- 撤销 ---
	revoked := invoke(http.MethodDelete, "/api/v1/admin/gamification/users/4242/badges/outstanding",
		"sforum.gamification.route.admin.grant",
		map[string]string{"userId": "4242", "badgeSlug": "outstanding"}, nil)
	if revoked["revoked"] != true {
		t.Fatalf("revoke: %#v", revoked)
	}
	afterRevoke := invoke(http.MethodGet, "/api/v1/gamification/users/4242/achievements",
		"sforum.gamification.route.profile.achievements", map[string]string{"userId": "4242"}, nil)
	badges = decodeIntegrationJSONArray(t, afterRevoke["badges"])
	if len(badges) != 0 {
		t.Fatalf("badge wall after revoke: %#v", afterRevoke)
	}

	// --- threshold 徽章拒绝 manual 路径 ---
	denied, err := invokeGamificationRouteErr(t, manager, active.Identity, extensionsruntime.ProtocolV2RouteRequest{
		RouteID: "sforum.gamification.route.admin.grants",
		ContractVersion: "sforum.gamification.route.admin.grants@1",
		RouteAction:     extensionmanifest.RouteActionAdd,
		InvocationStage: extensionsruntime.ProtocolV2RouteInvocationStageHandler,
		Method:          http.MethodPost, Path: "/api/v1/admin/gamification/grants",
		Body:           map[string]any{"userId": 4242, "badgeSlug": "ten-topics"}, BodyPresent: true,
		RequestSchema:  "sforum.gamification.route.admin.grants.request@1",
		ResponseSchema: "sforum.gamification.route.admin.grants.response@1",
		Authority:      commerceFilteredHostAuthority(),
		Actor:          extensionsruntime.NewProtocolV2RouteActor(7, true, map[string]bool{"*": true}),
		Timeout:        5 * time.Second,
	})
	if err != nil {
		t.Fatalf("threshold grant request: %v", err)
	}
	// threshold 徽章走不了 manual 路径：返回 422 + 稳定 reason。
	if denied.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("threshold badge must be rejected manually, status=%d body=%v", denied.StatusCode, denied.Body)
	}

	stopped = true
	if err := manager.Stop(ctx, extension); err != nil {
		t.Fatalf("stop: %v", err)
	}
	_ = strings.TrimSpace(schema)
}

// invokeGamificationRoute 经运行时调用租约执行一条声明路由（与生产
// Route Plan 一致：先 AcquireRuntimeCall 再 InvokeRouteInstance）。
func invokeGamificationRoute(t *testing.T, manager *extensionsruntime.Manager,
	identity extensionsruntime.RuntimeInstanceIdentity,
	request extensionsruntime.ProtocolV2RouteRequest,
) extensionsruntime.ProtocolV2RouteResponse {
	t.Helper()
	lease, err := manager.AcquireRuntimeCall(context.Background(), identity, extensionsruntime.RuntimeCallRoute)
	if err != nil {
		t.Fatalf("acquire runtime call: %v", err)
	}
	defer lease.Release()
	response, err := manager.InvokeRouteInstance(lease.Context, identity, request)
	if err != nil {
		t.Fatalf("invoke route %s: %v", request.RouteID, err)
	}
	return response
}

func invokeGamificationRouteErr(t *testing.T, manager *extensionsruntime.Manager,
	identity extensionsruntime.RuntimeInstanceIdentity,
	request extensionsruntime.ProtocolV2RouteRequest,
) (extensionsruntime.ProtocolV2RouteResponse, error) {
	t.Helper()
	lease, err := manager.AcquireRuntimeCall(context.Background(), identity, extensionsruntime.RuntimeCallRoute)
	if err != nil {
		t.Fatalf("acquire runtime call: %v", err)
	}
	defer lease.Release()
	return manager.InvokeRouteInstance(lease.Context, identity, request)
}

type gamificationSchemas struct{ request, response string }

// gamificationRouteSchemas 取 manifest 声明的精确 schema 引用：Host 的冻结
// 校验要求请求与响应的 schema 引用与声明完全一致。
func gamificationRouteSchemas(t *testing.T, extension extensions.Extension, routeID string) gamificationSchemas {
	t.Helper()
	for _, route := range extension.Manifest.Routes {
		if route.ID == routeID {
			return gamificationSchemas{request: route.RequestSchema, response: route.ResponseSchema}
		}
	}
	t.Fatalf("route %s is not declared", routeID)
	return gamificationSchemas{}
}

// decodeIntegrationJSONArray 解出以 JSON 字符串回传的结构化数组
// （TypedDocument 值域为原始类型）。
func decodeIntegrationJSONArray(t *testing.T, value any) []map[string]any {
	t.Helper()
	text, ok := value.(string)
	if !ok {
		t.Fatalf("expected JSON string payload, got %T (%v)", value, value)
	}
	var result []map[string]any
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	return result
}
