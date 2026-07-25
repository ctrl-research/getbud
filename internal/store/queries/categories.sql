-- name: CreateCategory :one
INSERT INTO categories (user_id, name, kind, color)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: SeedDefaultCategories :exec
-- Bulk-inserts the default category set for a new user (called inside the
-- user-create transaction).
INSERT INTO categories (user_id, name, kind, color)
SELECT @user_id, unnest(@names::text[]), unnest(@kinds::text[])::category_kind, unnest(@colors::text[])
ON CONFLICT DO NOTHING;

-- name: ListCategories :many
SELECT * FROM categories WHERE user_id = $1 ORDER BY kind, is_archived, name;

-- name: CategoryByID :one
SELECT * FROM categories WHERE id = $2 AND user_id = $1;

-- name: UpdateCategory :one
UPDATE categories
SET name = $3, color = $4, is_archived = $5
WHERE id = $2 AND user_id = $1
RETURNING *;

-- name: DeleteCategory :execrows
-- Transactions referencing the category go uncategorized via ON DELETE SET NULL.
DELETE FROM categories WHERE id = $2 AND user_id = $1;

-- name: ReassignCategory :execrows
-- Moves all transactions from one category to another (both owned by the
-- user; the target check is the WHERE EXISTS guard).
UPDATE transactions SET category_id = $3, updated_at = now()
WHERE transactions.user_id = $1 AND transactions.category_id = $2
  AND EXISTS (SELECT 1 FROM categories c WHERE c.id = $3 AND c.user_id = $1);
