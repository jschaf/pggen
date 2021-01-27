package golang

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_pgToGoType_splitPkg(t *testing.T) {
	tests := []struct {
		goType   goType
		nullable bool
		wantPkg  string
		wantType string
	}{
		{
			goType{"github.com/jackc/pgtype.Bool", "bool"},
			true,
			"github.com/jackc/pgtype",
			"pgtype.Bool",
		},
		{
			goType{"github.com/jackc/pgtype.Bool", "bool"},
			false,
			"",
			"bool",
		},
		{
			goType{"github.com/jackc/pgtype/v4.Bool", "bool"},
			true,
			"github.com/jackc/pgtype/v4",
			"pgtype.Bool",
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
