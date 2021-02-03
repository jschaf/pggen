package example

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
)

var update = flag.Bool("update", false, "update integration tests if true")

const projDir = ".." // hardcoded

func TestMain(m *testing.M) {
	flag.Parse()
	if *update {
		// update only disables the assertions. Running the test causes pggen
		// to overwrite generated code.
		fmt.Println("updating integration test generated files")
	}
	os.Exit(m.Run())
}

// Checks that running pggen doesn't generate a diff.
func TestExamples(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "example/enums",
			args: []string{
				"--schema-glob", "example/enums/schema.sql",
				"--query-glob", "example/enums/query.sql",
			},
		},
	}
	pggen := compilePggen(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runPggen(t, pggen, tt.args...)
			if !*update {
				assertNoDiff(t)
			}
		})
	}
}

func runPggen(t *testing.T, pggen string, args ...string) string {
	cmd := exec.Cmd{
		Path: pggen,
		Args: append([]string{pggen, "gen", "go"}, args...),
		Dir:  projDir,
	}
	t.Log("running pggen")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Log("pggen output:\n" + string(bytes.TrimSpace(output)))
		t.Fatalf("run pggen: %s", err)
	}
	return pggen
}

func compilePggen(t *testing.T) string {
	tempDir := t.TempDir()
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Fatalf("lookup go path: %s", err)
	}
	pggen := filepath.Join(tempDir, "pggen")
	cmd := exec.Cmd{
		Path: goBin,
		Args: []string{goBin, "build", "-o", pggen, "./cmd/pggen"},
		Env:  os.Environ(),
		Dir:  projDir,
	}
	t.Log("compiling pggen")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Log("go build output:\n" + string(bytes.TrimSpace(output)))
		t.Fatalf("compile pggen: %s", err)
	}
	return pggen
}

var (
	gitBin     string
	gitBinErr  error
	gitBinOnce = &sync.Once{}
)

func assertNoDiff(t *testing.T) {
	gitBinOnce.Do(func() {
		gitBin, gitBinErr = exec.LookPath("git")
		if gitBinErr != nil {
			gitBinErr = fmt.Errorf("lookup git path: %w", gitBinErr)
		}
	})
	if gitBinErr != nil {
		t.Fatal(gitBinErr)
	}
	updateIndexCmd := exec.Cmd{
		Path: gitBin,
		Args: []string{gitBin, "update-index", "--refresh"},
		Env:  os.Environ(),
		Dir:  projDir,
	}
	updateOutput, err := updateIndexCmd.CombinedOutput()
	if err != nil {
		t.Log("git update-index output:\n" + string(bytes.TrimSpace(updateOutput)))
		t.Fatalf("git update-index: %s", err)
	}
	diffIndexCmd := exec.Cmd{
		Path: gitBin,
		Args: []string{gitBin, "diff-index", "--quiet", "HEAD", "--"},
		Env:  os.Environ(),
		Dir:  projDir,
	}
	diffOutput, err := diffIndexCmd.CombinedOutput()
	if err != nil {
		t.Log("git diff-index output:\n" + string(bytes.TrimSpace(diffOutput)))
		t.Fatalf("git diff-index: %s", err)
	}
}
