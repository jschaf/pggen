-- This schema file exists solely so IntelliJ doesn't underline every
-- pggen.arg() expression in squiggly red.
CREATE SCHEMA pggen;

-- pggen.arg defines a named parameter that's eventually compiled into a
-- placeholder for a prepared query: $1, $2, etc.
CREATE FUNCTION pggen.arg(param text) RETURNS any AS
'';
