package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"maps"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	routes "github.com/zhuchunshu/sforum/apps/api/app/Support/Routes"
)

type ProductionRouteGuardAuthorizer struct {
	authorizer routes.CoreGuardAuthorizer
}

func NewProductionRouteGuardAuthorizer() ProductionRouteGuardAuthorizer {
	registry := routes.MustNewCoreGuardEvaluatorRegistry(productionCoreGuardEvaluatorRegistrations())
	return ProductionRouteGuardAuthorizer{authorizer: routes.CoreGuardAuthorizer{Evaluators: registry}}
}

func (a ProductionRouteGuardAuthorizer) Authorize(
	ctx context.Context,
	plan routes.RouteExecutionPlan,
	step routes.RouteExecutionStep,
	request routes.DispatchRequest,
) error {
	// Params 由不可变执行计划解析，不能接受上游中间件伪造的资源身份。
	if !maps.Equal(request.Params, plan.Params()) {
		return ErrRouteGuardUnavailable
	}
	err := a.authorizer.Authorize(ctx, plan, step, request)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, routes.ErrCoreGuardLoginRequired):
		return ErrRouteLoginRequired
	case errors.Is(err, routes.ErrCoreGuardGuestRequired):
		return ErrRouteGuestRequired
	case errors.Is(err, routes.ErrCoreGuardPermissionDenied):
		return ErrRoutePermissionDenied
	case errors.Is(err, routes.ErrCoreGuardEvaluatorUnavailable), errors.Is(err, routes.ErrCoreGuardRegistryInvalid):
		return ErrRouteGuardUnavailable
	default:
		return err
	}
}

func productionCoreGuardEvaluatorRegistrations() []routes.CoreGuardEvaluatorRegistration {
	return []routes.CoreGuardEvaluatorRegistration{
		productionCoreGuardEvaluator("core.guard.attachments.upload", requireDeclaredCoreGuardPermission),
		productionCoreGuardEvaluator("core.guard.forum.author_review", requireAuthenticatedCoreGuardActor),
		productionCoreGuardEvaluator("core.guard.forum.comment_write", requireForumCommentGlobalAuthority),
		productionCoreGuardEvaluator("core.guard.forum.settings", requireForumSettingsAuthority),
		productionCoreGuardEvaluator("core.guard.forum.topic_create", requireDeclaredCoreGuardPermission),
		productionCoreGuardEvaluator("core.guard.forum.topic_delete", requireForumTopicGlobalAuthority),
		productionCoreGuardEvaluator("core.guard.forum.topic_edit", requireForumTopicGlobalAuthority),
		productionCoreGuardEvaluator("core.guard.forum.topic_lock", requireDeclaredCoreGuardPermission),
		productionCoreGuardEvaluator("core.guard.forum.topic_state", requireDeclaredCoreGuardPermission),
		productionCoreGuardEvaluator("core.guard.moderation.report", requireAuthenticatedCoreGuardActor),
		productionCoreGuardEvaluator("core.guard.moderation.review", requireDeclaredCoreGuardPermission),
		productionCoreGuardEvaluator("core.guard.notifications.recipient", requireNotificationRecipientAuthority),
		productionCoreGuardEvaluator("core.guard.pages.admin", requirePagesAdminAuthority),
		productionCoreGuardEvaluator("core.guard.profile.self", requireProfileSelfAuthority),
	}
}

func productionCoreGuardEvaluator(id string, evaluator routes.CoreGuardEvaluatorFunc) routes.CoreGuardEvaluatorRegistration {
	return routes.CoreGuardEvaluatorRegistration{
		EvaluatorID: id, ContractVersion: routes.CoreGuardEvaluatorContractV1, Evaluator: evaluator,
	}
}

func requireAuthenticatedCoreGuardActor(_ context.Context, evaluation routes.CoreGuardEvaluation) error {
	if evaluation.Request.Authenticated && evaluation.Request.ActorID > 0 {
		return nil
	}
	return routes.ErrCoreGuardLoginRequired
}

func requireDeclaredCoreGuardPermission(_ context.Context, evaluation routes.CoreGuardEvaluation) error {
	return requireCoreGuardPermission(evaluation, evaluation.Descriptor.Permissions...)
}

