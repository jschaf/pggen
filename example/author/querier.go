package author

import (
	"context"
	"fmt"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

type Author struct {
	FirstName string
	LastName  string
}

// Querier is a typesafe Go interface backed by SQL queries.
//
// Methods ending with Batch enqueue a query to run later in a pgx.Batch. After
// calling SendBatch on pgx.Conn, pgxpool.Pool, or pgx.Tx, use the Scan methods
// to parse the results.
type Querier interface {
	// FindAuthors finds authors by first name.
	FindAuthors(ctx context.Context, firstName string) ([]Author, error)
	// FindAuthorsBatch enqueues a FindAuthors query into batch to be executed
	// later by the batch.
	FindAuthorsBatch(ctx context.Context, batch pgx.Batch, firstName string)
	// FindAuthorsScan scans the result of an executed FindAuthorsBatch query.
	FindAuthorsScan(ctx context.Context, results pgx.BatchResults) ([]Author, error)

	// DeleteAuthors deletes authors with a first name of "joe".
	DeleteAuthors(ctx context.Context) (pgconn.CommandTag, error)
	// DeleteAuthorsBatch enqueues a DeleteAuthors query into batch to be executed
	// later by the batch.
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

var _ Querier = &DBQuerier{}

// NewQuerier creates a DBQuerier that implements Querier.
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

const findAuthorsName = "FindAuthors"
const findAuthorsSQL = `SELECT first_name, last_name FROM author WHERE first_name = $1`

func (q *DBQuerier) FindAuthors(ctx context.Context, firstName string) ([]Author, error) {
	ctx = q.hooks.BeforeQuery(ctx, BeforeHookParams{QueryName: findAuthorsName, SQL: findAuthorsSQL, IsBatch: false})
	rows, err := q.conn.Query(ctx, findAuthorsSQL, firstName)
	cmdTag := pgconn.CommandTag{}
	if rows != nil {
		cmdTag = rows.CommandTag()
		defer rows.Close()
	}
	q.hooks.AfterQuery(ctx, AfterHookParams{
		QueryName: findAuthorsName, SQL: findAuthorsSQL, IsBatch: false, Rows: rows,
		CommandTag: cmdTag, QueryErr: err})
	if err != nil {
		return nil, fmt.Errorf("query FindAuthors: %w", err)
	}
	var items []Author
	for rows.Next() {
		var item Author
		if err := rows.Scan(&item.FirstName, &item.LastName); err != nil {
			return nil, fmt.Errorf("scan FindAuthors row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, err
}

func (q *DBQuerier) FindAuthorsBatch(ctx context.Context, batch pgx.Batch, firstName string) {
	ctx = q.hooks.BeforeQuery(ctx, BeforeHookParams{QueryName: findAuthorsName, SQL: findAuthorsSQL, IsBatch: true})
	batch.Queue(findAuthorsSQL, firstName)
}

func (q *DBQuerier) FindAuthorsScan(ctx context.Context, results pgx.BatchResults) ([]Author, error) {
	rows, err := results.Query()
	cmdTag := pgconn.CommandTag{}
	if rows != nil {
		cmdTag = rows.CommandTag()
		defer rows.Close()
	}
	q.hooks.AfterQuery(ctx, AfterHookParams{
		QueryName: findAuthorsName, SQL: findAuthorsSQL, IsBatch: false, Rows: rows,
		CommandTag: cmdTag, QueryErr: err})
	if err != nil {
		return nil, err
	}
	var items []Author
	for rows.Next() {
		var item Author
		if err := rows.Scan(&item.FirstName, &item.LastName); err != nil {
			return nil, fmt.Errorf("scan FindAuthors batch row: %w", err)
		}
		items = append(items, item)
	}
	if rows.Err() != nil {
		return nil, err
	}
	return items, err
}

const deleteAuthorsName = "DeleteAuthors"
const deleteAuthorsSQL = `DELETE FROM author where first_name = 'joe'`

func (q *DBQuerier) DeleteAuthors(ctx context.Context) (pgconn.CommandTag, error) {
	ctx = q.hooks.BeforeQuery(ctx, BeforeHookParams{QueryName: deleteAuthorsName, SQL: deleteAuthorsSQL, IsBatch: false})
	cmdTag, err := q.conn.Exec(ctx, deleteAuthorsSQL)
	q.hooks.AfterQuery(ctx, AfterHookParams{
		QueryName: findAuthorsName, SQL: findAuthorsSQL, IsBatch: false,
		CommandTag: cmdTag, QueryErr: err})
	return cmdTag, err
}

func (q *DBQuerier) DeleteAuthorsBatch(ctx context.Context, batch *pgx.Batch) {
	ctx = q.hooks.BeforeQuery(ctx, BeforeHookParams{QueryName: deleteAuthorsName, SQL: deleteAuthorsSQL, IsBatch: true})
	batch.Queue(deleteAuthorsSQL)
}

func (q *DBQuerier) DeleteAuthorsScan(ctx context.Context, results pgx.BatchResults) (pgconn.CommandTag, error) {
	cmdTag, err := results.Exec()
	q.hooks.AfterQuery(ctx, AfterHookParams{
		QueryName: findAuthorsName, SQL: findAuthorsSQL, IsBatch: false,
		CommandTag: cmdTag, QueryErr: err})
	return cmdTag, err
}
