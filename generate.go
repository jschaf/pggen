package pggen

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v4"
	"github.com/jschaf/pggen/internal/ast"
	"github.com/jschaf/pggen/internal/codegen"
	"github.com/jschaf/pggen/internal/codegen/golang"
	"github.com/jschaf/pggen/internal/errs"
	"github.com/jschaf/pggen/internal/parser"
	"github.com/jschaf/pggen/internal/pgdocker"
	"github.com/jschaf/pggen/internal/pginfer"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	gotok "go/token"
	"io/ioutil"
	"path/filepath"
	"time"
)

// Lang is a supported codegen language.
type Lang string

const (
	LangGo Lang = "go"
)

// GenerateOptions are the unparsed options that controls the generated Go code.
type GenerateOptions struct {
	// What language to generate code in.
	Language Lang
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
	// Schema files to run on Postgres init. Can be *.sql, *.sql.gz, or executable
	// *.sh files .
	SchemaFiles []string
	// The name of the Go package for the file. If empty, defaults to the
	// directory name.
	GoPackage string
	// Directory to write generated files. Writes one file for each query file.
	// If more than one query file, also writes querier.go.
	OutputDir string
	// A map of lowercase acronyms to the upper case equivalent, like:
	// "api" => "API", or "apis" => "APIs".
	Acronyms map[string]string
	// A map from a Postgres type name to a fully qualified Go type.
	TypeOverrides map[string]string
	// What log level to log at.
	LogLevel zapcore.Level
}

// Generate generates language specific code to safely wrap each SQL
// ast.SourceQuery in opts.QueryFiles.
//
// Generate must only be called once per output directory.
func Generate(opts GenerateOptions) (mErr error) {
	// Preconditions.
	if opts.Language == "" {
		return fmt.Errorf("generate language must be set; got empty string")
	}
	if len(opts.QueryFiles) == 0 {
		return fmt.Errorf("got 0 query files, at least 1 must be set")
	}
	if opts.OutputDir == "" {
		return fmt.Errorf("output dir must be set")
	}

	// Logger.
	logCfg := zap.NewDevelopmentConfig()
	logCfg.Level = zap.NewAtomicLevelAt(opts.LogLevel)
	logger, err := logCfg.Build()
	if err != nil {
		return fmt.Errorf("create zap logger: %w", err)
	}
	defer logger.Sync() // nolint
	l := logger.Sugar()

	// Postgres connection.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	pgConn, errEnricher, cleanup, err := connectPostgres(ctx, opts, l)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer errs.Capture(&mErr, cleanup, "close postgres connection")

	// Parse queries.
	inferrer := pginfer.NewInferrer(pgConn)
	queryFiles, err := parseQueryFiles(opts.QueryFiles, inferrer)
	if err != nil {
		return errEnricher(err)
	}

	// Codegen.
	if opts.Acronyms == nil {
		opts.Acronyms = make(map[string]string, 1)
	}
	opts.Acronyms["id"] = "ID"
	switch opts.Language {
	case LangGo:
		goOpts := golang.GenerateOptions{
			GoPkg:         opts.GoPackage,
			OutputDir:     opts.OutputDir,
			Acronyms:      opts.Acronyms,
			TypeOverrides: opts.TypeOverrides,
		}
		if err := golang.Generate(goOpts, queryFiles); err != nil {
			return fmt.Errorf("generate go code: %w", err)
		}
	default:
		return fmt.Errorf("unsupported output language %q", opts.Language)
	}
	return nil
}

