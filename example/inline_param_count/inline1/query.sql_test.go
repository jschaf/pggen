package inline1

import (
	"context"
	"errors"
	"github.com/jschaf/pggen/internal/errs"
	"github.com/stretchr/testify/require"
	"testing"

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
		got, err := q.CountAuthors(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 2, *got)
	})

	t.Run("FindAuthorByID", func(t *testing.T) {
		authorByID, err := q.FindAuthorByID(context.Background(), adamsID)
		require.NoError(t, err)
		assert.Equal(t, FindAuthorByIDRow{
			AuthorID:  adamsID,
			FirstName: "john",
			LastName:  "adams",
			Suffix:    nil,
		}, authorByID)
	})

	t.Run("FindAuthorByID - none-exists", func(t *testing.T) {
		missingAuthorByID, err := q.FindAuthorByID(context.Background(), 888)
		require.Error(t, err, "expected error when finding author ID that doesn't match")
		assert.Zero(t, missingAuthorByID, "expected zero value when error")
		if !errors.Is(err, pgx.ErrNoRows) {
			t.Fatalf("expected no rows error to wrap pgx.ErrNoRows; got %s", err)
		}
	})

	t.Run("FindAuthorByIDBatch", func(t *testing.T) {
		batch := &pgx.Batch{}
		q.FindAuthorByIDBatch(batch, adamsID)
		results := conn.SendBatch(context.Background(), batch)
		defer errs.CaptureT(t, results.Close, "close batch results")
		authors, err := q.FindAuthorByIDScan(results)
		require.NoError(t, err)
		assert.Equal(t, FindAuthorByIDRow{
			AuthorID:  adamsID,
			FirstName: "john",
			LastName:  "adams",
			Suffix:    nil,
		}, authors)
	})
}

func TestNewQuerier_DeleteAuthorsByFullName(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"../schema.sql"})
	defer cleanup()
	q := NewQuerier(conn)
	insertAuthor(t, q, "george", "washington")

	t.Run("DeleteAuthorsByFullName", func(t *testing.T) {
		tag, err := q.DeleteAuthorsByFullName(context.Background(), DeleteAuthorsByFullNameParams{
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
	authorID, err := q.InsertAuthor(context.Background(), InsertAuthorParams{
		FirstName: first,
		LastName:  last,
	})
	require.NoError(t, err, "insert author")
	return authorID
}
