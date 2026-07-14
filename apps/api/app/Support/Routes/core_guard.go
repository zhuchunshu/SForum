package routes

import (
	"fmt"
	"strings"
)

type CoreGuardKind string

const (
	CoreGuardPublic        CoreGuardKind = "public"
	CoreGuardLogin         CoreGuardKind = "login"
	CoreGuardGuest         CoreGuardKind = "guest"
	CoreGuardSuperAdmin    CoreGuardKind = "super_admin"
	CoreGuardPermissionAny CoreGuardKind = "permission_any"
	CoreGuardContextual    CoreGuardKind = "contextual"
)

// CoreGuardDescriptor is reviewed build-time policy data. Runtime code must
// never derive authorization behavior from the human-readable catalog policy.
type CoreGuardDescriptor struct {
	RouteID         string
	ContractVersion string
	Method          string
	Kind            CoreGuardKind
	Permissions     []string
	EvaluatorID     string
}

func cloneCoreGuardDescriptor(value CoreGuardDescriptor) CoreGuardDescriptor {
	value.Permissions = append([]string(nil), value.Permissions...)
	return value
}

func equalCoreGuardDescriptor(left, right CoreGuardDescriptor) bool {
	if left.RouteID != right.RouteID || left.ContractVersion != right.ContractVersion ||
		left.Method != right.Method || left.Kind != right.Kind || left.EvaluatorID != right.EvaluatorID ||
		len(left.Permissions) != len(right.Permissions) {
		return false
	}
	for index := range left.Permissions {
		if left.Permissions[index] != right.Permissions[index] {
			return false
		}
	}
	return true
}

func validateCoreGuardDescriptor(route CoreRoute) error {
	descriptor := route.Guard
	if descriptor.RouteID != route.ID || descriptor.ContractVersion != route.ContractVersion || descriptor.Method != route.Method {
		return fmt.Errorf("%w: core guard identity does not match route", ErrInvalidRoute)
	}
	seen := make(map[string]struct{}, len(descriptor.Permissions))
	for _, permission := range descriptor.Permissions {
		if permission == "" || permission != strings.TrimSpace(permission) || !routeIDPattern.MatchString(permission) {
			return fmt.Errorf("%w: invalid core guard permission", ErrInvalidRoute)
		}
		if _, duplicate := seen[permission]; duplicate {
			return fmt.Errorf("%w: duplicate core guard permission", ErrInvalidRoute)
		}
		seen[permission] = struct{}{}
	}
	switch descriptor.Kind {
	case CoreGuardPublic, CoreGuardLogin, CoreGuardGuest, CoreGuardSuperAdmin:
		if len(descriptor.Permissions) != 0 || descriptor.EvaluatorID != "" {
			return fmt.Errorf("%w: metadata core guard has contextual fields", ErrInvalidRoute)
		}
	case CoreGuardPermissionAny:
		if len(descriptor.Permissions) == 0 || descriptor.EvaluatorID != "" {
			return fmt.Errorf("%w: permission core guard is incomplete", ErrInvalidRoute)
		}
	case CoreGuardContextual:
		if !strings.HasPrefix(descriptor.EvaluatorID, "core.guard.") || !routeIDPattern.MatchString(descriptor.EvaluatorID) {
			return fmt.Errorf("%w: contextual core guard evaluator is invalid", ErrInvalidRoute)
		}
	default:
		return fmt.Errorf("%w: unknown core guard kind", ErrInvalidRoute)
	}
	return nil
}

func coreGuardDescriptorIsZero(value CoreGuardDescriptor) bool {
	return value.RouteID == "" && value.ContractVersion == "" && value.Method == "" &&
		value.Kind == "" && len(value.Permissions) == 0 && value.EvaluatorID == ""
}
