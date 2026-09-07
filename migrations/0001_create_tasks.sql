CREATE TABLE IF NOT EXISTS tasks (
    id         BIGSERIAL PRIMARY KEY,
    title      TEXT        NOT NULL CHECK (length(title) BETWEEN 1 AND 200),
    done       BOOLEAN     NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- The list endpoint sorts by id, which the primary key index already serves.
-- No extra index until a filter or a second sort order arrives.
