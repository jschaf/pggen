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
			overrides: map[string]string{"custom_type": "example.com/custom.Type"},
			pgType:    pg.BaseType{Name: "custom_type"},
			want:      gotype.NewOpaqueType("example.com/custom.Type"),
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
			overrides: map[string]string{"bigint": "example.com/custom.Type"},
			pgType:    pg.BaseType{Name: "int8", ID: pgtype.Int8OID},
			want:      gotype.NewOpaqueType("example.com/custom.Type"),
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
