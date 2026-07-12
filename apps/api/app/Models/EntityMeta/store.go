package entitymeta

import "context"

// Store 持久化字段定义与元数据值。
type Store interface {
	ListDefinitions(ctx context.Context, entityType string) ([]fieldRow, error)
	GetDefinitionByKey(ctx context.Context, fieldKey string) (fieldRow, error)
	CreateDefinition(ctx context.Context, row fieldRow) (fieldRow, error)
	UpdateDefinition(ctx context.Context, fieldKey string, row fieldRow) (fieldRow, error)
	DeleteDefinition(ctx context.Context, fieldKey string) error

	ListValues(ctx context.Context, entityType string, entityID int64) ([]valueRow, error)
	UpsertValue(ctx context.Context, row valueRow) (valueRow, error)
	DeleteValue(ctx context.Context, entityType string, entityID int64, fieldKey string) error

	// EntityExists 校验目标实体是否存在（user / topic）。
	EntityExists(ctx context.Context, entityType string, entityID int64) (bool, error)
	// TopicAuthorID 返回主题作者；不存在时 ok=false。
	TopicAuthorID(ctx context.Context, topicID int64) (authorID int64, ok bool, err error)
}
