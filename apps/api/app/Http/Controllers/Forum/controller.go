package forumcontroller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	forum "github.com/zhuchunshu/sforum/apps/api/app/Models/Forum"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	clientip "github.com/zhuchunshu/sforum/apps/api/app/Support/ClientIP"
	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Idempotency"
)

type Controller struct {
	service         *forum.Service
	searchService   SearchService
	reindexer       ReindexService
	searchProviders SearchProviderAdmin
	users           identity.ActorStore
	sessions        *authsession.Manager
	// idempotency 可选：注入后对发帖/评论写路径启用 Idempotency-Key（F3.2）。
	idempotency *idempotency.Store
}

// SearchService 抽象搜索查询，避免 controller 直接依赖 search 包。
// search.Service 经适配器实现此接口；nil 时搜索端点返回 503。
type SearchService interface {
	Search(ctx context.Context, input SearchInput) (SearchOutput, error)
}

// ReindexService 抽象搜索索引重建，由 search.ReindexManager 经适配器实现。
// nil 时重建端点返回 503。
type ReindexService interface {
	Reindex(ctx context.Context, startedByUserID int64) (ReindexRunOutput, error)
	ReindexStatus(ctx context.Context) (ReindexStatusOutput, error)
	ListReindexRuns(ctx context.Context) ([]ReindexRunOutput, error)
}

// ReindexRunOutput 是重建 run 的 controller 侧镜像类型，解耦 search 包。
type ReindexRunOutput struct {
	ID              int64      `json:"id"`
	Total           int64      `json:"total"`
	Status          string     `json:"status"`
	StartedAt       time.Time  `json:"startedAt"`
	FinishedAt      *time.Time `json:"finishedAt,omitempty"`
	StartedByUserID int64      `json:"startedByUserId,omitempty"`
	Error           string     `json:"error,omitempty"`
}

type ReindexStatusOutput struct {
	ReindexRunOutput
	Processed int64 `json:"processed"`
	Remaining int64 `json:"remaining"`
	Percent   int   `json:"percent"`
}

// SearchInput/SearchOutput 是 controller 与 search 实现之间的解耦结构。
type SearchInput struct {
	Query        string
	CategorySlug string
	TagSlug      string
	Page         int
	PerPage      int
}

// SearchOutput 与首页 TopicList 行同构：items 为 TopicSummary，前端复用列表组件。
type SearchOutput struct {
	Items   []forum.TopicSummary `json:"items"`
	Total   int64                `json:"total"`
	Page    int                  `json:"page"`
	PerPage int                  `json:"perPage"`
}

func NewController(service *forum.Service, users identity.ActorStore, sessions *authsession.Manager) *Controller {
	return &Controller{service: service, users: users, sessions: sessions}
}

// NewControllerWithSearch 注入搜索服务与索引重建服务。
func NewControllerWithSearch(service *forum.Service, searchSvc SearchService, reindexer ReindexService, users identity.ActorStore, sessions *authsession.Manager) *Controller {
	return &Controller{service: service, searchService: searchSvc, reindexer: reindexer, users: users, sessions: sessions}
}

// SearchProviderAdmin 抽象 search.provider 运营选择，避免 controller 依赖扩展存储细节。
// nil 时提供商端点返回 503。
type SearchProviderAdmin interface {
	List(ctx context.Context) (SearchProvidersState, error)
	Select(ctx context.Context, extensionID string) error
	RestoreDefault(ctx context.Context) error
}

// SearchProvidersState 是运营侧搜索提供商列表与当前解析结果。
type SearchProvidersState struct {
	Items                []SearchProviderItem `json:"items"`
	Selected             SearchProviderItem   `json:"selected"`
	Pinned               bool                 `json:"pinned"`
	DefaultExtensionID   string               `json:"defaultExtensionId"`
}

// SearchProviderItem 单个 search.provider 候选。
type SearchProviderItem struct {
	ExtensionID string `json:"extensionId"`
	Label       string `json:"label"`
	Healthy     bool   `json:"healthy"`
	IsDefault   bool   `json:"isDefault,omitempty"`
}

// WithSearchProviderAdmin 注入搜索提供商运营选择。
func (h *Controller) WithSearchProviderAdmin(admin SearchProviderAdmin) *Controller {
	if h != nil {
		h.searchProviders = admin
	}
	return h
}

// WithViewRecorder 注入公开详情浏览计数（D3）。
func (h *Controller) WithViewRecorder(recorder forum.TopicViewRecorder) *Controller {
	if h != nil && h.service != nil {
		h.service.WithViewRecorder(recorder)
	}
	return h
}

