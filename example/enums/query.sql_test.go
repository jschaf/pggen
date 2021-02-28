package enums

import (
	"context"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/jschaf/pggen/internal/pgtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net"
	"testing"
)

func TestQuerier(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()

	q := NewQuerier(conn)
	ctx := context.Background()
	mac, _ := net.ParseMAC("00:00:5e:00:53:01")

	{
		_, err := q.InsertDevice(ctx, pgtype.Macaddr{
			Addr:   mac,
			Status: pgtype.Present,
		}, DeviceTypeDesktop)
		if err != nil {
			t.Fatal(err)
		}
	}

	{
		devices, err := q.FindAllDevices(ctx)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, []FindAllDevicesRow{{
			Mac:  pgtype.Macaddr{Addr: mac, Status: pgtype.Present},
			Type: DeviceTypeDesktop,
		}}, devices)
	}

	allDeviceTypes := []DeviceType{
		DeviceTypeUndefined,
		DeviceTypePhone,
		DeviceTypeLaptop,
		DeviceTypeIpad,
		DeviceTypeDesktop,
		DeviceTypeIot,
	}

	{
		devices, err := q.FindOneDeviceArray(ctx)
		require.Nil(t, err)
		assert.Equal(t, allDeviceTypes, devices)
	}

	{
		batch := &pgx.Batch{}
		q.FindOneDeviceArrayBatch(batch)
		results := conn.SendBatch(ctx, batch)
		devices, err := q.FindOneDeviceArrayScan(results)
		require.Nil(t, err)
		require.Nil(t, results.Close())
		assert.Equal(t, allDeviceTypes, devices)
	}

	{
		devices, err := q.FindOneDeviceArray(ctx)
		require.Nil(t, err)
		assert.Equal(t, allDeviceTypes, devices)
	}

	{
		batch := &pgx.Batch{}
		q.FindOneDeviceArrayBatch(batch)
		results := conn.SendBatch(ctx, batch)
		devices, err := q.FindOneDeviceArrayScan(results)
		require.Nil(t, err)
		require.Nil(t, results.Close())
		assert.Equal(t, allDeviceTypes, devices)
	}

	{
		devices, err := q.FindManyDeviceArray(ctx)
		require.Nil(t, err)
		assert.Equal(t, [][]DeviceType{allDeviceTypes[3:], allDeviceTypes}, devices)
	}

	{
		batch := &pgx.Batch{}
		q.FindManyDeviceArrayBatch(batch)
		results := conn.SendBatch(ctx, batch)
		devices, err := q.FindManyDeviceArrayScan(results)
		require.Nil(t, err)
		require.Nil(t, results.Close())
		assert.Equal(t, [][]DeviceType{allDeviceTypes[3:], allDeviceTypes}, devices)
	}

	{
		devices, err := q.FindManyDeviceArrayWithNum(ctx)
		require.Nil(t, err)
		assert.Equal(t, []FindManyDeviceArrayWithNumRow{
			{Num: pgtype.Int4{Int: 1, Status: pgtype.Present}, DeviceTypes: allDeviceTypes[3:]},
			{Num: pgtype.Int4{Int: 2, Status: pgtype.Present}, DeviceTypes: allDeviceTypes},
		}, devices)
	}

	{
		batch := &pgx.Batch{}
		q.FindManyDeviceArrayWithNumBatch(batch)
		results := conn.SendBatch(ctx, batch)
		devices, err := q.FindManyDeviceArrayWithNumScan(results)
		require.Nil(t, err)
		require.Nil(t, results.Close())
		assert.Equal(t, []FindManyDeviceArrayWithNumRow{
			{Num: pgtype.Int4{Int: 1, Status: pgtype.Present}, DeviceTypes: allDeviceTypes[3:]},
			{Num: pgtype.Int4{Int: 2, Status: pgtype.Present}, DeviceTypes: allDeviceTypes},
		}, devices)
	}
}
