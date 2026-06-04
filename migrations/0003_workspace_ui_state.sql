-- 0003_workspace_ui_state.sql
-- Idempotent backfill for local databases that already recorded 0002 before
-- durable workspace UI state was added.

CREATE TABLE IF NOT EXISTS workspace_ui_state (
    workspace_id    TEXT        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    state_key       TEXT        NOT NULL,
    payload_json    JSONB       NOT NULL DEFAULT '{}',
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (workspace_id, state_key)
);

CREATE INDEX IF NOT EXISTS idx_workspace_ui_state_workspace
    ON workspace_ui_state (workspace_id, updated_at DESC);
