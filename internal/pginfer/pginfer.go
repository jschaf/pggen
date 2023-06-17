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
	// Qualified protocol buffer message type to use for each output row, like
	// "erp.api.Product". If empty, generate our own Row type.
	ProtobufType string
}

// InputParam is an input parameter for a prepared query.
type InputParam struct {
	// Name of the param, like 'FirstName' in pggen.arg('FirstName').
	PgName string
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
	inputs, outputs, err := inf.prepareTypes(query)
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
		Name:         query.Name,
		ResultKind:   query.ResultKind,
		Doc:          doc,
		PreparedSQL:  query.PreparedSQL,
		Inputs:       inputs,
		Outputs:      outputs,
		ProtobufType: query.Pragmas.ProtobufType,
	}, nil
}

func (inf *Inferrer) prepareTypes(query *ast.SourceQuery) (_a []InputParam, _ []OutputColumn, mErr error) {
	// Execute the query to get field descriptions of the output columns.
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	// If paramOIDs is null, Postgres infers the type for each parameter.
	var paramOIDs []uint32
	stmtDesc, err := inf.conn.PgConn().Prepare(ctx, "", query.PreparedSQL, paramOIDs)
	if err != nil {
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
			// Provide hint to use a returning clause. pggen ignores most errors but
			// only if there's output columns. If the user has an UPDATE or INSERT
			// without a RETURNING clause, pggen will surface the null constraint
			// errors because len(descriptions) == 0.
			if strings.Contains(strings.ToLower(query.PreparedSQL), "update") ||
				strings.Contains(strings.ToLower(query.PreparedSQL), "insert") {
				msg += "\n    HINT: if the main statement is an UPDATE or INSERT ensure that you have"
				msg += "\n          a RETURNING clause (this query is marked " + string(query.ResultKind) + ")."
				msg += "\n          Use :exec if you don't need the query output."
			}
			return nil, nil, fmt.Errorf(msg+"\n    %w", pgErr)
		}
		return nil, nil, fmt.Errorf("prepare query to infer types: %w", err)
	}

	// Validate.
	if len(stmtDesc.ParamOIDs) != len(query.ParamNames) {
		return nil, nil, fmt.Errorf("expected %d parameter types for query; got %d", len(query.ParamNames), len(stmtDesc.ParamOIDs))
	}

	// Build input params.
	var inputParams []InputParam
	if len(stmtDesc.ParamOIDs) > 0 {
		types, err := inf.typeFetcher.FindTypesByOIDs(stmtDesc.ParamOIDs...)
		if err != nil {
			return nil, nil, fmt.Errorf("fetch oid types: %w", err)
		}
		for i, oid := range stmtDesc.ParamOIDs {
			inputType, ok := types[pgtype.OID(oid)]
			if !ok {
				return nil, nil, fmt.Errorf("no postgres type name found for parameter %s with oid %d", query.ParamNames[i], oid)
			}
			inputParams = append(inputParams, InputParam{
				PgName: query.ParamNames[i],
				PgType: inputType,
			})
		}
	}

	// Resolve type names of output column data type OIDs.
	outputOIDs := make([]uint32, len(stmtDesc.Fields))
	for i, desc := range stmtDesc.Fields {
		outputOIDs[i] = desc.DataTypeOID
	}
	outputTypes, err := inf.typeFetcher.FindTypesByOIDs(outputOIDs...)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch oid types: %w", err)
	}

	// Output nullability.
	nullables, err := inf.inferOutputNullability(query, stmtDesc.Fields)
	if err != nil {
		return nil, nil, fmt.Errorf("infer output type nullability: %w", err)
	}

	// Create output columns
	var outputColumns []OutputColumn
	for i, desc := range stmtDesc.Fields {
		pgType, ok := outputTypes[pgtype.OID(desc.DataTypeOID)]
		if !ok {
			return nil, nil, fmt.Errorf("no postgrestype name found for column %s with oid %d", string(desc.Name), desc.DataTypeOID)
		}
		outputColumns = append(outputColumns, OutputColumn{
			PgName:   string(desc.Name),
			PgType:   pgType,
			Nullable: nullables[i],
		})
	}
	return inputParams, outputColumns, nil
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
