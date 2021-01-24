package codegen

import (
	"github.com/jschaf/pggen/internal/pgtest"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"path/filepath"
	"testing"
)

func TestGenerate_Example_Author(t *testing.T) {
	conn, cleanupFunc := pgtest.NewPostgresSchema(t, []string{
		"../example/author/schema.sql",
	})
	defer cleanupFunc()

	tmpDir := t.TempDir()
	err := Generate(
		GenerateOptions{
			ConnString: conn.Config().ConnString(),
			QueryFiles: []string{
				"../example/author/queries.sql",
			},
			Config:    Config{},
			OutputDir: tmpDir,
			GoPackage: "author",
		})
	if err != nil {
		t.Fatalf("Generate() example/author: %s", err)
	}

	wantQueriesFile := "../example/author/queries.sql.go"
	gotQueriesFile := filepath.Join(tmpDir, "queries.sql.go")
	assert.FileExists(t, gotQueriesFile,
		"Generate() should emit queries.sql.go")
	wantQueries, err := ioutil.ReadFile(wantQueriesFile)
	if err != nil {
		t.Fatalf("read wanted queries.go.sql: %s", err)
	}
	gotQueries, err := ioutil.ReadFile(gotQueriesFile)
	if err != nil {
		t.Fatalf("read generated queries.go.sql: %s", err)
	}
	assert.Equalf(t, string(wantQueries), string(gotQueries),
		"Got file %s; does not match contents of %s",
		gotQueriesFile, wantQueriesFile)
}
