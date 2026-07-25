-- name: UpsertSnapshot :one
INSERT INTO balance_snapshots (user_id, account_id, as_of, balance_cents)
VALUES ($1, $2, $3, $4)
ON CONFLICT (account_id, as_of)
DO UPDATE SET balance_cents = EXCLUDED.balance_cents
RETURNING *;

-- name: ListSnapshots :many
SELECT * FROM balance_snapshots
WHERE user_id = $1 AND account_id = $2
ORDER BY as_of DESC;

-- name: DeleteSnapshot :execrows
DELETE FROM balance_snapshots WHERE id = $2 AND user_id = $1;
