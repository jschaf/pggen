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
		fmt.Println("updating integration test generated files")
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestExamples(t *testing.T) {
	pggen := compilePggen(t)
	runPggen(t, pggen,
		"--schema-glob", "example/enums/schema.sql",
		"--query-glob", "example/enums/query.sql",
	)
	assertNoDiff(t)
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
