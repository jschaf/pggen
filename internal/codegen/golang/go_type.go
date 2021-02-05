package golang

import (
	"bytes"
	"fmt"
	"github.com/jschaf/pggen/internal/casing"
	"github.com/jschaf/pggen/internal/gomod"
	"github.com/jschaf/pggen/internal/pg"
	"regexp"
	"strconv"
	"strings"
)

// Type is a Go type.
type Type interface {
	// Stringer should return the name of the type.
	fmt.Stringer
	// Fully qualified package path, like "github.com/jschaf/pggen/foo". Empty
	// for builtin types.
	PackagePath() string
	// Last part of the package path, used to qualify type names, like "foo" in
	// "github.com/jschaf/pggen/foo".
	Package() string
}

type (
	// EnumType is a string type with constant values that maps to the labels of
	// a Postgres enum.
	EnumType struct {
		PgEnum  pg.EnumType // the original Postgres enum
		PkgPath string
		Pkg     string
		Name    string
		// Ordered labels of the Postgres enum formatted as Go identifiers.
		Labels []string
		// The string constant associated with a label.
		Values []string
	}
)

func NewEnumType(pgEnum pg.EnumType, caser casing.Caser) EnumType {
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
		PkgPath: "", // declared in same package for now so ignore
		Pkg:     "",
		Name:    name,
		Labels:  labels,
		Values:  values,
	}
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
