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

	_, err := q.CustomTypes(ctx)
	assert.NoError(t, err)

	batch := &pgx.Batch{}
	q.CustomTypesBatch(batch)
	results := conn.SendBatch(ctx, batch)
	_, err = q.CustomTypesScan(results)
	assert.NoError(t, err)
}
