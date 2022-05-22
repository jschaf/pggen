package golang

import (
	"github.com/jschaf/pggen/internal/codegen/golang/gotype"
	"github.com/jschaf/pggen/internal/pg"
	"strconv"
	"strings"
)

// NameCompositeTranscoderFunc returns the function name that creates a
// pgtype.ValueTranscoder for the composite type that's used to decode rows
// returned by Postgres.
func NameCompositeTranscoderFunc(typ *gotype.CompositeType) string {
	return "new" + typ.Name
}

// NameCompositeInitFunc returns the name of the function that creates an
// initialized pgtype.ValueTranscoder for the composite type used as a query
// parameters. This function is only necessary for top-level types. Descendant
// types use the raw functions, named by NameCompositeRawFunc.
func NameCompositeInitFunc(typ *gotype.CompositeType) string {
	return "new" + typ.Name + "Init"
}

// NameCompositeRawFunc returns the function name that creates the
// []interface{} array for the composite type so that we can use it with a
// parent encoder function, like NameCompositeInitFunc, in the pgtype.Value
// Set call.
func NameCompositeRawFunc(typ *gotype.CompositeType) string {
	return "new" + typ.Name + "Raw"
}

// CompositeTypeDeclarer declares a new Go struct to represent a Postgres
// composite type.
type CompositeTypeDeclarer struct {
	comp *gotype.CompositeType
}

