-- name: GenSeries1 :one
SELECT n from generate_series(0, 2) n LIMIT 1;

-- name: GenSeries :many
SELECT n from generate_series(0, 2) n;

-- name: GenSeriesStr1 :one
SELECT n::text from generate_series(0, 2) n LIMIT 1;

-- name: GenSeriesStr :many
SELECT n::text from generate_series(0, 2) n;
