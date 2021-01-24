// +build example

package author

import (
	"context"
	"github.com/jackc/pgx/v4"
	"github.com/jschaf/sqld/internal/pgtest"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewQuerier_FindAuthors(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()
	q := NewQuerier(conn)
	insertAuthor(t, conn, "john", "adams")
	insertAuthor(t, conn, "george", "washington")
	insertAuthor(t, conn, "george", "carver")

	tests := []struct {
		firstName string
		want      []FindAuthorsRow
	}{
		{"john", []FindAuthorsRow{{FirstName: "john", LastName: "adams"}}},
		{"george", []FindAuthorsRow{{FirstName: "george", LastName: "washington"}, {FirstName: "george", LastName: "carver"}}},
		{"joe", nil},
	}

	for _, tt := range tests {
		t.Run("FindAuthors "+tt.firstName, func(t *testing.T) {
			authors, err := q.FindAuthors(context.Background(), tt.firstName)
			if err != nil {
				t.Error(err)
			}
			assert.Equal(t, tt.want, authors, "expect authors to match expected")
		})
	}
}

func insertAuthor(t *testing.T, conn *pgx.Conn, first, last string) {
	t.Helper()
	_, err := conn.Exec(context.Background(),
		"INSERT INTO author (first_name, last_name) VALUES ($1, $2)",
		first, last)
	if err != nil {
		t.Fatalf("insert author: %s", err)
	}
}
