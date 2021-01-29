// Package gen contains common code shared between codegen and language
// specific code generators. Separate package to avoid dependency cycles.
package pggen

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
	// The name of the Go package for the file. If empty, defaults to the
	// directory name.
	GoPackage string
	// Directory to write generated files. Writes one file for each query file.
	// If more than one query file, also writes querier.go.
	OutputDir string
	// Docker init scripts to run in dockerized Postgres. Must be nil if
	// ConnString is set.
	DockerInitScripts []string
}
