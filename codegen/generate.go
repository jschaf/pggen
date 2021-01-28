package codegen

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v4"
	"github.com/jschaf/pggen/codegen/gen"
	"github.com/jschaf/pggen/codegen/golang"
	"github.com/jschaf/pggen/internal/ast"
	"github.com/jschaf/pggen/internal/errs"
	"github.com/jschaf/pggen/internal/parser"
	"github.com/jschaf/pggen/internal/pgdocker"
	"github.com/jschaf/pggen/internal/pginfer"
	_ "github.com/jschaf/pggen/statik" // bundled template files
	"go.uber.org/zap"
	gotok "go/token"
	"time"
)

// Generate generates language specific code to safely wrap each SQL
// ast.SourceQuery in opts.QueryFiles.
//
// Generate must only be called once per output directory.
func Generate(opts gen.GenerateOptions) (mErr error) {
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
	if opts.ConnString != "" && len(opts.DockerInitScripts) > 0 {
		return fmt.Errorf("cannot use both DockerInitScripts and ConnString together")
	}

	// Logger.
	logCfg := zap.NewDevelopmentConfig()
	// TODO: control by log-level flag
	logCfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	logger, err := logCfg.Build()
	if err != nil {
		return fmt.Errorf("create zap logger: %w", err)
	}
	defer logger.Sync()
	l := logger.Sugar()

	// Postgres connection.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	pgConn, cleanup, err := connectPostgres(ctx, opts, l)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer errs.Capture(&mErr, cleanup, "close postgres connection")

	// Parse queries.
	inferrer := pginfer.NewInferrer(pgConn)
	queryFiles, err := parseQueryFiles(opts.QueryFiles, inferrer)
	if err != nil {
		return err
	}

	// Codegen.
	switch opts.Language {
	case gen.LangGo:
		if err := golang.Generate(opts, queryFiles); err != nil {
			return fmt.Errorf("generate go code: %w", err)
		}
	default:
		return fmt.Errorf("unsupported output language %q", opts.Language)
	}
	return nil
}

// connectPostgres connects to postgres using connString if given, or by
// running a Docker postgres container and connecting to that.
func connectPostgres(ctx context.Context, opts gen.GenerateOptions, l *zap.SugaredLogger) (conn *pgx.Conn, cleanup func() error, mErr error) {
	cleanup = func() error { return nil }
	connString := opts.ConnString
	if connString == "" {
		// Create connection by starting dockerized Postgres.
		client, err := pgdocker.Start(ctx, opts.DockerInitScripts, l)
		if err != nil {
			return nil, nil, fmt.Errorf("start dockerized postgres: %w", err)
		}
		cleanup = func() error { return client.Stop(ctx) }
		conn, err := client.ConnString()
		if err != nil {
			return nil, nil, fmt.Errorf("get dockerized postgres conn string: %w", err)
		}
		connString = conn
	}
	pgConnConfig, err := pgx.ParseConfig(connString)
	if err != nil {
		return nil, nil, fmt.Errorf("parse postgres conn string: %w", err)
	}
	pgConn, err := pgx.ConnectConfig(context.TODO(), pgConnConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("connect to pggen postgres database: %w", err)
	}
	return pgConn, cleanup, nil
}

func parseQueryFiles(queryFiles []string, inferrer *pginfer.Inferrer) ([]gen.QueryFile, error) {
	files := make([]gen.QueryFile, len(queryFiles))
	for i, file := range queryFiles {
		queryFile, err := parseQueries(file, inferrer)
		if err != nil {
			return nil, fmt.Errorf("parse template query file %q: %w", file, err)
		}
		files[i] = queryFile
	}
	return files, nil
}

func parseQueries(file string, inferrer *pginfer.Inferrer) (gen.QueryFile, error) {
	astFile, err := parser.ParseFile(gotok.NewFileSet(), file, nil, 0)
	if err != nil {
		return gen.QueryFile{}, fmt.Errorf("parse query file %q: %w", file, err)
	}
	queries := make([]pginfer.TypedQuery, 0, len(astFile.Queries))
	for _, query := range astFile.Queries {
		switch query := query.(type) {
		case *ast.BadQuery:
			return gen.QueryFile{}, errors.New("parsed bad query instead of erroring")
		case *ast.SourceQuery:
			break // break switch - keep going
		default:
			return gen.QueryFile{}, fmt.Errorf("unhandled query ast type: %T", query)
		}

		// Infer types.
		srcQuery := query.(*ast.SourceQuery)
		typedQuery, err := inferrer.InferTypes(srcQuery)
		if err != nil {
			return gen.QueryFile{}, fmt.Errorf("infer typed named query %s: %w", srcQuery.Name, err)
		}

		queries = append(queries, typedQuery)
	}
	return gen.QueryFile{
		Src:     file,
		Queries: queries,
	}, nil
}
