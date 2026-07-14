package hostapi

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	protocolv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/protocol/v2"
)

type postgresProtocolV2QueryExecutor struct {
	pool *pgxpool.Pool
}

type postgresProtocolV2QueryAuthorityResolver struct {
	pool *pgxpool.Pool
}

func NewPostgresProtocolV2QueryAuthorityResolver(pool *pgxpool.Pool) ProtocolV2QueryAuthorityResolver {
	return &postgresProtocolV2QueryAuthorityResolver{pool: pool}
}

func NewPostgresProtocolV2QueryRuntime(
	pool *pgxpool.Pool,
	authority ProtocolV2QueryAuthorityResolver,
) (ProtocolV2QueryRuntime, error) {
	if pool == nil || authority == nil {
		return nil, errors.New("hostapi: PostgreSQL query pool and authority resolver are required")
	}
	return newProtocolV2QueryRuntime(
		&postgresProtocolV2QueryExecutor{pool: pool},
		authority,
		stableCoreProtocolV2QueryDefinitions()...,
	)
}

func (r *postgresProtocolV2QueryAuthorityResolver) ResolveProtocolV2QueryAuthority(
	ctx context.Context,
	identity *protocolv2.ExtensionIdentity,
) (ProtocolV2QueryAuthority, error) {
	if r == nil || r.pool == nil || !validProtocolV2QueryIdentity(identity) {
		return ProtocolV2QueryAuthority{}, ErrProtocolV2QueryRuntimeStale
	}
	var source, authority string
	var system bool
	err := r.pool.QueryRow(ctx, `
		SELECT extensions.source, extensions.is_system,
		       COALESCE(extension_versions.manifest #>> '{database,authority}', '')
		FROM extensions
		JOIN extension_versions ON extension_versions.id = extensions.active_version_id
		WHERE extensions.id = $1 AND extensions.type = 'plugin' AND extensions.status = 'enabled'
		  AND extension_versions.version = $2 AND extension_versions.package_digest = $3
	`, identity.GetExtensionId(), identity.GetExtensionVersion(), identity.GetArtifactDigest()).Scan(
		&source, &system, &authority,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProtocolV2QueryAuthority{}, ErrProtocolV2QueryRuntimeStale
	}
	if err != nil {
		return ProtocolV2QueryAuthority{}, fmt.Errorf("resolve stable query artifact: %w", err)
	}
	if identity.GetTrustGrantId() == "builtin" {
		if source != "builtin" || !system {
			return ProtocolV2QueryAuthority{}, ErrProtocolV2QueryRuntimeStale
		}
		return ProtocolV2QueryAuthority{ExactArtifact: true, CoreViews: authority == "core_views"}, nil
	}
	if source != "uploaded" || system {
		return ProtocolV2QueryAuthority{}, ErrProtocolV2QueryRuntimeStale
	}
	grantID, err := strconv.ParseInt(identity.GetTrustGrantId(), 10, 64)
	if err != nil || grantID <= 0 || strconv.FormatInt(grantID, 10) != identity.GetTrustGrantId() {
		return ProtocolV2QueryAuthority{}, ErrProtocolV2QueryRuntimeStale
	}
	var trusted bool
	err = r.pool.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1
		  FROM extension_trust_grants
		  WHERE id = $1 AND extension_id = $2 AND extension_version = $3
		    AND package_digest = $4 AND action = 'enable' AND revoked_at IS NULL
		    AND COALESCE(impact_document #>> '{database,authority}', '') = $5
		)
	`, grantID, identity.GetExtensionId(), identity.GetExtensionVersion(),
		identity.GetArtifactDigest(), authority).Scan(&trusted)
	if err != nil {
		return ProtocolV2QueryAuthority{}, fmt.Errorf("resolve stable query trust grant: %w", err)
	}
	if !trusted {
		return ProtocolV2QueryAuthority{}, ErrProtocolV2QueryRuntimeStale
	}
	return ProtocolV2QueryAuthority{ExactArtifact: true, CoreViews: authority == "core_views"}, nil
}

func validProtocolV2QueryIdentity(identity *protocolv2.ExtensionIdentity) bool {
	if identity == nil || strings.TrimSpace(identity.GetExtensionId()) == "" ||
		strings.TrimSpace(identity.GetExtensionVersion()) == "" ||
		len(identity.GetArtifactDigest()) != 64 || strings.Trim(identity.GetArtifactDigest(), "0123456789abcdef") != "" ||
		strings.TrimSpace(identity.GetTrustGrantId()) == "" || identity.GetRuntimeEpoch() == 0 ||
		strings.TrimSpace(identity.GetInstanceId()) == "" {
		return false
	}
	return true
}

func (e *postgresProtocolV2QueryExecutor) ExecuteProtocolV2Query(
	ctx context.Context,
	plan protocolV2QueryPlan,
) ([]map[string]any, error) {
	if e == nil || e.pool == nil {
		return nil, errors.New("Host Query PostgreSQL executor is unavailable")
	}
	tx, err := e.pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	query, args := postgresProtocolV2Query(plan)
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]any, 0, plan.FetchLimit)
	for rows.Next() {
		values, valuesErr := rows.Values()
		if valuesErr != nil {
			rows.Close()
			return nil, valuesErr
		}
		if len(values) != len(plan.Fields) {
			rows.Close()
			return nil, errors.New("Host Query returned an unexpected column count")
		}
		item := make(map[string]any, len(plan.Fields))
		for index, field := range plan.Fields {
			value, normalizeErr := normalizeProtocolV2QueryValue(values[index])
			if normalizeErr != nil {
				rows.Close()
				return nil, normalizeErr
			}
			item[field.Name] = value
		}
		result = append(result, item)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return result, nil
}

func postgresProtocolV2Query(plan protocolV2QueryPlan) (string, []any) {
	selected := make([]string, 0, len(plan.Fields))
	for _, field := range plan.Fields {
		selected = append(selected, field.Expression+" AS "+pgx.Identifier{field.Name}.Sanitize())
	}
	var builder strings.Builder
	builder.WriteString("SELECT ")
	builder.WriteString(strings.Join(selected, ", "))
	builder.WriteString(" FROM ")
	builder.WriteString(plan.Definition.From)
	args := make([]any, 0, len(plan.Filters)+2)
	if len(plan.Filters) > 0 {
		builder.WriteString(" WHERE ")
		for index, filter := range plan.Filters {
			if index > 0 {
				builder.WriteString(" AND ")
			}
			args = append(args, filter.Value)
			builder.WriteString(filter.Definition.Expression)
			builder.WriteString(" = $")
			builder.WriteString(strconv.Itoa(len(args)))
		}
	}
	if len(plan.Sorts) > 0 {
		builder.WriteString(" ORDER BY ")
		for index, sort := range plan.Sorts {
			if index > 0 {
				builder.WriteString(", ")
			}
			builder.WriteString(sort.Expression)
			if sort.Descending {
				builder.WriteString(" DESC")
			} else {
				builder.WriteString(" ASC")
			}
		}
	}
	args = append(args, plan.FetchLimit)
	builder.WriteString(" LIMIT $")
	builder.WriteString(strconv.Itoa(len(args)))
	args = append(args, plan.Offset)
	builder.WriteString(" OFFSET $")
	builder.WriteString(strconv.Itoa(len(args)))
	return builder.String(), args
}

func normalizeProtocolV2QueryValue(value any) (any, error) {
	switch typed := value.(type) {
	case nil, string, bool, float64:
		return typed, nil
	case int64:
		return strconv.FormatInt(typed, 10), nil
	case int32:
		return float64(typed), nil
	case int16:
		return float64(typed), nil
	case time.Time:
		return typed.UTC().Format(time.RFC3339Nano), nil
	case []byte:
		return string(typed), nil
	default:
		return nil, fmt.Errorf("Host Query result type %T is not supported", value)
	}
}

func stableCoreProtocolV2QueryDefinitions() []protocolV2QueryDefinition {
	safeUserFields := []protocolV2QueryField{
		{Name: "id", Expression: "stable.id"},
		{Name: "username", Expression: "stable.username"},
		{Name: "display_name", Expression: "stable.display_name"},
		{Name: "created_at", Expression: "stable.created_at"},
		{Name: "updated_at", Expression: "stable.updated_at"},
	}
	topicFields := []protocolV2QueryField{
		{Name: "id", Expression: "stable.id"},
		{Name: "category_id", Expression: "stable.category_id"},
		{Name: "category_slug", Expression: "stable.category_slug"},
		{Name: "author_user_id", Expression: "stable.author_user_id"},
		{Name: "title", Expression: "stable.title"},
		{Name: "slug", Expression: "stable.slug"},
		{Name: "status", Expression: "stable.status"},
		{Name: "is_pinned", Expression: "stable.is_pinned"},
		{Name: "comment_count", Expression: "stable.comment_count"},
		{Name: "view_count", Expression: "stable.view_count"},
		{Name: "last_activity_at", Expression: "stable.last_activity_at"},
		{Name: "created_at", Expression: "stable.created_at"},
		{Name: "updated_at", Expression: "stable.updated_at"},
		{Name: "html_content", Expression: "stable.html_content"},
		{Name: "plain_text", Expression: "stable.plain_text"},
		{Name: "source_format", Expression: "stable.source_format"},
		{Name: "render_version", Expression: "stable.render_version"},
		{Name: "content_hash", Expression: "stable.content_hash"},
	}
	attachmentFields := []protocolV2QueryField{
		{Name: "id", Expression: "stable.id"},
		{Name: "public_id", Expression: "stable.public_id"},
		{Name: "owner_user_id", Expression: "stable.owner_user_id"},
		{Name: "original_name", Expression: "stable.original_name"},
		{Name: "content_type", Expression: "stable.content_type"},
		{Name: "extension", Expression: "stable.extension"},
		{Name: "size_bytes", Expression: "stable.size_bytes"},
		{Name: "sha256", Expression: "stable.sha256"},
		{Name: "image_width", Expression: "stable.image_width"},
		{Name: "image_height", Expression: "stable.image_height"},
		{Name: "reference_count", Expression: "stable.reference_count"},
		{Name: "created_at", Expression: "stable.created_at"},
		{Name: "updated_at", Expression: "stable.updated_at"},
	}
	idFilter := protocolV2QueryFilterDefinition{
		Field: "id", Operator: "eq", Expression: "stable.id",
		SchemaID: QueryInt64ParameterSchemaID, Kind: "int64",
	}
	topicSorts := []protocolV2QuerySortDefinition{
		{Field: "id", Expression: "stable.id"},
		{Field: "created_at", Expression: "stable.created_at"},
		{Field: "updated_at", Expression: "stable.updated_at"},
		{Field: "last_activity_at", Expression: "stable.last_activity_at"},
	}
	return []protocolV2QueryDefinition{
		{
			ID: QuerySafeUserByID, PlanVersion: QueryStableCorePlanVersion,
			ResultSchemaID: QuerySafeUserResultSchemaID, ResultSchemaVersion: QueryStableCoreResultSchemaV1,
			From: "sforum_core_v1.safe_users AS stable", Fields: safeUserFields,
			Filters: []protocolV2QueryFilterDefinition{idFilter}, RequiredFilters: []string{"id"}, Single: true,
		},
		{
			ID: QueryPublicTopicsList, PlanVersion: QueryStableCorePlanVersion,
			ResultSchemaID: QueryPublicTopicResultSchemaID, ResultSchemaVersion: QueryStableCoreResultSchemaV1,
			From: "sforum_core_v1.forum_topics AS stable", Fields: topicFields,
			Filters: []protocolV2QueryFilterDefinition{
				{Field: "category_id", Operator: "eq", Expression: "stable.category_id", SchemaID: QueryInt64ParameterSchemaID, Kind: "int64"},
				{Field: "author_user_id", Operator: "eq", Expression: "stable.author_user_id", SchemaID: QueryInt64ParameterSchemaID, Kind: "int64"},
			},
			Sorts: topicSorts,
			DefaultSorts: []protocolV2QuerySort{
				{Field: "last_activity_at", Expression: "stable.last_activity_at", Descending: true},
				{Field: "id", Expression: "stable.id", Descending: true},
			},
		},
		{
			ID: QueryPublicTopicByID, PlanVersion: QueryStableCorePlanVersion,
			ResultSchemaID: QueryPublicTopicResultSchemaID, ResultSchemaVersion: QueryStableCoreResultSchemaV1,
			From: "sforum_core_v1.forum_topics AS stable", Fields: topicFields,
			Filters: []protocolV2QueryFilterDefinition{idFilter}, RequiredFilters: []string{"id"}, Single: true,
		},
		{
			ID: QueryPublicAttachmentByPublicID, PlanVersion: QueryStableCorePlanVersion,
			ResultSchemaID: QueryPublicAttachmentSchemaID, ResultSchemaVersion: QueryStableCoreResultSchemaV1,
			From: "sforum_core_v1.public_attachment_metadata AS stable", Fields: attachmentFields,
			Filters: []protocolV2QueryFilterDefinition{{
				Field: "public_id", Operator: "eq", Expression: "stable.public_id",
				SchemaID: QueryTextParameterSchemaID, Kind: "text",
			}},
			RequiredFilters: []string{"public_id"}, Single: true,
		},
	}
}
