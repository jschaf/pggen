package golang

import (
	"fmt"
	"github.com/jschaf/pggen/internal/ast"
	"github.com/jschaf/pggen/internal/codegen/golang/gotype"
	"github.com/jschaf/pggen/internal/pg"
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
	Pkg     TemplatedPackage // the parent package containing this file
	PkgPath string           // full package path, like "github.com/foo/bar"
	GoPkg   string           // the name of the Go package to use for the generated file
	Path    string           // the path to source SQL file
	Queries []TemplatedQuery // the queries with all template information
	Imports []string         // Go imports
	// True if this file is the leader file. The leader defines common code used
	// by by all queries in the same directory. Only one leader per directory.
	IsLeader bool
	// Any declarations this file should declare. Only set on leader.
	Declarers []Declarer
}

// TemplatedQuery is a query with all information required to execute the
// codegen template.
type TemplatedQuery struct {
	Name        string            // name of the query, from the comment preceding the query
	SQLVarName  string            // name of the string variable containing the SQL
	ResultKind  ast.ResultKind    // kind of result: :one, :many, or :exec
	Doc         string            // doc from the source query file, formatted for Go
	PreparedSQL string            // SQL query, ready to run with PREPARE statement
	Inputs      []TemplatedParam  // input parameters to the query
	Outputs     []TemplatedColumn // output columns of the query
}

type TemplatedParam struct {
	UpperName string // name of the param in UpperCamelCase, like 'FirstName' from pggen.arg('FirstName')
	LowerName string // name of the param in lowerCamelCase, like 'firstName' from pggen.arg('FirstName')
	QualType  string // package-qualified Go type to use for this param
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
	switch len(tq.Inputs) {
	case 0:
		return ""
	case 1, 2:
		sb := strings.Builder{}
		for _, input := range tq.Inputs {
			sb.WriteString(", ")
			sb.WriteString(input.LowerName)
			sb.WriteRune(' ')
			sb.WriteString(input.QualType)
		}
		return sb.String()
	default:
		return ", params " + tq.Name + "Params"
	}
}

func getLongestInput(inputs []TemplatedParam) int {
	max := 0
	for _, in := range inputs {
		if len(in.UpperName) > max {
			max = len(in.UpperName)
		}
	}
	return max
}

// EmitParamStruct emits the struct definition for query params if needed.
func (tq TemplatedQuery) EmitParamStruct() string {
	if len(tq.Inputs) < 3 {
		return ""
	}
	sb := &strings.Builder{}
	sb.WriteString("\n\ntype ")
	sb.WriteString(tq.Name)
	sb.WriteString("Params struct {\n")
	typeCol := getLongestInput(tq.Inputs) + 1 // 1 space
	for _, out := range tq.Inputs {
		sb.WriteString("\t")
		sb.WriteString(out.UpperName)
		sb.WriteString(strings.Repeat(" ", typeCol-len(out.UpperName)))
		sb.WriteString(out.QualType)
		sb.WriteRune('\n')
	}
	sb.WriteString("}")
	return sb.String()
}

