# Configuration

getbud is configured entirely through `GETBUD_*` environment variables.
The compose file reads them from `.env` (see `.env.example` for a commented
template).

## Reference

| Variable | Default | Description |
|---|---|---|
| `GETBUD_DATABASE_URL` | — | **Required.** Postgres connection string. |
| `GETBUD_ADDR` | `:8080` | Listen address. |
| `GETBUD_BASE_URL` | `http://localhost:8080` | Public URL of the instance. OAuth redirect URIs derive from it; session cookies are `Secure` when it is `https`. No trailing slash. |
| `GETBUD_GOOGLE_CLIENT_ID` | — | Google OAuth client id. Set together with the secret to enable Sign in with Google. |
| `GETBUD_GOOGLE_CLIENT_SECRET` | — | Google OAuth client secret. |
| `GETBUD_OIDC_ISSUER_URL` | — | Generic OIDC issuer URL (Authentik, Keycloak, …), used for discovery. Must match the provider's discovery document exactly, trailing slash included — copy it from the IdP. Set together with the client id and secret. |
| `GETBUD_OIDC_CLIENT_ID` / `GETBUD_OIDC_CLIENT_SECRET` | — | Client credentials for the generic OIDC provider. |
| `GETBUD_OIDC_NAME` | `SSO` | Label for the generic provider's login button. |
| `GETBUD_LOCAL_AUTH` | `false` | Enable email/password accounts (intended for dev/testing; `make seed` or the `seed` subcommand creates one). |
| `GETBUD_ALLOWED_EMAILS` | — | Comma-separated emails allowed to **sign up** after the first user. Empty means the instance is closed to new accounts. |

Validation rules enforced at startup:

- `GETBUD_DATABASE_URL` is required.
- The two Google variables must be set together.
- The three OIDC variables (`ISSUER_URL`, `CLIENT_ID`, `CLIENT_SECRET`) must
  be set together.

## Sign-in setup

### Google

1. In the [Google Cloud Console](https://console.cloud.google.com/apis/credentials),
   create a project and configure the OAuth consent screen (External; Testing
   mode is fine — add the accounts that will sign in as test users).
2. Create an **OAuth client ID** of type *Web application* with the
   authorized redirect URI `$GETBUD_BASE_URL/auth/google/callback`.
3. Set `GETBUD_GOOGLE_CLIENT_ID` and `GETBUD_GOOGLE_CLIENT_SECRET` and
   restart.

The redirect URI must match `GETBUD_BASE_URL` exactly (a mismatch shows
`redirect_uri_mismatch`). Google requires HTTPS except on localhost.

### Generic OIDC

Works with any discovery-capable provider (Authentik, Keycloak, Pocket ID,
…). Redirect URI: `$GETBUD_BASE_URL/auth/oidc/callback`.

### First user & allowlist

Sign in yourself first — **the first account becomes admin** and bypasses
the allowlist. After that, only `GETBUD_ALLOWED_EMAILS` addresses can
register; existing accounts always keep signing in. A returning SSO user is
matched by provider subject first, then linked by verified email.

## Local development ports

`make run` and `docker-compose.yml` shift ports to coexist with other apps
on the same machine: Postgres on host **5433**, the app on host **8081**.
The defaults inside the container/binary remain 5432/8080.
