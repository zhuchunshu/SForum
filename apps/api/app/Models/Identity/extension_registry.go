package identity

import (
	"context"
	"errors"

	identityregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/IdentityRegistry"
)

var ErrIdentityRegistryUnavailable = errors.New("identity: identity registry is unavailable")

// RoleSuggestionDecisionInput intentionally omits actor and role-permission
// mutation fields. The Host binds the authenticated actor before recording
// review evidence; assigning permissions remains a separate Host-owned action.
type RoleSuggestionDecisionInput struct {
	ID               int64
	ExpectedRevision int64
	ApprovalState    string
}

// WithIdentityRegistryStore injects the durable Identity Registry repository.
// Bootstrap must opt in explicitly; an unconfigured service fails closed.
func (s *Service) WithIdentityRegistryStore(store identityregistry.Store) *Service {
	if s == nil {
		return nil
	}
	s.identityRegistry = store
	return s
}

func (s *Service) ListRoleSuggestions(
	ctx context.Context,
	actor Actor,
	filter identityregistry.RoleSuggestionFilter,
) ([]identityregistry.RoleSuggestion, error) {
	if !actor.IsActive() || !actor.Can(PermissionRoleManage) {
		return nil, ErrPermissionDenied
	}
	if s == nil || s.identityRegistry == nil {
		return nil, ErrIdentityRegistryUnavailable
	}

	// 规范化与条数上限由 repository 统一执行，避免 Host 与持久化层漂移。
	return s.identityRegistry.ListRoleSuggestions(ctx, filter)
}

func (s *Service) DecideRoleSuggestion(
	ctx context.Context,
	actor Actor,
	input RoleSuggestionDecisionInput,
) (identityregistry.RoleSuggestion, error) {
	if !actor.IsActive() || !actor.Can(PermissionRoleManage) {
		return identityregistry.RoleSuggestion{}, ErrPermissionDenied
	}
	if s == nil || s.identityRegistry == nil {
		return identityregistry.RoleSuggestion{}, ErrIdentityRegistryUnavailable
	}

	// ActorUserID 只能来自已授权的 Host actor，调用者不能通过 input 伪造。
	return s.identityRegistry.DecideRoleSuggestion(ctx, identityregistry.DecideRoleSuggestionInput{
		ID:               input.ID,
		ExpectedRevision: input.ExpectedRevision,
		ApprovalState:    input.ApprovalState,
		ActorUserID:      actor.ID,
	})
}
