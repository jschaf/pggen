package golang

import (
	"strconv"
	"strings"
)

// Declarer is implemented by any value that needs to declare types or data
// before use. For example, Postgres enums map to a Go enum with a type
// declaration and const values. If we use the enum in any Querier function, we
// need to declare the enum.
type Declarer interface {
	// Declare returns the string of the Go declaration.
	Declare() (string, error)
}

// EnumDeclarer declares a new string type and the const values to map to a
// Postgres enum.
type EnumDeclarer struct {
	PgName   string   // original Postgres name of the enum
	GoName   string   // name of the enum formatted as a Go identifier
	GoLabels []string // the ordered labels of the enum formatted as Go identifiers
	PgLabels []string // original labels in Postgres
}

func (e EnumDeclarer) Declare() (string, error) {
	sb := &strings.Builder{}
	// Doc string.
	sb.WriteString("// ")
	sb.WriteString(e.GoName)
	sb.WriteString(" represents the Postgres enum type ")
	sb.WriteString(e.PgName)
	sb.WriteString(".\n")
	// Type declaration.
	sb.WriteString("type ")
	sb.WriteString(e.GoName)
	sb.WriteString(" string\n\n")
	// Const values.
	sb.WriteString("const (\n")
	for i, goLabel := range e.GoLabels {
		sb.WriteString("\t")
		sb.WriteString(goLabel)
		sb.WriteString(" ")
		sb.WriteString(e.GoName)
		sb.WriteString(` = `)
		sb.WriteString(strconv.Quote(e.PgLabels[i]))
		sb.WriteByte('\n')
	}
	sb.WriteString(")\n\n")
	// Stringer
	dispatcher := strings.ToLower(e.GoName)[0]
	sb.WriteString("func (")
	sb.WriteByte(dispatcher)
	sb.WriteByte(' ')
	sb.WriteString(e.GoName)
	sb.WriteString(") String() string { return string(")
	sb.WriteByte(dispatcher)
	sb.WriteString(") }\n")
	return sb.String(), nil
}
