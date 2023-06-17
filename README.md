[![Test](https://github.com/jschaf/pggen/workflows/Test/badge.svg)](https://github.com/jschaf/pggen/actions?query=workflow%3ATest) 
[![Lint](https://github.com/jschaf/pggen/workflows/Lint/badge.svg)](https://github.com/jschaf/pggen/actions?query=workflow%3ALint) 
[![GoReportCard](https://goreportcard.com/badge/github.com/jschaf/pggen)](https://goreportcard.com/report/github.com/jschaf/pggen)

# pggen - generate type safe Go methods from Postgres SQL queries

pggen generates Go code to provide a typesafe wrapper to run Postgres queries.
If Postgres can run the query, pggen can generate code for it. The generated 
code is strongly-typed with rich mappings between Postgres types and Go types
without relying on `interface{}`. pggen uses prepared queries, so you don't 
have to worry about SQL injection attacks. 

How to use pggen in three steps:

1.  Write arbitrarily complex SQL queries with a name and a `:one`, `:many`, or
    `:exec` annotation. Declare inputs with `pggen.arg('input_name')`.

    ```sql
    -- name: SearchScreenshots :many
    SELECT ss.id, array_agg(bl) AS blocks
    FROM screenshots ss
      JOIN blocks bl ON bl.screenshot_id = ss.id
    WHERE bl.body LIKE pggen.arg('body') || '%'
    GROUP BY ss.id
    ORDER BY ss.id
    LIMIT pggen.arg('limit') OFFSET pggen.arg('offset');
    ```

2.  Run pggen to generate Go code to create type-safe methods for each query.
   
    ```bash
    pggen gen go \
        --schema-glob schema.sql \
        --query-glob 'screenshots/*.sql' \
        --go-type 'int8=int' \
        --go-type 'text=string'
    ```
    
    That command generates methods and type definitions like below. The full
    example is in [./example/composite/query.sql.go].
    
    ```go
    type SearchScreenshotsParams struct {
        Body   string
        Limit  int
        Offset int
    }

    type SearchScreenshotsRow struct {
        ID     int      `json:"id"`
        Blocks []Blocks `json:"blocks"`
    }
    
    // Blocks represents the Postgres composite type "blocks".
    type Blocks struct {
        ID           int    `json:"id"`
        ScreenshotID int    `json:"screenshot_id"`
        Body         string `json:"body"`
    }
    
    func (q *DBQuerier) SearchScreenshots(
        ctx context.Context,
        params SearchScreenshotsParams,
    ) ([]SearchScreenshotsRow, error) {
        /* omitted */
    }
    ```
    
3.  Use the generated code.

    ```go
    var conn *pgx.Conn
	q := NewQuerier(conn)
    rows, err := q.SearchScreenshots(ctx, SearchScreenshotsParams{
        Body:   "some_prefix",
        Limit:  50,
        Offset: 200,
    })
    ```
[./example/composite/query.sql.go]: ./example/composite/query.sql.go

## Pitch

Why should you use `pggen` instead of the [myriad] of Go SQL bindings?

- pggen generates code by introspecting the database system catalogs, so you 
  can use *any* database extensions or custom methods, and it will just work.
  For database types that pggen doesn't recognize, you can provide your own
  type mappings.

- pggen scales to Postgres databases of any size and supports incremental 
  adoption. pggen is narrowly tailored to only generate code for queries you 
  write in SQL. pggen will not create a model for every database object. 
  Instead, pggen only generates structs necessary to run the queries you 
  specify.

- pggen works with any Postgres database with any extensions. Under the hood, 
  pggen runs each query and uses the Postgres catalog tables, `pg_type`, 
  `pg_class`, and `pg_attribute`, to get **perfect type information** for both 
  the query parameters and result columns.
  
- pggen works with all Postgres queries. If Postgres can run the query, pggen
  can generate Go code for the query.
  
- pggen uses [pgx], a faster replacement for [lib/pq], the original Go Postgres
  library that's now in maintenance mode.

- pggen provides a batch (aka query pipelining) interface for each generated 
  query with [`pgx.Batch`]. Query pipelining is the reason Postgres sits atop
  the [TechEmpower benchmarks]. Using a batch enables sending multiple queries
  in a single network round-trip instead of one network round-trip per query.
  
[TechEmpower benchmarks]: https://www.techempower.com/benchmarks/#section=data-r20&hw=ph&test=query
[pgx]: https://github.com/jackc/pgx
[lib/pq]: https://github.com/lib/pq

## Anti-pitch

I'd like to try to convince you why you *shouldn't* use pggen. Often, this
is far more revealing than the pitch.

- You want auto-generated models for every table in your database. pggen only
  generates code for each query in a query file. pggen requires custom SQL for
  even the simplest CRUD queries. Use [gorm] or any of alternatives listed
  at [awesome Go ORMs].

- You use a database other than Postgres. pggen only supports Postgres. [sqlc],
  a similar tool which inspired pggen, has early support for MySQL.

- You want an active-record pattern where models have methods like `find`, 
  `create`, `update`, and `delete`. pggen only generates code for queries you 
  write. Use [gorm].
  
- You prefer building queries in a Go dialect instead of SQL. I'd recommend 
  investing in really learning SQL; it will payoff. Otherwise, use 
  [squirrel], [goqu], or [go-sqlbuilder]
  
- You don't want to add a Postgres or Docker dependency to your build phase.
  Use [sqlc], though you might still need Docker. sqlc generates code by parsing
  the schema file and queries in Go without using Postgres.

[myriad]: https://github.com/d-tsuji/awesome-go-orms
[sqlc]: https://github.com/kyleconroy/sqlc
[gorm]: https://gorm.io/index.html
[squirrel]: https://github.com/Masterminds/squirrel
[goqu]: https://github.com/doug-martin/goqu
[go-sqlbuilder]: https://github.com/huandu/go-sqlbuilder
[awesome Go ORMs]: https://github.com/d-tsuji/awesome-go-orms

# Install

### Download precompiled binaries

Precompiled binaries from the latest release. Change `~/bin` if you want to
install to a different directory. All assets are listed on the [releases] page.

[releases]: https://github.com/jschaf/pggen/releases

-   MacOS Apple Silicon (arm64)

    ```shell
    mkdir -p ~/bin \
      && curl --silent --show-error --location --fail 'https://github.com/jschaf/pggen/releases/latest/download/pggen-darwin-arm64.tar.xz' \
      | tar -xJf - -C ~/bin/    
    ```
    
-   MacOS Intel (amd64)

    ```shell
    mkdir -p ~/bin \
      && curl --silent --show-error --location --fail 'https://github.com/jschaf/pggen/releases/latest/download/pggen-darwin-amd64.tar.xz' \
      | tar -xJf - -C ~/bin/    
    ```

-   Linux (amd64)

    ```shell
    mkdir -p ~/bin \
      && curl --silent --show-error --location --fail 'https://github.com/jschaf/pggen/releases/latest/download/pggen-linux-amd64.tar.xz' \
      | tar -xJf - -C ~/bin/    
    ```
    
-   Windows (amd64)

    ```shell
    mkdir -p ~/bin \
      && curl --silent --show-error --location --fail 'https://github.com/jschaf/pggen/releases/latest/download/pggen-windows-amd64.tar.xz' \
      | tar -xJf - -C ~/bin/    
    ```

Make sure pggen works:

```bash
pggen gen go --help
```

### Install from source

Requires Go 1.16 because pggen uses `go:embed`. Installs to `$GOPATH/bin`.

```shell
go install github.com/jschaf/pggen/cmd/pggen@latest
```
    
Make sure pggen works:

```bash
pggen gen go --help
```

## Usage

Generate code using Docker to create the Postgres database from a schema file:

```bash
# --schema-glob runs all matching files on Dockerized Postgres during database 
# creation.
pggen gen go \
    --schema-glob author/schema.sql \
    --query-glob author/query.sql

# Output: author/query.go.sql

# Or with multiple schema files. The schema files run on Postgres
# in the order they appear on the command line.
pggen gen go \
    --schema-glob author/schema.sql \
    --schema-glob book/schema.sql \
    --schema-glob publisher/schema.sql \
    --query-glob author/query.sql

# Output: author/query.sql.go
```

Generate code using an existing Postgres database (useful for custom setups):

```bash
pggen gen go \
    --query-glob author/query.sql \
    --postgres-connection "user=postgres port=5555 dbname=pggen"

# Output: author/query.sql.go
```

Generate code for multiple query files. All the query files must reside in
the same directory. If query files reside in different directories, you can use
`--output-dir` to set a single output directory:

```bash
pggen gen go \
    --schema-glob schema.sql \
    --query-glob author/fiction.sql \
    --query-glob author/nonfiction.sql \
    --query-glob author/bestselling.sql

# Output: author/fiction.sql.go
#         author/nonfiction.sql.go
#         author/bestselling.sql.go

# Or, using a glob. Notice quotes around glob pattern to prevent shell 
# expansion.
pggen gen go \
    --schema-glob schema.sql \
    --query-glob 'author/*.sql'
```

# Examples

Examples embedded in the repo:

- [./example/acceptance_test.go] - End-to-end examples of how to call pggen.
- [./example/author] - A single table schema with simple queries.
- [./example/composite] - Arrays of composite (aka row or table) types.
- [./example/custom_types] - Mapping new Postgres types to Go types.
- [./example/device] - Complex queries with a 1:many relationship between a 
  `user` table and `device` table.
- [./example/enums] - Postgres and Go enums.
- [./example/erp] - A few tables with mildly complex queries.
- [./example/go_pointer_types] - Mapping to pointer types like `*int` instead
  of `pgtype.Int8`.
- [./example/ltree] - Support for the ltree Postgres extension.
- [./example/nested] - Complex, nested composite (aka row or table) types.
- [./example/pgcrypto] - pgcrypto Postgres extension.
- [./example/syntax] - A smoke test of interesting SQL syntax.
- [./example/void] - Support for void in select columns.

[./example/acceptance_test.go]: ./example/acceptance_test.go
[./example/author]: ./example/author
[./example/composite]: ./example/composite
[./example/custom_types]: ./example/custom_types
[./example/device]: ./example/device
[./example/enums]: ./example/enums
[./example/erp]: ./example/erp
[./example/go_pointer_types]: ./example/go_pointer_types
[./example/ltree]: ./example/ltree
[./example/nested]: ./example/nested
[./example/syntax]: ./example/syntax
[./example/pgcrypto]: ./example/pgcrypto
[./example/void]: ./example/void

# Features

-   **JSON struct tags**: All `<query_name>Row` structs include JSON struct tags
    using the Postgres column name. To change the struct tag, use an SQL column 
    alias.
  
    ```sql
    -- name: FindAuthors :many
    SELECT first_name, last_name as family_name FROM author;
    ```
    
    Generates:
    
    ```go
    type FindAuthorsRow struct {
        FirstName   string `json:"first_name"`
        FamilyName  string `json:"family_name"`
    }
    ```

-   **Acronyms**: Custom acronym support so that `author_id` renders as 
    `AuthorID` instead of `AuthorId`. Supports two formats:
    
    1. Long form: `--acronym <word>=<relacement>`: replaces `<word>` with 
       `<replacement>` literally. Useful for plural acronyms like `author_ids` 
       which should render as `AuthorIDs`, not `AuthorIds`. For the IDs example,
        use `--acronym ids=IDs`.
       
    2. Short form: `--acronym <word>`: replaces `<word>` with uppercase 
       `<WORD>`. Equivalent to `--acronym <word>=<WORD>`
       
    By default, pggen includes `--acronym id` to render `id` as `ID`.

-   **Enums**: Postgres enums map to Go string constant enums. The Postgres 
    type:
    
    ```sql
    CREATE TYPE device_type AS ENUM ('undefined', 'phone', 'ipad');
    ```
    
    pggen generates the following Go code when used in a query:
    
    ```go
    // DeviceType represents the Postgres enum device_type.
    type DeviceType string

    const (
        DeviceTypeUndefined DeviceType = "undefined"
        DeviceTypePhone     DeviceType = "phone"
        DeviceTypeIpad      DeviceType = "ipad"
    )

    func (d DeviceType) String() string { return string(d) }
    ```

-   **Custom types**: Use a custom Go type to represent a Postgres type with the 
    `--go-type` flag. The format is `<pg_type>=<qualified_go_type>`. For 
    example:

    ```sh
    pggen gen go \
        --schema-glob example/custom_types/schema.sql \
        --query-glob example/custom_types/query.sql \
        --go-type 'int8=*int' \
        --go-type 'int4=int' \
        --go-type '_int4=[]int' \
        --go-type 'text=*github.com/jschaf/pggen/mytype.String' \
        --go-type '_text=[]*github.com/jschaf/pggen/mytype.String'
    ```
    
    pgx must be able to decode the Postgres type using the given Go type. That 
    means the Go type must fulfill at least one of following:
    
    - The Go type is a wrapper around primitive type, like `type AuthorID int`.
      pgx will use decode methods on the underlying primitive type.

    - The Go type implements both [`pgtype.BinaryDecoder`] and 
      [`pgtype.TextDecoder`]. pgx will use the correct decoder based on the wire
      format. See the [pgtype repo] for many example types.
      
    - The pgx connection executing the query must have registered a data type 
      using the Go type with [`ConnInfo.RegisterDataType`]. See the 
      [example/custom_types test] for an example.
      
      ```go
      ci := conn.ConnInfo()
      
      ci.RegisterDataType(pgtype.DataType{
      	Value: new(pgtype.Int2),
      	Name:  "my_int",
      	OID:   myIntOID,
      })
      ```
      
    - The Go type implements [`sql.Scanner`].
    
    - pgx is able to use reflection to build an object to write fields into.

-   **Nested structs (composite types)**: pggen creates child structs to 
    represent Postgres [composite types] that appear in output columns.

    ```sql
    -- name: FindCompositeUser :one
    SELECT ROW (15, 'qux')::"user" AS "user";
    ```
    
    pggen generates the following Go code:
    
    ```go
    // User represents the Postgres composite type "user".
    type User struct {
        ID   pgtype.Int8
        Name pgtype.Text
    }
    
    func (q *DBQuerier) FindCompositeUser(ctx context.Context) (User, error) {}
    ```

[pgtype repo]: https://github.com/jackc/pgtype
[`pgtype.BinaryDecoder`]: https://pkg.go.dev/github.com/jackc/pgtype#BinaryDecoder
[`pgtype.TextDecoder`]: https://pkg.go.dev/github.com/jackc/pgtype#TextDecoder
[`ConnInfo.RegisterDataType`]: https://pkg.go.dev/github.com/jackc/pgtype#ConnInfo.RegisterDataType
[`sql.Scanner`]: https://golang.org/pkg/database/sql/#Scanner
[composite types]: https://www.postgresql.org/docs/current/rowtypes.html
[example/custom_types test]: ./example/custom_types/query.sql_test.go

# IDE integration

If your IDE provides SQL autocomplete, you may want to get rid of its warnings
by declaring the following DDL schema.

```sql
-- Exists solely so editors don't underline every pggen.arg() expression in
-- squiggly red.
CREATE SCHEMA pggen;

-- pggen.arg defines a named parameter that's eventually compiled into a
-- placeholder for a prepared query: $1, $2, etc.
CREATE FUNCTION pggen.arg(param TEXT) RETURNS text AS $$SELECT null$$ LANGUAGE sql;
```

# Tutorial

Let's say we have a database with the following schema in `author/schema.sql`:

```sql
CREATE TABLE author (
  author_id  serial PRIMARY KEY,
  first_name text NOT NULL,
  last_name  text NOT NULL,
  suffix     text NULL
)
```

First, write a query in the file `author/query.sql`. The query name is 
`FindAuthors` and the query returns `:many` rows. A query can return `:many` 
rows, `:one` row, or `:exec` for update, insert, and delete queries.

```sql
-- FindAuthors finds authors by first name.
-- name: FindAuthors :many
SELECT * FROM author WHERE first_name = pggen.arg('first_name');
```

Second, use pggen to generate Go code to `author/query.sql.go`:

```bash
pggen gen go \
    --schema-glob author/schema.sql \
    --query-glob author/query.sql
```

We'll walk through the generated file `author/query.sql.go`:

-   The `Querier` interface defines the interface with methods for each SQL 
    query. Each SQL query compiles into three methods, one method for to run 
    the query by itself, and two methods to support batching a query with 
    [`pgx.Batch`]. 
  
    ```go
    // Querier is a typesafe Go interface backed by SQL queries.
    //
    // Methods ending with Batch enqueue a query to run later in a pgx.Batch. After
    // calling SendBatch on pgx.Conn, pgxpool.Pool, or pgx.Tx, use the Scan methods
    // to parse the results.
    type Querier interface {
        // FindAuthors finds authors by first name.
        FindAuthors(ctx context.Context, firstName string) ([]FindAuthorsRow, error)
        // FindAuthorsBatch enqueues a FindAuthors query into batch to be executed
        // later by the batch.
        FindAuthorsBatch(batch *pgx.Batch, firstName string)
        // FindAuthorsScan scans the result of an executed FindAuthorsBatch query.
        FindAuthorsScan(results pgx.BatchResults) ([]FindAuthorsRow, error)
    }
    ```
    
    To use the batch interface, create a `*pgx.Batch`, call the 
    `<query_name>Batch` methods, send the batch, and finally get the results 
    with the `<query_name>Scan` methods. See [example/author/query.sql_test.go] 
    for complete example.
    
    ```sql
	q := NewQuerier(conn)
	batch := &pgx.Batch{}
	q.FindAuthorsBatch(batch, "alice")
	q.FindAuthorsBatch(batch, "bob")
	results := conn.SendBatch(context.Background(), batch)
	aliceAuthors, err := q.FindAuthorsScan(results)
	bobAuthors, err := q.FindAuthorsScan(results)
    ```

-   The `DBQuerier` struct implements the `Querier` interface with concrete
    implementations of each query method.

    ```sql
    type DBQuerier struct {
        conn genericConn
    }
    ```

-   Create `DBQuerier` with `NewQuerier`. The `genericConn` parameter is an 
    interface over the different pgx connection transports so that `DBQuerier` 
    doesn't force you to use a specific connection transport. [`*pgx.Conn`], 
    [`pgx.Tx`], and [`*pgxpool.Pool`] all implement `genericConn`.

    ```sql
    // NewQuerier creates a DBQuerier that implements Querier. conn is typically
    // *pgx.Conn, pgx.Tx, or *pgxpool.Pool.
    func NewQuerier(conn genericConn) *DBQuerier {
        return &DBQuerier{
            conn: conn,
        }
    }
    ```
    
-   pggen embeds the SQL query formatted for a Postgres `PREPARE` statement with
    parameters indicated by `$1`, `$2`, etc. instead of 
    `pggen.arg('first_name')`.

    ```sql
    const findAuthorsSQL = `SELECT * FROM author WHERE first_name = $1;`
    ```
    
-   pggen generates a row struct for each query named `<query_name>Row`.
    pggen transforms the output column names into struct field names from
    `lower_snake_case` to `UpperCamelCase` in [internal/casing/casing.go]. 
    pggen derives JSON struct tags from the Postgres column names. To change the
    JSON struct name, change the column name in the query.
    
    ```sql
    type FindAuthorsRow struct {
        AuthorID  int32       `json:"author_id"`
        FirstName string      `json:"first_name"`
        LastName  string      `json:"last_name"`
        Suffix    pgtype.Text `json:"suffix"`
    }
    ```

    As a convenience, if a query only generates a single column, pggen skips
    creating the `<query_name>Row` struct and returns the type directly.  For
    example, the generated query for `SELECT author_id from author` returns 
    `int32`, not a `<query_name>Row` struct.
    
    pggen infers struct field types by preparing the query. When Postgres
    prepares a query, Postgres returns the parameter and column types as OIDs.
    pggen finds the type name from the returned OIDs in
    [internal/codegen/golang/gotype/types.go].
    
    Choosing an appropriate type is more difficult than might seem at first 
    glance due to `null`. When Postgres reports that a column has a type `text`,
    that column can have  both `text` and `null` values. So, the Postgres `text`
    represented in Go can be either a `string` or `nil`. [`pgtype`] provides 
    nullable types for all built-in Postgres types. pggen tries to infer if a 
    column is nullable or non-nullable. If a column is nullable, pggen uses a 
    `pgtype` Go type like `pgtype.Text`. If a column is non-nullable, pggen uses
     a more ergonomic type like `string`. pggen's nullability inference
     implemented in [internal/pginfer/nullability.go] is rudimentary; a proper
     approach requires a full explain-plan with some control flow analysis.
    
-   Lastly, pggen generates the implementation for each query.

    As a convenience, if a there are only one or two query parameters, pggen
    inlines the parameters into the method definition, as with `firstName` 
    below. If there are three or more parameters, pggen creates a struct named
    `<query_name>Params` to pass the parameters to the query method.
    
    ```sql
    // FindAuthors implements Querier.FindAuthors.
    func (q *DBQuerier) FindAuthors(ctx context.Context, firstName string) ([]FindAuthorsRow, error) {
        rows, err := q.conn.Query(ctx, findAuthorsSQL, firstName)
        if rows != nil {
            defer rows.Close()
        }
        if err != nil {
            return nil, fmt.Errorf("query FindAuthors: %w", err)
        }
        items := []FindAuthorsRow{}
        for rows.Next() {
            var item FindAuthorsRow
            if err := rows.Scan(&item.AuthorID, &item.FirstName, &item.LastName, &item.Suffix); err != nil {
                return nil, fmt.Errorf("scan FindAuthors row: %w", err)
            }
            items = append(items, item)
        }
        if err := rows.Err(); err != nil {
            return nil, err
        }
        return items, err
    }
    ```

[example/author/query.sql_test.go]: ./example/author/query.sql_test.go
[`pgx.Batch`]: https://pkg.go.dev/github.com/jackc/pgx#Batch
[`*pgx.Conn`]: https://pkg.go.dev/github.com/jackc/pgx#Conn
[`pgx.Tx`]: https://pkg.go.dev/github.com/jackc/pgx#Tx
[`*pgxpool.Pool`]: https://pkg.go.dev/github.com/jackc/pgx/v4/pgxpool#Pool
[internal/casing/casing.go]: ./internal/casing/casing.go
[internal/codegen/golang/gotype/types.go]: ./internal/codegen/golang/gotype/types.go
[`pgtype`]: https://pkg.go.dev/github.com/jackc/pgtype
[internal/pginfer/nullability.go]: ./internal/pginfer/nullability.go

# Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) and [ARCHITECTURE.md](ARCHITECTURE.md).

# Acknowledgments

pggen was directly inspired by [sqlc]. The primary difference between pggen and
sqlc is how each tool infers the type and nullability of the input parameters
and output columns for SQL queries.

sqlc parses the queries in Go code, using Cgo to call the Postgres `parser.c` 
library. After parsing, sqlc infers the types of the query parameters and result
columns using custom logic in Go. In contrast, pggen gets the same type 
information by running the queries on Postgres and then fetching the type 
information for Postgres catalog tables. 

Use sqlc if you don't wish to run Postgres to generate code or if you need
better nullability analysis than pggen provides.

Use pggen if you can run Postgres for code generation, and you use complex 
queries that sqlc is unable to parse. Additionally, use pggen if you have a 
custom database setup that's difficult to replicate in a schema file. pggen
supports running on any database with any extensions.
