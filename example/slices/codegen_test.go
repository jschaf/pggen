package slices

import (
	"github.com/jschaf/pggen"
	"github.com/jschaf/pggen/internal/difftest"
	"github.com/jschaf/pggen/internal/pgtest"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerate_Go_Example_Slices(t *testing.T) {
	conn, cleanupFunc := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanupFunc()

	tmpDir := t.TempDir()
	err := pggen.Generate(
		pggen.GenerateOptions{
			ConnString: conn.Config().ConnString(),
			QueryFiles: []string{"query.sql"},
			OutputDir:  tmpDir,
			GoPackage:  "slices",
			Language:   pggen.LangGo,
			TypeOverrides: map[string]string{
				"_bool":        "[]bool",
				"bool":         "bool",
				"timestamp":    "*time.Time",
				"_timestamp":   "[]*time.Time",
				"timestamptz":  "*time.Time",
				"_timestamptz": "[]time.Time",
			},
		})
	if err != nil {
		t.Fatalf("Generate() example/slices: %s", err)
	}

	wantQueriesFile := "query.sql.go"
	gotQueriesFile := filepath.Join(tmpDir, "query.sql.go")
	assert.FileExists(t, gotQueriesFile, "Generate() should emit query.sql.go")
	wantQueries, err := os.ReadFile(wantQueriesFile)
	if err != nil {
		t.Fatalf("read wanted query.go.sql: %s", err)
	}
	gotQueries, err := os.ReadFile(gotQueriesFile)
	if err != nil {
		t.Fatalf("read generated query.go.sql: %s", err)
	}
	difftest.AssertSame(t, wantQueries, gotQueries)
}
