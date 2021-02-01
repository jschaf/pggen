[![Test](https://github.com/jschaf/pggen/workflows/Test/badge.svg)](https://github.com/jschaf/pggen/actions?query=workflow%3ATest) 
[![Lint](https://github.com/jschaf/pggen/workflows/Lint/badge.svg)](https://github.com/jschaf/pggen/actions?query=workflow%3ALint) 
[![GoReportCard](https://goreportcard.com/badge/github.com/jschaf/pggen)](https://goreportcard.com/report/github.com/jschaf/pggen)

# pggen - generate type safe Go methods from Postgres SQL queries

pggen is a tool that generates Go code to provide a typesafe wrapper around
Postgres queries. If Postgres can run the query, pggen can generate code for it.

1.  Write SQL queries with a name and `:one`, `:many`, or `:exec` annotation.

    ```sql
    -- FindAuthors finds authors by first name.
    -- name: FindAuthors :many
    SELECT * FROM author WHERE first_name = pggen.arg('FirstName');
    ```

2.  Run pggen to generate Go code to create type-safe methods for each 
    query.
   
    ```bash
    pggen gen go \
        --schema-glob author/schema.sql \
        --query-glob 'author/*.sql'
    ```
    
3.  Use the generated code.

    ```go
    var conn *pgx.Conn
	q := NewQuerier(conn)
	aliceAuthors, err := q.FindAuthors(ctx, "alice")
    ```

## Pitch

Why should you use `pggen` instead of the [myriad] of Go SQL bindings?

- pggen is narrowly tailored to only generate code for queries you write in SQL.

- pggen works with any Postgres database. Under the hood, pggen runs each query
  and uses the Postgres catalog tables, `pg_type`, `pg_class`, and 
  `pg_attribute`, to get perfect type information for both the query parameters 
  and result columns.
  
- pggen works with all Postgres queries. If Postgres can run the query, pggen
  can generate Go code for the query.
  
- pggen uses [pgx], a faster replacement for [lib/pq], the original Go Postgres
  library that's now in maintenance mode.

- pggen provides a batch interface for each generated query with [`pgx.Batch`]. 
  Using a batch enables sending multiple queries in a single network round-trip
  instead of one network round-trip per query.
  
[pgx]: https://github.com/jackc/pgx
[lib/pq]: https://github.com/lib/pq

## Anti-pitch

I'd like to try to convince you why you *shouldn't* use pggen. Often, this
is far more revealing than the pitch.

- You want auto-generated models for every table in your database. pggen only
  generates code for each query in a query file. pggen requires custom SQL for
  even the simplest CRUD queries. Use [gorm] or any of alternatives listed
  at [awesome Go ORMs].

- You use database other than Postgres. pggen only supports Postgres. [sqlc], a
  similar tool which inspired pggen, has early support for MySQL.

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

```bash
go get github.com/jschaf/pggen

# Make sure pggen works.
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

- [./script/integration-test.sh] - End-to-end examples of how to call pggen.
- [./example/author] - A single table schema with simple queries.
- [./example/erp] - A few tables with mildly complex queries.
- [./example/syntax] - A smoke test of interesting SQL syntax.

[./script/integration-test.sh]: ./script/integration-test.sh
[./example/author]: ./example/author
[./example/erp]: ./example/erp
[./example/syntax]: ./example/syntax

# Features

-   **JSON struct tags**: All `<query_name>Row` structs include JSON struct tags
    using the Postgres column name. To change the struct tag, use a column 
    alias.
  
    ```sql
    -- name: FindAuthors :many
    SELECT author_id, first_name, last_name as family_name FROM author;
    ```
    
    Generates:
    
    ```go
    type FindAuthorsRow struct {
        AuthorID  int32  `json:"author_id"`
        FirstName string `json:"first_name"`
        LastName  string `json:"family_name"`
    }
    ```

-   **Acronyms**: Custom acronym support with `--acronym` flag. With a query 
    like `SELECT 10 as order_mrr`, the flag `--acronym mrr` generates the name 
    `OrderMRR` instead of `OrderMrr`.

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
SELECT * FROM author WHERE first_name = pggen.arg('FirstName');
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
    query by itself, and two methods to support batching a query with 
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
        FindAuthorsBatch(ctx context.Context, batch *pgx.Batch, firstName string)
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
	q.FindAuthorsBatch(context.Background(), batch, "alice")
	q.FindAuthorsBatch(context.Background(), batch, "bob")
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
    `pggen.arg('FirstName')`.

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
    
    pggen infers struct field types by running the query. When Postgres returns
    query results, Postgres also sends the column types as a header for the 
    results. pggen looks up the types in the header using the `pg_type` catalog 
    table and chooses an appropriate Go type in 
    [internal/codegen/golang/types.go].
    
    Choosing an appropriate type is more difficult than might seem at first 
    glance due to `null`. When Postgres reports that a column has a type `text`,
    that column can have  both `text` and `null` values. So, the Postgres `text`
    represented in Go can be either a `string` or `nil`. [`pgtype`] provides 
    nullable types for all built-in Postgres types. pggen tries to infer if a 
    column is nullable or non-nullable. If a column is nullable, pggen uses a 
    `pgtype` Go type like `pgtype.Text`. If a column is non-nullable, pggen uses
     a more ergonomic type like `string`. pggen's nullability inference in 
    [internal/pginfer/nullability.go] is rudimentary; a proper approach requires
     a full AST with some control flow analysis.
    
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
[internal/codegen/golang/types.go]: ./internal/codegen/golang/types.go
[`pgtype`]: https://pkg.go.dev/github.com/jackc/pgtype
[internal/pginfer/nullability.go]: ./internal/pginfer/nullability.go

# How it works

In a nutshell, pggen runs each query on Postgres to extract type information, 
and generates the appropriate code. In detail:

- pggen determines input parameters by using a `PREPARE` statement and querying
  the [`pg_prepared_statement`] table to get type information for each 
  parameter.
  
- pggen determines output columns by executing the query and reading the field
  descriptions returned with the query result rows. The field descriptions 
  contain the type ID for each output column. The type ID is a Postgres object 
  ID (OID), the primary key to identify a row in the [`pg_type`] catalog table.

- pggen determines if an output column can be null using heuristics. If a column
  cannot be null, pggen uses more ergonomic types to represent the output like
  `string` instead of `pgtype.Text`. The heuristics are quite simple; see
  [internal/pginfer/nullability.go]. A proper approach requires a full Postgres 
  SQL syntax parser with control flow analysis to determine nullability.
   
For more detail, see the original, outdated [design doc] and discussion with the 
[pgx author] and [sqlc author].

[`pg_prepared_statement`]: https://www.postgresql.org/docs/current/view-pg-prepared-statements.html
[`pg_type`]: https://www.postgresql.org/docs/13/catalog-pg-type.html
[design doc]: https://docs.google.com/document/d/1NvVKD6cyXvJLWUfqFYad76CWMDFoK9mzKuj1JawkL2A/edit#
[pgx author]: https://github.com/jackc/pgx/issues/915
[sqlc author]: https://github.com/kyleconroy/sqlc/issues/854

# Comparison to sqlc

The primary difference between pggen and sqlc is how each tool infers the type
and nullability of the input parameters and output columns for SQL queries.

sqlc parses the queries in Go code, using Cgo to call the Postgres `parser.c` 
library. After parsing, sqlc infers the types of the query parameters and result
columns using custom logic in Go. In contrast, pggen gets the same type 
information by running the queries on Postgres and then fetching the type 
information for Postgres catalog tables. 

Use sqlc if you don't wish to run Postgres to generate code or if you need
better nullability analysis than pggen provides.

Use pggen if you can run Postgres for code generation and you use complex 
queries that sqlc is unable to parse. Additionally, use pggen if you have a 
custom database setup that's difficult to replicate in a schema file. pggen
supports running on any database with any extensions.
