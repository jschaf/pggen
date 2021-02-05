package golang

import (
	"github.com/jschaf/pggen/internal/casing"
	"github.com/jschaf/pggen/internal/codegen/golang/gotype"
	"github.com/jschaf/pggen/internal/pg"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestType_QualifyRel(t *testing.T) {
	caser := casing.NewCaser()
	tests := []struct {
		typ          gotype.Type
		otherPkgPath string
		want         string
	}{
		{
			typ: gotype.NewEnumType(
				"example.com/foo",
				pg.EnumType{Name: "device", Labels: []string{"macos"}},
				caser,
			),
			otherPkgPath: "example.com/bar",
			want:         "foo.Device",
		},
		{
			typ: gotype.NewEnumType(
				"example.com/bar",
				pg.EnumType{Name: "device", Labels: []string{"macos"}},
				caser,
			),
			otherPkgPath: "example.com/bar",
			want:         "Device",
		},
		{
			typ:          gotype.NewOpaqueType("example.com/bar.Baz"),
			otherPkgPath: "example.com/bar",
			want:         "Baz",
		},
		{
			typ:          gotype.NewOpaqueType("string"),
			otherPkgPath: "example.com/bar",
			want:         "string",
		},
		{
			typ:          gotype.NewOpaqueType("string"),
			otherPkgPath: "",
			want:         "string",
		},
	}
	for _, tt := range tests {
		t.Run(tt.typ.Import()+"."+tt.typ.BaseName(), func(t *testing.T) {
			got := tt.typ.QualifyRel(tt.otherPkgPath)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCreateCompositeType(t *testing.T) {
	caser := casing.NewCaser()
	resolver := NewTypeResolver(caser, nil)
	tests := []struct {
		pkgPath string
		pgType  pg.CompositeType
		want    gotype.CompositeType
	}{
		{
			pkgPath: "example.com/foo",
			pgType: pg.CompositeType{
				Name:        "qux",
				ColumnNames: []string{"one", "two_a"},
				ColumnTypes: []pg.Type{pg.Text, pg.Int8},
			},
			want: gotype.CompositeType{
				PkgPath:    "example.com/foo",
				Pkg:        "foo",
				Name:       "Qux",
				FieldNames: []string{"One", "TwoA"},
				FieldTypes: []gotype.Type{gotype.PgText, gotype.PgInt8},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.pkgPath+" - "+tt.pgType.Name, func(t *testing.T) {
			got, err := CreateCompositeType(tt.pkgPath, tt.pgType, resolver, caser)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
