CREATE TABLE secrets (
    id              INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    access_id       TEXT NOT NULL,
    management_id   TEXT NOT NULL,
    cipher_text     TEXT NULL,
    ttl             NUMBER NOT NULL,
    maximum_views   NUMBER NOT NULL DEFAULT(0),
    deleted_at      NUMBER NULL,
    deletion_reason TEXT NULL,
    created_at      NUMBER NOT NULL
);

CREATE INDEX idx_secrets_access_id_deleted_at ON secrets (access_id, deleted_at);
CREATE INDEX idx_secrets_management_id_deleted_at ON secrets (management_id, deleted_at);
CREATE INDEX idx_secrets_alive_created_at_ttl_deleted_at ON secrets (created_at, ttl, deleted_at);

CREATE TABLE secret_views (
    id          INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    secret_id   INT NOT NULL,
    viewing_key TEXT NOT NULL,
    viewed_at   NUMBER NULL,
    created_at  NUMBER NOT NULL,

    FOREIGN KEY (secret_id) REFERENCES secrets (id)
);

CREATE INDEX idx_secret_views_secret_id ON secret_views (secret_id);
CREATE INDEX idx_secret_views_secret_id_viewing_key_viewed_at ON secret_views (secret_id, viewing_key, viewed_at);
CREATE INDEX idx_secret_views_secret_id_viewed_at ON secret_views (secret_id, viewed_at);
