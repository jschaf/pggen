-- name: GetBools :one
SELECT pggen.arg('data')::boolean[];

-- name: GetOneTimestamp :one
SELECT pggen.arg('data')::timestamp;

-- name: GetManyTimestamptzs :many
SELECT *
FROM unnest(pggen.arg('data')::timestamptz[]);

-- name: GetManyTimestamps :many
SELECT *
FROM unnest(pggen.arg('data')::timestamp[]);
