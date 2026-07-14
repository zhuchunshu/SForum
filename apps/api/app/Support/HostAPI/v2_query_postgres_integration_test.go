package hostapi

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	hostv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/host/v2"
	protocolv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/protocol/v2"
	"google.golang.org/protobuf/proto"
)

type postgresQueryHarness struct {
	ctx      context.Context
	pool     *pgxpool.Pool
	engine   *protocolV2QueryEngine
	identity *protocolv2.ExtensionIdentity
	grantID  int64
}

func TestPostgresProtocolV2QueryRuntimeStableRowsPaginationAndPIIAbsence(t *testing.T) {
	h := newPostgresQueryHarness(t)
	ctx := ContextWithProtocolV2RuntimeIdentity(h.ctx, h.identity)

	user := &hostv2.QueryRequest{
		Context: testProtocolV2RequestContext(), QueryId: QuerySafeUserByID, PlanVersion: QueryStableCorePlanVersion,
		Filters: []*hostv2.QueryFilter{{Field: "id", Operator: "eq", Value: queryParameter(t, QueryInt64ParameterSchemaID, "1")}},
	}
	user.Context.GrantedAuthority = []*protocolv2.AuthorityGrant{{Key: "database.raw_core", Source: "forged"}}
	response := h.engine.execute(ctx, user)
	if response.GetError() != nil || len(response.GetRows()) != 1 {
		t.Fatalf("safe user response = %#v", response)
	}
	userRow := response.GetRows()[0].GetValue().AsMap()
	for _, forbidden := range []string{"email", "password_hash", "locale", "is_admin", "token"} {
		if _, exists := userRow[forbidden]; exists {
			t.Fatalf("safe user leaked %s: %#v", forbidden, userRow)
		}
	}
	if userRow["id"] != "1" || userRow["username"] != "alice" {
		t.Fatalf("safe user row = %#v", userRow)
	}

	topics := &hostv2.QueryRequest{
		Context: testProtocolV2RequestContext(), QueryId: QueryPublicTopicsList, PlanVersion: QueryStableCorePlanVersion,
		Fields: []string{"id", "title", "last_activity_at"}, Page: &protocolv2.PageRequest{Limit: 2},
	}
	first := h.engine.execute(ctx, topics)
	if first.GetError() != nil || len(first.GetRows()) != 2 || !first.GetPage().GetHasMore() || first.GetPage().GetNextCursor() == "" {
		t.Fatalf("topic first page = %#v", first)
	}
	secondRequest := proto.Clone(topics).(*hostv2.QueryRequest)
	secondRequest.Page = &protocolv2.PageRequest{Limit: 2, Cursor: first.GetPage().GetNextCursor()}
	second := h.engine.execute(ctx, secondRequest)
	if second.GetError() != nil || len(second.GetRows()) != 1 || second.GetPage().GetHasMore() {
		t.Fatalf("topic second page = %#v", second)
	}

	topic := &hostv2.QueryRequest{
		Context: testProtocolV2RequestContext(), QueryId: QueryPublicTopicByID, PlanVersion: QueryStableCorePlanVersion,
		Fields:  []string{"id", "title"},
		Filters: []*hostv2.QueryFilter{{Field: "id", Operator: "eq", Value: queryParameter(t, QueryInt64ParameterSchemaID, "2")}},
	}
	topicResponse := h.engine.execute(ctx, topic)
	if topicResponse.GetError() != nil || len(topicResponse.GetRows()) != 1 || topicResponse.GetRows()[0].GetValue().AsMap()["title"] != "topic two" {
		t.Fatalf("topic by id = %#v", topicResponse)
	}

	attachment := &hostv2.QueryRequest{
		Context: testProtocolV2RequestContext(), QueryId: QueryPublicAttachmentByPublicID, PlanVersion: QueryStableCorePlanVersion,
		Filters: []*hostv2.QueryFilter{{Field: "public_id", Operator: "eq", Value: queryParameter(t, QueryTextParameterSchemaID, "public-1")}},
	}
	attachmentResponse := h.engine.execute(ctx, attachment)
	if attachmentResponse.GetError() != nil || len(attachmentResponse.GetRows()) != 1 {
		t.Fatalf("attachment response = %#v", attachmentResponse)
	}
	attachmentRow := attachmentResponse.GetRows()[0].GetValue().AsMap()
	for _, forbidden := range []string{"provider", "object_key", "storage_key", "quarantine_reason"} {
		if _, exists := attachmentRow[forbidden]; exists {
			t.Fatalf("attachment leaked %s: %#v", forbidden, attachmentRow)
		}
	}
}

