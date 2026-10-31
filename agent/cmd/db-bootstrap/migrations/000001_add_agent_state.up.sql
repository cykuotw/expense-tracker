CREATE SCHEMA expense_agent;

REVOKE ALL ON SCHEMA expense_agent FROM PUBLIC;

CREATE TABLE expense_agent.sessions(
    app_name text NOT NULL,
    owner_id text NOT NULL,
    session_id uuid NOT NULL,
    state jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz,
    PRIMARY KEY (session_id),
    UNIQUE (app_name, owner_id, session_id)
);

-- NULL run IDs identify thread anchors. Run mappings reference that same
-- runtime-owned session and a separate runtime execution UUID.
-- App/owner columns enforce the namespace for external thread/run IDs;
-- the composite session foreign key keeps that namespace consistent.
CREATE TABLE expense_agent.agui_ref(
    id uuid PRIMARY KEY,
    app_name text NOT NULL,
    owner_id text NOT NULL,
    thread_id text NOT NULL,
    run_id text,
    session_id uuid NOT NULL,
    runtime_run_id uuid UNIQUE,
    UNIQUE (runtime_run_id, session_id),
    created_at timestamptz NOT NULL DEFAULT NOW(),
    UNIQUE NULLS NOT DISTINCT (app_name, owner_id, thread_id, run_id),
    CHECK ((run_id IS NULL AND runtime_run_id IS NULL) OR (run_id IS NOT NULL AND run_id <> '' AND runtime_run_id IS NOT NULL)),
    FOREIGN KEY (app_name, owner_id, session_id) REFERENCES expense_agent.sessions(app_name, owner_id, session_id)
);

CREATE UNIQUE INDEX agent_ui_thread_session ON expense_agent.agui_ref(session_id)
WHERE
    run_id IS NULL;

-- All skills share these events. Explicit event types control history/UI
-- projections; JSON data carries behavior without skill tables.
-- Ownership is resolved through session_id, not copied into every event.
CREATE TABLE expense_agent.events(
    id uuid PRIMARY KEY,
    session_id uuid NOT NULL,
    run_id uuid,
    event_type text NOT NULL CHECK (event_type <> ''),
    author text NOT NULL CHECK (author <> ''),
    invocation_id text NOT NULL DEFAULT '',
    data jsonb NOT NULL,
    created_at timestamptz NOT NULL,
    FOREIGN KEY (session_id) REFERENCES expense_agent.sessions(session_id),
    FOREIGN KEY (run_id, session_id) REFERENCES expense_agent.agui_ref(runtime_run_id, session_id),
    CHECK (event_type = 'adk.event' OR run_id IS NOT NULL),
    CHECK (event_type <> 'resource.bound' OR (jsonb_typeof(data) = 'object' AND COALESCE(data ->> 'resourceType', '') <> '' AND COALESCE(data ->> 'resourceId', '') <> '' AND COALESCE(data ->> 'fingerprint', '') <> ''))
);

CREATE UNIQUE INDEX agent_resource_identity ON expense_agent.events(session_id,(data ->> 'resourceType'),(data ->> 'resourceId'))
WHERE
    event_type = 'resource.bound';
