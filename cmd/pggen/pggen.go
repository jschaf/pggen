package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/peterbourgon/ff/v3/ffcli"
	"os"
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
		return fmt.Errorf("run pggen cmd: %w", err)
	}
	return nil
}

func newGenCmd() *ffcli.Command {
	fset := flag.NewFlagSet("go", flag.ExitOnError)
	goSubCmd := &ffcli.Command{
		Name:       "go",
		ShortUsage: "pggen gen go [options...]",
		ShortHelp:  "generates go code for Postgres query files",
		FlagSet:    fset,
		Exec: func(ctx context.Context, args []string) error {
			fmt.Println("gen go")
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

func main() {
	if err := run(); err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
		os.Exit(1)
	}
}
