package enums

import (
	"context"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/jschaf/pggen/internal/errs"
	"github.com/jschaf/pggen/internal/pgtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net"
	"testing"
	"time"
)

func TestNewQuerier_FindAllDevices(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()

	q := NewQuerier(conn)
	mac, _ := net.ParseMAC("00:00:5e:00:53:01")

	insertDevice(t, q, mac, DeviceTypeIot)

	t.Run("FindAllDevices", func(t *testing.T) {
		devices, err := q.FindAllDevices(ctx)
		require.NoError(t, err)
		assert.Equal(t,
			[]FindAllDevicesRow{
				{Mac: pgtype.Macaddr{Addr: mac, Status: pgtype.Present}, Type: DeviceTypeIot},
			},
			devices,
		)
	})

	t.Run("FindAllDevicesScan", func(t *testing.T) {
		batch := &pgx.Batch{}
		q.FindAllDevicesBatch(batch)
		results := conn.SendBatch(context.Background(), batch)
		defer errs.CaptureT(t, results.Close, "close batch results")
		devices, err := q.FindAllDevicesScan(results)
		require.NoError(t, err)
		assert.Equal(t,
			[]FindAllDevicesRow{
				{Mac: pgtype.Macaddr{Addr: mac, Status: pgtype.Present}, Type: DeviceTypeIot},
			},
			devices,
		)
	})
}

var allDeviceTypes = []DeviceType{
	DeviceTypeUndefined,
	DeviceTypePhone,
	DeviceTypeLaptop,
	DeviceTypeIpad,
	DeviceTypeDesktop,
	DeviceTypeIot,
}

func TestNewQuerier_FindOneDeviceArray(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()

	q := NewQuerier(conn)

	t.Run("FindOneDeviceArray", func(t *testing.T) {
		devices, err := q.FindOneDeviceArray(ctx)
		require.NoError(t, err)
		assert.Equal(t, allDeviceTypes, devices)
	})

	t.Run("FindOneDeviceArrayBatch", func(t *testing.T) {
		batch := &pgx.Batch{}
		q.FindOneDeviceArrayBatch(batch)
		results := conn.SendBatch(context.Background(), batch)
		defer errs.CaptureT(t, results.Close, "close batch results")
		devices, err := q.FindOneDeviceArrayScan(results)
		require.NoError(t, err)
		assert.Equal(t, allDeviceTypes, devices)
	})
}

func TestNewQuerier_FindManyDeviceArray(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()

	q := NewQuerier(conn)

	t.Run("FindManyDeviceArray", func(t *testing.T) {
		devices, err := q.FindManyDeviceArray(ctx)
		require.NoError(t, err)
		assert.Equal(t, [][]DeviceType{allDeviceTypes[3:], allDeviceTypes}, devices)
	})

	t.Run("FindManyDeviceArrayBatch", func(t *testing.T) {
		batch := &pgx.Batch{}
		q.FindManyDeviceArrayBatch(batch)
		results := conn.SendBatch(context.Background(), batch)
		defer errs.CaptureT(t, results.Close, "close batch results")
		devices, err := q.FindManyDeviceArrayScan(results)
		require.NoError(t, err)
		assert.Equal(t, [][]DeviceType{allDeviceTypes[3:], allDeviceTypes}, devices)
	})
}

func TestNewQuerier_FindManyDeviceArrayWithNum(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()

	q := NewQuerier(conn)
	one, two := int32(1), int32(2)

	t.Run("FindManyDeviceArrayWithNum", func(t *testing.T) {
		devices, err := q.FindManyDeviceArrayWithNum(ctx)
		require.NoError(t, err)
		assert.Equal(t, []FindManyDeviceArrayWithNumRow{
			{Num: &one, DeviceTypes: allDeviceTypes[3:]},
			{Num: &two, DeviceTypes: allDeviceTypes},
		}, devices)
	})

	t.Run("FindManyDeviceArrayWithNumBatch", func(t *testing.T) {
		batch := &pgx.Batch{}
		q.FindManyDeviceArrayWithNumBatch(batch)
		results := conn.SendBatch(context.Background(), batch)
		defer errs.CaptureT(t, results.Close, "close batch results")
		devices, err := q.FindManyDeviceArrayWithNumScan(results)
		require.NoError(t, err)
		assert.Equal(t, []FindManyDeviceArrayWithNumRow{
			{Num: &one, DeviceTypes: allDeviceTypes[3:]},
			{Num: &two, DeviceTypes: allDeviceTypes},
		}, devices)
	})
}

func TestNewQuerier_EnumInsideComposite(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()

	q := NewQuerier(conn)
	mac, _ := net.ParseMAC("08:00:2b:01:02:03")

	t.Run("EnumInsideComposite", func(t *testing.T) {
		device, err := q.EnumInsideComposite(ctx)
		require.NoError(t, err)
		assert.Equal(t,
			Device{Mac: pgtype.Macaddr{Addr: mac, Status: pgtype.Present}, Type: DeviceTypePhone},
			device,
		)
	})

	t.Run("EnumInsideCompositeBatch", func(t *testing.T) {
		batch := &pgx.Batch{}
		q.EnumInsideCompositeBatch(batch)
		results := conn.SendBatch(context.Background(), batch)
		defer errs.CaptureT(t, results.Close, "close batch results")
		device, err := q.EnumInsideCompositeScan(results)
		require.NoError(t, err)
		assert.Equal(t,
			Device{Mac: pgtype.Macaddr{Addr: mac, Status: pgtype.Present}, Type: DeviceTypePhone},
			device,
		)
	})
}

func insertDevice(t *testing.T, q *DBQuerier, mac net.HardwareAddr, device DeviceType) {
	t.Helper()
	_, err := q.InsertDevice(context.Background(),
		pgtype.Macaddr{Addr: mac, Status: pgtype.Present},
		device,
	)
	require.NoError(t, err)
}