// WithIdempotency 启用选定写路由的 Idempotency-Key 去重。
// WithContentPostFilter wires the optional ContentRegistry post-render seam.
func (h *Controller) WithContentPostFilter(filter forum.ContentPostFilter) *Controller {
	if h != nil && h.service != nil {
		h.service.WithContentPostFilter(filter)
	}
	return h
}

// WithEditorDocumentSchema wires Editor Registry schema into Accept on write paths.
func (h *Controller) WithEditorDocumentSchema(provider forum.EditorDocumentSchemaProvider) *Controller {
	if h != nil && h.service != nil {
		h.service.WithEditorDocumentSchema(provider)
	}
	return h
}

func (h *Controller) WithIdempotency(store *idempotency.Store) *Controller {
	if h != nil {
		h.idempotency = store
	}
	return h
}

type createTopicRequest struct {
	CategorySlug string             `json:"categorySlug"`
	Title        string             `json:"title"`
	TagSlugs     []string           `json:"tagSlugs"`
	Content      forum.ContentInput `json:"content"`
}

// updateTopicRequest: 所有字段均可选，nil 表示不改。categorySlug/tagSlugs 为空切片表示清空标签。
type updateTopicRequest struct {
	CategorySlug *string             `json:"categorySlug"`
	Title        *string             `json:"title"`
	TagSlugs     []string            `json:"tagSlugs"`
	Content      *forum.ContentInput `json:"content"`
}

type createCommentRequest struct {
	ParentID *int64             `json:"parentId"`
	Content  forum.ContentInput `json:"content"`
}

type updateCommentRequest struct {
	Content forum.ContentInput `json:"content"`
}

func (h *Controller) categories(c fiber.Ctx) error {
	if err := h.requireGuestRead(c); err != nil {
		return err
	}
	items, err := h.service.ListCategories(c.Context())
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.OK(c, items)
}

func (h *Controller) categoryGroups(c fiber.Ctx) error {
	if err := h.requireGuestRead(c); err != nil {
		return err
	}
	items, err := h.service.ListCategoryGroups(c.Context())
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.OK(c, items)
}

func (h *Controller) composerToolbar(c fiber.Ctx) error {
	// 与发帖一致：需登录才能使用扩展工具栏动作。
	if _, err := h.actor(c); err != nil {
		return err
	}
	items, err := h.service.ListComposerToolbarActions(c.Context())
	if err != nil {
		return mapForumError(err)
	}
	if items == nil {
		items = []forum.ComposerToolbarAction{}
	}
	return apphttp.OK(c, items)
}

func (h *Controller) tags(c fiber.Ctx) error {
	if err := h.requireGuestRead(c); err != nil {
		return err
	}
	items, err := h.service.ListTags(c.Context(), false)
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.OK(c, items)
}

func (h *Controller) topics(c fiber.Ctx) error {
	if err := h.requireGuestRead(c); err != nil {
		return err
	}
	list, err := h.service.ListTopics(c.Context(), forum.TopicListInput{
		Page:         queryInt(c, "page"),
		PerPage:      queryInt(c, "perPage"),
		// M5：after 优先于 page（service/store 忽略 page when after set）
		After:        c.Query("after"),
		CategorySlug: c.Query("categorySlug"),
		TagSlug:      c.Query("tagSlug"),
		Query:        c.Query("query"),
		Sort:         c.Query("sort"),
	})
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.OK(c, list)
}

// search 提供主题全文检索（search.provider；默认站内 PostgreSQL 引擎）。
// 关键词检索已从 topics 列表迁移到此专用端点，避免 ILIKE 全表扫描。
func (h *Controller) search(c fiber.Ctx) error {
	if err := h.requireGuestRead(c); err != nil {
		return err
	}
	if h.searchService == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "forum.search_unavailable")
	}
	result, err := h.searchService.Search(c.Context(), SearchInput{
		Query:        c.Query("query"),
		CategorySlug: c.Query("categorySlug"),
		TagSlug:      c.Query("tagSlug"),
		Page:         queryInt(c, "page"),
		PerPage:      queryInt(c, "perPage"),
	})
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.OK(c, result)
}

func (h *Controller) createTopic(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	var req createTopicRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeInvalidTopic)
	}
	topic, err := h.service.CreateTopic(c.Context(), actor, forum.CreateTopicInput{
		CategorySlug: req.CategorySlug,
		Title:        req.Title,
		TagSlugs:     req.TagSlugs,
		Content:      req.Content,
		IPAddress:    clientip.FromCtx(c),
	})
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.Created(c, topic)
}

