// Code generated by pggen. DO NOT EDIT.

package slices

import (
	"context"
	"fmt"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"time"
)

// Querier is a typesafe Go interface backed by SQL queries.
type Querier interface {
	GetBools(ctx context.Context, data []bool) ([]bool, error)

	GetOneTimestamp(ctx context.Context, data *time.Time) (*time.Time, error)

	GetManyTimestamptzs(ctx context.Context, data []time.Time) ([]*time.Time, error)

	GetManyTimestamps(ctx context.Context, data []*time.Time) ([]*time.Time, error)
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
	return &DBQuerier{conn: conn, types: newTypeResolver()}
}

// WithTx creates a new DBQuerier that uses the transaction to run all queries.
func (q *DBQuerier) WithTx(tx pgx.Tx) (*DBQuerier, error) {
	return &DBQuerier{conn: tx}, nil
}

// typeResolver looks up the pgtype.ValueTranscoder by Postgres type name.
type typeResolver struct {
	connInfo *pgtype.ConnInfo // types by Postgres type name
}

func newTypeResolver() *typeResolver {
	ci := pgtype.NewConnInfo()
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

type compositeField struct {
	name       string                 // name of the field
	typeName   string                 // Postgres type name
	defaultVal pgtype.ValueTranscoder // default value to use
}

func (tr *typeResolver) newCompositeValue(name string, fields ...compositeField) pgtype.ValueTranscoder {
	if _, val, ok := tr.findValue(name); ok {
		return val
	}
	fs := make([]pgtype.CompositeTypeField, len(fields))
	vals := make([]pgtype.ValueTranscoder, len(fields))
	isBinaryOk := true
	for i, field := range fields {
		oid, val, ok := tr.findValue(field.typeName)
		if !ok {
			oid = unknownOID
			val = field.defaultVal
		}
		isBinaryOk = isBinaryOk && oid != unknownOID
		fs[i] = pgtype.CompositeTypeField{Name: field.name, OID: oid}
		vals[i] = val
	}
	// Okay to ignore error because it's only thrown when the number of field
	// names does not equal the number of ValueTranscoders.
	typ, _ := pgtype.NewCompositeTypeValues(name, fs, vals)
	if !isBinaryOk {
		return textPreferrer{ValueTranscoder: typ, typeName: name}
	}
	return typ
}

func (tr *typeResolver) newArrayValue(name, elemName string, defaultVal func() pgtype.ValueTranscoder) pgtype.ValueTranscoder {
	if _, val, ok := tr.findValue(name); ok {
		return val
	}
	elemOID, elemVal, ok := tr.findValue(elemName)
	elemValFunc := func() pgtype.ValueTranscoder {
		return pgtype.NewValue(elemVal).(pgtype.ValueTranscoder)
	}
	if !ok {
		elemOID = unknownOID
		elemValFunc = defaultVal
	}
	typ := pgtype.NewArrayType(name, elemOID, elemValFunc)
	if elemOID == unknownOID {
		return textPreferrer{ValueTranscoder: typ, typeName: name}
	}
	return typ
}

// newboolArrayRaw returns all elements for the Postgres array type '_bool'
// as a slice of interface{} for use with the pgtype.Value Set method.
func (tr *typeResolver) newboolArrayRaw(vs []bool) []interface{} {
	elems := make([]interface{}, len(vs))
	for i, v := range vs {
		elems[i] = v
	}
	return elems
}

const getBoolsSQL = `SELECT $1::boolean[];`

// GetBools implements Querier.GetBools.
func (q *DBQuerier) GetBools(ctx context.Context, data []bool) ([]bool, error) {
	ctx = context.WithValue(ctx, "pggen_query_name", "GetBools")
	row := q.conn.QueryRow(ctx, getBoolsSQL, data)
	item := []bool{}
	if err := row.Scan(&item); err != nil {
		return item, fmt.Errorf("query GetBools: %w", err)
	}
	return item, nil
}

const getOneTimestampSQL = `SELECT $1::timestamp;`

// GetOneTimestamp implements Querier.GetOneTimestamp.
func (q *DBQuerier) GetOneTimestamp(ctx context.Context, data *time.Time) (*time.Time, error) {
	ctx = context.WithValue(ctx, "pggen_query_name", "GetOneTimestamp")
	row := q.conn.QueryRow(ctx, getOneTimestampSQL, data)
	var item *time.Time
	if err := row.Scan(&item); err != nil {
		return item, fmt.Errorf("query GetOneTimestamp: %w", err)
	}
	return item, nil
}

const getManyTimestamptzsSQL = `SELECT *
FROM unnest($1::timestamptz[]);`

// GetManyTimestamptzs implements Querier.GetManyTimestamptzs.
func (q *DBQuerier) GetManyTimestamptzs(ctx context.Context, data []time.Time) ([]*time.Time, error) {
	ctx = context.WithValue(ctx, "pggen_query_name", "GetManyTimestamptzs")
	rows, err := q.conn.Query(ctx, getManyTimestamptzsSQL, data)
	if err != nil {
		return nil, fmt.Errorf("query GetManyTimestamptzs: %w", err)
	}
	defer rows.Close()
	items := []*time.Time{}
	for rows.Next() {
		var item time.Time
		if err := rows.Scan(&item); err != nil {
			return nil, fmt.Errorf("scan GetManyTimestamptzs row: %w", err)
		}
		items = append(items, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("close GetManyTimestamptzs rows: %w", err)
	}
	return items, err
}

const getManyTimestampsSQL = `SELECT *
FROM unnest($1::timestamp[]);`

// GetManyTimestamps implements Querier.GetManyTimestamps.
func (q *DBQuerier) GetManyTimestamps(ctx context.Context, data []*time.Time) ([]*time.Time, error) {
	ctx = context.WithValue(ctx, "pggen_query_name", "GetManyTimestamps")
	rows, err := q.conn.Query(ctx, getManyTimestampsSQL, data)
	if err != nil {
		return nil, fmt.Errorf("query GetManyTimestamps: %w", err)
	}
	defer rows.Close()
	items := []*time.Time{}
	for rows.Next() {
		var item time.Time
		if err := rows.Scan(&item); err != nil {
			return nil, fmt.Errorf("scan GetManyTimestamps row: %w", err)
		}
		items = append(items, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("close GetManyTimestamps rows: %w", err)
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
