-- +goose Up
CREATE TYPE account_type AS ENUM
    ('chequing', 'savings', 'credit_card', 'rrsp', 'tfsa', 'fhsa', 'non_registered', 'other');

CREATE TABLE accounts (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id               uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name                  text NOT NULL,
    type                  account_type NOT NULL,
    currency              char(3) NOT NULL DEFAULT 'CAD',
    institution           text NOT NULL DEFAULT '',
    -- Starting balance for cash accounts; derived balances are
    -- opening_balance_cents + SUM(transactions.amount_cents).
    opening_balance_cents bigint NOT NULL DEFAULT 0,
    is_archived           boolean NOT NULL DEFAULT false,
    created_at            timestamptz NOT NULL DEFAULT now(),

    UNIQUE (user_id, name)
);

-- +goose Down
DROP TABLE accounts;
DROP TYPE account_type;
