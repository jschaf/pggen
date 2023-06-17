-- CountAuthors returns the number of authors (zero params).
-- name: CountAuthors :one
SELECT count(*) FROM author;

-- FindAuthorById finds one (or zero) authors by ID (one param).
-- name: FindAuthorByID :one
SELECT * FROM author WHERE author_id = pggen.arg('AuthorID');

-- InsertAuthor inserts an author by name and returns the ID (two params).
-- name: InsertAuthor :one
INSERT INTO author (first_name, last_name)
VALUES (pggen.arg('FirstName'), pggen.arg('LastName'))
RETURNING author_id;

-- DeleteAuthorsByFullName deletes authors by the full name (three params).
-- name: DeleteAuthorsByFullName :exec
DELETE
FROM author
WHERE first_name = pggen.arg('FirstName')
  AND last_name = pggen.arg('LastName')
  AND CASE WHEN pggen.arg('Suffix') = '' THEN suffix IS NULL ELSE suffix = pggen.arg('Suffix') END;