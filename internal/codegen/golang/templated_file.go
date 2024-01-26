package golang

import (
	"fmt"
	"github.com/jschaf/pggen/internal/ast"
	"github.com/jschaf/pggen/internal/codegen/golang/gotype"
	"github.com/jschaf/pggen/internal/pginfer"
	"strconv"
	"strings"
)

// TemplatedPackage is all templated files in a pggen invocation. The templated
// files do not necessarily reside in the same directory.
type TemplatedPackage struct {
	Files []TemplatedFile // sorted lexicographically by path
}

// TemplatedFile is the Go version of a SQL query file with all information
// needed to execute the codegen template.
type TemplatedFile struct {
	Pkg        TemplatedPackage // the parent package containing this file
	PkgPath    string           // full package path, like "github.com/foo/bar"
	GoPkg      string           // the name of the Go package to use for the "package foo" declaration
	SourcePath string           // absolute path to source SQL file
	Queries    []TemplatedQuery // the queries with all template information
	Imports    []string         // Go imports
	// True if this file is the leader file. The leader defines common code used
	// by all queries in the same directory. Only one leader per directory.
	IsLeader bool
	// Any declarations this file should declare. Only set on leader.
	Declarers []Declarer
}

// TemplatedQuery is a query with all information required to execute the
// codegen template.
type TemplatedQuery struct {
	Name             string            // name of the query, from the comment preceding the query
	SQLVarName       string            // name of the string variable containing the SQL
	ResultKind       ast.ResultKind    // kind of result: :one, :many, or :exec
	Doc              string            // doc from the source query file, formatted for Go
	PreparedSQL      string            // SQL query, ready to run with PREPARE statement
	Inputs           []TemplatedParam  // input parameters to the query
	Outputs          []TemplatedColumn // non-void output columns of the query
	ScanCols         []TemplatedColumn // all columns of the query, including void columns
	InlineParamCount int               // inclusive count of params that will be inlined
}

type TemplatedParam struct {
	UpperName string // name of the param in UpperCamelCase, like 'FirstName' from pggen.arg('first_name')
	LowerName string // name of the param in lowerCamelCase, like 'firstName' from pggen.arg('first_name')
	QualType  string // package-qualified Go type to use for this param
	Type      gotype.Type
	RawName   pginfer.InputParam
}

type TemplatedColumn struct {
	PgName    string // original name of the Postgres column
	UpperName string // name in Go-style (UpperCamelCase) to use for the column
	LowerName string // name in Go-style (lowerCamelCase)
	Type      gotype.Type
	QualType  string // package qualified Go type to use for the column, like "pgtype.Text"
}

func (tf TemplatedFile) needsPgconnImport() bool {
	if tf.IsLeader {
		// Leader files define genericConn.Exec which returns pgconn.CommandTag.
		return true
	}
	for _, query := range tf.Queries {
		if query.ResultKind == ast.ResultKindExec {
			return true // :exec queries return pgconn.CommandTag
		}
	}
	return false
}

// EmitPreparedSQL emits the prepared SQL query with appropriate quoting.
func (tq TemplatedQuery) EmitPreparedSQL() string {
	if strings.ContainsRune(tq.PreparedSQL, '`') {
		return strconv.Quote(tq.PreparedSQL)
	}
	return "`" + tq.PreparedSQL + "`"
}

// EmitParams emits the TemplatedQuery.Inputs into method parameters with both
// a name and type based on the number of params. For use in a method
// definition.
func (tq TemplatedQuery) EmitParams() string {
	if !tq.isInlineParams() {
		return ", params " + tq.Name + "Params"
	}
	sb := strings.Builder{}
	for _, input := range tq.Inputs {
		sb.WriteString(", ")
		sb.WriteString(input.LowerName)
		sb.WriteRune(' ')
		sb.WriteString(input.QualType)
	}
	return sb.String()
}

// getLongestInput returns the length of the longest name and type name in all
// columns. Useful for struct definition alignment.
func getLongestInput(inputs []TemplatedParam) (int, int) {
	nameLen := 0
	for _, out := range inputs {
		if len(out.UpperName) > nameLen {
			nameLen = len(out.UpperName)
		}
	}
	nameLen++ // 1 space to separate name from type

	typeLen := 0
	for _, out := range inputs {
		if len(out.QualType) > typeLen {
			typeLen = len(out.QualType)
		}
	}
	typeLen++ // 1 space to separate type from struct tags.

	return nameLen, typeLen
}

