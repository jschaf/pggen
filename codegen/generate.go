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
	// Directory to write generated files. Writes one file for each query file as
	// well as querier.go.
	OutputDir string
}

// Generate generates Go code to safely wrap each SQL SourceQuery in
// opts.QueryFiles into a callable methods.
//
// Generate must only be called once per output directory.
func Generate(opts GenerateOptions) error {
	pgConnConfig, err := pgx.ParseConfig(opts.ConnString)
	if err != nil {
		return fmt.Errorf("parse postgres conn string: %w", err)
	}

	queriesFiles, err := parseQueryFiles(opts, opts.QueryFiles)
	if err != nil {
		return fmt.Errorf("parse query files: %w", err)
	}

	pgConn, err := pgx.ConnectConfig(context.TODO(), pgConnConfig)
	if err != nil {
		return fmt.Errorf("connect to pggen postgres database: %w", err)
	}
	inferrer := pginfer.NewInferrer(pgConn)
	for i, file := range queriesFiles {
		typedFile, err := inferTypedQueries(inferrer, file)
		if err != nil {
			return fmt.Errorf("infer typed query: %w", err)
		}
		queriesFiles[i] = typedFile
	}

	if err := emitAll(opts.OutputDir, queriesFiles); err != nil {
		return fmt.Errorf("emit generated code: %w", err)
	}

	return nil
}

// queryFile represents all of the SQL queries from a single file.
type queryFile struct {
	GoPkg         string               // the name of the Go package for the file
	Src           string               // the source file
	SourceQueries []*ast.SourceQuery   // the queries as they appeared in the source file
	TypedQueries  []pginfer.TypedQuery // the queries after inferring type information
}

func parseQueryFiles(opts GenerateOptions, queryFiles []string) ([]queryFile, error) {
	files := make([]queryFile, len(queryFiles))
	pkgName := opts.GoPackage
	if opts.GoPackage == "" {
		pkgName = filepath.Base(opts.OutputDir)
	}
	for i, file := range queryFiles {
		queryFile, err := parseSourceQueries(pkgName, file)
		if err != nil {
			return nil, fmt.Errorf("parse template query file %q: %w", file, err)
		}
		files[i] = queryFile
	}
	return files, nil
}

func parseSourceQueries(pkgName string, file string) (queryFile, error) {
	astFile, err := parser.ParseFile(gotok.NewFileSet(), file, nil, 0)
	if err != nil {
		return queryFile{}, fmt.Errorf("parse query file %q: %w", file, err)
	}
	srcQueries := make([]*ast.SourceQuery, 0, len(astFile.Queries))
	for _, query := range astFile.Queries {
		switch q := query.(type) {
		case *ast.BadQuery:
			return queryFile{}, errors.New("parsed bad query instead of erroring")
		case *ast.SourceQuery:
			srcQueries = append(srcQueries, q)
		default:
			return queryFile{}, fmt.Errorf("unhandled query ast type: %T", q)
		}
	}
	return queryFile{
		Src:           file,
		GoPkg:         pkgName,
		SourceQueries: srcQueries,
		TypedQueries:  nil,
	}, nil
}

func inferTypedQueries(inf *pginfer.Inferrer, file queryFile) (queryFile, error) {
	for _, query := range file.SourceQueries {
		typedQuery, err := inf.InferTypes(query)
		if err != nil {
			return queryFile{}, fmt.Errorf("infer typed named query %q: %w", query.Name, err)
		}
		file.TypedQueries = append(file.TypedQueries, typedQuery)
	}
	return file, nil
}
