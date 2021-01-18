-- name: FindAuthors
SELECT * FROM author where first_name = sqld.arg('FirstName');

-- name: DeleteAuthors
DELETE FROM author where first_name = 'joe';
