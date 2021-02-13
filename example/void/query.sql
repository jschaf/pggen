-- name: VoidOnly :exec
SELECT void_fn();

-- name: VoidOnlyTwoParams :exec
SELECT void_fn_two_params(pggen.arg('id'), 'text');

-- name: VoidTwo :one
SELECT void_fn(), 'foo' as name;

-- name: VoidThree :one
SELECT void_fn(), 'foo' as foo, 'bar' as bar;

-- name: VoidThree2 :many
SELECT 'foo' as foo, void_fn(), void_fn();
