package void

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

	if _, err := q.VoidOnly(ctx); err != nil {
		t.Fatal(err)
	}

	if _, err := q.VoidOnlyTwoParams(ctx, 33); err != nil {
		t.Fatal(err)
	}

	{
		row, err := q.VoidTwo(ctx)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, "foo", row)
	}

	{
		row, err := q.VoidThree(ctx)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, VoidThreeRow{Foo: "foo", Bar: "bar"}, row)
	}

	{
		foos, err := q.VoidThree2(ctx)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, []string{"foo"}, foos)
	}
}
