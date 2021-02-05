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
  ROW (u.id, u.name)::"user"
FROM device d
  LEFT JOIN "user" u ON u.id = d.owner;
