-- +goose Up
CREATE TYPE category_kind AS ENUM ('income', 'expense');

CREATE TABLE categories (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name        text NOT NULL,
    kind        category_kind NOT NULL,
    -- Hex color hint for charts and badges; '' means auto-assign in the UI.
    color       text NOT NULL DEFAULT '',
    is_archived boolean NOT NULL DEFAULT false,
    created_at  timestamptz NOT NULL DEFAULT now(),

    UNIQUE (user_id, kind, name)
);

-- +goose Down
DROP TABLE categories;
DROP TYPE category_kind;
