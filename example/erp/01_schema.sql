CREATE TABLE customer (
  customer_id serial PRIMARY KEY,
  first_name  text NOT NULL,
  last_name   text NOT NULL,
  email       text NOT NULL
);

CREATE TABLE orders (
  order_id    serial PRIMARY KEY,
  order_date  timestamptz NOT NULL,
  order_total numeric     NOT NULL,
  customer_id int REFERENCES customer
);

CREATE TABLE product (
  product_id  serial PRIMARY KEY,
  name        text    NOT NULL,
  description text    NOT NULL,
  list_price  numeric NOT NULL
);
