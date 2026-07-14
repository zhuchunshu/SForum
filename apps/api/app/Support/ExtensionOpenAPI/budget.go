package extensionopenapi

import "fmt"

const (
	maxManifestBytes       int64 = 2 << 20
	maxDocumentBytes       int64 = 4 << 20
	maxAggregateInputBytes int64 = 32 << 20
	maxAggregateBytes            = 64 << 20
	maxDocuments                 = 128
	maxReferences                = 4096
	maxDocumentDepth             = 128
)

type resourceBudget struct {
	documents  int
	references int
	bytes      int64
}

func (b *resourceBudget) reserveDocument(size int64) error {
	if size < 0 || size > maxDocumentBytes {
		return fmt.Errorf("%w: document size %d exceeds %d", ErrResourceBudget, size, maxDocumentBytes)
	}
	if b.documents+1 > maxDocuments {
		return fmt.Errorf("%w: document count exceeds %d", ErrResourceBudget, maxDocuments)
	}
	if b.bytes+size > maxAggregateInputBytes {
		return fmt.Errorf("%w: aggregate input exceeds %d bytes", ErrResourceBudget, maxAggregateInputBytes)
	}
	b.documents++
	b.bytes += size
	return nil
}

func (b *resourceBudget) reserveReferences(count int) error {
	if count < 0 || b.references+count > maxReferences {
		return fmt.Errorf("%w: reference count exceeds %d", ErrResourceBudget, maxReferences)
	}
	b.references += count
	return nil
}
