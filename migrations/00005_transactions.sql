-- +goose Up
CREATE TABLE import_batches (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    account_id     uuid NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    filename       text NOT NULL,
    -- Column mapping used for this import; prefills the next import for the
    -- same account.
    mapping        jsonb NOT NULL,
    row_count      int NOT NULL,
    imported_count int NOT NULL,
    skipped_count  int NOT NULL,
    created_at     timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX import_batches_user_idx ON import_batches (user_id, created_at DESC);

-- Sign convention: amount_cents > 0 is money into the account, < 0 is money
-- out. Category kind carries the income/expense semantics.
CREATE TABLE transactions (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    account_id        uuid NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    date              date NOT NULL,
    amount_cents      bigint NOT NULL,
    payee             text NOT NULL DEFAULT '',
    notes             text NOT NULL DEFAULT '',
    -- NULL means uncategorized (or a transfer leg, which is never
    -- income/expense).
    category_id       uuid REFERENCES categories (id) ON DELETE SET NULL,
    -- Both legs of a transfer between own accounts share one group id;
    -- aggregates exclude rows where this is set.
    transfer_group_id uuid,
    -- Reverting an import deletes the batch; the FK cascades to its rows.
    import_batch_id   uuid REFERENCES import_batches (id) ON DELETE CASCADE,
    -- sha256(date|amount_cents|normalized payee); used for soft duplicate
    -- detection on CSV import. Deliberately NOT unique - identical
    -- transactions on the same day are legitimate.
    dedup_hash        bytea NOT NULL,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX transactions_user_date_idx ON transactions (user_id, date DESC);
CREATE INDEX transactions_account_date_idx ON transactions (account_id, date DESC);
CREATE INDEX transactions_category_idx ON transactions (user_id, category_id);
CREATE INDEX transactions_dedup_idx ON transactions (account_id, dedup_hash);
CREATE INDEX transactions_batch_idx ON transactions (import_batch_id);
CREATE INDEX transactions_transfer_idx ON transactions (transfer_group_id)
    WHERE transfer_group_id IS NOT NULL;

-- +goose Down
DROP TABLE transactions;
DROP TABLE import_batches;