// EmitParamStruct emits the struct definition for query params if needed.
func (tq TemplatedQuery) EmitParamStruct() string {
	if tq.isInlineParams() {
		return ""
	}
	sb := &strings.Builder{}
	sb.WriteString("\n\ntype ")
	sb.WriteString(tq.Name)
	sb.WriteString("Params struct {\n")
	maxNameLen, maxTypeLen := getLongestInput(tq.Inputs)
	for _, out := range tq.Inputs {
		// Name
		sb.WriteString("\t")
		sb.WriteString(out.UpperName)
		// Type
		sb.WriteString(strings.Repeat(" ", maxNameLen-len(out.UpperName)))
		sb.WriteString(out.QualType)
		// JSON struct tag
		sb.WriteString(strings.Repeat(" ", maxTypeLen-len(out.QualType)))
		sb.WriteString("`json:")
		sb.WriteString(strconv.Quote(out.RawName.PgName))
		sb.WriteString("`")
		sb.WriteRune('\n')
	}
	sb.WriteString("}")
	return sb.String()
}

// EmitParamNames emits the TemplatedQuery.Inputs into comma separated names
// for use in a method invocation.
func (tq TemplatedQuery) EmitParamNames() string {
	appendParam := func(sb *strings.Builder, typ gotype.Type, name string) {
		switch typ := gotype.UnwrapNestedType(typ).(type) {
		case *gotype.CompositeType:
			sb.WriteString("q.types.")
			sb.WriteString(NameCompositeInitFunc(typ))
			sb.WriteString("(")
			sb.WriteString(name)
			sb.WriteString(")")
		case *gotype.ArrayType:
			if gotype.IsPgxSupportedArray(typ) {
				sb.WriteString(name)
				break
			}
			switch gotype.UnwrapNestedType(typ.Elem).(type) {
			case *gotype.CompositeType, *gotype.EnumType:
				sb.WriteString("q.types.")
				sb.WriteString(NameArrayInitFunc(typ))
				sb.WriteString("(")
				sb.WriteString(name)
				sb.WriteString(")")
			default:
				sb.WriteString(name)
			}
		default:
			sb.WriteString(name)
		}
	}
	switch {
	case tq.isInlineParams():
		sb := &strings.Builder{}
		for _, input := range tq.Inputs {
			sb.WriteString(", ")
			appendParam(sb, input.Type, input.LowerName)
		}
		return sb.String()
	default:
		sb := &strings.Builder{}
		for _, input := range tq.Inputs {
			sb.WriteString(", ")
			appendParam(sb, input.Type, "params."+input.UpperName)
		}
		return sb.String()
	}
}

func (tq TemplatedQuery) isInlineParams() bool {
	return len(tq.Inputs) <= tq.InlineParamCount
}

// EmitPlanScan emits the variable that hold the pgtype.ScanPlan for a query
// output.
func (tq TemplatedQuery) EmitPlanScan(idx int, out TemplatedColumn) (string, error) {
	switch tq.ResultKind {
	case ast.ResultKindExec:
		return "", fmt.Errorf("cannot EmitPlanScanArgs for :exec query %s", tq.Name)
	case ast.ResultKindMany, ast.ResultKindOne:
		break // okay
	default:
		return "", fmt.Errorf("unhandled EmitPlanScanArgs type: %s", tq.ResultKind)
	}
	return fmt.Sprintf("plan%d := planScan(pgtype.TextCodec{}, fds[%d], (*%s)(nil))", idx, idx, out.Type.BaseName()), nil
}

// EmitScanColumn emits scan call for a single TemplatedColumn.
func (tq TemplatedQuery) EmitScanColumn(idx int, out TemplatedColumn) (string, error) {
	sb := &strings.Builder{}
	_, _ = fmt.Fprintf(sb, "if err := plan%d.Scan(vals[%d], &item); err != nil {\n", idx, idx)
	sb.WriteString("\t\t\t")
	_, _ = fmt.Fprintf(sb, `return item, fmt.Errorf("scan %s.%s: %%w", err)`, tq.Name, out.PgName)
	sb.WriteString("\n")
	sb.WriteString("\t\t}")
	return sb.String(), nil
}

