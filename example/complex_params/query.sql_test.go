package complex_params

import (
	"context"
	"github.com/jackc/pgx/v4"
	"github.com/jschaf/pggen/internal/pgtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewQuerier_ParamNested1(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()

	q := NewQuerier(conn)
	ctx := context.Background()

	wantDim := Dimensions{Width: 77, Height: 77}

	t.Run("ParamNested1", func(t *testing.T) {
		row, err := q.ParamNested1(ctx, wantDim)
		require.NoError(t, err)
		assert.Equal(t, wantDim, row)
	})

	t.Run("ParamNested1Batch", func(t *testing.T) {
		batch := &pgx.Batch{}
		q.ParamNested1Batch(batch, wantDim)
		results := conn.SendBatch(ctx, batch)
		row, err := q.ParamNested1Scan(results)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, wantDim, row)
	})
}

func TestNewQuerier_ParamNested2(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()

	q := NewQuerier(conn)
	ctx := context.Background()

	wantDim := Dimensions{Width: 77, Height: 77}
	wantImg := ProductImageType{
		Source:     "src",
		Dimensions: wantDim,
	}

	t.Run("ParamNested2", func(t *testing.T) {
		row, err := q.ParamNested2(ctx, wantImg)
		require.NoError(t, err)
		assert.Equal(t, wantImg, row)
	})

	t.Run("ParamNested2Batch", func(t *testing.T) {
		batch := &pgx.Batch{}
		q.ParamNested2Batch(batch, wantImg)
		results := conn.SendBatch(ctx, batch)
		row, err := q.ParamNested2Scan(results)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, wantImg, row)
	})
}
