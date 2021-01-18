package main

import (
	"context"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

type Author struct {
	FirstName string
	LastName  string
}
type AuthorID int

type Querier interface {
	FindAuthors(ctx context.Context, id AuthorID) ([]Author, error)
	DeleteAuthors(ctx context.Context) (pgconn.CommandTag, error)
}

// BatchQuerier provides the batch interface to Querier. Methods ending with
// Batch enqueue a query to run later. After calling SendBatch use the Scan
// methods to parse the results.
type BatchQuerier interface {
	// FindAuthorsBatch enqueues a Querier.FindAuthors query into batch to be
	// executed later by the batch.
	FindAuthorsBatch(ctx context.Context, batch pgx.Batch, id AuthorID)
	// FindAuthorsScan scans the result of an executed FindAuthorsBatch query.
	FindAuthorsScan(ctx context.Context, results pgx.BatchResults) ([]Author, error)

	// DeleteAuthorsBatch enqueues a Querier.DeleteAuthors query into batch to be
	// executed later by the batch.
	DeleteAuthorsBatch(ctx context.Context, batch *pgx.Batch)
	// DeleteAuthorsScan scans the result of an executed DeleteAuthorsBatch query.
	DeleteAuthorsScan(ctx context.Context, results pgx.BatchResults) (pgconn.CommandTag, error)
}

// Conn is a connection to a Postgres database. This is usually backed by
// *pgx.Conn, pgx.Tx, or *pgxpool.Pool.
type Conn interface {
	// Query executes sql with args. If there is an error the returned Rows will
	// be returned in an error state. So it is allowed to ignore the error
	// returned from Query and handle it in Rows.
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)

	// QueryRow is a convenience wrapper over Query. Any error that occurs while
	// querying is deferred until calling Scan on the returned Row. That Row will
	// error with pgx.ErrNoRows if no rows are returned.
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row

	// Exec executes sql. sql can be either a prepared statement name or an SQL
	// string. arguments should be referenced positionally from the sql string
	// as $1, $2, etc.
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
}

type DBQuerier struct {
	conn  Conn
	hooks QuerierHook
}

// NewQuerier creates a DBQuerier that implements Querier and BatchQuerier.
func NewQuerier(conn Conn, hooks QuerierHook) *DBQuerier {
	return &DBQuerier{
		conn:  conn,
		hooks: hooks,
	}
}

// WithTx creates a new DBQuerier that uses the transaction to run all queries.
func (q *DBQuerier) WithTx(tx pgx.Tx) (*DBQuerier, error) {
	return &DBQuerier{conn: tx, hooks: q.hooks}, nil
}

const findAuthorsName = "find_authors"
const findAuthorsSQL = "SELECT * from authors"

func (q *DBQuerier) FindAuthors(ctx context.Context, id AuthorID) ([]Author, error) {
	sql := findAuthorsSQL // sql may be modified by BeforeQuery
	ctx = q.hooks.BeforeQuery(ctx, BeforeHookParams{QueryName: findAuthorsName, SQL: &sql, IsBatch: false})
	rows, err := q.conn.Query(ctx, sql, id) // err may be modified by AfterQuery
	cmdTag := pgconn.CommandTag{}
	if rows != nil {
		cmdTag = rows.CommandTag()
		defer rows.Close()
	}
	q.hooks.AfterQuery(ctx, AfterHookParams{
		QueryName: findAuthorsName, SQL: findAuthorsSQL, IsBatch: false, Rows: rows,
		CommandTag: cmdTag, QueryErr: &err})
	if err != nil {
		return nil, err
	}
	var items []Author
	for rows.Next() {
		var item Author
		if err := rows.Scan(&item.FirstName, &item.LastName); err != nil {
			items = append(items, item)
		}
	}
	// TODO: when and why would rows.Err get called?
	if rows.Err() != nil {
		return nil, err
	}
	return items, err
}

func (q *DBQuerier) FindAuthorsBatch(ctx context.Context, batch pgx.Batch, id AuthorID) {
	sql := findAuthorsSQL // sql may be modified by BeforeQuery
	ctx = q.hooks.BeforeQuery(ctx, BeforeHookParams{QueryName: findAuthorsName, SQL: &sql, IsBatch: true})
	batch.Queue(sql, id)
}

func (q *DBQuerier) FindAuthorsScan(ctx context.Context, results pgx.BatchResults) ([]Author, error) {
	rows, err := results.Query() // err may be modified by AfterQuery
	cmdTag := pgconn.CommandTag{}
	if rows != nil {
		cmdTag = rows.CommandTag()
		defer rows.Close()
	}
	q.hooks.AfterQuery(ctx, AfterHookParams{
		QueryName: findAuthorsName, SQL: findAuthorsSQL, IsBatch: false, Rows: rows,
		CommandTag: cmdTag, QueryErr: &err})
	if err != nil {
		return nil, err
	}
	var items []Author
	for rows.Next() {
		var item Author
		if err := rows.Scan(&item.FirstName, &item.LastName); err != nil {
			items = append(items, item)
		}
	}
	// TODO: when and why would rows.Err get called?
	if rows.Err() != nil {
		return nil, err
	}
	return items, err
}

const deleteAuthorsName = "delete_authors"
const deleteAuthorsSQL = "DELETE FROM authors where name = 'foo'"

func (q *DBQuerier) DeleteAuthors(ctx context.Context) (pgconn.CommandTag, error) {
	sql := deleteAuthorsSQL // sql may be modified by BeforeQuery
	ctx = q.hooks.BeforeQuery(ctx, BeforeHookParams{QueryName: deleteAuthorsName, SQL: &sql, IsBatch: false})
	cmdTag, err := q.conn.Exec(ctx, sql) // err may be modified by AfterQuery
	q.hooks.AfterQuery(ctx, AfterHookParams{
		QueryName: findAuthorsName, SQL: findAuthorsSQL, IsBatch: false,
		CommandTag: cmdTag, QueryErr: &err})
	return cmdTag, err
}

func (q *DBQuerier) DeleteAuthorsBatch(ctx context.Context, batch *pgx.Batch) {
	sql := deleteAuthorsSQL // sql may be modified by BeforeQuery
	ctx = q.hooks.BeforeQuery(ctx, BeforeHookParams{QueryName: deleteAuthorsName, SQL: &sql, IsBatch: true})
	batch.Queue(sql)
}

func (q *DBQuerier) DeleteAuthorsScan(ctx context.Context, results pgx.BatchResults) (pgconn.CommandTag, error) {
	cmdTag, err := results.Exec() // err may be modified by AfterQuery
	q.hooks.AfterQuery(ctx, AfterHookParams{
		QueryName: findAuthorsName, SQL: findAuthorsSQL, IsBatch: false,
		CommandTag: cmdTag, QueryErr: &err})
	return cmdTag, err
}
