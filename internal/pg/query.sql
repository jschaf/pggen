-- name: FindEnumTypes :many
WITH enums AS (
  SELECT enumtypid::int8                                   AS enum_type,
         -- pg_enum row identifier.
         -- The OIDs for pg_enum rows follow a special rule: even-numbered OIDs
         -- are guaranteed to be ordered in the same way as the sort ordering of
         -- their enum type. That is, if two even OIDs belong to the same enum
         -- type, the smaller OID must have the smaller enumsortorder value.
         -- Odd-numbered OID values need bear no relationship to the sort order.
         -- This rule allows the enum comparison routines to avoid catalog
         -- lookups in many common cases. The routines that create and alter enum
         -- types attempt to assign even OIDs to enum values whenever possible.
         array_agg(oid::int8 ORDER BY enumsortorder)       AS enum_oids,
         -- The sort position of this enum value within its enum type. Starts as
         -- 1..n but can be fractional or negative.
         array_agg(enumsortorder ORDER BY enumsortorder)   AS enum_orders,
         -- The textual label for this enum value
         array_agg(enumlabel::text ORDER BY enumsortorder) AS enum_labels
  FROM pg_enum
  GROUP BY pg_enum.enumtypid)
SELECT
  typ.oid           AS oid,
  -- typename: Data type name.
  typ.typname::text AS type_name,
  enum.enum_oids    AS child_oids,
  enum.enum_orders  AS orders,
  enum.enum_labels  AS labels,
  -- typtype: b for a base type, c for a composite type (e.g., a table's
  -- row type), d for a domain, e for an enum type, p for a pseudo-type,
  -- or r for a range type.
  typ.typtype       AS type_kind,
  -- typdefault is null if the type has no associated default value. If
  -- typdefaultbin is not null, typdefault must contain a human-readable
  -- version of the default expression represented by typdefaultbin. If
  -- typdefaultbin is null and typdefault is not, then typdefault is the
  -- external representation of the type's default value, which can be fed
  -- to the type's input converter to produce a constant.
  typ.typdefault    AS default_expr
FROM pg_type typ
  JOIN enums enum ON typ.oid = enum.enum_type
WHERE typ.typisdefined
  AND typ.typtype = 'e'
  AND typ.oid = ANY (pggen.arg('OIDs')::oid[]);

-- name: FindOIDByName :one
SELECT oid
FROM pg_type
WHERE typname::text = pggen.arg('Name');
