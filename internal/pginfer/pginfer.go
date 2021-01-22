package pginfer

import (
	"github.com/jackc/pgx/v4"
	"github.com/jschaf/sqld/internal/ast"
)

// CmdTag is the command tag reported by Postgres when running the TemplateQuery.
// See "command tag" in https://www.postgresql.org/docs/current/protocol-message-formats.html
type CmdTag string

const (
	TagSelect CmdTag = "select"
	TagInsert CmdTag = "insert"
	TagUpdate CmdTag = "update"
	TagDelete CmdTag = "delete"
)

// TypedQuery is an enriched form of TemplateQuery after running it on Postgres to get
// information about the TemplateQuery.
type TypedQuery struct {
	// Name of the query, from the comment preceding the query. Like 'FindAuthors'
	// in:
	//     -- name: FindAuthors :many
	Name string
	// The command tag that Postgres reports after running the query.
	Tag CmdTag
	// The SQL query, with pggen functions replaced with Postgres syntax. Ready
	// to run with PREPARE.
	PreparedSQL string
	// The input parameters to the query.
	Inputs []InputParam
	// The output columns of the query.
	Outputs []OutputColumn
}

// InputParam is an input parameter for a prepared query.
type InputParam struct {
	// Name of the param, like 'FirstName' in pggen.arg('FirstName').
	Name string
	// Default value to use for the param when executing the query on Postgres.
	// Like 'joe' in pggen.arg('FirstName', 'joe').
	DefaultVal string
	// The postgres type of this param as reported by Postgres.
	PgType string
	// The Go type to use generated for this param.
	GoType string
}

type OutputColumn struct {
	// Name of an output column, named by Postgres, like "foo" in "SELECT 1 as foo".
	PgName string
	// The postgres type of the column as reported by Postgres.
	PgType string
	// The Go type to use for the column.
	GoType string
}

type Inferrer struct {
	conn *pgx.Conn
}

// NewInferrer infers information about a query by running the query on
// Postgres and extracting information from the catalog tables.
func NewInferrer(conn *pgx.Conn) *Inferrer {
	return &Inferrer{conn: conn}
}

func (inf *Inferrer) InferTypes(query *ast.TemplateQuery) (TypedQuery, error) {
	return TypedQuery{
		Name:        query.Name,
		Tag:         TagSelect,
		PreparedSQL: query.PreparedSQL,
		Inputs:      nil,
		Outputs:     nil,
	}, nil
}
