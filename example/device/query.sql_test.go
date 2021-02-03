package device

import (
	"context"
	"github.com/jackc/pgx/v4"
	"github.com/jschaf/pggen/internal/pgtest"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestQuerier(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, nil)
	defer cleanup()
	q := NewQuerier(conn)
	ctx := context.Background()

	_, err := q.FindDevicesByUser(ctx, 8)
	assert.NoError(t, err)

	batch := &pgx.Batch{}
	q.FindDevicesByUserBatch(batch, 3)
	results := conn.SendBatch(ctx, batch)
	_, err = q.FindDevicesByUserScan(results)
	assert.NoError(t, err)
}
