package parser

import (
	gotok "go/token"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jschaf/pggen/internal/ast"
)

func ignoreCommentPos() cmp.Option {
	return cmpopts.IgnoreFields(ast.LineComment{}, "Start")
}

func ignoreQueryPos() cmp.Option {
	return cmpopts.IgnoreFields(ast.SourceQuery{}, "Start", "Semi")
}

func TestParseFile_Queries(t *testing.T) {
	tests := []struct {
		src  string
		want ast.Query
	}{
		{
			"-- name: Qux :many\nSELECT 1;",
			&ast.SourceQuery{
				Name:        "Qux",
				Doc:         &ast.CommentGroup{List: []*ast.LineComment{{Text: "-- name: Qux :many"}}},
				SourceSQL:   "SELECT 1;",
				PreparedSQL: "SELECT 1;",
				ResultKind:  ast.ResultKindMany,
			},
		},
		{
			"-- name: Foo :one\nSELECT 1;",
			&ast.SourceQuery{
				Name:        "Foo",
				Doc:         &ast.CommentGroup{List: []*ast.LineComment{{Text: "-- name: Foo :one"}}},
				SourceSQL:   "SELECT 1;",
				PreparedSQL: "SELECT 1;",
				ResultKind:  ast.ResultKindOne,
			},
		},
		{
			"-- name: Qux   :exec\nSELECT pggen.arg('Bar');",
			&ast.SourceQuery{
				Name:        "Qux",
				Doc:         &ast.CommentGroup{List: []*ast.LineComment{{Text: "-- name: Qux   :exec"}}},
				SourceSQL:   "SELECT pggen.arg('Bar');",
				PreparedSQL: "SELECT $1;",
				ParamNames:  []string{"Bar"},
				ResultKind:  ast.ResultKindExec,
			},
		},
		{
			"-- name: Qux   :exec\nSELECT pggen.arg ('Bar');",
			&ast.SourceQuery{
				Name:        "Qux",
				Doc:         &ast.CommentGroup{List: []*ast.LineComment{{Text: "-- name: Qux   :exec"}}},
				SourceSQL:   "SELECT pggen.arg ('Bar');",
				PreparedSQL: "SELECT $1;",
				ParamNames:  []string{"Bar"},
				ResultKind:  ast.ResultKindExec,
			},
		},
		{
			"-- name: Qux :one\nSELECT pggen.arg('A$_$$B123');",
			&ast.SourceQuery{
				Name:        "Qux",
				Doc:         &ast.CommentGroup{List: []*ast.LineComment{{Text: "-- name: Qux :one"}}},
				SourceSQL:   "SELECT pggen.arg('A$_$$B123');",
				PreparedSQL: "SELECT $1;",
				ParamNames:  []string{"A$_$$B123"},
				ResultKind:  ast.ResultKindOne,
			},
		},
		{
			"-- name: Qux :many\nSELECT pggen.arg('Bar'), pggen.arg('Qux'), pggen.arg('Bar');",
			&ast.SourceQuery{
				Name:        "Qux",
				Doc:         &ast.CommentGroup{List: []*ast.LineComment{{Text: "-- name: Qux :many"}}},
				SourceSQL:   "SELECT pggen.arg('Bar'), pggen.arg('Qux'), pggen.arg('Bar');",
				PreparedSQL: "SELECT $1, $2, $1;",
				ParamNames:  []string{"Bar", "Qux"},
				ResultKind:  ast.ResultKindMany,
			},
		},
		{
			"-- name: Qux :many\nSELECT /*pggen.arg('Bar'),*/ pggen.arg('Qux'), pggen.arg('Bar');",
			&ast.SourceQuery{
				Name:        "Qux",
				Doc:         &ast.CommentGroup{List: []*ast.LineComment{{Text: "-- name: Qux :many"}}},
				SourceSQL:   "SELECT /*pggen.arg('Bar'),*/ pggen.arg('Qux'), pggen.arg('Bar');",
				PreparedSQL: "SELECT /*pggen.arg('Bar'),*/ $1, $2;",
				ParamNames:  []string{"Qux", "Bar"},
				ResultKind:  ast.ResultKindMany,
			},
		},
		{
			"-- name: Qux :many proto-type=foo.Bar\nSELECT 1;",
			&ast.SourceQuery{
				Name:        "Qux",
				Doc:         &ast.CommentGroup{List: []*ast.LineComment{{Text: "-- name: Qux :many proto-type=foo.Bar"}}},
				SourceSQL:   "SELECT 1;",
				PreparedSQL: "SELECT 1;",
				ParamNames:  nil,
				ResultKind:  ast.ResultKindMany,
				Pragmas:     ast.Pragmas{ProtobufType: "foo.Bar"},
			},
		},
		{
			"-- name: Qux :many proto-type=Bar\nSELECT 1;",
			&ast.SourceQuery{
				Name:        "Qux",
				Doc:         &ast.CommentGroup{List: []*ast.LineComment{{Text: "-- name: Qux :many proto-type=Bar"}}},
				SourceSQL:   "SELECT 1;",
				PreparedSQL: "SELECT 1;",
				ParamNames:  nil,
				ResultKind:  ast.ResultKindMany,
				Pragmas:     ast.Pragmas{ProtobufType: "Bar"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			f, err := ParseFile(gotok.NewFileSet(), "", tt.src, Trace)
			if err != nil {
				t.Fatal(err)
			}

			got := f.Queries[0].(*ast.SourceQuery)

			if diff := cmp.Diff(tt.want, got, ignoreCommentPos(), ignoreQueryPos()); diff != "" {
				t.Errorf("ParseFile() query mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseFile_Queries_Fuzz(t *testing.T) {
	tests := []struct {
		src string
	}{
		{"-- name: Qux :many\nSELECT '`\\n' as \" joe!@#$%&*()-+=\";"},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			_, err := ParseFile(gotok.NewFileSet(), "", tt.src, Trace)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
