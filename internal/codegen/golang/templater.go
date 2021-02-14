package golang

import (
	"fmt"
	"github.com/jschaf/pggen/internal/ast"
	"github.com/jschaf/pggen/internal/casing"
	"github.com/jschaf/pggen/internal/codegen"
	"github.com/jschaf/pggen/internal/codegen/golang/gotype"
	"github.com/jschaf/pggen/internal/gomod"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// TemplatedFile is the Go version of a SQL query file with all information
// needed to execute the codegen template.
type TemplatedFile struct {
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

// TemplaterOpts is options to control the template logic.
type TemplaterOpts struct {
	Caser    casing.Caser
	Resolver TypeResolver
	Pkg      string // Go package name
}

// Templater creates query file templates.
type Templater struct {
	caser    casing.Caser
	resolver TypeResolver
	pkg      string // Go package name
}

func NewTemplater(opts TemplaterOpts) Templater {
	return Templater{
		pkg:      opts.Pkg,
		caser:    opts.Caser,
		resolver: opts.Resolver,
	}
}

// TemplateAll creates query template files for each codegen.QueryFile.
func (tm Templater) TemplateAll(files []codegen.QueryFile) ([]TemplatedFile, error) {
	goQueryFiles := make([]TemplatedFile, 0, len(files))
	declarers := make([]Declarer, 0, 8)

	for _, queryFile := range files {
		goFile, decls, err := tm.templateFile(queryFile)
		if err != nil {
			return nil, fmt.Errorf("template query file %s for go: %w", queryFile.Path, err)
		}
		goQueryFiles = append(goQueryFiles, goFile)
		declarers = append(declarers, decls...)
	}

	// Pick leader file to define common structs and interfaces via Declarer.
	firstIndex := -1
	firstName := string(unicode.MaxRune)
	for i, goFile := range goQueryFiles {
		if goFile.Path < firstName {
			firstIndex = i
			firstName = goFile.Path
		}
	}
	goQueryFiles[firstIndex].IsLeader = true
	// Add declarers to the leader in a stable sort order, removing duplicates.
	if len(declarers) > 0 {
		sort.Slice(declarers, func(i, j int) bool { return declarers[i].DedupeKey() < declarers[j].DedupeKey() })
		dedupeLen := 1
		for i := 1; i < len(declarers); i++ {
			if declarers[i].DedupeKey() == declarers[dedupeLen-1].DedupeKey() {
				continue
			}
			declarers[dedupeLen] = declarers[i]
			dedupeLen++
		}
		goQueryFiles[firstIndex].Declarers = declarers[:dedupeLen]
	}

	// Remove unneeded pgconn import if possible.
	for i, file := range goQueryFiles {
		if file.needsPgconnImport() {
			continue
		}
		pgconnIdx := -1
		imports := file.Imports
		for i, pkg := range imports {
			if pkg == "github.com/jackc/pgconn" {
				pgconnIdx = i
				break
			}
		}
		if pgconnIdx > -1 {
			copy(imports[pgconnIdx:], imports[pgconnIdx+1:])
			goQueryFiles[i].Imports = imports[:len(imports)-1]
		}
	}
	// Remove self imports.
	for i, file := range goQueryFiles {
		selfPkg, err := gomod.GuessPackage(file.Path)
		if err != nil || selfPkg == "" {
			continue // ignore error, assume it's not a self import
		}
		selfPkgIdx := -1
		imports := file.Imports
		for i, pkg := range file.Imports {
			if pkg == selfPkg {
				selfPkgIdx = i
				break
			}
		}
		if selfPkgIdx > -1 {
			copy(imports[selfPkgIdx:], imports[selfPkgIdx+1:])
			goQueryFiles[i].Imports = imports[:len(imports)-1]
		}
	}
	return goQueryFiles, nil
}

// templateFile creates the data needed to build a Go file for a query file.
// Also returns any declarations needed by this query file. The caller must
// dedupe declarations.
func (tm Templater) templateFile(file codegen.QueryFile) (TemplatedFile, []Declarer, error) {
	imports := NewImportSet()
	imports.AddPackage("context")
	imports.AddPackage("fmt")
	imports.AddPackage("github.com/jackc/pgconn")
	imports.AddPackage("github.com/jackc/pgx/v4")

	pkgPath := ""
	// NOTE: err == nil check
	// Attempt to guess package path. Ignore error if it doesn't work because
	// resolving the package isn't perfect. We'll fallback to an unqualified
	// type which will likely work since the type is probably declared in this
	// package.
	if pkg, err := gomod.GuessPackage(file.Path); err == nil {
		pkgPath = pkg
	}

	queries := make([]TemplatedQuery, 0, len(file.Queries))
	declarers := make([]Declarer, 0, 8)
	for _, query := range file.Queries {
		// Build doc string.
		docs := strings.Builder{}
		avgCharsPerLine := 40
		docs.Grow(len(query.Doc) * avgCharsPerLine)
		for i, d := range query.Doc {
			if i > 0 {
				docs.WriteByte('\t') // first line is already indented in the template
			}
			docs.WriteString("// ")
			docs.WriteString(d)
			docs.WriteRune('\n')
		}

		// Build inputs.
		inputs := make([]TemplatedParam, len(query.Inputs))
		for i, input := range query.Inputs {
			goType, err := tm.resolver.Resolve(input.PgType /*nullable*/, false, pkgPath)
			if err != nil {
				return TemplatedFile{}, nil, err
			}
			imports.AddType(goType)
			inputs[i] = TemplatedParam{
				UpperName: tm.chooseUpperName(input.PgName, "UnnamedParam", i, len(query.Inputs)),
				LowerName: tm.chooseLowerName(input.PgName, "unnamedParam", i, len(query.Inputs)),
				QualType:  goType.QualifyRel(pkgPath),
			}
			declarers = append(declarers, FindDeclarers(goType)...)
		}

		// Build outputs.
		outputs := make([]TemplatedColumn, len(query.Outputs))
		for i, out := range query.Outputs {
			goType, err := tm.resolver.Resolve(out.PgType, out.Nullable, pkgPath)
			if err != nil {
				return TemplatedFile{}, nil, err
			}
			imports.AddType(goType)
			outputs[i] = TemplatedColumn{
				PgName:    out.PgName,
				UpperName: tm.chooseUpperName(out.PgName, "UnnamedColumn", i, len(query.Outputs)),
				LowerName: tm.chooseLowerName(out.PgName, "UnnamedColumn", i, len(query.Outputs)),
				Type:      goType,
				QualType:  goType.QualifyRel(pkgPath),
			}
			declarers = append(declarers, FindDeclarers(goType)...)
		}

		queries = append(queries, TemplatedQuery{
			Name:        tm.caser.ToUpperGoIdent(query.Name),
			SQLVarName:  tm.caser.ToLowerGoIdent(query.Name) + "SQL",
			ResultKind:  query.ResultKind,
			Doc:         docs.String(),
			PreparedSQL: query.PreparedSQL,
			Inputs:      inputs,
			Outputs:     outputs,
		})
	}

	return TemplatedFile{
		PkgPath: pkgPath,
		GoPkg:   tm.pkg,
		Path:    file.Path,
		Queries: queries,
		Imports: imports.SortedPackages(),
	}, declarers, nil
}

// chooseUpperName converts pgName into an capitalized Go identifier name.
// If it's not possible to convert pgName into an identifier, uses fallback with
// a suffix using idx.
func (tm Templater) chooseUpperName(pgName string, fallback string, idx int, numOptions int) string {
	if name := tm.caser.ToUpperGoIdent(pgName); name != "" {
		return name
	}
	suffix := strconv.Itoa(idx)
	if numOptions > 9 {
		suffix = fmt.Sprintf("%2d", idx)
	}
	return fallback + suffix
}

// chooseLowerName converts pgName into an uncapitalized Go identifier name.
// If it's not possible to convert pgName into an identifier, uses fallback with
// a suffix using idx.
func (tm Templater) chooseLowerName(pgName string, fallback string, idx int, numOptions int) string {
	if name := tm.caser.ToLowerGoIdent(pgName); name != "" {
		return name
	}
	suffix := strconv.Itoa(idx)
	if numOptions > 9 {
		suffix = fmt.Sprintf("%2d", idx)
	}
	return fallback + suffix
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
		hasOnlyOneNonVoid := len(removeVoidColumns(tq.Outputs)) == 1
		sb := strings.Builder{}
		sb.Grow(15 * len(tq.Outputs))
		for i, out := range tq.Outputs {
			switch out.Type.(type) {
			case gotype.CompositeType:
				sb.WriteString(out.LowerName)
				sb.WriteString("Row")
			case gotype.VoidType:
				sb.WriteString("nil")
			default:
				if hasOnlyOneNonVoid {
					sb.WriteString("&item")
				} else {
					sb.WriteString("&item.")
					sb.WriteString(out.UpperName)
				}
			}
			if i < len(tq.Outputs)-1 {
				sb.WriteString(", ")
			}
		}
		return sb.String(), nil
	default:
		return "", fmt.Errorf("unhandled EmitRowScanArgs type: %s", tq.ResultKind)
	}
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
	if strings.HasPrefix(result, "[]") {
		return name + " := " + result + "{}", nil
	}
	return "var " + name + " " + result, nil
}

// EmitResultCompositeInits declares all pgtype.CompositeFields for composite
// types in the output columns. pggen uses pgtype.CompositeFields as args to the
// row scan methods.
func (tq TemplatedQuery) EmitResultCompositeInits(pkgPath string) (string, error) {
	sb := &strings.Builder{}
	for _, out := range tq.Outputs {
		typ, ok := out.Type.(gotype.CompositeType)
		if !ok {
			continue
		}
		// Emit definition like:
		//   userRow := pgtype.CompositeFields{
		//      &pgtype.Int8{},
		//      &pgtype.Text{},
		//   }
		sb.WriteString("\n\t") // 1 level indent inside querier method
		sb.WriteString(out.LowerName)
		sb.WriteString("Row := ")
		tq.appendResultCompositeInit(sb, pkgPath, typ, 0)
	}
	return sb.String(), nil
}

// appendResultCompositeInit appends the pgtype.CompositeFields declaration to
// a string builder, recursively appending child fields of the composite type.
func (tq TemplatedQuery) appendResultCompositeInit(
	sb *strings.Builder,
	pkgPath string,
	typ gotype.CompositeType,
	indent int,
) {
	sb.WriteString("pgtype.CompositeFields{")
	for _, fieldType := range typ.FieldTypes {
		sb.WriteString("\n")
		sb.WriteString(strings.Repeat("\t", indent+2)) // indent for method and slice literal
		switch child := fieldType.(type) {
		case gotype.CompositeType:
			tq.appendResultCompositeInit(sb, pkgPath, child, indent+1)
		case gotype.VoidType:
			// skip
		default:
			sb.WriteString("&") // pgx needs pointers to types
			// TODO: support builtin types and builtin wrappers that use a different
			// initialization syntax.
			sb.WriteString(fieldType.QualifyRel(pkgPath))
			sb.WriteString("{},")
		}
	}
	sb.WriteByte('\n')
	// close CompositeFields and slice literal
	sb.WriteString(strings.Repeat("\t", indent+1))
	sb.WriteByte('}')
	if indent > 0 {
		sb.WriteByte(',')
	}
}

// EmitResultCompositeAssigns writes all the assign statements for the
// pgtype.CompositeFields representing a Postgres composite type into the output
// struct.
func (tq TemplatedQuery) EmitResultCompositeAssigns(pkgPath string) (string, error) {
	sb := &strings.Builder{}
	for _, out := range tq.Outputs {
		typ, ok := out.Type.(gotype.CompositeType)
		if !ok {
			continue
		}
		indent := "\n\t"
		if tq.ResultKind == ast.ResultKindMany {
			indent += "\t" // a :many query processes items in a for loop
		}
		fields := []string{"item"}
		if len(tq.Outputs) > 1 {
			// Queries with more than 1 output use a struct to group output columns.
			fields = append(fields, typ.Name)
		}
		exprs := []string{"*" + out.LowerName + "Row"}
		assigns := tq.buildResultCompositeAssigns(typ, pkgPath, fields, exprs)
		for _, assign := range assigns {
			sb.WriteString(indent)
			sb.WriteString(strings.Join(assign.fields, "."))
			sb.WriteString(" = ")
			sb.WriteString(strings.Join(assign.exprs, ""))
		}
	}
	return sb.String(), nil
}

type compositeAssign struct {
	// Assign an expression to the path indicated by field names. Joined with ".".
	fields []string
	// Expression to assign to field names. Joined with an empty string.
	exprs []string
}

// buildResultCompositeAssigns recursively creates all assignment statements
// for a composite type using depth first search for nested composite types.
func (tq TemplatedQuery) buildResultCompositeAssigns(
	typ gotype.CompositeType,
	pkgPath string,
	fieldPath []string,
	exprPath []string,
) []compositeAssign {
	assigns := make([]compositeAssign, 0, len(typ.FieldTypes))
	for i, field := range typ.FieldTypes {
		fieldName := typ.FieldNames[i]
		switch child := field.(type) {
		case gotype.CompositeType:
			childAssigns := tq.buildResultCompositeAssigns(
				child,
				pkgPath,
				append(fieldPath, fieldName),
				append(exprPath, "["+strconv.Itoa(i)+"].(pgtype.CompositeFields)"),
			)
			assigns = append(assigns, childAssigns...)
		case gotype.VoidType:
			continue
		default:
			// Copy since fieldPath and exprPath mutate with each iteration.
			fields := make([]string, len(fieldPath), len(fieldPath)+1)
			exprs := make([]string, len(exprPath), len(exprPath)+1)
			copy(fields, fieldPath)
			copy(exprs, exprPath)
			assigns = append(assigns, compositeAssign{
				fields: append(fields, fieldName),
				exprs:  append(exprs, "["+strconv.Itoa(i)+"].(*"+child.QualifyRel(pkgPath)+")"),
			})
		}
	}
	return assigns
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
	return strings.TrimPrefix(result, "[]"), nil
}

// getLongestOutput returns the column of the longest name in all columns and
// the column of the longest type to enable struct alignment.
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
			return ""
		}
		sb := &strings.Builder{}
		sb.WriteString("\n\ntype ")
		sb.WriteString(tq.Name)
		sb.WriteString("Row struct {\n")
		typeCol, structCol := getLongestOutput(outs)
		for _, out := range outs {
			// Name
			sb.WriteString("\t")
			sb.WriteString(out.UpperName)
			// Type
			sb.WriteString(strings.Repeat(" ", typeCol-len(out.UpperName)))
			sb.WriteString(out.QualType)
			// JSON struct tag
			sb.WriteString(strings.Repeat(" ", structCol-len(out.QualType)))
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
