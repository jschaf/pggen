package golang

import (
	"fmt"
	"github.com/jschaf/pggen/internal/codegen/golang/gotype"
	"strconv"
	"strings"
)

// NameArrayTranscoderFunc returns the function name that creates a
// pgtype.ValueTranscoder for the array type that's used to decode rows returned
// by Postgres.
func NameArrayTranscoderFunc(typ *gotype.ArrayType) string {
	return "new" + typ.Elem.BaseName() + "Array"
}

// NameArrayInitFunc returns the name for the function that creates an
// initialized pgtype.ValueTranscoder for the array type that's used to encode
// query parameters. This function is only necessary for top-level types.
// Descendant types use the raw functions, named by NameArrayRawFunc.
func NameArrayInitFunc(typ *gotype.ArrayType) string {
	return "new" + typ.Elem.BaseName() + "ArrayInit"
}

// NameArrayRawFunc returns the function name that create the []interface{}
// array for the array type so that we can use it with a parent encoder
// function, like NameCompositeInitFunc, in the pgtype.Value Set call.
func NameArrayRawFunc(typ *gotype.ArrayType) string {
	return "new" + typ.Elem.BaseName() + "ArrayRaw"
}

// ArrayTranscoderDeclarer declares a new Go function that creates a
// pgtype.ValueTranscoder decoder for an array Postgres type.
type ArrayTranscoderDeclarer struct {
	typ *gotype.ArrayType
}

func NewArrayDecoderDeclarer(typ *gotype.ArrayType) ArrayTranscoderDeclarer {
	return ArrayTranscoderDeclarer{typ: typ}
}

func (a ArrayTranscoderDeclarer) DedupeKey() string {
	return "type_resolver::" + a.typ.BaseName() + "_01_transcoder"
}

func (a ArrayTranscoderDeclarer) Declare(string) (string, error) {
	sb := &strings.Builder{}
	funcName := NameArrayTranscoderFunc(a.typ)

	// Doc comment
	sb.WriteString("// ")
	sb.WriteString(funcName)
	sb.WriteString(" creates a new pgtype.ValueTranscoder for the Postgres\n")
	sb.WriteString("// '")
	sb.WriteString(a.typ.PgArray.Name)
	sb.WriteString("' array type.\n")

	// Function signature
	sb.WriteString("func (tr *typeResolver) ")
	sb.WriteString(funcName)
	sb.WriteString("() pgtype.ValueTranscoder {\n\t")

	// newArrayValue call
	sb.WriteString("return tr.newArrayValue(")
	sb.WriteString(strconv.Quote(a.typ.PgArray.Name))
	sb.WriteString(", ")
	sb.WriteString(strconv.Quote(a.typ.PgArray.Elem.String()))
	sb.WriteString(", ")

	// Default element transcoder
	switch elem := gotype.UnwrapNestedType(a.typ.Elem).(type) {
	case *gotype.CompositeType:
		sb.WriteString("tr.")
		sb.WriteString(NameCompositeTranscoderFunc(elem))
	case *gotype.EnumType:
		sb.WriteString(NameEnumTranscoderFunc(elem))
	default:
		return "", fmt.Errorf("array composite decoder only supports composite and enum elems; got %T", a.typ.Elem)
	}
	sb.WriteString(")")
	sb.WriteString("\n")
	sb.WriteString("}")

	return sb.String(), nil
}

// ArrayInitDeclarer declares a new Go function that creates an *initialized*
// pgtype.ValueTranscoder for the Postgres type represented by the
// gotype.ArrayType.
//
// We need a separate declarer from ArrayTranscoderDeclarer because setting a
// pgtype.ValueTranscoder is much less flexible on the values allowed compared
// to AssignTo. We can assign a pgtype.ArrayType to any struct but we can only
// set it with an [][]interface{} if the array elements are composite types.
//
// Additionally, we need to use the Postgres text format exclusively because the
// Postgres binary format requires the type OID but pggen doesn't necessarily
// know the OIDs of the types. The text format, however, doesn't require OIDs.
type ArrayInitDeclarer struct {
	typ *gotype.ArrayType
}

