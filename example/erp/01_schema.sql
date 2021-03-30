CREATE DOMAIN js_int AS bigint CHECK ( 0 < value AND value < 9007199254740991 );
-- tenant_id should be 3-5 chars in base 36.
CREATE DOMAIN tenant_id AS js_int CHECK ( 36 * 36 * 36 < value AND value < 36 * 36 * 36 * 36 * 36 );

CREATE TABLE tenant (
  tenant_id tenant_id PRIMARY KEY,
  rname     text UNIQUE GENERATED ALWAYS AS ( 'tenants/' || tenant_id::text ) STORED,
  name      text NOT NULL CHECK ( name != '' )
);

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

CREATE OR REPLACE FUNCTION base36_encode(
  IN digits bigint
) RETURNS text
AS
$$
DECLARE
  chars char[];
  ret   text;
  val   bigint;
BEGIN
  chars :=
    ARRAY ['0','1','2','3','4','5','6','7','8','9','a','b','c','d','e','f','g','h','i','j','k','l','m','n','o','p','q','r','s','t','u','v','w','x','y','z'];
  val := digits;
  ret := '';
  IF val < 0 THEN
    val := val * -1;
  END IF;
  WHILE val != 0
    LOOP
      ret := chars[(val % 36) + 1] || ret;
      val := val / 36;
    END LOOP;
  RETURN ret;
END;
$$ LANGUAGE plpgsql IMMUTABLE
                    PARALLEL SAFE;


CREATE OR REPLACE FUNCTION base36_decode(
  IN base36 text
)
  RETURNS bigint
AS
$$
DECLARE
  a     char[];
  ret   bigint;
  i     int;
  val   int;
  chars text;
BEGIN
  -- Check for null so pggen can pass in null.
  IF base36 IS NULL THEN
    RETURN 0;
  END IF;
  chars := '0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ';
  FOR i IN REVERSE char_length(base36)..1
    LOOP
      a := a || substring(upper(base36) FROM i FOR 1)::char;
    END LOOP;
  i := 0;
  ret := 0;
  WHILE i < (array_length(a, 1))
    LOOP
      val := position(a[i + 1] IN chars) - 1;
      ret := ret + (val * (36 ^ i));
      i := i + 1;
    END LOOP;
  RETURN ret;
END;
$$ LANGUAGE plpgsql IMMUTABLE
                    PARALLEL SAFE;
