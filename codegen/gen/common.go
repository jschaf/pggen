// Package gen contains common code shared between codegen and language
// specific code generators. Separate package to avoid dependency cycles.
package gen

import (
	"github.com/jschaf/pggen/internal/pginfer"
)

type Lang string

const (
	LangGo = "go"
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
	// The overall config after merging config files and flag options.
	Config Config
	// The name of the Go package for the file. If empty, defaults to the
	// directory name.
	GoPackage string
	// Directory to write generated files. Writes one file for each query file.
	// If more than one query file, also writes querier.go.
	OutputDir string
}

// QueryFile represents all of the SQL queries from a single file.
type QueryFile struct {
	Src     string               // the source SQL file base name
	Queries []pginfer.TypedQuery // the typed queries
}
