package device

import (
	"context"
	"github.com/jackc/pgtype"
	"github.com/jschaf/pggen/internal/pgtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net"
	"testing"
)

func TestQuerier_FindDevicesByUser(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()
	q := NewQuerier(conn)
	ctx := context.Background()
	userID := 18
	_, err := q.InsertUser(ctx, userID, "foo")
	require.NoError(t, err)
	mac1, _ := net.ParseMAC("11:22:33:44:55:66")
	_, err = q.InsertDevice(ctx, pgtype.Macaddr{Status: pgtype.Present, Addr: mac1}, userID)
	require.NoError(t, err)

	t.Run("FindDevicesByUser", func(t *testing.T) {
		val, err := q.FindDevicesByUser(ctx, userID)
		require.NoError(t, err)
		want := []FindDevicesByUserRow{
			{
				ID:   userID,
				Name: "foo",
				MacAddrs: pgtype.MacaddrArray{
					Elements: []pgtype.Macaddr{{Addr: mac1, Status: pgtype.Present}},
					Dimensions: []pgtype.ArrayDimension{{
						Length:     1,
						LowerBound: 1,
					}},
					Status: pgtype.Present,
				},
			},
		}
		assert.Equal(t, want, val)
	})
}

func TestQuerier_CompositeUser(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()
	q := NewQuerier(conn)
	ctx := context.Background()

	userID := 18
	name := "foo"
	_, err := q.InsertUser(ctx, userID, name)
	require.NoError(t, err)

	mac1, _ := net.ParseMAC("11:22:33:44:55:66")
	mac2, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	_, err = q.InsertDevice(ctx, pgtype.Macaddr{Status: pgtype.Present, Addr: mac1}, userID)
	require.NoError(t, err)
	_, err = q.InsertDevice(ctx, pgtype.Macaddr{Status: pgtype.Present, Addr: mac2}, userID)
	require.NoError(t, err)

	t.Run("CompositeUser", func(t *testing.T) {
		users, err := q.CompositeUser(ctx)
		require.NoError(t, err)
		want := []CompositeUserRow{
			{
				Mac:  pgtype.Macaddr{Addr: mac1, Status: pgtype.Present},
				Type: DeviceTypeUndefined,
				User: User{ID: &userID, Name: &name},
			},
			{
				Mac:  pgtype.Macaddr{Addr: mac2, Status: pgtype.Present},
				Type: DeviceTypeUndefined,
				User: User{ID: &userID, Name: &name},
			},
		}
		assert.Equal(t, want, users)
	})
}

func TestQuerier_CompositeUserOne(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()
	q := NewQuerier(conn)
	ctx := context.Background()
	id := 15
	name := "qux"
	wantUser := User{ID: &id, Name: &name}

	t.Run("CompositeUserOne", func(t *testing.T) {
		got, err := q.CompositeUserOne(ctx)
		require.NoError(t, err)
		assert.Equal(t, wantUser, got)
	})
}
