package golang

import (
	"fmt"
	"github.com/jschaf/pggen/internal/codegen/golang/gotype"
	"github.com/jschaf/pggen/internal/pg"
	"strconv"
	"strings"
)

// NameCompositeDecoderFunc returns the function name that creates a
// pgtype.ValueTranscoder for the composite type that's used to decode rows
// returned by Postgres.
func NameCompositeDecoderFunc(typ gotype.CompositeType) string {
	return "new" + typ.Name + "Decoder"
}

// NameCompositeEncoderFunc returns the function name that creates a textEncoder
// for the composite type that's used to encode query parameters. This function
// is only necessary for top-level types. Descendant types use the assigner
// functions.
func NameCompositeEncoderFunc(typ gotype.CompositeType) string {
	return "new" + typ.Name + "Encoder"
}

// NameCompositeAssignerFunc returns the function name that create the
// []interface{} array for the composite type so that we can use it with a
// parent encoder function, like NameCompositeEncoderFunc, in the pgtype.Value
// Set call.
func NameCompositeAssignerFunc(typ gotype.CompositeType) string {
	return "assign" + typ.Name + "Composite"
}

const newCompositeTypeDecl = `func newCompositeType(name string, fieldNames []string, vals ...pgtype.ValueTranscoder) *pgtype.CompositeType {
	fields := make([]pgtype.CompositeTypeField, len(fieldNames))
	for i, name := range fieldNames {
		fields[i] = pgtype.CompositeTypeField{Name: name, OID: ignoredOID}
	}
	// Okay to ignore error because it's only thrown when the number of field
	// names does not equal the number of ValueTranscoders.
	rowType, _ := pgtype.NewCompositeTypeValues(name, fields, vals)
	return rowType
}`

var newCompositeTypeDeclarer = NewConstantDeclarer("func::newCompositeType", newCompositeTypeDecl)

// CompositeTypeDeclarer declares a new Go struct to represent a Postgres
// composite type.
type CompositeTypeDeclarer struct {
	comp gotype.CompositeType
}

func NewCompositeTypeDeclarer(comp gotype.CompositeType) CompositeTypeDeclarer {
	return CompositeTypeDeclarer{comp: comp}
}

func (c CompositeTypeDeclarer) DedupeKey() string {
	return "composite::" + c.comp.Name
}

func (c CompositeTypeDeclarer) Declare(pkgPath string) (string, error) {
	sb := &strings.Builder{}
	// Doc string
	if c.comp.PgComposite.Name != "" {
		sb.WriteString("// ")
		sb.WriteString(c.comp.Name)
		sb.WriteString(" represents the Postgres composite type ")
		sb.WriteString(strconv.Quote(c.comp.PgComposite.Name))
		sb.WriteString(".\n")
	}
	// Struct declaration.
	sb.WriteString("type ")
	sb.WriteString(c.comp.Name)
	sb.WriteString(" struct")
	if len(c.comp.FieldNames) == 0 {
		sb.WriteString("{") // type Foo struct{}
	} else {
		sb.WriteString(" {\n") // type Foo struct {\n
	}
	// Struct fields.
	nameLen, typeLen := getLongestNameTypes(c.comp, pkgPath)
	for i, name := range c.comp.FieldNames {
		// Name
		sb.WriteRune('\t')
		sb.WriteString(name)
		// Type
		qualType := c.comp.FieldTypes[i].QualifyRel(pkgPath)
		sb.WriteString(strings.Repeat(" ", nameLen-len(name)))
		sb.WriteString(qualType)
		// JSON struct tag
		sb.WriteString(strings.Repeat(" ", typeLen-len(qualType)))
		sb.WriteString("`json:")
		sb.WriteString(strconv.Quote(c.comp.PgComposite.ColumnNames[i]))
		sb.WriteString("`")
		sb.WriteRune('\n')
	}
	sb.WriteString("}")
	return sb.String(), nil
}

