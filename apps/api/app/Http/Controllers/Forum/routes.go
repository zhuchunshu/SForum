package forumcontroller

import (
	"github.com/gofiber/fiber/v3"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Idempotency"
)

func (h *Controller) RegisterRoutes(api fiber.Router) {
	// 发帖/评论写路径可选携带 Idempotency-Key（F3.2）；未携带时行为不变。
	idem := h.idempotencyHandler()

	api.Get("/category-groups", h.categoryGroups)
	api.Get("/categories", h.categories)
	api.Get("/tags", h.tags)
	// F4.3：composer 工具栏扩展动作（登录后使用；guest 可读但通常无意义）。
	api.Get("/composer/toolbar", h.composerToolbar)
	api.Get("/search", h.search)
	api.Get("/me/content-review", h.authorReviewItems)
	api.Get("/topics", h.topics)
	if idem != nil {
		api.Post("/topics", idem, h.createTopic)
	} else {
		api.Post("/topics", h.createTopic)
	}
	// 注意：by-slug 必须先于 :topicID 注册，否则 "by-slug" 会被当作 topicID 捕获。
	api.Get("/topics/by-slug/:slug", h.topicBySlug)
	api.Get("/topics/:topicID/revisions", h.topicRevisions)
	api.Get("/topics/:topicID/revisions/:revisionNo", h.topicRevision)
	api.Post("/topics/:topicID/revisions/:revisionNo/restore", h.restoreTopicRevision)
	api.Post("/topics/:topicID/revisions/:revisionNo/redact", h.redactTopicRevision)
	api.Get("/topics/:topicID", h.topic)
	api.Patch("/topics/:topicID", h.updateTopic)
	api.Delete("/topics/:topicID", h.deleteTopic)
	api.Post("/topics/:topicID/hide", h.hideTopic)
	api.Post("/topics/:topicID/restore", h.restoreTopic)
	api.Post("/topics/:topicID/lock", h.lockTopic)
	api.Post("/topics/:topicID/unlock", h.unlockTopic)
	api.Post("/topics/:topicID/pin", h.pinTopic)
	api.Post("/topics/:topicID/unpin", h.unpinTopic)
	api.Get("/topics/:topicID/comments", h.comments)
	// 反查评论所在分页页码：供帖子详情页 #comment-{id} 锚点跨页定位（flat 视图）。
	api.Get("/topics/:topicID/comments/:commentID/page", h.commentPage)
	if idem != nil {
		api.Post("/topics/:topicID/comments", idem, h.createComment)
	} else {
		api.Post("/topics/:topicID/comments", h.createComment)
	}
	api.Get("/comments/:commentID/replies", h.replies)
	api.Get("/comments/:commentID/revisions", h.commentRevisions)
	api.Get("/comments/:commentID/revisions/:revisionNo", h.commentRevision)
	api.Post("/comments/:commentID/revisions/:revisionNo/restore", h.restoreCommentRevision)
	api.Post("/comments/:commentID/revisions/:revisionNo/redact", h.redactCommentRevision)
	api.Patch("/comments/:commentID", h.updateComment)
	api.Delete("/comments/:commentID", h.deleteComment)

	admin := api.Group("/admin/forum")
	admin.Get("/category-groups", h.adminCategoryGroups)
	admin.Post("/category-groups", h.adminCreateCategoryGroup)
	admin.Patch("/category-groups/:groupID", h.adminUpdateCategoryGroup)
	admin.Get("/categories", h.adminCategories)
	admin.Post("/categories", h.adminCreateCategory)
	admin.Patch("/categories/:categoryID", h.adminUpdateCategory)
	admin.Get("/tags", h.adminTags)
	admin.Post("/tags", h.adminCreateTag)
	admin.Patch("/tags/:tagID", h.adminUpdateTag)
	admin.Get("/settings", h.adminSettings)
	admin.Put("/settings", h.adminUpdateSettings)
	admin.Post("/settings/reset", h.adminResetSettings)
	admin.Get("/content/topics", h.adminContentTopics)
	admin.Get("/content/topics/:topicID", h.adminContentTopic)
	admin.Get("/content/comments", h.adminContentComments)
	admin.Get("/content/comments/:commentID", h.adminContentComment)
	admin.Get("/search/providers", h.adminListSearchProviders)
	admin.Put("/search/provider", h.adminSelectSearchProvider)
	admin.Post("/search/provider/reset", h.adminResetSearchProvider)
	admin.Post("/search/reindex", h.adminReindexSearch)
	admin.Get("/search/reindex", h.adminReindexStatus)
	admin.Get("/search/reindex/runs", h.adminReindexRuns)
}

func (h *Controller) idempotencyHandler() fiber.Handler {
	if h == nil || h.idempotency == nil || h.sessions == nil {
		return nil
	}
	return idempotency.Middleware(h.idempotency, func(c fiber.Ctx) (int64, error) {
		userID, ok, err := apphttp.ResolveUserID(c, h.sessions)
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, nil
		}
		return userID, nil
	})
}
