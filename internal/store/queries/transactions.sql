-- name: CreateTransaction :one
INSERT INTO transactions
    (user_id, account_id, date, amount_cents, payee, notes, category_id,
     transfer_group_id, import_batch_id, dedup_hash)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: TransactionByID :one
SELECT * FROM transactions WHERE id = $2 AND user_id = $1;

-- name: ListTransactions :many
-- Filters are optional: empty/NULL parameters match everything. Rows join
-- category and account names for display.
SELECT t.id, t.account_id, t.date, t.amount_cents, t.payee, t.notes,
       t.category_id, t.transfer_group_id, t.import_batch_id,
       t.created_at, t.updated_at,
       c.name AS category_name, c.kind AS category_kind, c.color AS category_color,
       a.name AS account_name, a.type AS account_type, a.currency AS account_currency
FROM transactions t
LEFT JOIN categories c ON c.id = t.category_id
JOIN accounts a ON a.id = t.account_id
WHERE t.user_id = @user_id
  AND (sqlc.narg('from_date')::date IS NULL OR t.date >= sqlc.narg('from_date'))
  AND (sqlc.narg('to_date')::date IS NULL OR t.date <= sqlc.narg('to_date'))
  AND (sqlc.narg('account_id')::uuid IS NULL OR t.account_id = sqlc.narg('account_id'))
  AND (sqlc.narg('category_id')::uuid IS NULL OR t.category_id = sqlc.narg('category_id'))
  AND (NOT @uncategorized::boolean OR (t.category_id IS NULL AND t.transfer_group_id IS NULL))
  AND (@search::text = '' OR t.payee ILIKE '%' || @search || '%' OR t.notes ILIKE '%' || @search || '%')
ORDER BY t.date DESC, t.created_at DESC
LIMIT @row_limit OFFSET @row_offset;

-- name: CountTransactions :one
SELECT count(*)
FROM transactions t
WHERE t.user_id = @user_id
  AND (sqlc.narg('from_date')::date IS NULL OR t.date >= sqlc.narg('from_date'))
  AND (sqlc.narg('to_date')::date IS NULL OR t.date <= sqlc.narg('to_date'))
  AND (sqlc.narg('account_id')::uuid IS NULL OR t.account_id = sqlc.narg('account_id'))
  AND (sqlc.narg('category_id')::uuid IS NULL OR t.category_id = sqlc.narg('category_id'))
  AND (NOT @uncategorized::boolean OR (t.category_id IS NULL AND t.transfer_group_id IS NULL))
  AND (@search::text = '' OR t.payee ILIKE '%' || @search || '%' OR t.notes ILIKE '%' || @search || '%');

-- name: UpdateTransaction :one
UPDATE transactions
SET date = $3, amount_cents = $4, payee = $5, notes = $6, category_id = $7,
    dedup_hash = $8, updated_at = now()
WHERE id = $2 AND user_id = $1
RETURNING *;

-- name: DeleteTransaction :execrows
-- Deleting a transfer leg removes both legs; the subquery resolves the group
-- (NULL for plain transactions, so only the row itself matches).
DELETE FROM transactions
WHERE transactions.user_id = $1
  AND (transactions.id = $2
       OR (transactions.transfer_group_id IS NOT NULL AND transactions.transfer_group_id = (
             SELECT tg.transfer_group_id FROM transactions tg
             WHERE tg.id = $2 AND tg.user_id = $1)));

-- name: FindByDedupHashes :many
-- Existing transactions in an account matching any of the given hashes, for
-- soft duplicate warnings during CSV import.
SELECT id, date, amount_cents, payee, dedup_hash
FROM transactions
WHERE user_id = $1 AND account_id = $2 AND dedup_hash = ANY($3::bytea[]);