// getLongestNameTypes returns the length of the longest name and type name for
// all child fields of a composite type. Useful for aligning struct definitions.
func getLongestNameTypes(typ gotype.CompositeType, pkgPath string) (int, int) {
	nameLen := 0
	for _, name := range typ.FieldNames {
		if n := len(name); n > nameLen {
			nameLen = n
		}
	}
	nameLen++ // 1 space to separate name from type

	typeLen := 0
	for _, childType := range typ.FieldTypes {
		if n := len(childType.QualifyRel(pkgPath)); n > typeLen {
			typeLen = n
		}
	}
	typeLen++ // 1 space to separate type from struct tags.

	return nameLen, typeLen
}

// CompositeDecoderDeclarer declares a new Go function that creates a pgx
// decoder for the Postgres type represented by the gotype.CompositeType.
type CompositeDecoderDeclarer struct {
	typ gotype.CompositeType
}

func NewCompositeDecoderDeclarer(typ gotype.CompositeType) CompositeDecoderDeclarer {
	return CompositeDecoderDeclarer{typ}
}

func (c CompositeDecoderDeclarer) DedupeKey() string {
	return "composite_decoder::" + c.typ.Name
}

func (c CompositeDecoderDeclarer) Declare(pkgPath string) (string, error) {
	funcName := NameCompositeDecoderFunc(c.typ)
	sb := &strings.Builder{}
	sb.Grow(256)

	// Doc comment
	sb.WriteString("// ")
	sb.WriteString(funcName)
	sb.WriteString(" creates a new decoder for the Postgres '")
	sb.WriteString(c.typ.PgComposite.Name)
	sb.WriteString("' composite type.\n")

	// Function signature
	sb.WriteString("func ")
	sb.WriteString(funcName)
	sb.WriteString("() pgtype.ValueTranscoder {\n\t")

	// newCompositeType call
	sb.WriteString("return newCompositeType(\n\t\t")
	sb.WriteString(strconv.Quote(c.typ.PgComposite.Name))
	sb.WriteString(",\n\t\t")

	// newCompositeType - field names of the composite type
	sb.WriteString(`[]string{`)
	for i := range c.typ.FieldNames {
		sb.WriteByte('"')
		sb.WriteString(c.typ.PgComposite.ColumnNames[i])
		sb.WriteByte('"')
		if i < len(c.typ.FieldNames)-1 {
			sb.WriteString(", ")
		}
	}
	sb.WriteString("},")

	// newCompositeType - child decoders
	for _, fieldType := range c.typ.FieldTypes {
		sb.WriteString("\n\t\t")
		switch fieldType := fieldType.(type) {
		case gotype.CompositeType:
			childFuncName := NameCompositeDecoderFunc(fieldType)
			sb.WriteString(childFuncName)
			sb.WriteString("(),")
		case gotype.EnumType:
			sb.WriteString(NameEnumDecoderFunc(fieldType))
			sb.WriteString("(),")
		case gotype.ArrayType:
			sb.WriteString(NameArrayDecoderFunc(fieldType))
			sb.WriteString("(),")
		case gotype.VoidType:
			// skip
		default:
			sb.WriteString("&") // pgx needs pointers to types
			// TODO: support builtin types and builtin wrappers that use a different
			// initialization syntax.
			pgType := fieldType.PgType()
			if pgType == nil || pgType == (pg.VoidType{}) {
				sb.WriteString("nil,")
			} else {
				// We need the pgx variant because it matches the interface expected by
				// newCompositeType, pgtype.ValueTranscoder.
				if decoderType, ok := gotype.FindKnownTypePgx(pgType.OID()); ok {
					fieldType = decoderType
				}
				sb.WriteString(fieldType.QualifyRel(pkgPath))
				sb.WriteString("{},")
			}
		}
	}
	sb.WriteString("\n\t")
	sb.WriteString(")")
	sb.WriteString("\n")
	sb.WriteString("}")
	return sb.String(), nil
}

// CompositeEncoderDeclarer declares a new Go function that creates a pgx
// Encoder for the Postgres type represented by the gotype.CompositeType.
//
// We need a separate encoder because setting a pgtype.ValueTranscoder is much
// less flexible on the values allowed compared to AssignTo. We can assign a
// pgtype.CompositeType to any struct but we can only set it with an
// []interface{}.
//
// Additionally, we need to use the Postgres text format exclusively because the
// Postgres binary format requires the type OID but pggen doesn't necessarily
// know the OIDs of the types. The text format, however, doesn't require OIDs.
type CompositeEncoderDeclarer struct {
	typ gotype.CompositeType
}

