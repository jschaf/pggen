package golang

import (
	"fmt"
	"github.com/jschaf/pggen/codegen/gen"
	"github.com/jschaf/pggen/internal/ast"
	"github.com/jschaf/pggen/internal/casing"
	"path/filepath"
	"sort"
	"strings"
)

// goQueryFile is the Go version of a SQL query file with all information needed
// to execute the codegen template.
type goQueryFile struct {
	GoPkg   string            // the name of the Go package to use for the generated file
	Src     string            // the source SQL file base name
	Queries []goTemplateQuery // the queries with all template information
	Imports []string          // Go imports
}

// goTemplateQuery is a query with all information required to execute the
// codegen template.
type goTemplateQuery struct {
	Name        string           // name of the query, from the comment preceding the query
	SQLVarName  string           // name of the string variable containing the SQL
	ResultKind  ast.ResultKind   // kind of result. :one, :many, or :exec
	Doc         string           // doc from the source query file, formatted for Go
	PreparedSQL string           // SQL query, ready to run with PREPARE statement
	Inputs      []goInputParam   // input parameters to the query
	Outputs     []goOutputColumn // output columns of the query
}

type goInputParam struct {
	Name string // name of the param, like 'FirstName' in pggen.arg('FirstName')
	Type string // Go type to use generated for this param
}

type goOutputColumn struct {
	Name string // name in Go-style to use for the column
	Type string // Go type to use for the column
}

// Generate emits generated Go files for each of the queryFiles.
func Generate(opts gen.GenerateOptions, queryFiles []gen.QueryFile) error {
	tmpl, err := parseQueryTemplate()
	if err != nil {
		return fmt.Errorf("parse generated Go code template: %w", err)
	}
	pkgName := opts.GoPackage
	if opts.GoPackage == "" {
		pkgName = filepath.Base(opts.OutputDir)
	}
	for _, queryFile := range queryFiles {
		qf, err := buildGoQueryFile(pkgName, queryFile)
		if err != nil {
			return fmt.Errorf("prepare query file %s for go: %w", queryFile.Src, err)
		}
		if err := emitQueryFile(opts.OutputDir, qf, tmpl); err != nil {
			return fmt.Errorf("emit generated Go code: %w", err)
		}
	}
	return nil
}

func buildGoQueryFile(pkgName string, file gen.QueryFile) (goQueryFile, error) {
	caser := casing.NewCaser()
	caser.AddAcronym("id", "ID")

	imports := map[string]struct{}{
		"context":                 {},
		"fmt":                     {},
		"github.com/jackc/pgconn": {},
		"github.com/jackc/pgx/v4": {},
	}

	queries := make([]goTemplateQuery, 0, len(file.Queries))
	for _, query := range file.Queries {
		// Build doc string.
		docs := strings.Builder{}
		avgCharsPerLine := 40
		docs.Grow(len(query.Doc) * avgCharsPerLine)
		for _, d := range query.Doc {
			docs.WriteString("// ")
			docs.WriteString(d)
			docs.WriteRune('\n')
		}

		// Build inputs.
		inputs := make([]goInputParam, len(query.Inputs))
		for i, input := range query.Inputs {
			pkg, goType, err := pgToGoType(input.PgType, false)
			if err != nil {
				return goQueryFile{}, err
			}
			imports[pkg] = struct{}{}
			inputs[i] = goInputParam{
				Name: caser.ToUpperCamel(input.PgName),
				Type: goType,
			}
		}

		// Build outputs.
		outputs := make([]goOutputColumn, len(query.Outputs))
		for i, out := range query.Outputs {
			pkg, goType, err := pgToGoType(out.PgType, out.Nullable)
			if err != nil {
				return goQueryFile{}, err
			}
			imports[pkg] = struct{}{}
			outputs[i] = goOutputColumn{
				Name: caser.ToUpperCamel(out.PgName),
				Type: goType,
			}
		}

		queries = append(queries, goTemplateQuery{
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

	return goQueryFile{
		GoPkg:   pkgName,
		Src:     file.Src,
		Queries: queries,
		Imports: sortedImports,
	}, nil
}
