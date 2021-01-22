package pginfer

import "github.com/jschaf/sqld/internal/pg"

type GoType interface {
	goType()
}

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
