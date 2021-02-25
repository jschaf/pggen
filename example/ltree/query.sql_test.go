package ltree

import (
	"context"
	"github.com/jackc/pgtype"
	"github.com/jschaf/pggen/internal/pgtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestQuerier(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()

	q := NewQuerier(conn)
	ctx := context.Background()

	if _, err := q.InsertSampleData(ctx); err != nil {
		t.Fatal(err)
	}

	{
		rows, err := q.FindTopScienceChildren(ctx)
		require.Nil(t, err)
		want := []pgtype.Text{
			{String: "Top.Science", Status: pgtype.Present},
			{String: "Top.Science.Astronomy", Status: pgtype.Present},
			{String: "Top.Science.Astronomy.Astrophysics", Status: pgtype.Present},
			{String: "Top.Science.Astronomy.Cosmology", Status: pgtype.Present},
		}
		assert.Equal(t, want, rows)
	}

	{
		rows, err := q.FindTopScienceChildrenAgg(ctx)
		require.Nil(t, err)
		want := pgtype.TextArray{
			Elements: []pgtype.Text{
				{String: "Top.Science", Status: pgtype.Present},
				{String: "Top.Science.Astronomy", Status: pgtype.Present},
				{String: "Top.Science.Astronomy.Astrophysics", Status: pgtype.Present},
				{String: "Top.Science.Astronomy.Cosmology", Status: pgtype.Present},
			},
			Status:     pgtype.Present,
			Dimensions: []pgtype.ArrayDimension{{Length: 4, LowerBound: 1}},
		}
		assert.Equal(t, want, rows)
	}
}
