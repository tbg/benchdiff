# cmpbench

A tool for automating the process of running and comparing Go benchmarks across code changes.

## Usage

```
usage: cmpbench [--old <commit>] [--new <commit>] <pkgs>...

cmpbench automates the process of running and comparing Go microbenchmarks
across code changes.

cmpbench runs all microbenchmarks in the specified packages against the old and
new commit. It then passes the benchmark output through benchstat to compute
statistics about the results.

By default, cmpbench outputs these results in a textual format. However, if the
--sheets flag is passed then it will upload the result to a Google Sheets
spreadsheet. To access this, users must have a Google service account. For
information, see https://cloud.google.com/iam/docs/service-accounts.

The Google service account must meet the following conditions:
1. The Google Sheets API must be enabled for the account's project
2. The Google Drive  API must be enabled for the account's project

When the --sheets flag is passed, cmpbench will search for a credentials file
containing the service account key using the GOOGLE_APPLICATION_CREDENTIALS
environment variable. See https://cloud.google.com/docs/authentication/production.

Options:
  -n, --new    <commit> measure the difference between this commit and old (default HEAD)
  -o, --old    <commit> measure the difference between this commit and new (default new~)
  -c, --count  <n>      run tests and benchmarks n times (default 1)
	  --post-checkout   an optional command to run after checking out each branch
	                    to configure the git repo so that 'go build' succeeds
      --sheets          output the results to a new Google sheets document
      --help            display this help

Example invocations:
  $ cmpbench --sheets ./pkg/...
  $ cmpbench --old=master~ --new=master ./pkg/kv ./pkg/storage/...
  $ cmpbench --new=d1fbdb2 --count=2 ./pkg/sql/...
  $ cmpbench --new=6299bd4 --sheets --post-checkout='make buildshort' ./pkg/workload/...
```
