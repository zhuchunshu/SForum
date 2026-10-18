package identity

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

// CoreSafeUserFields are the Host-owned non-secret user projections always
// available through GetUserSafe without extension field authority. All of
// them are public-by-design: username/display/status render on posts, the
// avatar URL is the public profile avatar, groups are the public group
// badges, and the topic/comment counts are aggregates of public content.
var CoreSafeUserFields = []string{"id", "username", "displayName", "status", "avatar", "groups", "topicCount", "commentCount"}

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
		"avatar":      user.Avatar.URL,
		"groups":      roleKeysOrEmpty(user.RoleKeys),
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
			} else if fieldID == "topicCount" || fieldID == "commentCount" {
				// 公开内容计数由宿主适配器按需查询后合并；未补齐时省略，
				// 避免把「未查询」伪装成真实的 0。
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
	case "id", "username", "displayName", "status", "avatar", "groups", "topicCount", "commentCount":
		return true
	default:
		return false
	}
}

// roleKeysOrEmpty 保证组标识列表序列化为数组而非 null。
//
// 返回值必须是 []any 而不是 []string：GetUserSafe 的文档经 structpb.NewStruct
// 编码，protobuf 的 Value 只接受 []interface{}，直接放 []string 会让整个
// 身份投影失败（host.identity_encode_failed: proto: invalid type: []string），
// 于是 userinfo 连 username 都读不到。
func roleKeysOrEmpty(roleKeys []string) []any {
	out := make([]any, 0, len(roleKeys))
	for _, key := range roleKeys {
		out = append(out, key)
	}
	return out
}
