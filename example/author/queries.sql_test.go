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

func TestNewQuerier_FindAuthorByID_Batch(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()
	q := NewQuerier(conn)
	adamsID := insertAuthor(t, q, "john", "adams")
	insertAuthor(t, q, "george", "washington")

	batch := &pgx.Batch{}
	q.FindAuthorByIDBatch(context.Background(), batch, adamsID)
	results := conn.SendBatch(context.Background(), batch)
	authors, err := q.FindAuthorByIDScan(results)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, FindAuthorByIDRow{
		AuthorID:  adamsID,
		FirstName: "john",
		LastName:  "adams",
		Suffix:    pgtype.Text{Status: pgtype.Null},
	}, authors, "authorByID should match")
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

func TestNewQuerier_FindAuthors_Batch(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()
	q := NewQuerier(conn)
	insertAuthor(t, q, "john", "adams")
	washingtonID := insertAuthor(t, q, "george", "washington")
	carverID := insertAuthor(t, q, "george", "carver")

	batch := &pgx.Batch{}
	q.FindAuthorsBatch(context.Background(), batch, "george")
	results := conn.SendBatch(context.Background(), batch)
	authors, err := q.FindAuthorsScan(results)
	if err != nil {
		t.Fatal(err)
	}

	want := []FindAuthorsRow{
		{AuthorID: washingtonID, FirstName: "george", LastName: "washington", Suffix: pgtype.Text{Status: pgtype.Null}},
		{AuthorID: carverID, FirstName: "george", LastName: "carver", Suffix: pgtype.Text{Status: pgtype.Null}},
	}
	assert.Equal(t, want, authors, "authorByID should match")
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

func TestNewQuerier_DeleteAuthorsByFirstName(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()
	q := NewQuerier(conn)
	insertAuthor(t, q, "john", "adams")
	insertAuthor(t, q, "george", "washington")
	insertAuthor(t, q, "george", "carver")

	_, err := q.DeleteAuthorsByFirstName(context.Background(), "george")
	if err != nil {
		t.Fatal(err)
	}

	authors, err := q.FindAuthors(context.Background(), "george")
	if err != nil {
		t.Fatal(err)
	}
	assert.Empty(t, authors, "no authors should remain with first name of george")
}

func TestNewQuerier_DeleteAuthorsByFirstName_Batch(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()
	q := NewQuerier(conn)
	insertAuthor(t, q, "john", "adams")
	insertAuthor(t, q, "george", "washington")
	insertAuthor(t, q, "george", "carver")

	batch := &pgx.Batch{}
	q.DeleteAuthorsByFirstNameBatch(context.Background(), batch, "george")
	results := conn.SendBatch(context.Background(), batch)
	_, err := q.DeleteAuthorsByFirstNameScan(results)
	if err != nil {
		t.Fatal(err)
	}
	if err := results.Close(); err != nil {
		t.Fatal(err)
	}

	georges, err := q.FindAuthors(context.Background(), "george")
	if err != nil {
		t.Fatal(err)
	}
	assert.Empty(t, georges, "should be no georges")
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
	if err != nil {
		t.Fatal(err)
	}

	_, err = q.DeleteAuthorsByFullName(context.Background(), DeleteAuthorsByFullNameParams{
		FirstName: "george",
		LastName:  "washington",
		Suffix:    "Jr.",
	})
	if err != nil {
		t.Fatal(err)
	}

	authors, err := q.FindAuthors(context.Background(), "george")
	if err != nil {
		t.Fatal(err)
	}
	want := []FindAuthorsRow{
		{
			AuthorID:  washingtonID,
			FirstName: "george",
			LastName:  "washington",
			Suffix:    pgtype.Text{Status: pgtype.Null},
		},
	}
	assert.Equal(t, want, authors, "only one author with first name george should remain")
}

func insertAuthor(t *testing.T, q *DBQuerier, first, last string) int32 {
	t.Helper()
	authorID, err := q.InsertAuthor(context.Background(), first, last)
	if err != nil {
		t.Fatalf("insert author: %s", err)
	}
	return authorID
}
