# linter

A command line tool to check go code for abnormal (rage) quits and panics.

## Checked literals

- __panic__ - not allowed at all
- call to __os.Exit__ - not allowed outside main function of package main
- call to __log.Fatal__ or __log.Fatalf__ - not allowed outside main function of package main

__linter__ supports selective package exclusion (by name only) to skip generated code

## Usage

``` shell
linter -ep="<excluded_packages> ./path/to/codebase/root
```

Here, __excluded_packages__ is a comma-separated list of packages to be excluded from analysis.
I.e. "mocks,swagger"