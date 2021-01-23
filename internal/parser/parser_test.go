package parser

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jschaf/sqld/internal/ast"
	gotok "go/token"
	"testing"
)

func ignoreCommentPos() cmp.Option {
	return cmpopts.IgnoreFields(ast.LineComment{}, "Start")
}

func ignoreQueryPos() cmp.Option {
	return cmpopts.IgnoreFields(ast.TemplateQuery{}, "Start", "Semi")
}

func TestParseFile_Queries(t *testing.T) {
	tests := []struct {
		src  string
		want ast.Query
	}{
		{
			"-- name: Qux :many\nSELECT 1;",
			&ast.TemplateQuery{
				Name:        "Qux",
				Doc:         &ast.CommentGroup{List: []*ast.LineComment{{Text: "-- name: Qux :many"}}},
				TemplateSQL: "SELECT 1;",
				PreparedSQL: "SELECT 1;",
				ResultKind:  ast.ResultKindMany,
			},
		},
		{
			"-- name: Foo :one\nSELECT 1;",
			&ast.TemplateQuery{
				Name:        "Foo",
				Doc:         &ast.CommentGroup{List: []*ast.LineComment{{Text: "-- name: Foo :one"}}},
				TemplateSQL: "SELECT 1;",
				PreparedSQL: "SELECT 1;",
				ResultKind:  ast.ResultKindOne,
			},
		},
		{
			"-- name: Qux   :exec\nSELECT pggen.arg('Bar');",
			&ast.TemplateQuery{
				Name:        "Qux",
				Doc:         &ast.CommentGroup{List: []*ast.LineComment{{Text: "-- name: Qux   :exec"}}},
				TemplateSQL: "SELECT pggen.arg('Bar');",
				PreparedSQL: "SELECT $1;",
				ParamNames:  []string{"Bar"},
				ResultKind:  ast.ResultKindExec,
			},
		},
		{
			"-- name: Qux :one\nSELECT pggen.arg('A$_$$B123');",
			&ast.TemplateQuery{
				Name:        "Qux",
				Doc:         &ast.CommentGroup{List: []*ast.LineComment{{Text: "-- name: Qux :one"}}},
				TemplateSQL: "SELECT pggen.arg('A$_$$B123');",
				PreparedSQL: "SELECT $1;",
				ParamNames:  []string{"A$_$$B123"},
				ResultKind:  ast.ResultKindOne,
			},
		},
		{
			"-- name: Qux :many\nSELECT pggen.arg('Bar'), pggen.arg('Qux'), pggen.arg('Bar');",
			&ast.TemplateQuery{
				Name:        "Qux",
				Doc:         &ast.CommentGroup{List: []*ast.LineComment{{Text: "-- name: Qux :many"}}},
				TemplateSQL: "SELECT pggen.arg('Bar'), pggen.arg('Qux'), pggen.arg('Bar');",
				PreparedSQL: "SELECT $1, $2, $1;",
				ParamNames:  []string{"Bar", "Qux"},
				ResultKind:  ast.ResultKindMany,
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

			if diff := cmp.Diff(tt.want, got, ignoreCommentPos(), ignoreQueryPos()); diff != "" {
				t.Errorf("ParseFile() query mismatch (-want +got):\n%s", diff)
			}
		})
	}

}
