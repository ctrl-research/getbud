-- name: CreateImportBatch :one
INSERT INTO import_batches (user_id, account_id, filename, mapping, row_count, imported_count, skipped_count)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListImportBatches :many
SELECT b.*, a.name AS account_name
FROM import_batches b
JOIN accounts a ON a.id = b.account_id
WHERE b.user_id = $1
ORDER BY b.created_at DESC;

-- name: LatestMappingForAccount :one
-- The last mapping used for an account prefills the next import.
SELECT mapping FROM import_batches
WHERE user_id = $1 AND account_id = $2
ORDER BY created_at DESC
LIMIT 1;

-- name: DeleteImportBatch :execrows
-- Reverting an import: the transactions FK cascades, removing its rows.
DELETE FROM import_batches WHERE id = $2 AND user_id = $1;
