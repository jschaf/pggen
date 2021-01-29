-- name: FindOrdersByPrice :many
SELECT * FROM orders WHERE order_total > pggen.arg('MinTotal');
