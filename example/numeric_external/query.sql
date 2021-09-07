-- name: InsertNumeric :exec
INSERT INTO numeric_external (num, num_arr)
VALUES (pggen.arg('num'), pggen.arg('num_arr'));

-- name: FindNumerics :many
SELECT num, num_arr
FROM numeric_external;
