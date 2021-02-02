package golang

import (
	"fmt"
	"github.com/jschaf/pggen/internal/ast"
	"github.com/jschaf/pggen/internal/casing"
	"github.com/jschaf/pggen/internal/codegen"
	"sort"
	"strings"
	"unicode"
)

// TemplatedFile is the Go version of a SQL query file with all information
// needed to execute the codegen template.
type TemplatedFile struct {
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
	Name string // name of the param, like 'FirstName' in pggen.arg('FirstName')
	Type string // package-qualified Go type to use for this param
}

type TemplatedColumn struct {
	PgName string // original name of the Postgres column
	Name   string // name in Go-style (UpperCamelCase) to use for the column
	Type   string // Go type to use for the column
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
			dedupeLen++
			declarers[dedupeLen] = declarers[i]
		}
		goQueryFiles[firstIndex].Declarers = declarers[:dedupeLen]
	}

	// Remove unneeded pgconn import if possible.
	for i, goFile := range goQueryFiles {
		if goFile.IsLeader {
			// Leader files define genericConn.Exec which returns pgconn.CommandTag.
			continue
		}
		for _, query := range goFile.Queries {
			if query.ResultKind == ast.ResultKindExec {
				continue // :exec queries return pgconn.CommandTag
			}
		}
		// By here, we don't need pgconn.
		pgconnIdx := -1
		imports := goFile.Imports
		for i, pkg := range imports {
			if pkg == "github.com/jackc/pgconn" {
				pgconnIdx = i
				break
			}
		}
		copy(imports[pgconnIdx:], imports[pgconnIdx+1:])
		goQueryFiles[i].Imports = imports[:len(imports)-1]
	}

	return goQueryFiles, nil
}

// templateFile creates the data needed to build a Go file for a query file.
// Also returns any declarations needed by this query file. The caller must
// dedupe declarations.
func (tm Templater) templateFile(file codegen.QueryFile) (TemplatedFile, []Declarer, error) {
	imports := map[string]struct{}{
		"context":                 {},
		"fmt":                     {},
		"github.com/jackc/pgconn": {},
		"github.com/jackc/pgx/v4": {},
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
			goType, err := tm.resolver.Resolve(input.PgType /*nullable*/, false)
			if err != nil {
				return TemplatedFile{}, nil, err
			}
			imports[goType.Pkg] = struct{}{}
			inputs[i] = TemplatedParam{
				Name: tm.caser.ToUpperCamel(input.PgName),
				Type: goType.Name,
			}
			if goType.Decl != nil {
				declarers = append(declarers, goType.Decl)
			}
		}

		// Build outputs.
		outputs := make([]TemplatedColumn, len(query.Outputs))
		for i, out := range query.Outputs {
			goType, err := tm.resolver.Resolve(out.PgType, out.Nullable)
			if err != nil {
				return TemplatedFile{}, nil, err
			}
			imports[goType.Pkg] = struct{}{}
			outputs[i] = TemplatedColumn{
				PgName: out.PgName,
				Name:   tm.caser.ToUpperCamel(out.PgName),
				Type:   goType.Name,
			}
			if goType.Decl != nil {
				declarers = append(declarers, goType.Decl)
			}
		}

		queries = append(queries, TemplatedQuery{
			Name:        query.Name,
			SQLVarName:  lowercaseFirstLetter(query.Name) + "SQL",
			ResultKind:  query.ResultKind,
			Doc:         docs.String(),
			PreparedSQL: query.PreparedSQL,
			Inputs:      inputs,
			Outputs:     outputs,
		})
	}

	// Build imports.
	sortedImports := make([]string, 0, len(imports))
	for pkg := range imports {
		if pkg != "" {
			sortedImports = append(sortedImports, pkg)
		}
	}
	sort.Strings(sortedImports)

	return TemplatedFile{
		GoPkg:   tm.pkg,
		Path:    file.Path,
		Queries: queries,
		Imports: sortedImports,
	}, declarers, nil
}

// isLast returns true if index is the last index in item.
func lowercaseFirstLetter(s string) string {
	if s == "" {
		return ""
	}
	first, rest := s[0], s[1:]
	return strings.ToLower(string(first)) + rest
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
			sb.WriteString(lowercaseFirstLetter(input.Name))
			sb.WriteRune(' ')
			sb.WriteString(input.Type)
		}
		return sb.String()
	default:
		return ", params " + tq.Name + "Params"
	}
}

