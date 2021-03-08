# Contributing to pggen

First off, thank you for contributing! I welcome PRs. 

tl;dr:

First, read [ARCHITECTURE.md](ARCHITECTURE.md) to get a lay of the land.

```shell
# Dependencies
go get github.com/rakyll/statik
brew install golangci-lint # macOS only, see below for other OS's

# Start a long-lived postgres for integration tests
make start 

# Hack
# Commit changes

# Validate changes
make lint && make test && make acceptance-test 
# make all - equivalent
# make     - equivalent

# Send PR to GitHub. Check that tests and lints passed.

# Stop docker.
make stop
```

## Design goals of pggen

-   Minimal API surface. There should be only 1 way to run pggen. For example,
    pggen only offers a `--query-glob` flag and not also a `--query-file`
    flag. The `--query-glob` flag can also be a normal file path.
    
-   If it's possible in SQL, don't add an option in pggen. If we can use SQL
    features to control output, prefer that over adding more controls to pggen.
  
-   Correctness over convenience. Prefer to expose the nitty-gritty details of
    Postgres instead of providing ergonomic APIs. For example, pggen uses the
    pgtype structs like `pgtype.Text` instead of `string` for columns that might
    be null. `pgtext.Text` has a state field that encodes null values.
    
-   Generated code should look like a human wrote it. The generated code should
    be near perfect, including formatting. pggen doesn't depend on gofmt.

## Setup

You need to install 2 dependencies:

-   [statik] to embed the Go template into the pggen binary. Once Go 1.16 is
    released, we'll use the native go:embed command.
    
    ```shell
    go get github.com/rakyll/statik
    ```

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

[statik]: https://github.com/rakyll/statik
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