func NewCompositeEncoderDeclarer(typ gotype.CompositeType) CompositeEncoderDeclarer {
	return CompositeEncoderDeclarer{typ}
}

func (c CompositeEncoderDeclarer) DedupeKey() string {
	return "composite_encoder::" + c.typ.Name
}

func (c CompositeEncoderDeclarer) Declare(string) (string, error) {
	funcName := NameCompositeEncoderFunc(c.typ)
	sb := &strings.Builder{}
	sb.Grow(256)

	// Doc comment
	sb.WriteString("// ")
	sb.WriteString(funcName)
	sb.WriteString(" creates a new encoder for the Postgres '")
	sb.WriteString(c.typ.PgComposite.Name)
	sb.WriteString("' composite type query params.\n")

	// Function signature
	sb.WriteString("func ")
	sb.WriteString(funcName)
	sb.WriteString("(p ")
	sb.WriteString(c.typ.Name)
	sb.WriteString(") textEncoder {\n\t")

	// Function body
	sb.WriteString("dec := ")
	sb.WriteString(NameCompositeDecoderFunc(c.typ))
	sb.WriteString("()\n\t")
	sb.WriteString("if err := dec.Set(")
	sb.WriteString(NameCompositeAssignerFunc(c.typ))
	sb.WriteString("(p)); err != nil {\n\t\t")
	sb.WriteString(fmt.Sprintf(`panic("encode %s: " + err.Error())`, c.typ.Name))
	sb.WriteString(" // should always succeed\n\t")
	sb.WriteString("}\n\t")
	sb.WriteString("return textEncoder{ValueTranscoder: dec}\n")
	sb.WriteString("}")
	return sb.String(), nil
}

// CompositeAssignerDeclarer declares a new Go function that returns all fields
// as a generic array: []interface{}. Necessary because we can only set
// pgtype.CompositeType from a []interface{}.
type CompositeAssignerDeclarer struct {
	typ gotype.CompositeType
}

func NewCompositeAssignerDeclarer(typ gotype.CompositeType) CompositeAssignerDeclarer {
	return CompositeAssignerDeclarer{typ}
}

func (c CompositeAssignerDeclarer) DedupeKey() string {
	return "composite_assigner::" + c.typ.Name
}

func (c CompositeAssignerDeclarer) Declare(string) (string, error) {
	funcName := NameCompositeAssignerFunc(c.typ)
	sb := &strings.Builder{}
	sb.Grow(256)

	// Doc comment
	sb.WriteString("// ")
	sb.WriteString(funcName)
	sb.WriteString(" returns all composite fields for the Postgres\n")
	sb.WriteString("// '")
	sb.WriteString(c.typ.PgComposite.Name)
	sb.WriteString("' composite type as a slice of interface{} for use with the\n")
	sb.WriteString("// pgtype.Value Set method.\n")

	// Function signature
	sb.WriteString("func ")
	sb.WriteString(funcName)
	sb.WriteString("(p ")
	sb.WriteString(c.typ.Name)
	sb.WriteString(") []interface{} {\n\t")

	// Function body
	sb.WriteString("return []interface{}{")

	// Field Assigners of the composite type
	for i, fieldType := range c.typ.FieldTypes {
		fieldName := c.typ.FieldNames[i]
		sb.WriteString("\n\t\t")
		switch fieldType := fieldType.(type) {
		case gotype.CompositeType:
			childFuncName := NameCompositeAssignerFunc(fieldType)
			sb.WriteString(childFuncName)
			sb.WriteString("(p.")
			sb.WriteString(fieldName)
			sb.WriteString(")")
		case gotype.ArrayType:
			sb.WriteString(NameArrayAssignerFunc(fieldType))
			sb.WriteString("(p.")
			sb.WriteString(fieldName)
			sb.WriteString(")")
		case gotype.VoidType:
			sb.WriteString("nil") // TODO: does this work?
		default:
			sb.WriteString("p.")
			sb.WriteString(fieldName)
		}
		sb.WriteString(",")
	}
	sb.WriteString("\n\t")
	sb.WriteString("}")
	sb.WriteString("\n")
	sb.WriteString("}")
	return sb.String(), nil
}
