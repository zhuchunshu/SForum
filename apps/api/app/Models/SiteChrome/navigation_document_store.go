package sitechrome

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// NavigationDocumentStore is intentionally separate from the legacy SiteChrome
// CRUD store. M1 reads the canonical document; M2 will add its transaction
// owning command boundary without widening the old compatibility interface.
type NavigationDocumentStore interface {
	ReadNavigationDocument(ctx context.Context) (NavigationDocument, error)
}

// NavigationCommandStore owns only the navigation tables and transaction
// lifetime. Domain policy, audit, and public-surface revision effects remain
// with Service and execute through the supplied transaction.
type NavigationCommandStore interface {
	NavigationDocumentStore
	ExecuteNavigationTransaction(ctx context.Context, expectedRevision uint64, mutation func(context.Context, pgx.Tx, NavigationDocument) (NavigationTransactionResult, error)) (NavigationDocument, error)
	ListNavigationSnapshots(ctx context.Context) ([]NavigationSnapshot, error)
	GetNavigationSnapshot(ctx context.Context, id int64) (NavigationSnapshot, error)
}

type NavigationTransactionResult struct {
	Document          NavigationDocument
	ActorUserID       int64
	Operation         string
	Reason            string
	AffectedLocations []string
}
