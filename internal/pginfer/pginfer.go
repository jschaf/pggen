package pginfer

import (
	"context"
	"fmt"
	"github.com/jackc/pgconn"
	"strings"
	"time"

	"github.com/jackc/pgproto3/v2"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/jschaf/pggen/internal/ast"
	"github.com/jschaf/pggen/internal/errs"
	"github.com/jschaf/pggen/internal/pg"
)

const defaultTimeout = 3 * time.Second

// TypedQuery is an enriched form of ast.SourceQuery after running it on
// Postgres to get information about the ast.SourceQuery.
type TypedQuery struct {
	// Name of the query, from the comment preceding the query. Like 'FindAuthors'
	// in the source SQL: "-- name: FindAuthors :many"
	Name string
	// The result output kind, :one, :many, or :exec.
	ResultKind ast.ResultKind
	// The comment lines preceding the query, without the SQL comment syntax and
	// excluding the :name line.
	Doc []string
	// The SQL query, with pggen functions replaced with Postgres syntax. Ready
	// to run on Postgres with the PREPARE statement.
	PreparedSQL string
	// The input parameters to the query.
	Inputs []InputParam
	// The output columns of the query.
	Outputs []OutputColumn
}

// InputParam is an input parameter for a prepared query.
type InputParam struct {
	// Name of the param, like 'FirstName' in pggen.arg('FirstName').
	PgName string
	// Default value to use for the param when executing the query on Postgres.
	// Like 'joe' in pggen.arg('FirstName', 'joe').
	DefaultVal string
	// The postgres type of this param as reported by Postgres.
	PgType pg.Type
}

// OutputColumn is a single column output from a select query or returning
// clause in an update, insert, or delete query.
type OutputColumn struct {
	// Name of an output column, named by Postgres, like "foo" in "SELECT 1 as foo".
	PgName string
	// The postgres type of the column as reported by Postgres.
	PgType pg.Type
	// If the type can be null; depends on the query. A column defined
	// with a NOT NULL constraint can still be null in the output with a left
	// join. Nullability is determined using rudimentary control-flow analysis.
	Nullable bool
}

type Inferrer struct {
	conn        *pgx.Conn
	typeFetcher *pg.TypeFetcher
}

// NewInferrer infers information about a query by running the query on
// Postgres and extracting information from the catalog tables.
func NewInferrer(conn *pgx.Conn) *Inferrer {
	return &Inferrer{
		conn:        conn,
		typeFetcher: pg.NewTypeFetcher(conn),
	}
}

func (inf *Inferrer) InferTypes(query *ast.SourceQuery) (TypedQuery, error) {
	inputs, err := inf.inferInputTypes(query)
	if err != nil {
		return TypedQuery{}, fmt.Errorf("infer input types for query: %w", err)
	}
	outputs, err := inf.inferOutputTypes(query)
	if err != nil {
		return TypedQuery{}, fmt.Errorf("infer output types for query: %w", err)
	}
	if query.ResultKind != ast.ResultKindExec && len(outputs) == 0 {
		return TypedQuery{}, fmt.Errorf(
			"query %s has incompatible result kind %s; the query doesn't return any columns; "+
				"use :exec if query shouldn't return any columns",
			query.Name, query.ResultKind)
	}
	if query.ResultKind != ast.ResultKindExec && countVoids(outputs) == len(outputs) {
		return TypedQuery{}, fmt.Errorf(
			"query %s has incompatible result kind %s; the query only has void columns; "+
				"use :exec if query shouldn't return any columns",
			query.Name, query.ResultKind)
	}
	doc := extractDoc(query)
	return TypedQuery{
		Name:        query.Name,
		ResultKind:  query.ResultKind,
		Doc:         doc,
		PreparedSQL: query.PreparedSQL,
		Inputs:      inputs,
		Outputs:     outputs,
	}, nil
}

func (inf *Inferrer) inferInputTypes(query *ast.SourceQuery) (ps []InputParam, mErr error) {
	if len(query.ParamNames) == 0 {
		return nil, nil
	}

	// Prepare the query so we can get the parameter types from Postgres.
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	prepareName := "pggen_" + query.Name
	prepareQuery := fmt.Sprintf(`PREPARE %s AS %s`, prepareName, query.PreparedSQL)
	_, err := inf.conn.Exec(ctx, prepareQuery)
	if err != nil {
		return nil, fmt.Errorf("exec prepare statement to infer input query types: %w", err)
	}
	defer errs.Capture(&mErr, func() error { return inf.deallocateQuery(prepareName) }, "")

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
	types, err := inf.typeFetcher.FindTypesByOIDs(oids...)
	if err != nil {
		return nil, fmt.Errorf("fetch oid types: %w", err)
	}

	// Build up the input params.
	params := make([]InputParam, len(query.ParamNames))
	for i := 0; i < len(params); i++ {
		pgType := types[pgtype.OID(oids[i])]
		params[i] = InputParam{
			PgName:     query.ParamNames[i],
			DefaultVal: "",
			PgType:     pgType,
		}
	}
	return params, nil
}

