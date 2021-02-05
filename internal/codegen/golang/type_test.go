package golang

import (
	"github.com/jschaf/pggen/internal/casing"
	"github.com/jschaf/pggen/internal/pg"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestType_QualifyRel(t *testing.T) {
	caser := casing.NewCaser()
	tests := []struct {
		typ          Type
		otherPkgPath string
		want         string
	}{
		{
			typ: NewEnumType(
				"example.com/foo",
				pg.EnumType{Name: "device", Labels: []string{"macos"}},
				caser,
			),
			otherPkgPath: "example.com/bar",
			want:         "foo.Device",
		},
		{
			typ: NewEnumType(
				"example.com/bar",
				pg.EnumType{Name: "device", Labels: []string{"macos"}},
				caser,
			),
			otherPkgPath: "example.com/bar",
			want:         "Device",
		},
	}

	for _, tt := range tests {
		t.Run(tt.typ.Import()+"."+tt.typ.BaseName(), func(t *testing.T) {
			got := tt.typ.QualifyRel(tt.otherPkgPath)
			assert.Equal(t, tt.want, got)
		})
	}
}
