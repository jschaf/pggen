-- name: GenSeries1 :one
SELECT n from generate_series(0, 2) n LIMIT 1;

-- name: GenSeries :many
SELECT n from generate_series(0, 2) n;