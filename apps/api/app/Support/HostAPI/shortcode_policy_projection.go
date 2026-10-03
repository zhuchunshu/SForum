package hostapi

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrShortcodePolicyProjectionUnavailable = errors.New("hostapi: shortcode policy projection unavailable")

// ShortcodePolicyProjection executes the frozen M4 projections for Host-owned
// protected-shortcode policy. It does not expose the Query Registry outlet to
// plugins and never caches actor-sensitive rows.
type ShortcodePolicyProjection struct {
	executor protocolV2QueryExecutor
}

func NewPostgresShortcodePolicyProjection(pool *pgxpool.Pool) *ShortcodePolicyProjection {
	if pool == nil {
		return nil
	}
	return &ShortcodePolicyProjection{executor: &postgresProtocolV2QueryExecutor{pool: pool}}
}

func (p *ShortcodePolicyProjection) AuthorDecision(
	ctx context.Context,
	actorUserID int64,
	resourceType string,
	resourceID int64,
) (authenticated, resourceAuthor, topicAuthor, found bool, err error) {
	if resourceType != "topic" && resourceType != "comment" {
		return false, false, false, false, ErrShortcodePolicyProjectionUnavailable
	}
	rows, err := p.execute(ctx, QueryShortcodeAuthorDecisionsBatch, actorUserID, resourceType, []int64{resourceID})
	if err != nil || len(rows) != 1 {
		return false, false, false, false, err
	}
	authenticated, authOK := rows[0]["authenticated"].(bool)
	resourceAuthor, resourceOK := rows[0]["is_resource_author"].(bool)
	topicAuthor, topicOK := rows[0]["is_topic_author"].(bool)
	if !authOK || !resourceOK || !topicOK {
		return false, false, false, false, ErrShortcodePolicyProjectionUnavailable
	}
	return authenticated, resourceAuthor, topicAuthor, true, nil
}

func (p *ShortcodePolicyProjection) ReplyEligibility(
	ctx context.Context,
	actorUserID int64,
	topicID int64,
) (authenticated, eligible, found bool, err error) {
	rows, err := p.execute(ctx, QueryShortcodeReplyEligibilityBatch, actorUserID, "", []int64{topicID})
	if err != nil || len(rows) != 1 {
		return false, false, false, err
	}
	authenticated, authOK := rows[0]["authenticated"].(bool)
	eligible, eligibleOK := rows[0]["eligible"].(bool)
	if !authOK || !eligibleOK {
		return false, false, false, ErrShortcodePolicyProjectionUnavailable
	}
	return authenticated, eligible, true, nil
}

func (p *ShortcodePolicyProjection) PublicCommentTopicID(ctx context.Context, commentID int64) (int64, bool, error) {
	rows, err := p.execute(ctx, QueryShortcodePublicCommentsBatch, 0, "", []int64{commentID})
	if err != nil || len(rows) != 1 {
		return 0, false, err
	}
	topicID, ok := projectionInt64(rows[0]["topic_id"])
	if !ok || topicID <= 0 {
		return 0, false, ErrShortcodePolicyProjectionUnavailable
	}
	return topicID, true, nil
}

func (p *ShortcodePolicyProjection) execute(
	ctx context.Context,
	queryID string,
	actorUserID int64,
	resourceType string,
	ids []int64,
) ([]map[string]any, error) {
	if p == nil || p.executor == nil || ctx == nil || actorUserID < 0 || len(ids) != 1 || ids[0] <= 0 {
		return nil, ErrShortcodePolicyProjectionUnavailable
	}
	definition, ok := shortcodePolicyProjectionDefinition(queryID)
	if !ok {
		return nil, ErrShortcodePolicyProjectionUnavailable
	}
	filters := make([]protocolV2QueryFilter, 0, len(definition.Filters))
	for _, filter := range definition.Filters {
		switch filter.Field {
		case "resource_type":
			filters = append(filters, protocolV2QueryFilter{Definition: filter, Value: resourceType})
		case "ids":
			filters = append(filters, protocolV2QueryFilter{Definition: filter, Value: ids})
		}
	}
	plan := protocolV2QueryPlan{
		Definition: definition, Fields: definition.Fields, Filters: filters,
		Sorts: definition.DefaultSorts, Limit: ShortcodeProjectionMaxBatch,
		FetchLimit: ShortcodeProjectionMaxBatch + 1,
	}
	if definition.ActorScoped {
		ctx = contextWithShortcodeProjectionActor(ctx, actorUserID)
	}
	return p.executor.ExecuteProtocolV2Query(ctx, plan)
}

func shortcodePolicyProjectionDefinition(queryID string) (protocolV2QueryDefinition, bool) {
	for _, definition := range shortcodeProjectionProtocolV2QueryDefinitions() {
		if definition.ID == queryID {
			return definition, true
		}
	}
	return protocolV2QueryDefinition{}, false
}

func projectionInt64(value any) (int64, bool) {
	switch typed := value.(type) {
	case string:
		var result int64
		for _, character := range typed {
			if character < '0' || character > '9' {
				return 0, false
			}
			result = result*10 + int64(character-'0')
		}
		return result, typed != ""
	case int64:
		return typed, true
	default:
		return 0, false
	}
}
