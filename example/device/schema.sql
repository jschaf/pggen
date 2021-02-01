CREATE TYPE device_type AS ENUM (
  'undefined',
  'phone',
  'laptop',
  'ipad',
  'desktop',
  'iot'
);

CREATE TABLE "user" (
  id   bigint PRIMARY KEY,
  name text NOT NULL
);

CREATE TABLE device (
  mac   MACADDR PRIMARY KEY,
  owner bigint REFERENCES "user",
  type  device_type NOT NULL DEFAULT 'undefined'
);
