package gotype

import (
	"github.com/jackc/pgtype"
	"github.com/jschaf/pggen/internal/pg/pgoid"
)

// FindKnownTypeByOID returns the type, if known, for a Postgres OID.
// If there is no known type, returns nil.
func FindKnownTypeByOID(oid pgtype.OID, nullable bool) (Type, bool) {
	typ, ok := knownTypesByOID[oid]
	if !ok {
		return nil, false
	}
	if nullable || typ.nonNullable == nil {
		return typ.nullable, true
	}
	return typ.nonNullable, true
}

//goland:noinspection GoUnusedGlobalVariable
var (
	// Native go types are not prefixed.
	Bool         = NewOpaqueType("bool")
	Int          = NewOpaqueType("int")
	IntSlice     = NewOpaqueType("[]int")
	Int16        = NewOpaqueType("int16")
	Int16Slice   = NewOpaqueType("[]int16")
	Int32        = NewOpaqueType("int32")
	Int32Slice   = NewOpaqueType("[]int32")
	Int64        = NewOpaqueType("int64")
	Int64Slice   = NewOpaqueType("[]int64")
	Uint         = NewOpaqueType("uint")
	UintSlice    = NewOpaqueType("[]uint")
	Uint16       = NewOpaqueType("uint16")
	Uint16Slice  = NewOpaqueType("[]uint16")
	Uint32       = NewOpaqueType("uint32")
	Uint32Slice  = NewOpaqueType("[]uint32")
	Uint64       = NewOpaqueType("uint64")
	Uint64Slice  = NewOpaqueType("[]uint64")
	String       = NewOpaqueType("string")
	StringSlice  = NewOpaqueType("[]string")
	Float32      = NewOpaqueType("float32")
	Float32Slice = NewOpaqueType("[]float32")
	Float64      = NewOpaqueType("float64")
	Float64Slice = NewOpaqueType("[]float64")
	ByteSlice    = NewOpaqueType("[]byte")

	// pgtype types prefixed with "pg".
	PgBool             = NewOpaqueType("github.com/jackc/pgtype.Bool")
	PgQChar            = NewOpaqueType("github.com/jackc/pgtype.QChar")
	PgName             = NewOpaqueType("github.com/jackc/pgtype.Name")
	PgInt8             = NewOpaqueType("github.com/jackc/pgtype.Int8")
	PgInt2             = NewOpaqueType("github.com/jackc/pgtype.Int2")
	PgInt4             = NewOpaqueType("github.com/jackc/pgtype.Int4")
	PgText             = NewOpaqueType("github.com/jackc/pgtype.Text")
	PgBytea            = NewOpaqueType("github.com/jackc/pgtype.Bytea")
	PgOID              = NewOpaqueType("github.com/jackc/pgtype.OID")
	PgTID              = NewOpaqueType("github.com/jackc/pgtype.TID")
	PgXID              = NewOpaqueType("github.com/jackc/pgtype.XID")
	PgCID              = NewOpaqueType("github.com/jackc/pgtype.CID")
	PgJSON             = NewOpaqueType("github.com/jackc/pgtype.JSON")
	PgPoint            = NewOpaqueType("github.com/jackc/pgtype.Point")
	PgLseg             = NewOpaqueType("github.com/jackc/pgtype.Lseg")
	PgPath             = NewOpaqueType("github.com/jackc/pgtype.Path")
	PgBox              = NewOpaqueType("github.com/jackc/pgtype.Box")
	PgPolygon          = NewOpaqueType("github.com/jackc/pgtype.Polygon")
	PgLine             = NewOpaqueType("github.com/jackc/pgtype.Line")
	PgCIDR             = NewOpaqueType("github.com/jackc/pgtype.CIDR")
	PgCIDRArray        = NewOpaqueType("github.com/jackc/pgtype.CIDRArray")
	PgFloat4           = NewOpaqueType("github.com/jackc/pgtype.Float4")
	PgFloat8           = NewOpaqueType("github.com/jackc/pgtype.Float8")
	PgUnknown          = NewOpaqueType("github.com/jackc/pgtype.Unknown")
	PgCircle           = NewOpaqueType("github.com/jackc/pgtype.Circle")
	PgMacaddr          = NewOpaqueType("github.com/jackc/pgtype.Macaddr")
	PgInet             = NewOpaqueType("github.com/jackc/pgtype.Inet")
	PgBoolArray        = NewOpaqueType("github.com/jackc/pgtype.BoolArray")
	PgByteaArray       = NewOpaqueType("github.com/jackc/pgtype.ByteaArray")
	PgInt2Array        = NewOpaqueType("github.com/jackc/pgtype.Int2Array")
	PgInt4Array        = NewOpaqueType("github.com/jackc/pgtype.Int4Array")
	PgTextArray        = NewOpaqueType("github.com/jackc/pgtype.TextArray")
	PgBPCharArray      = NewOpaqueType("github.com/jackc/pgtype.BPCharArray")
	PgVarcharArray     = NewOpaqueType("github.com/jackc/pgtype.VarcharArray")
	PgInt8Array        = NewOpaqueType("github.com/jackc/pgtype.Int8Array")
	PgFloat4Array      = NewOpaqueType("github.com/jackc/pgtype.Float4Array")
	PgFloat8Array      = NewOpaqueType("github.com/jackc/pgtype.Float8Array")
	PgACLItem          = NewOpaqueType("github.com/jackc/pgtype.ACLItem")
	PgACLItemArray     = NewOpaqueType("github.com/jackc/pgtype.ACLItemArray")
	PgInetArray        = NewOpaqueType("github.com/jackc/pgtype.InetArray")
	PgMacaddrArray     = NewOpaqueType("github.com/jackc/pgtype.MacaddrArray")
	PgBPChar           = NewOpaqueType("github.com/jackc/pgtype.BPChar")
	PgVarchar          = NewOpaqueType("github.com/jackc/pgtype.Varchar")
	PgDate             = NewOpaqueType("github.com/jackc/pgtype.Date")
	PgTime             = NewOpaqueType("github.com/jackc/pgtype.Time")
	PgTimestamp        = NewOpaqueType("github.com/jackc/pgtype.Timestamp")
	PgTimestampArray   = NewOpaqueType("github.com/jackc/pgtype.TimestampArray")
	PgDateArray        = NewOpaqueType("github.com/jackc/pgtype.DateArray")
	PgTimestamptz      = NewOpaqueType("github.com/jackc/pgtype.Timestamptz")
	PgTimestamptzArray = NewOpaqueType("github.com/jackc/pgtype.TimestamptzArray")
	PgInterval         = NewOpaqueType("github.com/jackc/pgtype.Interval")
	PgNumericArray     = NewOpaqueType("github.com/jackc/pgtype.NumericArray")
	PgBit              = NewOpaqueType("github.com/jackc/pgtype.Bit")
	PgVarbit           = NewOpaqueType("github.com/jackc/pgtype.Varbit")
	PgNumeric          = NewOpaqueType("github.com/jackc/pgtype.Numeric")
	PgRecord           = NewOpaqueType("github.com/jackc/pgtype.Record")
	PgUUID             = NewOpaqueType("github.com/jackc/pgtype.UUID")
	PgUUIDArray        = NewOpaqueType("github.com/jackc/pgtype.UUIDArray")
	PgJSONB            = NewOpaqueType("github.com/jackc/pgtype.JSONB")
	PgJSONBArray       = NewOpaqueType("github.com/jackc/pgtype.JSONBArray")
	PgInt4range        = NewOpaqueType("github.com/jackc/pgtype.Int4range")
	PgNumrange         = NewOpaqueType("github.com/jackc/pgtype.Numrange")
	PgTsrange          = NewOpaqueType("github.com/jackc/pgtype.Tsrange")
	PgTstzrange        = NewOpaqueType("github.com/jackc/pgtype.Tstzrange")
	PgDaterange        = NewOpaqueType("github.com/jackc/pgtype.Daterange")
	PgInt8range        = NewOpaqueType("github.com/jackc/pgtype.Int8range")
)

