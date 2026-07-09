package database

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) ListTables(ctx context.Context) ([]TableSummary, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			n.nspname,
			c.relname,
			CASE c.relkind
				WHEN 'r' THEN 'table'
				WHEN 'p' THEN 'partitioned_table'
				WHEN 'v' THEN 'view'
				WHEN 'm' THEN 'materialized_view'
				ELSE c.relkind::text
			END AS kind,
			GREATEST(c.reltuples::bigint, 0),
			pg_total_relation_size(c.oid),
			COALESCE(obj_description(c.oid, 'pg_class'), '')
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind IN ('r', 'p', 'v', 'm')
		  AND n.nspname NOT IN ('pg_catalog', 'information_schema')
		  AND n.nspname NOT LIKE 'pg_toast%'
		ORDER BY n.nspname, c.relname
	`)
	if err != nil {
		return nil, fmt.Errorf("list database tables: %w", err)
	}
	defer rows.Close()

	tables := []TableSummary{}
	for rows.Next() {
		var table TableSummary
		if err := rows.Scan(&table.Schema, &table.Name, &table.Kind, &table.EstimatedRows, &table.SizeBytes, &table.Comment); err != nil {
			return nil, fmt.Errorf("scan database table: %w", err)
		}
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate database tables: %w", err)
	}
	return tables, nil
}

func (s *PostgresStore) TableDetail(ctx context.Context, ref TableRef) (TableDetail, error) {
	oid, kind, err := s.tableOID(ctx, ref)
	if err != nil {
		return TableDetail{}, err
	}

	columns, err := s.columns(ctx, oid)
	if err != nil {
		return TableDetail{}, err
	}
	primaryKey, err := s.primaryKey(ctx, oid)
	if err != nil {
		return TableDetail{}, err
	}
	indexes, err := s.indexes(ctx, oid)
	if err != nil {
		return TableDetail{}, err
	}
	constraints, err := s.constraints(ctx, oid)
	if err != nil {
		return TableDetail{}, err
	}

	return TableDetail{
		Schema:      ref.Schema,
		Name:        ref.Name,
		Kind:        kind,
		Columns:     columns,
		PrimaryKey:  primaryKey,
		Indexes:     indexes,
		Constraints: constraints,
	}, nil
}

func (s *PostgresStore) TableRows(ctx context.Context, ref TableRef, detail TableDetail, input RowsInput) ([]map[string]any, bool, error) {
	if len(detail.Columns) == 0 {
		return []map[string]any{}, false, nil
	}

	args := []any{}
	query := strings.Builder{}
	query.WriteString("SELECT ")
	query.WriteString(columnList(detail.Columns))
	query.WriteString(" FROM ")
	query.WriteString(pgx.Identifier{ref.Schema, ref.Name}.Sanitize())
	if where := filterClause(input, &args); where != "" {
		query.WriteString(" WHERE ")
		query.WriteString(where)
	}
	query.WriteString(orderClause(input, detail))
	args = append(args, input.PerPage+1, (input.Page-1)*input.PerPage)
	query.WriteString(fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args)))

	rows, err := s.pool.Query(ctx, query.String(), args...)
	if err != nil {
		return nil, false, fmt.Errorf("query database table rows: %w", err)
	}
	defer rows.Close()

	items := []map[string]any{}
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, false, fmt.Errorf("read database table row: %w", err)
		}
		item := make(map[string]any, len(detail.Columns))
		for index, column := range detail.Columns {
			item[column.Name] = normalizePostgresValue(values[index])
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("iterate database table rows: %w", err)
	}

	hasNext := len(items) > input.PerPage
	if hasNext {
		items = items[:input.PerPage]
	}
	return items, hasNext, nil
}

func (s *PostgresStore) RevealCell(ctx context.Context, ref TableRef, detail TableDetail, input RevealInput) (any, error) {
	args := []any{}
	clauses := make([]string, 0, len(detail.PrimaryKey))
	for _, key := range detail.PrimaryKey {
		args = append(args, input.RowKeyValues[key])
		clauses = append(clauses, fmt.Sprintf("%s::text = $%d", pgx.Identifier{key}.Sanitize(), len(args)))
	}
	query := fmt.Sprintf(
		"SELECT %s FROM %s WHERE %s LIMIT 1",
		pgx.Identifier{input.Column}.Sanitize(),
		pgx.Identifier{ref.Schema, ref.Name}.Sanitize(),
		strings.Join(clauses, " AND "),
	)

	var value any
	if err := s.pool.QueryRow(ctx, query, args...).Scan(&value); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrRowNotFound
		}
		return nil, fmt.Errorf("reveal database cell: %w", err)
	}
	return normalizePostgresValue(value), nil
}

func (s *PostgresStore) tableOID(ctx context.Context, ref TableRef) (uint32, string, error) {
	var oid uint32
	var kind string
	err := s.pool.QueryRow(ctx, `
		SELECT
			c.oid,
			CASE c.relkind
				WHEN 'r' THEN 'table'
				WHEN 'p' THEN 'partitioned_table'
				WHEN 'v' THEN 'view'
				WHEN 'm' THEN 'materialized_view'
				ELSE c.relkind::text
			END AS kind
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = $1
		  AND c.relname = $2
		  AND c.relkind IN ('r', 'p', 'v', 'm')
		  AND n.nspname NOT IN ('pg_catalog', 'information_schema')
		  AND n.nspname NOT LIKE 'pg_toast%'
	`, ref.Schema, ref.Name).Scan(&oid, &kind)
	if err == pgx.ErrNoRows {
		return 0, "", ErrTableNotFound
	}
	if err != nil {
		return 0, "", fmt.Errorf("find database table: %w", err)
	}
	return oid, kind, nil
}

func (s *PostgresStore) columns(ctx context.Context, oid uint32) ([]Column, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			a.attname,
			format_type(a.atttypid, a.atttypmod),
			NOT a.attnotnull,
			COALESCE(pg_get_expr(ad.adbin, ad.adrelid), '')
		FROM pg_attribute a
		LEFT JOIN pg_attrdef ad ON ad.adrelid = a.attrelid AND ad.adnum = a.attnum
		WHERE a.attrelid = $1
		  AND a.attnum > 0
		  AND NOT a.attisdropped
		ORDER BY a.attnum
	`, oid)
	if err != nil {
		return nil, fmt.Errorf("list database columns: %w", err)
	}
	defer rows.Close()

	columns := []Column{}
	for rows.Next() {
		var column Column
		if err := rows.Scan(&column.Name, &column.DataType, &column.Nullable, &column.DefaultValue); err != nil {
			return nil, fmt.Errorf("scan database column: %w", err)
		}
		columns = append(columns, column)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate database columns: %w", err)
	}
	return columns, nil
}

