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
	Type string // package-qualified Go type to use generated for this param
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

// TemplateAll creates query template files for each of the codegen.QueryFile.
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
