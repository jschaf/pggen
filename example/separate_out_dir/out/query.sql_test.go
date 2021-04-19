package out

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/jschaf/pggen/internal/pgtest"
	"github.com/stretchr/testify/assert"
)

func TestNewQuerier_FindAuthorByID(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"../schema.sql"})
	defer cleanup()

	q := NewQuerier(conn)

	t.Run("AlphaNested", func(t *testing.T) {
		got, err := q.AlphaNested(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "alpha_nested", got)
	})

	t.Run("Alpha", func(t *testing.T) {
		got, err := q.Alpha(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "alpha", got)
	})

	t.Run("Bravo", func(t *testing.T) {
		got, err := q.Bravo(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "bravo", got)
	})
}
