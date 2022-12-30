package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar"
	"github.com/jschaf/pggen"
	"github.com/jschaf/pggen/internal/flags"
	"github.com/jschaf/pggen/internal/texts"
	"github.com/peterbourgon/ff/v3/ffcli"
	"go.uber.org/zap"
)

// Set via ldflags for release binaries.
var (
	version = "dev"
	commit  = "head"
)

var flagHelp = `pggen generates type-safe code from files containing Postgres queries by running
the queries on Postgres to get type information.

EXAMPLES
  # Generate code for a single query file using an existing postgres database.
  pggen gen go --query-glob author/queries.sql --postgres-connection "user=postgres port=5555 dbname=pggen"

  # Generate code using Docker to create the postgres database with a schema 
  # file. --schema-glob arg implies using Dockerized postgres.
  pggen gen go --schema-glob author/schema.sql --query-glob author/queries.sql

  # Generate code for all queries underneath a directory. Glob should be quoted
  # to prevent shell expansion.
  pggen gen go --schema-glob author/schema.sql --query-glob 'author/**/*.sql'

  # Use custom acronym when converting from camel_case_api to camelCaseAPI.
  pggen gen go --schema-glob schema.sql --query-glob query.sql --acronym api
`

func run() error {
	rootFlagSet := flag.NewFlagSet("root", flag.ExitOnError)
	rootCmd := &ffcli.Command{
		ShortUsage: "pggen <subcommand> [options...]",
		LongHelp:   flagHelp,
		FlagSet:    rootFlagSet,
		Subcommands: []*ffcli.Command{
			newGenCmd(),
			newVersionCmd(),
		},
	}
	rootCmd.Exec = func(ctx context.Context, args []string) error {
		fmt.Println(ffcli.DefaultUsageFunc(rootCmd))
		os.Exit(1)
		return nil
	}
	if err := rootCmd.ParseAndRun(context.Background(), os.Args[1:]); err != nil {
		return err
	}
	return nil
}

func newVersionCmd() *ffcli.Command {
	cmd := &ffcli.Command{
		Name:       "version",
		ShortUsage: "prints pggen version",
		Exec: func(ctx context.Context, args []string) error {
			fmt.Printf("pggen version %s, commit %s\n", version, commit)
			return nil
		},
	}
	return cmd
}

