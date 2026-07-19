package identity

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

// CoreSafeUserFields are the Host-owned non-secret user projections always
// available through GetUserSafe without extension field authority.
var CoreSafeUserFields = []string{"id", "username", "displayName", "status"}

// SafeUserFieldReader is the narrow read dependency for extension user fields.
// Production uses the Host-owned IdentityUserFieldValueStore.
type SafeUserFieldReader interface {
	Get(context.Context, ReadIdentityUserFieldValueInput) (IdentityUserFieldValueRead, error)
}

// ProjectSafeUser builds the Host GetUserSafe document. Core fields never
// consult the extension store. Extension field ids require a live actor, the
// field store, and the store's live readPermission + Schema checks. Actorless
// extension field reads fail closed. Missing extension values are omitted
// rather than invented; permission/schema failures fail the whole projection
// so a partial unauthorized map cannot look successful.
func ProjectSafeUser(
	ctx context.Context,
	user CurrentUser,
	actorUserID int64,
	declaredFields []string,
	fields SafeUserFieldReader,
) (map[string]any, error) {
	if ctx == nil || user.ID <= 0 {
		return nil, ErrUserNotFound
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	core := map[string]any{
		"id":          user.ID,
		"username":    user.Username,
		"displayName": user.DisplayName,
		"status":      string(user.Status),
	}

	requested := normalizeDeclaredSafeUserFields(declaredFields)
	if len(requested) == 0 {
		// Empty projection request keeps the historical full core safe user.
		return core, nil
	}

	result := make(map[string]any, len(requested))
	for _, fieldID := range requested {
		if isCoreSafeUserField(fieldID) {
			if value, ok := core[fieldID]; ok {
				result[fieldID] = value
			}
			continue
		}
		// Extension user-field ids are Host-resolved against the live Registry
		// declaration inside the value store. The declared_fields list is not
		// authority by itself.
		if actorUserID <= 0 {
			return nil, ErrIdentityUserFieldPermissionDenied
		}
		if fields == nil {
			return nil, ErrIdentityUserFieldValueStoreUnavailable
		}
		read, err := fields.Get(ctx, ReadIdentityUserFieldValueInput{
			ActorUserID: actorUserID,
			UserID:      user.ID,
			FieldID:     fieldID,
		})
		if err != nil {
			if errors.Is(err, ErrIdentityUserFieldValueNotFound) {
				// Absent values stay omitted; denial and Schema failures do not.
				continue
			}
			return nil, err
		}
		var decoded any
		if len(read.Value) > 0 {
			if err := json.Unmarshal(read.Value, &decoded); err != nil {
				return nil, ErrIdentityUserFieldSchemaInvalid
			}
		}
		result[fieldID] = decoded
	}
	return result, nil
}

func normalizeDeclaredSafeUserFields(fields []string) []string {
	if len(fields) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(fields))
	result := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		// Core fields keep their published camelCase; extension field ids are
		// lower-case namespaced handles.
		if isCoreSafeUserField(field) {
			if _, ok := seen[field]; ok {
				continue
			}
			seen[field] = struct{}{}
			result = append(result, field)
			continue
		}
		normalized := strings.ToLower(field)
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func isCoreSafeUserField(field string) bool {
	switch field {
	case "id", "username", "displayName", "status":
		return true
	default:
		return false
	}
}
