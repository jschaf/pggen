package golang

import (
	"github.com/jschaf/pggen/internal/codegen/golang/gotype"
	"github.com/jschaf/pggen/internal/pg"
	"sort"
	"strconv"
	"strings"
)

// Declarer is implemented by any value that needs to declare types, data, or
// functions before use. For example, Postgres enums map to a Go enum with a
// type declaration and const values. If we use the enum in any Querier
// function, we need to declare the enum.
type Declarer interface {
	// DedupeKey uniquely identifies the declaration so that we only emit
	// declarations once. Should be namespaced like enum::some_enum.
	DedupeKey() string
	// Declare returns the string of the Go code for the declaration.
	Declare(pkgPath string) (string, error)
}

// DeclarerSet is a set of declarers, identified by the dedupe key.
type DeclarerSet map[string]Declarer

func NewDeclarerSet(decls ...Declarer) DeclarerSet {
	d := DeclarerSet(make(map[string]Declarer, len(decls)))
	d.AddAll(decls...)
	return d
}

func (d DeclarerSet) AddAll(decls ...Declarer) {
	for _, decl := range decls {
		d[decl.DedupeKey()] = decl
	}
}

// ListAll gets all declarers in the set in a stable sort order.
func (d DeclarerSet) ListAll() []Declarer {
	decls := make([]Declarer, 0, len(d))
	for _, decl := range d {
		decls = append(decls, decl)
	}
	sort.Slice(decls, func(i, j int) bool { return decls[i].DedupeKey() < decls[j].DedupeKey() })
	return decls
}

// FindDeclarers finds all necessary Declarers for a type or nil if no
// declarers are needed. Composite types might depend on enums or other
// composite types.
func FindDeclarers(typ gotype.Type) DeclarerSet {
	decls := NewDeclarerSet()
	findDeclsHelper(typ, decls, false)
	return decls
}

func findDeclsHelper(typ gotype.Type, decls DeclarerSet, hadCompositeParent bool) {
	switch typ := typ.(type) {
	case gotype.EnumType:
		decls.AddAll(NewEnumTypeDeclarer(typ))
		if hadCompositeParent {
			decls.AddAll(NewEnumPgTypeDeclarer(typ))
		}

	case gotype.CompositeType:
		decls.AddAll(
			NewCompositeTypeDeclarer(typ),
			NewCompositeDecoderDeclarer(typ),
			ignoredOIDDeclarer,
			newCompositeTypeDeclarer,
		)
		for _, childType := range typ.FieldTypes {
			findDeclsHelper(childType, decls, true)
		}

	case gotype.ArrayType:
		decls.AddAll(ignoredOIDDeclarer)
		findDeclsHelper(typ.Elem, decls, hadCompositeParent)

	default:
		return
	}
}

// ConstantDeclarer declares a new string literal.
type ConstantDeclarer struct {
	key string
	str string
}

func NewConstantDeclarer(key, str string) ConstantDeclarer {
	return ConstantDeclarer{key, str}
}

func (c ConstantDeclarer) DedupeKey() string              { return c.key }
func (c ConstantDeclarer) Declare(string) (string, error) { return c.str, nil }

const ignoredOIDDecl = `// ignoredOID means we don't know or care about the OID for a type. This is okay
// because pgx only uses the OID to encode values and lookup a decoder. We only
// use ignoredOID for decoding and we always specify a concrete decoder for scan
// methods.
const ignoredOID = 0`

var ignoredOIDDeclarer = NewConstantDeclarer("const::ignoredOID", ignoredOIDDecl)

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
	funcName := nameCompositeDecoderFunc(c.typ)
	sb := &strings.Builder{}
	sb.Grow(256)

	// Doc comment.
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
			childFuncName := nameCompositeDecoderFunc(fieldType)
			sb.WriteString(childFuncName)
			sb.WriteString("(),")
		case gotype.EnumType:
			sb.WriteString("enumDecoder")
			sb.WriteString(fieldType.Name)
			sb.WriteString(",")
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

func nameCompositeDecoderFunc(typ gotype.CompositeType) string {
	return "new" + typ.Name + "Decoder"
}

// EnumTypeDeclarer declares a new string type and the const values to map to a
// Postgres enum.
type EnumTypeDeclarer struct {
	enum gotype.EnumType
}

func NewEnumTypeDeclarer(enum gotype.EnumType) EnumTypeDeclarer {
	return EnumTypeDeclarer{enum: enum}
}

func (e EnumTypeDeclarer) DedupeKey() string {
	return "enum_type::" + e.enum.Name
}

func (e EnumTypeDeclarer) Declare(string) (string, error) {
	sb := &strings.Builder{}
	// Doc string.
	if e.enum.PgEnum.Name != "" {
		sb.WriteString("// ")
		sb.WriteString(e.enum.Name)
		sb.WriteString(" represents the Postgres enum ")
		sb.WriteString(strconv.Quote(e.enum.PgEnum.Name))
		sb.WriteString(".\n")
	}
	// Type declaration.
	sb.WriteString("type ")
	sb.WriteString(e.enum.Name)
	sb.WriteString(" string\n\n")
	// Const enum values.
	sb.WriteString("const (\n")
	nameLen := 0
	for _, label := range e.enum.Labels {
		if len(label) > nameLen {
			nameLen = len(label)
		}
	}
	for i, label := range e.enum.Labels {
		sb.WriteString("\t")
		sb.WriteString(label)
		sb.WriteString(strings.Repeat(" ", nameLen+1-len(label)))
		sb.WriteString(e.enum.Name)
		sb.WriteString(` = `)
		sb.WriteString(strconv.Quote(e.enum.Values[i]))
		sb.WriteByte('\n')
	}
	sb.WriteString(")\n\n")
	// Stringer
	dispatcher := strings.ToLower(e.enum.Name)[0]
	sb.WriteString("func (")
	sb.WriteByte(dispatcher)
	sb.WriteByte(' ')
	sb.WriteString(e.enum.Name)
	sb.WriteString(") String() string { return string(")
	sb.WriteByte(dispatcher)
	sb.WriteString(") }")
	return sb.String(), nil
}

// EnumPgTypeDeclarer declares a new pgtype.EnumType for use in the generated
// function newCompositeType.
type EnumPgTypeDeclarer struct {
	enum gotype.EnumType
}

func NewEnumPgTypeDeclarer(enum gotype.EnumType) EnumPgTypeDeclarer {
	return EnumPgTypeDeclarer{enum: enum}
}

func (e EnumPgTypeDeclarer) DedupeKey() string {
	return "enum_pgtype::" + e.enum.Name
}

func (e EnumPgTypeDeclarer) Declare(string) (string, error) {
	sb := &strings.Builder{}
	sb.WriteString("var enumDecoder")
	sb.WriteString(e.enum.Name)
	sb.WriteString(` = pgtype.NewEnumType(`)
	sb.WriteString(strconv.Quote(e.enum.PgEnum.Name))
	sb.WriteString(`, []string{`)
	for _, label := range e.enum.Labels {
		sb.WriteString("\n\t")
		sb.WriteString("string(")
		sb.WriteString(label)
		sb.WriteString("),")
	}
	sb.WriteString("\n})")
	return sb.String(), nil
}
