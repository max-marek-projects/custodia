DO $$
BEGIN
    CREATE TYPE secret_type AS ENUM (
    'credentials',
    'text',
    'binary',
    'card'
);
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

CREATE TABLE IF NOT EXISTS secrets (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    type secret_type NOT NULL,
    name TEXT NOT NULL,
    data BYTEA NOT NULL,
    salt BYTEA NOT NULL,
    iv BYTEA NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP DEFAULT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_secret_version_per_user ON secrets (user_id, name, version)
WHERE
    deleted_at IS NULL;