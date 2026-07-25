-- name: CreateAccount :one
INSERT INTO accounts (user_id, name, type, currency, institution, opening_balance_cents)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListAccounts :many
-- Balance is derived per account kind: investment-style accounts use the
-- latest snapshot, cash accounts sum their transactions on top of the
-- opening balance.
SELECT sqlc.embed(a),
       CASE WHEN a.type IN ('rrsp', 'tfsa', 'fhsa', 'non_registered')
            THEN coalesce(s.balance_cents, 0)
            ELSE a.opening_balance_cents + coalesce(t.sum_cents, 0)
       END::bigint AS balance_cents
FROM accounts a
LEFT JOIN LATERAL (
    SELECT bs.balance_cents FROM balance_snapshots bs
    WHERE bs.account_id = a.id
    ORDER BY bs.as_of DESC LIMIT 1
) s ON true
LEFT JOIN LATERAL (
    SELECT sum(tx.amount_cents) AS sum_cents FROM transactions tx
    WHERE tx.account_id = a.id
) t ON true
WHERE a.user_id = $1
ORDER BY a.is_archived, a.created_at;

-- name: AccountByID :one
SELECT * FROM accounts WHERE id = $2 AND user_id = $1;

-- name: UpdateAccount :one
UPDATE accounts
SET name = $3, type = $4, currency = $5, institution = $6,
    opening_balance_cents = $7, is_archived = $8
WHERE id = $2 AND user_id = $1
RETURNING *;

-- name: DeleteAccount :execrows
DELETE FROM accounts WHERE id = $2 AND user_id = $1;

-- name: CountAccountTransactions :one
SELECT count(*) FROM transactions WHERE account_id = $2 AND user_id = $1;
