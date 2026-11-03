-- migrate:up
CREATE TABLE notes (
    id bigserial PRIMARY KEY,
    text text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- migrate:down
DROP TABLE notes;
