package golang

import (
	"fmt"
	"github.com/jschaf/pggen/internal/casing"
	"github.com/jschaf/pggen/internal/codegen/golang/gotype"
	"github.com/jschaf/pggen/internal/pg"
	"strconv"
)

// CreateCompositeType creates a struct to represent a Postgres composite type.
// The type is rooted under pkgPath.
func CreateCompositeType(pkgPath string, pgt pg.CompositeType, resolver TypeResolver, caser casing.Caser) (gotype.CompositeType, error) {
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
		PkgPath:    pkgPath,
		Pkg:        gotype.ExtractShortPackage([]byte(pkgPath)),
		Name:       name,
		FieldNames: fieldNames,
		FieldTypes: fieldTypes,
	}
	return ct, nil
}
