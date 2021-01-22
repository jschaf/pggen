package pginfer

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
	"github.com/jschaf/sqld/errs"
	"github.com/jschaf/sqld/internal/ast"
	"time"
)

const defaultTimeout = 3 * time.Second

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
	inputs, err := inf.inferInputTypes(query)
	if err != nil {
		return TypedQuery{}, fmt.Errorf("infer input types for query %s: %w", query.Name, err)
	}
	// determine input types
	return TypedQuery{
		Name:        query.Name,
		Tag:         TagSelect,
		PreparedSQL: query.PreparedSQL,
		Inputs:      inputs,
		Outputs:     nil,
	}, nil
}

func (inf *Inferrer) inferInputTypes(query *ast.TemplateQuery) (inputs []InputParam, mErr error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	// Prepare the query so we can get the parameter types from Postgres.
	name := "pggen_" + query.Name
	prepareQuery := fmt.Sprintf(`PREPARE %s AS %s`, name, query.PreparedSQL)
	_, err := inf.conn.Exec(ctx, prepareQuery)
	if err != nil {
		return nil, fmt.Errorf("exec prepare statement to infer input query types for query %s: %w", query.Name, err)
	}
	// Deallocate in case we reuse this database.
	defer errs.Capture(&mErr,
		func() error { return inf.deallocatePreparedQuery(name) },
		"deallocate prepared query")

	ctx, cancel = context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	_, err = inf.conn.Exec(ctx, `SET search_path TO public`)
	if err != nil {
		return nil, fmt.Errorf("set search_path to public: %w", err)
	}

	// Get the parameter types from the pg_prepared_statements table.
	ctx, cancel = context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	catalogQuery := fmt.Sprintf(`
      SELECT parameter_types 
      FROM pg_prepared_statements 
      WHERE name = '%s';
  `, name)
	row := inf.conn.QueryRow(ctx, catalogQuery)
	types := make([]string, 0, len(query.ParamNames))
	if err := row.Scan(types); err != nil {
		return nil, fmt.Errorf("scan parameter_types for query %s: %w", query.Name, err)
	}
	if len(types) != len(query.ParamNames) {
		return nil, fmt.Errorf("expected %d parameter types for query %s; got %d",
			len(query.ParamNames), query.Name, len(types))
	}

	// Build up the input params.
	params := make([]InputParam, len(query.ParamNames))
	for i := 0; i < len(params); i++ {
		params[i].Name = query.ParamNames[i]
		params[i].PgType = types[i]
		params[i].GoType = chooseGoType(types[i])
	}
	return params, nil
}

func (inf *Inferrer) deallocatePreparedQuery(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	query := `DEALLOCATE ` + name
	_, err := inf.conn.Exec(ctx, query)
	return err
}

func chooseGoType(s string) string {
	switch s {
	case "text":
		return "string"
	default:
		return s
	}
}

func createParamArgs(query *ast.TemplateQuery) []interface{} {
	args := make([]interface{}, len(query.ParamNames))
	for i := range query.ParamNames {
		args[i] = nil
	}
	return args
}
