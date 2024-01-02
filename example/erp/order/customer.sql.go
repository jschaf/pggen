// Code generated by pggen. DO NOT EDIT.

package order

import (
	"context"
	"fmt"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
)

// Querier is a typesafe Go interface backed by SQL queries.
type Querier interface {
	CreateTenant(ctx context.Context, key string, name string) (CreateTenantRow, error)

	FindOrdersByCustomer(ctx context.Context, customerID int32) ([]FindOrdersByCustomerRow, error)

	FindProductsInOrder(ctx context.Context, orderID int32) ([]FindProductsInOrderRow, error)

	InsertCustomer(ctx context.Context, params InsertCustomerParams) (InsertCustomerRow, error)

	InsertOrder(ctx context.Context, params InsertOrderParams) (InsertOrderRow, error)

	FindOrdersByPrice(ctx context.Context, minTotal pgtype.Numeric) ([]FindOrdersByPriceRow, error)

	FindOrdersMRR(ctx context.Context) ([]FindOrdersMRRRow, error)
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

const createTenantSQL = `INSERT INTO tenant (tenant_id, name)
VALUES (base36_decode($1::text)::tenant_id, $2::text)
RETURNING *;`

type CreateTenantRow struct {
	TenantID int     `json:"tenant_id"`
	Rname    *string `json:"rname"`
	Name     string  `json:"name"`
}

// CreateTenant implements Querier.CreateTenant.
func (q *DBQuerier) CreateTenant(ctx context.Context, key string, name string) (CreateTenantRow, error) {
	ctx = context.WithValue(ctx, "pggen_query_name", "CreateTenant")
	row := q.conn.QueryRow(ctx, createTenantSQL, key, name)
	var item CreateTenantRow
	if err := row.Scan(&item.TenantID, &item.Rname, &item.Name); err != nil {
		return item, fmt.Errorf("query CreateTenant: %w", err)
	}
	return item, nil
}

const findOrdersByCustomerSQL = `SELECT *
FROM orders
WHERE customer_id = $1;`

type FindOrdersByCustomerRow struct {
	OrderID    int32              `json:"order_id"`
	OrderDate  pgtype.Timestamptz `json:"order_date"`
	OrderTotal pgtype.Numeric     `json:"order_total"`
	CustomerID *int32             `json:"customer_id"`
}

// FindOrdersByCustomer implements Querier.FindOrdersByCustomer.
func (q *DBQuerier) FindOrdersByCustomer(ctx context.Context, customerID int32) ([]FindOrdersByCustomerRow, error) {
	ctx = context.WithValue(ctx, "pggen_query_name", "FindOrdersByCustomer")
	rows, err := q.conn.Query(ctx, findOrdersByCustomerSQL, customerID)
	if err != nil {
		return nil, fmt.Errorf("query FindOrdersByCustomer: %w", err)
	}
	defer rows.Close()
	items := []FindOrdersByCustomerRow{}
	for rows.Next() {
		var item FindOrdersByCustomerRow
		if err := rows.Scan(&item.OrderID, &item.OrderDate, &item.OrderTotal, &item.CustomerID); err != nil {
			return nil, fmt.Errorf("scan FindOrdersByCustomer row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("close FindOrdersByCustomer rows: %w", err)
	}
	return items, err
}

const findProductsInOrderSQL = `SELECT o.order_id, p.product_id, p.name
FROM orders o
  INNER JOIN order_product op USING (order_id)
  INNER JOIN product p USING (product_id)
WHERE o.order_id = $1;`

type FindProductsInOrderRow struct {
	OrderID   *int32  `json:"order_id"`
	ProductID *int32  `json:"product_id"`
	Name      *string `json:"name"`
}

// FindProductsInOrder implements Querier.FindProductsInOrder.
func (q *DBQuerier) FindProductsInOrder(ctx context.Context, orderID int32) ([]FindProductsInOrderRow, error) {
	ctx = context.WithValue(ctx, "pggen_query_name", "FindProductsInOrder")
	rows, err := q.conn.Query(ctx, findProductsInOrderSQL, orderID)
	if err != nil {
		return nil, fmt.Errorf("query FindProductsInOrder: %w", err)
	}
	defer rows.Close()
	items := []FindProductsInOrderRow{}
	for rows.Next() {
		var item FindProductsInOrderRow
		if err := rows.Scan(&item.OrderID, &item.ProductID, &item.Name); err != nil {
			return nil, fmt.Errorf("scan FindProductsInOrder row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("close FindProductsInOrder rows: %w", err)
	}
	return items, err
}

const insertCustomerSQL = `INSERT INTO customer (first_name, last_name, email)
VALUES ($1, $2, $3)
RETURNING *;`

type InsertCustomerParams struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}

type InsertCustomerRow struct {
	CustomerID int32  `json:"customer_id"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Email      string `json:"email"`
}

// InsertCustomer implements Querier.InsertCustomer.
func (q *DBQuerier) InsertCustomer(ctx context.Context, params InsertCustomerParams) (InsertCustomerRow, error) {
	ctx = context.WithValue(ctx, "pggen_query_name", "InsertCustomer")
	row := q.conn.QueryRow(ctx, insertCustomerSQL, params.FirstName, params.LastName, params.Email)
	var item InsertCustomerRow
	if err := row.Scan(&item.CustomerID, &item.FirstName, &item.LastName, &item.Email); err != nil {
		return item, fmt.Errorf("query InsertCustomer: %w", err)
	}
	return item, nil
}

const insertOrderSQL = `INSERT INTO orders (order_date, order_total, customer_id)
VALUES ($1, $2, $3)
RETURNING *;`

type InsertOrderParams struct {
	OrderDate  pgtype.Timestamptz `json:"order_date"`
	OrderTotal pgtype.Numeric     `json:"order_total"`
	CustID     int32              `json:"cust_id"`
}

type InsertOrderRow struct {
	OrderID    int32              `json:"order_id"`
	OrderDate  pgtype.Timestamptz `json:"order_date"`
	OrderTotal pgtype.Numeric     `json:"order_total"`
	CustomerID *int32             `json:"customer_id"`
}

// InsertOrder implements Querier.InsertOrder.
func (q *DBQuerier) InsertOrder(ctx context.Context, params InsertOrderParams) (InsertOrderRow, error) {
	ctx = context.WithValue(ctx, "pggen_query_name", "InsertOrder")
	row := q.conn.QueryRow(ctx, insertOrderSQL, params.OrderDate, params.OrderTotal, params.CustID)
	var item InsertOrderRow
	if err := row.Scan(&item.OrderID, &item.OrderDate, &item.OrderTotal, &item.CustomerID); err != nil {
		return item, fmt.Errorf("query InsertOrder: %w", err)
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
