# Deployment

getbud is a single container (Go binary with the web UI embedded) plus
PostgreSQL. State lives entirely in the database — a `pg_dump` is a
complete backup. Migrations run automatically at startup, so upgrades are
just a new image.

See [CONFIGURATION.md](CONFIGURATION.md) for every environment variable.

## Docker Compose

The repo's `docker-compose.yml` builds from source:

```sh
git clone https://github.com/ctrl-research/getbud
cd getbud
cp .env.example .env      # set POSTGRES_PASSWORD at minimum
docker compose up -d      # app on host :8081
```

For a first look, the compose file enables local auth — create a user with:

```sh
docker compose exec app getbud seed   # dev@getbud.local / getbud-dev
```

For a real deployment, set in `.env`:

- `POSTGRES_PASSWORD` — a real password (used by both services);
- `GETBUD_BASE_URL` — the public HTTPS URL you serve getbud on;
- Google or OIDC credentials (see CONFIGURATION.md);
- `GETBUD_ALLOWED_EMAILS` — who may sign up after you;
- and disable `GETBUD_LOCAL_AUTH`.

## Reverse proxy

Terminate TLS in front (Caddy, Traefik, nginx) and proxy to the app port.
Cookies are marked `Secure` automatically when `GETBUD_BASE_URL` is https.
Example Caddyfile:

```
budget.example.com {
    reverse_proxy localhost:8081
}
```

## Building the image yourself

```sh
docker build -t getbud .
```

The Dockerfile is a 3-stage build: Node builds the SPA, Go compiles the
binary with the UI embedded (`-tags embedwebui`), and the runtime stage is
a minimal Alpine image running as a non-root user, listening on 8080.

## Bare binary (no Docker)

```sh
make build          # builds web UI + embedded binary at bin/getbud
GETBUD_DATABASE_URL=postgres://... ./bin/getbud
```

Point it at any Postgres 16+ database; migrations apply on startup.

## Backups & upgrades

- **Backup**: `pg_dump` of the database. That's everything.
- **Upgrade**: pull/build the new image and restart. Goose migrations are
  append-only and run automatically; restoring an older dump into a newer
  version works.
- **Health**: `GET /healthz` returns `{"status":"ok"}` and checks the
  database; wire it to your uptime monitoring or container healthcheck.
