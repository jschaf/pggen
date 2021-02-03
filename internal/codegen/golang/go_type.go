package golang

import (
	"bytes"
	"github.com/jschaf/pggen/internal/gomod"
	"strings"
)

// GoType represents Go type including how to import it and how to reference the
// type.
type GoType struct {
	PkgPath string   // fully qualified package path, like "github.com/jschaf/pggen/foo"
	Pkg     string   // last part of the package path, used to qualify type names if necessary
	Name    string   // name of the type, like Foo in "type Foo int"
	Decl    Declarer // optional Declarer for the type
}

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
