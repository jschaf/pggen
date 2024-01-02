// Code generated by pggen. DO NOT EDIT.

package enums

import (
	"context"
	"fmt"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
)

// Querier is a typesafe Go interface backed by SQL queries.
type Querier interface {
	FindAllDevices(ctx context.Context) ([]FindAllDevicesRow, error)

	InsertDevice(ctx context.Context, mac pgtype.Macaddr, typePg DeviceType) (pgconn.CommandTag, error)

	// Select an array of all device_type enum values.
	FindOneDeviceArray(ctx context.Context) ([]DeviceType, error)

	// Select many rows of device_type enum values.
	FindManyDeviceArray(ctx context.Context) ([][]DeviceType, error)

	// Select many rows of device_type enum values with multiple output columns.
	FindManyDeviceArrayWithNum(ctx context.Context) ([]FindManyDeviceArrayWithNumRow, error)

	// Regression test for https://github.com/jschaf/pggen/issues/23.
	EnumInsideComposite(ctx context.Context) (Device, error)
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

// Device represents the Postgres composite type "device".
type Device struct {
	Mac  pgtype.Macaddr `json:"mac"`
	Type DeviceType     `json:"type"`
}

// newDeviceTypeEnum creates a new pgtype.ValueTranscoder for the
// Postgres enum type 'device_type'.
func newDeviceTypeEnum() pgtype.ValueTranscoder {
	return pgtype.NewEnumType(
		"device_type",
		[]string{
			string(DeviceTypeUndefined),
			string(DeviceTypePhone),
			string(DeviceTypeLaptop),
			string(DeviceTypeIpad),
			string(DeviceTypeDesktop),
			string(DeviceTypeIot),
		},
	)
}

// DeviceType represents the Postgres enum "device_type".
type DeviceType string

const (
	DeviceTypeUndefined DeviceType = "undefined"
	DeviceTypePhone     DeviceType = "phone"
	DeviceTypeLaptop    DeviceType = "laptop"
	DeviceTypeIpad      DeviceType = "ipad"
	DeviceTypeDesktop   DeviceType = "desktop"
	DeviceTypeIot       DeviceType = "iot"
)

func (d DeviceType) String() string { return string(d) }

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

// newDevice creates a new pgtype.ValueTranscoder for the Postgres
// composite type 'device'.
func (tr *typeResolver) newDevice() pgtype.ValueTranscoder {
	return tr.newCompositeValue(
		"device",
		compositeField{name: "mac", typeName: "macaddr", defaultVal: &pgtype.Macaddr{}},
		compositeField{name: "type", typeName: "device_type", defaultVal: newDeviceTypeEnum()},
	)
}

// newDeviceTypeArray creates a new pgtype.ValueTranscoder for the Postgres
// '_device_type' array type.
func (tr *typeResolver) newDeviceTypeArray() pgtype.ValueTranscoder {
	return tr.newArrayValue("_device_type", "device_type", newDeviceTypeEnum)
}

const findAllDevicesSQL = `SELECT mac, type
FROM device;`

type FindAllDevicesRow struct {
	Mac  pgtype.Macaddr `json:"mac"`
	Type DeviceType     `json:"type"`
}

// FindAllDevices implements Querier.FindAllDevices.
func (q *DBQuerier) FindAllDevices(ctx context.Context) ([]FindAllDevicesRow, error) {
	ctx = context.WithValue(ctx, "pggen_query_name", "FindAllDevices")
	rows, err := q.conn.Query(ctx, findAllDevicesSQL)
	if err != nil {
		return nil, fmt.Errorf("query FindAllDevices: %w", err)
	}
	defer rows.Close()
	items := []FindAllDevicesRow{}
	for rows.Next() {
		var item FindAllDevicesRow
		if err := rows.Scan(&item.Mac, &item.Type); err != nil {
			return nil, fmt.Errorf("scan FindAllDevices row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("close FindAllDevices rows: %w", err)
	}
	return items, err
}

const insertDeviceSQL = `INSERT INTO device (mac, type)
VALUES ($1, $2);`

// InsertDevice implements Querier.InsertDevice.
func (q *DBQuerier) InsertDevice(ctx context.Context, mac pgtype.Macaddr, typePg DeviceType) (pgconn.CommandTag, error) {
	ctx = context.WithValue(ctx, "pggen_query_name", "InsertDevice")
	cmdTag, err := q.conn.Exec(ctx, insertDeviceSQL, mac, typePg)
	if err != nil {
		return cmdTag, fmt.Errorf("exec query InsertDevice: %w", err)
	}
	return cmdTag, err
}

const findOneDeviceArraySQL = `SELECT enum_range(NULL::device_type) AS device_types;`

// FindOneDeviceArray implements Querier.FindOneDeviceArray.
func (q *DBQuerier) FindOneDeviceArray(ctx context.Context) ([]DeviceType, error) {
	ctx = context.WithValue(ctx, "pggen_query_name", "FindOneDeviceArray")
	row := q.conn.QueryRow(ctx, findOneDeviceArraySQL)
	item := []DeviceType{}
	deviceTypesArray := q.types.newDeviceTypeArray()
	if err := row.Scan(deviceTypesArray); err != nil {
		return item, fmt.Errorf("query FindOneDeviceArray: %w", err)
	}
	if err := deviceTypesArray.AssignTo(&item); err != nil {
		return item, fmt.Errorf("assign FindOneDeviceArray row: %w", err)
	}
	return item, nil
}

const findManyDeviceArraySQL = `SELECT enum_range('ipad'::device_type, 'iot'::device_type) AS device_types
UNION ALL
SELECT enum_range(NULL::device_type) AS device_types;`

// FindManyDeviceArray implements Querier.FindManyDeviceArray.
func (q *DBQuerier) FindManyDeviceArray(ctx context.Context) ([][]DeviceType, error) {
	ctx = context.WithValue(ctx, "pggen_query_name", "FindManyDeviceArray")
	rows, err := q.conn.Query(ctx, findManyDeviceArraySQL)
	if err != nil {
		return nil, fmt.Errorf("query FindManyDeviceArray: %w", err)
	}
	defer rows.Close()
	items := [][]DeviceType{}
	deviceTypesArray := q.types.newDeviceTypeArray()
	for rows.Next() {
		var item []DeviceType
		if err := rows.Scan(deviceTypesArray); err != nil {
			return nil, fmt.Errorf("scan FindManyDeviceArray row: %w", err)
		}
		if err := deviceTypesArray.AssignTo(&item); err != nil {
			return nil, fmt.Errorf("assign FindManyDeviceArray row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("close FindManyDeviceArray rows: %w", err)
	}
	return items, err
}

const findManyDeviceArrayWithNumSQL = `SELECT 1 AS num, enum_range('ipad'::device_type, 'iot'::device_type) AS device_types
UNION ALL
SELECT 2 as num, enum_range(NULL::device_type) AS device_types;`

type FindManyDeviceArrayWithNumRow struct {
	Num         *int32       `json:"num"`
	DeviceTypes []DeviceType `json:"device_types"`
}

// FindManyDeviceArrayWithNum implements Querier.FindManyDeviceArrayWithNum.
func (q *DBQuerier) FindManyDeviceArrayWithNum(ctx context.Context) ([]FindManyDeviceArrayWithNumRow, error) {
	ctx = context.WithValue(ctx, "pggen_query_name", "FindManyDeviceArrayWithNum")
	rows, err := q.conn.Query(ctx, findManyDeviceArrayWithNumSQL)
	if err != nil {
		return nil, fmt.Errorf("query FindManyDeviceArrayWithNum: %w", err)
	}
	defer rows.Close()
	items := []FindManyDeviceArrayWithNumRow{}
	deviceTypesArray := q.types.newDeviceTypeArray()
	for rows.Next() {
		var item FindManyDeviceArrayWithNumRow
		if err := rows.Scan(&item.Num, deviceTypesArray); err != nil {
			return nil, fmt.Errorf("scan FindManyDeviceArrayWithNum row: %w", err)
		}
		if err := deviceTypesArray.AssignTo(&item.DeviceTypes); err != nil {
			return nil, fmt.Errorf("assign FindManyDeviceArrayWithNum row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("close FindManyDeviceArrayWithNum rows: %w", err)
	}
	return items, err
}

const enumInsideCompositeSQL = `SELECT ROW('08:00:2b:01:02:03'::macaddr, 'phone'::device_type) ::device;`

// EnumInsideComposite implements Querier.EnumInsideComposite.
func (q *DBQuerier) EnumInsideComposite(ctx context.Context) (Device, error) {
	ctx = context.WithValue(ctx, "pggen_query_name", "EnumInsideComposite")
	row := q.conn.QueryRow(ctx, enumInsideCompositeSQL)
	var item Device
	rowRow := q.types.newDevice()
	if err := row.Scan(rowRow); err != nil {
		return item, fmt.Errorf("query EnumInsideComposite: %w", err)
	}
	if err := rowRow.AssignTo(&item); err != nil {
		return item, fmt.Errorf("assign EnumInsideComposite row: %w", err)
	}
	return item, nil
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
