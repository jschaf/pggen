-- name: GenSeries1 :one
SELECT n
FROM generate_series(0, 2) n
LIMIT 1;

-- name: GenSeries :many
SELECT n
FROM generate_series(0, 2) n;

-- name: GenSeriesArr1 :one
SELECT array_agg(n)
FROM generate_series(0, 2) n;

-- name: GenSeriesArr :many
SELECT array_agg(n)
FROM generate_series(0, 2) n;

-- name: GenSeriesStr1 :one
SELECT n::text
FROM generate_series(0, 2) n
LIMIT 1;

-- name: GenSeriesStr :many
SELECT n::text
FROM generate_series(0, 2) n;
