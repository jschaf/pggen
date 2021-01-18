-- FindAuthors finds authors by first name.
-- name: FindAuthors
SELECT * FROM author where first_name = sqld.arg('FirstName');

-- DeleteAuthors deletes authors with a first name of "joe".
-- name: DeleteAuthors
DELETE FROM author where first_name = 'joe';
