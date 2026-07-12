package entitymeta

import (
	"encoding/json"
	"errors"
	"time"
)

const (
	EntityUser  = "user"
	EntityTopic = "topic"

	ValueString  = "string"
	ValueText    = "text"
	ValueNumber  = "number"
	ValueBoolean = "boolean"

	VisibilityPublic = "public"
	VisibilityOwner  = "owner"
	VisibilityAdmin  = "admin"
)

var (
	ErrNotFound       = errors.New("entity_meta: not found")
	ErrInvalid        = errors.New("entity_meta: invalid")
	ErrFieldDisabled  = errors.New("entity_meta: field disabled")
	ErrPermission     = errors.New("entity_meta: permission denied")
	ErrEntityNotFound = errors.New("entity_meta: entity not found")
)

// FieldDefinition 是宿主拥有的字段目录项。
type FieldDefinition struct {
	ID                 int64           `json:"id"`
	FieldKey           string          `json:"fieldKey"`
	EntityType         string          `json:"entityType"`
	ValueType          string          `json:"valueType"`
	Visibility         string          `json:"visibility"`
	Label              map[string]string `json:"label"`
	Description        map[string]string `json:"description,omitempty"`
	OwnerExtensionID   string          `json:"ownerExtensionId,omitempty"`
	Required           bool            `json:"required"`
	Enabled            bool            `json:"enabled"`
	SortOrder          int             `json:"sortOrder"`
	Constraints        json.RawMessage `json:"constraints,omitempty"`
	CreatedAt          time.Time       `json:"createdAt"`
	UpdatedAt          time.Time       `json:"updatedAt"`
}

// MetaValue 是对某实体某字段的当前值（已按可见性过滤后返回）。
type MetaValue struct {
	FieldKey   string `json:"fieldKey"`
	EntityType string `json:"entityType"`
	EntityID   int64  `json:"entityId"`
	ValueType  string `json:"valueType"`
	// Value 为已解析的 JSON 兼容值（string / number / bool）。
	Value      any               `json:"value"`
	Visibility string            `json:"visibility"`
	Label      map[string]string `json:"label,omitempty"`
	UpdatedAt  time.Time         `json:"updatedAt,omitempty"`
}

type CreateFieldInput struct {
	FieldKey         string
	EntityType       string
	ValueType        string
	Visibility       string
	LabelZHCN        string
	LabelENUS        string
	DescriptionZHCN  string
	DescriptionENUS  string
	OwnerExtensionID string
	Required         bool
	Enabled          *bool
	SortOrder        *int
	Constraints      json.RawMessage
}

type UpdateFieldInput struct {
	Visibility       *string
	LabelZHCN        *string
	LabelENUS        *string
	DescriptionZHCN  *string
	DescriptionENUS  *string
	OwnerExtensionID *string
	Required         *bool
	Enabled          *bool
	SortOrder        *int
	Constraints      *json.RawMessage
}

type UpsertValueInput struct {
	FieldKey string
	// Value 接受 string / float64 / bool / nil（清除）。
	Value any
}

// fieldRow 是 store 层原始行。
type fieldRow struct {
	ID                 int64
	FieldKey           string
	EntityType         string
	ValueType          string
	Visibility         string
	LabelZHCN          string
	LabelENUS          string
	DescriptionZHCN    string
	DescriptionENUS    string
	OwnerExtensionID   string
	Required           bool
	Enabled            bool
	SortOrder          int
	Constraints        []byte
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type valueRow struct {
	EntityType       string
	EntityID         int64
	FieldKey         string
	ValueText        string
	UpdatedAt        time.Time
	UpdatedByUserID  *int64
}
