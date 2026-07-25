-- name: UpsertContributionRoom :one
INSERT INTO contribution_room (user_id, account_type, tax_year, room_cents, notes)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id, account_type, tax_year)
DO UPDATE SET room_cents = EXCLUDED.room_cents, notes = EXCLUDED.notes
RETURNING *;

-- name: ListContributionRoom :many
SELECT * FROM contribution_room WHERE user_id = $1 AND tax_year = $2;

-- name: ContributionTotals :many
-- Calendar-year contributions and withdrawals per registered account type,
-- derived from transactions. Transfers between two accounts of the SAME
-- type (e.g. TFSA -> TFSA) are excluded - moving money within a type is not
-- a new contribution - while cross-type transfers (chequing -> TFSA) count.
SELECT a.type AS account_type,
       coalesce(sum(t.amount_cents) FILTER (WHERE t.amount_cents > 0), 0)::bigint AS contributed_cents,
       coalesce(-sum(t.amount_cents) FILTER (WHERE t.amount_cents < 0), 0)::bigint AS withdrawn_cents
FROM transactions t
JOIN accounts a ON a.id = t.account_id
WHERE t.user_id = $1
  AND a.type IN ('rrsp', 'tfsa', 'fhsa')
  AND t.date >= make_date($2, 1, 1)
  AND t.date < make_date($2 + 1, 1, 1)
  AND NOT EXISTS (
      SELECT 1 FROM transactions peer
      JOIN accounts pa ON pa.id = peer.account_id
      WHERE peer.transfer_group_id = t.transfer_group_id
        AND peer.id <> t.id
        AND pa.type = a.type)
GROUP BY a.type;
