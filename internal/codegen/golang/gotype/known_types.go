package gotype

import (
	"github.com/jackc/pgtype"
	"github.com/jschaf/pggen/internal/pg/pgoid"
)

// FindKnownTypeNullable returns the native pgx type, like pgtype.Text, if
// known, for a Postgres OID. If there is no known type, returns nil.
func FindKnownTypePgx(oid pgtype.OID) (Type, bool) {
	typ, ok := knownTypesByOID[oid]
	return typ.pgNative, ok
}

// FindKnownTypeNullable returns the nullable type, like *string, if known, for
// a Postgres OID. Falls back to the pgNative type. If there is no known type
// for the OID, returns nil.
func FindKnownTypeNullable(oid pgtype.OID) (Type, bool) {
	typ, ok := knownTypesByOID[oid]
	if !ok {
		return nil, false
	}
	if typ.nullable != nil {
		return typ.nullable, true
	}
	return typ.pgNative, true
}

// FindKnownTypeNonNullable returns the non-nullable type like string, if known,
// for a Postgres OID. Falls back to the nullable type and pgNative type. If
// there is no known type for the OID, returns nil.
func FindKnownTypeNonNullable(oid pgtype.OID) (Type, bool) {
	typ, ok := knownTypesByOID[oid]
	if !ok {
		return nil, false
	}
	if typ.nonNullable != nil {
		return typ.nonNullable, true
	}
	if typ.nullable != nil {
		return typ.nullable, true
	}
	return typ.pgNative, true
}

// Native go types are not prefixed.
//goland:noinspection GoUnusedGlobalVariable
var (
	Bool          = NewOpaqueType("bool")
	Boolp         = NewOpaqueType("*bool")
	Int           = NewOpaqueType("int")
	Intp          = NewOpaqueType("*int")
	IntSlice      = NewOpaqueType("[]int")
	IntpSlice     = NewOpaqueType("[]*int")
	Int16         = NewOpaqueType("int16")
	Int16p        = NewOpaqueType("*int16")
	Int16Slice    = NewOpaqueType("[]int16")
	Int16pSlice   = NewOpaqueType("[]*int16")
	Int32         = NewOpaqueType("int32")
	Int32p        = NewOpaqueType("*int32")
	Int32Slice    = NewOpaqueType("[]int32")
	Int32pSlice   = NewOpaqueType("[]*int32")
	Int64         = NewOpaqueType("int64")
	Int64p        = NewOpaqueType("*int64")
	Int64Slice    = NewOpaqueType("[]int64")
	Int64pSlice   = NewOpaqueType("[]*int64")
	Uint          = NewOpaqueType("uint")
	UintSlice     = NewOpaqueType("[]uint")
	Uint16        = NewOpaqueType("uint16")
	Uint16Slice   = NewOpaqueType("[]uint16")
	Uint32        = NewOpaqueType("uint32")
	Uint32Slice   = NewOpaqueType("[]uint32")
	Uint64        = NewOpaqueType("uint64")
	Uint64Slice   = NewOpaqueType("[]uint64")
	String        = NewOpaqueType("string")
	Stringp       = NewOpaqueType("*string")
	StringSlice   = NewOpaqueType("[]string")
	StringpSlice  = NewOpaqueType("[]*string")
	Float32       = NewOpaqueType("float32")
	Float32p      = NewOpaqueType("*float32")
	Float32Slice  = NewOpaqueType("[]float32")
	Float32pSlice = NewOpaqueType("[]*float32")
	Float64       = NewOpaqueType("float64")
	Float64p      = NewOpaqueType("*float64")
	Float64Slice  = NewOpaqueType("[]float64")
	Float64pSlice = NewOpaqueType("[]*float64")
	ByteSlice     = NewOpaqueType("[]byte")
)

