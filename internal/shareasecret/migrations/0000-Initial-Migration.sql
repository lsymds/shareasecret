CREATE TABLE secrets (
    viewing_id      TEXT NOT NULL,
    management_id   TEXT NOT NULL,
    cipher_text     TEXT NOT NULL,
    ttl             NUMBER NOT NULL,
    expires_at      NUMBER NOT NULL,
    created_at      NUMBER NOT NULL
);

CREATE INDEX idx_secrets_viewing_id_expires_at ON secrets (viewing_id, expires_at);
CREATE INDEX idx_secrets_management_id_expires_at ON secrets (management_id, expires_at);
CREATE INDEX idx_secrets_alive_expires_at ON secrets (expires_at);
