-- name: ReportSummary :one
-- Dashboard tiles for a date range. Income/expense are sign-based over
-- non-transfer transactions, so uncategorized rows still count.
SELECT
    coalesce(sum(t.amount_cents) FILTER (WHERE t.amount_cents > 0), 0)::bigint AS income_cents,
    coalesce(-sum(t.amount_cents) FILTER (WHERE t.amount_cents < 0), 0)::bigint AS expense_cents,
    count(*) FILTER (WHERE t.category_id IS NULL)::bigint AS uncategorized_count
FROM transactions t
WHERE t.user_id = $1 AND t.date >= $2 AND t.date <= $3
  AND t.transfer_group_id IS NULL;

-- name: ReportCategoryTotals :many
-- Net flow per category over a range (transfers excluded). Feeds the Sankey
-- and the category breakdown. Uncategorized rows come back with a NULL
-- category id; the server buckets them by sign.
SELECT c.id AS category_id,
       coalesce(c.name, 'Uncategorized') AS category_name,
       c.kind AS category_kind,
       coalesce(c.color, '') AS category_color,
       coalesce(sum(t.amount_cents) FILTER (WHERE t.amount_cents > 0), 0)::bigint AS inflow_cents,
       coalesce(-sum(t.amount_cents) FILTER (WHERE t.amount_cents < 0), 0)::bigint AS outflow_cents
FROM transactions t
LEFT JOIN categories c ON c.id = t.category_id
WHERE t.user_id = $1 AND t.date >= $2 AND t.date <= $3
  AND t.transfer_group_id IS NULL
GROUP BY c.id, c.name, c.kind, c.color;

-- name: ReportTrends :many
-- Monthly net flow per category (transfers excluded) for trend charts.
SELECT date_trunc('month', t.date)::date AS month,
       c.id AS category_id,
       coalesce(c.name, 'Uncategorized') AS category_name,
       c.kind AS category_kind,
       coalesce(c.color, '') AS category_color,
       coalesce(sum(t.amount_cents) FILTER (WHERE t.amount_cents > 0), 0)::bigint AS inflow_cents,
       coalesce(-sum(t.amount_cents) FILTER (WHERE t.amount_cents < 0), 0)::bigint AS outflow_cents
FROM transactions t
LEFT JOIN categories c ON c.id = t.category_id
WHERE t.user_id = $1 AND t.date >= $2 AND t.date <= $3
  AND t.transfer_group_id IS NULL
GROUP BY 1, c.id, c.name, c.kind, c.color
ORDER BY 1;

-- name: ReportNetWorth :many
-- Month-end net worth per account type: for each month in the range, carry
-- forward the latest snapshot on or before month end for every unarchived
-- account, and sum by type. Months before an account's first snapshot
-- contribute nothing.
SELECT m.month::date AS month,
       a.type AS account_type,
       sum(s.balance_cents)::bigint AS total_cents
FROM generate_series(date_trunc('month', $2::date), date_trunc('month', $3::date), interval '1 month') AS m(month)
CROSS JOIN accounts a
JOIN LATERAL (
    SELECT bs.balance_cents
    FROM balance_snapshots bs
    WHERE bs.account_id = a.id
      AND bs.as_of < (m.month + interval '1 month')
    ORDER BY bs.as_of DESC
    LIMIT 1
) s ON true
WHERE a.user_id = $1 AND NOT a.is_archived
GROUP BY 1, a.type
ORDER BY 1;
