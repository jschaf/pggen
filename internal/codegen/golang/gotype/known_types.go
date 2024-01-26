package gotype

import (
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jschaf/pggen/internal/pg"
	"github.com/jschaf/pggen/internal/pg/pgoid"
)

// FindKnownTypePgx returns the native pgx type, like pgtype.Text, if known, for
// a Postgres OID. If there is no known type, returns nil.
func FindKnownTypePgx(oid uint32) (Type, bool) {
	typ, ok := knownTypesByOID[oid]
	return typ.pgNative, ok
}

// FindKnownTypeNullable returns the nullable type, like *string, if known, for
// a Postgres OID. Falls back to the pgNative type. If there is no known type
// for the OID, returns nil.
func FindKnownTypeNullable(oid uint32) (Type, bool) {
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
func FindKnownTypeNonNullable(oid uint32) (Type, bool) {
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
//
//goland:noinspection GoUnusedGlobalVariable
var (
	Bool          = MustParseKnownType("bool", pg.Bool)
	Boolp         = MustParseKnownType("*bool", pg.Bool)
	Int           = MustParseKnownType("int", pg.Int8)
	Intp          = MustParseKnownType("*int", pg.Int8)
	IntSlice      = MustParseKnownType("[]int", pg.Int8Array)
	IntpSlice     = MustParseKnownType("[]*int", pg.Int8Array)
	Int16         = MustParseKnownType("int16", pg.Int2)
	Int16p        = MustParseKnownType("*int16", pg.Int2)
	Int16Slice    = MustParseKnownType("[]int16", pg.Int2Array)
	Int16pSlice   = MustParseKnownType("[]*int16", pg.Int2Array)
	Int32         = MustParseKnownType("int32", pg.Int4)
	Int32p        = MustParseKnownType("*int32", pg.Int4)
	Int32Slice    = MustParseKnownType("[]int32", pg.Int4Array)
	Int32pSlice   = MustParseKnownType("[]*int32", pg.Int4Array)
	Int64         = MustParseKnownType("int64", pg.Int8)
	Int64p        = MustParseKnownType("*int64", pg.Int8)
	Int64Slice    = MustParseKnownType("[]int64", pg.Int8Array)
	Int64pSlice   = MustParseKnownType("[]*int64", pg.Int8Array)
	Uint          = MustParseKnownType("uint", pg.Int8)
	UintSlice     = MustParseKnownType("[]uint", pg.Int8Array)
	Uint16        = MustParseKnownType("uint16", pg.Int2)
	Uint16Slice   = MustParseKnownType("[]uint16", pg.Int2Array)
	Uint32        = MustParseKnownType("uint32", pg.Int4)
	Uint32Slice   = MustParseKnownType("[]uint32", pg.Int4Array)
	Uint64        = MustParseKnownType("uint64", pg.Int8)
	Uint64Slice   = MustParseKnownType("[]uint64", pg.Int8Array)
	String        = MustParseKnownType("string", pg.Text)
	Stringp       = MustParseKnownType("*string", pg.Text)
	StringSlice   = MustParseKnownType("[]string", pg.TextArray)
	StringpSlice  = MustParseKnownType("[]*string", pg.TextArray)
	Float32       = MustParseKnownType("float32", pg.Float4)
	Float32p      = MustParseKnownType("*float32", pg.Float4)
	Float32Slice  = MustParseKnownType("[]float32", pg.Float4Array)
	Float32pSlice = MustParseKnownType("[]*float32", pg.Float4Array)
	Float64       = MustParseKnownType("float64", pg.Float8)
	Float64p      = MustParseKnownType("*float64", pg.Float8)
	Float64Slice  = MustParseKnownType("[]float64", pg.Float8Array)
	Float64pSlice = MustParseKnownType("[]*float64", pg.Float8Array)
	ByteSlice     = MustParseKnownType("[]byte", pg.Bytea)
)

// pgtype types prefixed with "pg".
var (
	PgBool             = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Bool", pg.Bool)
	PgQChar            = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.QChar", pg.QChar)
	PgName             = MustParseKnownType("string", pg.Name)
	PgInt8             = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Int8", pg.Int8)
	PgInt2             = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Int2", pg.Int2)
	PgInt4             = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Int4", pg.Int4)
	PgText             = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Text", pg.Text)
	PgBytea            = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Bytea", pg.Bytea)
	PgOID              = MustParseKnownType("uint32", pg.OID)
	PgTID              = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.TID", pg.TID)
	PgXID              = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.XID", pg.XID)
	PgCID              = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.CID", pg.CID)
	PgJSON             = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.JSON", pg.JSON)
	PgPoint            = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Point", pg.Point)
	PgLseg             = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Lseg", pg.Lseg)
	PgPath             = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Path", pg.Path)
	PgBox              = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Box", pg.Box)
	PgPolygon          = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Polygon", pg.Polygon)
	PgLine             = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Line", pg.Line)
	PgCIDR             = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.CIDR", pg.CIDR)
	PgCIDRArray        = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.CIDRArray", pg.CIDRArray)
	PgFloat4           = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Float4", pg.Float4)
	PgFloat8           = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Float8", pg.Float8)
	PgUnknown          = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Unknown", pg.Unknown)
	PgCircle           = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Circle", pg.Circle)
	PgMacaddr          = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Macaddr", pg.Macaddr)
	PgInet             = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Inet", pg.Inet)
	PgBoolArray        = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.BoolArray", pg.BoolArray)
	PgByteaArray       = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.ByteaArray", pg.ByteaArray)
	PgInt2Array        = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Int2Array", pg.Int2Array)
	PgInt4Array        = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Int4Array", pg.Int4Array)
	PgTextArray        = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.TextArray", pg.TextArray)
	PgBPCharArray      = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.BPCharArray", pg.BPCharArray)
	PgVarcharArray     = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.VarcharArray", pg.VarcharArray)
	PgInt8Array        = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Int8Array", pg.Int8Array)
	PgFloat4Array      = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Float4Array", pg.Float4Array)
	PgFloat8Array      = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Float8Array", pg.Float8Array)
	PgACLItem          = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.ACLItem", pg.ACLItem)
	PgACLItemArray     = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.ACLItemArray", pg.ACLItemArray)
	PgInetArray        = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.InetArray", pg.InetArray)
	PgMacaddrArray     = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.MacaddrArray", pg.MacaddrArray)
	PgBPChar           = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.BPChar", pg.BPChar)
	PgVarchar          = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Varchar", pg.Varchar)
	PgDate             = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Date", pg.Date)
	PgTime             = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Time", pg.Time)
	PgTimestamp        = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Timestamp", pg.Timestamp)
	PgTimestampArray   = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.TimestampArray", pg.TimestampArray)
	PgDateArray        = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.DateArray", pg.DateArray)
	PgTimestamptz      = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Timestamptz", pg.Timestamptz)
	PgTimestamptzArray = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.TimestamptzArray", pg.TimestamptzArray)
	PgInterval         = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Interval", pg.Interval)
	PgNumericArray     = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.NumericArray", pg.NumericArray)
	PgBit              = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Bit", pg.Bit)
	PgVarbit           = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Varbit", pg.Varbit)
	PgVoid             = &VoidType{}
	PgNumeric          = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Numeric", pg.Numeric)
	PgRecord           = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Record", pg.Record)
	PgUUID             = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.UUID", pg.UUID)
	PgUUIDArray        = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.UUIDArray", pg.UUIDArray)
	PgJSONB            = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.JSONB", pg.JSONB)
	PgJSONBArray       = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.JSONBArray", pg.JSONBArray)
	PgInt4range        = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Int4range", pg.Int4range)
	PgNumrange         = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Numrange", pg.Numrange)
	PgTsrange          = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Tsrange", pg.Tsrange)
	PgTstzrange        = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Tstzrange", pg.Tstzrange)
	PgDaterange        = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Daterange", pg.Daterange)
	PgInt8range        = MustParseKnownType("github.com/jackc/pgx/v5/pgtype.Int8range", pg.Int8range)
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

var knownTypesByOID = map[uint32]knownGoType{
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
