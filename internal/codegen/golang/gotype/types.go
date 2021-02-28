package gotype

import (
	"bytes"
	"github.com/jschaf/pggen/internal/casing"
	"github.com/jschaf/pggen/internal/pg"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// Type is a Go type.
type Type interface {
	// QualifyRel qualifies the type relative to another pkgPath. If this type's
	// package path is the same, return the BaseName. Otherwise, qualify the type
	// with Package.
	QualifyRel(pkgPath string) string
	// Import returns the full package path, like "github.com/jschaf/pggen/foo".
	// Empty for builtin types.
	Import() string
	// Last part of the package path like "qux" for the package
	// "github.com/jschaf/pggen/qux". Empty for builtin types.
	Package() string
	// The base name of the type, like the "Foo" in:
	//   type Foo int
	BaseName() string
}

type (
	// VoidType is a placeholder type that should never appear in output. We need
	// a placeholder to scan pgx rows but we ultimately ignore the results in the
	// return values.
	VoidType struct{}

	// ArrayType is a Go slice type.
	ArrayType struct {
		PgArray pg.ArrayType // original Postgres array type
		PkgPath string       // fully qualified package path, like "github.com/jschaf/pggen"
		Pkg     string       // last part of the package path like "pggen" or empty for builtin types
		Name    string       // name of Go slice type in UpperCamelCase with leading brackets, like "[]Foo"
		Elem    Type         // base type of the slice, like int for []int
	}

	// EnumType is a string type with constant values that maps to the labels of
	// a Postgres enum.
	EnumType struct {
		PgEnum  pg.EnumType // the original Postgres enum type
		PkgPath string
		Pkg     string
		Name    string
		// Labels of the Postgres enum formatted as Go identifiers ordered in the
		// same order as in Postgres.
		Labels []string
		// The string constant associated with a label. Labels[i] represents
		// Values[i].
		Values []string
	}

	// OpaqueType is a type where only the name is known, as with a user-provided
	// custom type.
	OpaqueType struct {
		PkgPath string
		Pkg     string
		Name    string
	}

	// CompositeType is a struct type that represents a Postgres composite type,
	// typically from a table.
	CompositeType struct {
		PgComposite pg.CompositeType // original Postgres composite type
		PkgPath     string
		Pkg         string
		Name        string   // Go-style type name in UpperCamelCase
		FieldNames  []string // Go-style child names in UpperCamelCase
		FieldTypes  []Type
	}
)

func (e VoidType) QualifyRel(pkgPath string) string { return qualifyRel(e, pkgPath) }
func (e VoidType) Import() string                   { return "" }
func (e VoidType) Package() string                  { return "" }
func (e VoidType) BaseName() string                 { return "" }

func (a ArrayType) QualifyRel(pkgPath string) string { return qualifyRel(a, pkgPath) }
func (a ArrayType) Import() string                   { return a.PkgPath }
func (a ArrayType) Package() string                  { return a.Pkg }
func (a ArrayType) BaseName() string                 { return a.Name }

func (e EnumType) QualifyRel(pkgPath string) string { return qualifyRel(e, pkgPath) }
func (e EnumType) Import() string                   { return e.PkgPath }
func (e EnumType) Package() string                  { return e.Pkg }
func (e EnumType) BaseName() string                 { return e.Name }

func (o OpaqueType) QualifyRel(pkgPath string) string { return qualifyRel(o, pkgPath) }
func (o OpaqueType) Import() string                   { return o.PkgPath }
func (o OpaqueType) Package() string                  { return o.Pkg }
func (o OpaqueType) BaseName() string                 { return o.Name }

func (c CompositeType) QualifyRel(pkgPath string) string { return qualifyRel(c, pkgPath) }
func (c CompositeType) Import() string                   { return c.PkgPath }
func (c CompositeType) Package() string                  { return c.Pkg }
func (c CompositeType) BaseName() string                 { return c.Name }

func qualifyRel(typ Type, otherPkgPath string) string {
	if typ.Import() == otherPkgPath || typ.Import() == "" || typ.Package() == "" {
		return typ.BaseName()
	}
	if !strings.ContainsRune(otherPkgPath, '.') && typ.Package() == otherPkgPath {
		// If the otherPkgPath is unqualified and matches the package path, assume
		// the same package.
		return typ.BaseName()
	}
	sb := strings.Builder{}
	sb.Grow(len(typ.BaseName()))
	if typ.Import() != "" {
		shortPkg := typ.Package()
		sb.Grow(len(shortPkg) + 1)
		sb.WriteString(shortPkg)
		sb.WriteRune('.')
	}
	sb.WriteString(typ.BaseName())
	return sb.String()
}

func NewArrayType(pkgPath string, pgArray pg.ArrayType, caser casing.Caser, elemType Type) ArrayType {
	name := caser.ToUpperGoIdent(pgArray.Name)
	if name == "" {
		name = ChooseFallbackName(pgArray.Name, "UnnamedArray")
	}
	return ArrayType{
		PgArray: pgArray,
		PkgPath: pkgPath,
		Pkg:     ExtractShortPackage([]byte(pkgPath)),
		Name:    "[]" + name,
		Elem:    elemType,
	}
}

func NewEnumType(pkgPath string, pgEnum pg.EnumType, caser casing.Caser) EnumType {
	name := caser.ToUpperGoIdent(pgEnum.Name)
	if name == "" {
		name = ChooseFallbackName(pgEnum.Name, "UnnamedEnum")
	}
	labels := make([]string, len(pgEnum.Labels))
	values := make([]string, len(pgEnum.Labels))
	for i, label := range pgEnum.Labels {
		ident := caser.ToUpperGoIdent(label)
		if ident == "" {
			ident = ChooseFallbackName(label, "UnnamedLabel"+strconv.Itoa(i))
		}
		labels[i] = name + ident
		values[i] = pgEnum.Labels[i]
	}
	return EnumType{
		PgEnum:  pgEnum,
		PkgPath: pkgPath,
		Pkg:     ExtractShortPackage([]byte(pkgPath)),
		Name:    name,
		Labels:  labels,
		Values:  values,
	}
}

// NewOpaqueType creates a OpaqueType by parsing the fully qualified Go type
// like "github.com/jschaf/pggen.GenerateOpts", or a builtin type like "string".
func NewOpaqueType(qualType string) OpaqueType {
	if !strings.ContainsRune(qualType, '.') {
		return OpaqueType{Name: qualType} // builtin type like "string"
	}
	isArr := qualType[:2] == "[]"
	if isArr {
		qualType = qualType[2:]
	}
	bs := []byte(qualType)
	idx := bytes.LastIndexByte(bs, '.')
	name := string(bs[idx+1:])
	if isArr {
		name = "[]" + name
	}
	pkgPath := bs[:idx]
	shortPkg := ExtractShortPackage(pkgPath)
	return OpaqueType{
		PkgPath: string(pkgPath),
		Pkg:     shortPkg,
		Name:    name,
	}
}

var majorVersionRegexp = regexp.MustCompile(`^v[0-9]+$`)

// ExtractShortPackage gets the last part of a package path like "generate" in
// "github.com/jschaf/pggen/generate".
func ExtractShortPackage(pkgPath []byte) string {
	parts := bytes.Split(pkgPath, []byte{'/'})
	shortPkg := parts[len(parts)-1]
	// Skip major version suffixes got get package name.
	if bytes.HasPrefix(shortPkg, []byte{'v'}) && majorVersionRegexp.Match(shortPkg) {
		shortPkg = parts[len(parts)-2]
	}
	return string(shortPkg)
}

func ChooseFallbackName(pgName string, prefix string) string {
	sb := strings.Builder{}
	sb.WriteString(prefix)
	for _, ch := range pgName {
		if unicode.IsLetter(ch) || ch == '_' || unicode.IsDigit(ch) {
			sb.WriteRune(ch)
		}
	}
	return sb.String()
}