// EmitParamNames emits the TemplatedQuery.Inputs into comma separated names
// for use in a method invocation.
func (tq TemplatedQuery) EmitParamNames() string {
	switch len(tq.Inputs) {
	case 0:
		return ""
	case 1, 2:
		sb := strings.Builder{}
		for _, input := range tq.Inputs {
			sb.WriteString(", ")
			sb.WriteString(input.LowerName)
		}
		return sb.String()
	default:
		sb := strings.Builder{}
		for _, input := range tq.Inputs {
			sb.WriteString(", params.")
			sb.WriteString(input.UpperName)
		}
		return sb.String()
	}
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

	hasOnlyOneNonVoid := len(removeVoidColumns(tq.Outputs)) == 1
	sb := strings.Builder{}
	sb.Grow(15 * len(tq.Outputs))
	for i, out := range tq.Outputs {
		switch typ := out.Type.(type) {
		case gotype.ArrayType:
			switch typ.Elem.(type) {
			case gotype.EnumType, gotype.CompositeType:
				sb.WriteString(out.LowerName)
				sb.WriteString("Array")
			default:
				if hasOnlyOneNonVoid {
					sb.WriteString("&item")
				} else {
					sb.WriteString("&item.")
					sb.WriteString(out.UpperName)
				}
			}

		case gotype.CompositeType:
			sb.WriteString(out.LowerName)
			sb.WriteString("Row")

		case gotype.VoidType:
			sb.WriteString("nil")

		case gotype.EnumType, gotype.OpaqueType:
			if hasOnlyOneNonVoid {
				sb.WriteString("&item")
			} else {
				sb.WriteString("&item.")
				sb.WriteString(out.UpperName)
			}

		default:
			return "", fmt.Errorf("unhandled type to emit row scan: %s %T", typ.BaseName(), typ)
		}
		if i < len(tq.Outputs)-1 {
			sb.WriteString(", ")
		}
	}
	return sb.String(), nil
}

// EmitResultType returns the string representing the overall query result type,
// meaning the return result.
func (tq TemplatedQuery) EmitResultType() (string, error) {
	outs := removeVoidColumns(tq.Outputs)
	switch tq.ResultKind {
	case ast.ResultKindExec:
		return "pgconn.CommandTag", nil
	case ast.ResultKindMany:
		switch len(outs) {
		case 0:
			return "pgconn.CommandTag", nil
		case 1:
			return "[]" + outs[0].QualType, nil
		default:
			return "[]" + tq.Name + "Row", nil
		}
	case ast.ResultKindOne:
		switch len(outs) {
		case 0:
			return "pgconn.CommandTag", nil
		case 1:
			return outs[0].QualType, nil
		default:
			return tq.Name + "Row", nil
		}
	default:
		return "", fmt.Errorf("unhandled EmitResultType kind: %s", tq.ResultKind)
	}
}

// EmitResultTypeInit returns the initialization code for the result type with
// name, typically "item" or "items". For array types, we take care to not use a
// var declaration so that JSON serialization returns an empty array instead of
// null.
func (tq TemplatedQuery) EmitResultTypeInit(name string) (string, error) {
	result, err := tq.EmitResultType()
	if err != nil {
		return "", fmt.Errorf("create result type for EmitResultTypeInit: %w", err)
	}
	if tq.ResultKind != ast.ResultKindMany && tq.ResultKind != ast.ResultKindOne {
		return "", fmt.Errorf("unhandled EmitResultTypeInit type %s for kind %s", result, tq.ResultKind)
	}
	isArr := strings.HasPrefix(result, "[]")
	if isArr {
		return name + " := " + result + "{}", nil
	}
	// Remove pointer. Return the right type by adding an address operator, "&",
	// where needed.
	raw := strings.TrimPrefix(result, "*")
	return "var " + name + " " + raw, nil
}

