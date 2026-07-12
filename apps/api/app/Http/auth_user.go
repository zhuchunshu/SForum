package http

import (
	"github.com/gofiber/fiber/v3"

	apitokens "github.com/zhuchunshu/sforum/apps/api/app/Models/APITokens"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
)

// ResolveUserID 解析当前请求用户：Bearer PAT 优先，否则 cookie session。
func ResolveUserID(c fiber.Ctx, sessions *authsession.Manager) (int64, bool, error) {
	if userID, ok := apitokens.UserIDFromContext(c.Context()); ok {
		return userID, true, nil
	}
	if sessions == nil {
		return 0, false, nil
	}
	return sessions.CurrentUserID(c)
}

// LoadActor 加载 Actor，并在 PAT 请求上按 scopes 收窄权限。
func LoadActor(c fiber.Ctx, sessions *authsession.Manager, users identity.ActorStore) (identity.Actor, error) {
	userID, ok, err := ResolveUserID(c, sessions)
	if err != nil {
		return identity.Actor{}, err
	}
	if !ok {
		return identity.Actor{}, fiber.NewError(fiber.StatusUnauthorized, "auth.required")
	}
	actor, err := users.LoadActor(c.Context(), userID)
	if err != nil {
		return identity.Actor{}, err
	}
	if scopes := apitokens.ScopesFromContext(c.Context()); len(scopes) > 0 {
		actor = apitokens.RestrictActor(actor, scopes)
	}
	return actor, nil
}
