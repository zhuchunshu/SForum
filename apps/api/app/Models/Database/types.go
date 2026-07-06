package database

import "errors"

const (
	SensitiveMask = "••••••••"

	CodeInvalid           = "database.invalid"
	CodeTableNotFound     = "database.table_not_found"
	CodeColumnInvalid     = "database.column_invalid"
	CodeRevealUnavailable = "database.reveal_unavailable"
	CodeRowNotFound       = "database.row_not_found"
)

var (
	ErrInvalidTable      = errors.New("database: invalid table")
	ErrTableNotFound     = errors.New("database: table not found")
	ErrInvalidColumn     = errors.New("database: invalid column")
	ErrInvalidFilter     = errors.New("database: invalid filter")
	ErrRevealUnavailable = errors.New("database: reveal unavailable")
	ErrRowNotFound       = errors.New("database: row not found")
)

type TableRef struct {
	Schema string
	Name   string
}

type TableSummary struct {
	Schema        string `json:"schema"`
	Name          string `json:"name"`
	Kind          string `json:"kind"`
	EstimatedRows int64  `json:"estimatedRows"`
	SizeBytes     int64  `json:"sizeBytes"`
	Comment       string `json:"comment"`
}

type Column struct {
	Name         string `json:"name"`
	DataType     string `json:"dataType"`
	Nullable     bool   `json:"nullable"`
	DefaultValue string `json:"defaultValue"`
	IsPrimaryKey bool   `json:"isPrimaryKey"`
	IsSensitive  bool   `json:"isSensitive"`
}

type Index struct {
	Name       string `json:"name"`
	Unique     bool   `json:"unique"`
	Primary    bool   `json:"primary"`
	Definition string `json:"definition"`
}

type Constraint struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Definition string `json:"definition"`
}

type TableDetail struct {
	Schema      string       `json:"schema"`
	Name        string       `json:"name"`
	Kind        string       `json:"kind"`
	Columns     []Column     `json:"columns"`
	PrimaryKey  []string     `json:"primaryKey"`
	Indexes     []Index      `json:"indexes"`
	Constraints []Constraint `json:"constraints"`
}

type RowsInput struct {
	Page           int
	PerPage        int
	Sort           string
	Direction      string
	FilterColumn   string
	FilterOperator string
	FilterValue    string
}

type RowsResult struct {
	Columns []Column   `json:"columns"`
	Rows    []TableRow `json:"rows"`
	Page    int        `json:"page"`
	PerPage int        `json:"perPage"`
	HasNext bool       `json:"hasNext"`
}

type TableRow struct {
	RowKey string               `json:"rowKey,omitempty"`
	Values map[string]CellValue `json:"values"`
}

type CellValue struct {
	Value     any  `json:"value"`
	Sensitive bool `json:"sensitive"`
	Masked    bool `json:"masked"`
}

type RevealInput struct {
	RowKey       string
	Column       string
	RowKeyValues map[string]string
}

type RevealResult struct {
	Schema string `json:"schema"`
	Table  string `json:"table"`
	Column string `json:"column"`
	Value  any    `json:"value"`
}
