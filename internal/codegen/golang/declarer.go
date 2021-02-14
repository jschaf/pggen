package golang

import (
	"github.com/jschaf/pggen/internal/codegen/golang/gotype"
	"strconv"
	"strings"
)

// Declarer is implemented by any value that needs to declare types or data
// before use. For example, Postgres enums map to a Go enum with a type
// declaration and const values. If we use the enum in any Querier function, we
// need to declare the enum.
type Declarer interface {
	// DedupeKey uniquely identifies the declaration so that we only emit
	// declarations once. Should be namespaced like enum::some_enum.
	DedupeKey() string
	// Declare returns the string of the Go declaration.
	Declare(pkgPath string) (string, error)
}

// FindDeclarers finds all necessary Declarers for a type or nil if no
// declarers are needed. Composite type might depends on enums or other
// composite types.
func FindDeclarers(typ gotype.Type) []Declarer {
	return findDeclsHelper(typ, make(map[string]struct{}, 4))
}

func findDeclsHelper(typ gotype.Type, visited map[string]struct{}) []Declarer {
	switch typ := typ.(type) {
	case gotype.EnumType:
		d := NewEnumDeclarer(typ)
		if _, ok := visited[d.DedupeKey()]; ok {
			return nil
		}
		visited[d.DedupeKey()] = struct{}{}
		return []Declarer{d}

	case gotype.CompositeType:
		d := NewCompositeDeclarer(typ)
		if _, ok := visited[d.DedupeKey()]; ok {
			return nil
		}
		visited[d.DedupeKey()] = struct{}{}
		decls := make([]Declarer, 1, 4)
		decls[0] = d
		for _, childType := range typ.FieldTypes {
			childDecls := findDeclsHelper(childType, visited)
			decls = append(decls, childDecls...)
		}
		return decls

	default:
		return nil
	}
}

// CompositeDeclarer declare a new struct to represent a Postgres composite
// type.
type CompositeDeclarer struct {
	comp gotype.CompositeType
}

func NewCompositeDeclarer(comp gotype.CompositeType) CompositeDeclarer {
	return CompositeDeclarer{comp: comp}
}

func (c CompositeDeclarer) DedupeKey() string {
	return "composite::" + c.comp.Name
}

func (c CompositeDeclarer) Declare(pkgPath string) (string, error) {
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

// EnumDeclarer declares a new string type and the const values to map to a
// Postgres enum.
type EnumDeclarer struct {
	enum gotype.EnumType
}

func NewEnumDeclarer(enum gotype.EnumType) EnumDeclarer {
	return EnumDeclarer{enum: enum}
}

func (e EnumDeclarer) DedupeKey() string {
	return "enum::" + e.enum.Name
}

func (e EnumDeclarer) Declare(string) (string, error) {
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