func TestPostgresProtocolV2QueryRuntimeRejectsDeniedStaleAndRevokedAuthority(t *testing.T) {
	h := newPostgresQueryHarness(t)
	request := &hostv2.QueryRequest{Context: testProtocolV2RequestContext(), QueryId: QueryPublicTopicsList, PlanVersion: QueryStableCorePlanVersion}

	if _, err := h.pool.Exec(h.ctx, `
		UPDATE extension_versions SET manifest = '{"database":{"authority":"own_schema"}}'::jsonb
		WHERE extension_id = $1 AND version = $2
	`, h.identity.GetExtensionId(), h.identity.GetExtensionVersion()); err != nil {
		t.Fatal(err)
	}
	if _, err := h.pool.Exec(h.ctx, `
		UPDATE extension_trust_grants SET impact_document = '{"database":{"authority":"own_schema"}}'::jsonb
		WHERE id = $1
	`, h.grantID); err != nil {
		t.Fatal(err)
	}
	denied := h.engine.execute(ContextWithProtocolV2RuntimeIdentity(h.ctx, h.identity), request)
	if denied.GetError().GetReason() != "host.query_core_views_denied" {
		t.Fatalf("own-schema authority = %#v", denied)
	}
	if _, err := h.pool.Exec(h.ctx, `
		UPDATE extension_versions SET manifest = '{"database":{"authority":"core_views"}}'::jsonb
		WHERE extension_id = $1 AND version = $2
	`, h.identity.GetExtensionId(), h.identity.GetExtensionVersion()); err != nil {
		t.Fatal(err)
	}
	if _, err := h.pool.Exec(h.ctx, `
		UPDATE extension_trust_grants SET impact_document = '{"database":{"authority":"core_views"}}'::jsonb
		WHERE id = $1
	`, h.grantID); err != nil {
		t.Fatal(err)
	}

	staleDigest := proto.Clone(h.identity).(*protocolv2.ExtensionIdentity)
	staleDigest.ArtifactDigest = strings.Repeat("f", 64)
	stale := h.engine.execute(ContextWithProtocolV2RuntimeIdentity(h.ctx, staleDigest), request)
	if stale.GetError().GetReason() != "host.query_runtime_stale" {
		t.Fatalf("stale digest = %#v", stale)
	}

	stagedDigest := strings.Repeat("b", 64)
	if _, err := h.pool.Exec(h.ctx, `
		INSERT INTO extension_versions (extension_id, version, package_digest, manifest)
		VALUES ($1, '2.0.0', $2, '{"database":{"authority":"core_views"}}'::jsonb)
	`, h.identity.GetExtensionId(), stagedDigest); err != nil {
		t.Fatal(err)
	}
	staged := proto.Clone(h.identity).(*protocolv2.ExtensionIdentity)
	staged.ExtensionVersion = "2.0.0"
	staged.ArtifactDigest = stagedDigest
	stagedResponse := h.engine.execute(ContextWithProtocolV2RuntimeIdentity(h.ctx, staged), request)
	if stagedResponse.GetError().GetReason() != "host.query_runtime_stale" {
		t.Fatalf("inactive staged version = %#v", stagedResponse)
	}

	if _, err := h.pool.Exec(h.ctx, `UPDATE extension_trust_grants SET revoked_at = statement_timestamp() WHERE id = $1`, h.grantID); err != nil {
		t.Fatal(err)
	}
	revoked := h.engine.execute(ContextWithProtocolV2RuntimeIdentity(h.ctx, h.identity), request)
	if revoked.GetError().GetReason() != "host.query_runtime_stale" {
		t.Fatalf("revoked grant = %#v", revoked)
	}
}

