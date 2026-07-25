# getbud — Data Model

Postgres schema, evolved via goose migrations in `migrations/`. This
document is the design reference; the migrations are the source of truth.

## Entity overview

```
users ─┬─ sessions
       ├─ accounts ─┬─ transactions ── import_batches
       │            └─ balance_snapshots
       ├─ categories (referenced by transactions)
       └─ contribution_room
```

Every domain table carries `user_id` with `ON DELETE CASCADE`; every query
filters on it. All money columns are `bigint` **integer cents**.

## Conventions

- **Sign convention**: `amount_cents > 0` is money into the account,
  `< 0` is money out. Category `kind` carries income/expense semantics.
- Dates are `date` (no times) — budgeting is day-granular.

## Core tables

### users / sessions (M2)

Copied from waypoint: `users` (uuid PK, citext unique email, `google_sub` /
`oidc_sub` / `password_hash` — at least one required by a CHECK constraint,
`is_admin`), `sessions` (`token_hash bytea` PK storing only the SHA-256 of
the cookie token, expiry + last-seen). Creating a user also seeds the
default category set in the same transaction.

### accounts (M3)

| column | type | notes |
|---|---|---|
| `type` | enum | `chequing, savings, credit_card, rrsp, tfsa, fhsa, non_registered, other` |
| `currency` | char(3) | default `CAD` |
| `opening_balance_cents` | bigint | basis for derived cash balances |
| `is_archived` | boolean | archived accounts keep history, leave pickers |

Cash balance = opening balance + `SUM(transactions.amount_cents)`.
Investment types (`rrsp/tfsa/fhsa/non_registered`) use the latest
`balance_snapshots` row instead. Unique per user by name.

### categories (M4)

Flat (no hierarchy), per user, `kind` enum (`income`/`expense`), a stable
hex `color` used across all charts, `is_archived`. Unique per (user, kind,
name). A default set of 6 income + 17 expense categories is seeded for
every new user.

### import_batches + transactions (M5)

`import_batches`: one row per CSV import — filename, the jsonb column
`mapping` (prefills the account's next import), row/imported/skipped counts.

`transactions`:

| column | notes |
|---|---|
| `amount_cents` | signed bigint |
| `category_id` | nullable FK, `ON DELETE SET NULL` — NULL means uncategorized (or a transfer leg) |
| `transfer_group_id` | both legs of a transfer share one uuid; aggregates exclude rows where set |
| `import_batch_id` | FK with `ON DELETE CASCADE` — reverting an import deletes the batch row and its transactions follow |
| `dedup_hash` | `sha256(date\|amount\|normalized payee)`, **non-unique** index; soft duplicate warnings only |

Indexes: (user, date), (account, date), (user, category), (account,
dedup_hash), batch, and a partial index on `transfer_group_id`.

### balance_snapshots (M6)

Point-in-time balances per account (`UNIQUE (account_id, as_of)`; PUT
upserts). Source for investment balances and the net-worth report, which
carries each account's latest snapshot on or before month-end forward.

### contribution_room (M7)

User-entered available room per (user, `account_type ∈ rrsp/tfsa/fhsa`,
`tax_year`). Contributions/withdrawals are **not stored** — they are derived
from transactions in accounts of that type for the calendar year, excluding
transfers whose peer account has the same type (TFSA → TFSA moves aren't
new contributions). The published annual limits live in Go as UI hints
only.

## Reporting queries

All aggregation is SQL (`internal/store/queries/reports.sql`):

- **Summary** — sign-based income/expense totals + uncategorized count over
  a range, transfers excluded.
- **Category totals** — net flow per category (feeds the Sankey and the
  donut); uncategorized rows bucket by sign.
- **Trends** — `date_trunc('month')` × category matrix.
- **Net worth** — `generate_series` of months × `LATERAL` latest-snapshot
  lookup, summed by account type.

## Known simplifications

- RRSP first-60-days rule is not modelled (calendar-year attribution).
- Reports assume a single currency; no FX conversion.
- No per-holding investment tracking (balances only), by design.
