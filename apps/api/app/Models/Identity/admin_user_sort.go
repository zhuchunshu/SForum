package identity

import (
	"context"
	"strings"
)

const (
	UserListSortCreatedAt   = "createdAt"
	UserListSortUpdatedAt   = "updatedAt"
	UserListSortUsername    = "username"
	UserListSortDisplayName = "displayName"
	UserListSortEmail       = "email"
	UserListSortStatus      = "status"
	UserListSortOrderAsc    = "asc"
	UserListSortOrderDesc   = "desc"
)

func (s *Service) ListUsers(ctx context.Context, actor Actor, input UserListInput) (AdminUserList, error) {
	// 只读列表：user.view；user.manage 父权限通过兼容层也可通过。
	if !actor.Can(PermissionUserView) {
		return AdminUserList{}, ErrPermissionDenied
	}
	input.Page, input.PerPage = normalizePage(input.Page, input.PerPage)
	input.Query = escapeLike(strings.TrimSpace(input.Query))
	input.Status = strings.TrimSpace(input.Status)
	input.RoleKey = strings.TrimSpace(input.RoleKey)
	input.SortBy, input.SortOrder = normalizeUserListSorting(input.SortBy, input.SortOrder)
	return s.store.ListUsers(ctx, input)
}

func normalizeUserListSorting(sortBy, sortOrder string) (string, string) {
	switch strings.TrimSpace(sortBy) {
	case UserListSortUpdatedAt, UserListSortUsername, UserListSortDisplayName, UserListSortEmail, UserListSortStatus:
		sortBy = strings.TrimSpace(sortBy)
	default:
		sortBy = UserListSortCreatedAt
	}

	if strings.EqualFold(strings.TrimSpace(sortOrder), UserListSortOrderAsc) {
		sortOrder = UserListSortOrderAsc
	} else {
		sortOrder = UserListSortOrderDesc
	}
	return sortBy, sortOrder
}