func TestPostgresProtocolV2QueryAuthorityAcceptsOnlySystemBuiltinIdentity(t *testing.T) {
	h := newPostgresQueryHarness(t)
	digest := strings.Repeat("c", 64)
	var versionID int64
	if err := h.pool.QueryRow(h.ctx, `
		INSERT INTO extension_versions (extension_id, version, package_digest, manifest)
		VALUES ('fixture.builtin-query', '1.0.0', $1, '{"database":{"authority":"core_views"}}'::jsonb)
		RETURNING id
	`, digest).Scan(&versionID); err != nil {
		t.Fatal(err)
	}
	if _, err := h.pool.Exec(h.ctx, `
		INSERT INTO extensions (id, type, status, source, is_system, active_version_id)
		VALUES ('fixture.builtin-query', 'plugin', 'enabled', 'builtin', true, $1)
	`, versionID); err != nil {
		t.Fatal(err)
	}
	identity := &protocolv2.ExtensionIdentity{
		ExtensionId: "fixture.builtin-query", ExtensionVersion: "1.0.0", ArtifactDigest: digest,
		TrustGrantId: "builtin", RuntimeEpoch: 1, InstanceId: "builtin-instance",
	}
	resolver := NewPostgresProtocolV2QueryAuthorityResolver(h.pool)
	authority, err := resolver.ResolveProtocolV2QueryAuthority(h.ctx, identity)
	if err != nil || !authority.ExactArtifact || !authority.CoreViews {
		t.Fatalf("builtin authority = %#v, %v", authority, err)
	}
	if _, err := h.pool.Exec(h.ctx, `UPDATE extensions SET is_system = false WHERE id = 'fixture.builtin-query'`); err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.ResolveProtocolV2QueryAuthority(h.ctx, identity); err == nil {
		t.Fatal("non-system builtin identity must be stale")
	}
}

