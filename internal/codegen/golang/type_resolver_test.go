package golang

import (
	"github.com/jackc/pgtype"
	"github.com/jschaf/pggen/internal/casing"
	"github.com/jschaf/pggen/internal/codegen/golang/gotype"
	"github.com/jschaf/pggen/internal/pg"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTypeResolver_Resolve(t *testing.T) {
	testPkgPath := "github.com/jschaf/pggen/internal/codegen/golang/test_resolve"
	caser := casing.NewCaser()
	caser.AddAcronym("ios", "IOS")
	caser.AddAcronym("id", "ID")
	tests := []struct {
		name      string
		overrides map[string]string
		pgType    pg.Type
		nullable  bool
		want      gotype.Type
	}{
		{
			name:   "enum",
			pgType: pg.EnumType{Name: "device_type", Labels: []string{"macos", "ios", "web"}},
			want: gotype.NewEnumType(
				testPkgPath,
				pg.EnumType{Name: "device_type", Labels: []string{"macos", "ios", "web"}},
				caser,
			),
		},
		{
			name:      "override",
			overrides: map[string]string{"custom_type": "example.com/custom.QualType"},
			pgType:    pg.BaseType{Name: "custom_type"},
			want:      gotype.NewOpaqueType("example.com/custom.QualType"),
		},
		{
			name:     "known nonNullable empty",
			pgType:   pg.BaseType{Name: "text", ID: pgtype.PointOID},
			nullable: false,
			want:     gotype.NewOpaqueType("github.com/jackc/pgtype.Point"),
		},
		{
			name:     "known nullable",
			pgType:   pg.BaseType{Name: "text", ID: pgtype.PointOID},
			nullable: true,
			want:     gotype.NewOpaqueType("github.com/jackc/pgtype.Point"),
		},
		{
			name:      "bigint - int8",
			overrides: map[string]string{"bigint": "example.com/custom.QualType"},
			pgType:    pg.BaseType{Name: "int8", ID: pgtype.Int8OID},
			want:      gotype.NewOpaqueType("example.com/custom.QualType"),
		},
		{
			name: "composite",
			pgType: pg.CompositeType{
				Name:        "qux",
				ColumnNames: []string{"id", "foo"},
				ColumnTypes: []pg.Type{pg.Text, pg.Int8},
			},
			nullable: true,
			want: gotype.CompositeType{
				PgComposite: pg.CompositeType{
					Name:        "qux",
					ColumnNames: []string{"id", "foo"},
					ColumnTypes: []pg.Type{pg.Text, pg.Int8},
				},
				PkgPath:    testPkgPath,
				Pkg:        "test_resolve",
				Name:       "Qux",
				FieldNames: []string{"ID", "Foo"},
				FieldTypes: []gotype.Type{gotype.PgText, gotype.PgInt8},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewTypeResolver(caser, tt.overrides)
			got, err := resolver.Resolve(tt.pgType, tt.nullable, testPkgPath)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

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
				PgComposite: pg.CompositeType{
					Name:        "qux",
					ColumnNames: []string{"one", "two_a"},
					ColumnTypes: []pg.Type{pg.Text, pg.Int8},
				},
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