func newGenCmd() *ffcli.Command {
	fset := flag.NewFlagSet("go", flag.ExitOnError)
	outputDir := fset.String("output-dir", "",
		"where to write generated code; defaults to same directory as query files")
	postgresConn := fset.String("postgres-connection", "",
		`optional connection string to a postgres database, like: `+
			`"user=postgres host=localhost dbname=pggen"`)
	queryGlobs := flags.Strings(fset, "query-glob", nil,
		"generate code for all SQL files that match glob, like 'queries/**/*.sql'")
	schemaGlobs := flags.Strings(fset, "schema-glob", nil,
		"create schema in Postgres from all sql, sql.gz, or shell "+
			"scripts (*.sh) that match a glob, like 'migrations/*.sql'")
	acronyms := flags.Strings(fset, "acronym", nil,
		"lowercase acronym that should convert to all caps like 'api', "+
			"or custom mapping like 'apis=APIs'")
	goTypes := flags.Strings(fset, "go-type", nil,
		"custom type mapping from Postgres to fully qualified Go type, "+
			"like 'device_type=github.com/jschaf/pggen.DeviceType'")
	logLvl := zap.InfoLevel
	fset.Var(&logLvl, "log", "log level: debug, info, or error")
	goSubCmd := &ffcli.Command{
		Name:       "go",
		ShortUsage: "pggen gen go --query-glob glob [--schema-glob <glob>]... [flags]",
		ShortHelp:  "generates go code for Postgres query files",
		FlagSet:    fset,
		LongHelp: flagHelp + "\n" + texts.Dedent(`
			pggen uses the provided --postgres-connection to query the database. If not 
			present, pggen creates a Docker container to query the database.
		`),
		Exec: func(ctx context.Context, args []string) error {
			// Preconditions.
			if len(*queryGlobs) == 0 {
				return fmt.Errorf("pggen gen go: at least one file in --query-glob must match")
			}
			queries, err := expandSortGlobs(*queryGlobs)
			if err != nil {
				return err
			}
			schemas, err := expandSortGlobs(*schemaGlobs)
			if err != nil {
				return err
			}
			// Deduce output directory.
			outDir := *outputDir
			if outDir == "" {
				for _, file := range queries {
					dir := filepath.Dir(file)
					if outDir != "" && dir != outDir {
						return fmt.Errorf("cannot deduce output dir because query files use different dirs; " +
							"specify explicitly with --output-dir")
					}
					outDir = dir
				}
			}

			outDir, _ = filepath.Abs(outDir)

			// Parse two acronym formats: "--acronym api" and "--acronym oids=OIDs"
			acros := make(map[string]string)
			for _, acro := range *acronyms {
				ss := strings.SplitN(acro, "=", 2)
				word := ss[0]
				if word != strings.ToLower(word) {
					return fmt.Errorf("acronym %q should be lower case", word)
				}
				replacement := strings.ToUpper(word)
				if len(ss) > 1 {
					replacement = ss[1]
				}
				acros[word] = replacement
			}

			typeOverrides := make(map[string]string, len(*goTypes))
			for _, typeAssoc := range *goTypes {
				if strings.Count(typeAssoc, "=") != 1 {
					return fmt.Errorf("--go-type must have format <pgType>=<goType>; got %s", typeAssoc)
				}
				ss := strings.SplitN(typeAssoc, "=", 2)
				typeOverrides[ss[0]] = ss[1]
			}

			// Codegen.
			err = pggen.Generate(pggen.GenerateOptions{
				Language:      pggen.LangGo,
				ConnString:    *postgresConn,
				SchemaFiles:   schemas,
				QueryFiles:    queries,
				OutputDir:     outDir,
				Acronyms:      acros,
				TypeOverrides: typeOverrides,
				LogLevel:      logLvl,
			})
			if err != nil {
				return err
			}

			filesDesc := "files"
			if len(queries) == 1 {
				filesDesc = "file"
			}
			fmt.Printf("generated %d query %s\n", len(queries), filesDesc)
			return nil
		},
	}
	cmd := &ffcli.Command{
		Name:        "gen",
		ShortUsage:  "pggen gen (go|<lang>) [options...]",
		ShortHelp:   "generates code in specific language for Postgres query files",
		FlagSet:     nil,
		Subcommands: []*ffcli.Command{goSubCmd},
	}
	cmd.Exec = func(ctx context.Context, args []string) error {
		fmt.Println(ffcli.DefaultUsageFunc(cmd))
		os.Exit(1)
		return nil
	}
	return cmd
}

// expandSortGlobs gets the absolute paths for all files matching globs. Order
// files lexicographically within each glob but not across all globs. The order
// of a glob relative to other globs is important for schemas where a schema
// might depend on a previous schema.
func expandSortGlobs(globs []string) ([]string, error) {
	files := make([]string, 0, len(globs)*4)
	for _, glob := range globs {
		var matches []string
		if !strings.ContainsAny(glob, "*?[{") {
			// A regular file, not a glob. Check if it exists.
			if _, err := os.Stat(glob); os.IsNotExist(err) {
				return nil, fmt.Errorf("file does not exist: %w", err)
			}
			matches = append(matches, glob)
		} else {
			ms, err := doublestar.Glob(glob)
			if err != nil {
				// Ignore err, it's not helpful.
				return nil, fmt.Errorf("bad glob pattern: %s", glob)
			}
			sort.Strings(ms)
			matches = ms
		}
		files = append(files, matches...)
	}
	for i, schema := range files {
		abs, err := filepath.Abs(schema)
		if err != nil {
			return nil, fmt.Errorf("absolute path for %s: %w", schema, err)
		}
		files[i] = abs
	}
	return files, nil
}

func main() {
	if err := run(); err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
		os.Exit(1)
	}
}
