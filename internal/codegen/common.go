// Package codegen contains common code shared between codegen and language
// specific code generators. Separate package to avoid dependency cycles.
package codegen

import (
	"github.com/jschaf/pggen/internal/pginfer"
)

// QueryFile represents all of the SQL queries from a single file.
type QueryFile struct {
	SourcePath string               // absolute path to the source SQL query file
	Queries    []pginfer.TypedQuery // the typed queries
}