// connectPostgres connects to postgres using connString if given or by
// running a Docker postgres container and connecting to that.
func connectPostgres(
	ctx context.Context,
	opts GenerateOptions,
	l *zap.SugaredLogger,
) (*pgx.Conn, func(error) error, func() error, error) {
	// Create connection by starting dockerized Postgres.
	if opts.ConnString == "" {
		client, err := pgdocker.Start(ctx, opts.SchemaFiles, l)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("start dockerized postgres: %w", err)
		}
		stopDocker := func() error { return client.Stop(ctx) }
		connStr, err := client.ConnString()
		if err != nil {
			return nil, nil, nil, fmt.Errorf("get dockerized postgres conn string: %w", err)
		}
		pgConn, err := pgx.Connect(ctx, connStr)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("connect to pggen dockerized postgres database: %w", err)
		}
		errEnricher := func(e error) error {
			if e == nil {
				return e
			}
			logs, err := client.GetContainerLogs()
			if err != nil {
				return multierr.Append(e, err)
			}
			return fmt.Errorf("Container logs for Postgres container:\n\n%s\n\n%w", logs, e)
		}
		return pgConn, errEnricher, stopDocker, nil
	}
	// Use existing Postgres.
	nopCleanup := func() error { return nil }
	nopErrEnricher := func(e error) error { return e }
	pgConn, err := pgx.Connect(ctx, opts.ConnString)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("connect to pggen postgres database: %w", err)
	}
	// Run SQL init scripts. pgdocker runs these in the other case by copying
	// the files into the entrypoint folder. Emulate the behavior for a subset of
	// supported files.
	for _, script := range opts.SchemaFiles {
		if filepath.Ext(script) != ".sql" {
			return nil, nopErrEnricher, nopCleanup, fmt.Errorf("cannot run non-sql schema file on Postgres "+
				"(*.sh and *.sql.gz files only supported without --postgres-connection): %s", script)
		}
		bs, err := ioutil.ReadFile(script)
		if err != nil {
			return nil, nil, nopCleanup, fmt.Errorf("read schema file: %w", err)
		}
		if _, err := pgConn.Exec(ctx, string(bs)); err != nil {
			return nil, nopErrEnricher, nopCleanup, fmt.Errorf("load schema file into Postgres: %w", err)
		}
	}
	return pgConn, nopErrEnricher, nopCleanup, nil
}

func parseQueryFiles(queryFiles []string, inferrer *pginfer.Inferrer) ([]codegen.QueryFile, error) {
	files := make([]codegen.QueryFile, len(queryFiles))
	for i, file := range queryFiles {
		srcPath, err := filepath.Abs(file)
		if err != nil {
			return nil, fmt.Errorf("resovle absolute path for %q: %w", file, err)
		}
		queryFile, err := parseQueries(srcPath, inferrer)
		if err != nil {
			return nil, fmt.Errorf("parse template query file %q: %w", file, err)
		}
		files[i] = queryFile
	}
	return files, nil
}

func parseQueries(srcPath string, inferrer *pginfer.Inferrer) (codegen.QueryFile, error) {
	astFile, err := parser.ParseFile(gotok.NewFileSet(), srcPath, nil, 0)
	if err != nil {
		return codegen.QueryFile{}, fmt.Errorf("parse query file %q: %w", srcPath, err)
	}

	// Check for duplicate query names and bad queries.
	srcQueries := make([]*ast.SourceQuery, 0, len(astFile.Queries))
	seenNames := make(map[string]struct{}, len(astFile.Queries))
	for _, query := range astFile.Queries {
		switch query := query.(type) {
		case *ast.BadQuery:
			return codegen.QueryFile{}, errors.New("parsed bad query instead of erroring")
		case *ast.SourceQuery:
			if _, ok := seenNames[query.Name]; ok {
				return codegen.QueryFile{}, fmt.Errorf("duplicate query name %s", query.Name)
			}
			seenNames[query.Name] = struct{}{}
			srcQueries = append(srcQueries, query)
		default:
			return codegen.QueryFile{}, fmt.Errorf("unhandled query ast type: %T", query)
		}
	}

	// Infer types.
	queries := make([]pginfer.TypedQuery, 0, len(astFile.Queries))
	for _, srcQuery := range srcQueries {
		typedQuery, err := inferrer.InferTypes(srcQuery)
		if err != nil {
			return codegen.QueryFile{}, fmt.Errorf("infer typed named query %s: %w", srcQuery.Name, err)
		}
		queries = append(queries, typedQuery)
	}
	return codegen.QueryFile{
		SourcePath: srcPath,
		Queries:    queries,
	}, nil
}