// knownGoType is the nullable and non-nullable types for a Postgres type.
// We use a non-nullable type when possible because it offers better ergonomics.
// It's nicer to get a string as an output column rather than pgtype.Text which
// requires checking for a null value.
type knownGoType struct{ nullable, nonNullable Type }

var knownTypesByOID = map[pgtype.OID]knownGoType{
	pgtype.BoolOID:             {PgBool, Bool},
	pgtype.QCharOID:            {PgQChar, nil},
	pgtype.NameOID:             {PgName, nil},
	pgtype.Int8OID:             {PgInt8, Int},
	pgtype.Int2OID:             {PgInt2, Int16},
	pgtype.Int4OID:             {PgInt4, Int32},
	pgtype.TextOID:             {PgText, String},
	pgtype.ByteaOID:            {PgBytea, ByteSlice},
	pgtype.OIDOID:              {PgOID, nil},
	pgtype.TIDOID:              {PgTID, nil},
	pgtype.XIDOID:              {PgXID, nil},
	pgtype.CIDOID:              {PgCID, nil},
	pgtype.JSONOID:             {PgJSON, nil},
	pgtype.PointOID:            {PgPoint, nil},
	pgtype.LsegOID:             {PgLseg, nil},
	pgtype.PathOID:             {PgPath, nil},
	pgtype.BoxOID:              {PgBox, nil},
	pgtype.PolygonOID:          {PgPolygon, nil},
	pgtype.LineOID:             {PgLine, nil},
	pgtype.CIDROID:             {PgCIDR, nil},
	pgtype.CIDRArrayOID:        {PgCIDRArray, nil},
	pgtype.Float4OID:           {PgFloat4, nil},
	pgtype.Float8OID:           {PgFloat8, nil},
	pgoid.OIDArray:             {Uint32Slice, nil},
	pgtype.UnknownOID:          {PgUnknown, nil},
	pgtype.CircleOID:           {PgCircle, nil},
	pgtype.MacaddrOID:          {PgMacaddr, nil},
	pgtype.InetOID:             {PgInet, nil},
	pgtype.BoolArrayOID:        {PgBoolArray, nil},
	pgtype.ByteaArrayOID:       {PgByteaArray, nil},
	pgtype.Int2ArrayOID:        {PgInt2Array, Int16Slice},
	pgtype.Int4ArrayOID:        {PgInt4Array, Int32Slice},
	pgtype.TextArrayOID:        {PgTextArray, StringSlice},
	pgtype.BPCharArrayOID:      {PgBPCharArray, nil},
	pgtype.VarcharArrayOID:     {PgVarcharArray, nil},
	pgtype.Int8ArrayOID:        {PgInt8Array, IntSlice},
	pgtype.Float4ArrayOID:      {PgFloat4Array, Float32Slice},
	pgtype.Float8ArrayOID:      {PgFloat8Array, Float64Slice},
	pgtype.ACLItemOID:          {PgACLItem, nil},
	pgtype.ACLItemArrayOID:     {PgACLItemArray, nil},
	pgtype.InetArrayOID:        {PgInetArray, nil},
	pgoid.MacaddrArray:         {PgMacaddrArray, nil},
	pgtype.BPCharOID:           {PgBPChar, nil},
	pgtype.VarcharOID:          {PgVarchar, nil},
	pgtype.DateOID:             {PgDate, nil},
	pgtype.TimeOID:             {PgTime, nil},
	pgtype.TimestampOID:        {PgTimestamp, nil},
	pgtype.TimestampArrayOID:   {PgTimestampArray, nil},
	pgtype.DateArrayOID:        {PgDateArray, nil},
	pgtype.TimestamptzOID:      {PgTimestamptz, nil},
	pgtype.TimestamptzArrayOID: {PgTimestamptzArray, nil},
	pgtype.IntervalOID:         {PgInterval, nil},
	pgtype.NumericArrayOID:     {PgNumericArray, nil},
	pgtype.BitOID:              {PgBit, nil},
	pgtype.VarbitOID:           {PgVarbit, nil},
	pgtype.NumericOID:          {PgNumeric, nil},
	pgtype.RecordOID:           {PgRecord, nil},
	pgtype.UUIDOID:             {PgUUID, nil},
	pgtype.UUIDArrayOID:        {PgUUIDArray, nil},
	pgtype.JSONBOID:            {PgJSONB, nil},
	pgtype.JSONBArrayOID:       {PgJSONBArray, nil},
	pgtype.Int4rangeOID:        {PgInt4range, nil},
	pgtype.NumrangeOID:         {PgNumrange, nil},
	pgtype.TsrangeOID:          {PgTsrange, nil},
	pgtype.TstzrangeOID:        {PgTstzrange, nil},
	pgtype.DaterangeOID:        {PgDaterange, nil},
	pgtype.Int8rangeOID:        {PgInt8range, nil},
}
