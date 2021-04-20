package golang

import (
	"github.com/jschaf/pggen/internal/codegen/golang/gotype"
	"sort"
)

// Declarer is implemented by any value that needs to declare types, data, or
// functions before use. For example, Postgres enums map to a Go enum with a
// type declaration and const values. If we use the enum in any Querier
// function, we need to declare the enum.
type Declarer interface {
	// DedupeKey uniquely identifies the declaration so that we only emit
	// declarations once. Should be namespaced like enum::some_enum.
	DedupeKey() string
	// Declare returns the string of the Go code for the declaration.
	Declare(pkgPath string) (string, error)
}

// DeclarerSet is a set of declarers, identified by the dedupe key.
type DeclarerSet map[string]Declarer

func NewDeclarerSet(decls ...Declarer) DeclarerSet {
	d := DeclarerSet(make(map[string]Declarer, len(decls)))
	d.AddAll(decls...)
	return d
}

func (d DeclarerSet) AddAll(decls ...Declarer) {
	for _, decl := range decls {
		d[decl.DedupeKey()] = decl
	}
}

// ListAll gets all declarers in the set in a stable sort order.
func (d DeclarerSet) ListAll() []Declarer {
	decls := make([]Declarer, 0, len(d))
	for _, decl := range d {
		decls = append(decls, decl)
	}
	sort.Slice(decls, func(i, j int) bool { return decls[i].DedupeKey() < decls[j].DedupeKey() })
	return decls
}

// FindDeclarers finds all necessary Declarers for a type or nil if no
// declarers are needed. Composite types might depend on enums or other
// composite types.
func FindDeclarers(typ gotype.Type) DeclarerSet {
	decls := NewDeclarerSet()
	findDeclsHelper(typ, decls, false)
	return decls
}

func findDeclsHelper(typ gotype.Type, decls DeclarerSet, hadCompositeParent bool) {
	switch typ := typ.(type) {
	case gotype.EnumType:
		decls.AddAll(
			NewEnumTypeDeclarer(typ),
		)
		if hadCompositeParent {
			// We can use a string as the decoder except if the enum is part of a
			// composite type.
			decls.AddAll(NewEnumDecoderDeclarer(typ))
		}

	case gotype.CompositeType:
		decls.AddAll(
			NewCompositeTypeDeclarer(typ),
			NewCompositeDecoderDeclarer(typ),
			ignoredOIDDeclarer,
			newCompositeTypeDeclarer,
		)
		for _, childType := range typ.FieldTypes {
			findDeclsHelper(childType, decls, true)
		}

	case gotype.ArrayType:
		decls.AddAll(ignoredOIDDeclarer)
		switch typ.Elem.(type) {
		case gotype.CompositeType, gotype.EnumType:
			decls.AddAll(NewArrayDecoderDeclarer(typ))
		}
		findDeclsHelper(typ.Elem, decls, hadCompositeParent)

	default:
		return
	}
}

// ConstantDeclarer declares a new string literal.
type ConstantDeclarer struct {
	key string
	str string
}

func NewConstantDeclarer(key, str string) ConstantDeclarer {
	return ConstantDeclarer{key, str}
}

func (c ConstantDeclarer) DedupeKey() string              { return c.key }
func (c ConstantDeclarer) Declare(string) (string, error) { return c.str, nil }

const ignoredOIDDecl = `// ignoredOID means we don't know or care about the OID for a type. This is okay
// because pgx only uses the OID to encode values and lookup a decoder. We only
// use ignoredOID for decoding and we always specify a concrete decoder for scan
// methods.
const ignoredOID = 0`

var ignoredOIDDeclarer = NewConstantDeclarer("const::ignoredOID", ignoredOIDDecl)
