-- name: FindAllDevices :many
SELECT mac, type
FROM device;

-- name: InsertDevice :exec
INSERT INTO device (mac, type)
VALUES (pggen.arg('Mac'), pggen.arg('TypePg'));

-- Select an array of all device_type enum values.
-- name: FindOneDeviceArray :one
SELECT enum_range(NULL::device_type) AS device_types;

-- Select many rows of device_type enum values.
-- name: FindManyDeviceArray :many
SELECT enum_range('ipad'::device_type, 'iot'::device_type) AS device_types
UNION ALL
SELECT enum_range(NULL::device_type) AS device_types;

-- Select many rows of device_type enum values with multiple output columns.
-- name: FindManyDeviceArrayWithNum :many
SELECT 1 AS num, enum_range('ipad'::device_type, 'iot'::device_type) AS device_types
UNION ALL
SELECT 2 as num, enum_range(NULL::device_type) AS device_types;

-- Regression test for https://github.com/jschaf/pggen/issues/23.
-- name: EnumInsideComposite :one
SELECT ROW('08:00:2b:01:02:03'::macaddr, 'phone'::device_type) ::device;
