package gomod

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolvePackage(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{
			path: "Foo.go",
			want: "github.com/jschaf/pggen/internal/gomod",
		},
		{
			path: "../Foo.go",
			want: "github.com/jschaf/pggen/internal",
		},
		{
			path: "./Foo.go",
			want: "github.com/jschaf/pggen/internal/gomod",
		},
		{
			path: "blah/qux/Foo.go",
			want: "github.com/jschaf/pggen/internal/gomod/blah/qux",
		},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got, err := GuessPackage(tt.path)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