func NewArrayInitDeclarer(typ *gotype.ArrayType) ArrayInitDeclarer {
	return ArrayInitDeclarer{typ}
}

func (a ArrayInitDeclarer) DedupeKey() string {
	return "type_resolver::" + a.typ.BaseName() + "_02_init"
}

func (a ArrayInitDeclarer) Declare(string) (string, error) {
	funcName := NameArrayInitFunc(a.typ)
	sb := &strings.Builder{}
	sb.Grow(256)

	// Doc comment
	sb.WriteString("// ")
	sb.WriteString(funcName)
	sb.WriteString(" creates an initialized pgtype.ValueTranscoder for the\n")
	sb.WriteString("// Postgres array type '")
	sb.WriteString(a.typ.PgArray.Name)
	sb.WriteString("' to encode query parameters.\n")

	// Function signature
	sb.WriteString("func (tr *typeResolver) ")
	sb.WriteString(funcName)
	sb.WriteString("(ps ")
	sb.WriteString(a.typ.BaseName())
	sb.WriteString(") pgtype.ValueTranscoder {\n\t")

	// Function body
	sb.WriteString("dec := tr.")
	sb.WriteString(NameArrayTranscoderFunc(a.typ))
	sb.WriteString("()\n\t")
	sb.WriteString("if err := dec.Set(tr.")
	sb.WriteString(NameArrayRawFunc(a.typ))
	sb.WriteString("(ps)); err != nil {\n\t\t")
	sb.WriteString(fmt.Sprintf(`panic("encode %s: " + err.Error())`, a.typ.BaseName()))
	sb.WriteString(" // should always succeed\n\t")
	sb.WriteString("}\n\t")
	sb.WriteString("return textPreferrer{ValueTranscoder: dec, typeName: ")
	sb.WriteString(strconv.Quote(a.typ.PgArray.Name))
	sb.WriteString("}\n")
	sb.WriteString("}")
	return sb.String(), nil
}

// ArrayRawDeclarer declares a new Go function that returns all fields
// as a generic array: []interface{}. Necessary because we can only set
// pgtype.ArrayType from a []interface{}.
type ArrayRawDeclarer struct {
	typ *gotype.ArrayType
}

func NewArrayRawDeclarer(typ *gotype.ArrayType) ArrayRawDeclarer {
	return ArrayRawDeclarer{typ}
}

func (a ArrayRawDeclarer) DedupeKey() string {
	return "type_resolver::" + a.typ.BaseName() + "_03_raw"
}

func (a ArrayRawDeclarer) Declare(string) (string, error) {
	funcName := NameArrayRawFunc(a.typ)
	sb := &strings.Builder{}
	sb.Grow(256)

	// Doc comment
	sb.WriteString("// ")
	sb.WriteString(funcName)
	sb.WriteString(" returns all elements for the Postgres array type '")
	sb.WriteString(a.typ.PgArray.Name)
	sb.WriteString("'\n// as a slice of interface{} for use with the pgtype.Value Set method.\n")

	// Function signature
	sb.WriteString("func (tr *typeResolver) ")
	sb.WriteString(funcName)
	sb.WriteString("(vs ")
	sb.WriteString(a.typ.BaseName())
	sb.WriteString(") []interface{} {\n\t")

	// Function body
	sb.WriteString("elems := make([]interface{}, len(vs))\n\t")
	sb.WriteString("for i, v := range vs {\n\t\t")
	sb.WriteString("elems[i] = ")
	switch elem := gotype.UnwrapNestedType(a.typ.Elem).(type) {
	case *gotype.CompositeType:
		sb.WriteString("tr.")
		sb.WriteString(NameCompositeRawFunc(elem))
		sb.WriteString("(v)")
	default:
		sb.WriteString("v")
	}
	sb.WriteString("\n\t")
	sb.WriteString("}\n\t")
	sb.WriteString("return elems\n")
	sb.WriteString("}")
	return sb.String(), nil
}
