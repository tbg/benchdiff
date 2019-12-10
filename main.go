package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"

	"github.com/nvanbenschoten/cmpbench/ui"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"golang.org/x/perf/benchstat"
)

const usage = `usage: cmpbench [--old <commit>] [--new <commit>] <pkgs>...`

const helpString = `cmpbench automates the process of running and comparing Go microbenchmarks
across code changes.

Options:
  -n, --new   <commit> measure the difference between this commit and old (default HEAD)
  -o, --old   <commit> measure the difference between this commit and new (default new~)
  -c, --count <n>      run tests and benchmarks n times (default 1)
      --help           display this help

Example invocations:
  $ cmpbench ./pkg/...
  $ cmpbench --old=master~ --new=master ./pkg/kv ./pkg/storage/...
  $ cmpbench --new=d1fbdb2 --count=2 ./pkg/sql/...`

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %s\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	var help bool
	var oldRef, newRef string
	var itersPerTest int

	pflag.Usage = func() { fmt.Fprintln(os.Stderr, usage) }
	pflag.BoolVarP(&help, "help", "h", false, "")
	pflag.StringVarP(&oldRef, "old", "o", "", "")
	pflag.StringVarP(&newRef, "new", "n", "", "")
	pflag.IntVarP(&itersPerTest, "count", "c", 10, "")
	pflag.Parse()
	prArgs := pflag.Args()

	if help {
		return runHelp(ctx)
	}
	if len(prArgs) == 0 {
		return runHelp(ctx)
	}
	pkgFilter := prArgs
	sort.Strings(pkgFilter)

	// Parse the specified git refs.
	var err error
	oldRef, newRef, err = parseGitRefs(oldRef, newRef)
	if err != nil {
		return err
	}

	// Build the benchmark suites.
	oldSuite := makeBenchSuite(oldRef)
	newSuite := makeBenchSuite(newRef)
	defer oldSuite.close()
	defer newSuite.close()
	if err := buildBenches(ctx, pkgFilter, &oldSuite, &newSuite); err != nil {
		return err
	}

	// Run the benchmarks.
	tests := oldSuite.intersectTests(&newSuite)
	err = runCmpBenches(ctx, &oldSuite, &newSuite, tests.sorted(), itersPerTest)
	if err != nil {
		return err
	}

	// Process the benchmark output.
	processBenchOutput(ctx, &oldSuite, &newSuite)
	return nil
}

func runHelp(ctx context.Context) error {
	fmt.Fprintln(os.Stderr, usage)
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, helpString)
	return nil
}

func parseGitRefs(oldRef, newRef string) (string, string, error) {
	var err error
	if newRef == "" {
		if oldRef != "" {
			return "", "", errors.New("if --old is provided, --new must be provided")
		}
		newRef, err = getCurRef()
		if err != nil {
			return "", "", err
		}
	}
	newRef = shortenRef(newRef)
	if ok, err := checkValidRef(newRef); err != nil {
		return "", "", err
	} else if !ok {
		return "", "", errors.Errorf("invalid git ref %q", newRef)
	}

	if oldRef == "" {
		oldRef, err = getPrevRef(newRef)
		if err != nil {
			return "", "", err
		}
	}
	oldRef = shortenRef(oldRef)
	if ok, err := checkValidRef(oldRef); err != nil {
		return "", "", err
	} else if !ok {
		return "", "", errors.Errorf("invalid git ref %q", oldRef)
	}

	return oldRef, newRef, nil
}

func buildBenches(ctx context.Context, pkgFilter []string, bss ...*benchSuite) error {
	// Get the current branch so we can revert to it after, if possible.
	if ref, ok, err := getCurSymbolicRef(); err != nil {
		return err
	} else if ok {
		defer checkoutRef(ref)
	}
	now := time.Now() // used to uniquely name artifact files
	for _, bs := range bss {
		if err := bs.build(pkgFilter, now); err != nil {
			return err
		}
	}
	return nil
}

func runCmpBenches(ctx context.Context, bs1, bs2 *benchSuite, tests []string, itersPerTest int) error {
	var spinner ui.Spinner
	spinner.Start(os.Stdout, "running benchmark binaries")
	defer spinner.Stop()
	for i, t := range tests {
		spinner.Update(fmt.Sprintf(": %s %s", testBinToPkg(t), ui.Fraction(i, len(tests))))
		err := runCmpBench(bs1, bs2, t, itersPerTest)
		if err != nil {
			return err
		}
	}
	spinner.Update(ui.Fraction(len(tests), len(tests)))
	return nil
}

func runCmpBench(bs1, bs2 *benchSuite, test string, itersPerTest int) error {
	for i := 0; i < itersPerTest; i++ {
		// Interleave test suite runs instead of using -count=itersPerTest. The
		// idea is that this reduces the chance that we pick up external noise
		// with a time correlation.
		if err := runSingleBench(bs1, test); err != nil {
			return err
		}
		if err := runSingleBench(bs2, test); err != nil {
			return err
		}
	}
	return nil
}

