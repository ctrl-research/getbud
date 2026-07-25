-- +goose Up
-- User-entered available contribution room per registered account type and
-- tax year (TFSA room is cumulative and personal; RRSP room comes from the
-- CRA notice of assessment). Contributions/withdrawals are derived from
-- transactions, not stored here.
CREATE TABLE contribution_room (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    account_type account_type NOT NULL CHECK (account_type IN ('rrsp', 'tfsa', 'fhsa')),
    tax_year     int NOT NULL,
    room_cents   bigint NOT NULL,
    notes        text NOT NULL DEFAULT '',

    UNIQUE (user_id, account_type, tax_year)
);

-- +goose Down
DROP TABLE contribution_room;
