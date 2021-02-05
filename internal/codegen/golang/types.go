package golang

import (
	"bytes"
	"github.com/jschaf/pggen/internal/casing"
	"github.com/jschaf/pggen/internal/pg"
	"regexp"
	"strconv"
	"strings"
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
	// Last part of the package path, used to qualify type names, like "foo" in
	// "github.com/jschaf/pggen/foo". Empty for builtin types.
	Package() string
	// The name of the type, like the "Foo" in:
	//   type Foo int
	BaseName() string
}

type (
	// EnumType is a string type with constant values that maps to the labels of
	// a Postgres enum.
	EnumType struct {
		PgEnum  pg.EnumType // the original Postgres enum
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

	// Opaque is a type where only the name is known, as with a user-provided
	// custom type.
	OpaqueType struct {
		PkgPath string
		Pkg     string
		Name    string
	}
)

func (e EnumType) QualifyRel(pkgPath string) string { return qualifyRel(e, pkgPath) }
func (e EnumType) Import() string                   { return e.PkgPath }
func (e EnumType) Package() string                  { return e.Pkg }
func (e EnumType) BaseName() string                 { return e.Name }

func (o OpaqueType) QualifyRel(pkgPath string) string { return qualifyRel(o, pkgPath) }
func (o OpaqueType) Import() string                   { return o.PkgPath }
func (o OpaqueType) Package() string                  { return o.Pkg }
func (o OpaqueType) BaseName() string                 { return o.Name }

func qualifyRel(typ Type, otherPkgPath string) string {
	if typ.Import() == otherPkgPath || typ.Import() == "" || typ.Package() == "" {
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

func NewEnumType(pkgPath string, pgEnum pg.EnumType, caser casing.Caser) EnumType {
	name := caser.ToUpperGoIdent(pgEnum.Name)
	if name == "" {
		name = chooseFallbackName(pgEnum.Name, "UnnamedEnum")
	}
	labels := make([]string, len(pgEnum.Labels))
	values := make([]string, len(pgEnum.Labels))
	for i, label := range pgEnum.Labels {
		ident := caser.ToUpperGoIdent(label)
		if ident == "" {
			ident = chooseFallbackName(label, "UnnamedLabel"+strconv.Itoa(i))
		}
		labels[i] = name + ident
		values[i] = pgEnum.Labels[i]
	}
	return EnumType{
		PgEnum:  pgEnum,
		PkgPath: pkgPath,
		Pkg:     extractShortPackage([]byte(pkgPath)),
		Name:    name,
		Labels:  labels,
		Values:  values,
	}
}

// NewOpaqueType creates a Opaque by parsing the fully qualified Go type like:
// "github.com/jschaf/pggen.GenerateOpts", or a builtin type like "string".
func NewOpaqueType(qualType string) OpaqueType {
	if !strings.ContainsRune(qualType, '.') {
		return OpaqueType{Name: qualType} // builtin type like "string"
	}
	bs := []byte(qualType)
	idx := bytes.LastIndexByte(bs, '.')
	name := string(bs[idx+1:])
	pkgPath := bs[:idx]
	shortPkg := extractShortPackage(pkgPath)
	return OpaqueType{
		PkgPath: string(pkgPath),
		Pkg:     shortPkg,
		Name:    name,
	}
}

var majorVersionRegexp = regexp.MustCompile(`^v[0-9]+$`)

// extractShortPackage gets the last part of a package path like "generate" in
// "github.com/jschaf/pggen/generate".
func extractShortPackage(pkgPath []byte) string {
	parts := bytes.Split(pkgPath, []byte{'/'})
	shortPkg := parts[len(parts)-1]
	// Skip major version suffixes got get package name.
	if bytes.HasPrefix(shortPkg, []byte{'v'}) && majorVersionRegexp.Match(shortPkg) {
		shortPkg = parts[len(parts)-2]
	}
	return string(shortPkg)
}
