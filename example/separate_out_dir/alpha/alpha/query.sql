-- name: AlphaNested :one
SELECT 'alpha_nested' as output;

-- name: AlphaCompositeArray :one
SELECT ARRAY[ROW('key')]::alpha[];