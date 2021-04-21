-- name: ParamNested1 :one
SELECT pggen.arg('dimensions')::dimensions;

-- name: ParamNested2 :one
SELECT pggen.arg('image')::product_image_type;

