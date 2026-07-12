package attachments

import (
	"context"
	"time"
)

type Store interface {
	Create(ctx context.Context, input CreateAttachmentInput) (Attachment, error)
	GetByPublicID(ctx context.Context, publicID string) (Attachment, error)
	GetByID(ctx context.Context, id int64) (Attachment, error)
	List(ctx context.Context, input AttachmentListInput) (AttachmentList, error)
	ListReferences(ctx context.Context, attachmentID int64) ([]AttachmentReference, error)
	ListReferenceAccess(ctx context.Context, attachmentID int64) ([]ReferenceAccess, error)
	UpdateStatus(ctx context.Context, id int64, status string, deleted bool) (Attachment, error)
	ListCleanupCandidates(ctx context.Context, cutoff time.Time, limit int) ([]Attachment, error)
	DeleteMetadata(ctx context.Context, id int64) error
}
