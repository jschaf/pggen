package golang

import (
	"github.com/jschaf/pggen/internal/codegen/golang/gotype"
	"strconv"
	"strings"
)

func NameEnumTranscoderFunc(typ *gotype.EnumType) string {
	return "new" + typ.Name + "Enum"
}

// EnumTypeDeclarer declares a new string type and the const values to map to a
// Postgres enum.
type EnumTypeDeclarer struct {
	enum *gotype.EnumType
}

func NewEnumTypeDeclarer(enum *gotype.EnumType) EnumTypeDeclarer {
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

// EnumTranscoderDeclarer declares a new Go function that creates a pgx decoder
// for the Postgres type represented by the gotype.EnumType.
type EnumTranscoderDeclarer struct {
	typ *gotype.EnumType
}

func NewEnumTranscoderDeclarer(enum *gotype.EnumType) EnumTranscoderDeclarer {
	return EnumTranscoderDeclarer{typ: enum}
}

func (e EnumTranscoderDeclarer) DedupeKey() string {
	return "enum_decoder::" + e.typ.Name
}

func (e EnumTranscoderDeclarer) Declare(string) (string, error) {
	sb := &strings.Builder{}
	funcName := NameEnumTranscoderFunc(e.typ)

	// Doc comment
	sb.WriteString("// ")
	sb.WriteString(funcName)
	sb.WriteString(" creates a new pgtype.ValueTranscoder for the\n")
	sb.WriteString("// Postgres enum type '")
	sb.WriteString(e.typ.PgEnum.Name)
	sb.WriteString("'.\n")

	// Function signature
	sb.WriteString("func ")
	sb.WriteString(funcName)
	sb.WriteString("() pgtype.ValueTranscoder {\n\t")

	// NewEnumType call
	sb.WriteString("return pgtype.NewEnumType(\n\t\t")
	sb.WriteString(strconv.Quote(e.typ.PgEnum.Name))
	sb.WriteString(",\n\t\t")
	sb.WriteString(`[]string{`)
	for _, label := range e.typ.Labels {
		sb.WriteString("\n\t\t\t")
		sb.WriteString("string(")
		sb.WriteString(label)
		sb.WriteString("),")
	}
	sb.WriteString("\n\t\t")
	sb.WriteString("},")
	sb.WriteString("\n\t")
	sb.WriteString(")")
	sb.WriteString("\n")
	sb.WriteString("}")

	return sb.String(), nil
}
