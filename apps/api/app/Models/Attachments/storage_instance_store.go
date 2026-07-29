package attachments

import "context"

type StorageInstanceStore interface {
	CreateStorageInstance(ctx context.Context, input StorageInstanceCreate) (StorageInstance, error)
	UpdateStorageInstance(ctx context.Context, id string, expectedRevision int64, name string, settings map[string]string) (StorageInstance, error)
	GetStorageInstance(ctx context.Context, id string) (StorageInstance, error)
	ListStorageInstances(ctx context.Context) ([]StorageInstance, error)
	UpdateStorageInstanceProbe(ctx context.Context, id, status, message string) error
	DeleteStorageInstance(ctx context.Context, id string) error
}

type StorageInstanceCreate struct {
	ID              string
	ExtensionID     string
	Name            string
	Settings        map[string]string
	CreatedByUserID int64
}
