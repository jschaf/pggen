package go_pointer_types

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

	{
		got, err := q.GenSeries(ctx)
		if err != nil {
			t.Fatal(err)
		}
		zero, one, two := 0, 1, 2
		assert.Equal(t, []*int{&zero, &one, &two}, got)
	}
}
