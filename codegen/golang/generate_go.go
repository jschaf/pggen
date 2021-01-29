package golang

import (
	"fmt"
	"github.com/jschaf/pggen/codegen/gen"
	"github.com/jschaf/pggen/internal/ast"
	"github.com/jschaf/pggen/internal/casing"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// goQueryFile is the Go version of a SQL query file with all information needed
// to execute the codegen template.
type goQueryFile struct {
	GoPkg    string            // the name of the Go package to use for the generated file
	BaseName string            // the source SQL file base name
	Queries  []goTemplateQuery // the queries with all template information
	Imports  []string          // Go imports
	// True if this file is the leader file. The leader defines common interfaces
	// used by by all queries in the same directory.
	IsLeader bool
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

	// Build go specific query files.
	goQueryFiles := make([]goQueryFile, 0, len(queryFiles))
	for _, queryFile := range queryFiles {
		goFile, err := buildGoQueryFile(pkgName, queryFile)
		if err != nil {
			return fmt.Errorf("prepare query file %s for go: %w", queryFile.Src, err)
		}
		goQueryFiles = append(goQueryFiles, goFile)
	}

	// Pick leader file to define common structs and interfaces.
	firstIndex := -1
	firstName := string(unicode.MaxRune)
	for i, goFile := range goQueryFiles {
		if goFile.BaseName < firstName {
			firstIndex = i
			firstName = goFile.BaseName
		}
	}
	goQueryFiles[firstIndex].IsLeader = true

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

	// Emit the files.
	for _, qf := range goQueryFiles {
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
		for i, d := range query.Doc {
			if i > 0 {
				docs.WriteByte('\t') // first line is already indented in the template
			}
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
		GoPkg:    pkgName,
		BaseName: file.Src,
		Queries:  queries,
		Imports:  sortedImports,
	}, nil
}
