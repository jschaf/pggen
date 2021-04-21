-- name: ParamNested1 :one
SELECT pggen.arg('dimensions')::dimensions;

-- name: ParamNested2 :one
SELECT pggen.arg('image')::product_image_type;

-- name: ParamNested2Array :one
SELECT pggen.arg('images')::product_image_type[];

