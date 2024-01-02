package composite

import (
	"context"
	"github.com/jackc/pgtype"
	"github.com/jschaf/pggen/internal/difftest"
	"github.com/jschaf/pggen/internal/pgtest"
	"github.com/jschaf/pggen/internal/ptrs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewQuerier_SearchScreenshots(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()

	q := NewQuerier(conn)
	screenshotID := 99
	screenshot1 := insertScreenshotBlock(t, q, screenshotID, "body1")
	screenshot2 := insertScreenshotBlock(t, q, screenshotID, "body2")
	want := []SearchScreenshotsRow{
		{
			ID: screenshotID,
			Blocks: []Blocks{
				{
					ID:           screenshot1.ID,
					ScreenshotID: screenshotID,
					Body:         screenshot1.Body,
				},
				{
					ID:           screenshot2.ID,
					ScreenshotID: screenshotID,
					Body:         screenshot2.Body,
				},
			},
		},
	}

	t.Run("SearchScreenshots", func(t *testing.T) {
		rows, err := q.SearchScreenshots(context.Background(), SearchScreenshotsParams{
			Body:   "body",
			Limit:  5,
			Offset: 0,
		})
		require.NoError(t, err)
		assert.Equal(t, want, rows)
	})

	t.Run("SearchScreenshotsOneCol", func(t *testing.T) {
		rows, err := q.SearchScreenshotsOneCol(context.Background(), SearchScreenshotsOneColParams{
			Body:   "body",
			Limit:  5,
			Offset: 0,
		})
		require.NoError(t, err)
		assert.Equal(t, [][]Blocks{want[0].Blocks}, rows)
	})
}

func TestNewQuerier_ArraysInput(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()

	q := NewQuerier(conn)

	t.Run("ArraysInput", func(t *testing.T) {
		want := Arrays{
			Texts:  []string{"foo", "bar"},
			Int8s:  []*int{ptrs.Int(1), ptrs.Int(2), ptrs.Int(3)},
			Bools:  []bool{true, true, false},
			Floats: []*float64{ptrs.Float64(33.3), ptrs.Float64(66.6)},
		}
		got, err := q.ArraysInput(context.Background(), want)
		require.NoError(t, err)
		difftest.AssertSame(t, want, got)
	})
}

func TestNewQuerier_UserEmails(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()

	q := NewQuerier(conn)

	got, err := q.UserEmails(context.Background())
	require.NoError(t, err)
	want := UserEmail{
		ID:    "foo",
		Email: pgtype.Text{String: "bar@example.com", Status: pgtype.Present},
	}
	difftest.AssertSame(t, want, got)
}

func insertScreenshotBlock(t *testing.T, q *DBQuerier, screenID int, body string) InsertScreenshotBlocksRow {
	t.Helper()
	row, err := q.InsertScreenshotBlocks(context.Background(), screenID, body)
	require.NoError(t, err, "insert screenshot blocks")
	return row
}
