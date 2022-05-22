package golang

import (
	"fmt"
	"github.com/jschaf/pggen/internal/casing"
	"github.com/jschaf/pggen/internal/codegen/golang/gotype"
	"github.com/jschaf/pggen/internal/pg"
	"strconv"
	"strings"
)

// TypeResolver handles the mapping between Postgres and Go types.
type TypeResolver struct {
	caser     casing.Caser
	overrides map[string]string
}

func NewTypeResolver(c casing.Caser, overrides map[string]string) TypeResolver {
	overs := make(map[string]string, len(overrides))
	for k, v := range overrides {
		for _, alias := range listAliases(k) {
			overs[alias] = v
		}
	}
	return TypeResolver{caser: c, overrides: overs}
}

// Resolve maps a Postgres type to a Go type.
func (tr TypeResolver) Resolve(pgt pg.Type, nullable bool, pkgPath string) (gotype.Type, error) {
	// Custom user override.
	if goType, ok := tr.overrides[pgt.String()]; ok {
		opaque, err := gotype.ParseOpaqueType(goType, pgt)
		if err != nil {
			return nil, fmt.Errorf("resolve custom type: %w", err)
		}
		return opaque, nil
	}

	// Known type.
	var typ gotype.Type
	var isKnownType bool
	if nullable {
		typ, isKnownType = gotype.FindKnownTypeNullable(pgt.OID())
	} else {
		typ, isKnownType = gotype.FindKnownTypeNonNullable(pgt.OID())
	}
	if isKnownType {
		switch typ := typ.(type) {
		case *gotype.ArrayType:
			arrTyp, ok := pgt.(pg.ArrayType)
			if !ok {
				return nil, fmt.Errorf("resolve known type %q does not have pg array type %q", typ, pgt)
			}
			typ.PgArray = arrTyp
			return typ, nil
		case *gotype.CompositeType:
			typ.PgComposite = pgt.(pg.CompositeType)
			return typ, nil
		case *gotype.ImportType:
			ot := typ.Type.(*gotype.OpaqueType)
			ot.PgType = pgt
			return typ, nil
		case *gotype.EnumType:
			typ.PgEnum = pgt.(pg.EnumType)
			return typ, nil
		case *gotype.OpaqueType:
			typ.PgType = pgt
			return typ, nil
		case *gotype.PointerType:
			return typ, nil
		case *gotype.VoidType:
			return &gotype.VoidType{}, nil
		default:
			return nil, fmt.Errorf("resolve unhandled known postgres type %T", typ)
		}
	}

	// New type that pggen will define in generated source code.
	switch pgt := pgt.(type) {
	case pg.ArrayType:
		elemType, err := tr.Resolve(pgt.Elem, nullable, pkgPath)
		if err != nil {
			return nil, fmt.Errorf("resolve array elem type for array type %q: %w", pgt.Name, err)
		}
		return gotype.NewArrayType(pgt, elemType), nil
	case pg.EnumType:
		enum := gotype.NewEnumType(pkgPath, pgt, tr.caser)
		return enum, nil
	case pg.CompositeType:
		comp, err := CreateCompositeType(pkgPath, pgt, tr, tr.caser)
		if err != nil {
			return nil, fmt.Errorf("create composite type: %w", err)
		}
		return comp, nil
	}

	return nil, fmt.Errorf("no go type found for Postgres type %s oid=%d", pgt.String(), pgt.OID())
}

// CreateCompositeType creates a struct to represent a Postgres composite type.
// The type is rooted under pkgPath.
func CreateCompositeType(
	pkgPath string,
	pgt pg.CompositeType,
	resolver TypeResolver,
	caser casing.Caser,
) (gotype.Type, error) {
	name := caser.ToUpperGoIdent(pgt.Name)
	if name == "" {
		name = gotype.ChooseFallbackName(pgt.Name, "UnnamedStruct")
	}
	fieldNames := make([]string, len(pgt.ColumnNames))
	fieldTypes := make([]gotype.Type, len(pgt.ColumnTypes))
	for i, colName := range pgt.ColumnNames {
		ident := caser.ToUpperGoIdent(colName)
		if ident == "" {
			ident = gotype.ChooseFallbackName(colName, "UnnamedField"+strconv.Itoa(i))
		}
		fieldNames[i] = ident
		fieldType, err := resolver.Resolve(pgt.ColumnTypes[i] /*nullable*/, true, pkgPath)
		if err != nil {
			return nil, fmt.Errorf("resolve composite column type %s.%s: %w", pgt.Name, colName, err)
		}
		fieldTypes[i] = fieldType
	}
	ct := &gotype.CompositeType{
		PgComposite: pgt,
		Name:        name,
		FieldNames:  fieldNames,
		FieldTypes:  fieldTypes,
	}
	if pkgPath != "" {
		return &gotype.ImportType{PkgPath: pkgPath, Type: ct}, nil
	}
	return ct, nil
}

func listAliases(name string) []string {
	if strings.HasPrefix(name, "_") {
		aliases := listElemAliases(name[1:])
		for i, alias := range aliases {
			aliases[i] = "_" + alias
		}
		return aliases
	}
	return listElemAliases(name)
}

// listElemAliases lists all known type aliases for a type name. The requested
// type name is included in the list.
// https://www.postgresql.org/docs/13/datatype.html#DATATYPE-TABLE
func listElemAliases(name string) []string {
	switch name {
	case "bigint", "int8":
		return []string{"bigint", "int8"}

	case "bigserial", "serial8":
		return []string{"bigserial", "serial8"}

	case "bool", "boolean":
		return []string{"bool", "boolean"}

	case "float8", "double precision":
		return []string{"float8", "double precision"}

	case "int", "integer", "int4":
		return []string{"int", "integer", "int4"}

	case "real", "float4":
		return []string{"real", "float4"}

	case "smallint", "int2":
		return []string{"smallint", "int2"}

	case "smallserial", "serial2":
		return []string{"smallserial", "serial2"}

	case "serial", "serial4":
		return []string{"serial", "serial4"}

	default:
		// TODO: numeric, multi word aliases
		return []string{name}
	}
}
