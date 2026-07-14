package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"net/url"
	"path"
	"strings"

	apitokens "github.com/zhuchunshu/sforum/apps/api/app/Models/APITokens"
	attachments "github.com/zhuchunshu/sforum/apps/api/app/Models/Attachments"
	entitymeta "github.com/zhuchunshu/sforum/apps/api/app/Models/EntityMeta"
	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	humanverify "github.com/zhuchunshu/sforum/apps/api/app/Support/HumanVerify"
	pages "github.com/zhuchunshu/sforum/apps/api/app/Support/Pages"
	routes "github.com/zhuchunshu/sforum/apps/api/app/Support/Routes"
)

type ProductionRouteGuardAuthorizer struct {
	authorizer routes.CoreGuardAuthorizer
}

type ForumReadPolicy interface {
	ForumReadPolicySnapshot() (guestRead string, softDeleteVisibility string, revision uint64, ok bool)
}

type ExtensionGuardPolicy interface {
	Lookup(extensionID string) (extensions.GuardPolicyLookup, bool)
}

type DeclaredExtensionRoutePolicy interface {
	LookupDeclaredRoute(extensionID, method, routePath string) (extensions.DeclaredRouteGuardLookup, bool)
}

type OptionsOwnerPolicy interface {
	OptionGuardManagePermissions(names []string) ([]string, bool)
}

type PageResolvePolicy interface {
	Revision() uint64
	Resolve(context.Context, string) (pages.ResolvedPage, error)
	ResolveAddedPathMatch(string) (pages.RouteMatch, bool)
}

type IdentityAdminGuardPolicy interface {
	LoadAdminGuardSubject(context.Context, int64) (identity.AdminGuardSubject, error)
}

type IdentitySessionGuardPolicy interface {
	LoadSessionGuardSubject(context.Context, string) (identity.SessionGuardSubject, error)
}

type IdentityAPITokenGuardPolicy interface {
	LoadGuardSubject(context.Context, int64) (apitokens.GuardSubject, error)
}

type EntityMetaValueGuardPolicy interface {
	LoadValueGuardSubject(context.Context, string, int64, []string) (entitymeta.ValueGuardSubject, error)
}

type AttachmentReadGuardPolicy interface {
	LoadReadGuardSubject(context.Context, string) (attachments.ReadGuardSubject, error)
}

type ProductionRouteGuardPolicies struct {
	ForumRead         ForumReadPolicy
	Extensions        ExtensionGuardPolicy
	DeclaredRoutes    DeclaredExtensionRoutePolicy
	Options           OptionsOwnerPolicy
	Pages             PageResolvePolicy
	IdentityAdmins    IdentityAdminGuardPolicy
	IdentitySessions  IdentitySessionGuardPolicy
	IdentityAPITokens IdentityAPITokenGuardPolicy
	EntityMetaValues  EntityMetaValueGuardPolicy
	AttachmentReads   AttachmentReadGuardPolicy
	ForumComments     ForumCommentCreateGuardPolicy
}

func NewProductionRouteGuardAuthorizer() ProductionRouteGuardAuthorizer {
	return NewProductionRouteGuardAuthorizerWithPolicies(ProductionRouteGuardPolicies{})
}

func NewProductionRouteGuardAuthorizerWithPolicies(policies ProductionRouteGuardPolicies) ProductionRouteGuardAuthorizer {
	registry := routes.MustNewCoreGuardEvaluatorRegistry(productionCoreGuardEvaluatorRegistrationsWithPolicies(policies))
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
	return productionCoreGuardEvaluatorRegistrationsWithPolicies(ProductionRouteGuardPolicies{})
}

