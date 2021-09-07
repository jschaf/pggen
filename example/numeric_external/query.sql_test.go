package numeric_external

import (
	"context"
	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgtype"
	"github.com/jschaf/pggen/internal/pgtest"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	shopspring "github.com/jackc/pgtype/ext/shopspring-numeric"
	"testing"
)

func TestNewQuerier_FindNumerics(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()

	conn.ConnInfo()
	q := NewQuerierConfig(conn, QuerierConfig{
		DataTypes: []pgtype.DataType{
			{
				Value: &shopspring.Numeric{},
				Name:  "numeric",
				OID:   pgtype.NumericOID,
			},
		},
	})
	_, err := q.InsertNumeric(context.Background(), decimal.New(10, 0), []NumericExternalType{
		{Num: decimal.New(11, 0)},
	})
	require.NoError(t, err)
	_, err = q.InsertNumeric(context.Background(), decimal.New(20, 0), []NumericExternalType{
		{Num: decimal.New(21, 0)},
		{Num: decimal.New(22, 0)},
	})
	require.NoError(t, err)

	t.Run("FindNumerics", func(t *testing.T) {
		rows, err := q.FindNumerics(context.Background())
		require.NoError(t, err)
		want := []FindNumericsRow{
			{
				Num:    decimal.New(10, 0),
				NumArr: []NumericExternalType{{Num: decimal.New(11, 0)}},
			},
			{
				Num: decimal.New(20, 0),
				NumArr: []NumericExternalType{
					{Num: decimal.New(21, 0)},
					{Num: decimal.New(22, 0)},
				},
			},
		}
		if diff := cmp.Diff(want, rows); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})
}
