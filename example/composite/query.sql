-- name: SearchScreenshots :many
SELECT
  screenshots.id,
  array_agg(blocks) AS blocks
FROM screenshots
  JOIN blocks ON blocks.screenshot_id = screenshots.id
WHERE  blocks.body LIKE pggen.arg('Body') || '%'
GROUP BY screenshots.id
ORDER BY id
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
