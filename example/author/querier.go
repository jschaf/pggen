package author

import (
	"context"
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
