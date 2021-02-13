-- name: Nested3 :many
SELECT ROW (ROW ('item_name', ROW ('sku_id')::sku)::inventory_item, 88)::qux;
