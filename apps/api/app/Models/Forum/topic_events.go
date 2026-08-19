package forum

import (
	"context"
	"strconv"
)

// TopicEventLinkResolver resolves the operator-configured public URL for a topic.
// Resolution failures must not block the write path; callers retain the relative path.
type TopicEventLinkResolver interface {
	TopicEventURL(ctx context.Context, topic TopicSummary) (string, error)
}

type TopicEventSnapshotStore interface {
	GetTopicEventSnapshot(ctx context.Context, topicID int64) (TopicSummary, error)
}

func resolveTopicEventSnapshot(ctx context.Context, source Store, fallback TopicSummary) TopicSummary {
	store, ok := source.(TopicEventSnapshotStore)
	if !ok || fallback.ID <= 0 {
		return fallback
	}
	snapshot, err := store.GetTopicEventSnapshot(ctx, fallback.ID)
	if err != nil {
		return fallback
	}
	return snapshot
}

func buildTopicEventPayload(ctx context.Context, resolver TopicEventLinkResolver, topic TopicSummary) map[string]any {
	path := topicEventPath(topic)
	publicURL := path
	if resolver != nil {
		if resolved, err := resolver.TopicEventURL(ctx, topic); err == nil && resolved != "" {
			publicURL = resolved
		}
	}

	tags := make([]map[string]any, 0, len(topic.Tags))
	tagSlugs := make([]string, 0, len(topic.Tags))
	for _, tag := range topic.Tags {
		tags = append(tags, map[string]any{
			"id":     tag.ID,
			"slug":   tag.Slug,
			"name":   tag.Name,
			"status": tag.Status,
		})
		tagSlugs = append(tagSlugs, tag.Slug)
	}

	payload := map[string]any{
		// Legacy flat fields remain stable for existing webhook and plugin consumers.
		"topicId":      topic.ID,
		"authorUserId": topic.AuthorUserID,
		"categorySlug": topic.CategorySlug,
		"tagSlugs":     tagSlugs,
		"title":        topic.Title,
		"path":         path,
		"url":          publicURL,
		"topic": map[string]any{
			"id":     topic.ID,
			"title":  topic.Title,
			"slug":   topic.Slug,
			"status": topic.Status,
			"path":   path,
			"url":    publicURL,
		},
		"category": map[string]any{
			"id":   topic.CategoryID,
			"slug": topic.CategorySlug,
			"name": topic.CategoryName,
		},
		"tags": tags,
	}
	if topic.Author != nil {
		payload["author"] = map[string]any{
			"id":          topic.Author.ID,
			"username":    topic.Author.Username,
			"displayName": topic.Author.DisplayName,
		}
	}
	return payload
}

// The id+slug route remains readable after operators switch URL modes and is a
// useful fail-open path when the live site URL setting cannot be resolved.
func topicEventPath(topic TopicSummary) string {
	if topic.Slug == "" {
		return "/t/" + strconv.FormatInt(topic.ID, 10)
	}
	return "/t/" + strconv.FormatInt(topic.ID, 10) + "/" + topic.Slug
}
