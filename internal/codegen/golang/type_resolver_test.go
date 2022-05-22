package golang

import (
	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgtype"
	"github.com/jschaf/pggen/internal/casing"
	"github.com/jschaf/pggen/internal/codegen/golang/gotype"
	"github.com/jschaf/pggen/internal/difftest"
	"github.com/jschaf/pggen/internal/pg"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTypeResolver_Resolve(t *testing.T) {
	testPkgPath := "github.com/jschaf/pggen/internal/codegen/golang/test_resolve"
	caser := casing.NewCaser()
	caser.AddAcronym("ios", "IOS")
	caser.AddAcronym("macos", "MacOS")
	caser.AddAcronym("id", "ID")
	pgDeviceEnum := pg.EnumType{Name: "device_type", Labels: []string{"macos", "ios", "web"}}
	goDeviceEnum := &gotype.EnumType{
		PgEnum: pgDeviceEnum,
		Name:   "DeviceType",
		Labels: []string{"DeviceTypeMacOS", "DeviceTypeIOS", "DeviceTypeWeb"},
		Values: []string{"macos", "ios", "web"},
	}
	tests := []struct {
		name      string
		overrides map[string]string
		pgType    pg.Type
		nullable  bool
		want      gotype.Type
	}{
		{
			name:   "enum",
			pgType: pgDeviceEnum,
			want:   &gotype.ImportType{PkgPath: testPkgPath, Type: goDeviceEnum},
		},
		{
			name:   "enum array",
			pgType: pg.ArrayType{Name: "_device_type", Elem: pgDeviceEnum},
			want: &gotype.ArrayType{
				PgArray: pg.ArrayType{Name: "_device_type", Elem: pgDeviceEnum},
				Elem:    &gotype.ImportType{PkgPath: testPkgPath, Type: goDeviceEnum},
			},
		},
		{
			name:   "void",
			pgType: pg.VoidType{},
			want:   &gotype.VoidType{},
		},
		{
			name:      "override",
			overrides: map[string]string{"custom_type": "example.com/custom.QualType"},
			pgType:    pg.BaseType{Name: "custom_type"},
			want: &gotype.ImportType{
				PkgPath: "example.com/custom",
				Type:    &gotype.OpaqueType{PgType: pg.BaseType{Name: "custom_type"}, Name: "QualType"},
			},
		},
		{
			name:      "override pointer",
			overrides: map[string]string{"custom_type": "*example.com/custom.QualType"},
			pgType:    pg.BaseType{Name: "custom_type"},
			want: &gotype.PointerType{
				Elem: &gotype.ImportType{
					PkgPath: "example.com/custom",
					Type:    &gotype.OpaqueType{PgType: pg.BaseType{Name: "custom_type"}, Name: "QualType"},
				},
			},
		},
		{
			name:      "override pointer array",
			overrides: map[string]string{"_custom_type": "[]*example.com/custom.QualType"},
			pgType:    pg.ArrayType{Name: "_custom_type", Elem: pg.BaseType{Name: "custom_type"}},
			want: &gotype.ArrayType{
				PgArray: pg.ArrayType{Name: "_custom_type", Elem: pg.BaseType{Name: "custom_type"}},
				Elem: &gotype.PointerType{
					Elem: &gotype.ImportType{
						PkgPath: "example.com/custom",
						Type:    &gotype.OpaqueType{Name: "QualType"},
					},
				},
			},
		},
		{
			name:     "known nonNullable empty",
			pgType:   pg.BaseType{Name: "point", ID: pgtype.PointOID},
			nullable: false,
			want: &gotype.ImportType{
				PkgPath: "github.com/jackc/pgtype",
				Type: &gotype.OpaqueType{
					PgType: pg.BaseType{Name: "point", ID: pgtype.PointOID},
					Name:   "Point",
				},
			},
		},
		{
			name:     "known nullable",
			pgType:   pg.BaseType{Name: "point", ID: pgtype.PointOID},
			nullable: true,
			want: &gotype.ImportType{
				PkgPath: "github.com/jackc/pgtype",
				Type: &gotype.OpaqueType{
					PgType: pg.BaseType{Name: "point", ID: pgtype.PointOID},
					Name:   "Point",
				},
			},
		},
		{
			name:      "bigint - int8",
			overrides: map[string]string{"bigint": "example.com/custom.QualType"},
			pgType:    pg.BaseType{Name: "int8", ID: pgtype.Int8OID},
			want: &gotype.ImportType{
				PkgPath: "example.com/custom",
				Type: &gotype.OpaqueType{
					PgType: pg.BaseType{Name: "int8", ID: pgtype.Int8OID},
					Name:   "QualType",
				},
			},
		},
		{
			name:      "_bigint - _int8",
			overrides: map[string]string{"_bigint": "[]uint16"},
			pgType:    pg.ArrayType{Name: "_int8", Elem: pg.BaseType{Name: "int8", ID: pgtype.Int8OID}},
			want: &gotype.ArrayType{
				PgArray: pg.ArrayType{Name: "_int8", Elem: pg.BaseType{Name: "int8", ID: pgtype.Int8OID}},
				Elem:    &gotype.OpaqueType{Name: "uint16"},
			},
		},
		{
			name:      "_real - _float4 custom type",
			overrides: map[string]string{"_real": "[]example.com/custom.F32"},
			pgType:    pg.ArrayType{ID: pgtype.Float4ArrayOID, Name: "_float4", Elem: pg.BaseType{Name: "_float4", ID: pgtype.Float4OID}},
			want: &gotype.ArrayType{
				PgArray: pg.ArrayType{ID: pgtype.Float4ArrayOID, Name: "_float4", Elem: pg.BaseType{Name: "_float4", ID: pgtype.Float4OID}},
				Elem: &gotype.ImportType{
					PkgPath: "example.com/custom",
					Type:    &gotype.OpaqueType{Name: "F32"},
				},
			},
		},
		{
			name: "composite",
			pgType: pg.CompositeType{
				Name:        "qux",
				ColumnNames: []string{"id", "foo"},
				ColumnTypes: []pg.Type{pg.Text, pg.Int8},
			},
			nullable: true,
			want: &gotype.ImportType{
				PkgPath: testPkgPath,
				Type: &gotype.CompositeType{
					PgComposite: pg.CompositeType{
						Name:        "qux",
						ColumnNames: []string{"id", "foo"},
						ColumnTypes: []pg.Type{pg.Text, pg.Int8},
					},
					Name:       "Qux",
					FieldNames: []string{"ID", "Foo"},
					FieldTypes: []gotype.Type{
						&gotype.PointerType{Elem: &gotype.OpaqueType{Name: "string", PgType: pg.Text}},
						&gotype.PointerType{Elem: &gotype.OpaqueType{Name: "int", PgType: pg.Int8}},
					},
				},
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
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
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
			typ:          gotype.MustParseOpaqueType("example.com/bar.Baz"),
			otherPkgPath: "example.com/bar",
			want:         "Baz",
		},
		{
			typ:          gotype.MustParseKnownType("string", pg.Text),
			otherPkgPath: "example.com/bar",
			want:         "string",
		},
		{
			typ:          gotype.MustParseKnownType("string", pg.Text),
			otherPkgPath: "",
			want:         "string",
		},
	}
	for _, tt := range tests {
		t.Run(tt.typ.Import()+"."+tt.typ.BaseName(), func(t *testing.T) {
			got := gotype.QualifyType(tt.typ, tt.otherPkgPath)
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
		want    gotype.Type
	}{
		{
			pkgPath: "example.com/foo",
			pgType: pg.CompositeType{
				Name:        "qux",
				ColumnNames: []string{"one", "two_a"},
				ColumnTypes: []pg.Type{pg.Text, pg.Int8},
			},
			want: &gotype.ImportType{
				PkgPath: "example.com/foo",
				Type: &gotype.CompositeType{
					PgComposite: pg.CompositeType{
						Name:        "qux",
						ColumnNames: []string{"one", "two_a"},
						ColumnTypes: []pg.Type{pg.Text, pg.Int8},
					},
					Name:       "Qux",
					FieldNames: []string{"One", "TwoA"},
					FieldTypes: []gotype.Type{
						&gotype.PointerType{Elem: &gotype.OpaqueType{PgType: pg.Text, Name: "string"}},
						&gotype.PointerType{Elem: &gotype.OpaqueType{PgType: pg.Int8, Name: "int"}},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.pkgPath+" "+tt.pgType.Name, func(t *testing.T) {
			got, err := CreateCompositeType(tt.pkgPath, tt.pgType, resolver, caser)
			assert.NoError(t, err)
			difftest.AssertSame(t, tt.want, got)
		})
	}
}
