package golang

import (
	"github.com/jschaf/pggen/internal/casing"
	"github.com/jschaf/pggen/internal/pg"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_pgToGoType_splitPkg(t *testing.T) {
	tests := []struct {
		goType   knownGoType
		nullable bool
		wantPkg  string
		wantType string
	}{
		{
			goType:   knownGoType{"github.com/jackc/pgtype.Bool", "bool"},
			nullable: true,
			wantPkg:  "github.com/jackc/pgtype",
			wantType: "pgtype.Bool",
		},
		{
			goType:   knownGoType{"github.com/jackc/pgtype.Bool", "bool"},
			nullable: false,
			wantPkg:  "",
			wantType: "bool",
		},
		{
			goType:   knownGoType{"github.com/jackc/pgtype/v4.Bool", "bool"},
			nullable: true,
			wantPkg:  "github.com/jackc/pgtype/v4",
			wantType: "pgtype.Bool",
		},
	}
	for _, tt := range tests {
		t.Run(tt.goType.nullable+" "+tt.goType.nonNullable, func(t *testing.T) {
			gotPkg, gotType := tt.goType.splitPkg(tt.nullable)
			assert.Equal(t, tt.wantPkg, gotPkg, "packages should match")
			assert.Equal(t, tt.wantType, gotType, "types should match")
		})
	}
}

func TestTypeResolver_Resolve(t *testing.T) {
	tests := []struct {
		name   string
		pgType pg.Type
		want   GoType
	}{
		{
			name: "enum",
			pgType: pg.EnumType{
				Name:   "device_type",
				Labels: []string{"macos", "ios", "web"},
			},
			want: GoType{
				Pkg:  "",
				Name: "DeviceType",
				Decl: NewEnumDeclarer("device_type", []string{"macos", "ios", "web"}, casing.NewCaser()),
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewTypeResolver(casing.NewCaser())
			got, err := resolver.Resolve(tt.pgType, false)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