func newPostgresQueryHarness(t *testing.T) *postgresQueryHarness {
	t.Helper()
	databaseURL := strings.TrimSpace(os.Getenv("SFORUM_TEST_DATABASE_URL"))
	if databaseURL == "" {
		databaseURL = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	}
	if databaseURL == "" {
		t.Skip("SFORUM_TEST_DATABASE_URL or DATABASE_URL is required for Host Query integration tests")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("host_query_%d", time.Now().UnixNano())
	identifier := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+identifier); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), "DROP SCHEMA IF EXISTS "+identifier+" CASCADE")
		admin.Close()
	})
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	installPostgresQueryFixture(t, ctx, pool)

	const extensionID = "fixture.query"
	const extensionVersion = "1.0.0"
	digest := strings.Repeat("a", 64)
	var versionID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO extension_versions (extension_id, version, package_digest, manifest)
		VALUES ($1, $2, $3, '{"database":{"authority":"core_views"}}'::jsonb) RETURNING id
	`, extensionID, extensionVersion, digest).Scan(&versionID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO extensions (id, type, status, source, is_system, active_version_id)
		VALUES ($1, 'plugin', 'enabled', 'uploaded', false, $2)
	`, extensionID, versionID); err != nil {
		t.Fatal(err)
	}
	var grantID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO extension_trust_grants (extension_id, extension_version, package_digest, action, impact_document)
		VALUES ($1, $2, $3, 'enable', '{"database":{"authority":"core_views"}}'::jsonb) RETURNING id
	`, extensionID, extensionVersion, digest).Scan(&grantID); err != nil {
		t.Fatal(err)
	}
	identity := &protocolv2.ExtensionIdentity{
		ExtensionId: extensionID, ExtensionVersion: extensionVersion, ArtifactDigest: digest,
		TrustGrantId: fmt.Sprintf("%d", grantID), RuntimeEpoch: 1, InstanceId: "query-fixture-instance",
	}
	definitions := stableCoreProtocolV2QueryDefinitions()
	for index := range definitions {
		parts := strings.SplitN(definitions[index].From, ".", 2)
		definitions[index].From = identifier + "." + parts[1]
	}
	resolver := NewPostgresProtocolV2QueryAuthorityResolver(pool)
	engine, err := newProtocolV2QueryEngine(&postgresProtocolV2QueryExecutor{pool: pool}, resolver, definitions...)
	if err != nil {
		t.Fatal(err)
	}
	return &postgresQueryHarness{ctx: ctx, pool: pool, engine: engine, identity: identity, grantID: grantID}
}

func installPostgresQueryFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	statements := []string{
		`CREATE TABLE extension_versions (id BIGSERIAL PRIMARY KEY, extension_id TEXT NOT NULL, version TEXT NOT NULL, package_digest TEXT NOT NULL, manifest JSONB NOT NULL)`,
		`CREATE TABLE extensions (id TEXT PRIMARY KEY, type TEXT NOT NULL, status TEXT NOT NULL, source TEXT NOT NULL, is_system BOOLEAN NOT NULL, active_version_id BIGINT NOT NULL)`,
		`CREATE TABLE extension_trust_grants (id BIGSERIAL PRIMARY KEY, extension_id TEXT NOT NULL, extension_version TEXT NOT NULL, package_digest TEXT NOT NULL, action TEXT NOT NULL, impact_document JSONB NOT NULL, revoked_at TIMESTAMPTZ)`,
		`CREATE TABLE raw_users (id BIGINT, username TEXT, display_name TEXT, email TEXT, password_hash TEXT, locale TEXT, is_admin BOOLEAN, token TEXT, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ)`,
		`CREATE VIEW safe_users AS SELECT id, username, display_name, created_at, updated_at FROM raw_users`,
		`CREATE TABLE forum_topics (id BIGINT, category_id BIGINT, category_slug TEXT, author_user_id BIGINT, title TEXT, slug TEXT, status TEXT, is_pinned BOOLEAN, comment_count BIGINT, view_count BIGINT, last_activity_at TIMESTAMPTZ, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ, html_content TEXT, plain_text TEXT, source_format TEXT, render_version TEXT, content_hash TEXT)`,
		`CREATE TABLE raw_attachments (id BIGINT, public_id TEXT, owner_user_id BIGINT, original_name TEXT, content_type TEXT, extension TEXT, size_bytes BIGINT, sha256 TEXT, image_width INTEGER, image_height INTEGER, reference_count BIGINT, provider TEXT, object_key TEXT, storage_key TEXT, quarantine_reason TEXT, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ)`,
		`CREATE VIEW public_attachment_metadata AS SELECT id, public_id, owner_user_id, original_name, content_type, extension, size_bytes, sha256, image_width, image_height, reference_count, created_at, updated_at FROM raw_attachments`,
		`INSERT INTO raw_users VALUES (1, 'alice', 'Alice', 'private@example.test', 'secret-hash', 'zh-CN', true, 'secret-token', now(), now())`,
		`INSERT INTO forum_topics VALUES (1, 1, 'general', 1, 'topic one', 'one', 'published', false, 0, 1, now() - interval '2 hours', now(), now(), '<p>one</p>', 'one', 'html', '1', 'h1'), (2, 1, 'general', 1, 'topic two', 'two', 'published', false, 0, 2, now() - interval '1 hour', now(), now(), '<p>two</p>', 'two', 'html', '1', 'h2'), (3, 1, 'general', 1, 'topic three', 'three', 'published', false, 0, 3, now(), now(), now(), '<p>three</p>', 'three', 'html', '1', 'h3')`,
		`INSERT INTO raw_attachments VALUES (1, 'public-1', 1, 'image.png', 'image/png', 'png', 42, 'abc', 10, 20, 1, 's3', 'private/object', 'storage-key', 'private-note', now(), now())`,
	}
	for _, statement := range statements {
		if _, err := pool.Exec(ctx, statement); err != nil {
			t.Fatalf("install Host Query fixture: %v\n%s", err, statement)
		}
	}
}
