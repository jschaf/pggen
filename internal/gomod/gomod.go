// Package gomod provides utilities for getting information about the current
// Go module.
package gomod

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/jschaf/pggen/internal/paths"
	"golang.org/x/mod/modfile"
)

//nolint:gochecknoglobals
var (
	goModDirOnce = &sync.Once{}
	goModDir     string
	errGoModDir  error

	goModNameOnce = &sync.Once{}
	goModPath     string
	errGoModPath  error
)

// FindDir finds the nearest directory containing a go.mod file. Checks
// the current dir and then walks up parent directories.
func FindDir() (string, error) {
	goModDirOnce.Do(func() {
		wd, err := os.Getwd()
		if err != nil {
			errGoModDir = fmt.Errorf("FindDir working dir: %w", err)
			return
		}
		goModDir, errGoModDir = paths.WalkUp(wd, "go.mod")
	})
	return goModDir, errGoModDir
}

// ParsePath finds the module path in the nearest go.mod file.
func ParsePath() (string, error) {
	goModNameOnce.Do(func() {
		dir, err := FindDir()
		if err != nil {
			errGoModPath = fmt.Errorf("find go.mod dir: %w", err)
			return
		}
		p := filepath.Join(dir, "go.mod")
		bs, err := os.ReadFile(p)
		if err != nil {
			errGoModPath = fmt.Errorf("read go.mod: %w", err)
			return
		}
		goModPath = modfile.ModulePath(bs)
	})
	return goModPath, errGoModPath
}

// GuessPackage guesses the full Go package path for a file name, relative to
// current working directory.
// Imperfect. Assumes package names always match directory names.
func GuessPackage(fileName string) (string, error) {
	goModDir, err := FindDir()
	if err != nil {
		return "", fmt.Errorf("find go.mod dir: %w", err)
	}
	goModPath, err := ParsePath()
	if err != nil {
		return "", fmt.Errorf("parse go.mod dir: %w", err)
	}
	abs, err := filepath.Abs(fileName)
	if err != nil {
		return "", fmt.Errorf("abs path for %s: %w", fileName, err)
	}
	rel, err := filepath.Rel(goModDir, abs)
	if err != nil {
		return "", fmt.Errorf("rel path to go.mod for %s: %w", fileName, err)
	}
	// Dir to remove file name. Clean to remove "./" suffix. Convert to slash to
	// get forward slashes, which match Go package paths.
	relDir := filepath.ToSlash(filepath.Clean(filepath.Dir(rel)))
	return goModPath + "/" + relDir, nil
}
