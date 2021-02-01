-- name: FindDevicesByUser :many
SELECT
  id,
  name,
  (SELECT array_agg(mac) FROM device WHERE owner = id)
FROM "user"
WHERE id = pggen.arg('ID');
