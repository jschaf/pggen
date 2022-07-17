CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE screenshots (
  id bigint PRIMARY KEY
);

CREATE TABLE blocks (
  id            serial PRIMARY KEY,
  screenshot_id bigint NOT NULL REFERENCES screenshots (id),
  body          text NOT NULL
);

CREATE TYPE arrays AS (
  texts  text[],
  int8s  int8[],
  bools  boolean[],
  floats float8[]
);


CREATE TYPE user_email AS (
  id text,
  email citext
);
