package syntax

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

	val, err := q.Backtick(ctx)
	assert.NoError(t, err, "Backtick")
	assert.Equal(t, "`", val, "Backtick")

	val, err = q.BacktickDoubleQuote(ctx)
	assert.NoError(t, err, "BacktickDoubleQuote")
	assert.Equal(t, "`\"", val, "BacktickDoubleQuote")

	val, err = q.BacktickQuoteBacktick(ctx)
	assert.NoError(t, err, "BacktickQuoteBacktick")
	assert.Equal(t, "`\"`", val, "BacktickQuoteBacktick")

	val, err = q.BacktickNewline(ctx)
	assert.NoError(t, err, "BacktickNewline")
	assert.Equal(t, "`\n", val, "BacktickNewline")

	val, err = q.BacktickBackslashN(ctx)
	assert.NoError(t, err, "BacktickBackslashN")
	assert.Equal(t, "`\\n", val, "BacktickBackslashN")

	batch := &pgx.Batch{}
	q.BacktickBatch(ctx, batch)
	q.BacktickDoubleQuoteBatch(ctx, batch)
	q.BacktickQuoteBacktickBatch(ctx, batch)
	q.BacktickNewlineBatch(ctx, batch)
	q.BacktickBackslashNBatch(ctx, batch)
	results := conn.SendBatch(ctx, batch)

	val, err = q.BacktickScan(results)
	assert.NoError(t, err, "BacktickScan")
	assert.Equal(t, "`", val, "BacktickScan")

	val, err = q.BacktickDoubleQuoteScan(results)
	assert.NoError(t, err, "BacktickDoubleQuoteScan")
	assert.Equal(t, "`\"", val, "BacktickDoubleQuoteScan")

	val, err = q.BacktickQuoteBacktickScan(results)
	assert.NoError(t, err, "BacktickQuoteBacktickScan")
	assert.Equal(t, "`\"`", val, "BacktickQuoteBacktickScan")

	val, err = q.BacktickNewlineScan(results)
	assert.NoError(t, err, "BacktickNewlineScan")
	assert.Equal(t, "`\n", val, "BacktickNewlineScan")

	val, err = q.BacktickBackslashNScan(results)
	assert.NoError(t, err, "BacktickBackslashNScan")
	assert.Equal(t, "`\\n", val, "BacktickBackslashNScan")
}