// EmitRowScanArgs emits the args to scan a single row from a pgx.Row or
// pgx.Rows.
func (tq TemplatedQuery) EmitRowScanArgs() (string, error) {
	switch tq.ResultKind {
	case ast.ResultKindExec:
		return "", fmt.Errorf("cannot EmitRowScanArgs for :exec query %s", tq.Name)
	case ast.ResultKindMany, ast.ResultKindOne:
		break // okay
	default:
		return "", fmt.Errorf("unhandled EmitRowScanArgs type: %s", tq.ResultKind)
	}

	hasOnlyOneOutput := len(tq.Outputs) == 1
	sb := strings.Builder{}
	sb.Grow(15 * len(tq.Outputs))
	for i, out := range tq.Outputs {
		switch typ := gotype.UnwrapNestedType(out.Type).(type) {
		case *gotype.ArrayType:
			switch gotype.UnwrapNestedType(typ.Elem).(type) {
			case *gotype.EnumType, *gotype.CompositeType:
				sb.WriteString(out.LowerName)
				sb.WriteString("Array")
			default:
				if hasOnlyOneOutput {
					sb.WriteString("&item")
				} else {
					sb.WriteString("&item.")
					sb.WriteString(out.UpperName)
				}
			}

		case *gotype.CompositeType:
			sb.WriteString(out.LowerName)
			sb.WriteString("Row")

		case *gotype.EnumType, *gotype.OpaqueType:
			if hasOnlyOneOutput {
				sb.WriteString("&item")
			} else {
				sb.WriteString("&item.")
				sb.WriteString(out.UpperName)
			}

		case *gotype.VoidType:
			sb.WriteString("nil")

		default:
			return "", fmt.Errorf("unhandled type to emit row scan: %s %T", typ.BaseName(), typ)
		}
		if i < len(tq.Outputs)-1 {
			sb.WriteString(", ")
		}
	}
	return sb.String(), nil
}

func (tq TemplatedQuery) EmitCollectionFunc() (string, error) {
	switch tq.ResultKind {
	case ast.ResultKindExec:
		return "", fmt.Errorf("cannot EmitCollectionFunc for :exec query %s", tq.Name)
	case ast.ResultKindMany:
		return "pgx.CollectRows", nil
	case ast.ResultKindOne:
		return "pgx.CollectExactlyOneRow", nil
	default:
		return "", fmt.Errorf("unhandled EmitCollectionFunc type: %s", tq.ResultKind)
	}
}

// EmitSingularResultType returns the string representing a single element
// of the overall query result type, like FindAuthorsRow when the overall return
// type is []FindAuthorsRow.
func (tq TemplatedQuery) EmitSingularResultType() string {
	if tq.ResultKind == ast.ResultKindExec {
		return "pgconn.CommandTag"
	}
	if len(tq.Outputs) == 0 {
		return "pgconn.CommandTag"
	}
	if len(tq.Outputs) == 1 {
		return tq.Outputs[0].QualType
	}
	return tq.Name + "Row"
}

// EmitResultType returns the string representing the overall query result type,
// meaning the return result.
func (tq TemplatedQuery) EmitResultType() (string, error) {
	switch tq.ResultKind {
	case ast.ResultKindExec:
		return "pgconn.CommandTag", nil
	case ast.ResultKindMany:
		return "[]" + tq.EmitSingularResultType(), nil
	case ast.ResultKindOne:
		return tq.EmitSingularResultType(), nil
	default:
		return "", fmt.Errorf("unhandled EmitResultType kind: %s", tq.ResultKind)
	}
}

// EmitResultTypeInit returns the initialization code for the result type with
// name, typically "item" or "items". For array types, we take care to not use a
// var declaration so that JSON serialization returns an empty array instead of
// null.
func (tq TemplatedQuery) EmitResultTypeInit(name string) (string, error) {
	switch tq.ResultKind {
	case ast.ResultKindOne:
		result, err := tq.EmitResultType()
		if err != nil {
			return "", fmt.Errorf("create result type for EmitResultTypeInit: %w", err)
		}
		isArr := strings.HasPrefix(result, "[]")
		if isArr {
			return name + " := " + result + "{}", nil
		}
		return "var " + name + " " + result, nil

	case ast.ResultKindMany:
		result, err := tq.EmitResultType()
		if err != nil {
			return "", fmt.Errorf("create result type for EmitResultTypeInit: %w", err)
		}
		isArr := strings.HasPrefix(result, "[]")
		if isArr {
			return name + " := " + result + "{}", nil
		}
		// Remove pointer. Return the right type by adding an address operator, "&",
		// where needed.
		result = strings.TrimPrefix(result, "*")
		return "var " + name + " " + result, nil

	default:
		return "", fmt.Errorf("unhandled EmitResultTypeInit for kind %s", tq.ResultKind)
	}
}

