CREATE TYPE list_item AS (
  name  text,
  color text
);

CREATE TYPE list_stats AS (
  val1 text,
  val2 int[]
);

CREATE OR REPLACE FUNCTION out_params(
  OUT _items list_item[],
  OUT _stats list_stats
)
  LANGUAGE plpgsql AS $$
BEGIN
  _items := ARRAY [('some_name', 'some_color')::list_item];
  _stats := ('abc', ARRAY [1, 2])::list_stats;
END
$$;
