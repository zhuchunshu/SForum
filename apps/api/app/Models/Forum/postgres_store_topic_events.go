package forum

import (
	"context"
	"fmt"
)

// GetTopicEventSnapshot loads a public-safe topic snapshot after a successful
// mutation. Callers treat failures as fail-open so webhook enrichment never
// changes the mutation result.
func (s *PostgresStore) GetTopicEventSnapshot(ctx context.Context, topicID int64) (TopicSummary, error) {
	topic, err := s.GetTopicForAction(ctx, topicID)
	if err != nil {
		return TopicSummary{}, err
	}
	items := []TopicSummary{topic}
	if err := s.attachActiveTagsToTopicSummaries(ctx, items); err != nil {
		return TopicSummary{}, fmt.Errorf("get topic tags for event: %w", err)
	}
	return items[0], nil
}