func (h *Controller) authorReviewItems(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	items, err := h.service.ListAuthorReviewItems(c.Context(), actor)
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.OK(c, items)
}

func (h *Controller) topic(c fiber.Ctx) error {
	if err := h.requireGuestRead(c); err != nil {
		return err
	}
	topic, err := h.service.GetTopic(c.Context(), int64(paramInt(c, "topicID")))
	if err != nil {
		return mapForumError(err)
	}
	// D3：公开详情 GET 成功后计浏览（Redis 去重+增量）；不阻断响应。
	h.service.RecordTopicView(c.Context(), topic.ID, h.topicVisitorKey(c))
	return apphttp.OK(c, topic)
}

// topicBySlug 处理 "纯 slug" URL 模式下的公开主题查询。
// 路由参数 :slug 由前端 forumTopicPath 在 slug 模式下产出。
func (h *Controller) topicBySlug(c fiber.Ctx) error {
	if err := h.requireGuestRead(c); err != nil {
		return err
	}
	topic, err := h.service.GetTopicBySlug(c.Context(), c.Params("slug"))
	if err != nil {
		return mapForumError(err)
	}
	h.service.RecordTopicView(c.Context(), topic.ID, h.topicVisitorKey(c))
	return apphttp.OK(c, topic)
}

// topicVisitorKey：登录用户 → u:{id}；否则会话 sid → s:{sid}；再否则 IP+UA 哈希。
// 仅用于 30m 去重，不信任客户端自报 visitor id。
func (h *Controller) topicVisitorKey(c fiber.Ctx) string {
	if h.sessions != nil {
		if userID, ok, err := h.sessions.CurrentUserIDWithoutRenewal(c); err == nil && ok && userID > 0 {
			return fmt.Sprintf("u:%d", userID)
		}
		if sid, err := h.sessions.CurrentSID(c); err == nil && strings.TrimSpace(sid) != "" {
			return "s:" + strings.TrimSpace(sid)
		}
	}
	ip := strings.TrimSpace(clientip.FromCtx(c))
	ua := strings.TrimSpace(string(c.Request().Header.UserAgent()))
	sum := sha256.Sum256([]byte(ip + "\n" + ua))
	return "a:" + hex.EncodeToString(sum[:16])
}

func (h *Controller) updateTopic(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	var req updateTopicRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeInvalidTopic)
	}
	// 区分 tagSlugs 字段"缺失"与"显式空数组"：仅当请求体里出现了 tagSlugs 才替换标签。
	hasTagSlugs, err := bodyHasKey(c, "tagSlugs")
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeInvalidTopic)
	}
	input := forum.UpdateTopicInput{
		TopicID:      int64(paramInt(c, "topicID")),
		CategorySlug: req.CategorySlug,
		Title:        req.Title,
		Content:      req.Content,
		IPAddress:    clientip.FromCtx(c),
	}
	if hasTagSlugs {
		input.TagSlugs = req.TagSlugs
	}
	topic, err := h.service.UpdateTopic(c.Context(), actor, input)
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.OK(c, topic)
}

// bodyHasKey 判断 JSON 请求体是否包含指定顶层字段，用于区分"未提供"与"显式 null/空"。
func bodyHasKey(c fiber.Ctx, key string) (bool, error) {
	body := c.Body()
	if len(body) == 0 {
		return false, nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return false, err
	}
	_, ok := raw[key]
	return ok, nil
}

func (h *Controller) deleteTopic(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	topic, err := h.service.DeleteTopic(c.Context(), actor, int64(paramInt(c, "topicID")))
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.OK(c, topic)
}

// topicAction 是统一的主题生命周期处理入口，对应 hide/restore/lock/unlock/pin/unpin。
func (h *Controller) topicAction(c fiber.Ctx, action string) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	result, err := h.service.ApplyTopicAction(c.Context(), actor, forum.TopicLifecycleInput{
		TopicID: int64(paramInt(c, "topicID")),
		Action:  action,
	})
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.OK(c, result)
}

func (h *Controller) hideTopic(c fiber.Ctx) error {
	return h.topicAction(c, forum.TopicActionHide)
}

func (h *Controller) restoreTopic(c fiber.Ctx) error {
	return h.topicAction(c, forum.TopicActionRestore)
}

