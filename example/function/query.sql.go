// Code generated by pggen. DO NOT EDIT.

package function

import (
	"context"
	"fmt"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
)

// Querier is a typesafe Go interface backed by SQL queries.
type Querier interface {
	OutParams(ctx context.Context) ([]OutParamsRow, error)
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

// ListItem represents the Postgres composite type "list_item".
type ListItem struct {
	Name  *string `json:"name"`
	Color *string `json:"color"`
}

// ListStats represents the Postgres composite type "list_stats".
type ListStats struct {
	Val1 *string  `json:"val1"`
	Val2 []*int32 `json:"val2"`
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

// newListItem creates a new pgtype.ValueTranscoder for the Postgres
// composite type 'list_item'.
func (tr *typeResolver) newListItem() pgtype.ValueTranscoder {
	return tr.newCompositeValue(
		"list_item",
		compositeField{name: "name", typeName: "text", defaultVal: &pgtype.Text{}},
		compositeField{name: "color", typeName: "text", defaultVal: &pgtype.Text{}},
	)
}

// newListStats creates a new pgtype.ValueTranscoder for the Postgres
// composite type 'list_stats'.
func (tr *typeResolver) newListStats() pgtype.ValueTranscoder {
	return tr.newCompositeValue(
		"list_stats",
		compositeField{name: "val1", typeName: "text", defaultVal: &pgtype.Text{}},
		compositeField{name: "val2", typeName: "_int4", defaultVal: &pgtype.Int4Array{}},
	)
}

// newListItemArray creates a new pgtype.ValueTranscoder for the Postgres
// '_list_item' array type.
func (tr *typeResolver) newListItemArray() pgtype.ValueTranscoder {
	return tr.newArrayValue("_list_item", "list_item", tr.newListItem)
}

const outParamsSQL = `SELECT * FROM out_params();`

type OutParamsRow struct {
	Items []ListItem `json:"_items"`
	Stats ListStats  `json:"_stats"`
}

// OutParams implements Querier.OutParams.
func (q *DBQuerier) OutParams(ctx context.Context) ([]OutParamsRow, error) {
	ctx = context.WithValue(ctx, "pggen_query_name", "OutParams")
	rows, err := q.conn.Query(ctx, outParamsSQL)
	if err != nil {
		return nil, fmt.Errorf("query OutParams: %w", err)
	}
	defer rows.Close()
	items := []OutParamsRow{}
	itemsArray := q.types.newListItemArray()
	statsRow := q.types.newListStats()
	for rows.Next() {
		var item OutParamsRow
		if err := rows.Scan(itemsArray, statsRow); err != nil {
			return nil, fmt.Errorf("scan OutParams row: %w", err)
		}
		if err := itemsArray.AssignTo(&item.Items); err != nil {
			return nil, fmt.Errorf("assign OutParams row: %w", err)
		}
		if err := statsRow.AssignTo(&item.Stats); err != nil {
			return nil, fmt.Errorf("assign OutParams row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("close OutParams rows: %w", err)
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
