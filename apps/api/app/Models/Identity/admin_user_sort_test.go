package identity

import "testing"

func TestNormalizeUserListSorting(t *testing.T) {
	tests := []struct {
		name      string
		sortBy    string
		sortOrder string
		wantBy    string
		wantOrder string
	}{
		{name: "defaults", wantBy: UserListSortCreatedAt, wantOrder: UserListSortOrderDesc},
		{name: "accepted", sortBy: UserListSortUsername, sortOrder: "ASC", wantBy: UserListSortUsername, wantOrder: UserListSortOrderAsc},
		{name: "trimmed", sortBy: " updatedAt ", sortOrder: " asc ", wantBy: UserListSortUpdatedAt, wantOrder: UserListSortOrderAsc},
		{name: "invalid", sortBy: "created_at; DROP TABLE users", sortOrder: "sideways", wantBy: UserListSortCreatedAt, wantOrder: UserListSortOrderDesc},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBy, gotOrder := normalizeUserListSorting(tt.sortBy, tt.sortOrder)
			if gotBy != tt.wantBy || gotOrder != tt.wantOrder {
				t.Fatalf("normalizeUserListSorting() = (%q, %q), want (%q, %q)", gotBy, gotOrder, tt.wantBy, tt.wantOrder)
			}
		})
	}
}

func TestAdminUserOrderByUsesWhitelistedStableOrdering(t *testing.T) {
	tests := []struct {
		name      string
		sortBy    string
		sortOrder string
		want      string
	}{
		{name: "created newest", sortBy: UserListSortCreatedAt, sortOrder: UserListSortOrderDesc, want: "created_at DESC, id DESC"},
		{name: "updated oldest", sortBy: UserListSortUpdatedAt, sortOrder: UserListSortOrderAsc, want: "updated_at ASC, id ASC"},
		{name: "username", sortBy: UserListSortUsername, sortOrder: UserListSortOrderAsc, want: "username_lower ASC, id ASC"},
		{name: "display name", sortBy: UserListSortDisplayName, sortOrder: UserListSortOrderDesc, want: "lower(display_name) DESC, id DESC"},
		{name: "email", sortBy: UserListSortEmail, sortOrder: UserListSortOrderAsc, want: "email_lower ASC, id ASC"},
		{name: "status", sortBy: UserListSortStatus, sortOrder: UserListSortOrderAsc, want: "status ASC, id ASC"},
		{name: "rejects SQL fragments", sortBy: "id; DELETE FROM users", sortOrder: "DESC NULLS FIRST", want: "created_at DESC, id DESC"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := adminUserOrderBy(tt.sortBy, tt.sortOrder); got != tt.want {
				t.Fatalf("adminUserOrderBy() = %q, want %q", got, tt.want)
			}
		})
	}
}