// EmitResultInits declares all initialization required for output types.
func (tq TemplatedQuery) EmitResultInits(pkgPath string) (string, error) {
	sb := &strings.Builder{}
	const indent = "\n\t" // 1 level indent inside querier method
	for _, out := range tq.Outputs {
		switch typ := out.Type.(type) {
		case gotype.CompositeType:
			// Declares pgtype.CompositeFields types for composite types in the output
			// columns. pggen uses pgtype.CompositeFields as args to the row scan
			// methods. Emits definitions like:
			//   userRow := pgtype.CompositeFields{
			//      &pgtype.Int8{},
			//      &pgtype.Text{},
			//   }
			sb.WriteString(indent)
			sb.WriteString(out.LowerName)
			sb.WriteString("Row := ")
			tq.appendResultCompositeInit(sb, pkgPath, typ, 1)
		case gotype.ArrayType:
			switch elem := typ.Elem.(type) {
			case gotype.EnumType:
				// Declares pgtype.EnumArray types for enum arrays in the output columns.
				// pggen uses pgtype.EnumArray as arg to the pgx row scan methods. Emits
				// definitions like:
				//   deviceTypeArray := &pgtype.EnumArray{}
				sb.WriteString(indent)
				sb.WriteString(out.LowerName)
				sb.WriteString("Array := &pgtype.EnumArray{}")

			case gotype.CompositeType:
				// Declares a composite type and an array type like:
				//   blockRow, _ := pgtype.NewCompositeTypeValues("blocks", []pgtype.CompositeTypeField{
				//   	{Name: "id", OID: pgtype.Int4OID},
				//   }, []pgtype.ValueTranscoder{
				//   	&pgtype.Int4{},
				//   })
				//   blockArray := pgtype.NewArrayType("_block", ignoredOID, func() pgtype.ValueTranscoder {
				//   	return blockRow.NewTypeValue().(*pgtype.CompositeType)
				//   })
				sb.WriteString(indent)
				rowName := out.LowerName + "Row"
				sb.WriteString(rowName)
				sb.WriteString(" := ")
				tq.appendResultCompositeInit(sb, pkgPath, elem, 1)
				sb.WriteString(indent)
				sb.WriteString(out.LowerName)
				sb.WriteString("Array := ")
				tq.appendResultArrayComposite(sb, typ, rowName)
			}
		default:
			continue
		}
	}
	return sb.String(), nil
}

func (tq TemplatedQuery) appendResultCompositeInit(
	sb *strings.Builder,
	pkgPath string,
	typ gotype.CompositeType,
	indent int,
) {
	sb.WriteString("newCompositeType(\n")
	sb.WriteString(strings.Repeat("\t", indent+1))
	sb.WriteByte('"')
	sb.WriteString(typ.PgComposite.Name)
	sb.WriteString("\",\n")
	sb.WriteString(strings.Repeat("\t", indent+1))
	sb.WriteString(`[]string{`)
	for i := range typ.FieldNames {
		sb.WriteByte('"')
		sb.WriteString(typ.PgComposite.ColumnNames[i])
		sb.WriteByte('"')
		if i < len(typ.FieldNames)-1 {
			sb.WriteString(", ")
		}
	}
	sb.WriteString("},")
	for _, fieldType := range typ.FieldTypes {
		sb.WriteString("\n")
		sb.WriteString(strings.Repeat("\t", indent+1)) // indent for method and slice literal
		switch fieldType := fieldType.(type) {
		case gotype.CompositeType:
			tq.appendResultCompositeInit(sb, pkgPath, fieldType, indent+1)
		case gotype.VoidType:
			// skip
		default:
			sb.WriteString("&") // pgx needs pointers to types
			// TODO: support builtin types and builtin wrappers that use a different
			// initialization syntax.
			pgType := fieldType.PgType()
			if pgType == nil || pgType == (pg.VoidType{}) {
				sb.WriteString("nil,")
			} else {
				// We need the pgx variant because it matches the interface expected by
				// newCompositeType, pgtype.ValueTranscoder.
				if decoderType, ok := gotype.FindKnownTypePgx(pgType.OID()); ok {
					fieldType = decoderType
				}
				sb.WriteString(fieldType.QualifyRel(pkgPath))
				sb.WriteString("{},")
			}
		}
	}
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("\t", indent))
	sb.WriteString(")")
	if indent > 1 {
		sb.WriteByte(',') // nested call so add trailing comma
	}
}

func (tq TemplatedQuery) appendResultArrayComposite(
	sb *strings.Builder,
	arr gotype.ArrayType,
	elemVarName string,
) {
	sb.WriteString(`pgtype.NewArrayType("`)
	sb.WriteString(arr.PgArray.Name)
	sb.WriteString(`", ignoredOID, func() pgtype.ValueTranscoder {`)
	sb.WriteString("\n\t\t")
	sb.WriteString("return ")
	sb.WriteString(elemVarName)
	sb.WriteString(".NewTypeValue().(*pgtype.CompositeType)\n\t")
	sb.WriteString("})")
}

