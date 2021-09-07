CREATE TYPE numeric_external_type AS (
  num numeric(8, 2)
);

CREATE TABLE numeric_external (
  num     numeric(10, 6),
  num_arr numeric_external_type[]
);