CREATE TABLE outbox_relay_checkpoints (
    partition_token         STRING(MAX)        NOT NULL,
    watermark               TIMESTAMP          NOT NULL,
    state                   STRING(32)         NOT NULL,
    parent_partition_tokens ARRAY<STRING(MAX)>,
    created_at              TIMESTAMP          NOT NULL OPTIONS (allow_commit_timestamp=true),
    updated_at              TIMESTAMP          NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (partition_token);
