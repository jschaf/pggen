package domain

import (
	"context"
	"github.com/jackc/pgx/v4"
	"github.com/jschaf/pggen/internal/errs"
	"github.com/jschaf/pggen/internal/pgtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestQuerier_DomainOne(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()

	q := NewQuerier(conn)
	ctx := context.Background()

	t.Run("DomainOne", func(t *testing.T) {
		postCode, err := q.DomainOne(ctx)
		require.NoError(t, err)
		assert.Equal(t, "90210", postCode)
	})

	t.Run("DomainOneBatch", func(t *testing.T) {
		batch := &pgx.Batch{}
		q.DomainOneBatch(batch)
		results := conn.SendBatch(context.Background(), batch)
		got, err := q.DomainOneScan(results)
		defer errs.CaptureT(t, results.Close, "close batch")
		require.NoError(t, err)
		assert.Equal(t, "90210", got)
	})
}