// Deallocates a prepared query. Implemented mostly for tests so we can reuse
// the same query name. Postgres doesn't allow PREPARE with duplicates.
func (inf *Inferrer) deallocateQuery(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	_, err := inf.conn.Exec(ctx, "DEALLOCATE "+name)
	if err != nil {
		return fmt.Errorf("deallocate query %s: %w", name, err)
	}
	return nil
}

func (inf *Inferrer) inferOutputTypes(query *ast.SourceQuery) ([]OutputColumn, error) {
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
	// We can ignore the error if we got the field descriptions. Most queries
	// will error with a not-null constraint violation since we use null for all
	// parameters in createParamArgs. For :exec queries, ignore the error since we
	// don't need the field descriptions.
	haveDescriptions := len(descriptions) > 0 || query.ResultKind == ast.ResultKindExec
	if err := rows.Err(); err != nil && !haveDescriptions {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			msg := "fetch field descriptions: " + pgErr.Message
			if pgErr.Where != "" {
				msg += "\n    WHERE: " + pgErr.Where
			}
			if pgErr.Detail != "" {
				msg += "\n    DETAIL: " + pgErr.Detail
			}
			if pgErr.Hint != "" {
				msg += "\n    HINT: " + pgErr.Hint
			}
			if pgErr.DataTypeName != "" {
				msg += "\n    DataType: " + pgErr.DataTypeName
			}
			if pgErr.TableName != "" {
				msg += "\n    TableName: " + pgErr.TableName
			}
			return nil, fmt.Errorf(msg+"\n    %w", pgErr)
		}
		return nil, fmt.Errorf("fetch field descriptions: %w", err)
	}

	// Resolve type names of output column data type OIDs.
	oids := make([]uint32, len(descriptions))
	for i, desc := range descriptions {
		oids[i] = desc.DataTypeOID
	}
	types, err := inf.typeFetcher.FindTypesByOIDs(oids...)
	if err != nil {
		return nil, fmt.Errorf("fetch oid types: %w", err)
	}

	// Output nullability.
	nullables, err := inf.inferOutputNullability(query, descriptions)
	if err != nil {
		return nil, fmt.Errorf("infer output type nullability: %w", err)
	}

	// Create output columns
	var outs []OutputColumn
	for i, desc := range descriptions {
		pgType, ok := types[pgtype.OID(desc.DataTypeOID)]
		if !ok {
			return nil, fmt.Errorf("no type name found for oid %d", desc.DataTypeOID)
		}

		outs = append(outs, OutputColumn{
			PgName:   string(desc.Name),
			PgType:   pgType,
			Nullable: nullables[i],
		})
	}
	return outs, nil
}

// inferOutputNullability infers which of the output columns produced by the
// query and described by descs can be null.
func (inf *Inferrer) inferOutputNullability(query *ast.SourceQuery, descs []pgproto3.FieldDescription) ([]bool, error) {
	if len(descs) == 0 {
		return nil, nil
	}
	plan, err := inf.explainQuery(query)
	if err != nil {
		return nil, err
	}

	columnKeys := make([]pg.ColumnKey, len(descs))
	for i, desc := range descs {
		if desc.TableOID > 0 {
			columnKeys[i] = pg.ColumnKey{
				TableOID: pgtype.OID(desc.TableOID),
				Number:   desc.TableAttributeNumber,
			}
		}
	}
	cols, err := pg.FetchColumns(inf.conn, columnKeys)
	if err != nil {
		return nil, fmt.Errorf("fetch column for nullability: %w", err)
	}

	// The nth entry determines if the output column described by descs[n] is
	// nullable. plan.Outputs might contain more entries than cols because the
	// plan output also contains information like sort columns.
	nullables := make([]bool, len(descs))
	for i := range nullables {
		nullables[i] = true // assume nullable until proven otherwise
	}
	for i, col := range cols {
		if i == len(plan.Outputs) {
			// plan.Outputs might not have the same output because the top level node
			// joins child outputs like with append.
			break
		}
		nullables[i] = isColNullable(query, plan, plan.Outputs[i], col)
	}
	return nullables, nil
}

func createParamArgs(query *ast.SourceQuery) []interface{} {
	args := make([]interface{}, len(query.ParamNames))
	for i := range query.ParamNames {
		args[i] = nil
	}
	return args
}

func extractDoc(query *ast.SourceQuery) []string {
	if query.Doc == nil || len(query.Doc.List) <= 1 {
		return nil
	}
	// Drop last line, like: "-- name: Foo :exec"
	lines := make([]string, len(query.Doc.List)-1)
	for i := range lines {
		comment := query.Doc.List[i].Text
		// TrimLeft to remove runs of dashes. TrimPrefix only removes fixed number.
		noDashes := strings.TrimLeft(comment, "-")
		lines[i] = strings.TrimSpace(noDashes)
	}
	return lines
}

func countVoids(outputs []OutputColumn) int {
	n := 0
	for _, out := range outputs {
		if _, ok := out.PgType.(pg.VoidType); ok {
			n++
		}
	}
	return n
}
