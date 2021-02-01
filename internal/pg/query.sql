-- name: FindEnumTypes :many
SELECT typ.oid            AS oid,
       -- typename: Data type name.
       typ.typname        AS type_name,
       -- typtype: b for a base type, c for a composite type (e.g., a table's
       -- row type), d for a domain, e for an enum type, p for a pseudo-type,
       -- or r for a range type.
       typ.typtype        AS type_kind,
       -- pg_enum row identifier.
       -- The OIDs for pg_enum rows follow a special rule: even-numbered OIDs
       -- are guaranteed to be ordered in the same way as the sort ordering of
       -- their enum type. That is, if two even OIDs belong to the same enum
       -- type, the smaller OID must have the smaller enumsortorder value.
       -- Odd-numbered OID values need bear no relationship to the sort order.
       -- This rule allows the enum comparison routines to avoid catalog
       -- lookups in many common cases. The routines that create and alter enum
       -- types attempt to assign even OIDs to enum values whenever possible.
       enum.oid           AS enum_oid,
       -- The sort position of this enum value within its enum type. Starts as
       -- 1..n but can be fractional or negative.
       enum.enumsortorder AS enum_order,
       -- The textual label for this enum value
       enum.enumlabel     AS enum_label,
       -- If typelem is not 0 then it identifies another row in pg_type. The
       -- current type can then be subscripted like an array yielding values of
       -- type typelem. A “true” array type is variable length (typlen = -1),
       -- but some fixed-length (typlen > 0) types also have nonzero typelem,
       -- for example name and point. If a fixed-length type has a typelem then
       -- its internal representation must be some number of values of the
       -- typelem data type with no other data. Variable-length array types
       -- have a header defined by the array subroutines.
       typ.typelem        AS elem_type,
       -- If typarray is not 0 then it identifies another row in pg_type, which
       -- is the “true” array type having this type as element
       typ.typarray       AS array_type,
       -- typrelid: If this is a composite type (see typtype), then this column
       -- points to the pg_class entry that defines the corresponding table.
       -- (For a free-standing composite type, the pg_class entry doesn't really
       -- represent a table, but it is needed anyway for the type's pg_attribute
       -- entries to link to.) Zero for non-composite types.
       typ.typrelid       AS composite_type_id,
       -- typnotnull represents a not-null constraint on a type. Used for
       -- domains only.
       typ.typnotnull     AS domain_not_null_constraint,
       -- typbasetype: If this is a domain (see typtype), then typbasetype
       -- identifies the type that this one is based on. Zero if this type is
       -- not a domain.
       typ.typbasetype    AS domain_base_type,
       -- typdefaultbin: If typdefaultbin is not null, it is the nodeToString()
       -- representation of a default expression for the type. This is only
       -- used for domains.
--        typ.typdefaultbin  AS domain_default_expr,
       -- typndims: the number of array dimensions for a domain over an array
       -- (that is, typbasetype is an array type). Zero for types other than
       -- domains over array types.
       typ.typndims       AS num_dimensions,
       -- typdefault is null if the type has no associated default value. If
       -- typdefaultbin is not null, typdefault must contain a human-readable
       -- version of the default expression represented by typdefaultbin. If
       -- typdefaultbin is null and typdefault is not, then typdefault is the
       -- external representation of the type's default value, which can be fed
       -- to the type's input converter to produce a constant.
       typ.typdefault     AS default_expr
FROM pg_type typ
  JOIN pg_enum enum ON typ.oid = enum.enumtypid
WHERE typ.typisdefined
  AND typ.typtype = 'e'
  AND typ.oid = ANY (pggen.arg('OIDs')::oid[])
ORDER BY typ.oid DESC;
