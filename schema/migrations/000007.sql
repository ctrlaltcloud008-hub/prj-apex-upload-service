CREATE CHANGE STREAM outbox_stream
    FOR outbox(shard_id, video_id, topic, payload, status, created_at)
    OPTIONS (
        value_capture_type = 'NEW_VALUES',
        retention_period   = '7d'
    );
