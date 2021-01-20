package author

import (
	"context"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/jschaf/sqld"
)

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

type DBQuerier struct {
	conn sqld.Conn
}

var _ Querier = &DBQuerier{}

// NewQuerier creates a DBQuerier that implements Querier. conn is typically
// *pgx.Conn, pgx.Tx, or *pgxpool.Pool.
func NewQuerier(conn sqld.Conn) *DBQuerier {
	return &DBQuerier{
		conn: conn,
	}
}

// WithTx creates a new DBQuerier that uses the transaction to run all queries.
func (q *DBQuerier) WithTx(tx pgx.Tx) (*DBQuerier, error) {
	return &DBQuerier{conn: tx}, nil
}

func traceEnqueueQuery(trace *sqld.ClientTrace, sql string) {
	if trace != nil && trace.EnqueueQuery != nil {
		trace.EnqueueQuery(sql)
	}
}

func traceSendQuery(trace *sqld.ClientTrace, config *pgx.ConnConfig, sql string) {
	if trace != nil && trace.SendQuery != nil {
		trace.SendQuery(config, sql)
	}
}

func traceGotResponse(trace *sqld.ClientTrace, r pgx.Rows, tag pgconn.CommandTag, err error) {
	if trace != nil && trace.SendQuery != nil {
		trace.GotResponse(r, tag, err)
	}
}

func traceScanResponse(trace *sqld.ClientTrace, err error) {
	if trace != nil && trace.ScanResponse != nil {
		trace.ScanResponse(err)
	}
}

func extractConfig(conn sqld.Conn) *pgx.ConnConfig {
	switch c := conn.(type) {
	case *pgx.Conn:
		return c.Config()
	case *pgxpool.Pool:
		return c.Config().ConnConfig
	case pgx.Tx:
		return c.Conn().Config()
	default:
		return nil
	}
}
