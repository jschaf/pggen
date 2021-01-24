package codegen

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v4"
	"github.com/jschaf/sqld/internal/ast"
	"github.com/jschaf/sqld/internal/parser"
	"github.com/jschaf/sqld/internal/pginfer"
	_ "github.com/jschaf/sqld/statik"
	gotok "go/token"
	"path/filepath"
)

// GenerateOptions are the unparsed options that controls the generated Go code.
type GenerateOptions struct {
	// The connection string to the running Postgres database to use to get type
	// information for each query in QueryFiles.
	//
	// Must be parseable by pgconn.ParseConfig, like:
	//
	//   # Example DSN
	//   user=jack password=secret host=pg.example.com port=5432 dbname=foo_db sslmode=verify-ca
	//
	//   # Example URL
	//   postgres://jack:secret@pg.example.com:5432/foo_db?sslmode=verify-ca
	ConnString string
	// Generate code for each of the SQL query file paths.
	QueryFiles []string
	// The overall config after merging config files and flag options.
	Config Config
	// The name of the Go package for the file. If empty, defaults to the
	// directory name.
	GoPackage string
	// Directory to write generated files. Writes one file for each query file.
	// If more than one query file, also writes querier.go.
	OutputDir string
}

// Generate generates Go code to safely wrap each SQL ast.SourceQuery in
// opts.QueryFiles into a callable methods.
//
// Generate must only be called once per output directory.
func Generate(opts GenerateOptions) error {
	pgConnConfig, err := pgx.ParseConfig(opts.ConnString)
	if err != nil {
		return fmt.Errorf("parse postgres conn string: %w", err)
	}

	pgConn, err := pgx.ConnectConfig(context.TODO(), pgConnConfig)
	if err != nil {
		return fmt.Errorf("connect to pggen postgres database: %w", err)
	}
	inferrer := pginfer.NewInferrer(pgConn)

	queriesFiles, err := parseQueryFiles(opts, opts.QueryFiles, inferrer)
	if err != nil {
		return err
	}

	if err := emitAll(opts.OutputDir, queriesFiles); err != nil {
		return fmt.Errorf("emit generated code: %w", err)
	}

	return nil
}

// templateQuery is a query ready for rendering to a Go template.
type templateQuery struct {
	// Name of the query, from the comment preceding the query. Like 'FindAuthors'
	// in the source SQL: "-- name: FindAuthors :many"
	Name string
	// Each line of documentation from the source query file, formatted for Go.
	Docs []string
	// The SQL query, with pggen functions replaced with Postgres syntax. Ready
	// to run on Postgres with the PREPARE statement.
	PreparedSQL    string
	ResultTypeName string // name of the result type
	ParamTypeName  string // name of the input parameter struct
	// The input parameters to the query.
	Inputs []pginfer.InputParam
	// The output columns of the query.
	Outputs []pginfer.OutputColumn
}

// queryFile represents all of the SQL queries from a single file.
type queryFile struct {
	GoPkg   string          // the name of the Go package to use for the generated file
	Src     string          // the source SQL file base name
	Queries []templateQuery // the queries with all template information
}

func parseQueryFiles(opts GenerateOptions, queryFiles []string, inferrer *pginfer.Inferrer) ([]queryFile, error) {
	files := make([]queryFile, len(queryFiles))
	pkgName := opts.GoPackage
	if opts.GoPackage == "" {
		pkgName = filepath.Base(opts.OutputDir)
	}
	for i, file := range queryFiles {
		queryFile, err := parseQueries(pkgName, file, inferrer)
		if err != nil {
			return nil, fmt.Errorf("parse template query file %q: %w", file, err)
		}
		files[i] = queryFile
	}
	return files, nil
}

func parseQueries(pkgName string, file string, inferrer *pginfer.Inferrer) (queryFile, error) {
	astFile, err := parser.ParseFile(gotok.NewFileSet(), file, nil, 0)
	if err != nil {
		return queryFile{}, fmt.Errorf("parse query file %q: %w", file, err)
	}
	queries := make([]templateQuery, 0, len(astFile.Queries))
	for _, query := range astFile.Queries {
		switch query := query.(type) {
		case *ast.BadQuery:
			return queryFile{}, errors.New("parsed bad query instead of erroring")
		case *ast.SourceQuery:
			break // break switch - keep going
		default:
			return queryFile{}, fmt.Errorf("unhandled query ast type: %T", query)
		}

		srcQuery := query.(*ast.SourceQuery)
		typedQuery, err := inferrer.InferTypes(srcQuery)
		if err != nil {
			return queryFile{}, fmt.Errorf("infer typed named query %s: %w", srcQuery.Name, err)
		}

		tmplQuery := templateQuery{
			Name:           typedQuery.Name,
			Docs:           nil, // TODO: pipe ast.CommentGroup through here
			PreparedSQL:    typedQuery.PreparedSQL,
			ResultTypeName: "FOO_TODO",
			ParamTypeName:  "BAR_TODO",
			Inputs:         typedQuery.Inputs,
			Outputs:        typedQuery.Outputs,
		}

		queries = append(queries, tmplQuery)
	}
	return queryFile{
		Src:     file,
		GoPkg:   pkgName,
		Queries: queries,
	}, nil
}
