package golang

import (
	"fmt"
	"github.com/jschaf/pggen/internal/casing"
	"github.com/jschaf/pggen/internal/codegen/golang/gotype"
	"github.com/jschaf/pggen/internal/pg"
	"strconv"
)

// TypeResolver handles the mapping between Postgres and Go types.
type TypeResolver struct {
	caser     casing.Caser
	overrides map[string]string
}

func NewTypeResolver(c casing.Caser, overrides map[string]string) TypeResolver {
	overs := make(map[string]string, len(overrides))
	for k, v := range overrides {
		overs[k] = v
		// Type aliases.
		// https://www.postgresql.org/docs/13/datatype.html#DATATYPE-TABLE
		switch k {
		case "bigint":
			overs["int8"] = v
		case "int8":
			overs["bigint"] = v

		case "bigserial":
			overs["serial8"] = v
		case "serial8":
			overs["bigserial"] = v

		case "boolean":
			overs["bool"] = v
		case "bool":
			overs["boolean"] = v

		case "double precision":
			overs["float8"] = v
		case "float8":
			overs["double precision"] = v

		case "int":
			overs["integer"] = v
			overs["int4"] = v
		case "integer":
			overs["int"] = v
			overs["int4"] = v
		case "int4":
			overs["integer"] = v
			overs["int"] = v

			// TODO: numeric, multi word aliases

		case "real":
			overs["float4"] = v
		case "float4":
			overs["real"] = v

		case "smallint":
			overs["int2"] = v
		case "int2":
			overs["smallint"] = v

		case "smallserial":
			overs["serial2"] = v
		case "serial2":
			overs["smallserial"] = v

		case "serial":
			overs["serial4"] = v
		case "serial4":
			overs["serial"] = v
		}
	}
	return TypeResolver{caser: c, overrides: overs}
}

// Resolve maps a Postgres type to a Go type.
func (tr TypeResolver) Resolve(pgt pg.Type, nullable bool, pkgPath string) (gotype.Type, error) {
	// Custom user override.
	if goType, ok := tr.overrides[pgt.String()]; ok {
		return gotype.NewOpaqueType(goType), nil
	}

	// Known type.
	if typ, ok := gotype.FindKnownTypeByOID(pgt.OID(), nullable); ok {
		return typ, nil
	}

	// New type that pggen will define in generated source code.
	switch pgt := pgt.(type) {
	case pg.ArrayType:
		elemType, err := tr.Resolve(pgt.ElemType, nullable, pkgPath)
		if err != nil {
			return nil, fmt.Errorf("resolve array elem type for array type %q: %w", pgt.Name, err)
		}
		return gotype.NewArrayType(pkgPath, pgt, tr.caser, elemType), nil
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
) (gotype.CompositeType, error) {
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
			return gotype.CompositeType{}, fmt.Errorf("resolve composite column type %s.%s: %w", pgt.Name, colName, err)
		}
		fieldTypes[i] = fieldType
	}
	ct := gotype.CompositeType{
		PgComposite: pgt,
		PkgPath:     pkgPath,
		Pkg:         gotype.ExtractShortPackage([]byte(pkgPath)),
		Name:        name,
		FieldNames:  fieldNames,
		FieldTypes:  fieldTypes,
	}
	return ct, nil
}
