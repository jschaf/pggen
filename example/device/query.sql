-- name: FindDevicesByUser :many
SELECT
  id,
  name,
  (SELECT array_agg(mac) FROM device WHERE owner = id) AS mac_addrs
FROM "user"
WHERE id = pggen.arg('ID');

-- name: CompositeUser :many
SELECT
  d.mac,
  d.type,
  ROW (u.id, u.name)::"user" AS "user"
FROM device d
  LEFT JOIN "user" u ON u.id = d.owner;

-- name: CompositeUserOne :one
SELECT ROW (15, 'qux')::"user" AS "user";

-- name: CompositeUserOneTwoCols :one
SELECT 1 AS num, ROW (15, 'qux')::"user" AS "user";

-- name: CompositeUserMany :many
SELECT ROW (15, 'qux')::"user" AS "user";

-- name: InsertUser :exec
INSERT INTO "user" (id, name)
VALUES (pggen.arg('user_id'), pggen.arg('name'));

-- name: InsertDevice :exec
INSERT INTO device (mac, owner)
VALUES (pggen.arg('mac'), pggen.arg('owner'));
