CREATE TABLE users (
    id            TEXT PRIMARY KEY,
    login         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    salt          BYTEA NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL
);

CREATE TABLE objects (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    type       TEXT NOT NULL,
    salt       BYTEA NOT NULL,
    ciphertext BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX idx_objects_user_id ON objects(user_id);

CREATE TABLE metadata (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    object_id    TEXT NOT NULL REFERENCES objects(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    order_number INTEGER NOT NULL DEFAULT 0,
    options      JSONB NOT NULL DEFAULT '{}'
);
CREATE INDEX idx_metadata_object_id ON metadata(object_id);
