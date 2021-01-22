package pginfer

import (
	"github.com/jschaf/sqld/internal/ast"
	"github.com/jschaf/sqld/internal/pg"
	"github.com/jschaf/sqld/internal/pgtest"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInferrer_InferTypes(t *testing.T) {
	conn, cleanupFunc := pgtest.NewPostgresSchema(t, []string{
		"../../example/author/schema.sql",
	})
	defer cleanupFunc()

	tests := []struct {
		query *ast.TemplateQuery
		want  TypedQuery
	}{
		{
			&ast.TemplateQuery{
				Name:        "FindByFirstName",
				PreparedSQL: "SELECT first_name FROM author WHERE first_name = $1;",
				ParamNames:  []string{"FirstName"},
			},
			TypedQuery{
				Name:        "FindByFirstName",
				PreparedSQL: "SELECT first_name FROM author WHERE first_name = $1;",
				Inputs: []InputParam{
					{Name: "FirstName", PgType: pg.Text, GoType: "string"},
				},
				Outputs: []OutputColumn{
					{PgName: "first_name", GoName: "first_name", PgType: pg.Text, GoType: "string"},
				},
			},
		},
		{
			&ast.TemplateQuery{
				Name:        "DeleteAuthorByID",
				PreparedSQL: "DELETE FROM author WHERE author_id = $1;",
				ParamNames:  []string{"AuthorID"},
			},
			TypedQuery{
				Name:        "DeleteAuthorByID",
				PreparedSQL: "DELETE FROM author WHERE author_id = $1;",
				Inputs: []InputParam{
					{Name: "AuthorID", PgType: pg.Int4, GoType: "int32"},
				},
				Outputs: nil,
			},
		},
		{
			&ast.TemplateQuery{
				Name:        "DeleteAuthorByIDReturning",
				PreparedSQL: "DELETE FROM author WHERE author_id = $1 RETURNING author_id, first_name;",
				ParamNames:  []string{"AuthorID"},
			},
			TypedQuery{
				Name:        "DeleteAuthorByIDReturning",
				PreparedSQL: "DELETE FROM author WHERE author_id = $1 RETURNING author_id, first_name;",
				Inputs: []InputParam{
					{Name: "AuthorID", PgType: pg.Int4, GoType: "int32"},
				},
				Outputs: []OutputColumn{
					{PgName: "author_id", GoName: "author_id", PgType: pg.Int4, GoType: "int32"},
					{PgName: "first_name", GoName: "first_name", PgType: pg.Text, GoType: "string"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.query.Name, func(t *testing.T) {
			inferrer := NewInferrer(conn)
			got, err := inferrer.InferTypes(tt.query)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tt.want, got, "typed query should match")
		})
	}
}
