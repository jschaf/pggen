-- name: FindOrdersByCustomer :many
SELECT * FROM orders WHERE customer_id = pggen.arg('CustomerID');

-- name: FindProductsInOrder :many
SELECT o.order_id, p.product_id, p.name
FROM orders o
  INNER JOIN order_product op USING (order_id)
  INNER JOIN product p USING (product_id)
WHERE o.order_id = pggen.arg('OrderID');