func productionCoreGuardEvaluatorRegistrationsWithPolicies(policies ProductionRouteGuardPolicies) []routes.CoreGuardEvaluatorRegistration {
	return []routes.CoreGuardEvaluatorRegistration{
		productionCoreGuardEvaluator("core.guard.attachments.read", attachmentReadGuardEvaluator(policies.AttachmentReads, policies.ForumRead)),
		productionCoreGuardEvaluator("core.guard.attachments.upload", requireDeclaredCoreGuardPermission),
		productionCoreGuardEvaluator("core.guard.extensions.mutation", extensionsMutationGuardEvaluator(policies.Extensions)),
		productionCoreGuardEvaluator("core.guard.extensions.read", extensionsReadGuardEvaluator(policies.Extensions)),
		productionCoreGuardEvaluator("core.guard.extensions.declared_route", declaredExtensionRouteGuardEvaluator(policies.DeclaredRoutes)),
		productionCoreGuardEvaluator("core.guard.entity_meta.read", entityMetaReadGuardEvaluator(policies.EntityMetaValues)),
		productionCoreGuardEvaluator("core.guard.entity_meta.write", entityMetaWriteGuardEvaluator(policies.EntityMetaValues)),
		productionCoreGuardEvaluator("core.guard.forum.author_review", requireAuthenticatedCoreGuardActor),
		productionCoreGuardEvaluator("core.guard.forum.comment_write", requireForumCommentGlobalAuthority),
		productionCoreGuardEvaluator("core.guard.forum.comment_create", forumCommentCreateGuardEvaluator(policies.ForumComments)),
		productionCoreGuardEvaluator("core.guard.forum.read", forumReadGuardEvaluator(policies.ForumRead)),
		productionCoreGuardEvaluator("core.guard.forum.settings", requireForumSettingsAuthority),
		productionCoreGuardEvaluator("core.guard.forum.topic_create", requireDeclaredCoreGuardPermission),
		productionCoreGuardEvaluator("core.guard.forum.topic_delete", requireForumTopicGlobalAuthority),
		productionCoreGuardEvaluator("core.guard.forum.topic_edit", requireForumTopicGlobalAuthority),
		productionCoreGuardEvaluator("core.guard.forum.topic_lock", requireDeclaredCoreGuardPermission),
		productionCoreGuardEvaluator("core.guard.forum.topic_state", requireDeclaredCoreGuardPermission),
		productionCoreGuardEvaluator("core.guard.identity.admin", identityAdminGuardEvaluator(policies.IdentityAdmins)),
		productionCoreGuardEvaluator("core.guard.identity.bootstrap", identityBootstrapGuardEvaluator),
		productionCoreGuardEvaluator("core.guard.identity.human_verification", requireHumanVerificationChallengeAuthority),
		productionCoreGuardEvaluator("core.guard.identity.self_credentials", identitySelfCredentialsGuardEvaluator(
			policies.IdentitySessions, policies.IdentityAPITokens,
		)),
		productionCoreGuardEvaluator("core.guard.moderation.report", requireAuthenticatedCoreGuardActor),
		productionCoreGuardEvaluator("core.guard.moderation.review", requireDeclaredCoreGuardPermission),
		productionCoreGuardEvaluator("core.guard.notifications.recipient", requireNotificationRecipientAuthority),
		productionCoreGuardEvaluator("core.guard.options.owner", optionsOwnerGuardEvaluator(policies.Options)),
		productionCoreGuardEvaluator("core.guard.pages.admin", requirePagesAdminAuthority),
		productionCoreGuardEvaluator("core.guard.pages.catalog", requirePagesCatalogAuthority),
		productionCoreGuardEvaluator("core.guard.pages.resolve", pagesResolveGuardEvaluator(policies.Pages)),
		productionCoreGuardEvaluator("core.guard.webhooks.inbound", requireInboundWebhookAuthority),
		productionCoreGuardEvaluator("core.guard.pages.theme_asset", themeAssetGuardEvaluator(policies.Extensions)),
		productionCoreGuardEvaluator("core.guard.profile.self", requireProfileSelfAuthority),
		productionCoreGuardEvaluator("core.guard.seo.read", requireSEOReadAuthority),
	}
}