func runSingleBench(bs *benchSuite, test string) error {
	bin := bs.getTestBinary(test)

	// Determine whether the binary has a --logtostderr flag. Use CombinedOutput
	// and ignore the error because --help creates a failed error status. If there
	// is a real error we'll hit it below.
	cmd := exec.Command(bin, "--help")
	out, _ := cmd.CombinedOutput()
	hasLogToStderr := bytes.Contains(out, []byte("logtostderr"))

	// Run the benchmark binary.
	args := []string{bin, "-test.run", "-", "-test.bench", ".", "-test.benchmem"}
	if hasLogToStderr {
		args = append(args, "--logtostderr", "NONE")
	}
	return spawnWith(os.Stdin, bs.outFile, bs.outFile, args...)
}

func processBenchOutput(ctx context.Context, bs1, bs2 *benchSuite) {
	// We're going to be reading the output files, so seek to the beginning.
	bs1.outFile.Seek(0, io.SeekStart)
	bs2.outFile.Seek(0, io.SeekStart)

	var c benchstat.Collection
	c.Order = benchstat.ByDelta
	c.AddFile("old", bs1.outFile)
	c.AddFile("new", bs2.outFile)
	tables := c.Tables()
	for i, table := range tables {
		fmt.Println(table.Metric)
		// norange=true suppresses the "±" range columns.
		benchstat.FormatCSV(os.Stdout, tables[i:i+1], true /* norange */)
	}
}

type benchSuite struct {
	ref       string
	artDir    string
	outFile   *os.File
	binDir    string
	testFiles fileSet
}
type fileSet map[string]struct{}

func makeBenchSuite(ref string) benchSuite {
	return benchSuite{
		ref:       ref,
		testFiles: make(fileSet),
	}
}

func (bs *benchSuite) build(pkgFilter []string, t time.Time) error {
	if len(bs.testFiles) != 0 {
		panic("benchSuite already built")
	}

	fmt.Printf("checking out '%s'\n", bs.ref)
	if err := checkoutRef(bs.ref); err != nil {
		return err
	}

	// Create the artifacts directory: ./cmpbench.<ref>/artifacts
	bs.artDir = testArtifactsDir(bs.ref)
	err := os.MkdirAll(bs.artDir, 0744)
	if err != nil {
		return err
	}

	// Create output file: ./cmpbench.<ref>/artifacts/out.<time>
	outFileName := bs.getOutputFile(t)
	bs.outFile, err = os.OpenFile(outFileName, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}

	// Create the binary directory: ./cmpbench.<ref>/bin.<hash(pkgFilter)>
	bs.binDir = testBinDir(bs.ref, pkgFilter)
	_, err = os.Stat(bs.binDir)
	if err == nil {
		fmt.Println("test binaries already exist; skipping build")
		files, err := ioutil.ReadDir(bs.binDir)
		if err != nil {
			return err
		}
		for _, f := range files {
			if f.IsDir() {
				return errors.Errorf("unexpected directory %q", f.Name())
			}
			bs.testFiles[f.Name()] = struct{}{}
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return errors.Wrap(err, "looking for test directory")
	}
	if err := os.MkdirAll(bs.binDir, 0700); err != nil {
		return err
	}

	// Determine which packages to build.
	pkgs, err := expandPackages(pkgFilter)
	if err != nil {
		return err
	}

	var spinner ui.Spinner
	spinner.Start(os.Stdout, "building benchmark binaries for "+bs.ref)
	defer spinner.Stop()
	for i, pkg := range pkgs {
		spinner.Update(ui.Fraction(i, len(pkgs)))
		if testBin, ok, err := buildTestBin(pkg, bs.binDir); err != nil {
			return err
		} else if ok {
			bs.testFiles[testBin] = struct{}{}
		}
	}
	spinner.Update(ui.Fraction(len(pkgs), len(pkgs)))
	return nil
}

func (bs *benchSuite) close() {
	_ = bs.outFile.Close()
}

func (bs *benchSuite) getOutputFile(t time.Time) string {
	const timeFormat = "2006-01-02T15_04_05Z07:00"
	return filepath.Join(bs.artDir, "out."+t.Format(timeFormat))
}

func (bs *benchSuite) getTestBinary(bin string) string {
	return filepath.Join(bs.binDir, bin)
}

func (bs *benchSuite) intersectTests(bs2 *benchSuite) fileSet {
	intersect := make(fileSet)
	for f := range bs.testFiles {
		if _, ok := bs2.testFiles[f]; ok {
			intersect[f] = struct{}{}
		}
	}
	return intersect
}

func (fs fileSet) sorted() []string {
	s := make([]string, 0, len(fs))
	for t := range fs {
		s = append(s, t)
	}
	sort.Strings(s)
	return s
}
