package golang

import (
	"github.com/jschaf/pggen/internal/codegen/golang/gotype"
	"github.com/jschaf/pggen/internal/pg"
	"strconv"
	"strings"
)

func NameCompositeDecoderFunc(typ gotype.CompositeType) string {
	return "new" + typ.Name + "Decoder"
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
