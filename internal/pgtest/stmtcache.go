package pgtest

import (
	"context"
	"fmt"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgconn/stmtcache"
	"github.com/jackc/pgx/v4"
)

// WithGuardedStmtCache is a functional option to initialize a the pgtest conn
// with a guarded cache that fails if pgx attempts to cache any SQL query in
// names. The names are typically SQL statements.
func WithGuardedStmtCache(names ...string) Option {
	return func(config *pgx.ConnConfig) {
		config.BuildStatementCache = func(conn *pgconn.PgConn) stmtcache.Cache {
			return NewGuardedStmtCache(conn, names...)
		}
	}
}

// GuardedStmtCache errors if any name in names is used to get a cached statement.
// Allows verifying that PrepareAllQueries works by creating prepared statements
// ahead of time. pgx accesses a map of prepared statements directly rather than
// calling Get.
type GuardedStmtCache struct {
	*stmtcache.LRU
	names map[string]struct{}
}

func NewGuardedStmtCache(conn *pgconn.PgConn, names ...string) *GuardedStmtCache {
	size := 1024 // so we never expire
	nameMap := make(map[string]struct{})
	for _, n := range names {
		nameMap[n] = struct{}{}
	}
	return &GuardedStmtCache{
		LRU:   stmtcache.NewLRU(conn, stmtcache.ModePrepare, size),
		names: nameMap,
	}
}

func (sc *GuardedStmtCache) Get(ctx context.Context, sql string) (*pgconn.StatementDescription, error) {
	if _, ok := sc.names[sql]; ok {
		return nil, fmt.Errorf("guard statement cache attempted to get %s;"+
			" should already exist as prepared statement", sql)
	}
	return sc.LRU.Get(ctx, sql)
}