func NewCompositeTypeDeclarer(comp *gotype.CompositeType) CompositeTypeDeclarer {
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
		qualType := gotype.QualifyType(c.comp.FieldTypes[i], pkgPath)
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
func getLongestNameTypes(typ *gotype.CompositeType, pkgPath string) (int, int) {
	nameLen := 0
	for _, name := range typ.FieldNames {
		if n := len(name); n > nameLen {
			nameLen = n
		}
	}
	nameLen++ // 1 space to separate name from type

	typeLen := 0
	for _, childType := range typ.FieldTypes {
		if n := len(gotype.QualifyType(childType, pkgPath)); n > typeLen {
			typeLen = n
		}
	}
	typeLen++ // 1 space to separate type from struct tags.

	return nameLen, typeLen
}

// CompositeTranscoderDeclarer declares a new Go function that creates a pgx
// decoder for the Postgres type represented by the gotype.CompositeType.
type CompositeTranscoderDeclarer struct {
	typ *gotype.CompositeType
}

func NewCompositeTranscoderDeclarer(typ *gotype.CompositeType) CompositeTranscoderDeclarer {
	return CompositeTranscoderDeclarer{typ}
}

func (c CompositeTranscoderDeclarer) DedupeKey() string {
	return "type_resolver::" + c.typ.Name + "_01_transcoder"
}

func (c CompositeTranscoderDeclarer) Declare(pkgPath string) (string, error) {
	funcName := NameCompositeTranscoderFunc(c.typ)
	sb := &strings.Builder{}
	sb.Grow(256)

	// Doc comment
	sb.WriteString("// ")
	sb.WriteString(funcName)
	sb.WriteString(" creates a new pgtype.ValueTranscoder for the Postgres\n")
	sb.WriteString("// composite type '")
	sb.WriteString(c.typ.PgComposite.Name)
	sb.WriteString("'.\n")

	// Function signature
	sb.WriteString("func (tr *typeResolver) ")
	sb.WriteString(funcName)
	sb.WriteString("() pgtype.ValueTranscoder {\n\t")

	// newCompositeValue call
	sb.WriteString("return tr.newCompositeValue(\n\t\t")
	sb.WriteString(strconv.Quote(c.typ.PgComposite.Name))
	sb.WriteString(",")

	// newCompositeValue - field names of the composite type
	for i := range c.typ.FieldNames {
		sb.WriteString("\n\t\t")
		sb.WriteString(`compositeField{`)
		sb.WriteString(strconv.Quote(c.typ.PgComposite.ColumnNames[i])) // field name
		sb.WriteString(", ")
		sb.WriteString(strconv.Quote(c.typ.PgComposite.ColumnTypes[i].String())) // field type name
		sb.WriteString(", ")

		// field default pgtype.ValueTranscoder
		switch fieldType := gotype.UnwrapNestedType(c.typ.FieldTypes[i]).(type) {
		case *gotype.CompositeType:
			childFuncName := NameCompositeTranscoderFunc(fieldType)
			sb.WriteString("tr.")
			sb.WriteString(childFuncName)
			sb.WriteString("()")
		case *gotype.EnumType:
			sb.WriteString(NameEnumTranscoderFunc(fieldType))
			sb.WriteString("()")
		case *gotype.ArrayType:
			sb.WriteString("tr.")
			sb.WriteString(NameArrayTranscoderFunc(fieldType))
			sb.WriteString("()")
		case *gotype.VoidType:
			// skip
		default:
			sb.WriteString("&") // pgx needs pointers to types
			// TODO: support builtin types and builtin wrappers that use a different
			// initialization syntax.
			pgType := c.typ.PgComposite.ColumnTypes[i]
			if pgType == nil || pgType == (pg.VoidType{}) {
				sb.WriteString("nil,")
			} else {
				// We need the pgx variant because it matches the interface expected by
				// newCompositeValue, pgtype.ValueTranscoder.
				if decoderType, ok := gotype.FindKnownTypePgx(pgType.OID()); ok {
					fieldType = decoderType
				}

				sb.WriteString(gotype.QualifyType(fieldType, pkgPath))
				sb.WriteString("{}")
			}
		}
		sb.WriteString(`},`)
	}

	sb.WriteString("\n\t")
	sb.WriteString(")")
	sb.WriteString("\n")
	sb.WriteString("}")
	return sb.String(), nil
}

// CompositeInitDeclarer declares a new Go function that creates an initialized
// pgtype.ValueTranscoder for the Postgres type represented by the
// gotype.CompositeType.
//
// We need a separate encoder because setting a pgtype.ValueTranscoder is much
// less flexible on the values allowed compared to AssignTo. We can assign a
// pgtype.CompositeType to any struct but we can only set it with an
// []interface{}.
//
// Additionally, we need to use the Postgres text format exclusively because the
// Postgres binary format requires the type OID but pggen doesn't necessarily
// know the OIDs of the types. The text format, however, doesn't require OIDs.
type CompositeInitDeclarer struct {
	typ *gotype.CompositeType
}

func NewCompositeInitDeclarer(typ *gotype.CompositeType) CompositeInitDeclarer {
	return CompositeInitDeclarer{typ}
}

func (c CompositeInitDeclarer) DedupeKey() string {
	return "type_resolver::" + c.typ.Name + "_02_init"
}

func (c CompositeInitDeclarer) Declare(string) (string, error) {
	funcName := NameCompositeInitFunc(c.typ)
	sb := &strings.Builder{}
	sb.Grow(256)

	// Doc comment
	sb.WriteString("// ")
	sb.WriteString(funcName)
	sb.WriteString(" creates an initialized pgtype.ValueTranscoder for the\n")
	sb.WriteString("// Postgres composite type '")
	sb.WriteString(c.typ.PgComposite.Name)
	sb.WriteString("' to encode query parameters.\n")

	// Function signature
	sb.WriteString("func (tr *typeResolver) ")
	sb.WriteString(funcName)
	sb.WriteString("(v ")
	sb.WriteString(c.typ.Name)
	sb.WriteString(") pgtype.ValueTranscoder {\n\t")

	// Function body
	sb.WriteString("return tr.setValue(tr.")
	sb.WriteString(NameCompositeTranscoderFunc(c.typ))
	sb.WriteString("(), tr.")
	sb.WriteString(NameCompositeRawFunc(c.typ))
	sb.WriteString("(v))\n")
	sb.WriteString("}")
	return sb.String(), nil
}

// CompositeRawDeclarer declares a new Go function that returns all fields
// of a composite type as a generic array: []interface{}. Necessary because we
// can only set pgtype.CompositeType from a []interface{}.
//
// Revisit after https://github.com/jackc/pgtype/pull/100 to see if we can
// simplify
type CompositeRawDeclarer struct {
	typ *gotype.CompositeType
}

func NewCompositeRawDeclarer(typ *gotype.CompositeType) CompositeRawDeclarer {
	return CompositeRawDeclarer{typ}
}

func (c CompositeRawDeclarer) DedupeKey() string {
	return "type_resolver::" + c.typ.Name + "_03_raw"
}

func (c CompositeRawDeclarer) Declare(string) (string, error) {
	funcName := NameCompositeRawFunc(c.typ)
	sb := &strings.Builder{}
	sb.Grow(256)

	// Doc comment
	sb.WriteString("// ")
	sb.WriteString(funcName)
	sb.WriteString(" returns all composite fields for the Postgres composite\n")
	sb.WriteString("// type '")
	sb.WriteString(c.typ.PgComposite.Name)
	sb.WriteString("' as a slice of interface{} to encode query parameters.\n")

	// Function signature
	sb.WriteString("func (tr *typeResolver) ")
	sb.WriteString(funcName)
	sb.WriteString("(v ")
	sb.WriteString(c.typ.Name)
	sb.WriteString(") []interface{} {\n\t")

	// Function body
	sb.WriteString("return []interface{}{")

	// Field Assigners of the composite type
	for i, fieldType := range c.typ.FieldTypes {
		fieldName := c.typ.FieldNames[i]
		sb.WriteString("\n\t\t")
		switch fieldType := gotype.UnwrapNestedType(fieldType).(type) {
		case *gotype.CompositeType:
			childFuncName := NameCompositeRawFunc(fieldType)
			sb.WriteString("tr.")
			sb.WriteString(childFuncName)
			sb.WriteString("(v.")
			sb.WriteString(fieldName)
			sb.WriteString(")")
		case *gotype.ArrayType:
			sb.WriteString("tr.")
			sb.WriteString(NameArrayRawFunc(fieldType))
			sb.WriteString("(v.")
			sb.WriteString(fieldName)
			sb.WriteString(")")
		case *gotype.VoidType:
			sb.WriteString("nil")
		default:
			sb.WriteString("v.")
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
