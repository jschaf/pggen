# pggen - Experiment to generate type safe Go from Postgres SQL

This is very much a work in progress.

**[Design Doc]**: comments welcome.

[Design Doc]: https://docs.google.com/document/d/1NvVKD6cyXvJLWUfqFYad76CWMDFoK9mzKuj1JawkL2A/edit#

See the design doc for a complete overview. To summarize:

pggen is a binary that generates Go code that provides a typesafe wrapper to 
Postgres queries. pggen has the same goals as sqlc, to "compile SQL to type-safe 
Go". The sqlc documentation provides a concise overview of the benefits of the 
code generation approach:

> sqlc generates fully-type safe idiomatic Go code from SQL.
>
> - You write SQL queries
> - You run sqlc to generate Go code that presents type-safe interfaces to 
>   those queries
> - You write application code that calls the methods sqlc generated.

The primary difference between pggen and sqlc is how pggen generates the Go code. 
sqlc parses the queries in Go code, using Cgo to call the Postgres parser.c 
code. After parsing, sqlc infers the types of the query parameters and result 
columns using custom logic in Go. In contrast, pggen gets the same type 
information by running the queries on Postgres and then fetching the 
type information for Postgres catalog tables. 

## TODO

- [ ] Make author example runnable and testable with Docker scaffolding.
- [ ] Request review from pgx and sqlc authors.
- [ ] Maybe add line comments to link to SQL source.