func attachmentReadGuardEvaluator(policy AttachmentReadGuardPolicy, forumRead ForumReadPolicy) routes.CoreGuardEvaluatorFunc {
	return func(ctx context.Context, evaluation routes.CoreGuardEvaluation) error {
		if policy == nil || (evaluation.Descriptor.RouteID != "core.route.attachments.get" &&
			evaluation.Descriptor.RouteID != "core.route.attachments.content") ||
			len(evaluation.Request.Body) != 0 || evaluation.Request.Query != "" {
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
		publicID := evaluation.Request.Params["publicId"]
		if publicID == "" || publicID != strings.TrimSpace(publicID) {
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
		subject, err := policy.LoadReadGuardSubject(ctx, publicID)
		if err != nil || !subject.Exists || subject.PublicID != publicID {
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
		actor := identity.Actor{ID: evaluation.Request.ActorID, Permissions: maps.Clone(evaluation.Request.Permissions)}
		if evaluation.Request.Authenticated && evaluation.Request.ActorID > 0 {
			actor.Status = identity.UserStatusActive
		}
		if evaluation.Request.Permissions["*"] {
			actor.RoleKeys = []string{identity.RoleSuperAdmin}
		}
		guestLoginRequired := true
		if forumRead != nil {
			guestRead, _, _, ok := forumRead.ForumReadPolicySnapshot()
			if ok {
				guestLoginRequired = strings.TrimSpace(guestRead) != "public"
			}
		}
		err = attachments.AuthorizeReadGuardSubject(actor, subject, guestLoginRequired)
		switch {
		case err == nil:
			return nil
		case errors.Is(err, attachments.ErrGuestLoginRequired):
			return routes.ErrCoreGuardLoginRequired
		case errors.Is(err, identity.ErrPermissionDenied):
			return routes.ErrCoreGuardPermissionDenied
		default:
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
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

func requireExtensionsReadAuthority(_ context.Context, evaluation routes.CoreGuardEvaluation) error {
	switch evaluation.Descriptor.RouteID {
	case "core.route.extensions.list",
		"core.route.extensions.events",
		"core.route.extensions.lifecycle_operations",
		"core.route.extensions.lifecycle_operation",
		"core.route.extensions.contribution_points",
		"core.route.extensions.contributions",
		"core.route.extensions.event_definitions",
		"core.route.extensions.event_deliveries",
		"core.route.extensions.navigation",
		"core.route.extensions.inspect_provider_slots",
		"core.route.extensions.inspect_route",
		"core.route.extensions.route_provider_conflicts",
		"core.route.extensions.route_provider_events",
		"core.route.extensions.route_provider_selection":
		return requireCoreGuardPermission(evaluation,
			identity.PermissionExtensionView,
			identity.PermissionExtensionManage,
		)
	case "core.route.extensions.list_migrations":
		return requireCoreGuardPermission(evaluation,
			identity.PermissionExtensionView,
			identity.PermissionExtensionPluginManage,
			identity.PermissionExtensionManage,
		)
	case "core.route.extensions.executable_trust_status":
		return requireCoreGuardPermission(evaluation,
			identity.PermissionExtensionView,
			identity.PermissionExtensionPluginManage,
			identity.PermissionExtensionThemeManage,
			identity.PermissionExtensionManage,
		)
	default:
		// settings/frontend 读取依赖目标扩展类型、provider 与精确制品状态。
		return routes.ErrCoreGuardEvaluatorUnavailable
	}
}

func extensionsReadGuardEvaluator(policy ExtensionGuardPolicy) routes.CoreGuardEvaluatorFunc {
	return func(ctx context.Context, evaluation routes.CoreGuardEvaluation) error {
		if err := requireExtensionsReadAuthority(ctx, evaluation); !errors.Is(err, routes.ErrCoreGuardEvaluatorUnavailable) {
			return err
		}
		lookup, ok := extensionGuardPolicyLookup(policy, evaluation)
		if !ok || !lookup.Found {
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
		switch evaluation.Descriptor.RouteID {
		case "core.route.extensions.settings":
			return requireExtensionSettingsAuthority(evaluation, lookup.Entry)
		case "core.route.extensions.frontend_status":
			return requireExtensionFrontendStatusAuthority(evaluation, lookup.Entry)
		case "core.route.extensions.frontend_asset":
			if lookup.SafeMode || !lookup.Entry.HasPrebuiltAdmin || !lookup.Entry.FrontendArtifactTrusted ||
				evaluation.Request.Params["digest"] != lookup.Entry.AdminFrontendDigest ||
				(evaluation.Request.Params["asset"] != "entry" && evaluation.Request.Params["asset"] != "style") {
				return routes.ErrCoreGuardEvaluatorUnavailable
			}
			return requireExtensionSettingsAuthority(evaluation, lookup.Entry)
		default:
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
	}
}

func requireExtensionsMutationAuthority(_ context.Context, evaluation routes.CoreGuardEvaluation) error {
	switch evaluation.Descriptor.RouteID {
	case "core.route.extensions.disable", "core.route.extensions.rollback":
		return requireCoreGuardPermission(evaluation,
			identity.PermissionExtensionPluginManage,
			identity.PermissionExtensionManage,
		)
	case "core.route.extensions.activate":
		return requireCoreGuardPermission(evaluation,
			identity.PermissionExtensionThemeManage,
			identity.PermissionExtensionManage,
		)
	case "core.route.extensions.recover_lifecycle_operation":
		if err := requireCoreGuardPermission(evaluation,
			identity.PermissionExtensionPluginManage,
			identity.PermissionExtensionManage,
		); err != nil {
			return err
		}
		var input extensions.LifecycleRecoveryInput
		if err := decodeGuardJSON(evaluation.Request.Body, &input); err != nil {
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
		if input.EscalateForced {
			return requireCoreGuardPermission(evaluation, "*")
		}
		return nil
	case "core.route.extensions.revoke_executable_trust",
		"core.route.extensions.issue_executable_trust_challenge",
		"core.route.extensions.select_route_provider",
		"core.route.extensions.reset_route_provider":
		return requireCoreGuardPermission(evaluation, "*")
	default:
		// 安装、启用、升级、迁移、校验和设置写入都依赖目标制品策略。
		return routes.ErrCoreGuardEvaluatorUnavailable
	}
}

func extensionsMutationGuardEvaluator(policy ExtensionGuardPolicy) routes.CoreGuardEvaluatorFunc {
	return func(ctx context.Context, evaluation routes.CoreGuardEvaluation) error {
		if err := requireExtensionsMutationAuthority(ctx, evaluation); !errors.Is(err, routes.ErrCoreGuardEvaluatorUnavailable) {
			return err
		}
		lookup, ok := extensionGuardPolicyLookup(policy, evaluation)
		if !ok {
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
		if evaluation.Descriptor.RouteID == "core.route.extensions.install" {
			if !lookup.TrustChallengesEnabled {
				return requireCoreGuardPermission(evaluation, "*")
			}
			return requireCoreGuardPermission(evaluation,
				identity.PermissionExtensionPluginManage,
				identity.PermissionExtensionManage,
			)
		}
		if !lookup.Found {
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
		entry := lookup.Entry
		switch evaluation.Descriptor.RouteID {
		case "core.route.extensions.uninstall":
			if entry.Source != extensions.SourceBuiltin && entry.HasExecutableBackend &&
				!(entry.LifecycleV2 && (entry.Status == extensions.StatusEnabled || entry.Status == extensions.StatusDisabled)) {
				return requireCoreGuardPermission(evaluation, "*")
			}
			return requireExtensionPluginAuthority(evaluation)
		case "core.route.extensions.enable":
			if lookup.SafeMode || entry.ExtensionType != extensions.TypePlugin {
				return routes.ErrCoreGuardEvaluatorUnavailable
			}
			if lookup.TrustChallengesEnabled {
				if entry.ReviewTrustRequired && !entry.ReviewArtifactTrusted {
					return requireCoreGuardPermission(evaluation, "*")
				}
			} else if entry.Source != extensions.SourceBuiltin && entry.HasExecutableBackend {
				return requireCoreGuardPermission(evaluation, "*")
			}
			return requireExtensionPluginAuthority(evaluation)
		case "core.route.extensions.apply_migrations", "core.route.extensions.verify":
			if lookup.SafeMode {
				return routes.ErrCoreGuardEvaluatorUnavailable
			}
			if entry.Source != extensions.SourceBuiltin && entry.HasExecutableBackend {
				return requireCoreGuardPermission(evaluation, "*")
			}
			return requireExtensionPluginAuthority(evaluation)
		case "core.route.extensions.update_settings", "core.route.extensions.reset_settings":
			return requireExtensionSettingsAuthority(evaluation, entry)
		case "core.route.extensions.execute_settings_action":
			if lookup.SafeMode {
				return routes.ErrCoreGuardEvaluatorUnavailable
			}
			return requireExtensionSettingsAuthority(evaluation, entry)
		case "core.route.extensions.upgrade":
			if lookup.SafeMode || entry.ExtensionType != extensions.TypePlugin || !entry.LifecycleV2 ||
				entry.Status != extensions.StatusEnabled || !entry.HasStagedArtifact {
				return routes.ErrCoreGuardEvaluatorUnavailable
			}
			if entry.StagedTrustRequired && !entry.StagedArtifactTrusted {
				return requireCoreGuardPermission(evaluation, "*")
			}
			return requireExtensionPluginAuthority(evaluation)
		default:
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
	}
}

func extensionGuardPolicyLookup(policy ExtensionGuardPolicy, evaluation routes.CoreGuardEvaluation) (extensions.GuardPolicyLookup, bool) {
	if policy == nil {
		return extensions.GuardPolicyLookup{}, false
	}
	id := ""
	if evaluation.Descriptor.RouteID != "core.route.extensions.install" {
		id = evaluation.Request.Params["id"]
		if id == "" {
			return extensions.GuardPolicyLookup{}, false
		}
	}
	lookup, ok := policy.Lookup(id)
	if !ok || lookup.Revision == 0 || lookup.Found && lookup.Entry.ExtensionID != id {
		return extensions.GuardPolicyLookup{}, false
	}
	return lookup, true
}

func requireExtensionPluginAuthority(evaluation routes.CoreGuardEvaluation) error {
	return requireCoreGuardPermission(evaluation,
		identity.PermissionExtensionPluginManage,
		identity.PermissionExtensionManage,
	)
}

func requireExtensionSettingsAuthority(evaluation routes.CoreGuardEvaluation, entry extensions.GuardPolicyEntry) error {
	switch entry.ExtensionType {
	case extensions.TypeTheme:
		return requireCoreGuardPermission(evaluation,
			identity.PermissionExtensionThemeManage,
			identity.PermissionExtensionManage,
		)
	case extensions.TypePlugin:
		permissions := []string{identity.PermissionExtensionPluginManage, identity.PermissionExtensionManage}
		if entry.HasMailProvider {
			permissions = append(permissions, identity.PermissionSettingsMailManage)
		}
		return requireCoreGuardPermission(evaluation, permissions...)
	default:
		return routes.ErrCoreGuardEvaluatorUnavailable
	}
}

func requireExtensionFrontendStatusAuthority(evaluation routes.CoreGuardEvaluation, entry extensions.GuardPolicyEntry) error {
	permissions := []string{
		identity.PermissionExtensionView,
		identity.PermissionExtensionPluginManage,
		identity.PermissionExtensionThemeManage,
		identity.PermissionExtensionManage,
	}
	if entry.HasMailProvider {
		permissions = append(permissions, identity.PermissionSettingsMailManage)
	}
	return requireCoreGuardPermission(evaluation, permissions...)
}

func declaredExtensionRouteGuardEvaluator(policy DeclaredExtensionRoutePolicy) routes.CoreGuardEvaluatorFunc {
	return func(_ context.Context, evaluation routes.CoreGuardEvaluation) error {
		if evaluation.Descriptor.RouteID != "core.route.extensions.proxy_extension_route" || policy == nil {
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
		extensionID := evaluation.Request.Params["extensionId"]
		routePath := evaluation.Request.Params["*"]
		if routePath == "" {
			routePath = evaluation.Request.Params["path"]
		}
		if extensionID == "" || routePath == "" {
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
		lookup, ok := policy.LookupDeclaredRoute(extensionID, evaluation.RequestMethod, "/"+strings.TrimPrefix(routePath, "/"))
		if !ok || lookup.Revision == 0 || lookup.ExtensionID != extensionID ||
			lookup.ExtensionVersion == "" || lookup.PackageDigest == "" {
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
		switch lookup.Access {
		case extensions.RouteAccessPublic:
			return nil
		case extensions.RouteAccessLogin:
			return requireAuthenticatedCoreGuardActor(context.Background(), evaluation)
		case extensions.RouteAccessPermission:
			if strings.TrimSpace(lookup.Permission) == "" {
				return routes.ErrCoreGuardEvaluatorUnavailable
			}
			return requireCoreGuardPermission(evaluation, lookup.Permission)
		default:
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
	}
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

func optionsOwnerGuardEvaluator(policy OptionsOwnerPolicy) routes.CoreGuardEvaluatorFunc {
	return func(_ context.Context, evaluation routes.CoreGuardEvaluation) error {
		if policy == nil {
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
		var names []string
		switch evaluation.Descriptor.RouteID {
		case "core.route.options.list_admin":
			// nil 明确请求静态目录中的 any-of 权限。
			permissions, ok := policy.OptionGuardManagePermissions(nil)
			if !ok {
				return routes.ErrCoreGuardEvaluatorUnavailable
			}
			return requireCoreGuardPermission(evaluation, permissions...)
		case "core.route.options.update":
			var input optionGuardUpdateInput
			if err := decodeGuardJSON(evaluation.Request.Body, &input); err != nil {
				return routes.ErrCoreGuardEvaluatorUnavailable
			}
			names = []string{input.Name}
		case "core.route.options.update_admin":
			var input optionGuardUpdateManyInput
			if err := decodeGuardJSON(evaluation.Request.Body, &input); err != nil || len(input.Options) == 0 {
				return routes.ErrCoreGuardEvaluatorUnavailable
			}
			names = make([]string, 0, len(input.Options))
			for _, item := range input.Options {
				names = append(names, item.Name)
			}
		default:
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
		permissions, ok := policy.OptionGuardManagePermissions(names)
		if !ok {
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
		return requireAllCoreGuardPermissions(evaluation, permissions...)
	}
}

type optionGuardUpdateInput struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type optionGuardUpdateManyInput struct {
	Options []optionGuardUpdateInput `json:"options"`
}

// identityBootstrapGuardEvaluator keeps executable authentication mutations
// Host-owned. Login, registration, and recovery combine one-use verification,
// credential mutation, and raw session authority, so an inherited route guard
// cannot delegate them without weakening core policy. A plugin must instead
// declare the separately confirmed custom-guard/raw-request authority.
func identityBootstrapGuardEvaluator(_ context.Context, evaluation routes.CoreGuardEvaluation) error {
	switch evaluation.Descriptor.RouteID {
	case "core.route.identity.registration_status":
		return nil
	case "core.route.identity.login",
		"core.route.identity.register",
		"core.route.identity.password_reset_request",
		"core.route.identity.password_reset_confirm":
		return routes.ErrCoreGuardEvaluatorUnavailable
	default:
		return routes.ErrCoreGuardEvaluatorUnavailable
	}
}

func requireHumanVerificationChallengeAuthority(_ context.Context, evaluation routes.CoreGuardEvaluation) error {
	if evaluation.Descriptor.RouteID != "core.route.identity.human_verification_challenge" {
		return routes.ErrCoreGuardEvaluatorUnavailable
	}
	query, err := url.ParseQuery(evaluation.Request.Query)
	if err != nil || len(query) != 1 || len(query["purpose"]) != 1 {
		return routes.ErrCoreGuardEvaluatorUnavailable
	}
	switch humanverify.Purpose(query.Get("purpose")) {
	case humanverify.PurposeRegister, humanverify.PurposePasswordReset,
		humanverify.PurposeLoginRisk, humanverify.PurposePostRisk:
		return nil
	default:
		return routes.ErrCoreGuardEvaluatorUnavailable
	}
}

func requirePagesCatalogAuthority(_ context.Context, evaluation routes.CoreGuardEvaluation) error {
	if evaluation.Descriptor.RouteID == "core.route.pages.public_catalog" {
		return nil
	}
	return routes.ErrCoreGuardEvaluatorUnavailable
}

func pagesResolveGuardEvaluator(policy PageResolvePolicy) routes.CoreGuardEvaluatorFunc {
	return func(ctx context.Context, evaluation routes.CoreGuardEvaluation) error {
		if policy == nil {
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
		query, err := url.ParseQuery(evaluation.Request.Query)
		if err != nil {
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
		before := policy.Revision()
		if before == 0 {
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
		var access pages.Access
		var permission string
		switch evaluation.Descriptor.RouteID {
		case "core.route.pages.resolve":
			if len(query) != 1 || len(query["id"]) != 1 || strings.TrimSpace(query.Get("id")) == "" {
				return routes.ErrCoreGuardEvaluatorUnavailable
			}
			resolved, resolveErr := policy.Resolve(ctx, strings.TrimSpace(query.Get("id")))
			if resolveErr != nil {
				return routes.ErrCoreGuardEvaluatorUnavailable
			}
			access = resolved.Page.Access
		case "core.route.pages.resolve_path":
			if len(query) != 1 || len(query["path"]) != 1 || strings.TrimSpace(query.Get("path")) == "" ||
				pages.IsReservedPath(query.Get("path")) {
				return routes.ErrCoreGuardEvaluatorUnavailable
			}
			match, found := policy.ResolveAddedPathMatch(query.Get("path"))
			if !found {
				return routes.ErrCoreGuardEvaluatorUnavailable
			}
			access, permission = match.Contribution.Access, match.Contribution.Permission
		default:
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
		if policy.Revision() != before {
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
		return requirePageAccessAuthority(evaluation, access, permission)
	}
}

func requirePageAccessAuthority(evaluation routes.CoreGuardEvaluation, access pages.Access, permission string) error {
	normalized, err := pages.NormalizeAccess(string(access))
	if err != nil {
		return routes.ErrCoreGuardEvaluatorUnavailable
	}
	switch normalized {
	case pages.AccessPublic:
		return nil
	case pages.AccessLogin:
		return requireAuthenticatedCoreGuardActor(context.Background(), evaluation)
	case pages.AccessGuest:
		if evaluation.Request.Authenticated || evaluation.Request.ActorID > 0 {
			return routes.ErrCoreGuardGuestRequired
		}
		return nil
	case pages.AccessModeration:
		return requireCoreGuardPermission(evaluation, identity.PermissionModerationReview)
	case pages.AccessPermission:
		permission = strings.TrimSpace(permission)
		if permission == "" {
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
		return requireCoreGuardPermission(evaluation, permission)
	default:
		return routes.ErrCoreGuardEvaluatorUnavailable
	}
}

func requireEntityMetaReadAuthority(_ context.Context, evaluation routes.CoreGuardEvaluation) error {
	if evaluation.Descriptor.RouteID != "core.route.entity_meta.list_public_definitions" {
		return routes.ErrCoreGuardEvaluatorUnavailable
	}
	query, err := url.ParseQuery(evaluation.Request.Query)
	if err != nil || len(query) != 1 || len(query["entityType"]) != 1 {
		return routes.ErrCoreGuardEvaluatorUnavailable
	}
	switch strings.TrimSpace(query.Get("entityType")) {
	case entitymeta.EntityUser, entitymeta.EntityTopic:
		return nil
	default:
		return routes.ErrCoreGuardEvaluatorUnavailable
	}
}

func requireSEOReadAuthority(_ context.Context, evaluation routes.CoreGuardEvaluation) error {
	if evaluation.Descriptor.RouteID == "core.route.seo.list" {
		return nil
	}
	return routes.ErrCoreGuardEvaluatorUnavailable
}

func requireInboundWebhookAuthority(_ context.Context, evaluation routes.CoreGuardEvaluation) error {
	if evaluation.Descriptor.RouteID != "core.route.webhooks.inbound" {
		return routes.ErrCoreGuardEvaluatorUnavailable
	}
	source := strings.TrimSpace(evaluation.Request.Params["source"])
	if source == "" || len(source) > 64 || len(evaluation.Request.Body) == 0 {
		return routes.ErrCoreGuardEvaluatorUnavailable
	}
	return nil
}

func themeAssetGuardEvaluator(policy ExtensionGuardPolicy) routes.CoreGuardEvaluatorFunc {
	return func(_ context.Context, evaluation routes.CoreGuardEvaluation) error {
		if evaluation.Descriptor.RouteID != "core.route.pages.theme_asset" || policy == nil {
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
		extensionID := evaluation.Request.Params["extensionId"]
		lookup, ok := policy.Lookup(extensionID)
		if !ok || lookup.Revision == 0 || lookup.SafeMode || !lookup.Found ||
			lookup.Entry.ExtensionID != extensionID || lookup.Entry.ExtensionType != extensions.TypeTheme ||
			lookup.Entry.Status != extensions.StatusEnabled || lookup.Entry.PackageDigest == "" {
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
		query, err := url.ParseQuery(evaluation.Request.Query)
		if err != nil || len(query) != 1 {
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
		digests := query["v"]
		if len(digests) != 1 || !strings.EqualFold(digests[0], lookup.Entry.PackageDigest) {
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
		relative := evaluation.Request.Params["*"]
		if relative == "" {
			relative = evaluation.Request.Params["path"]
		}
		if relative == "" || strings.ContainsAny(relative, "\\\x00") ||
			strings.TrimPrefix(path.Clean("/"+relative), "/") != relative {
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
		if _, allowed := pages.AllowedThemeAssetExt[strings.ToLower(path.Ext(relative))]; !allowed {
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
		return nil
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

func forumReadGuardEvaluator(policy ForumReadPolicy) routes.CoreGuardEvaluatorFunc {
	return func(ctx context.Context, evaluation routes.CoreGuardEvaluation) error {
		if evaluation.Descriptor.RouteID == "core.route.forum.composer_toolbar" {
			// 工具栏与发帖入口一致，只对当前活跃主体开放，不接收目标资源。
			return requireAuthenticatedCoreGuardActor(ctx, evaluation)
		}
		if !forumReadPolicyRoute(evaluation.Descriptor.RouteID) || policy == nil {
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
		guestRead, softDeleteVisibility, revision, ok := policy.ForumReadPolicySnapshot()
		if !ok || revision == 0 || !validForumSoftDeleteVisibility(softDeleteVisibility) {
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
		switch guestRead {
		case "public":
			return nil
		case "login_required":
			return requireAuthenticatedCoreGuardActor(ctx, evaluation)
		default:
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
	}
}

func forumReadPolicyRoute(routeID string) bool {
	switch routeID {
	case "core.route.forum.categories",
		"core.route.forum.category_groups",
		"core.route.forum.replies",
		"core.route.forum.search",
		"core.route.forum.tags",
		"core.route.forum.topics",
		"core.route.forum.topic",
		"core.route.forum.comments",
		"core.route.forum.topic_by_slug":
		return true
	default:
		return false
	}
}

func validForumSoftDeleteVisibility(value string) bool {
	switch value {
	case "hidden", "staff_only", "author_and_staff":
		return true
	default:
		return false
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
	if err := decodeGuardJSON(body, &input); err != nil {
		return forumSettingsGuardInput{}, err
	}
	return input, nil
}

func decodeGuardJSON(body []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("route guard: trailing JSON value")
	}
	return nil
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

func requireIdentityAdminAuthority(_ context.Context, evaluation routes.CoreGuardEvaluation) error {
	switch evaluation.Descriptor.RouteID {
	case "core.route.identity.list_permissions", "core.route.identity.permission_matrix":
		return requireCoreGuardPermission(evaluation,
			identity.PermissionRoleManage,
			identity.PermissionUserManage,
			identity.PermissionUserView,
			identity.PermissionUserPermissionOverride,
		)
	case "core.route.identity.list_roles",
		"core.route.identity.create_role",
		"core.route.identity.update_role":
		return requireCoreGuardPermission(evaluation, identity.PermissionRoleManage)
	case "core.route.identity.list_users", "core.route.identity.get_user":
		// user.manage 是 user.view 的兼容父权限；生产会话通常已展开，
		// 这里仍显式接受父权限，避免旧会话在 Guard 层被错误收窄。
		return requireCoreGuardPermission(evaluation, identity.PermissionUserView, identity.PermissionUserManage)
	case "core.route.identity.delete_role":
		roleKey := strings.TrimSpace(evaluation.Request.Params["roleKey"])
		if roleKey == "" || roleKey == identity.RoleMember || identity.IsBuiltInSystemRole(roleKey) {
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
		return requireCoreGuardPermission(evaluation, identity.PermissionRoleManage)
	case "core.route.identity.replace_role_permissions":
		roleKey := strings.TrimSpace(evaluation.Request.Params["roleKey"])
		if roleKey == "" || roleKey == identity.RoleSuperAdmin {
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
		var input replaceRolePermissionsGuardInput
		if err := decodeGuardJSON(evaluation.Request.Body, &input); err != nil {
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
		for _, permission := range input.Permissions {
			if strings.TrimSpace(permission) == "" {
				return routes.ErrCoreGuardEvaluatorUnavailable
			}
		}
		return requireCoreGuardPermission(evaluation, identity.PermissionRoleManage)
	default:
		// 删除/改角色权限和用户写操作都依赖目标资源或请求字段，
		// 当前 Guard 输入不能完整复现 Service 的保护，继续保持关闭。
		return routes.ErrCoreGuardEvaluatorUnavailable
	}
}

type replaceRolePermissionsGuardInput struct {
	Permissions []string `json:"permissions"`
}

func requireIdentitySelfCredentialsAuthority(ctx context.Context, evaluation routes.CoreGuardEvaluation) error {
	switch evaluation.Descriptor.RouteID {
	case "core.route.identity.list_sessions", "core.route.identity.revoke_other_sessions":
		// 两条路径始终以 Host 认证的 ActorID 查询/更新，不接收目标 user_id。
		return requireAuthenticatedCoreGuardActor(ctx, evaluation)
	case "core.route.identity.list_apitokens":
		if err := requireCookieCredentialAuthority(ctx, evaluation); err != nil {
			return err
		}
		query, err := url.ParseQuery(evaluation.Request.Query)
		if err != nil || len(query) > 1 {
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
		if len(query) == 1 {
			values, exists := query["includeRevoked"]
			if !exists || len(values) != 1 || (values[0] != "true" && values[0] != "false") {
				return routes.ErrCoreGuardEvaluatorUnavailable
			}
		}
		return nil
	case "core.route.identity.create_apitoken":
		if err := requireCookieCredentialAuthority(ctx, evaluation); err != nil {
			return err
		}
		var input createAPITokenGuardInput
		if err := decodeGuardJSON(evaluation.Request.Body, &input); err != nil {
			return routes.ErrCoreGuardEvaluatorUnavailable
		}
		for _, scope := range input.Scopes {
			if strings.TrimSpace(scope) == "" {
				return routes.ErrCoreGuardEvaluatorUnavailable
			}
		}
		return nil
	default:
		// 单会话撤销依赖 sid 所有权；PAT 管理还要求真实 cookie 会话。
		return routes.ErrCoreGuardEvaluatorUnavailable
	}
}

type createAPITokenGuardInput struct {
	Name      string   `json:"name"`
	Scopes    []string `json:"scopes"`
	ExpiresAt *string  `json:"expiresAt"`
}

func requireCookieCredentialAuthority(ctx context.Context, evaluation routes.CoreGuardEvaluation) error {
	if err := requireAuthenticatedCoreGuardActor(ctx, evaluation); err != nil {
		return err
	}
	if evaluation.Request.CredentialSource != routes.DispatchCredentialCookie {
		return routes.ErrCoreGuardPermissionDenied
	}
	return nil
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

func requireAllCoreGuardPermissions(evaluation routes.CoreGuardEvaluation, permissions ...string) error {
	if !evaluation.Request.Authenticated || evaluation.Request.ActorID <= 0 {
		return routes.ErrCoreGuardLoginRequired
	}
	if evaluation.Request.Permissions["*"] {
		return nil
	}
	for _, permission := range permissions {
		if permission == "" || !evaluation.Request.Permissions[permission] {
			return routes.ErrCoreGuardPermissionDenied
		}
	}
	return nil
}
