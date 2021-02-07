-- name: FindDevicesByUser :many
SELECT
  id,
  name,
  (SELECT array_agg(mac) FROM device WHERE owner = id)
FROM "user"
WHERE id = pggen.arg('ID');

-- name: CompositeUser :many
SELECT
  d.mac,
  d.type,
  ROW (u.id, u.name)::"user" AS "user"
FROM device d
  LEFT JOIN "user" u ON u.id = d.owner;

-- name: InsertUser :exec
INSERT INTO "user" (id, name) VALUES (pggen.arg('user_id'), pggen.arg('name'));

-- name: InsertDevice :exec
INSERT INTO device (mac, owner) VALUES (pggen.arg('mac'), pggen.arg('owner'));
