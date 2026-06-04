-- 0002_workspace_schema.sql
-- Core persistence layer: workspaces, ingest events, entities, relationships,
-- mismatches, and audit log.  All tables are workspace-scoped.

-- ────────────────────────────────────────────────────────────────────────────
-- Workspaces
-- ────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS workspaces (
    id            TEXT        PRIMARY KEY,          -- slugified workspace path, e.g. "/home/user/proj"
    name          TEXT        NOT NULL,
    path          TEXT        NOT NULL UNIQUE,      -- absolute local folder path used as project key
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ────────────────────────────────────────────────────────────────────────────
-- Ingested source events (raw pipeline input captured for replay)
-- ────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS ingest_events (
    id              TEXT        PRIMARY KEY,         -- stable event_id from source connector
    workspace_id    TEXT        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    connector       TEXT        NOT NULL,            -- e.g. "github", "slack"
    source_uri      TEXT        NOT NULL DEFAULT '', -- URI that produced this event
    event_type      TEXT        NOT NULL,            -- e.g. "document.ingested"
    title           TEXT        NOT NULL DEFAULT '',
    body            TEXT        NOT NULL DEFAULT '',
    content_hash    TEXT        NOT NULL DEFAULT '', -- SHA-256 of normalized body for dedup
    metadata        JSONB       NOT NULL DEFAULT '{}',
    schema_version  TEXT        NOT NULL DEFAULT 'v1',
    ingested_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- idempotency: same event_id in same workspace is a no-op
    UNIQUE (id, workspace_id)
);

CREATE INDEX IF NOT EXISTS idx_ingest_events_workspace
    ON ingest_events (workspace_id, ingested_at DESC);
CREATE INDEX IF NOT EXISTS idx_ingest_events_connector
    ON ingest_events (workspace_id, connector);
CREATE INDEX IF NOT EXISTS idx_ingest_events_hash
    ON ingest_events (workspace_id, content_hash);

-- ────────────────────────────────────────────────────────────────────────────
-- Canonical entities (identity-resolved, workspace-scoped)
-- ────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS entities (
    id                  TEXT        PRIMARY KEY,     -- deterministic from source + canonical key
    workspace_id        TEXT        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    entity_type         TEXT        NOT NULL,        -- e.g. "api_field", "service", "requirement"
    name                TEXT        NOT NULL,
    raw_mention         TEXT        NOT NULL DEFAULT '',
    source_id           TEXT        NOT NULL DEFAULT '', -- ingest_events.id that produced this entity
    confidence          FLOAT       NOT NULL DEFAULT 0,
    extraction_method   TEXT        NOT NULL DEFAULT '',
    aliases             TEXT[]      NOT NULL DEFAULT '{}',
    needs_human         BOOLEAN     NOT NULL DEFAULT FALSE,
    match_layer         TEXT        NOT NULL DEFAULT '',
    conflict_reason     TEXT        NOT NULL DEFAULT '',
    metadata            JSONB       NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (id, workspace_id)
);

CREATE INDEX IF NOT EXISTS idx_entities_workspace
    ON entities (workspace_id);
CREATE INDEX IF NOT EXISTS idx_entities_source
    ON entities (workspace_id, source_id);
CREATE INDEX IF NOT EXISTS idx_entities_type
    ON entities (workspace_id, entity_type);
-- Full-text search on entity name
CREATE INDEX IF NOT EXISTS idx_entities_name_fts
    ON entities USING GIN (to_tsvector('english', name));

-- ────────────────────────────────────────────────────────────────────────────
-- Relationships between entities
-- ────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS relationships (
    id              TEXT        PRIMARY KEY,         -- deterministic from from_id + to_id + kind
    workspace_id    TEXT        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    from_id         TEXT        NOT NULL,            -- entities.id
    to_id           TEXT        NOT NULL,            -- entities.id
    kind            TEXT        NOT NULL,            -- e.g. "co_occurs_in_document"
    confidence      FLOAT       NOT NULL DEFAULT 0,
    evidence        TEXT[]      NOT NULL DEFAULT '{}',
    metadata        JSONB       NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (id, workspace_id)
);

CREATE INDEX IF NOT EXISTS idx_relationships_workspace
    ON relationships (workspace_id);
CREATE INDEX IF NOT EXISTS idx_relationships_from
    ON relationships (workspace_id, from_id);
CREATE INDEX IF NOT EXISTS idx_relationships_to
    ON relationships (workspace_id, to_id);

-- ────────────────────────────────────────────────────────────────────────────
-- Mismatches (reasoning findings)
-- ────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mismatches (
    id              TEXT        PRIMARY KEY,
    workspace_id    TEXT        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    mismatch_type   TEXT        NOT NULL,
    summary         TEXT        NOT NULL DEFAULT '',
    entity_ids      TEXT[]      NOT NULL DEFAULT '{}',
    severity        TEXT        NOT NULL DEFAULT 'low',
    confidence      FLOAT       NOT NULL DEFAULT 0,
    impact          TEXT        NOT NULL DEFAULT '',
    evidence        TEXT[]      NOT NULL DEFAULT '{}',
    affected_roles  TEXT[]      NOT NULL DEFAULT '{}',
    recommended     TEXT        NOT NULL DEFAULT '',
    trace_id        TEXT        NOT NULL DEFAULT '',   -- correlates to ingest pipeline run
    detected_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (id, workspace_id)
);

CREATE INDEX IF NOT EXISTS idx_mismatches_workspace
    ON mismatches (workspace_id, detected_at DESC);
CREATE INDEX IF NOT EXISTS idx_mismatches_severity
    ON mismatches (workspace_id, severity);

-- ────────────────────────────────────────────────────────────────────────────
-- Workspace UI state (durable local workflow state)
-- ────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS workspace_ui_state (
    workspace_id    TEXT        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    state_key       TEXT        NOT NULL,
    payload_json    JSONB       NOT NULL DEFAULT '{}',
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (workspace_id, state_key)
);

CREATE INDEX IF NOT EXISTS idx_workspace_ui_state_workspace
    ON workspace_ui_state (workspace_id, updated_at DESC);

-- ────────────────────────────────────────────────────────────────────────────
-- Connector sync state (cursor + last sync per workspace per connector)
-- ────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS connector_syncs (
    workspace_id    TEXT        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    connector       TEXT        NOT NULL,
    source_uri      TEXT        NOT NULL DEFAULT '',
    cursor          TEXT        NOT NULL DEFAULT '',  -- replay cursor for incremental sync
    last_synced_at  TIMESTAMPTZ,
    event_count     INTEGER     NOT NULL DEFAULT 0,
    status          TEXT        NOT NULL DEFAULT 'idle', -- idle | syncing | error
    last_error      TEXT        NOT NULL DEFAULT '',

    PRIMARY KEY (workspace_id, connector, source_uri)
);

-- ────────────────────────────────────────────────────────────────────────────
-- Audit log (immutable append-only per workspace)
-- ────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS audit_log (
    id              BIGSERIAL   PRIMARY KEY,
    workspace_id    TEXT        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    event_type      TEXT        NOT NULL,   -- e.g. "ingest.started", "entity.created", "mismatch.detected"
    actor           TEXT        NOT NULL DEFAULT 'system',
    connector       TEXT        NOT NULL DEFAULT '',
    source_uri      TEXT        NOT NULL DEFAULT '',
    entity_id       TEXT        NOT NULL DEFAULT '',
    trace_id        TEXT        NOT NULL DEFAULT '',
    payload         JSONB       NOT NULL DEFAULT '{}',
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_log_workspace
    ON audit_log (workspace_id, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_event_type
    ON audit_log (workspace_id, event_type);
CREATE INDEX IF NOT EXISTS idx_audit_log_trace
    ON audit_log (trace_id) WHERE trace_id <> '';
