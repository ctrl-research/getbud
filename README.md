# getbud

Self-hosted personal budgeting for Canadians. Track income and expenses,
registered investment accounts (RRSP / TFSA / FHSA), contribution room, and
net worth — with a Sankey cash-flow diagram, trend charts, and CSV import
from your bank.

Single Go binary + Postgres. The React UI is embedded in the binary; deploy
with `docker compose up`.

## Features

- **Accounts** — chequing, savings, credit cards, and investment accounts
  (RRSP, TFSA, FHSA, non-registered). Cash balances derive from transactions;
  investment balances from point-in-time snapshots.
- **Transactions** — manual entry, transfers between accounts (never counted
  as income/expense), inline categorization, search and filters.
- **CSV import** — upload bank/credit-card exports, auto-guessed column
  mapping (delimiter, date format, signed vs debit/credit columns), soft
  duplicate detection, and one-click revert of any import batch.
- **Contribution room** — enter your CRA room per year; contributions and
  withdrawals are derived from transactions (transfers between two accounts
  of the same type don't count as new contributions).
- **Reports** — Sankey income → expenses flow, monthly category trends,
  net-worth-over-time by account type, category breakdown.
- **Auth** — Google sign-in, any generic OIDC provider (Authentik, Keycloak,
  …), or local email/password for development. First user becomes admin;
  further signups are restricted to an email allowlist. Multi-user with
  fully private per-user data.

## Quick start (Docker)

```sh
cp .env.example .env    # set POSTGRES_PASSWORD (and auth vars for SSO)
docker compose up -d
```

Prebuilt multi-arch images (amd64/arm64) are published on every release if
you'd rather not build from source:

```sh
docker pull ghcr.io/ctrl-research/getbud:latest   # or a specific X.Y.Z
```

The app listens on http://localhost:8081. With `GETBUD_LOCAL_AUTH=true`
(compose default) create a login user:

```sh
docker compose exec app getbud seed   # dev@getbud.local / getbud-dev
```

For real deployments set `GETBUD_BASE_URL` and the Google or OIDC variables
in `.env` — see `.env.example` for the full contract.

## Documentation

The project site with a user guide lives at
[ctrl-research.github.io/getbud](https://ctrl-research.github.io/getbud/)
(published from `site/`). Reference docs live in [`docs/`](docs/):

- [ARCHITECTURE.md](docs/ARCHITECTURE.md) — how the pieces fit together
- [CONFIGURATION.md](docs/CONFIGURATION.md) — every `GETBUD_*` variable and sign-in setup
- [DATA_MODEL.md](docs/DATA_MODEL.md) — schema and domain decisions
- [DEPLOYMENT.md](docs/DEPLOYMENT.md) — compose, reverse proxy, backups, upgrades

## Development

Requirements: Go and Node as pinned in `.tool-versions` (asdf/mise), Docker.

```sh
make db        # start postgres (host port 5433)
make seed      # create the local dev user
make run       # Go API on :8081 (local auth enabled)
make web       # Vite dev server on :5173, proxying /api and /auth
```

Open http://localhost:5173 and sign in with `dev@getbud.local` /
`getbud-dev`.

Other targets:

```sh
make generate  # regenerate sqlc code after editing internal/store/queries/
make test      # vet + unit tests
make test-db   # also runs postgres-backed store tests (needs make db)
make build     # embedded-UI production binary in bin/getbud
```

Migrations (goose SQL files in `migrations/`) run automatically at server
startup — no separate migrate step.

## Architecture

- `cmd/server` — entrypoint, config from `GETBUD_*` env vars only, `seed`
  subcommand.
- `internal/auth` — OIDC (PKCE + state) and local login; opaque session
  cookie whose SHA-256 hash is stored server-side.
- `internal/server` — std-lib `net/http` routes, one file per resource.
- `internal/store` — hand-written wrappers over sqlc-generated queries
  (`internal/store/queries/*.sql` → `internal/store/sqlcgen`). Every query
  is scoped by `user_id`.
- `internal/csvimport` — pure CSV parsing/mapping (no HTTP or DB).
- `web/` — React 19 + TypeScript + Vite + Tailwind + TanStack Router/Query,
  charts via Apache ECharts. Embedded into the binary with
  `go build -tags embedwebui`.

All money is integer cents (`bigint`); floats never touch amounts.

### Known simplifications

- Contribution tracking attributes RRSP contributions to the calendar year
  (the first-60-days rule is not modelled).
- Contribution room is user-entered — TFSA/RRSP room is personal (CRA
  MyAccount / notice of assessment); the app only shows the published annual
  limits as hints.
- Reports assume a single currency (CAD by default); mixed-currency
  aggregates are not converted.

## License

MIT — see [LICENSE](LICENSE).
