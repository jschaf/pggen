package syntax

import (
	"context"
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
}
