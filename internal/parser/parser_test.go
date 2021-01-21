package parser

import (
	"github.com/google/go-cmp/cmp"
	"github.com/jschaf/sqld/internal/ast"
	gotok "go/token"
	"testing"
)

func TestParseFile_Queries(t *testing.T) {
	tests := []struct {
		src  string
		want ast.Query
	}{
		{
			"-- name: Qux\nSELECT 1;",
			&ast.TemplateQuery{
				Name: "Qux",
				Doc: &ast.CommentGroup{List: []*ast.LineComment{
					{Start: 1, Text: "-- name: Qux"},
				}},
				Entry: 14,
				SQL:   "SELECT 1;",
				Semi:  22,
			},
		},
		{
			"-- name: Foo\nSELECT 1;",
			&ast.TemplateQuery{
				Name: "Foo",
				Doc: &ast.CommentGroup{List: []*ast.LineComment{
					{Start: 1, Text: "-- name: Foo"},
				}},
				Entry: 14,
				SQL:   "SELECT 1;",
				Semi:  22,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			f, err := ParseFile(gotok.NewFileSet(), "", tt.src, Trace)
			if err != nil {
				t.Fatal(err)
			}

			got := f.Queries[0].(*ast.TemplateQuery)

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ParseFile() query mismatch (-want +got):\n%s", diff)
			}
		})
	}

}
