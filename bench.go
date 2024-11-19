package main

import (
	"hash/fnv"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// expandPackages expands the package filter into all of the packages that it
// references using `go list`.
func expandPackages(pkgFilter []string) ([]string, error) {
	args := []string{"go", "list"}
	args = append(args, pkgFilter...)
	pkgs, err := capture(args...)
	if err != nil {
		return nil, errors.Wrap(err, "expanding packages")
	}
	return strings.Split(pkgs, "\n"), nil
}

// testDir returns the directory to store benchdiff artifacts and binaries for
// specified git ref.
func testDir(ref string) string {
	return filepath.Join("benchdiff", ref)
}

// testArtifactsDir returns the directory to store benchdiff artifacts for
// specified git ref.
func testArtifactsDir(ref string) string {
	return filepath.Join(testDir(ref), "artifacts")
}

func hash(s []string) string {
	h := fnv.New32a()
	for _, ss := range s {
		h.Write([]byte(ss))
	}
	u := h.Sum32()
	return strconv.Itoa(int(u))
}

// testArtifactsDir returns the directory to store benchdiff binaries for
// specified git ref.
func testBinDir(ref string, pkgFilter []string) string {
	return filepath.Join(testDir(ref), "bin", hash(pkgFilter))
}

// pkgToTestBin translates a Go package name into a test binary name.
func pkgToTestBin(pkg string) string {
	// Strip github.com prefix.
	f := strings.TrimPrefix(pkg, "github.com")
	// Turn forward-slashes into underscores.
	f = strings.ReplaceAll(f, "/", "_")
	// Trim leading underscores.
	return strings.TrimLeft(f, "_")
}

// testBinToPkg translates a test binary name to a Go package name. This
// tranlation does not round-trip, but comes close enough.
func testBinToPkg(bin string) string {
	return strings.ReplaceAll(bin, "_", "/")
}

// buildTestBin builds a test binary for the specified package and moves it to
// the destination directory if successful.
func buildTestBin(pkg, dst string) (string, bool, error) {
	dstFile := pkgToTestBin(pkg)
	// Capture to silence warnings from pkgs with no test files.
	if _, err := capture("go", "test", "-c", "-o", dstFile, pkg); err != nil {
		return "", false, errors.Wrap(err, "building test binary")
	}
	// If there were no tests in the package, no file will have been created.
	if _, err := os.Stat(dstFile); err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, errors.Wrap(err, "looking for test binary")
	}
	if err := spawn("mv", dstFile, dst); err != nil {
		return "", false, errors.Wrap(err, "moving test binary")
	}
	return dstFile, true, nil
}
