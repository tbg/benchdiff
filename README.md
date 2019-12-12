# cmpbench

A tool for automating the process of running and comparing Go benchmarks across code changes.

## Usage

```
$ cmpbench --help
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

## Examples

Using text output:

```
$ cmpbench --new=6299bd4 ./pkg/workload/...
test binaries already exist for 'efcf66c'; skipping build
test binaries already exist for '6299bd4'; skipping build

running benchmarks:
  pkg=1/7 iter=2/2 cockroachdb/cockroach/pkg/workload -
  pkg=2/7 iter=2/2 cockroachdb/cockroach/pkg/workload/bank /
  pkg=3/7 iter=2/2 cockroachdb/cockroach/pkg/workload/faker |
  pkg=4/7 iter=2/2 cockroachdb/cockroach/pkg/workload/movr |
  pkg=5/7 iter=2/2 cockroachdb/cockroach/pkg/workload/tpcc /
  pkg=6/7 iter=2/2 cockroachdb/cockroach/pkg/workload/workloadsql \
  pkg=7/7 iter=2/2 cockroachdb/cockroach/pkg/workload/ycsb |

name                             old time/op    new time/op     delta
InitialData/tpcc/warehouses=1-8     304ms ± 4%      195ms ± 1%  -35.61%  (p=0.008 n=5+5)
InitialData/bank/rows=1000-8        281µs ± 3%      282µs ± 2%     ~     (p=0.548 n=5+5)
CSVRowsReader-8                    17.2µs ± 1%     17.5µs ± 0%   +1.80%  (p=0.016 n=5+4)
WriteCSVRows-8                     14.8µs ± 3%     15.6µs ± 5%   +5.28%  (p=0.032 n=5+5)

name                             old speed      new speed       delta
InitialData/tpcc/warehouses=1-8   363MB/s ± 4%    563MB/s ± 1%  +55.23%  (p=0.008 n=5+5)
CSVRowsReader-8                  98.1MB/s ± 1%  101.0MB/s ± 0%   +2.96%  (p=0.016 n=5+4)
InitialData/bank/rows=1000-8      413MB/s ± 3%    411MB/s ± 2%     ~     (p=0.548 n=5+5)
WriteCSVRows-8                    113MB/s ± 3%    113MB/s ± 5%     ~     (p=1.000 n=5+5)

name                             old alloc/op   new alloc/op    delta
InitialData/tpcc/warehouses=1-8     128kB ± 0%       81kB ± 0%  -36.19%  (p=0.008 n=5+5)
InitialData/bank/rows=1000-8       19.1kB ± 0%     19.1kB ± 0%     ~     (all equal)
WriteCSVRows-8                     5.70kB ± 0%     5.70kB ± 0%     ~     (all equal)
CSVRowsReader-8                    7.38kB ± 0%     7.38kB ± 0%     ~     (all equal)

name                             old allocs/op  new allocs/op   delta
InitialData/tpcc/warehouses=1-8       587 ± 0%        583 ± 0%   -0.61%  (p=0.008 n=5+5)
InitialData/bank/rows=1000-8        1.02k ± 0%      1.02k ± 0%     ~     (all equal)
WriteCSVRows-8                       50.0 ± 0%       50.0 ± 0%     ~     (all equal)
CSVRowsReader-8                      55.0 ± 0%       55.0 ± 0%     ~     (all equal)
```

Using Google Sheets output:

```
$ export GOOGLE_APPLICATION_CREDENTIALS=/Users/nathan/.service-account-creds.json
$ cmpbench --new=6299bd4 --sheets ./pkg/workload/...
test binaries already exist for 'efcf66c'; skipping build
test binaries already exist for '6299bd4'; skipping build

running benchmarks:
  pkg=1/7 iter=2/2 cockroachdb/cockroach/pkg/workload -
  pkg=2/7 iter=2/2 cockroachdb/cockroach/pkg/workload/bank /
  pkg=3/7 iter=2/2 cockroachdb/cockroach/pkg/workload/faker |
  pkg=4/7 iter=2/2 cockroachdb/cockroach/pkg/workload/movr |
  pkg=5/7 iter=2/2 cockroachdb/cockroach/pkg/workload/tpcc /
  pkg=6/7 iter=2/2 cockroachdb/cockroach/pkg/workload/workloadsql \
  pkg=7/7 iter=2/2 cockroachdb/cockroach/pkg/workload/ycsb |

generated sheet: https://docs.google.com/spreadsheets/d/...
```