// EmitZeroResult returns the string representing the zero value of a result.
func (tq TemplatedQuery) EmitZeroResult() (string, error) {
	switch tq.ResultKind {
	case ast.ResultKindExec:
		return "pgconn.CommandTag{}", nil
	case ast.ResultKindMany:
		return "nil", nil // empty slice
	case ast.ResultKindOne:
		if len(tq.Outputs) > 1 {
			return tq.Name + "Row{}", nil
		}
		typ := tq.Outputs[0].Type.BaseName()
		switch {
		case strings.HasPrefix(typ, "[]"):
			return "nil", nil // empty slice
		case strings.HasPrefix(typ, "*"):
			return "nil", nil // nil pointer
		}
		switch typ {
		case "int", "int32", "int64", "float32", "float64":
			return "0", nil
		case "string":
			return `""`, nil
		case "bool":
			return "false", nil
		default:
			return typ + "{}", nil // won't work for type Foo int
		}
	default:
		return "", fmt.Errorf("unhandled EmitZeroResult kind: %s", tq.ResultKind)
	}
}

// EmitResultExpr returns the string representation of a single item to return
// for :one queries or to append for :many queries. Useful for figuring out if
// we need to use the address operator. Controls the string item and &item in:
//
//	items = append(items, item)
//	items = append(items, &item)
func (tq TemplatedQuery) EmitResultExpr(name string) (string, error) {
	switch tq.ResultKind {
	case ast.ResultKindOne:
		return name, nil

	case ast.ResultKindMany:
		result, err := tq.EmitResultType()
		if err != nil {
			return "", fmt.Errorf("unhandled EmitResultExpr type: %w", err)
		}
		isPtr := strings.HasPrefix(result, "[]*") || strings.HasPrefix(result, "*")
		if isPtr {
			return "&" + name, nil
		}
		return name, nil

	default:
		return "", fmt.Errorf("unhandled EmitResultExpr type: %s", tq.ResultKind)
	}
}

// getLongestOutput returns the length of the longest name and type name in all
// columns. Useful for struct definition alignment.
func getLongestOutput(outs []TemplatedColumn) (int, int) {
	nameLen := 0
	for _, out := range outs {
		if len(out.UpperName) > nameLen {
			nameLen = len(out.UpperName)
		}
	}
	nameLen++ // 1 space to separate name from type

	typeLen := 0
	for _, out := range outs {
		if len(out.QualType) > typeLen {
			typeLen = len(out.QualType)
		}
	}
	typeLen++ // 1 space to separate type from struct tags.

	return nameLen, typeLen
}

// EmitRowStruct writes the struct definition for query output row if one is
// needed.
func (tq TemplatedQuery) EmitRowStruct() string {
	switch tq.ResultKind {
	case ast.ResultKindExec:
		return ""
	case ast.ResultKindOne, ast.ResultKindMany:
		if len(tq.Outputs) == 1 {
			return "" // if there's only 1 output column, return it directly
		}
		sb := &strings.Builder{}
		sb.WriteString("\n\ntype ")
		sb.WriteString(tq.Name)
		sb.WriteString("Row struct {\n")
		maxNameLen, maxTypeLen := getLongestOutput(tq.Outputs)
		for _, out := range tq.Outputs {
			// Name
			sb.WriteString("\t")
			sb.WriteString(out.UpperName)
			// Type
			sb.WriteString(strings.Repeat(" ", maxNameLen-len(out.UpperName)))
			sb.WriteString(out.QualType)
			// JSON struct tag
			sb.WriteString(strings.Repeat(" ", maxTypeLen-len(out.QualType)))
			sb.WriteString("`json:")
			sb.WriteString(strconv.Quote(out.PgName))
			sb.WriteString("`")
			sb.WriteRune('\n')
		}
		sb.WriteString("}")
		return sb.String()
	default:
		panic("unhandled result type: " + tq.ResultKind)
	}
}
