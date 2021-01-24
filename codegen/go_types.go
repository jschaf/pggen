package codegen

import "github.com/jschaf/pggen/internal/pg"

// pgToGoType maps a Postgres type to a Go type.
func pgToGoType(pgType pg.Type) string {
	switch pgType.String() {
	case "bool":
		return "bool"
	case "bytea":
		return "[]byte"
	case "text":
		return "string"
	case "integer", "int4":
		return "int32"
	case "int8", "bigint":
		return "int64"
	default:
		return pgType.String()
	}
}
