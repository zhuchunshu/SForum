package identity

func (a Actor) IsActive() bool {
	return a.Status == UserStatusActive
}

func (a Actor) IsSuperAdmin() bool {
	if !a.IsActive() {
		return false
	}
	for _, roleKey := range a.RoleKeys {
		if roleKey == RoleSuperAdmin {
			return true
		}
	}
	return false
}

func (a Actor) Can(permission string) bool {
	if !a.IsActive() {
		return false
	}
	if a.IsSuperAdmin() {
		return true
	}
	if a.Permissions == nil {
		return false
	}
	return a.Permissions[permission]
}

func CanEditPost(actor Actor, post PostSummary) bool {
	if actor.Can(PermissionPostEditAny) {
		return true
	}
	return post.AuthorUserID == actor.ID && actor.Can(PermissionPostEditOwn)
}