func (s *PostgresStore) primaryKey(ctx context.Context, oid uint32) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT a.attname
		FROM pg_index i
		JOIN LATERAL unnest(i.indkey) WITH ORDINALITY AS key(attnum, ord) ON true
		JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = key.attnum
		WHERE i.indrelid = $1
		  AND i.indisprimary
		ORDER BY key.ord
	`, oid)
	if err != nil {
		return nil, fmt.Errorf("list database primary key: %w", err)
	}
	defer rows.Close()

	keys := []string{}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("scan database primary key: %w", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate database primary key: %w", err)
	}
	return keys, nil
}

func (s *PostgresStore) indexes(ctx context.Context, oid uint32) ([]Index, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT i.relname, ix.indisunique, ix.indisprimary, pg_get_indexdef(ix.indexrelid)
		FROM pg_index ix
		JOIN pg_class i ON i.oid = ix.indexrelid
		WHERE ix.indrelid = $1
		ORDER BY ix.indisprimary DESC, i.relname
	`, oid)
	if err != nil {
		return nil, fmt.Errorf("list database indexes: %w", err)
	}
	defer rows.Close()

	indexes := []Index{}
	for rows.Next() {
		var index Index
		if err := rows.Scan(&index.Name, &index.Unique, &index.Primary, &index.Definition); err != nil {
			return nil, fmt.Errorf("scan database index: %w", err)
		}
		indexes = append(indexes, index)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate database indexes: %w", err)
	}
	return indexes, nil
}

func (s *PostgresStore) constraints(ctx context.Context, oid uint32) ([]Constraint, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT conname, contype::text, pg_get_constraintdef(oid)
		FROM pg_constraint
		WHERE conrelid = $1
		ORDER BY conname
	`, oid)
	if err != nil {
		return nil, fmt.Errorf("list database constraints: %w", err)
	}
	defer rows.Close()

	constraints := []Constraint{}
	for rows.Next() {
		var constraint Constraint
		if err := rows.Scan(&constraint.Name, &constraint.Type, &constraint.Definition); err != nil {
			return nil, fmt.Errorf("scan database constraint: %w", err)
		}
		constraints = append(constraints, constraint)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate database constraints: %w", err)
	}
	return constraints, nil
}

func columnList(columns []Column) string {
	parts := make([]string, 0, len(columns))
	for _, column := range columns {
		parts = append(parts, pgx.Identifier{column.Name}.Sanitize())
	}
	return strings.Join(parts, ", ")
}

func filterClause(input RowsInput, args *[]any) string {
	if input.FilterColumn == "" {
		return ""
	}
	column := pgx.Identifier{input.FilterColumn}.Sanitize()
	switch input.FilterOperator {
	case "eq":
		*args = append(*args, input.FilterValue)
		return fmt.Sprintf("%s::text = $%d", column, len(*args))
	case "contains":
		*args = append(*args, "%"+escapeLike(input.FilterValue)+"%")
		return fmt.Sprintf("%s::text ILIKE $%d ESCAPE '\\'", column, len(*args))
	case "is_null":
		return fmt.Sprintf("%s IS NULL", column)
	case "not_null":
		return fmt.Sprintf("%s IS NOT NULL", column)
	default:
		return ""
	}
}

// escapeLike 转义 SQL LIKE/ILIKE 元字符，配合 ESCAPE '\' 使用（M6/L4）。
func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	value = strings.ReplaceAll(value, `_`, `\_`)
	return value
}

func orderClause(input RowsInput, detail TableDetail) string {
	sort := input.Sort
	if sort == "" && len(detail.PrimaryKey) > 0 {
		sort = detail.PrimaryKey[0]
	}
	if sort == "" && len(detail.Columns) > 0 {
		sort = detail.Columns[0].Name
	}
	direction := "ASC"
	if input.Direction == "desc" {
		direction = "DESC"
	}
	return fmt.Sprintf(" ORDER BY %s %s", pgx.Identifier{sort}.Sanitize(), direction)
}

func normalizePostgresValue(value any) any {
	switch typed := value.(type) {
	case nil:
		return nil
	case time.Time:
		return typed.UTC().Format(time.RFC3339Nano)
	case []byte:
		if utf8.Valid(typed) {
			return string(typed)
		}
		return base64.StdEncoding.EncodeToString(typed)
	default:
		return typed
	}
}
