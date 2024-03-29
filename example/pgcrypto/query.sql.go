// Code generated by pggen. DO NOT EDIT.

package pgcrypto

import (
	"context"
	"fmt"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
)

// Querier is a typesafe Go interface backed by SQL queries.
type Querier interface {
	CreateUser(ctx context.Context, email string, password string) (pgconn.CommandTag, error)

	FindUser(ctx context.Context, email string) (FindUserRow, error)
}

var _ Querier = &DBQuerier{}

type DBQuerier struct {
	conn  genericConn   // underlying Postgres transport to use
	types *typeResolver // resolve types by name
}

// genericConn is a connection like *pgx.Conn, pgx.Tx, or *pgxpool.Pool.
type genericConn interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

// NewQuerier creates a DBQuerier that implements Querier.
func NewQuerier(conn genericConn) *DBQuerier {
	return &DBQuerier{conn: conn, types: newTypeResolver()}
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

const createUserSQL = `INSERT INTO "user" (email, pass)
VALUES ($1, crypt($2, gen_salt('bf')));`

// CreateUser implements Querier.CreateUser.
func (q *DBQuerier) CreateUser(ctx context.Context, email string, password string) (pgconn.CommandTag, error) {
	ctx = context.WithValue(ctx, "pggen_query_name", "CreateUser")
	cmdTag, err := q.conn.Exec(ctx, createUserSQL, email, password)
	if err != nil {
		return cmdTag, fmt.Errorf("exec query CreateUser: %w", err)
	}
	return cmdTag, err
}

const findUserSQL = `SELECT email, pass from "user"
where email = $1;`

type FindUserRow struct {
	Email string `json:"email"`
	Pass  string `json:"pass"`
}

// FindUser implements Querier.FindUser.
func (q *DBQuerier) FindUser(ctx context.Context, email string) (FindUserRow, error) {
	ctx = context.WithValue(ctx, "pggen_query_name", "FindUser")
	row := q.conn.QueryRow(ctx, findUserSQL, email)
	var item FindUserRow
	if err := row.Scan(&item.Email, &item.Pass); err != nil {
		return item, fmt.Errorf("query FindUser: %w", err)
	}
	return item, nil
}

// textPreferrer wraps a pgtype.ValueTranscoder and sets the preferred encoding
// format to text instead binary (the default). pggen uses the text format
// when the OID is unknownOID because the binary format requires the OID.
// Typically occurs for unregistered types.
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
