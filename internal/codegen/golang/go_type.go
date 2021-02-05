package golang

import (
	"bytes"
	"github.com/jschaf/pggen/internal/casing"
	"github.com/jschaf/pggen/internal/gomod"
	"github.com/jschaf/pggen/internal/pg"
	"regexp"
	"strconv"
	"strings"
)

// Type is a Go type.
type Type interface {
	// String should return the fully qualified name of the type, like:
	// "github.com/jschaf/pggen.GenerateOptions". Implements fmt.Stringer.
	String() string
	// Fully qualified package path, like "github.com/jschaf/pggen/foo". Empty
	// for builtin types.
	PackagePath() string
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

func (e EnumType) String() string      { return qualifyType(e.PkgPath, e.Name) }
func (e EnumType) PackagePath() string { return e.PkgPath }
func (e EnumType) Package() string     { return e.Pkg }
func (e EnumType) BaseName() string    { return e.Name }

func (o OpaqueType) String() string      { return qualifyType(o.PkgPath, o.Name) }
func (o OpaqueType) PackagePath() string { return o.PkgPath }
func (o OpaqueType) Package() string     { return o.Pkg }
func (o OpaqueType) BaseName() string    { return o.Name }

func qualifyType(pkgPath, baseName string) string {
	sb := strings.Builder{}
	sb.Grow(len(pkgPath) + 1 + len(baseName))
	sb.WriteString(pkgPath)
	if pkgPath != "" {
		sb.WriteRune('.')
	}
	sb.WriteString(baseName)
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

// GoType represents Go type including how to import it and how to reference the
// type.
type GoType struct {
	PkgPath string   // fully qualified package path, like "github.com/jschaf/pggen/foo"
	Pkg     string   // last part of the package path, used to qualify type names if necessary
	Name    string   // name of the type, like Foo in "type Foo int"
	Decl    Declarer // optional Declarer for the type
}

var majorVersionRegexp = regexp.MustCompile(`^v[0-9]+$`)

// NewGoType creates a GoType by parsing the fully qualified Go type like:
// "github.com/jschaf/pggen.GenerateOpts", or a builtin type like "string".
func NewGoType(qualType string) GoType {
	if !strings.ContainsRune(qualType, '.') {
		return GoType{Name: qualType} // builtin type like "string"
	}
	bs := []byte(qualType)
	idx := bytes.LastIndexByte(bs, '.')
	name := string(bs[idx+1:])
	pkgFull := bs[:idx]
	parts := bytes.Split(pkgFull, []byte{'/'})
	shortPkg := parts[len(parts)-1]
	// Skip major version suffixes got get package name.
	if bytes.HasPrefix(shortPkg, []byte{'v'}) && majorVersionRegexp.Match(shortPkg) {
		shortPkg = parts[len(parts)-2]
	}
	return GoType{
		PkgPath: string(pkgFull),
		Pkg:     string(shortPkg),
		Name:    name,
	}
}

// PackageQualified returns the package qualified type, like
// "pggen.GenerateOpts" for the file at path.
func (t GoType) PackageQualified(fileName string) string {
	if t.PkgPath == "" {
		return t.Name // builtin type
	}
	// Try to determine if fileName and the GoType both live in the same package.
	// If they're the same, don't qualify the type. This is imperfect. Go package
	// names don't have to match the dir name. Some dirs might be symlinked.
	// It's good enough as a heuristic.
	qualName := t.Pkg + "." + t.Name
	pkgPath, err := gomod.ResolvePackage(fileName)
	if err != nil {
		return qualName
	}
	if pkgPath == t.PkgPath {
		// Same package, don't qualify with the package name.
		return t.Name
	}
	return qualName
}
