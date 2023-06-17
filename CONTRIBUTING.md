# Contributing to pggen

First off, thank you for contributing! I welcome PRs. 

tl;dr:

First, read [ARCHITECTURE.md](ARCHITECTURE.md) to get a lay of the land.

```shell
# Dependencies - see Setup below

# Start a long-lived Postgres server in Docker for integration tests.
# Connect with "make psql"
make start 

# Hack
# Commit changes

# Validate changes
make lint && make test && make acceptance-test 
# make all - equivalent
# make     - equivalent

# Send PR to GitHub. Check that tests and lints passed.

# Stop Postgres server running in Docker.
make stop
```

## Design goals of pggen

-   Minimal API surface. There should be only 1 way to run pggen. For example,
    pggen only offers a `--query-glob` flag and not also a `--query-file`
    flag. The `--query-glob` flag can also be a normal file path.
    
-   If it's possible in SQL, don't add an option in pggen. If we can use SQL
    features to control output, prefer that over adding more controls to pggen.
  
-   Correctness over convenience. Prefer to expose the nitty-gritty details of
    Postgres instead of providing ergonomic APIs.
    
-   Generated code should look like a human wrote it. The generated code should
    be near perfect, including formatting. pggen output doesn't depend on gofmt.

## Setup

You need to install 1 dependency:

-   [golangci-lint] to lint the project locally.

    For macOS:
    
    ```shell
    brew install golangci-lint
    ```
    
    For Windows and Linux:

    ```shell
    # binary will be $(go env GOPATH)/bin/golangci-lint
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.36.0
    golangci-lint --version
    ````

[golangci-lint]: https://golangci-lint.run/

## Testing

To test pggen, you'll typically start a long-lived Docker container with a 
Postgres instance.  The pggen tests create a new Postgres schema to isolate
tests from one another. Creating a new schema is much faster than spinning up a
new Dockerized Postgres instance.

```shell
make start
make test # all unit tests
```

To run the acceptance tests to validate that pggen produces the same code as 
the checked-in example code:

```shell
# Acceptance tests check that there's no Git diffs so commit code first.
git commit -m "some message" 

make acceptance-test
```

To update the acceptance tests after changing the code generator:

```shell
make update-acceptance-test
```

### Testing hierarchy

pggen has tests at most parts of the testing hierarchy.

-   Unit tests to test the logic of small, independent components, like 
    [casing_test.go]. Run with `make test`.
    
-   Integration tests like the [pginfer_test.go] to test that the code works
    (integrates) with different subsystems like Postgres, Docker, or other Go
    packages. As with unit tests, run with `make test`.
    
-   Acceptance tests like [example/nested/codegen_test.go] to test that pggen
    produces the exact same output as the checked-in examples. Run with 
    `make acceptance-test`.
    
[casing_test.go]: internal/casing/casing_test.go
[pginfer_test.go]: internal/pginfer/pginfer_test.go
[example/nested/codegen_test.go]: example/nested/codegen_test.go

## Debugging

For unit-testable things, like type resolution, there should be a test you can 
debug.

For debugging codegen bugs, the best place to start is the `codegen_test.go`
file in each folder in ./example.

To debug generated query execution, start with the `query.sql_test.go` file in 
each example. I've structured the tests (at least the recent ones like 
`example/author`) so that every generated query has an isolated subtest you can
debug.

For tests that use a Postgres instance, you can find the schema used in the test
in the test output. You can connect to that schema with:

```
PGPASSWORD=hunter2 psql --host=127.0.0.1 --port=5555 --username=postgres pggen

postgres> set search_path to 'pggen_test_<SOME_NUMBER>'
```