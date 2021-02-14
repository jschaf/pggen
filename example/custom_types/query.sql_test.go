package custom_types

import (
	"context"
	"github.com/jackc/pgx/v4"
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
		_, err := q.CustomTypes(ctx)
		assert.NoError(t, err)
	}

	{
		batch := &pgx.Batch{}
		q.CustomTypesBatch(batch)
		results := conn.SendBatch(ctx, batch)
		_, err := q.CustomTypesScan(results)
		_ = results.Close()
		assert.NoError(t, err)
	}

	{
		array, err := q.IntArray(ctx)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, [][]int32{{5, 6, 7}}, array)
	}
}
