package http

import (
	"context"
	"errors"

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
		productionCoreGuardEvaluator("core.guard.forum.topic_create", requireDeclaredCoreGuardPermission),
		productionCoreGuardEvaluator("core.guard.forum.topic_delete", requireForumTopicGlobalAuthority),
		productionCoreGuardEvaluator("core.guard.forum.topic_edit", requireForumTopicGlobalAuthority),
		productionCoreGuardEvaluator("core.guard.forum.topic_lock", requireDeclaredCoreGuardPermission),
		productionCoreGuardEvaluator("core.guard.forum.topic_state", requireDeclaredCoreGuardPermission),
		productionCoreGuardEvaluator("core.guard.moderation.report", requireAuthenticatedCoreGuardActor),
		productionCoreGuardEvaluator("core.guard.moderation.review", requireDeclaredCoreGuardPermission),
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
