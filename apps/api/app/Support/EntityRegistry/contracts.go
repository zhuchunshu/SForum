package entityregistry

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// ActorPermissions is the set of permission keys the caller currently holds.
// Host RBAC resolves keys; this package only checks declared keys against the
// supplied set (fail-closed on empty).
type ActorPermissions map[string]struct{}

// NewActorPermissions builds a fail-closed permission set from raw keys.
func NewActorPermissions(keys ...string) ActorPermissions {
	result := make(ActorPermissions, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		result[key] = struct{}{}
	}
	return result
}

func (p ActorPermissions) has(key string) bool {
	if len(p) == 0 || key == "" {
		return false
	}
	_, ok := p[key]
	return ok
}

// EvaluatePermission checks one Host action against a frozen contribution.
// actor may be nil/empty (always deny). fieldID is required for field actions.
func (r *Registry) EvaluatePermission(
	action string,
	targetID string,
	fieldID string,
	actor ActorPermissions,
) (PermissionDecision, error) {
	if r == nil {
		return PermissionDecision{}, ErrInvalid
	}
	action = strings.ToLower(strings.TrimSpace(action))
	targetID = strings.ToLower(strings.TrimSpace(targetID))
	fieldID = strings.ToLower(strings.TrimSpace(fieldID))
	if action == "" || targetID == "" {
		return PermissionDecision{}, ErrInvalid
	}
	contribution, err := r.Resolve(targetID)
	if err != nil {
		return PermissionDecision{}, err
	}
	decision := PermissionDecision{Action: action, TargetID: targetID}
	var permissionKey string
	var policy string
	switch contribution.Kind {
	case KindEntity:
		permissionKey, policy, err = entityActionPermission(contribution, action)
	case KindTaxonomy:
		permissionKey, err = taxonomyActionPermission(contribution, action)
	case KindField:
		return PermissionDecision{}, fmt.Errorf("%w: use field actions against entity target with fieldId", ErrInvalid)
	default:
		return PermissionDecision{}, ErrInvalid
	}
	if err != nil {
		return PermissionDecision{}, err
	}
	// Field-scoped actions re-target through the entity and a field declaration.
	if action == ActionReadField || action == ActionWriteField {
		if fieldID == "" {
			return PermissionDecision{}, ErrInvalid
		}
		field, fieldErr := r.Resolve(fieldID)
		if fieldErr != nil {
			return PermissionDecision{}, fieldErr
		}
		if field.Kind != KindField || field.EntityID != targetID {
			return PermissionDecision{}, ErrInvalid
		}
		permissionKey, err = fieldActionPermission(field, action)
		if err != nil {
			return PermissionDecision{}, err
		}
		decision.TargetID = fieldID
	}
	decision.PermissionKey = permissionKey
	decision.Policy = policy
	// Policy is Host-final and evaluated before permission keys so export-only
	// / deny / retain surfaces fail closed even when the permission key is empty.
	if action == ActionImport || action == ActionExport {
		if action == ActionImport && !policyAllowsImport(contribution.ImportExportPolicy) {
			decision.Allowed = false
			decision.Reason = "import_policy_denied"
			decision.Policy = contribution.ImportExportPolicy
			return decision, ErrPolicyDenied
		}
		if action == ActionExport && !policyAllowsExport(contribution.ImportExportPolicy) {
			decision.Allowed = false
			decision.Reason = "export_policy_denied"
			decision.Policy = contribution.ImportExportPolicy
			return decision, ErrPolicyDenied
		}
	}
	if action == ActionDelete && contribution.Kind == KindEntity &&
		contribution.DeletionPolicy == DeletionRetain {
		decision.Allowed = false
		decision.Reason = "deletion_retain_policy"
		decision.Policy = contribution.DeletionPolicy
		return decision, ErrPolicyDenied
	}
	if permissionKey == "" || !actor.has(permissionKey) {
		decision.Allowed = false
		decision.Reason = "permission_denied"
		return decision, ErrPermissionDenied
	}
	decision.Allowed = true
	return decision, nil
}

func entityActionPermission(contribution Contribution, action string) (string, string, error) {
	switch action {
	case ActionCreate:
		return contribution.PermissionCreate, "", nil
	case ActionRead:
		return contribution.PermissionRead, "", nil
	case ActionUpdate:
		return contribution.PermissionUpdate, "", nil
	case ActionDelete:
		return contribution.PermissionDelete, contribution.DeletionPolicy, nil
	case ActionImport:
		return contribution.PermissionImport, contribution.ImportExportPolicy, nil
	case ActionExport:
		return contribution.PermissionExport, contribution.ImportExportPolicy, nil
	case ActionReadField, ActionWriteField:
		// Permission key comes from the field declaration.
		return "", "", nil
	default:
		return "", "", ErrInvalid
	}
}

func taxonomyActionPermission(contribution Contribution, action string) (string, error) {
	switch action {
	case ActionManageTerms:
		return contribution.PermissionManage, nil
	case ActionAssignTerms:
		return contribution.PermissionAssign, nil
	default:
		return "", ErrInvalid
	}
}

func fieldActionPermission(contribution Contribution, action string) (string, error) {
	switch action {
	case ActionReadField:
		return contribution.PermissionFieldRead, nil
	case ActionWriteField:
		return contribution.PermissionFieldWrite, nil
	default:
		return "", ErrInvalid
	}
}

