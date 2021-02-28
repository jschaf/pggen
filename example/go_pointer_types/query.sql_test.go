package go_pointer_types

import (
	"context"
	"github.com/jackc/pgx/v4"
	"github.com/jschaf/pggen/internal/errs"
	"github.com/jschaf/pggen/internal/pgtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestQuerier_GenSeries1(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()

	q := NewQuerier(conn)
	ctx := context.Background()

	t.Run("GenSeries1", func(t *testing.T) {
		got, err := q.GenSeries1(ctx)
		require.Nil(t, err)
		zero := 0
		assert.Equal(t, &zero, got)
	})

	t.Run("GenSeries1 - Scan", func(t *testing.T) {
		batch := &pgx.Batch{}
		q.GenSeries1Batch(batch)
		results := conn.SendBatch(ctx, batch)
		defer errs.CaptureT(t, results.Close, "close batch")
		got, err := q.GenSeries1Scan(results)
		require.Nil(t, err)
		zero := 0
		assert.Equal(t, &zero, got)
	})
}

func TestQuerier_GenSeries(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()

	q := NewQuerier(conn)
	ctx := context.Background()

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
		require.Nil(t, err)
		zero, one, two := 0, 1, 2
		assert.Equal(t, []*int{&zero, &one, &two}, got)
	})
}

func TestQuerier_GenSeriesArr1(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()

	q := NewQuerier(conn)
	ctx := context.Background()

	t.Run("GenSeriesArr1", func(t *testing.T) {
		got, err := q.GenSeriesArr1(ctx)
		require.Nil(t, err)
		assert.Equal(t, []int{0, 1, 2}, got)
	})

	t.Run("GenSeriesArr1 - Scan", func(t *testing.T) {
		batch := &pgx.Batch{}
		q.GenSeriesArr1Batch(batch)
		results := conn.SendBatch(ctx, batch)
		defer errs.CaptureT(t, results.Close, "close batch")
		got, err := q.GenSeriesArr1Scan(results)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, []int{0, 1, 2}, got)
	})
}

func TestQuerier_GenSeriesArr(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()

	q := NewQuerier(conn)
	ctx := context.Background()

	t.Run("GenSeriesArr", func(t *testing.T) {
		got, err := q.GenSeriesArr(ctx)
		require.Nil(t, err)
		assert.Equal(t, [][]int{{0, 1, 2}}, got)
	})

	t.Run("GenSeriesArr - Scan", func(t *testing.T) {
		batch := &pgx.Batch{}
		q.GenSeriesArrBatch(batch)
		results := conn.SendBatch(ctx, batch)
		defer errs.CaptureT(t, results.Close, "close batch")
		got, err := q.GenSeriesArrScan(results)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, [][]int{{0, 1, 2}}, got)
	})
}

func TestQuerier_GenSeriesStr(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()

	q := NewQuerier(conn)
	ctx := context.Background()

	t.Run("GenSeriesStr1", func(t *testing.T) {
		got, err := q.GenSeriesStr1(ctx)
		require.Nil(t, err)
		zero := "0"
		assert.Equal(t, &zero, got)
	})

	t.Run("GenSeriesStr1 - Scan", func(t *testing.T) {
		batch := &pgx.Batch{}
		q.GenSeriesStr1Batch(batch)
		results := conn.SendBatch(ctx, batch)
		defer errs.CaptureT(t, results.Close, "close batch")
		got, err := q.GenSeriesStr1Scan(results)
		require.Nil(t, err)
		zero := "0"
		assert.Equal(t, &zero, got)
	})

	t.Run("GenSeriesStr", func(t *testing.T) {
		got, err := q.GenSeriesStr(ctx)
		require.Nil(t, err)
		zero, one, two := "0", "1", "2"
		assert.Equal(t, []*string{&zero, &one, &two}, got)
	})

	t.Run("GenSeriesStr - Scan", func(t *testing.T) {
		batch := &pgx.Batch{}
		q.GenSeriesStrBatch(batch)
		results := conn.SendBatch(ctx, batch)
		defer errs.CaptureT(t, results.Close, "close batch")
		got, err := q.GenSeriesStrScan(results)
		require.Nil(t, err)
		zero, one, two := "0", "1", "2"
		assert.Equal(t, []*string{&zero, &one, &two}, got)
	})
}