func requireProfileSelfAuthority(_ context.Context, evaluation routes.CoreGuardEvaluation) error {
	if err := requireAuthenticatedCoreGuardActor(context.Background(), evaluation); err != nil {
		return err
	}
	switch evaluation.Descriptor.RouteID {
	case "core.route.profile.my_profile", "core.route.profile.update_my_profile", "core.route.profile.delete_avatar":
		return nil
	case "core.route.profile.upload_avatar":
		return requireCoreGuardPermission(evaluation, identity.PermissionAttachmentUpload)
	default:
		return routes.ErrCoreGuardEvaluatorUnavailable
	}
}

func requireNotificationRecipientAuthority(ctx context.Context, evaluation routes.CoreGuardEvaluation) error {
	switch evaluation.Descriptor.RouteID {
	case "core.route.notifications.list",
		"core.route.notifications.mark_read",
		"core.route.notifications.mark_all_read",
		"core.route.notifications.unread_count":
		// 这些路由没有可选 recipient 参数。核心 Store 始终用当前 ActorID
		// 约束 recipient_user_id，Guard 只负责阻止匿名主体进入该所有权边界。
		return requireAuthenticatedCoreGuardActor(ctx, evaluation)
	default:
		return routes.ErrCoreGuardEvaluatorUnavailable
	}
}

func requirePagesAdminAuthority(_ context.Context, evaluation routes.CoreGuardEvaluation) error {
	switch evaluation.Descriptor.RouteID {
	case "core.route.pages.admin_list",
		"core.route.pages.admin_get",
		"core.route.pages.activate_preview",
		"core.route.pages.admin_added":
		return requireCoreGuardPermission(evaluation,
			identity.PermissionExtensionView,
			identity.PermissionExtensionThemeManage,
			identity.PermissionExtensionManage,
		)
	case "core.route.pages.admin_approve", "core.route.pages.admin_restore":
		// 页面替换批准和恢复会改变公共呈现的可信提供者，只允许超管。
		return requireCoreGuardPermission(evaluation, "*")
	default:
		return routes.ErrCoreGuardEvaluatorUnavailable
	}
}

type forumSettingsGuardInput struct {
	DefaultCategorySlug *string `json:"defaultCategorySlug"`
	TagCreationMode     *string `json:"tagCreationMode"`
	TagPublicPages      *bool   `json:"tagPublicPages"`
	TagMinPerTopic      *int    `json:"tagMinPerTopic"`
	TagMaxPerTopic      *int    `json:"tagMaxPerTopic"`
	TopicsPerPage       *int    `json:"topicsPerPage"`
	CommentsPerPage     *int    `json:"commentsPerPage"`

	TopicTitleMinRunes       *int `json:"topicTitleMinRunes"`
	TopicTitleMaxRunes       *int `json:"topicTitleMaxRunes"`
	TopicContentMinRunes     *int `json:"topicContentMinRunes"`
	TopicContentMaxRunes     *int `json:"topicContentMaxRunes"`
	TopicEditWindowMinutes   *int `json:"topicEditWindowMinutes"`
	TopicCooldownSeconds     *int `json:"topicCooldownSeconds"`
	DailyTopicLimit          *int `json:"dailyTopicLimit"`
	CommentMinRunes          *int `json:"commentMinRunes"`
	CommentMaxRunes          *int `json:"commentMaxRunes"`
	CommentMaxNestingDepth   *int `json:"commentMaxNestingDepth"`
	CommentEditWindowMinutes *int `json:"commentEditWindowMinutes"`
	CommentCooldownSeconds   *int `json:"commentCooldownSeconds"`
	DailyCommentLimit        *int `json:"dailyCommentLimit"`
	ExcerptRuneLimit         *int `json:"excerptRuneLimit"`

	GuestRead               *string `json:"guestRead"`
	ListDefaultSort         *string `json:"listDefaultSort"`
	ListHotWindowDays       *int    `json:"listHotWindowDays"`
	AllowAuthorCloseReplies *bool   `json:"allowAuthorCloseReplies"`
	AllowAuthorDelete       *bool   `json:"allowAuthorDelete"`
	AutoLockIdleDays        *int    `json:"autoLockIdleDays"`
	ShowTopicEditMark       *bool   `json:"showTopicEditMark"`
	DuplicateTitlePolicy    *string `json:"duplicateTitlePolicy"`
	ShowCommentEditMark     *bool   `json:"showCommentEditMark"`
	SoftDeleteVisibility    *string `json:"softDeleteVisibility"`
	MentionsEnabled         *bool   `json:"mentionsEnabled"`
	MentionsMaxPerPost      *int    `json:"mentionsMaxPerPost"`
}

