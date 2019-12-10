# cmpbench

A tool for automating the process of running and comparing micro-benchmarks across code changes.

## Usage

```
$ cmpbench --help
usage: cmpbench [--old <commit>] [--new <commit>] <pkgs>...

cmpbench automates the process of running and comparing Go microbenchmarks
across code changes.

Options:
  -n, --new   <commit> measure the difference between this commit and old (default HEAD)
  -o, --old   <commit> measure the difference between this commit and new (default new~)
  -c, --count <n>      run tests and benchmarks n times (default 1)
      --help           display this help

Example invocations:
  $ cmpbench ./pkg/...
  $ cmpbench --old=master~ --new=master ./pkg/kv ./pkg/storage/...
  $ cmpbench --new=d1fbdb2 --count=2 ./pkg/sql/...
```
