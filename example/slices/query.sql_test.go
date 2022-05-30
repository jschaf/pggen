package slices

import (
	"context"
	"github.com/jackc/pgx/v4"
	"github.com/jschaf/pggen/internal/difftest"
	"github.com/jschaf/pggen/internal/errs"
	"github.com/jschaf/pggen/internal/pgtest"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestNewQuerier_GetBools(t *testing.T) {
	ctx := context.Background()
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()

	q := NewQuerier(conn)

	t.Run("GetBools", func(t *testing.T) {
		want := []bool{true, true, false}
		got, err := q.GetBools(ctx, want)
		require.NoError(t, err)
		difftest.AssertSame(t, want, got)
	})

	t.Run("GetBoolsBatch", func(t *testing.T) {
		batch := &pgx.Batch{}
		want := []bool{true, true, false}
		q.GetBoolsBatch(batch, want)
		results := conn.SendBatch(ctx, batch)
		defer errs.CaptureT(t, results.Close, "close batch results")
		got, err := q.GetBoolsScan(results)
		require.NoError(t, err)
		difftest.AssertSame(t, want, got)
	})
}

func TestNewQuerier_GetOneTimestamp(t *testing.T) {
	ctx := context.Background()
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()

	q := NewQuerier(conn)
	ts := time.Date(2020, 1, 1, 11, 11, 11, 0, time.UTC)

	t.Run("GetOneTimestamp", func(t *testing.T) {
		got, err := q.GetOneTimestamp(ctx, &ts)
		require.NoError(t, err)
		difftest.AssertSame(t, &ts, got)
	})

	t.Run("GetOneTimestampBatch", func(t *testing.T) {
		batch := &pgx.Batch{}
		q.GetOneTimestampBatch(batch, &ts)
		results := conn.SendBatch(ctx, batch)
		defer errs.CaptureT(t, results.Close, "close batch results")
		got, err := q.GetOneTimestampScan(results)
		require.NoError(t, err)
		difftest.AssertSame(t, &ts, got)
	})
}

func TestNewQuerier_GetManyTimestamptzs(t *testing.T) {
	ctx := context.Background()
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()

	q := NewQuerier(conn)
	ts1 := time.Date(2020, 1, 1, 11, 11, 11, 0, time.UTC)
	ts2 := time.Date(2022, 2, 2, 22, 22, 22, 0, time.UTC)

	t.Run("GetManyTimestamptzs", func(t *testing.T) {
		got, err := q.GetManyTimestamptzs(ctx, []time.Time{ts1, ts2})
		require.NoError(t, err)
		difftest.AssertSame(t, []*time.Time{&ts1, &ts2}, got)
	})

	t.Run("GetManyTimestamptzsBatch", func(t *testing.T) {
		batch := &pgx.Batch{}
		q.GetManyTimestamptzsBatch(batch, []time.Time{ts1, ts2})
		results := conn.SendBatch(ctx, batch)
		defer errs.CaptureT(t, results.Close, "close batch results")
		got, err := q.GetManyTimestamptzsScan(results)
		require.NoError(t, err)
		difftest.AssertSame(t, []*time.Time{&ts1, &ts2}, got)
	})
}
