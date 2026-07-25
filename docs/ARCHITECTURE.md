# getbud — Architecture

getbud is a self-hosted budgeting app for Canadians. It ships as a single
binary (Go server with the built web UI embedded) plus a Postgres database.
The architecture deliberately mirrors
[waypoint](https://github.com/ctrl-research/waypoint).

## High-level design

```
┌──────────────────────────────┐
│  Browser (React SPA)         │
│  Vite + TS + Tailwind        │
│  Apache ECharts for charts   │
└──────────────┬───────────────┘
               │ HTTPS (JSON API + static assets)
┌──────────────▼───────────────┐
│  Go server (single binary)   │
│  ├─ /api/v1/*  JSON API      │
│  ├─ /auth/*    OIDC + local  │
│  └─ /*         embedded SPA  │
└──────────────┬───────────────┘
               │ pgx pool
┌──────────────▼───────────────┐
│  PostgreSQL 16               │
└──────────────────────────────┘
```

## Backend (Go)

- Standard-library `net/http` with the Go 1.22+ `ServeMux` method-and-pattern
  routing — no router framework.
- **Layout**: `cmd/server` (entrypoint + `seed` subcommand),
  `internal/config` (env-only configuration), `internal/auth`,
  `internal/server` (one handler file per resource), `internal/store`
  (data access), `internal/csvimport` (pure CSV parsing), `internal/webui`
  (SPA embedding), `migrations/`.
- **Data access**: SQL lives in `internal/store/queries/*.sql` and is
  compiled by [sqlc](https://sqlc.dev) into `internal/store/sqlcgen`
  (generated, never hand-edited; CI fails if it's stale). Hand-written
  wrappers in `internal/store` own transactions and error translation
  (`ErrNotFound`, `ErrConflict`).
- **Migrations**: [goose](https://github.com/pressly/goose) SQL files
  embedded via `go:embed`, applied automatically at server startup.
  Append-only, `NNNNN_description.sql`.
- **Auth**: Google OIDC and generic OIDC (authorization-code flow with PKCE +
  state), plus optional local email/password for development. The ID token
  is used only at the callback; the app mints its own opaque session token,
  stores only its SHA-256 hash, and sets it as an HttpOnly SameSite=Lax
  cookie (30-day TTL, hourly GC). First user becomes admin; later signups
  are gated by an email allowlist.
- **Privacy model**: every domain table carries `user_id` and every query
  filters on it. There are no cross-user code paths; the store tests include
  explicit isolation tests.
- **Money**: integer cents (`bigint`) everywhere. Floats never touch
  amounts — parsing goes through strings on the client and `int64` on the
  server.
- **Aggregation**: all reporting (Sankey flows, monthly trends, net-worth
  carry-forward, contribution totals) is computed in SQL
  (`internal/store/queries/reports.sql`); Go only reshapes rows into
  chart-friendly JSON.

## Key domain decisions

- **Transfers** between a user's own accounts are two transaction rows
  sharing a `transfer_group_id`, created and deleted atomically. Every
  income/expense aggregate adds `AND transfer_group_id IS NULL`.
- **Balances**: cash accounts = opening balance + transaction sum;
  investment accounts (RRSP/TFSA/FHSA/non-registered) = latest balance
  snapshot. Net worth carries each account's latest snapshot forward per
  month.
- **Contribution tracking** derives from transactions in registered
  accounts, excluding same-type transfers (TFSA → TFSA is not a new
  contribution). Room is user-entered; published annual limits are hints
  only.
- **CSV import** is stateless on the server: the browser holds the file
  across the wizard and re-uploads on commit, guarded by a SHA-256 of the
  file contents (409 on mismatch). Duplicate detection hashes
  `date|amount|normalized payee` — a soft warning, never a unique
  constraint, because identical same-day purchases are legitimate. Reverting
  an import deletes its `import_batches` row; the FK cascade removes its
  transactions.

## Frontend (web/)

- React 19 + TypeScript, Vite, Tailwind CSS v4, TanStack Router (code-based
  routes) and TanStack Query.
- Charts via Apache ECharts (`echarts/core` imports, tree-shaken; a small
  `useECharts` hook handles theme switching and resize). Chart colors come
  from a validated colorblind-safe palette (`web/src/charts/tokens.ts`);
  categories keep stable identity colors set at seed time and editable in
  Settings.
- Auth is cookie-based; no tokens in JS. During development Vite proxies
  `/api` and `/auth` to the Go server.
- **Embedding**: `internal/webui` has a build-tag split — plain `go build`
  serves a "run the Vite dev server" stub, while `make build` / the
  Dockerfile copy `web/dist` into the package and build with
  `-tags embedwebui`, serving real files with an `index.html` fallback for
  client-side routes.

## Testing

- **Store tests** run against real Postgres (no DB mocks) via
  `internal/store/storetest`, which creates a per-package database and
  truncates between tests. Skipped unless `GETBUD_TEST_DATABASE_URL` is set;
  `make test-db` and CI set it.
- **Handler tests** verify every `/api/v1` route rejects unauthenticated
  requests.
- **Unit tests** cover the CSV parser/mapping heuristics with realistic bank
  fixtures, and dedup-hash normalization.
