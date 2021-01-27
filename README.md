![Tests](https://github.com/jschaf/pggen/workflows/Tests/badge.svg)

# pggen - generate type safe Go from Postgres SQL

pggen is a binary that generates Go code to provide a typesafe wrapper around
Postgres queries. pggen has the same goals as [sqlc], a similar tool that 
compiles SQL to type-safe Go. 

1. Write SQL queries.
2. Run pggen to generate a Go code the provides a type-safe interface to running
   the SQL queries.
3. Use the generated code in your application.   

[sqlc]: https://github.com/kyleconroy/sqlc

# Install

```bash
go get github.com/jschaf/pggen
```

# Examples

Examples embedded in the repo:

- [./example/author] - A single table schema with simple queries.

[./example/author]: ./example/author

### Tutorial

Let's say we have a database with the following schema:

```sql
CREATE TABLE author (
  author_id  serial PRIMARY KEY,
  first_name text NOT NULL,
  last_name  text NOT NULL,
  suffix text NULL
)
```

First, write a query in the file `author/queries.sql`:

```sql
-- FindAuthors finds authors by first name.
-- name: FindAuthors :many
SELECT * FROM author WHERE first_name = pggen.arg('FirstName');
```

Second, use pggen to generate the following Go code to `author/queries.sql.go`:

```bash
pggen gen go --query-file author/queries.sql --postgres-connection "user=postgres port=5555 dbname=pggen"
```

The generated file `author/queries.sql.go` looks like:

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
	FindAuthorsScan(ctx context.Context, results pgx.BatchResults) ([]FindAuthorsRow, error)
}

const findAuthorsSQL = `SELECT * FROM author WHERE first_name = $1;`

type FindAuthorsRow struct {
	AuthorID  int32
	FirstName string
	LastName  string
	Suffix    pgtype.Text
}

// FindAuthors implements Querier.FindAuthors.
func (q *DBQuerier) FindAuthors(ctx context.Context, firstName string) ([]FindAuthorsRow, error) {
	rows, err := q.conn.Query(ctx, findAuthorsSQL, firstName)
	if rows != nil {
		defer rows.Close()
	}
	if err != nil {
		return nil, fmt.Errorf("query FindAuthors: %w", err)
	}
	var items []FindAuthorsRow
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

# How it works

In a nutshell, pggen runs each query on Postgres to extract type information, 
and generates the appropriate code. In detail:

- pggen determines input parameters by using a `PREPARE` statement and querying
  the `pg_prepared_statement` table to get type information for each parameter.
  
- pggen determines output columns by executing the query and reading the field
  descriptions returned with the rows of data. The field descriptions contain
  the type ID for each output column. The type ID is a Postgres object ID
  (OID), the primary key to identify a row in the `pg_type` catalog table.

- pggen determines if an output column can be null using heuristics. If a column
  cannot be null, pggen uses more ergonomic types to represent the output like
  `string` instead of `pgtype.Text`. The heuristics are quite simple, see
  [nullability.go]. A proper approach requires a full Postgres SQL syntax parser
   with control flow analysis to determine nullability.
   
For more detail, see the original, slightly outdated [design doc] and discussion
with the [pgx author] and [sqlc author].

[nullability.go]: https://github.com/jschaf/pggen/blob/main/internal/pginfer/nullability.go
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
