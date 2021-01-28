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
	insertAuthor(t, conn, 77, "john", "adams")
	insertAuthor(t, conn, 78, "george", "washington")

	authorByID, err := q.FindAuthorByID(context.Background(), 77)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, FindAuthorByIDRow{
		AuthorID:  77,
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
	insertAuthor(t, conn, 1, "john", "adams")
	insertAuthor(t, conn, 2, "george", "washington")
	insertAuthor(t, conn, 3, "george", "carver")

	tests := []struct {
		firstName string
		want      []FindAuthorsRow
	}{
		{"john", []FindAuthorsRow{
			{
				FirstName: "john",
				LastName:  "adams",
				Suffix:    pgtype.Text{Status: pgtype.Null},
			},
		}},
		{"george", []FindAuthorsRow{
			{FirstName: "george", LastName: "washington", Suffix: pgtype.Text{Status: pgtype.Null}},
			{FirstName: "george", LastName: "carver", Suffix: pgtype.Text{Status: pgtype.Null}},
		}},
		{"joe", nil},
	}

	for _, tt := range tests {
		t.Run("FindAuthors "+tt.firstName, func(t *testing.T) {
			authors, err := q.FindAuthors(context.Background(), tt.firstName)
			if err != nil {
				t.Error(err)
			}
			// author_id isn't reproducible between runs.
			for i := range authors {
				tt.want[i].AuthorID = authors[i].AuthorID
			}
			assert.Equal(t, tt.want, authors, "expect authors to match expected")
		})
	}
}

func insertAuthor(t *testing.T, conn *pgx.Conn, id int32, first, last string) {
	t.Helper()
	_, err := conn.Exec(context.Background(),
		"INSERT INTO author (author_id,  first_name, last_name) VALUES ($1, $2, $3)",
		id, first, last)
	if err != nil {
		t.Fatalf("insert author: %s", err)
	}
}
