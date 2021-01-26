package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/jschaf/pggen/codegen"
	"github.com/jschaf/pggen/codegen/gen"
	"github.com/jschaf/pggen/internal/flags"
	"github.com/peterbourgon/ff/v3/ffcli"
	"os"
	"path/filepath"
	"strings"
)

const flagHelp = `
pggen generates type-safe code from a files containing Postgres queries.
`

func run() error {
	genCmd := newGenCmd()
	rootFlagSet := flag.NewFlagSet("root", flag.ExitOnError)
	rootCmd := &ffcli.Command{
		ShortUsage: "pggen <subcommand> [options...]",
		LongHelp:   flagHelp[1 : len(flagHelp)-1], // remove lead/trail newlines
		FlagSet:    rootFlagSet,
		Subcommands: []*ffcli.Command{
			genCmd,
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

func newGenCmd() *ffcli.Command {
	fset := flag.NewFlagSet("go", flag.ExitOnError)
	outputDir := fset.String("output-dir", "", "where to write generated code; defaults to query file dir")
	queryFiles := flags.Strings(fset, "query-file", nil, "generate code for query file")
	goSubCmd := &ffcli.Command{
		Name:       "go",
		ShortUsage: "pggen gen go [options...]",
		ShortHelp:  "generates go code for Postgres query files",
		FlagSet:    fset,
		Exec: func(ctx context.Context, args []string) error {
			if len(*queryFiles) == 0 {
				return fmt.Errorf("pggen gen go: at least one --query-file path must be specified")
			}
			// Get absolute paths.
			files := make([]string, len(*queryFiles))
			for i, file := range *queryFiles {
				abs, err := filepath.Abs(file)
				if err != nil {
					return fmt.Errorf("absolute path for %s: %w", file, err)
				}
				files[i] = abs
			}
			// Deduce output directory.
			outDir := *outputDir
			if outDir == "" {
				for _, file := range files {
					dir := filepath.Dir(file)
					if outDir != "" && dir != outDir {
						return fmt.Errorf("cannot deduce output dir because query files use different dirs; " +
							"specify explicitly with --output-dir")
					}
					outDir = dir
				}
			}
			// Codegen.
			err := codegen.Generate(gen.GenerateOptions{
				Language:   gen.LangGo,
				ConnString: "user=postgres password=hunter2 host=localhost port=5555 dbname=pggen",
				QueryFiles: files,
				Config:     gen.Config{},
				OutputDir:  outDir,
			})
			fmt.Printf("gen go: out_dir=%s files=%s\n", outDir, strings.Join(files, ","))
			return err
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

func main() {
	if err := run(); err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
		os.Exit(1)
	}
}
