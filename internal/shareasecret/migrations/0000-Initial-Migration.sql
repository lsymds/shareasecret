CREATE TABLE secrets (
    viewing_id      TEXT NOT NULL,
    management_id   TEXT NOT NULL,
    cipher_text     TEXT NULL,
    ttl             NUMBER NOT NULL,
    deleted_at      NUMBER NULL,
    deletion_reason TEXT NULL,
    created_at      NUMBER NOT NULL
);

CREATE INDEX idx_secrets_viewing_id_deleted_at ON secrets (viewing_id, deleted_at);
CREATE INDEX idx_secrets_management_id_deleted_at ON secrets (management_id, deleted_at);
CREATE INDEX idx_secrets_alive_created_at_ttl_deleted_at ON secrets (created_at, ttl, deleted_at);
