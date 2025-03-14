package inline0

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jackc/pgx/v4"
	"github.com/jschaf/pggen/internal/pgtest"
	"github.com/stretchr/testify/assert"
)

func TestNewQuerier_FindAuthorByID(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"../schema.sql"})
	defer cleanup()

	q := NewQuerier(conn)
	adamsID := insertAuthor(t, q, "john", "adams")
	insertAuthor(t, q, "george", "washington")

	t.Run("CountAuthors two", func(t *testing.T) {
		got, err := q.CountAuthors(t.Context())
		require.NoError(t, err)
		assert.Equal(t, 2, *got)
	})

	t.Run("FindAuthorByID", func(t *testing.T) {
		authorByID, err := q.FindAuthorByID(t.Context(), FindAuthorByIDParams{AuthorID: adamsID})
		require.NoError(t, err)
		assert.Equal(t, FindAuthorByIDRow{
			AuthorID:  adamsID,
			FirstName: "john",
			LastName:  "adams",
			Suffix:    nil,
		}, authorByID)
	})

	t.Run("FindAuthorByID - none-exists", func(t *testing.T) {
		missingAuthorByID, err := q.FindAuthorByID(t.Context(), FindAuthorByIDParams{AuthorID: 888})
		require.Error(t, err, "expected error when finding author ID that doesn't match")
		assert.Zero(t, missingAuthorByID, "expected zero value when error")
		if !errors.Is(err, pgx.ErrNoRows) {
			t.Fatalf("expected no rows error to wrap pgx.ErrNoRows; got %s", err)
		}
	})
}

func TestNewQuerier_DeleteAuthorsByFullName(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"../schema.sql"})
	defer cleanup()
	q := NewQuerier(conn)
	insertAuthor(t, q, "george", "washington")

	t.Run("DeleteAuthorsByFullName", func(t *testing.T) {
		tag, err := q.DeleteAuthorsByFullName(t.Context(), DeleteAuthorsByFullNameParams{
			FirstName: "george",
			LastName:  "washington",
			Suffix:    "",
		})
		require.NoError(t, err)
		assert.Truef(t, tag.Delete(), "expected delete tag; got %s", tag.String())
		assert.Equal(t, int64(1), tag.RowsAffected())
	})
}

func insertAuthor(t *testing.T, q *DBQuerier, first, last string) int32 {
	t.Helper()
	authorID, err := q.InsertAuthor(t.Context(), InsertAuthorParams{
		FirstName: first,
		LastName:  last,
	})
	require.NoError(t, err, "insert author")
	return authorID
}
