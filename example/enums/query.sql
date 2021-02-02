-- name: FindAllDevices :many
SELECT mac, type from device;

-- name: InsertDevice :exec
INSERT INTO device (mac, type) VALUES (pggen.arg('Mac'), pggen.arg('TypePg'));
