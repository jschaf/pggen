package custom_types

import (
	"context"
	"github.com/jackc/pgx/v4"
	"github.com/jschaf/pggen/internal/pgtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

func TestQuerier_CustomTypes(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()
	q := NewQuerier(conn)
	ctx := context.Background()

	t.Run("CustomTypes", func(t *testing.T) {
		val, err := q.CustomTypes(ctx)
		require.NoError(t, err)
		want := CustomTypesRow{
			Column: "some_text",
			Int8:   1,
		}
		assert.Equal(t, want, val)
	})

	t.Run("CustomTypesBatch", func(t *testing.T) {
		batch := &pgx.Batch{}
		q.CustomTypesBatch(batch)
		results := conn.SendBatch(context.Background(), batch)
		val, err := q.CustomTypesScan(results)
		require.NoError(t, err)
		want := CustomTypesRow{
			Column: "some_text",
			Int8:   1,
		}
		assert.Equal(t, want, val)
	})
}

func TestQuerier_CustomMyInt(t *testing.T) {
	t.SkipNow() // TODO: https://github.com/jackc/pgx/issues/953
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()
	val := 0
	conn.ConnInfo().RegisterDefaultPgType(val, "my_int")
	valueType := reflect.TypeOf(val)
	conn.ConnInfo().RegisterDefaultPgType(reflect.New(valueType).Interface(), "my_int")

	q := NewQuerier(conn)
	ctx := context.Background()

	t.Run("CustomMyInt", func(t *testing.T) {
		val, err := q.CustomMyInt(ctx)
		require.NoError(t, err)
		assert.Equal(t, 5, val)
	})

	t.Run("CustomMyIntBatch", func(t *testing.T) {
		batch := &pgx.Batch{}
		q.CustomMyIntBatch(batch)
		results := conn.SendBatch(context.Background(), batch)
		val, err := q.CustomMyIntScan(results)
		require.NoError(t, err)
		assert.Equal(t, 5, val)
	})
}

func TestQuerier_IntArray(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()
	q := NewQuerier(conn)
	ctx := context.Background()

	t.Run("IntArray", func(t *testing.T) {
		array, err := q.IntArray(ctx)
		require.NoError(t, err)
		assert.Equal(t, [][]int32{{5, 6, 7}}, array)
	})

	t.Run("IntArrayBatch", func(t *testing.T) {
		batch := &pgx.Batch{}
		q.IntArrayBatch(batch)
		results := conn.SendBatch(context.Background(), batch)
		val, err := q.IntArrayScan(results)
		assert.NoError(t, err)
		assert.Equal(t, [][]int32{{5, 6, 7}}, val)
	})
}
