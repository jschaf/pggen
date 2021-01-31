CREATE TABLE order_product (
  order_id     int REFERENCES orders,
  product_id   int REFERENCES product
);

