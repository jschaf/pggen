-- FindAuthorById finds one (or zero) authors by ID.
-- name: FindAuthorByID :one
SELECT * FROM author WHERE author_id = pggen.arg('AuthorID');

-- FindAuthors finds authors by first name.
-- name: FindAuthors :many
SELECT * FROM author WHERE first_name = pggen.arg('FirstName');

-- DeleteAuthors deletes authors with a first name of "joe".
-- name: DeleteAuthors :exec
DELETE FROM author WHERE first_name = 'joe';
