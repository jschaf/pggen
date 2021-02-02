package golang

import (
	"fmt"
	"github.com/jschaf/pggen/internal/ast"
	"github.com/jschaf/pggen/internal/casing"
	"github.com/jschaf/pggen/internal/codegen"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// GenerateOptions are options to control generated Go output.
type GenerateOptions struct {
	GoPkg     string
	OutputDir string
	// A map of lowercase acronyms to the upper case equivalent, like:
	// "api" => "API". ID is included by default.
	Acronyms map[string]string
}

// goQueryFile is the Go version of a SQL query file with all information needed
// to execute the codegen template.
type goQueryFile struct {
	GoPkg   string            // the name of the Go package to use for the generated file
	Path    string            // the path to source SQL file
	Queries []goTemplateQuery // the queries with all template information
	Imports []string          // Go imports
	// True if this file is the leader file. The leader defines common code used
	// by by all queries in the same directory. Only one leader per directory.
	IsLeader bool
	// Any declarations this file should declare. Only set on leader.
	Declarers []Declarer
}

// goTemplateQuery is a query with all information required to execute the
// codegen template.
type goTemplateQuery struct {
	Name        string           // name of the query, from the comment preceding the query
	SQLVarName  string           // name of the string variable containing the SQL
	ResultKind  ast.ResultKind   // kind of result: :one, :many, or :exec
	Doc         string           // doc from the source query file, formatted for Go
	PreparedSQL string           // SQL query, ready to run with PREPARE statement
	Inputs      []goInputParam   // input parameters to the query
	Outputs     []goOutputColumn // output columns of the query
}

type goInputParam struct {
	Name string // name of the param, like 'FirstName' in pggen.arg('FirstName')
	Type string // package-qualified Go type to use generated for this param
}

type goOutputColumn struct {
	PgName string // original name of the Postgres column
	Name   string // name in Go-style (UpperCamelCase) to use for the column
	Type   string // Go type to use for the column
}

// Generate emits generated Go files for each of the queryFiles.
func Generate(opts GenerateOptions, queryFiles []codegen.QueryFile) error {
	tmpl, err := parseQueryTemplate()
	if err != nil {
		return fmt.Errorf("parse generated Go code template: %w", err)
	}
	pkgName := opts.GoPkg
	if pkgName == "" {
		pkgName = filepath.Base(opts.OutputDir)
	}
	caser := casing.NewCaser()
	caser.AddAcronym("id", "ID")
	caser.AddAcronyms(opts.Acronyms)
	typeResolver := NewTypeResolver(caser)

	// Build go specific query files.
	goQueryFiles := make([]goQueryFile, 0, len(queryFiles))
	declarers := make([]Declarer, 0, 8)
	for _, queryFile := range queryFiles {
		goFile, decls, err := buildGoQueryFile(pkgName, caser, queryFile, typeResolver)
		if err != nil {
			return fmt.Errorf("prepare query file %s for go: %w", queryFile.Path, err)
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

	// Emit the files.
	emitter := NewEmitter(opts.OutputDir, tmpl)
	for _, qf := range goQueryFiles {
		if err := emitter.EmitQueryFile(qf); err != nil {
			return fmt.Errorf("emit generated Go code: %w", err)
		}
	}
	return nil
}

// buildGoQueryFile creates the data needed to build a Go file for a query file.
// Also returns any declarations needed by this query file. The caller must
// dedupe declarations.
func buildGoQueryFile(pkgName string, caser casing.Caser, file codegen.QueryFile, typeResolver TypeResolver) (goQueryFile, []Declarer, error) {
	imports := map[string]struct{}{
		"context":                 {},
		"fmt":                     {},
		"github.com/jackc/pgconn": {},
		"github.com/jackc/pgx/v4": {},
	}

	queries := make([]goTemplateQuery, 0, len(file.Queries))
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
		inputs := make([]goInputParam, len(query.Inputs))
		for i, input := range query.Inputs {
			goType, err := typeResolver.Resolve(input.PgType /*nullable*/, false)
			if err != nil {
				return goQueryFile{}, nil, err
			}
			imports[goType.Pkg] = struct{}{}
			inputs[i] = goInputParam{
				Name: caser.ToUpperCamel(input.PgName),
				Type: goType.Name,
			}
			if goType.Decl != nil {
				declarers = append(declarers, goType.Decl)
			}
		}

		// Build outputs.
		outputs := make([]goOutputColumn, len(query.Outputs))
		for i, out := range query.Outputs {
			goType, err := typeResolver.Resolve(out.PgType, out.Nullable)
			if err != nil {
				return goQueryFile{}, nil, err
			}
			imports[goType.Pkg] = struct{}{}
			outputs[i] = goOutputColumn{
				PgName: out.PgName,
				Name:   caser.ToUpperCamel(out.PgName),
				Type:   goType.Name,
			}
			if goType.Decl != nil {
				declarers = append(declarers, goType.Decl)
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
		Path:    file.Path,
		Queries: queries,
		Imports: sortedImports,
	}, declarers, nil
}
