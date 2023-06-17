# Architecture of pggen

In a nutshell, pggen runs each query on Postgres to extract type information,
and generates the appropriate code. In detail, pggen processes a query file
in the following steps.

1.  Resolve the query files from the `--query-glob` flag and schema files from
    the `--schema-glob` flag in [cmd/pggen/pggen.go]. Pass the normalized 
    options to `pggen.Generate` in [generate.go].
    
2.  Start Postgres by either connecting to the database specified in 
    `--postgres-connection` or by starting a new Dockerized Postgres instance.
    [internal/pgdocker/pgdocker.go] creates and destroys Docker images for 
    pggen.

3.  Parse each query files into an `*ast.File` containing many 
    `*ast.SourceQuery` nodes in [internal/parser/interface.go].

4.  Infer the Postgres types and nullability for the input parameters and output
    columns of an `*ast.SourceQuery` and store the results in 
    `pginfer.TypedQuery` in [internal/pginfer/pginfer.go].
    
    To determine the Postgres types, pggen uses itself to compile the queries
    in [internal/pg/query.sql]. The queries leverage the Postgres prepare 
    command to find the input parameter types.

    pggen determines output columns types and names by preparing the query and
    reading the field descriptions returned with the query result rows. The 
    field descriptions contain the type ID for each output column. The type ID 
    is a Postgres object ID (OID), the primary key to identify a row in the 
    [`pg_type`] catalog table.

    pggen determines if an output column can be null using heuristics. If a
    column cannot be null, pggen uses more ergonomic types to represent the
    output like `string` instead of `pgtype.Text`. The heuristics are quite
    simple; see [internal/pginfer/nullability.go]. A proper approach requires a
    control flow analysis to determine nullability. I've started down that road
    in [pgplan.go](./internal/pgplan/pgplan.go).

5.  Transform each `*ast.File` into `codegen.QueryFile` in [generate.go]
    `parseQueries`.

6.  Use a language-specific code generator to transform `codegen.QueryFile`
    into a `golang.TemplatedFile` like with [internal/codegen/golang/templater.go].

7.  Emit the generated code from `golang.TemplateFile` in
    [internal/codegen/golang/templated_file.go]
    
[cmd/pggen/pggen.go]: cmd/pggen/pggen.go
[internal/parser/interface.go]: internal/parser/interface.go
[internal/pgdocker/pgdocker.go]: internal/pgdocker/pgdocker.go
[internal/pginfer/pginfer.go]: internal/pginfer/pginfer.go
[internal/pg/query.sql]: internal/pg/query.sql
[generate.go]: ./generate.go
[internal/codegen/golang/templater.go]: internal/codegen/golang/templater.go
[internal/codegen/golang/templated_file.go]: internal/codegen/golang/templated_file.go
[`pg_prepared_statement`]: https://www.postgresql.org/docs/current/view-pg-prepared-statements.html
[`pg_type`]: https://www.postgresql.org/docs/13/catalog-pg-type.html

For additional detail, see the original, outdated [design doc] and discussion with the
[pgx author] and [sqlc author].

[design doc]: https://docs.google.com/document/d/1NvVKD6cyXvJLWUfqFYad76CWMDFoK9mzKuj1JawkL2A/edit#
[pgx author]: https://github.com/jackc/pgx/issues/915
[sqlc author]: https://github.com/kyleconroy/sqlc/issues/854
