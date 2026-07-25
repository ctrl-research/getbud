-- +goose Up
-- Point-in-time account balances, the source for net-worth charts and
-- investment account balances. Net worth carries the latest snapshot per
-- account forward.
CREATE TABLE balance_snapshots (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    account_id    uuid NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    as_of         date NOT NULL,
    balance_cents bigint NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),

    -- One snapshot per account per day; PUT upserts on conflict.
    UNIQUE (account_id, as_of)
);

CREATE INDEX balance_snapshots_lookup_idx ON balance_snapshots (account_id, as_of DESC);

-- +goose Down
DROP TABLE balance_snapshots;
