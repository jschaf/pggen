package example

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "update integration tests if true")

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
}

func runPggen(t *testing.T, pggen string, args ...string) string {
	cmd := exec.Cmd{
		Path: pggen,
		Args: append([]string{pggen, "gen", "go"}, args...),
		Dir:  "..",
	}
	t.Log("running pggen")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(output))
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
		Dir:  "..",
	}
	t.Log("compiling pggen")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(output))
		t.Fatalf("compile pggen: %s", err)
	}
	return pggen
}
