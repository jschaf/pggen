-- name: CustomTypes :one
SELECT 'some_text', 1::bigint;

-- name: CustomMyInt :one
SELECT '5'::my_int as int5;

-- name: IntArray :many
SELECT ARRAY ['5', '6', '7']::int[] as ints;