// pgtype types prefixed with "pg".
var (
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
	PgVoid             = VoidType{}
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

// knownGoType is the native pgtype type, the nullable and non-nullable types
// for a Postgres type.
//
// pgNative means a type that implements the pgx decoder methods directly.
// Such types are typically provided by the pgtype package. Used as the fallback
// type and for cases like composite types where we need a
// pgtype.ValueTranscoder.
//
// A nullable type is one that can represent a nullable column, like *string for
// a Postgres text type that can be null. A nullable type is nicer to work with
// than the corresponding pgNative type, i.e. "*string" is easier to work with
// than pgtype.Text{}.
//
// A nonNullable type is one that can represent a column that's never null, like
// "string" for a Postgres text type.
type knownGoType struct{ pgNative, nullable, nonNullable Type }

var knownTypesByOID = map[pgtype.OID]knownGoType{
	pgtype.BoolOID:             {PgBool, Boolp, Bool},
	pgtype.QCharOID:            {PgQChar, nil, nil},
	pgtype.NameOID:             {PgName, nil, nil},
	pgtype.Int8OID:             {PgInt8, Intp, Int},
	pgtype.Int2OID:             {PgInt2, Int16p, Int16},
	pgtype.Int4OID:             {PgInt4, Int32p, Int32},
	pgtype.TextOID:             {PgText, Stringp, String},
	pgtype.ByteaOID:            {PgBytea, PgBytea, ByteSlice},
	pgtype.OIDOID:              {PgOID, nil, nil},
	pgtype.TIDOID:              {PgTID, nil, nil},
	pgtype.XIDOID:              {PgXID, nil, nil},
	pgtype.CIDOID:              {PgCID, nil, nil},
	pgtype.JSONOID:             {PgJSON, nil, nil},
	pgtype.PointOID:            {PgPoint, nil, nil},
	pgtype.LsegOID:             {PgLseg, nil, nil},
	pgtype.PathOID:             {PgPath, nil, nil},
	pgtype.BoxOID:              {PgBox, nil, nil},
	pgtype.PolygonOID:          {PgPolygon, nil, nil},
	pgtype.LineOID:             {PgLine, nil, nil},
	pgtype.CIDROID:             {PgCIDR, nil, nil},
	pgtype.CIDRArrayOID:        {PgCIDRArray, nil, nil},
	pgtype.Float4OID:           {PgFloat4, nil, nil},
	pgtype.Float8OID:           {PgFloat8, nil, nil},
	pgoid.OIDArray:             {Uint32Slice, nil, nil},
	pgtype.UnknownOID:          {PgUnknown, nil, nil},
	pgtype.CircleOID:           {PgCircle, nil, nil},
	pgtype.MacaddrOID:          {PgMacaddr, nil, nil},
	pgtype.InetOID:             {PgInet, nil, nil},
	pgtype.BoolArrayOID:        {PgBoolArray, nil, nil},
	pgtype.ByteaArrayOID:       {PgByteaArray, nil, nil},
	pgtype.Int2ArrayOID:        {PgInt2Array, Int16pSlice, Int16Slice},
	pgtype.Int4ArrayOID:        {PgInt4Array, Int32pSlice, Int32Slice},
	pgtype.TextArrayOID:        {PgTextArray, StringSlice, nil},
	pgtype.BPCharArrayOID:      {PgBPCharArray, nil, nil},
	pgtype.VarcharArrayOID:     {PgVarcharArray, nil, nil},
	pgtype.Int8ArrayOID:        {PgInt8Array, IntpSlice, IntSlice},
	pgtype.Float4ArrayOID:      {PgFloat4Array, Float32pSlice, Float32Slice},
	pgtype.Float8ArrayOID:      {PgFloat8Array, Float64pSlice, Float64Slice},
	pgtype.ACLItemOID:          {PgACLItem, nil, nil},
	pgtype.ACLItemArrayOID:     {PgACLItemArray, nil, nil},
	pgtype.InetArrayOID:        {PgInetArray, nil, nil},
	pgoid.MacaddrArray:         {PgMacaddrArray, nil, nil},
	pgtype.BPCharOID:           {PgBPChar, nil, nil},
	pgtype.VarcharOID:          {PgVarchar, nil, nil},
	pgtype.DateOID:             {PgDate, nil, nil},
	pgtype.TimeOID:             {PgTime, nil, nil},
	pgtype.TimestampOID:        {PgTimestamp, nil, nil},
	pgtype.TimestampArrayOID:   {PgTimestampArray, nil, nil},
	pgtype.DateArrayOID:        {PgDateArray, nil, nil},
	pgtype.TimestamptzOID:      {PgTimestamptz, nil, nil},
	pgtype.TimestamptzArrayOID: {PgTimestamptzArray, nil, nil},
	pgtype.IntervalOID:         {PgInterval, nil, nil},
	pgtype.NumericArrayOID:     {PgNumericArray, nil, nil},
	pgtype.BitOID:              {PgBit, nil, nil},
	pgtype.VarbitOID:           {PgVarbit, nil, nil},
	pgoid.Void:                 {PgVoid, nil, nil},
	pgtype.NumericOID:          {PgNumeric, nil, nil},
	pgtype.RecordOID:           {PgRecord, nil, nil},
	pgtype.UUIDOID:             {PgUUID, nil, nil},
	pgtype.UUIDArrayOID:        {PgUUIDArray, nil, nil},
	pgtype.JSONBOID:            {PgJSONB, nil, nil},
	pgtype.JSONBArrayOID:       {PgJSONBArray, nil, nil},
	pgtype.Int4rangeOID:        {PgInt4range, nil, nil},
	pgtype.NumrangeOID:         {PgNumrange, nil, nil},
	pgtype.TsrangeOID:          {PgTsrange, nil, nil},
	pgtype.TstzrangeOID:        {PgTstzrange, nil, nil},
	pgtype.DaterangeOID:        {PgDaterange, nil, nil},
	pgtype.Int8rangeOID:        {PgInt8range, nil, nil},
}
