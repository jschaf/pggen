// Code generated by pggen. DO NOT EDIT.

package void

import (
	"context"
	"fmt"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
)

// Querier is a typesafe Go interface backed by SQL queries.
type Querier interface {
	VoidOnly(ctx context.Context) (pgconn.CommandTag, error)

	VoidOnlyTwoParams(ctx context.Context, id int32) (pgconn.CommandTag, error)

	VoidTwo(ctx context.Context) (string, error)

	VoidThree(ctx context.Context) (VoidThreeRow, error)

	VoidThree2(ctx context.Context) ([]string, error)
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

const voidOnlySQL = `SELECT void_fn();`

// VoidOnly implements Querier.VoidOnly.
func (q *DBQuerier) VoidOnly(ctx context.Context) (pgconn.CommandTag, error) {
	ctx = context.WithValue(ctx, "pggen_query_name", "VoidOnly")
	cmdTag, err := q.conn.Exec(ctx, voidOnlySQL)
	if err != nil {
		return cmdTag, fmt.Errorf("exec query VoidOnly: %w", err)
	}
	return cmdTag, err
}

const voidOnlyTwoParamsSQL = `SELECT void_fn_two_params($1, 'text');`

// VoidOnlyTwoParams implements Querier.VoidOnlyTwoParams.
func (q *DBQuerier) VoidOnlyTwoParams(ctx context.Context, id int32) (pgconn.CommandTag, error) {
	ctx = context.WithValue(ctx, "pggen_query_name", "VoidOnlyTwoParams")
	cmdTag, err := q.conn.Exec(ctx, voidOnlyTwoParamsSQL, id)
	if err != nil {
		return cmdTag, fmt.Errorf("exec query VoidOnlyTwoParams: %w", err)
	}
	return cmdTag, err
}

const voidTwoSQL = `SELECT void_fn(), 'foo' as name;`

// VoidTwo implements Querier.VoidTwo.
func (q *DBQuerier) VoidTwo(ctx context.Context) (string, error) {
	ctx = context.WithValue(ctx, "pggen_query_name", "VoidTwo")
	row := q.conn.QueryRow(ctx, voidTwoSQL)
	var item string
	if err := row.Scan(nil, &item); err != nil {
		return item, fmt.Errorf("query VoidTwo: %w", err)
	}
	return item, nil
}

const voidThreeSQL = `SELECT void_fn(), 'foo' as foo, 'bar' as bar;`

type VoidThreeRow struct {
	Foo string `json:"foo"`
	Bar string `json:"bar"`
}

// VoidThree implements Querier.VoidThree.
func (q *DBQuerier) VoidThree(ctx context.Context) (VoidThreeRow, error) {
	ctx = context.WithValue(ctx, "pggen_query_name", "VoidThree")
	row := q.conn.QueryRow(ctx, voidThreeSQL)
	var item VoidThreeRow
	if err := row.Scan(nil, &item.Foo, &item.Bar); err != nil {
		return item, fmt.Errorf("query VoidThree: %w", err)
	}
	return item, nil
}

const voidThree2SQL = `SELECT 'foo' as foo, void_fn(), void_fn();`

// VoidThree2 implements Querier.VoidThree2.
func (q *DBQuerier) VoidThree2(ctx context.Context) ([]string, error) {
	ctx = context.WithValue(ctx, "pggen_query_name", "VoidThree2")
	rows, err := q.conn.Query(ctx, voidThree2SQL)
	if err != nil {
		return nil, fmt.Errorf("query VoidThree2: %w", err)
	}
	defer rows.Close()
	items := []string{}
	for rows.Next() {
		var item string
		if err := rows.Scan(&item, nil, nil); err != nil {
			return nil, fmt.Errorf("scan VoidThree2 row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("close VoidThree2 rows: %w", err)
	}
	return items, err
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
