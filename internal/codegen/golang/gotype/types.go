package gotype

import (
	"bytes"
	"fmt"
	"github.com/jschaf/pggen/internal/casing"
	"github.com/jschaf/pggen/internal/pg"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// Type is a Go type.
type Type interface {
	// Import returns the full package path, like "github.com/jschaf/pggen/foo".
	// Empty for builtin types.
	Import() string
	// BaseName returns the unqualified, base name of the type, like "Foo" in:
	//   type Foo int, or "[]*Foo".
	BaseName() string
}

type (
	// ArrayType is a Go slice type.
	ArrayType struct {
		PgArray pg.ArrayType // original Postgres array type
		Elem    Type         // element type of the slice, like int for []int
	}

	// CompositeType is a struct type that represents a Postgres composite type.
	CompositeType struct {
		PgComposite pg.CompositeType // original Postgres composite type
		Name        string           // Go-style type name in UpperCamelCase
		FieldNames  []string         // Go-style child names in UpperCamelCase
		FieldTypes  []Type
	}

	// EnumType is a string type with constant values that maps to the labels of
	// a Postgres enum.
	EnumType struct {
		PgEnum pg.EnumType // the original Postgres enum type
		Name   string      // name of the unqualified Go type
		// Labels of the Postgres enum formatted as Go identifiers ordered in the
		// same order as in Postgres.
		Labels []string
		// The string constant associated with a label. Labels[i] represents
		// Values[i].
		Values []string
	}

	// ImportType is an imported type.
	ImportType struct {
		PkgPath string // fully qualified package path, like "github.com/jschaf/pggen"
		Type    Type   // type to import
	}

	// OpaqueType is a type where only the name is known, as with a user-provided
	// custom type.
	OpaqueType struct {
		PgType pg.Type // original Postgres type
		Name   string  // name of the unqualified Go type
	}

	// PointerType is a pointer to another Go type.
	PointerType struct {
		Elem Type // the pointed-to type
	}

	// VoidType is a placeholder type that should never appear in output. We need
	// a placeholder to scan pgx rows, but we ultimately ignore the results in the
	// return values.
	VoidType struct{}
)

func (a *ArrayType) Import() string   { return a.Elem.Import() }
func (a *ArrayType) BaseName() string { return "[]" + a.Elem.BaseName() }

func (c *CompositeType) Import() string   { return "" }
func (c *CompositeType) BaseName() string { return c.Name }

func (e *EnumType) Import() string   { return "" }
func (e *EnumType) BaseName() string { return e.Name }

func (e *ImportType) Import() string   { return e.PkgPath }
func (e *ImportType) BaseName() string { return e.Type.BaseName() }

func (o *OpaqueType) Import() string   { return "" }
func (o *OpaqueType) BaseName() string { return o.Name }

func (o *PointerType) Import() string   { return "" }
func (o *PointerType) BaseName() string { return "*" + o.Elem.BaseName() }

func (e *VoidType) Import() string   { return "" }
func (e *VoidType) BaseName() string { return "" }

func getTypePackage(typ Type) string {
	switch typ := typ.(type) {
	case *ArrayType:
		return getTypePackage(typ.Elem)
	case *CompositeType:
		return ""
	case *EnumType:
		return ""
	case *ImportType:
		return typ.PkgPath
	case *OpaqueType:
		return ""
	case *PointerType:
		return getTypePackage(typ.Elem)
	case *VoidType:
		return ""
	default:
		panic(fmt.Sprintf("unhandled getTypePackage type %T", typ))
	}
}

func QualifyType(typ Type, otherPkgPath string) string {
	sb := &strings.Builder{}
	arrType, isArr := typ.(*ArrayType)
	if isArr {
		sb.WriteString("[]")
		typ = arrType.Elem
	}
	ptrType, isPtr := typ.(*PointerType)
	if isPtr {
		sb.WriteString("*")
		typ = ptrType.Elem
	}

	pkg := getTypePackage(typ)
	if typ.Import() == otherPkgPath || typ.Import() == "" || pkg == "" {
		sb.WriteString(typ.BaseName())
		return sb.String()
	}
	if !strings.ContainsRune(otherPkgPath, '.') && pkg == otherPkgPath {
		// If the otherPkgPath is unqualified and matches the package path, assume
		// the same package.
		return typ.BaseName()
	}
	sb.Grow(len(typ.BaseName()))
	if typ.Import() != "" {
		shortPkg := ExtractShortPackage([]byte(pkg))
		sb.WriteString(shortPkg)
		sb.WriteRune('.')
	}
	sb.WriteString(typ.BaseName())
	return sb.String()
}

func NewArrayType(pgArray pg.ArrayType, elemType Type) Type {
	return &ArrayType{
		PgArray: pgArray,
		Elem:    elemType,
	}
}

func NewEnumType(pkgPath string, pgEnum pg.EnumType, caser casing.Caser) Type {
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
	typ := &EnumType{
		PgEnum: pgEnum,
		Name:   name,
		Labels: labels,
		Values: values,
	}
	if pkgPath != "" {
		return &ImportType{
			PkgPath: pkgPath,
			Type:    typ,
		}
	}
	return typ
}

// ParseOpaqueType creates a Type by parsing a fully qualified Go type like
// "github.com/jschaf/custom.Int4" with the backing pg.Type.
//
//   - []int
//   - []*int
//   - *example.com/foo.Qux
//   - []*example.com/foo.Qux
func ParseOpaqueType(qualType string, pgType pg.Type) (Type, error) {
	bs := []byte(qualType)
	isArr := bs[0] == '['
	if isArr {
		if bs[1] != ']' {
			return nil, fmt.Errorf("malformed custom type %q; must have closing bracket", qualType)
		}
		bs = bs[2:]
	}
	isPtr := bs[0] == '*'
	if isPtr {
		bs = bs[1:]
	}
	idx := bytes.LastIndexByte(bs, '.')
	name := string(bs[idx+1:])
	var typ Type = &OpaqueType{Name: name}
	// On array types, the PgType goes on the Array. In all other cases, it
	// goes on the OpaqueType.
	if t, ok := typ.(*OpaqueType); ok && !isArr {
		t.PgType = pgType
	}

	if isQualifiedType := idx != -1; isQualifiedType {
		pkgPath := bs[:idx]
		typ = &ImportType{
			PkgPath: string(pkgPath),
			Type:    typ,
		}
	}

	if isPtr {
		typ = &PointerType{Elem: typ}
	}

	if isArr {
		pgArr, ok := pgType.(pg.ArrayType)
		// Ensure that if we have a Go slice type that the Postgres type is also
		// an array. []byte is special since it maps to the Postgres bytea type.
		if !ok && pgType != nil && qualType != "[]byte" {
			return nil, fmt.Errorf("opaque pg type %T{%+v} for go type %q is not a pg.ArrayType", pgType, pgType, qualType)
		}
		typ = &ArrayType{PgArray: pgArr, Elem: typ}
	}

	return typ, nil
}

// MustParseKnownType creates a gotype.Type by parsing a fully qualified Go type
// that pgx supports natively like "github.com/jackc/pgtype.Int4Array", or most
// builtin types like "string" and []*int16.
func MustParseKnownType(qualType string, pgType pg.Type) Type {
	typ, err := ParseOpaqueType(qualType, pgType)
	if err != nil {
		panic(err.Error())
	}
	return typ
}

// MustParseOpaqueType creates a gotype.Type by parsing a fully qualified Go
// type unsupported by pgx supports natively like "github.com/example/Foo"
func MustParseOpaqueType(qualType string) Type {
	typ, err := ParseOpaqueType(qualType, nil)
	if err != nil {
		panic(err.Error())
	}
	return typ
}

var majorVersionRegexp = regexp.MustCompile(`^v[0-9]+$`)

// ExtractShortPackage gets the last part of a package path like "generate" in
// "github.com/jschaf/pggen/generate".
func ExtractShortPackage(pkgPath []byte) string {
	parts := bytes.Split(pkgPath, []byte{'/'})
	shortPkg := parts[len(parts)-1]
	// Skip major version suffixes to get package name.
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

// UnwrapNestedType returns the first type under gotype.ImportType or
// gotype.PointerType.
func UnwrapNestedType(typ Type) Type {
	switch typ := typ.(type) {
	case *ImportType:
		return UnwrapNestedType(typ.Type)
	case *PointerType:
		return UnwrapNestedType(typ.Elem)
	default:
		return typ
	}
}

// IsPgxSupportedArray returns true if pgx can handle the translation from the
// Go array type into the Postgres type.
func IsPgxSupportedArray(typ *ArrayType) bool {
	elem := typ.Elem
	if ptr, ok := elem.(*PointerType); ok {
		elem = ptr.Elem
	}

	var pkgPath string
	if imp, ok := elem.(*ImportType); ok {
		pkgPath = imp.PkgPath
		elem = imp.Type
	}

	base, ok := elem.(*OpaqueType)
	if !ok {
		return false
	}

	name := base.Name
	if pkgPath != "" {
		name = pkgPath + "." + name
	}

	switch name {
	case "string", "byte", "rune",
		"int", "int16", "int32", "int64",
		"uint", "uint16", "uint32", "uint64",
		"float32", "float64",
		"time.Time":
		return true
	default:
		return false
	}
}
