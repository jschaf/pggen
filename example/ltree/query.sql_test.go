package ltree

import (
	"context"
	"github.com/jackc/pgx/v5/pgtype"
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
		require.NoError(t, err)
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
		require.NoError(t, err)
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

	{
		in1 := pgtype.Text{String: "foo", Status: pgtype.Present}
		in2 := []string{"qux", "qux"}
		in2Txt := newTextArray(in2)
		rows, err := q.FindLtreeInput(ctx, in1, in2)
		require.NoError(t, err)
		assert.Equal(t, FindLtreeInputRow{
			Ltree:   in1,
			TextArr: in2Txt,
		}, rows)
	}
}

// newTextArray creates a one dimensional text array from the string slice with
// no null elements.
func newTextArray(ss []string) pgtype.TextArray {
	elems := make([]pgtype.Text, len(ss))
	for i, s := range ss {
		elems[i] = pgtype.Text{String: s, Status: pgtype.Present}
	}
	return pgtype.TextArray{
		Elements:   elems,
		Dimensions: []pgtype.ArrayDimension{{Length: int32(len(ss)), LowerBound: 1}},
		Status:     pgtype.Present,
	}
}