// EmitResultAssigns writes all the assign statements after scanning the result
// from pgx.
//
// Copies pgtype.CompositeFields representing a Postgres composite type into the
// output struct.
//
// Copies pgtype.EnumArray fields into Go enum array types.
func (tq TemplatedQuery) EmitResultAssigns() (string, error) {
	sb := &strings.Builder{}
	indent := "\n\t"
	if tq.ResultKind == ast.ResultKindMany {
		indent += "\t" // a :many query processes items in a for loop
	}
	for _, out := range tq.Outputs {
		switch typ := out.Type.(type) {
		case gotype.CompositeType:
			sb.WriteString(indent)
			sb.WriteString(out.LowerName)
			sb.WriteString("Row.AssignTo(&item")
			if len(removeVoidColumns(tq.Outputs)) > 1 {
				sb.WriteRune('.')
				sb.WriteString(out.UpperName)
			}
			sb.WriteString(")")
		case gotype.ArrayType:
			switch typ.Elem.(type) {
			case gotype.EnumType:
				sb.WriteString(indent)
				sb.WriteString(out.LowerName)
				sb.WriteString("Array.AssignTo((*[]string)(unsafe.Pointer(&item")
				if len(removeVoidColumns(tq.Outputs)) > 1 {
					sb.WriteRune('.')
					sb.WriteString(out.UpperName)
				}
				sb.WriteString(")))")
				sb.WriteString(" // safe cast; enum array is []string")

			case gotype.CompositeType:
				sb.WriteString(indent)
				sb.WriteString(out.LowerName)
				sb.WriteString("Array.AssignTo(&item")
				if len(removeVoidColumns(tq.Outputs)) > 1 {
					sb.WriteRune('.')
					sb.WriteString(out.UpperName)
				}
				sb.WriteString(")")
			}
		}
	}
	return sb.String(), nil
}

// EmitResultElem returns the string representing a single item in the overall
// query result type. For :one and :exec queries, this is the same as
// EmitResultType. For :many queries, this is the element type of the slice
// result type.
func (tq TemplatedQuery) EmitResultElem() (string, error) {
	result, err := tq.EmitResultType()
	if err != nil {
		return "", fmt.Errorf("unhandled EmitResultElem type: %w", err)
	}
	// Unwrap arrays because we build the array with append.
	arr := strings.TrimPrefix(result, "[]")
	// Unwrap pointers because we add "&" to return the correct types.
	ptr := strings.TrimPrefix(arr, "*")
	return ptr, nil
}

// EmitResultExpr returns the string representation of a single item to return
// for :one queries or to append for :many queries. Useful for figuring out if
// we need to use the address operator. Controls the string item and &item in:
//
//   items = append(items, item)
//   items = append(items, &item)
func (tq TemplatedQuery) EmitResultExpr(name string) (string, error) {
	result, err := tq.EmitResultType()
	if err != nil {
		return "", fmt.Errorf("unhandled EmitResultExpr type: %w", err)
	}
	isPtr := strings.HasPrefix(result, "[]*") || strings.HasPrefix(result, "*")
	if isPtr {
		return "&" + name, nil
	}
	return name, nil
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
		outs := removeVoidColumns(tq.Outputs)
		if len(outs) <= 1 {
			return "" // if there's only 1 output column, return it directly
		}
		sb := &strings.Builder{}
		sb.WriteString("\n\ntype ")
		sb.WriteString(tq.Name)
		sb.WriteString("Row struct {\n")
		maxNameLen, maxTypeLen := getLongestOutput(outs)
		for _, out := range outs {
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

// removeVoidColumns makes a copy of cols with all VoidType columns removed.
// Useful because return types shouldn't contain the void type but we need
// to use a nil placeholder for void types when scanning a pgx.Row.
func removeVoidColumns(cols []TemplatedColumn) []TemplatedColumn {
	outs := make([]TemplatedColumn, 0, len(cols))
	for _, col := range cols {
		if _, ok := col.Type.(gotype.VoidType); ok {
			continue
		}
		outs = append(outs, col)
	}
	return outs
}
