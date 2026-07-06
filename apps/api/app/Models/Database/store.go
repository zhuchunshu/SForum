package database

import "context"

type Store interface {
	ListTables(ctx context.Context) ([]TableSummary, error)
	TableDetail(ctx context.Context, ref TableRef) (TableDetail, error)
	TableRows(ctx context.Context, ref TableRef, detail TableDetail, input RowsInput) ([]map[string]any, bool, error)
	RevealCell(ctx context.Context, ref TableRef, detail TableDetail, input RevealInput) (any, error)
}
