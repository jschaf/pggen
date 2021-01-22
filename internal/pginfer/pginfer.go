package pginfer

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
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
	// The Go name to use for the column.
	GoName string
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
	outputs, err := inf.inferOutputTypes(query)
	if err != nil {
		return TypedQuery{}, fmt.Errorf("infer output types for query %s: %w", query.Name, err)
	}
	return TypedQuery{
		Name:        query.Name,
		Tag:         TagSelect,
		PreparedSQL: query.PreparedSQL,
		Inputs:      inputs,
		Outputs:     outputs,
	}, nil
}

func (inf *Inferrer) inferInputTypes(query *ast.TemplateQuery) ([]InputParam, error) {
	// Prepare the query so we can get the parameter types from Postgres.
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	prepareName := "pggen_" + query.Name
	prepareQuery := fmt.Sprintf(`PREPARE %s AS %s`, prepareName, query.PreparedSQL)
	_, err := inf.conn.Exec(ctx, prepareQuery)
	if err != nil {
		return nil, fmt.Errorf("exec prepare statement to infer input query types: %w", err)
	}

	// Get the parameter types from the pg_prepared_statements table.
	ctx, cancel = context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	catalogQuery := `SELECT parameter_types::text[] FROM pg_prepared_statements WHERE lower(name) = lower($1)`
	row := inf.conn.QueryRow(ctx, catalogQuery, prepareName)
	types := make([]string, 0, len(query.ParamNames))
	if err := row.Scan(&types); err != nil {
		return nil, fmt.Errorf("scan prepared parameter types: %w", err)
	}
	if len(types) != len(query.ParamNames) {
		return nil, fmt.Errorf("expected %d parameter types for query %s; got %d",
			len(query.ParamNames), query.Name, len(types))
	}

	// Build up the input params, mapping from Postgres types to Go types.
	params := make([]InputParam, len(query.ParamNames))
	for i := 0; i < len(params); i++ {
		params[i].Name = query.ParamNames[i]
		params[i].PgType = types[i]
		params[i].GoType = chooseGoType(types[i])
	}
	return params, nil
}

func (inf *Inferrer) inferOutputTypes(query *ast.TemplateQuery) ([]OutputColumn, error) {
	if hasOutput, err := inf.hasOutput(query); err != nil {
		return nil, fmt.Errorf("check query has output: %w", err)
	} else if !hasOutput {
		// If the query has no output, we don't have to infer the output types.
		return nil, nil
	}

	// Create a temp table from the query to determine the output params.
	// https://stackoverflow.com/questions/65733271
	tblName := `pggen_table_` + query.Name
	ctasQuery := fmt.Sprintf(`CREATE TEMP TABLE %s AS %s`, tblName, query.PreparedSQL)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	_, err := inf.conn.Exec(ctx, ctasQuery, createParamArgs(query)...)
	if err != nil {
		return nil, fmt.Errorf("create temp table %s: %w", tblName, err)
	}

	// Query the pg_attribute table to get the columns of the temp table, which
	// correspond to the output columns of the original query.
	attrQuery := `
		SELECT attname, format_type(atttypid, atttypmod) AS type
			FROM pg_attribute
		WHERE attrelid = $1::regclass
		AND attnum > 0
		AND NOT attisdropped
		ORDER BY attnum;
  `
	ctx, cancel = context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	rows, err := inf.conn.Query(ctx, attrQuery, tblName)
	if err != nil {
		return nil, fmt.Errorf("query temp table %s attributes: %w", tblName, err)
	}
	outCols := make([]OutputColumn, 0, 4)
	for rows.Next() {
		out := OutputColumn{}
		if err := rows.Scan(&out.PgName, &out.PgType); err != nil {
			return nil, fmt.Errorf("scan temp table attributes: %w", err)
		}
		out.GoType = chooseGoType(out.PgType)
		out.GoName = chooseGoName(out.PgName)
		outCols = append(outCols, out)
	}
	return outCols, nil
}

// hasOutput explains the query to determine if it has any output columns.
func (inf *Inferrer) hasOutput(query *ast.TemplateQuery) (bool, error) {
	explainQuery := `EXPLAIN (VERBOSE, FORMAT JSON) ` + query.PreparedSQL
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	row := inf.conn.QueryRow(ctx, explainQuery, createParamArgs(query)...)
	explain := make([]map[string]map[string]interface{}, 0, 1)
	if err := row.Scan(&explain); err != nil {
		return false, fmt.Errorf("explain prepared query: %w", err)
	}
	if len(explain) == 0 {
		return false, fmt.Errorf("no explain output")
	}
	plan, ok := explain[0]["Plan"]
	if !ok {
		return false, fmt.Errorf("explain output no 'Plan' node")
	}
	rawOuts, ok := plan["Output"]
	if !ok {
		return false, nil
	}
	outs, ok := rawOuts.([]interface{})
	if !ok {
		return false, fmt.Errorf("explain output 'Plan.Output' is not []interface")
	}
	return len(outs) > 0, nil
}

func chooseGoName(s string) string {
	return s
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
