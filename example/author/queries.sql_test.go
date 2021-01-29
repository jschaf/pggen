// +build example

package author

import (
	"context"
	"errors"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/jschaf/pggen/internal/pgtest"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewQuerier_FindAuthorByID(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()
	q := NewQuerier(conn)
	adamsID := insertAuthor(t, q, "john", "adams")
	insertAuthor(t, q, "george", "washington")

	authorByID, err := q.FindAuthorByID(context.Background(), adamsID)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, FindAuthorByIDRow{
		AuthorID:  adamsID,
		FirstName: "john",
		LastName:  "adams",
		Suffix:    pgtype.Text{Status: pgtype.Null},
	}, authorByID, "authorByID should match")

	missingAuthorByID, err := q.FindAuthorByID(context.Background(), 888)
	if err == nil {
		t.Fatal("expected error when finding author ID that doesn't match")
	}
	if missingAuthorByID != (FindAuthorByIDRow{}) {
		t.Fatal("expected zero value when error")
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("expected no rows error to wrap pgx.ErrNoRows; got %s", err)
	}
}

func TestNewQuerier_FindAuthors(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()
	q := NewQuerier(conn)
	adamsID := insertAuthor(t, q, "john", "adams")
	washingtonID := insertAuthor(t, q, "george", "washington")
	carverID := insertAuthor(t, q, "george", "carver")

	tests := []struct {
		firstName string
		want      []FindAuthorsRow
	}{
		{"john", []FindAuthorsRow{
			{
				AuthorID:  adamsID,
				FirstName: "john",
				LastName:  "adams",
				Suffix:    pgtype.Text{Status: pgtype.Null},
			},
		}},
		{"george", []FindAuthorsRow{
			{AuthorID: washingtonID, FirstName: "george", LastName: "washington", Suffix: pgtype.Text{Status: pgtype.Null}},
			{AuthorID: carverID, FirstName: "george", LastName: "carver", Suffix: pgtype.Text{Status: pgtype.Null}},
		}},
		{"joe", nil},
	}

	for _, tt := range tests {
		t.Run("FindAuthors "+tt.firstName, func(t *testing.T) {
			authors, err := q.FindAuthors(context.Background(), tt.firstName)
			if err != nil {
				t.Error(err)
			}
			assert.Equal(t, tt.want, authors, "expect authors to match expected")
		})
	}
}

func TestNewQuerier_InsertAuthorSuffix(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()
	q := NewQuerier(conn)
	author, err := q.InsertAuthorSuffix(context.Background(), InsertAuthorSuffixParams{
		FirstName: "john",
		LastName:  "adams",
		Suffix:    "Jr.",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := InsertAuthorSuffixRow{
		AuthorID:  author.AuthorID,
		FirstName: "john",
		LastName:  "adams",
		Suffix:    pgtype.Text{Status: pgtype.Present, String: "Jr."},
	}
	assert.Equal(t, want, author, "InsertAuthorSuffix should match")
}

func insertAuthor(t *testing.T, q *DBQuerier, first, last string) int32 {
	t.Helper()
	authorID, err := q.InsertAuthor(context.Background(), first, last)
	if err != nil {
		t.Fatalf("insert author: %s", err)
	}
	return authorID
}
