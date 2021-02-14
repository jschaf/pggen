package domain

import (
	"context"
	"github.com/jschaf/pggen/internal/pgtest"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestQuerier(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()

	q := NewQuerier(conn)
	ctx := context.Background()

	postCode, err := q.DomainOne(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "90210", postCode)
}
