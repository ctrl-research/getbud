# AGENTS.md

## Purpose

getbud is a self-hosted budgeting app: Go API + Postgres backend with an
embedded React SPA. Canadian-focused — registered accounts (RRSP/TFSA/FHSA),
contribution-room tracking, Sankey/trend/net-worth reports, CSV import.
The architecture deliberately mirrors `ctrl-research/waypoint`.

## Tech stack

- **Go** (std-lib `net/http` ServeMux — no router framework), **pgx/v5**,
  **sqlc** (generated code in `internal/store/sqlcgen`, never hand-edited),
  **goose** migrations (embedded, run at server startup)
- **Postgres 16**
- **React 19 + TypeScript + Vite + Tailwind v4 + TanStack Router/Query**,
  charts via **Apache ECharts** (`echarts/core`, tree-shaken)
- **Docker** 3-stage build; SPA embedded via `go build -tags embedwebui`

## Key commands

```sh
make db        # postgres via docker compose (host port 5433)
make seed      # create/reset dev@getbud.local / getbud-dev
make run       # API on :8081 with GETBUD_LOCAL_AUTH=true
make web       # vite dev server on :5173 (proxies /api, /auth)
make generate  # sqlc generate — run after editing internal/store/queries/*.sql
make test      # go vet + unit tests
make test-db   # + postgres-backed store tests (requires make db)
make build     # production binary with embedded UI -> bin/getbud
```

Local ports are shifted (+1) to coexist with waypoint on the same machine:
postgres 5433, app 8081.

## Conventions

- **Money is integer cents (`bigint`) everywhere.** Never floats. Sign
  convention: `amount_cents > 0` = inflow, `< 0` = outflow.
- **Every domain table and every query is scoped by `user_id`** — user data
  is fully private. New store methods must take and filter on the user id;
  add a cross-user isolation test when adding a store.
- Transfers are two transaction rows sharing a `transfer_group_id`; all
  income/expense aggregates exclude rows where it is set.
- Reverting a CSV import deletes the `import_batches` row; the FK cascade
  removes its transactions. No other code path deletes batches.
- Aggregation happens in SQL (`internal/store/queries/reports.sql`), not in
  Go or the browser.
- `.tool-versions` pins Go/Node versions (asdf/mise).
- Store tests use `internal/store/storetest` against real Postgres (no DB
  mocks); they skip unless `GETBUD_TEST_DATABASE_URL` is set.
- Migrations are append-only `NNNNN_description.sql` goose files.
- Versioning: SemVer, bare `X.Y.Z` (no `v` prefix).
- Branch protection: all changes via PR with review; never push to `main`.
