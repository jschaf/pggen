-- name: FindOrdersByPrice :many
SELECT * FROM orders WHERE order_total > pggen.arg('MinTotal');

-- name: FindOrdersMRR :many
SELECT date_trunc('month', order_date) AS month, sum(order_total) AS order_mrr
FROM orders
GROUP BY date_trunc('month', order_date);
