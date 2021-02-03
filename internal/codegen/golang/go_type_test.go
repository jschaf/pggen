package golang

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGoType_PackageQualified(t *testing.T) {
	tests := []struct {
		goType GoType
		path   string
		want   string
	}{
		{
			goType: NewGoType("string"),
			path:   "../../Foo.go",
			want:   "string",
		},
		{
			goType: NewGoType("github.com/jschaf/pggen/internal/codegen/golang.CustomType"),
			path:   "Foo.go",
			want:   "CustomType",
		},
		{
			goType: NewGoType("github.com/jschaf/pggen/internal/codegen/golang/nested.CustomType"),
			path:   "Foo.go",
			want:   "nested.CustomType",
		},
		{
			goType: NewGoType("github.com/golang.CustomType"),
			path:   "../../Foo.go",
			want:   "golang.CustomType",
		},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := tt.goType.PackageQualified(tt.path)
			assert.Equal(t, tt.want, got)
		})
	}
}
