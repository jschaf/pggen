-- name: ParamArrayInt :one
SELECT pggen.arg('ints')::bigint[];

-- name: ParamNested1 :one
SELECT pggen.arg('dimensions')::dimensions;

-- name: ParamNested2 :one
SELECT pggen.arg('image')::product_image_type;

-- name: ParamNested2Array :one
SELECT pggen.arg('images')::product_image_type[];

-- name: ParamNested3 :one
SELECT pggen.arg('image_set')::product_image_set_type;

-- name: ParamArrayIntWithDefaultValue :one
SELECT pggen.arg('ints', null::bigint[])::bigint[];
