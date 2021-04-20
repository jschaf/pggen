package golang

import (
	"fmt"
	"github.com/jschaf/pggen/internal/codegen/golang/gotype"
	"strconv"
	"strings"
)

func NameArrayDecoderFunc(typ gotype.ArrayType) string {
	return "new" + strings.TrimPrefix(typ.Name, "[]") + "ArrayDecoder"
}

// ArrayDecoderDeclarer declares a new Go function that creates a pgx
// decoder for an array Postgres type.
type ArrayDecoderDeclarer struct {
	typ gotype.ArrayType
}

func NewArrayDecoderDeclarer(typ gotype.ArrayType) ArrayDecoderDeclarer {
	return ArrayDecoderDeclarer{typ: typ}
}

func (e ArrayDecoderDeclarer) DedupeKey() string {
	return "array_decoder::" + e.typ.Name
}

func (e ArrayDecoderDeclarer) Declare(string) (string, error) {
	sb := &strings.Builder{}
	funcName := NameArrayDecoderFunc(e.typ)

	// Doc comment
	sb.WriteString("// ")
	sb.WriteString(funcName)
	sb.WriteString(" creates a new decoder for the Postgres '")
	sb.WriteString(e.typ.PgArray.Name)
	sb.WriteString("' array type.\n")

	// Function signature
	sb.WriteString("func ")
	sb.WriteString(funcName)
	sb.WriteString("() pgtype.ValueTranscoder {\n\t")

	// NewArrayType call
	sb.WriteString("return pgtype.NewArrayType(")
	sb.WriteString(strconv.Quote(e.typ.PgArray.Name))
	sb.WriteString(", ")
	sb.WriteString("ignoredOID")
	sb.WriteString(", ")

	// Elem decoder
	switch elem := e.typ.Elem.(type) {
	case gotype.CompositeType:
		sb.WriteString(NameCompositeDecoderFunc(elem))
	case gotype.EnumType:
		sb.WriteString(NameEnumDecoderFunc(elem))
	default:
		return "", fmt.Errorf("array composite decoder only supports composite and enum elems; got %T", e.typ.Elem)
	}
	sb.WriteString(")")
	sb.WriteString("\n")
	sb.WriteString("}")

	return sb.String(), nil
}
