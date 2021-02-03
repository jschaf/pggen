package golang

import (
	"github.com/jackc/pgtype"
	"github.com/jschaf/pggen/internal/casing"
	"github.com/jschaf/pggen/internal/pg"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTypeResolver_Resolve(t *testing.T) {
	tests := []struct {
		name      string
		overrides map[string]string
		pgType    pg.Type
		nullable  bool
		want      GoType
	}{
		{
			name: "enum",
			pgType: pg.EnumType{
				Name:   "device_type",
				Labels: []string{"macos", "ios", "web"},
			},
			want: GoType{
				PkgPath: "github.com/jschaf/pggen/internal/codegen",
				Pkg:     "codegen",
				Name:    "DeviceType",
				Decl:    NewEnumDeclarer("device_type", []string{"macos", "ios", "web"}, casing.NewCaser()),
			},
		},
		{
			name:      "override",
			overrides: map[string]string{"custom_type": "example.com/custom.Type"},
			pgType:    pg.BaseType{Name: "custom_type"},
			want: GoType{
				PkgPath: "example.com/custom",
				Pkg:     "custom",
				Name:    "Type",
			},
		},
		{
			name:     "known nonNullable empty",
			pgType:   pg.BaseType{Name: "text", ID: pgtype.PointOID},
			nullable: false,
			want: GoType{
				PkgPath: "github.com/jackc/pgtype",
				Pkg:     "pgtype",
				Name:    "Point",
			},
		},
		{
			name:     "known nullable",
			pgType:   pg.BaseType{Name: "text", ID: pgtype.PointOID},
			nullable: true,
			want: GoType{
				PkgPath: "github.com/jackc/pgtype",
				Pkg:     "pgtype",
				Name:    "Point",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewTypeResolver(casing.NewCaser(), tt.overrides)
			got, err := resolver.Resolve(tt.pgType, tt.nullable, "")
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
