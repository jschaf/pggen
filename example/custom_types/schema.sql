-- New base type my_int.
-- https://stackoverflow.com/a/45190420/30900
CREATE TYPE my_int;

CREATE FUNCTION my_int_in(cstring) RETURNS my_int
  LANGUAGE internal
  IMMUTABLE STRICT PARALLEL SAFE AS
'int2in';

CREATE FUNCTION my_int_out(my_int) RETURNS cstring
  LANGUAGE internal
  IMMUTABLE STRICT PARALLEL SAFE AS
'int2out';

CREATE FUNCTION my_int_recv(internal) RETURNS my_int
  LANGUAGE internal
  IMMUTABLE STRICT PARALLEL SAFE AS
'int2recv';

CREATE FUNCTION my_int_send(my_int) RETURNS bytea
  LANGUAGE internal
  IMMUTABLE STRICT PARALLEL SAFE AS
'int2send';

CREATE TYPE my_int (
  INPUT = my_int_in,
  OUTPUT = my_int_out,
  RECEIVE = my_int_recv,
  SEND = my_int_send,
  LIKE = smallint,
  CATEGORY = 'N',
  PREFERRED = FALSE,
  DELIMITER = ',',
  COLLATABLE = FALSE
);
