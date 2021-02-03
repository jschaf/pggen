package golang

import (
	"github.com/jschaf/pggen/internal/casing"
	"strconv"
	"strings"
	"unicode"
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

// EnumDeclarer declares a new string type and the const values to map to a
// Postgres enum.
type EnumDeclarer struct {
	PgName   string   // original Postgres name of the enum
	GoName   string   // name of the enum formatted as a Go identifier
	GoLabels []string // the ordered labels of the enum formatted as Go identifiers
	PgLabels []string // original labels in Postgres
}

func NewEnumDeclarer(pgName string, pgLabels []string, caser casing.Caser) EnumDeclarer {
	goName := caser.ToUpperGoIdent(pgName)
	if goName == "" {
		goName = chooseFallbackName(pgName, "UnnamedEnum")
	}
	goLabels := make([]string, len(pgLabels))
	for i, label := range pgLabels {
		ident := caser.ToUpperGoIdent(label)
		if ident == "" {
			ident = chooseFallbackName(label, "UnnamedLabel"+strconv.Itoa(i))
		}
		goLabels[i] = goName + ident
	}
	return EnumDeclarer{
		PgName:   pgName,
		GoName:   goName,
		GoLabels: goLabels,
		PgLabels: pgLabels,
	}
}

func (e EnumDeclarer) DedupeKey() string {
	return "enum::" + e.PgName
}

func (e EnumDeclarer) Declare() (string, error) {
	sb := &strings.Builder{}
	// Doc string.
	sb.WriteString("// ")
	sb.WriteString(e.GoName)
	sb.WriteString(" represents the Postgres enum ")
	sb.WriteString(strconv.Quote(e.PgName))
	sb.WriteString(".\n")
	// Type declaration.
	sb.WriteString("type ")
	sb.WriteString(e.GoName)
	sb.WriteString(" string\n\n")
	// Const enum values.
	sb.WriteString("const (\n")
	nameLen := 0
	for _, label := range e.GoLabels {
		if len(label) > nameLen {
			nameLen = len(label)
		}
	}
	for i, goLabel := range e.GoLabels {
		sb.WriteString("\t")
		sb.WriteString(goLabel)
		sb.WriteString(strings.Repeat(" ", nameLen+1-len(goLabel)))
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
	sb.WriteString(") }")
	return sb.String(), nil
}

func chooseFallbackName(pgName string, prefix string) string {
	sb := strings.Builder{}
	sb.WriteString(prefix)
	for _, ch := range pgName {
		if unicode.IsLetter(ch) || ch == '_' || unicode.IsDigit(ch) {
			sb.WriteRune(ch)
		}
	}
	return sb.String()
}
