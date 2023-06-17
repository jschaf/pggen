package pg

import (
	"context"
	"fmt"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/jschaf/pggen/internal/texts"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Column stores information about a column in a TableOID.
// https://www.postgresql.org/docs/13/catalog-pg-attribute.html
type Column struct {
	Name      string     // pg_attribute.attname: column name
	TableOID  pgtype.OID // pg_attribute:attrelid: table the column belongs to
	TableName string     // pg_class.relname: name of table that owns the column
	Number    uint16     // pg_attribute.attnum: the number of column starting from 1
	Type      Type       // pg_attribute.atttypid: data type of the column
	Null      bool       // pg_attribute.attnotnull: represents a not-null constraint
}

// ColumnKey is a composite key of a table OID and the number of the column
// within the table.
type ColumnKey struct {
	TableOID pgtype.OID
	Number   uint16 // the number of column starting from 1
}

var (
	columnsMu   = &sync.Mutex{}
	columnCache = make(map[ColumnKey]Column, 32)
)

// FetchColumns fetches meta information about a Postgres column from the
// pg_class and pg_attribute catalog tables.
func FetchColumns(conn *pgx.Conn, keys []ColumnKey) ([]Column, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	// Try cache first.
	uncachedKeys := make([]ColumnKey, 0, len(keys))
	columnsMu.Lock()
	for _, key := range keys {
		if _, ok := columnCache[key]; !ok && key.TableOID > 0 {
			uncachedKeys = append(uncachedKeys, key)
		}
	}
	columnsMu.Unlock()
	if len(uncachedKeys) == 0 {
		return fetchCachedColumns(keys)
	}

	// Build query predicate.
	predicate := &strings.Builder{}
	predicate.Grow(len(uncachedKeys) * 40)
	for i, key := range uncachedKeys {
		predicate.WriteString("(cls.oid = ")
		predicate.WriteString(strconv.Itoa(int(key.TableOID)))
		predicate.WriteString(" AND attr.attnum = ")
		predicate.WriteString(strconv.Itoa(int(key.Number)))
		predicate.WriteString(")")
		if i < len(uncachedKeys)-1 {
			predicate.WriteString("\n    OR ")
		}
	}

	// Execute query.
	q := texts.Dedent(`
		SELECT cls.oid         AS table_oid,
					 cls.relname     AS table_name,
					 attr.attname    AS col_name,
					 attr.attnum     AS col_num,
					 attr.attnotnull AS col_null
		FROM pg_class cls
					 JOIN pg_attribute attr ON (attr.attrelid = cls.oid)
	`) + "\nWHERE " + predicate.String()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rows, err := conn.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("fetch column metadata: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		col := Column{}
		notNull := false
		if err := rows.Scan(&col.TableOID, &col.TableName, &col.Name, &col.Number, &notNull); err != nil {
			return nil, fmt.Errorf("scan fetch column row: %w", err)
		}
		col.Null = !notNull
		columnCache[ColumnKey{col.TableOID, col.Number}] = col
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("close fetch column rows: %w", err)
	}

	return fetchCachedColumns(keys)
}

func fetchCachedColumns(keys []ColumnKey) ([]Column, error) {
	cols := make([]Column, 0, len(keys))
	columnsMu.Lock()
	defer columnsMu.Unlock()
	for _, key := range keys {
		col, ok := columnCache[key]
		// Ignore columns not directly backed by a table.
		if !ok && col.TableOID > 0 {
			return nil, fmt.Errorf("missing column in fetch cache table_oid=%d Number=%d", key.TableOID, key.Number)
		}
		cols = append(cols, col)
	}
	return cols, nil
}
