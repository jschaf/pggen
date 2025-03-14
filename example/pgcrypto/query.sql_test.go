package pgcrypto

import (
	"strings"
	"testing"

	"github.com/jschaf/pggen/internal/pgtest"
	"github.com/stretchr/testify/assert"
)

func TestQuerier(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()

	q := NewQuerier(conn)
	ctx := t.Context()

	_, err := q.CreateUser(ctx, "foo", "hunter2")
	if err != nil {
		t.Fatal(err)
	}

	row, err := q.FindUser(ctx, "foo")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "foo", row.Email, "email should match")
	if !strings.HasPrefix(row.Pass, "$2a$") {
		t.Fatalf("expected hashed password to have prefix $2a$; got %s", row.Pass)
	}
}
