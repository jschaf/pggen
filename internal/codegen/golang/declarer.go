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
	Declare() (string, error)
}

// FindDeclarer finds the appropriate Declarer for a type or nil if no declare
// is needed.
func FindDeclarer(typ gotype.Type) Declarer {
	switch typ := typ.(type) {
	case gotype.EnumType:
		return NewEnumDeclarer(typ)
	default:
		return nil
	}
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

func (e EnumDeclarer) Declare() (string, error) {
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
