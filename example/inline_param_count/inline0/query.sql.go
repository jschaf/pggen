// Code generated by pggen. DO NOT EDIT.

package inline0

import (
	"context"
	"fmt"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
)

// Querier is a typesafe Go interface backed by SQL queries.
type Querier interface {
	// CountAuthors returns the number of authors (zero params).
	CountAuthors(ctx context.Context) (*int, error)

	// FindAuthorById finds one (or zero) authors by ID (one param).
	FindAuthorByID(ctx context.Context, params FindAuthorByIDParams) (FindAuthorByIDRow, error)

	// InsertAuthor inserts an author by name and returns the ID (two params).
	InsertAuthor(ctx context.Context, params InsertAuthorParams) (int32, error)

	// DeleteAuthorsByFullName deletes authors by the full name (three params).
	DeleteAuthorsByFullName(ctx context.Context, params DeleteAuthorsByFullNameParams) (pgconn.CommandTag, error)
}

type DBQuerier struct {
	conn  genericConn   // underlying Postgres transport to use
	types *typeResolver // resolve types by name
}

var _ Querier = &DBQuerier{}

// genericConn is a connection to a Postgres database. This is usually backed by
// *pgx.Conn, pgx.Tx, or *pgxpool.Pool.
type genericConn interface {
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

// NewQuerier creates a DBQuerier that implements Querier. conn is typically
// *pgx.Conn, pgx.Tx, or *pgxpool.Pool.
func NewQuerier(conn genericConn) *DBQuerier {
	return NewQuerierConfig(conn, QuerierConfig{})
}

type QuerierConfig struct {
	// DataTypes contains pgtype.Value to use for encoding and decoding instead
	// of pggen-generated pgtype.ValueTranscoder.
	//
	// If OIDs are available for an input parameter type and all of its
	// transitive dependencies, pggen will use the binary encoding format for
	// the input parameter.
	DataTypes []pgtype.DataType
}

// NewQuerierConfig creates a DBQuerier that implements Querier with the given
// config. conn is typically *pgx.Conn, pgx.Tx, or *pgxpool.Pool.
func NewQuerierConfig(conn genericConn, cfg QuerierConfig) *DBQuerier {
	return &DBQuerier{conn: conn, types: newTypeResolver(cfg.DataTypes)}
}

// WithTx creates a new DBQuerier that uses the transaction to run all queries.
func (q *DBQuerier) WithTx(tx pgx.Tx) (*DBQuerier, error) {
	return &DBQuerier{conn: tx}, nil
}

// typeResolver looks up the pgtype.ValueTranscoder by Postgres type name.
type typeResolver struct {
	connInfo *pgtype.ConnInfo // types by Postgres type name
}

func newTypeResolver(types []pgtype.DataType) *typeResolver {
	ci := pgtype.NewConnInfo()
	for _, typ := range types {
		if txt, ok := typ.Value.(textPreferrer); ok && typ.OID != unknownOID {
			typ.Value = txt.ValueTranscoder
		}
		ci.RegisterDataType(typ)
	}
	return &typeResolver{connInfo: ci}
}

// findValue find the OID, and pgtype.ValueTranscoder for a Postgres type name.
func (tr *typeResolver) findValue(name string) (uint32, pgtype.ValueTranscoder, bool) {
	typ, ok := tr.connInfo.DataTypeForName(name)
	if !ok {
		return 0, nil, false
	}
	v := pgtype.NewValue(typ.Value)
	return typ.OID, v.(pgtype.ValueTranscoder), true
}

// setValue sets the value of a ValueTranscoder to a value that should always
// work and panics if it fails.
func (tr *typeResolver) setValue(vt pgtype.ValueTranscoder, val interface{}) pgtype.ValueTranscoder {
	if err := vt.Set(val); err != nil {
		panic(fmt.Sprintf("set ValueTranscoder %T to %+v: %s", vt, val, err))
	}
	return vt
}

const countAuthorsSQL = `SELECT count(*) FROM author;`

// CountAuthors implements Querier.CountAuthors.
func (q *DBQuerier) CountAuthors(ctx context.Context) (*int, error) {
	ctx = context.WithValue(ctx, "pggen_query_name", "CountAuthors")
	row := q.conn.QueryRow(ctx, countAuthorsSQL)
	var item *int
	if err := row.Scan(&item); err != nil {
		return item, fmt.Errorf("query CountAuthors: %w", err)
	}
	return item, nil
}

const findAuthorByIDSQL = `SELECT * FROM author WHERE author_id = $1;`

type FindAuthorByIDParams struct {
	AuthorID int32 `json:"AuthorID"`
}

type FindAuthorByIDRow struct {
	AuthorID  int32   `json:"author_id"`
	FirstName string  `json:"first_name"`
	LastName  string  `json:"last_name"`
	Suffix    *string `json:"suffix"`
}

// FindAuthorByID implements Querier.FindAuthorByID.
func (q *DBQuerier) FindAuthorByID(ctx context.Context, params FindAuthorByIDParams) (FindAuthorByIDRow, error) {
	ctx = context.WithValue(ctx, "pggen_query_name", "FindAuthorByID")
	row := q.conn.QueryRow(ctx, findAuthorByIDSQL, params.AuthorID)
	var item FindAuthorByIDRow
	if err := row.Scan(&item.AuthorID, &item.FirstName, &item.LastName, &item.Suffix); err != nil {
		return item, fmt.Errorf("query FindAuthorByID: %w", err)
	}
	return item, nil
}

const insertAuthorSQL = `INSERT INTO author (first_name, last_name)
VALUES ($1, $2)
RETURNING author_id;`

type InsertAuthorParams struct {
	FirstName string `json:"FirstName"`
	LastName  string `json:"LastName"`
}

// InsertAuthor implements Querier.InsertAuthor.
func (q *DBQuerier) InsertAuthor(ctx context.Context, params InsertAuthorParams) (int32, error) {
	ctx = context.WithValue(ctx, "pggen_query_name", "InsertAuthor")
	row := q.conn.QueryRow(ctx, insertAuthorSQL, params.FirstName, params.LastName)
	var item int32
	if err := row.Scan(&item); err != nil {
		return item, fmt.Errorf("query InsertAuthor: %w", err)
	}
	return item, nil
}

const deleteAuthorsByFullNameSQL = `DELETE
FROM author
WHERE first_name = $1
  AND last_name = $2
  AND CASE WHEN $3 = '' THEN suffix IS NULL ELSE suffix = $3 END;`

type DeleteAuthorsByFullNameParams struct {
	FirstName string `json:"FirstName"`
	LastName  string `json:"LastName"`
	Suffix    string `json:"Suffix"`
}

// DeleteAuthorsByFullName implements Querier.DeleteAuthorsByFullName.
func (q *DBQuerier) DeleteAuthorsByFullName(ctx context.Context, params DeleteAuthorsByFullNameParams) (pgconn.CommandTag, error) {
	ctx = context.WithValue(ctx, "pggen_query_name", "DeleteAuthorsByFullName")
	cmdTag, err := q.conn.Exec(ctx, deleteAuthorsByFullNameSQL, params.FirstName, params.LastName, params.Suffix)
	if err != nil {
		return cmdTag, fmt.Errorf("exec query DeleteAuthorsByFullName: %w", err)
	}
	return cmdTag, err
}

// textPreferrer wraps a pgtype.ValueTranscoder and sets the preferred encoding
// format to text instead binary (the default). pggen uses the text format
// when the OID is unknownOID because the binary format requires the OID.
// Typically occurs if the results from QueryAllDataTypes aren't passed to
// NewQuerierConfig.
type textPreferrer struct {
	pgtype.ValueTranscoder
	typeName string
}

// PreferredParamFormat implements pgtype.ParamFormatPreferrer.
func (t textPreferrer) PreferredParamFormat() int16 { return pgtype.TextFormatCode }

func (t textPreferrer) NewTypeValue() pgtype.Value {
	return textPreferrer{ValueTranscoder: pgtype.NewValue(t.ValueTranscoder).(pgtype.ValueTranscoder), typeName: t.typeName}
}

func (t textPreferrer) TypeName() string {
	return t.typeName
}

// unknownOID means we don't know the OID for a type. This is okay for decoding
// because pgx call DecodeText or DecodeBinary without requiring the OID. For
// encoding parameters, pggen uses textPreferrer if the OID is unknown.
const unknownOID = 0
