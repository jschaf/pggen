package go_pointer_types

import (
	"context"
	"github.com/jackc/pgx/v4"
	"github.com/jschaf/pggen/internal/errs"
	"github.com/jschaf/pggen/internal/pgtest"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestQuerier(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()

	q := NewQuerier(conn)
	ctx := context.Background()

	t.Run("GenSeries1", func(t *testing.T) {
		got, err := q.GenSeries1(ctx)
		if err != nil {
			t.Fatal(err)
		}
		zero := 0
		assert.Equal(t, &zero, got)
	})

	t.Run("GenSeries1 - Scan", func(t *testing.T) {
		batch := &pgx.Batch{}
		q.GenSeries1Batch(batch)
		results := conn.SendBatch(ctx, batch)
		defer errs.CaptureT(t, results.Close, "close batch")
		got, err := q.GenSeries1Scan(results)
		if err != nil {
			t.Fatal(err)
		}
		zero := 0
		assert.Equal(t, &zero, got)
	})

	t.Run("GenSeries", func(t *testing.T) {
		got, err := q.GenSeries(ctx)
		if err != nil {
			t.Fatal(err)
		}
		zero, one, two := 0, 1, 2
		assert.Equal(t, []*int{&zero, &one, &two}, got)
	})

	t.Run("GenSeries - Scan", func(t *testing.T) {
		batch := &pgx.Batch{}
		q.GenSeriesBatch(batch)
		results := conn.SendBatch(ctx, batch)
		defer errs.CaptureT(t, results.Close, "close batch")
		got, err := q.GenSeriesScan(results)
		if err != nil {
			t.Fatal(err)
		}
		zero, one, two := 0, 1, 2
		assert.Equal(t, []*int{&zero, &one, &two}, got)
	})
}
