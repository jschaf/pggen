CREATE TYPE dimensions AS (
  width  int4,
  height int4
);

CREATE TYPE product_image_type AS (
  source text,
  dimensions dimensions
);

CREATE TYPE product_image_set_type AS (
  name       text,
  orig_image product_image_type,
  images     product_image_type[]
);
