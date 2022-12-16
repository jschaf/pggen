-- FindAuthorById finds one (or zero) authors by ID.
-- name: FindAuthorByID :one
SELECT * FROM author WHERE author_id = pggen.arg('AuthorID');

-- FindAuthors finds authors by first name.
-- name: FindAuthors :many
SELECT * FROM author WHERE first_name = pggen.arg('FirstName');

-- FindAuthorNames finds one (or zero) authors by ID.
-- name: FindAuthorNames :many
SELECT first_name, last_name FROM author ORDER BY author_id = pggen.arg('AuthorID');

-- FindFirstNames finds one (or zero) authors by ID.
-- name: FindFirstNames :many
SELECT first_name FROM author ORDER BY author_id = pggen.arg('AuthorID');

-- DeleteAuthors deletes authors with a first name of "joe".
-- name: DeleteAuthors :exec
DELETE FROM author WHERE first_name = 'joe';

-- DeleteAuthorsByFirstName deletes authors by first name.
-- name: DeleteAuthorsByFirstName :exec
DELETE FROM author WHERE first_name = pggen.arg('FirstName');

-- DeleteAuthorsByFullName deletes authors by the full name.
-- name: DeleteAuthorsByFullName :exec
DELETE
FROM author
WHERE first_name = pggen.arg('FirstName')
  AND last_name = pggen.arg('LastName')
  AND suffix = pggen.arg('Suffix');

-- InsertAuthor inserts an author by name and returns the ID.
-- name: InsertAuthor :one
INSERT INTO author (first_name, last_name)
VALUES (pggen.arg('FirstName'), pggen.arg('LastName'))
RETURNING author_id;

-- InsertAuthorSuffix inserts an author by name and suffix and returns the
-- entire row.
-- name: InsertAuthorSuffix :one
INSERT INTO author (first_name, last_name, suffix)
VALUES (pggen.arg('FirstName'), pggen.arg('LastName'), pggen.arg('Suffix'))
RETURNING author_id, first_name, last_name, suffix;

-- name: StringAggFirstName :one
SELECT string_agg(first_name, ',') AS names FROM author WHERE author_id = pggen.arg('author_id');

-- name: ArrayAggFirstName :one
SELECT array_agg(first_name) AS names FROM author WHERE author_id = pggen.arg('author_id');
