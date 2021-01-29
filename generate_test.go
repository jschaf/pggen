package pggen

import (
	"github.com/jschaf/pggen/internal/pgtest"
	"github.com/jschaf/pggen/internal/texts"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"path/filepath"
	"testing"
)

func TestGenerate_Golang_Error(t *testing.T) {
	tests := []struct {
		name       string
		schema     string
		queries    string
		wantErrMsg string
	}{
		{
			name:   "duplicate query name",
			schema: "",
			queries: texts.Dedent(`
			-- name: Foo :many
			SELECT 1;
			-- name: Foo :many
			SELECT 1;
			`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, cleanupFunc := pgtest.NewPostgresSchemaString(t, tt.schema)
			defer cleanupFunc()
			tmpDir := t.TempDir()
			queryFile := filepath.Join(tmpDir, "query.sql")
			err := ioutil.WriteFile(queryFile, []byte(tt.queries), 0644)
			if err != nil {
				t.Fatal(err)
			}

			err = Generate(
				GenerateOptions{
					ConnString: conn.Config().ConnString(),
					QueryFiles: []string{queryFile},
					OutputDir:  tmpDir,
					GoPackage:  "error_test",
					Language:   LangGo,
				})

			if err == nil {
				t.Fatal("expected error from generate")
			}
			assert.Contains(t, err.Error(), tt.wantErrMsg, "error message should contain substring")
		})
	}
}