// EmitPreparedSQL emits the prepared SQL query with appropriate quoting.
func (tq TemplatedQuery) EmitPreparedSQL() string {
	hasBacktick := strings.ContainsRune(tq.PreparedSQL, '`')
	if !hasBacktick {
		return "`" + tq.PreparedSQL + "`"
	}
	hasDoubleQuote := strings.ContainsRune(tq.PreparedSQL, '"')
	hasNewline := strings.ContainsAny(tq.PreparedSQL, "\r\n")
	if !hasDoubleQuote && !hasNewline {
		hasBackslash := strings.ContainsRune(tq.PreparedSQL, '\\')
		sql := tq.PreparedSQL
		if hasBackslash {
			sql = strings.ReplaceAll(sql, `\`, `\\`)
		}
		return `"` + sql + `"`
	}
	// The SQL query contains both '`' and '"'.
	// We can't use unicode escapes like U&'d\0061t\+000061' because the backtick
	// can appear in either a double-quoted identifier like "abc`" or a string
	// literal. Similarly, a double quote either delimits an identifier or can
	// appear in a string literal. We'll break up the string using Go string
	// concatenation using both types of Go string literals. Meaning, convert:
	//     sql := `SELECT '`"'`
	// Into:
	//     sql := `SELECT '` + "`" + `"'`
	return "`" + strings.ReplaceAll(tq.PreparedSQL, "`", "` + \"`\" + `") + "`"
}

func getLongestInput(inputs []TemplatedParam) int {
	max := 0
	for _, in := range inputs {
		if len(in.Name) > max {
			max = len(in.Name)
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
		sb.WriteString(out.Name)
		sb.WriteString(strings.Repeat(" ", typeCol-len(out.Name)))
		sb.WriteString(out.Type)
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
			sb.WriteString(lowercaseFirstLetter(input.Name))
		}
		return sb.String()
	default:
		sb := strings.Builder{}
		for _, input := range tq.Inputs {
			sb.WriteString(", params.")
			sb.WriteString(input.Name)
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
		switch len(tq.Outputs) {
		case 0:
			return "", nil
		case 1:
			// If there's only 1 output column, we return it directly, without
			// wrapping in a struct.
			return "&item", nil
		default:
			sb := strings.Builder{}
			sb.Grow(15 * len(tq.Outputs))
			for i, out := range tq.Outputs {
				sb.WriteString("&item.")
				sb.WriteString(out.Name)
				if i < len(tq.Outputs)-1 {
					sb.WriteString(", ")
				}
			}
			return sb.String(), nil
		}
	default:
		return "", fmt.Errorf("unhandled EmitRowScanArgs type: %s", tq.ResultKind)
	}
}

// EmitResultType returns the string representing the overall query result type,
// meaning the return result.
func (tq TemplatedQuery) EmitResultType() (string, error) {
	switch tq.ResultKind {
	case ast.ResultKindExec:
		return "pgconn.CommandTag", nil
	case ast.ResultKindMany:
		switch len(tq.Outputs) {
		case 0:
			return "pgconn.CommandTag", nil
		case 1:
			return "[]" + tq.Outputs[0].Type, nil
		default:
			return "[]" + tq.Name + "Row", nil
		}
	case ast.ResultKindOne:
		switch len(tq.Outputs) {
		case 0:
			return "pgconn.CommandTag", nil
		case 1:
			return tq.Outputs[0].Type, nil
		default:
			return tq.Name + "Row", nil
		}
	default:
		return "", fmt.Errorf("unhandled EmitResultType kind: %s", tq.ResultKind)
	}
}

// EmitResultTypeInit returns the initialization code for the result type with
// name, typically "item" or "items". For array type, we take care to not use a
// var declaration so that JSON serialization returns an empty array instead of
// null.
func (tq TemplatedQuery) EmitResultTypeInit(name string) (string, error) {
	result, err := tq.EmitResultType()
	if err != nil {
		return "", fmt.Errorf("create result type for EmitResultTypeInit: %w", err)
	}
	if strings.HasPrefix(result, "[]") {
		switch tq.ResultKind {
		case ast.ResultKindMany:
			return name + " := " + result + "{}", nil
		case ast.ResultKindOne:
			return name + " := " + result + "{}", nil
		default:
			return "", fmt.Errorf("unhandled EmitResultTypeInit type %s for kind %s", result, tq.ResultKind)
		}
	}
	switch tq.ResultKind {
	case ast.ResultKindMany, ast.ResultKindOne:
		return "var " + name + " " + result, nil
	default:
		return "", fmt.Errorf("unhandled EmitResultTypeInit type %s for kind %s", result, tq.ResultKind)
	}
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
		if len(out.Name) > nameLen {
			nameLen = len(out.Name)
		}
	}
	nameLen++ // 1 space to separate name from type

	typeLen := 0
	for _, out := range outs {
		if len(out.Type) > typeLen {
			typeLen = len(out.Type)
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
		if len(tq.Outputs) <= 1 {
			return ""
		}
		sb := &strings.Builder{}
		sb.WriteString("\n\ntype ")
		sb.WriteString(tq.Name)
		sb.WriteString("Row struct {\n")
		typeCol, structCol := getLongestOutput(tq.Outputs)
		for _, out := range tq.Outputs {
			// Name
			sb.WriteString("\t")
			sb.WriteString(out.Name)
			// Type
			sb.WriteString(strings.Repeat(" ", typeCol-len(out.Name)))
			sb.WriteString(out.Type)
			// JSON struct tag
			sb.WriteString(strings.Repeat(" ", structCol-len(out.Type)))
			sb.WriteString("`json:\"")
			sb.WriteString(out.PgName)
			sb.WriteString("\"`")
			sb.WriteRune('\n')
		}
		sb.WriteString("}")
		return sb.String()
	default:
		panic("unhandled result type: " + tq.ResultKind)
	}
}