func requireForumSettingsAuthority(_ context.Context, evaluation routes.CoreGuardEvaluation) error {
	if err := requireCoreGuardPermission(evaluation, evaluation.Descriptor.Permissions...); err != nil {
		return err
	}
	switch evaluation.Descriptor.RouteID {
	case "core.route.forum.admin_settings", "core.route.forum.admin_reset_settings":
		return nil
	case "core.route.forum.admin_update_settings":
		input, err := decodeForumSettingsGuardInput(evaluation.Request.Body)
		if err != nil {
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
		if input.DefaultCategorySlug != nil {
			if err := requireCoreGuardPermission(evaluation, identity.PermissionCategoryManage); err != nil {
				return err
			}
		}
		if forumTagSettingsPresent(input) {
			if err := requireCoreGuardPermission(evaluation, identity.PermissionTagManage); err != nil {
				return err
			}
		}
		if forumRuntimeSettingsPresent(input) {
			return requireCoreGuardPermission(evaluation, identity.PermissionForumSettingsManage)
		}
		return nil
	default:
		return routes.ErrCoreGuardEvaluatorUnavailable
	}
}

func decodeForumSettingsGuardInput(body []byte) (forumSettingsGuardInput, error) {
	var input forumSettingsGuardInput
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return forumSettingsGuardInput{}, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return forumSettingsGuardInput{}, errors.New("forum settings guard: trailing JSON value")
	}
	return input, nil
}

func forumTagSettingsPresent(input forumSettingsGuardInput) bool {
	return input.TagCreationMode != nil || input.TagPublicPages != nil ||
		input.TagMinPerTopic != nil || input.TagMaxPerTopic != nil
}

func forumRuntimeSettingsPresent(input forumSettingsGuardInput) bool {
	return input.TopicsPerPage != nil || input.CommentsPerPage != nil ||
		input.TopicTitleMinRunes != nil || input.TopicTitleMaxRunes != nil ||
		input.TopicContentMinRunes != nil || input.TopicContentMaxRunes != nil ||
		input.TopicEditWindowMinutes != nil || input.TopicCooldownSeconds != nil ||
		input.DailyTopicLimit != nil || input.CommentMinRunes != nil ||
		input.CommentMaxRunes != nil || input.CommentMaxNestingDepth != nil ||
		input.CommentEditWindowMinutes != nil || input.CommentCooldownSeconds != nil ||
		input.DailyCommentLimit != nil || input.ExcerptRuneLimit != nil ||
		input.GuestRead != nil || input.ListDefaultSort != nil || input.ListHotWindowDays != nil ||
		input.AllowAuthorCloseReplies != nil || input.AllowAuthorDelete != nil || input.AutoLockIdleDays != nil ||
		input.ShowTopicEditMark != nil || input.DuplicateTitlePolicy != nil || input.ShowCommentEditMark != nil ||
		input.SoftDeleteVisibility != nil || input.MentionsEnabled != nil || input.MentionsMaxPerPost != nil
}

func requireForumTopicGlobalAuthority(_ context.Context, evaluation routes.CoreGuardEvaluation) error {
	switch evaluation.Descriptor.RouteID {
	case "core.route.forum.update_topic":
		return requireCoreGuardPermission(evaluation, identity.PermissionTopicEditAny)
	case "core.route.forum.delete_topic":
		return requireCoreGuardPermission(evaluation, identity.PermissionTopicDeleteAny)
	default:
		return routes.ErrCoreGuardEvaluatorUnavailable
	}
}

func requireForumCommentGlobalAuthority(_ context.Context, evaluation routes.CoreGuardEvaluation) error {
	switch evaluation.Descriptor.RouteID {
	case "core.route.forum.update_comment":
		return requireCoreGuardPermission(evaluation, identity.PermissionPostEditAny)
	case "core.route.forum.delete_comment":
		return requireCoreGuardPermission(evaluation, identity.PermissionPostDeleteAny)
	default:
		return routes.ErrCoreGuardEvaluatorUnavailable
	}
}

func requireCoreGuardPermission(evaluation routes.CoreGuardEvaluation, permissions ...string) error {
	if !evaluation.Request.Authenticated || evaluation.Request.ActorID <= 0 {
		return routes.ErrCoreGuardLoginRequired
	}
	if evaluation.Request.Permissions["*"] {
		return nil
	}
	for _, permission := range permissions {
		if permission != "" && evaluation.Request.Permissions[permission] {
			return nil
		}
	}
	return routes.ErrCoreGuardPermissionDenied
}
