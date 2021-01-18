package author

import (
	"context"
	"fmt"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

const findAuthorsName = "FindAuthors"
const findAuthorsSQL = `SELECT first_name, last_name FROM author WHERE first_name = $1`

// FindAuthors implements Querier.FindAuthors.
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

// FindAuthorsBatch implements Querier.FindAuthorsBatch.
func (q *DBQuerier) FindAuthorsBatch(ctx context.Context, batch pgx.Batch, firstName string) {
	ctx = q.hooks.BeforeQuery(ctx, BeforeHookParams{QueryName: findAuthorsName, SQL: findAuthorsSQL, IsBatch: true})
	batch.Queue(findAuthorsSQL, firstName)
}

// FindAuthorsScan implements Querier.FindAuthorsScan.
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

// DeleteAuthors implements Querier.DeleteAuthors.
func (q *DBQuerier) DeleteAuthors(ctx context.Context) (pgconn.CommandTag, error) {
	ctx = q.hooks.BeforeQuery(ctx, BeforeHookParams{QueryName: deleteAuthorsName, SQL: deleteAuthorsSQL, IsBatch: false})
	cmdTag, err := q.conn.Exec(ctx, deleteAuthorsSQL)
	q.hooks.AfterQuery(ctx, AfterHookParams{
		QueryName: findAuthorsName, SQL: findAuthorsSQL, IsBatch: false,
		CommandTag: cmdTag, QueryErr: err})
	return cmdTag, err
}

// DeleteAuthorsBatch implements Querier.DeleteAuthorsBatch.
func (q *DBQuerier) DeleteAuthorsBatch(ctx context.Context, batch *pgx.Batch) {
	ctx = q.hooks.BeforeQuery(ctx, BeforeHookParams{QueryName: deleteAuthorsName, SQL: deleteAuthorsSQL, IsBatch: true})
	batch.Queue(deleteAuthorsSQL)
}

// DeleteAuthorsScan implements Querier.DeleteAuthorsScan.
func (q *DBQuerier) DeleteAuthorsScan(ctx context.Context, results pgx.BatchResults) (pgconn.CommandTag, error) {
	cmdTag, err := results.Exec()
	q.hooks.AfterQuery(ctx, AfterHookParams{
		QueryName: findAuthorsName, SQL: findAuthorsSQL, IsBatch: false,
		CommandTag: cmdTag, QueryErr: err})
	return cmdTag, err
}
