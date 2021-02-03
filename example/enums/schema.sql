CREATE TYPE device_type AS ENUM (
  'undefined',
  'phone',
  'laptop',
  'ipad',
  'desktop',
  'iot'
  );

CREATE TABLE IF NOT EXISTS device (
  mac  MACADDR PRIMARY KEY,
  type device_type NOT NULL DEFAULT 'undefined'
);
