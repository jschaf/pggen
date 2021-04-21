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

func NameArrayEncoderFunc(typ gotype.ArrayType) string {
	return "encode" + typ.Name
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

// ArrayEncoderDeclarer declares a new Go function that creates a pgx
// Encoder for the Postgres type represented by the gotype.ArrayType.
//
// We need a separate encoder because setting a pgtype.ValueTranscoder is much
// less flexible on the values allowed compared to AssignTo. We can assign a
// pgtype.ArrayType to any struct but we can only set it with an
// []interface{}.
//
// Additionally, we need to use the Postgres text format exclusively because the
// Postgres binary format requires the type OID but pggen doesn't necessarily
// know the OIDs of the types. The text format, however, doesn't require OIDs.
type ArrayEncoderDeclarer struct {
	typ gotype.ArrayType
}

func NewArrayEncoderDeclarer(typ gotype.ArrayType) ArrayEncoderDeclarer {
	return ArrayEncoderDeclarer{typ}
}

func (c ArrayEncoderDeclarer) DedupeKey() string {
	return "array_encoder::" + c.typ.Name
}

func (c ArrayEncoderDeclarer) Declare(string) (string, error) {
	funcName := NameArrayEncoderFunc(c.typ)
	sb := &strings.Builder{}
	sb.Grow(256)

	// Doc comment
	sb.WriteString("// ")
	sb.WriteString(funcName)
	sb.WriteString(" creates a new encoder for the Postgres '")
	sb.WriteString(c.typ.PgArray.Name)
	sb.WriteString("' array type query params.\n")

	// Function signature
	sb.WriteString("func ")
	sb.WriteString(funcName)
	sb.WriteString("(p ")
	sb.WriteString(c.typ.Name)
	sb.WriteString(") textEncoder {\n\t")

	// Function body
	sb.WriteString("dec := ")
	sb.WriteString(funcName)
	sb.WriteString("()\n\t")
	sb.WriteString("dec.Set([]interface[}{")

	// TODO: Create interface array containing each element.

	sb.WriteString("\n\t")
	sb.WriteString("})")
	sb.WriteString("\n")
	sb.WriteString("}")
	return sb.String(), nil
}