// IndexPlanForEntity projects search/index fields for one entity type.
func (r *Registry) IndexPlanForEntity(entityID string) (IndexPlan, error) {
	if r == nil {
		return IndexPlan{}, ErrInvalid
	}
	entity, err := r.Resolve(entityID)
	if err != nil {
		return IndexPlan{}, err
	}
	if entity.Kind != KindEntity {
		return IndexPlan{}, ErrInvalid
	}
	plan := IndexPlan{
		EntityID:   entity.ID,
		StorageKey: entity.StorageKey,
	}
	for _, field := range r.ListFieldsForEntity(entity.ID) {
		if !field.Indexed || field.IndexKind == IndexNone {
			continue
		}
		plan.Fields = append(plan.Fields, IndexFieldPlan{
			FieldID:   field.ID,
			IndexKind: field.IndexKind,
			Required:  field.Required,
			Schema:    field.Schema,
		})
	}
	sort.Slice(plan.Fields, func(i, j int) bool {
		return plan.Fields[i].FieldID < plan.Fields[j].FieldID
	})
	return plan, nil
}

// ImportExportPlanForEntity projects Host import/export contract for one entity.
func (r *Registry) ImportExportPlanForEntity(entityID string) (ImportExportPlan, error) {
	if r == nil {
		return ImportExportPlan{}, ErrInvalid
	}
	entity, err := r.Resolve(entityID)
	if err != nil {
		return ImportExportPlan{}, err
	}
	if entity.Kind != KindEntity {
		return ImportExportPlan{}, ErrInvalid
	}
	plan := ImportExportPlan{
		EntityID:  entity.ID,
		Policy:    entity.ImportExportPolicy,
		CanImport: policyAllowsImport(entity.ImportExportPolicy),
		CanExport: policyAllowsExport(entity.ImportExportPolicy),
	}
	for _, field := range r.ListFieldsForEntity(entity.ID) {
		plan.FieldIDs = append(plan.FieldIDs, field.ID)
	}
	sort.Strings(plan.FieldIDs)
	plan.TaxonomyIDs = append([]string(nil), entity.TaxonomyIDs...)
	return plan, nil
}

// ImportExportDryRunSchemaVersion is the Host dry-run contract (no store I/O).
const ImportExportDryRunSchemaVersion = "sforum.entity-import-export-dry-run@1"

// ImportExportDryRun is a non-executing projection of import/export admission.
// Executes is always false; Host handlers must not treat this as a receipt.
type ImportExportDryRun struct {
	SchemaVersion string             `json:"schemaVersion"`
	DryRun        bool               `json:"dryRun"`
	Executes      bool               `json:"executes"`
	Action        string             `json:"action"`
	Plan          ImportExportPlan   `json:"plan"`
	Decision      PermissionDecision `json:"decision"`
}

// DryRunImportExport returns plan + permission decision without durable I/O.
// Permission and policy denials are returned inside Decision with nil error so
// admin/devtools can inspect Allowed/Reason; only invalid inputs error.
func (r *Registry) DryRunImportExport(
	entityID string,
	action string,
	actor ActorPermissions,
) (ImportExportDryRun, error) {
	if r == nil {
		return ImportExportDryRun{}, ErrInvalid
	}
	action = strings.ToLower(strings.TrimSpace(action))
	if action != ActionImport && action != ActionExport {
		return ImportExportDryRun{}, ErrInvalid
	}
	plan, err := r.ImportExportPlanForEntity(entityID)
	if err != nil {
		return ImportExportDryRun{}, err
	}
	decision, evalErr := r.EvaluatePermission(action, entityID, "", actor)
	result := ImportExportDryRun{
		SchemaVersion: ImportExportDryRunSchemaVersion,
		DryRun:        true,
		Executes:      false,
		Action:        action,
		Plan:          plan,
		Decision:      decision,
	}
	if evalErr != nil &&
		!errors.Is(evalErr, ErrPermissionDenied) &&
		!errors.Is(evalErr, ErrPolicyDenied) {
		return result, evalErr
	}
	return result, nil
}

// DeletionPlanForEntity projects Host deletion contract for one entity type.
func (r *Registry) DeletionPlanForEntity(entityID string) (DeletionPlan, error) {
	if r == nil {
		return DeletionPlan{}, ErrInvalid
	}
	entity, err := r.Resolve(entityID)
	if err != nil {
		return DeletionPlan{}, err
	}
	if entity.Kind != KindEntity {
		return DeletionPlan{}, ErrInvalid
	}
	plan := DeletionPlan{
		EntityID: entity.ID,
		Policy:   entity.DeletionPolicy,
	}
	switch entity.DeletionPolicy {
	case DeletionSoft:
		plan.SoftDelete = true
	case DeletionHard:
		plan.HardDelete = true
	case DeletionRetain:
		plan.Retain = true
	}
	return plan, nil
}

// AllowPermission is a convenience that returns nil on allow and a typed error
// on deny. Host handlers should prefer EvaluatePermission for audit metadata.
func (r *Registry) AllowPermission(
	action string,
	targetID string,
	fieldID string,
	actor ActorPermissions,
) error {
	_, err := r.EvaluatePermission(action, targetID, fieldID, actor)
	return err
}