func (h *Controller) lockTopic(c fiber.Ctx) error {
	return h.topicAction(c, forum.TopicActionLock)
}

func (h *Controller) unlockTopic(c fiber.Ctx) error {
	return h.topicAction(c, forum.TopicActionUnlock)
}

func (h *Controller) pinTopic(c fiber.Ctx) error {
	return h.topicAction(c, forum.TopicActionPin)
}

func (h *Controller) unpinTopic(c fiber.Ctx) error {
	return h.topicAction(c, forum.TopicActionUnpin)
}

func (h *Controller) comments(c fiber.Ctx) error {
	if err := h.requireGuestRead(c); err != nil {
		return err
	}
	// 可选 viewer：登录用户用于 softDeleteVisibility 墓碑判定；匿名为零值。
	viewer, _ := apphttp.LoadActor(c, h.sessions, h.users)
	list, err := h.service.ListComments(c.Context(), forum.CommentListInput{
		TopicID: int64(paramInt(c, "topicID")),
		View:    c.Query("view", "tree"),
		Page:    queryInt(c, "page"),
		PerPage: queryInt(c, "perPage"),
		// M5：flat keyset；非空 after 优先于 page
		After:  c.Query("after"),
		Viewer: viewer,
	})
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.OK(c, list)
}

func (h *Controller) createComment(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	var req createCommentRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeInvalidContent)
	}
	comment, err := h.service.CreateComment(c.Context(), actor, forum.CreateCommentInput{
		TopicID:   int64(paramInt(c, "topicID")),
		ParentID:  req.ParentID,
		Content:   req.Content,
		IPAddress: clientip.FromCtx(c),
	})
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.Created(c, comment)
}

func (h *Controller) replies(c fiber.Ctx) error {
	if err := h.requireGuestRead(c); err != nil {
		return err
	}
	viewer, _ := apphttp.LoadActor(c, h.sessions, h.users)
	items, err := h.service.ListCommentRepliesForViewer(c.Context(), int64(paramInt(c, "commentID")), viewer)
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.OK(c, items)
}

func (h *Controller) updateComment(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	var req updateCommentRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeInvalidContent)
	}
	comment, err := h.service.UpdateComment(c.Context(), actor, forum.UpdateCommentInput{
		CommentID: int64(paramInt(c, "commentID")),
		Content:   req.Content,
		IPAddress: clientip.FromCtx(c),
	})
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.OK(c, comment)
}

func (h *Controller) deleteComment(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	comment, err := h.service.DeleteComment(c.Context(), actor, int64(paramInt(c, "commentID")))
	if err != nil {
		return mapForumError(err)
	}
	return apphttp.OK(c, comment)
}

func (h *Controller) actor(c fiber.Ctx) (identity.Actor, error) {
	return apphttp.LoadActor(c, h.sessions, h.users)
}

// requireGuestRead：forum.guest.read=login_required 时，匿名读请求返回 401。
// 已登录用户（含任意角色）始终可走公开阅读接口。
func (h *Controller) requireGuestRead(c fiber.Ctx) error {
	settings, err := h.service.PublicForumSettings(c.Context())
	if err != nil {
		return err
	}
	if settings.GuestRead != "login_required" {
		return nil
	}
	_, ok, err := apphttp.ResolveUserID(c, h.sessions)
	if err != nil {
		return err
	}
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, forum.CodeGuestLoginRequired)
	}
	return nil
}

