-- FindAuthors finds authors by first name.
-- name: FindAuthors
SELECT * FROM author WHERE first_name = pggen.arg('FirstName');

-- DeleteAuthors deletes authors with a first name of "joe".
-- name: DeleteAuthors
DELETE FROM author where first_name = 'joe';
