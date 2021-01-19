package sqld

import (
	"context"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

// Unique type to prevent assignment.
type clientEventContextKey struct{}

// ContextClientTrace returns the ClientTrace associated with the
// provided context. If none, it returns nil.
func ContextClientTrace(ctx context.Context) *ClientTrace {
	trace, _ := ctx.Value(clientEventContextKey{}).(*ClientTrace)
	return trace
}

// WithClientTrace returns a new context based on the provided parent ctx.
// pgx client requests made with the returned context will use the provided
// trace hooks, in addition to any previous hooks registered with ctx. Any hooks
// defined in the provided trace will be called first.
func WithClientTrace(ctx context.Context, trace *ClientTrace) context.Context {
	if trace == nil {
		panic("nil trace")
	}
	old := ContextClientTrace(ctx)
	trace.compose(old)

	return context.WithValue(ctx, clientEventContextKey{}, trace)
}

// ClientTrace is a set of hooks to run at various stages of an outgoing
// Postgres query. Any particular hook may be nil. Functions may be called
// concurrently from different goroutines and some may be called after the
// request has completed or failed.
//
// Inspired by httptrace.ClientTrace.
type ClientTrace struct {
	// EnqueueQuery is called before queueing a query into a pgx.Batch.
	// EnqueueQuery is only called for batch queries.
	EnqueueQuery func(sql string)
	// SendQuery is called before the query is written into the underlying
	// transport. Called before pgx.Conn Query, Exec, or SendBatch methods.
	SendQuery func(config *pgx.ConnConfig, sql string)
	// GotResponse is called after pgx parsed the response into pgx.Rows for
	// select statements or pgconn.CommandTag for modify statements (insert,
	// update, delete)
	GotResponse func(pgx.Rows, pgconn.CommandTag, error)
	// ScanResponse is called after all rows have been scanned or if an error
	// occurs while scanning the response.
	ScanResponse func(error)
}

// compose modifies t such that it respects the previously-registered hooks in
// old.
func (t *ClientTrace) compose(old *ClientTrace) {
	if old == nil {
		return
	}

	if old.EnqueueQuery != nil {
		if t.EnqueueQuery == nil {
			t.EnqueueQuery = old.EnqueueQuery
		} else {
			cur := t.EnqueueQuery
			t.EnqueueQuery = func(sql string) { cur(sql); old.EnqueueQuery(sql) }
		}
	}

	if old.SendQuery != nil {
		if t.SendQuery == nil {
			t.SendQuery = old.SendQuery
		} else {
			cur := t.SendQuery
			t.SendQuery = func(c *pgx.ConnConfig, sql string) { cur(c, sql); old.SendQuery(c, sql) }
		}
	}

	if old.GotResponse != nil {
		if t.GotResponse == nil {
			t.GotResponse = old.GotResponse
		} else {
			cur := t.GotResponse
			t.GotResponse = func(r pgx.Rows, t pgconn.CommandTag, err error) {
				cur(r, t, err)
				old.GotResponse(r, t, err)
			}
		}
	}

	if old.ScanResponse != nil {
		if t.ScanResponse == nil {
			t.ScanResponse = old.ScanResponse
		} else {
			cur := t.ScanResponse
			t.ScanResponse = func(err error) { cur(err); old.ScanResponse(err) }
		}
	}
}
