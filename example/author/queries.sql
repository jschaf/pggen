-- FindAuthorById finds one (or zero) authors by ID.
-- name: FindAuthorByID :one
SELECT * FROM author WHERE author_id = pggen.arg('AuthorID');

-- FindAuthors finds authors by first name.
-- name: FindAuthors :many
SELECT * FROM author WHERE first_name = pggen.arg('FirstName');

-- DeleteAuthors deletes authors with a first name of "joe".
-- name: DeleteAuthors :exec
DELETE FROM author WHERE first_name = 'joe';

-- InsertAuthor inserts an author by name and returns the ID.
-- name: InsertAuthor :one
INSERT INTO author (first_name, last_name)
VALUES (pggen.arg('FirstName'), pggen.arg('LastName'))
RETURNING author_id;
