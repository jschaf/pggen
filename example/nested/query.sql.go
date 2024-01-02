// Code generated by pggen. DO NOT EDIT.

package nested

import (
	"context"
	"fmt"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
)

// Querier is a typesafe Go interface backed by SQL queries.
type Querier interface {
	ArrayNested2(ctx context.Context) ([]ProductImageType, error)

	Nested3(ctx context.Context) ([]ProductImageSetType, error)
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

// Dimensions represents the Postgres composite type "dimensions".
type Dimensions struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// ProductImageSetType represents the Postgres composite type "product_image_set_type".
type ProductImageSetType struct {
	Name      string             `json:"name"`
	OrigImage ProductImageType   `json:"orig_image"`
	Images    []ProductImageType `json:"images"`
}

// ProductImageType represents the Postgres composite type "product_image_type".
type ProductImageType struct {
	Source     string     `json:"source"`
	Dimensions Dimensions `json:"dimensions"`
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

// newDimensions creates a new pgtype.ValueTranscoder for the Postgres
// composite type 'dimensions'.
func (tr *typeResolver) newDimensions() pgtype.ValueTranscoder {
	return tr.newCompositeValue(
		"dimensions",
		compositeField{name: "width", typeName: "int4", defaultVal: &pgtype.Int4{}},
		compositeField{name: "height", typeName: "int4", defaultVal: &pgtype.Int4{}},
	)
}

// newProductImageSetType creates a new pgtype.ValueTranscoder for the Postgres
// composite type 'product_image_set_type'.
func (tr *typeResolver) newProductImageSetType() pgtype.ValueTranscoder {
	return tr.newCompositeValue(
		"product_image_set_type",
		compositeField{name: "name", typeName: "text", defaultVal: &pgtype.Text{}},
		compositeField{name: "orig_image", typeName: "product_image_type", defaultVal: tr.newProductImageType()},
		compositeField{name: "images", typeName: "_product_image_type", defaultVal: tr.newProductImageTypeArray()},
	)
}

// newProductImageType creates a new pgtype.ValueTranscoder for the Postgres
// composite type 'product_image_type'.
func (tr *typeResolver) newProductImageType() pgtype.ValueTranscoder {
	return tr.newCompositeValue(
		"product_image_type",
		compositeField{name: "source", typeName: "text", defaultVal: &pgtype.Text{}},
		compositeField{name: "dimensions", typeName: "dimensions", defaultVal: tr.newDimensions()},
	)
}

// newProductImageTypeArray creates a new pgtype.ValueTranscoder for the Postgres
// '_product_image_type' array type.
func (tr *typeResolver) newProductImageTypeArray() pgtype.ValueTranscoder {
	return tr.newArrayValue("_product_image_type", "product_image_type", tr.newProductImageType)
}

const arrayNested2SQL = `SELECT
  ARRAY [
    ROW ('img2', ROW (22, 22)::dimensions)::product_image_type,
    ROW ('img3', ROW (33, 33)::dimensions)::product_image_type
    ] AS images;`

// ArrayNested2 implements Querier.ArrayNested2.
func (q *DBQuerier) ArrayNested2(ctx context.Context) ([]ProductImageType, error) {
	ctx = context.WithValue(ctx, "pggen_query_name", "ArrayNested2")
	row := q.conn.QueryRow(ctx, arrayNested2SQL)
	item := []ProductImageType{}
	imagesArray := q.types.newProductImageTypeArray()
	if err := row.Scan(imagesArray); err != nil {
		return item, fmt.Errorf("query ArrayNested2: %w", err)
	}
	if err := imagesArray.AssignTo(&item); err != nil {
		return item, fmt.Errorf("assign ArrayNested2 row: %w", err)
	}
	return item, nil
}

const nested3SQL = `SELECT
  ROW (
    'name', -- name
    ROW ('img1', ROW (11, 11)::dimensions)::product_image_type, -- orig_image
    ARRAY [ --images
      ROW ('img2', ROW (22, 22)::dimensions)::product_image_type,
      ROW ('img3', ROW (33, 33)::dimensions)::product_image_type
      ]
    )::product_image_set_type;`

// Nested3 implements Querier.Nested3.
func (q *DBQuerier) Nested3(ctx context.Context) ([]ProductImageSetType, error) {
	ctx = context.WithValue(ctx, "pggen_query_name", "Nested3")
	rows, err := q.conn.Query(ctx, nested3SQL)
	if err != nil {
		return nil, fmt.Errorf("query Nested3: %w", err)
	}
	defer rows.Close()
	items := []ProductImageSetType{}
	rowRow := q.types.newProductImageSetType()
	for rows.Next() {
		var item ProductImageSetType
		if err := rows.Scan(rowRow); err != nil {
			return nil, fmt.Errorf("scan Nested3 row: %w", err)
		}
		if err := rowRow.AssignTo(&item); err != nil {
			return nil, fmt.Errorf("assign Nested3 row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("close Nested3 rows: %w", err)
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
