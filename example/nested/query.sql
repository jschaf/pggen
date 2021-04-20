-- name: ArrayNested2 :one
SELECT
  ARRAY [
    ROW ('img2', ROW (22, 22)::dimensions)::product_image_type,
    ROW ('img3', ROW (33, 33)::dimensions)::product_image_type
    ] AS images;

-- name: Nested3 :many
SELECT
  ROW (
    'name', -- name
    ROW ('img1', ROW (11, 11)::dimensions)::product_image_type, -- orig_image
    ARRAY [ --images
      ROW ('img2', ROW (22, 22)::dimensions)::product_image_type,
      ROW ('img3', ROW (33, 33)::dimensions)::product_image_type
      ]
    )::product_image_set_type;

