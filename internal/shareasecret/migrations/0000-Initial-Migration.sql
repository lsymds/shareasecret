CREATE TABLE secrets (
    view_id     TEXT NOT NULL,
    manage_id   TEXT NOT NULL,
    cipher_text TEXT NOT NULL,
    ttl         NUMBER NOT NULL,
    alive_until NUMBER NOT NULL,
    created_at  NUMBER NOT NULL
);

CREATE INDEX idx_secrets_view_id_alive_until ON secrets (view_id, alive_until);
CREATE INDEX idx_secrets_manage_id_alive_until ON secrets (manage_id, alive_until);
CREATE INDEX idx_secrets_alive_until ON secrets (alive_until);