func mapForumError(err error) error {
	var rejected *appevents.RejectedError
	switch {
	case errors.As(err, &rejected):
		return fiber.NewError(fiber.StatusUnprocessableEntity, rejected.Reason)
	case errors.Is(err, ErrSearchUnavailable):
		return fiber.NewError(fiber.StatusServiceUnavailable, "forum.search_unavailable")
	case errors.Is(err, identity.ErrPermissionDenied):
		return fiber.NewError(fiber.StatusForbidden, "permission.denied")
	case errors.Is(err, forum.ErrInvalidContent):
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeInvalidContent)
	case errors.Is(err, forum.ErrInvalidTopic):
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeInvalidTopic)
	case errors.Is(err, forum.ErrTopicNotFound):
		return fiber.NewError(fiber.StatusNotFound, forum.CodeTopicNotFound)
	case errors.Is(err, forum.ErrCommentNotFound):
		return fiber.NewError(fiber.StatusNotFound, forum.CodeCommentNotFound)
	case errors.Is(err, forum.ErrTopicClosed):
		return fiber.NewError(fiber.StatusConflict, forum.CodeTopicClosed)
	case errors.Is(err, forum.ErrInvalidTag):
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeInvalidTag)
	case errors.Is(err, forum.ErrTagNotFound):
		return fiber.NewError(fiber.StatusNotFound, forum.CodeTagNotFound)
	case errors.Is(err, forum.ErrInvalidSettings):
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeInvalidSettings)
	case errors.Is(err, forum.ErrInvalidAction):
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeInvalidAction)
	case errors.Is(err, forum.ErrTitleTooShort):
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeTitleTooShort)
	case errors.Is(err, forum.ErrTitleTooLong):
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeTitleTooLong)
	case errors.Is(err, forum.ErrContentTooShort):
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeContentTooShort)
	case errors.Is(err, forum.ErrContentTooLong):
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeContentTooLong)
	case errors.Is(err, forum.ErrCommentTooShort):
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeCommentTooShort)
	case errors.Is(err, forum.ErrCommentTooLong):
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeCommentTooLong)
	case errors.Is(err, forum.ErrCommentNestingDeep):
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeCommentNestingDeep)
	case errors.Is(err, forum.ErrEditWindowExpired):
		return fiber.NewError(fiber.StatusConflict, forum.CodeEditWindowExpired)
	case errors.Is(err, forum.ErrTopicCooldown):
		return fiber.NewError(fiber.StatusTooManyRequests, forum.CodeTopicCooldown)
	case errors.Is(err, forum.ErrCommentCooldown):
		return fiber.NewError(fiber.StatusTooManyRequests, forum.CodeCommentCooldown)
	case errors.Is(err, forum.ErrDailyTopicLimit):
		return fiber.NewError(fiber.StatusTooManyRequests, forum.CodeDailyTopicLimit)
	case errors.Is(err, forum.ErrDailyCommentLimit):
		return fiber.NewError(fiber.StatusTooManyRequests, forum.CodeDailyCommentLimit)
	case errors.Is(err, forum.ErrTagMinRequired):
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeTagMinRequired)
	case errors.Is(err, forum.ErrOutboundLinkForbidden):
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeOutboundLinkForbidden)
	case errors.Is(err, forum.ErrMentionsLimit):
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeMentionsLimit)
	case errors.Is(err, forum.ErrDuplicateTitle):
		return fiber.NewError(fiber.StatusUnprocessableEntity, forum.CodeDuplicateTitle)
	case errors.Is(err, forum.ErrGuestLoginRequired):
		return fiber.NewError(fiber.StatusUnauthorized, forum.CodeGuestLoginRequired)
	case errors.Is(err, forum.ErrUseSearchEndpoint):
		return fiber.NewError(fiber.StatusBadRequest, forum.CodeUseSearch)
	case errors.Is(err, forum.ErrInvalidCursor):
		return fiber.NewError(fiber.StatusBadRequest, forum.CodeInvalidCursor)
	default:
		return err
	}
}

// reindex / search 相关 sentinel error。controller 不依赖 search 包，
// 由 bootstrap adapter 将 search 错误转换为这些 controller 级别错误。
// 导出以便 bootstrap adapter 通过 errors.Is 引用。
var (
	ErrReindexRunning     = errors.New("forum: reindex already running")
	ErrReindexNoRun       = errors.New("forum: no reindex run")
	ErrSearchUnavailable  = errors.New("forum: search engine unavailable")
)

// mapReindexError 将 reindex 错误映射为 HTTP 响应。
func mapReindexError(err error) error {
	switch {
	case errors.Is(err, ErrReindexRunning):
		return fiber.NewError(fiber.StatusConflict, forum.CodeReindexRunning)
	case errors.Is(err, ErrReindexNoRun):
		return fiber.NewError(fiber.StatusNotFound, forum.CodeReindexNoRun)
	default:
		return err
	}
}

// queryInt 解析查询参数为 int，解析失败返回 0（L3）。
// 注意：与 Identity paramInt64 返回 error→422 不同，这里静默返回 0，
// 依赖下游 service 的 ID<=0 → 404/422 兜底保证安全（如 GetTopic 对 topicID<=0 返回未找到）。
func queryInt(c fiber.Ctx, key string) int {
	value, _ := strconv.Atoi(c.Query(key))
	return value
}

// paramInt 解析路径参数为 int，解析失败返回 0（L3），安全兜底同 queryInt。
func paramInt(c fiber.Ctx, key string) int {
	value, _ := strconv.Atoi(c.Params(key))
	return value
}
