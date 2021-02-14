CREATE TYPE SKU AS (
  sku_id text
);

CREATE TYPE inventory_item AS (
  item_name text,
  sku       SKU
);

CREATE TABLE qux (
  inv_item inventory_item,
  foo  int8
);
