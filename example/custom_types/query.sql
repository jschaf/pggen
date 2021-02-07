-- name: CustomTypes :one
SELECT 'some_text', 1::bigint;

-- name: CustomMyInt :one
SELECT '5'::my_int as int5;
