package pginfer

import (
	"context"
	"fmt"
	"github.com/jackc/pgproto3/v2"
	"github.com/jackc/pgx/v4"
	"github.com/jschaf/sqld/internal/ast"
	"github.com/jschaf/sqld/internal/pg"
	"time"
)

const defaultTimeout = 3 * time.Second

// TypedQuery is an enriched form of SourceQuery after running it on Postgres
// to get information about the SourceQuery.
type TypedQuery struct {
	// Name of the query, from the comment preceding the query. Like 'FindAuthors'
	// in:
	//     -- name: FindAuthors :many
	Name string
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
	PgType pg.Type
	// The Go type to use generated for this param.
	GoType string
}

type OutputColumn struct {
	// Name of an output column, named by Postgres, like "foo" in "SELECT 1 as foo".
	PgName string
	// The Go name to use for the column.
	GoName string
	// The postgres type of the column as reported by Postgres.
	PgType pg.Type
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

func (inf *Inferrer) InferTypes(query *ast.SourceQuery) (TypedQuery, error) {
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
		PreparedSQL: query.PreparedSQL,
		Inputs:      inputs,
		Outputs:     outputs,
	}, nil
}

func (inf *Inferrer) inferInputTypes(query *ast.SourceQuery) ([]InputParam, error) {
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
	catalogQuery := `SELECT parameter_types::int[] FROM pg_prepared_statements WHERE lower(name) = lower($1)`
	row := inf.conn.QueryRow(ctx, catalogQuery, prepareName)
	oids := make([]uint32, 0, len(query.ParamNames))
	if err := row.Scan(&oids); err != nil {
		return nil, fmt.Errorf("scan prepared parameter types: %w", err)
	}
	if len(oids) != len(query.ParamNames) {
		return nil, fmt.Errorf("expected %d parameter types for query; got %d",
			len(query.ParamNames), len(oids))
	}
	types, err := pg.FetchOIDTypes(inf.conn, oids...)
	if err != nil {
		return nil, fmt.Errorf("fetch oid types: %w", err)
	}

	// Build up the input params, mapping from Postgres types to Go types.
	params := make([]InputParam, len(query.ParamNames))
	for i := 0; i < len(params); i++ {
		pgType := types[oids[i]]
		params[i] = InputParam{
			Name:       query.ParamNames[i],
			DefaultVal: "",
			PgType:     pgType,
			GoType:     pgToGoType(pgType),
		}
	}
	return params, nil
}

func (inf *Inferrer) inferOutputTypes(query *ast.SourceQuery) ([]OutputColumn, error) {
	// If the query has no output, we don't have to infer the output types.
	if hasOutput, err := inf.hasOutput(query); err != nil {
		return nil, fmt.Errorf("check query has output: %w", err)
	} else if !hasOutput {
		return nil, nil
	}

	// Execute the query to get field descriptions of the output columns.
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	rows, err := inf.conn.Query(ctx, query.PreparedSQL, createParamArgs(query)...)
	if err != nil {
		return nil, fmt.Errorf("execute output query: %w", err)
	}
	descriptions := make([]pgproto3.FieldDescription, len(rows.FieldDescriptions()))
	copy(descriptions, rows.FieldDescriptions()) // pgx reuses row objects
	rows.Close()

	// Resolve type names of output column data type OIDs.
	typeOIDs := make([]uint32, len(descriptions))
	for i, desc := range descriptions {
		typeOIDs[i] = desc.DataTypeOID
	}
	types, err := pg.FetchOIDTypes(inf.conn, typeOIDs...)
	if err != nil {
		return nil, fmt.Errorf("fetch oid types: %w", err)
	}

	// Create output columns
	outs := make([]OutputColumn, len(descriptions))
	for i, desc := range descriptions {
		pgType, ok := types[desc.DataTypeOID]
		if !ok {
			return nil, fmt.Errorf("no type name found for oid %d", desc.DataTypeOID)
		}
		outs[i] = OutputColumn{
			PgName: string(desc.Name),
			GoName: chooseGoName(string(desc.Name)),
			PgType: pgType,
			GoType: pgToGoType(pgType),
		}
		typeOIDs[i] = desc.DataTypeOID
	}
	return outs, nil
}

// hasOutput explains the query to determine if it has any output columns.
func (inf *Inferrer) hasOutput(query *ast.SourceQuery) (bool, error) {
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

func createParamArgs(query *ast.SourceQuery) []interface{} {
	args := make([]interface{}, len(query.ParamNames))
	for i := range query.ParamNames {
		args[i] = nil
	}
	return args
}
