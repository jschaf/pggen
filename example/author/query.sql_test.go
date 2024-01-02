package author

import (
	"context"
	"errors"
	"github.com/jschaf/pggen/internal/ptrs"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/jackc/pgx/v4"
	"github.com/jschaf/pggen/internal/pgtest"
	"github.com/stretchr/testify/assert"
)

func TestNewQuerier_FindAuthorByID(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()

	q := NewQuerier(conn)
	adamsID := insertAuthor(t, q, "john", "adams")
	insertAuthor(t, q, "george", "washington")

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
}

func TestNewQuerier_FindAuthors(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()
	q := NewQuerier(conn)
	adamsID := insertAuthor(t, q, "john", "adams")
	washingtonID := insertAuthor(t, q, "george", "washington")
	carverID := insertAuthor(t, q, "george", "carver")

	t.Run("FindAuthors - 1 row - john", func(t *testing.T) {
		authors, err := q.FindAuthors(context.Background(), "john")
		require.NoError(t, err)
		want := []FindAuthorsRow{
			{
				AuthorID:  adamsID,
				FirstName: "john",
				LastName:  "adams",
				Suffix:    nil,
			},
		}
		assert.Equal(t, want, authors)
	})

	t.Run("FindAuthors - 2 rows - george", func(t *testing.T) {
		authors, err := q.FindAuthors(context.Background(), "george")
		require.NoError(t, err)
		want := []FindAuthorsRow{
			{AuthorID: washingtonID, FirstName: "george", LastName: "washington", Suffix: nil},
			{AuthorID: carverID, FirstName: "george", LastName: "carver", Suffix: nil},
		}
		assert.Equal(t, want, authors)
	})

	t.Run("FindAuthors - 0 rows - joe", func(t *testing.T) {
		authors, err := q.FindAuthors(context.Background(), "joe")
		require.NoError(t, err)
		assert.Equal(t, []FindAuthorsRow{}, authors)
	})
}

func TestNewQuerier_FindFirstNames(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()

	q := NewQuerier(conn)
	adamsID := insertAuthor(t, q, "john", "adams")
	insertAuthor(t, q, "george", "washington")

	t.Run("FindAuthorByID", func(t *testing.T) {
		firstNames, err := q.FindFirstNames(context.Background(), adamsID)
		require.NoError(t, err)
		assert.Equal(t, []*string{ptrs.String("george"), ptrs.String("john")}, firstNames)
	})
}

func TestNewQuerier_InsertAuthorSuffix(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()
	q := NewQuerier(conn)

	t.Run("InsertAuthorSuffix", func(t *testing.T) {
		author, err := q.InsertAuthorSuffix(context.Background(), InsertAuthorSuffixParams{
			FirstName: "john",
			LastName:  "adams",
			Suffix:    "Jr.",
		})
		jr := "Jr."
		require.NoError(t, err)
		want := InsertAuthorSuffixRow{
			AuthorID:  author.AuthorID,
			FirstName: "john",
			LastName:  "adams",
			Suffix:    &jr,
		}
		assert.Equal(t, want, author)
	})
}

func TestNewQuerier_DeleteAuthorsByFirstName(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()
	q := NewQuerier(conn)
	insertAuthor(t, q, "john", "adams")
	insertAuthor(t, q, "george", "washington")
	insertAuthor(t, q, "george", "carver")

	t.Run("DeleteAuthorsByFirstName", func(t *testing.T) {
		tag, err := q.DeleteAuthorsByFirstName(context.Background(), "george")
		require.NoError(t, err)
		assert.Truef(t, tag.Delete(), "expected delete tag; got %s", tag.String())
		assert.Equal(t, int64(2), tag.RowsAffected())

		authors, err := q.FindAuthors(context.Background(), "george")
		require.NoError(t, err)
		assert.Empty(t, authors, "no authors should remain with first name of george")
	})
}

func TestNewQuerier_DeleteAuthorsByFullName(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()
	q := NewQuerier(conn)
	washingtonID := insertAuthor(t, q, "george", "washington")
	_, err := q.InsertAuthorSuffix(context.Background(), InsertAuthorSuffixParams{
		FirstName: "george",
		LastName:  "washington",
		Suffix:    "Jr.",
	})
	require.NoError(t, err)

	t.Run("DeleteAuthorsByFullName", func(t *testing.T) {
		tag, err := q.DeleteAuthorsByFullName(context.Background(), DeleteAuthorsByFullNameParams{
			FirstName: "george",
			LastName:  "washington",
			Suffix:    "Jr.",
		})
		require.NoError(t, err)
		assert.Truef(t, tag.Delete(), "expected delete tag; got %s", tag.String())
		assert.Equal(t, int64(1), tag.RowsAffected())

		authors, err := q.FindAuthors(context.Background(), "george")
		require.NoError(t, err)
		want := []FindAuthorsRow{
			{
				AuthorID:  washingtonID,
				FirstName: "george",
				LastName:  "washington",
				Suffix:    nil,
			},
		}
		assert.Equal(t, want, authors, "only one author with first name george should remain")
	})
}

func TestNewQuerier_StringAggFirstName(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()
	q := NewQuerier(conn)
	washingtonID := insertAuthor(t, q, "george", "washington")
	_, err := q.InsertAuthorSuffix(context.Background(), InsertAuthorSuffixParams{
		FirstName: "george",
		LastName:  "washington",
		Suffix:    "Jr.",
	})
	require.NoError(t, err)

	t.Run("StringAggFirstName - null", func(t *testing.T) {
		firstNames, err := q.StringAggFirstName(context.Background(), 999)
		require.NoError(t, err)
		require.Nil(t, firstNames)
	})

	t.Run("StringAggFirstName - one", func(t *testing.T) {
		firstNames, err := q.StringAggFirstName(context.Background(), washingtonID)
		require.NoError(t, err)
		assert.Equal(t, "george", *firstNames)
	})
}

func TestNewQuerier_ArrayAggFirstName(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()
	q := NewQuerier(conn)
	washingtonID := insertAuthor(t, q, "george", "washington")
	_, err := q.InsertAuthorSuffix(context.Background(), InsertAuthorSuffixParams{
		FirstName: "george",
		LastName:  "washington",
		Suffix:    "Jr.",
	})
	require.NoError(t, err)

	t.Run("ArrayAggFirstName - null", func(t *testing.T) {
		firstNames, err := q.ArrayAggFirstName(context.Background(), 999)
		require.NoError(t, err)
		require.Nil(t, firstNames)
	})

	t.Run("ArrayAggFirstName - one", func(t *testing.T) {
		firstNames, err := q.ArrayAggFirstName(context.Background(), washingtonID)
		require.NoError(t, err)
		assert.Equal(t, []string{"george"}, firstNames)
	})
}

func insertAuthor(t *testing.T, q *DBQuerier, first, last string) int32 {
	t.Helper()
	authorID, err := q.InsertAuthor(context.Background(), first, last)
	require.NoError(t, err, "insert author")
	return authorID
}
