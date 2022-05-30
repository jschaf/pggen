-- name: SearchScreenshots :many
SELECT
  ss.id,
  array_agg(bl) AS blocks
FROM screenshots ss
  JOIN blocks bl ON bl.screenshot_id = ss.id
WHERE bl.body LIKE pggen.arg('Body') || '%'
GROUP BY ss.id
ORDER BY ss.id
LIMIT pggen.arg('Limit') OFFSET pggen.arg('Offset');

-- name: SearchScreenshotsOneCol :many
SELECT
  array_agg(bl) AS blocks
FROM screenshots ss
  JOIN blocks bl ON bl.screenshot_id = ss.id
WHERE bl.body LIKE pggen.arg('Body') || '%'
GROUP BY ss.id
ORDER BY ss.id
LIMIT pggen.arg('Limit') OFFSET pggen.arg('Offset');

-- name: InsertScreenshotBlocks :one
WITH screens AS (
  INSERT INTO screenshots (id) VALUES (pggen.arg('ScreenshotID'))
    ON CONFLICT DO NOTHING
)
INSERT
INTO blocks (screenshot_id, body)
VALUES (pggen.arg('ScreenshotID'), pggen.arg('Body'))
RETURNING id, screenshot_id, body;


-- name: ArraysInput :one
SELECT pggen.arg('arrays')::arrays;
