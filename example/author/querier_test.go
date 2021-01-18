// +build example

package author

import (
	"context"
	"github.com/jackc/pgx/v4"
	"github.com/jschaf/sqld/pgtest"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewQuerier_FindAuthors(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()
	insertAuthor(t, conn, "john", "adams")

	q := NewQuerier(conn, NewNopHook())
	authors, err := q.FindAuthors(context.Background(), "john")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t,
		[]Author{{FirstName: "john", LastName: "adams"}},
		authors,
		"expect authors to contain john adams")
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
