-- name: FindEnumTypes :many
WITH enums AS (
  SELECT
    enumtypid::int8                                   AS enum_type,
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

-- name: FindArrayTypes :many
SELECT
  arr_typ.oid           AS oid,
  -- typename: Data type name.
  arr_typ.typname::text AS type_name,
  elem_typ.oid          AS elem_oid,
  -- typtype: b for a base type, c for a composite type (e.g., a table's
  -- row type), d for a domain, e for an enum type, p for a pseudo-type,
  -- or r for a range type.
  arr_typ.typtype       AS type_kind
FROM pg_type arr_typ
  JOIN pg_type elem_typ ON arr_typ.typelem = elem_typ.oid
WHERE arr_typ.typisdefined
  AND arr_typ.typtype = 'b' -- Array types are base types
  -- If typelem is not 0 then it identifies another row in pg_type. The current
  -- type can then be subscripted like an array yielding values of type typelem.
  -- A “true” array type is variable length (typlen = -1), but some
  -- fixed-length (typlen > 0) types also have nonzero typelem, for example
  -- name and point. If a fixed-length type has a typelem then its internal
  -- representation must be some number of values of the typelem data type with
  -- no other data. Variable-length array types have a header defined by the
  -- array subroutines.
  AND arr_typ.typelem > 0
  -- For a fixed-size type, typlen is the number of bytes in the internal
  -- representation of the type. But for a variable-length type, typlen is
  -- negative. -1 indicates a "varlena" type (one that has a length word), -2
  -- indicates a null-terminated C string.
  AND arr_typ.typlen = -1
  AND arr_typ.oid = ANY (pggen.arg('OIDs')::oid[]);


-- A composite type represents a row or record, defined implicitly for each
-- table, or explicitly with CREATE TYPE.
-- https://www.postgresql.org/docs/13/rowtypes.html
-- name: FindCompositeTypes :many
WITH table_cols AS (
  SELECT
    cls.relname                                         AS table_name,
    cls.oid                                             AS table_oid,
    array_agg(attr.attname::text ORDER BY attr.attnum)  AS col_names,
    array_agg(attr.atttypid::int8 ORDER BY attr.attnum) AS col_oids,
    array_agg(attr.attnum::int8 ORDER BY attr.attnum)   AS col_orders,
    array_agg(attr.attnotnull ORDER BY attr.attnum)     AS col_not_nulls,
    array_agg(typ.typname::text ORDER BY attr.attnum)   AS col_type_names
  FROM pg_attribute attr
    JOIN pg_class cls ON attr.attrelid = cls.oid
    JOIN pg_type typ ON typ.oid = attr.atttypid
  WHERE attr.attnum > 0 -- Postgres represents system columns with attnum <= 0
    AND NOT attr.attisdropped
  GROUP BY cls.relname, cls.oid
)
SELECT
  typ.typname::text AS table_type_name,
  typ.oid           AS table_type_oid,
  table_name,
  col_names,
  col_oids,
  col_orders,
  col_not_nulls,
  col_type_names
FROM pg_type typ
  JOIN table_cols cols ON typ.typrelid = cols.table_oid
WHERE typ.oid = ANY (pggen.arg('oids')::oid[])
  AND typ.typtype = 'c';

-- Recursively expands all given OIDs to all descendants through composite
-- types.
-- name: FindDescendantOIDs :many
WITH RECURSIVE oid_descs(oid) AS (
  SELECT oid
  FROM unnest(pggen.arg('oids')::oid[]) AS t(oid)
  UNION
  -- Recursive case.
  SELECT oid
  FROM (
    WITH all_oids AS (SELECT oid FROM oid_descs)
    -- All composite children.
    SELECT attr.atttypid AS oid
    FROM pg_type typ
      JOIN pg_class cls ON typ.oid = cls.reltype
      JOIN pg_attribute attr ON attr.attrelid = cls.oid
      JOIN all_oids od ON typ.oid = od.oid
    WHERE attr.attnum > 0 -- Postgres represents system columns with attnum <= 0
      AND NOT attr.attisdropped
    UNION
    -- All array elements.
    SELECT elem_typ.oid
    FROM pg_type arr_typ
      JOIN pg_type elem_typ ON arr_typ.typelem = elem_typ.oid
      JOIN all_oids od ON arr_typ.oid = od.oid
  ) t
)
SELECT oid
FROM oid_descs;

-- name: FindOIDByName :one
SELECT oid
FROM pg_type
WHERE typname::text = pggen.arg('name')
ORDER BY oid DESC
LIMIT 1;

-- name: FindOIDName :one
SELECT typname AS name
FROM pg_type
WHERE oid = pggen.arg('oid');

-- name: FindOIDNames :many
SELECT oid, typname AS name, typtype AS kind
FROM pg_type
WHERE oid = ANY (pggen.arg('oid')::oid[]);
