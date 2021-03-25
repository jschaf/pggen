-- name: FindOrdersByCustomer :many
SELECT *
FROM orders
WHERE customer_id = pggen.arg('CustomerID');

-- name: FindProductsInOrder :many
SELECT o.order_id, p.product_id, p.name
FROM orders o
  INNER JOIN order_product op USING (order_id)
  INNER JOIN product p USING (product_id)
WHERE o.order_id = pggen.arg('OrderID');

-- name: InsertCustomer :one
INSERT INTO customer (first_name, last_name, email)
VALUES (pggen.arg('first_name'), pggen.arg('last_name'), pggen.arg('email'))
RETURNING *;

-- name: InsertOrder :one
INSERT INTO orders (order_date, order_total, customer_id)
VALUES (pggen.arg('order_date'), pggen.arg('order_total'), pggen.arg('cust_id'))
RETURNING *;
