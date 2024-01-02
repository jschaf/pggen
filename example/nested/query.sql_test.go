package nested

import (
	"context"
	"github.com/jschaf/pggen/internal/pgtest"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewQuerier_ArrayNested2(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()

	q := NewQuerier(conn)
	ctx := context.Background()

	want := []ProductImageType{
		{Source: "img2", Dimensions: Dimensions{22, 22}},
		{Source: "img3", Dimensions: Dimensions{33, 33}},
	}
	t.Run("ArrayNested2", func(t *testing.T) {
		rows, err := q.ArrayNested2(ctx)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, want, rows)
	})
}

func TestNewQuerier_Nested3(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()

	q := NewQuerier(conn)
	ctx := context.Background()

	want := []ProductImageSetType{
		{
			Name: "name",
			OrigImage: ProductImageType{
				Source:     "img1",
				Dimensions: Dimensions{Width: 11, Height: 11},
			},
			Images: []ProductImageType{
				{Source: "img2", Dimensions: Dimensions{22, 22}},
				{Source: "img3", Dimensions: Dimensions{33, 33}},
			},
		},
	}
	t.Run("Nested3", func(t *testing.T) {
		t.Skipf("https://github.com/jackc/pgx/issues/874")
		rows, err := q.Nested3(ctx)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, want, rows)
	})
}
