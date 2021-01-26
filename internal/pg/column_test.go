package pg

import (
	"context"
	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v4"
	"github.com/jschaf/pggen/internal/pgtest"
	"github.com/jschaf/pggen/internal/texts"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestFetchColumns(t *testing.T) {
	tests := []struct {
		name    string
		schema  string
		colNums []int
		want    []Column
	}{
		{"empty", "", nil, nil},
		{
			"one col null",
			"CREATE TABLE author ( first_name text );",
			[]int{1},
			[]Column{{Name: "first_name", TableName: "author", Order: 1, Null: true}},
		},
		{
			"one col not null",
			"CREATE TABLE author ( first_name text NOT NULL);",
			[]int{1},
			[]Column{{Name: "first_name", TableName: "author", Order: 1, Null: false}},
		},
		{
			"two col mixed",
			"CREATE TABLE author ( first_name text NOT NULL, last_name text);",
			[]int{2, 1},
			[]Column{
				{Name: "last_name", TableName: "author", Order: 2, Null: true},
				{Name: "first_name", TableName: "author", Order: 1, Null: false},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, cleanup := pgtest.NewPostgresSchemaString(t, tt.schema)
			defer cleanup()
			oid := findTableOID(t, conn, "author")
			keys := make([]ColumnKey, len(tt.colNums))
			for i, num := range tt.colNums {
				keys[i] = ColumnKey{oid, num}
			}
			cols, err := FetchColumns(conn, keys)
			if err != nil {
				t.Fatal(err)
			}
			// Add table OID to each key.
			for i, col := range tt.want {
				col.TableOID = oid
				tt.want[i] = col
			}
			if diff := cmp.Diff(tt.want, cols); diff != "" {
				t.Errorf("FetchColumns() query mismatch (-want +got):\n%s", diff)
			}

			// Test cache.
			cols2, err := FetchColumns(conn, keys)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, cols, cols2, "same fetch columns in succession")
		})
	}
}

func findTableOID(t *testing.T, conn *pgx.Conn, table string) OIDInt {
	sql := texts.Dedent(`
		SELECT oid AS table_oid
		FROM pg_class
		WHERE relname = $1
		ORDER BY table_oid DESC
		LIMIT 1;
	`)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	row := conn.QueryRow(ctx, sql, table)
	var oid OIDInt = 0
	if err := row.Scan(&oid); err != nil {
		t.Fatal(err)
	}
	return oid
}
