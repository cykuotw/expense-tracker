# Agent runtime

The Agent is a read-only assistant runtime built with Go ADK. Bundled skills
register a Chat orchestrator and Task capabilities; Go code owns tool grants,
model selection, and execution limits. The current executable serves a
loopback-only HTTP/SSE interface backed by Gemini. Export `GEMINI_API_KEY` before
starting it:

```sh
export GEMINI_API_KEY='<your Gemini API key>'
go run ./agent/cmd/agent
```

`internal/api/server.go` assembles the HTTP server. `api/routes` declares routes,
`api/handlers` parses requests and encodes JSON/SSE responses, and `api/middleware`
owns the local bearer and fixture boundaries. Handlers consume small interfaces
using shared data contracts; API implementation packages do not import the Agent
runtime, ADK, model services, tools, or Worker clients. Both current entry points
use the same chat handler. `cmd/agent` constructs the concrete runtime and injects
it into the API; signal/shutdown handling stays in that executable.
`internal/adapters/expenseapi` implements backend calls for domain tools and an
independent authentication function. `internal/runtime` coordinates execution;
`cmd/agent` injects constructed tools, their completion checks and authentication.
The runtime never queries balances or other expense-domain data directly.
Shared chat validation, data and public error contracts live in `internal/models`.
`internal/credentials` carries request-scoped access tokens and the secrets
used for credential filtering, without depending on HTTP or Agent services.
`internal/runtime/app.go` composes the ADK engine and chat/history services,
retaining the API-facing application methods. `runtime/engine` owns ADK agent
construction, tools/callback wiring, skill metadata and one-turn event processing
with visible output and delegation logs. It does not own request authentication,
conversation admission or recording lifecycle state.
`runtime/services/chat` owns authenticated chat turns, process-local concurrency
and duplicate guards, execution budgets/deadlines and persisted output/failure
handling. It calls `engine.RunTurn` after admission. `runtime/services/conversation`
reads and deletes owned history through injected storage and the chat service's
identity/run-status callbacks; it does not depend on the engine or chat package.
Neither service imports the HTTP API. Tests live beside their responsibilities;
full delegation/tool/recording and PostgreSQL flows use the public App under
`runtime/integration_tests`.
`internal/llm` groups Gemini construction (`gemini.go`), configured model-role selection
(`policy.go`), shared turn budgets (`budget.go`), model limits and diagnostic
recording (`model_wrapper.go`), and response limits/cleanup (`response.go`).
The runtime uses `SelectModel`, `WithRunBudget` and `WithLimitsAndRecording`;
provider errors never trigger a model switch. LLM tests live beside these files.
The injected PostgreSQL/session/recording service lives in `internal/store/session`; bundled skill loading and
validation live in `internal/runtime/skill`. FunctionTools use the same layout:
`internal/tools/functiontool/updatestate` and `internal/tools/functiontool/getbalance`.
Their constructors, instructions and validation are wired by `internal/tools/catalog.go`.
Runtime receives generic instruction/response callbacks and coordinator grants,
without importing either concrete tool. Deterministic model and balance fixtures
live in `internal/fixtures`. Tests inject model fixtures; the executable uses
`llm.NewGeminiFactory` and injects fixture balance data only in `AGENT_MODE=fixture`.

It listens on `127.0.0.1:8090` by default. Set `AGENT_LISTEN_ADDR` to change the
address, for example `AGENT_LISTEN_ADDR=127.0.0.1:9090`; only literal loopback
addresses with a port are accepted. `POST /chat` accepts `conversationId`, `messageId`,
`text`, and an optional `runId` in a JSON body. In the default `fixture` mode, requests such as `balance demo` exercise synthetic
balance data through the real model. This mode rejects remote
peers and credential-bearing requests and cannot start in Lambda. Authenticated
browser cookie chat and Lambda streaming startup are not available in this executable.

## Environment reference

See [.env.example](.env.example) for runtime, database bootstrap and optional
integration-test settings with placeholder connection URLs. Copy it to
`agent/.env` for personal values. Agent commands read exported environment
variables; they do not load dotenv files automatically. Export the variables
needed for the command you are running. Keep bootstrap operator credentials
in a separate shell from the Agent runtime. Both runtime modes use real Gemini
inference and require `GEMINI_API_KEY`; deterministic models remain test fixtures.

`config.Config` defines the runtime settings; `config.Load()` in `config/env.go`
loads and validates them once at startup:
`AGENT_MODE`, `AGENT_LISTEN_ADDR`, `AGENT_WORKER_URL`, and
`AGENT_STATE_DATABASE_URL`, plus `GEMINI_API_KEY`. Assembly and HTTP serving consume that validated
configuration instead of reading environment variables independently. Local
mode requires the Worker and state database URLs; fixture mode uses synthetic
balance data and in-memory sessions. Neither mode substitutes a fixture model. Shared local-only guards also reject Lambda/release execution.

## Models and execution limits

`config/env.go` is the single source for environment settings and defaults.
The following values are optional overrides; changing them requires a process
restart, without rebuilding the executable. Export them before starting Agent;
dotenv files are not loaded automatically.

| Environment variable | Default | Scope |
| --- | --- | --- |
| `AGENT_DEFAULT_MODEL` | `gemini-3.8-flash` | Specialist/default model |
| `AGENT_DEFAULT_MAX_OUTPUT_TOKENS` | `2048` | Default-role output ceiling per call |
| `AGENT_ROUTER_MODEL` | `gemini-3.5-flash-lite` | Orchestrator model |
| `AGENT_ROUTER_MAX_OUTPUT_TOKENS` | `2048` | Router-role output ceiling per call |
| `AGENT_MAX_MODEL_CALLS` | `6` | Shared across all agents in one turn |
| `AGENT_MAX_TOOL_CALLS` | `2` | Budgeted backend reads in one turn |
| `AGENT_MAX_STATE_UPDATES` | `4` | State-update attempts in one turn |
| `AGENT_MAX_CONTEXT_BYTES` | `65536` | Serialized conversation contents per model call |
| `AGENT_MAX_RESPONSE_BYTES` | `32768` | Serialized provider response event |
| `AGENT_MAX_TOTAL_RESPONSE_BYTES` | `262144` | Cumulative serialized responses per model call |
| `AGENT_MAX_RESPONSE_EVENTS` | `256` | Provider response events per model call |

All numeric settings must be positive 32-bit integers. Explicitly blank model
names or invalid numeric values fail startup. The total-response byte ceiling
must be at least the per-response byte ceiling.

Bundled skill `metadata.model` selects the trusted role `default` or `router`;
it does not contain a provider model ID. Missing/unknown roles fall back to the
configured default profile and expose fallback metadata. Role-specific token
ceilings remain independent even when both roles use the same model. The wrapper
preserves smaller positive request token limits and clamps larger limits to the
selected profile's ceiling. Provider failures do not switch models.

Shared `ModelProfile`, `ModelPolicy`, `ModelSelection` and `LLMLimits` contracts live in
`internal/models`, without SDK dependencies. Runtime receives validated settings
and passes copied limits to model wrappers and per-turn counters. Tests inject
explicit fixture profiles/limits without loading runtime environment settings.

Both executable modes use Gemini. Changing model IDs does not add support for
another provider protocol; an additional provider needs a compatible model
factory. Client construction performs no inference; chat execution makes model
calls. Byte, call-count and token ceilings remain independent of fixed transport,
visible-output and turn-deadline limits.

## Local mode with real Worker authorization

Set `AGENT_MODE=local` on both standalone processes. The Worker must listen on a
literal loopback address, and the Agent must receive its URL (including `API_URL`
prefix, if configured):

```sh
# Terminal 1: provide your local PostgreSQL and application JWT configuration.
AGENT_MODE=local MODE=debug BACKEND_URL=127.0.0.1:8000 go run ./backend/cmd/tracker

# Terminal 2: export GEMINI_API_KEY and AGENT_STATE_DATABASE_URL for the scoped Agent state login.
# No AWS credentials or IAM role is needed.
AGENT_MODE=local AGENT_WORKER_URL=http://127.0.0.1:8000 go run ./agent/cmd/agent
```

The database must already have the application migrations applied. Use a valid
local user's access JWT issued by the existing login flow; Agent forwards only
that access JWT to the Worker. Send it as `Authorization: Bearer <access JWT>`
on `POST http://127.0.0.1:8090/chat`, with a JSON body such as:

```json
{"conversationId":"local-chat","messageId":"one","text":"balance <group UUID>"}
```

Replace `<group UUID>` with a group the user belongs to and use a fresh
`messageId` per turn. Gemini selects tools to retrieve and summarize real Worker
balance results. The default fixture mode remains available by omitting
`AGENT_MODE` or setting it to `fixture`; `balance demo` uses synthetic balance
data in that mode while still calling Gemini.

Local mode skips only AWS IAM/SigV4. Preflight still verifies JWT type/signature,
expiry and active account before any model execution; balance calls recheck group
membership. Missing/invalid/expired JWTs return 401, inactive accounts return 403,
and tool failures after streaming starts produce `RUN_ERROR`. Cookies and actor
headers are rejected; the JWT is never a model argument or session field.
Both services reject remote peers, foreign hosts/origins and forwarded requests.
Use `127.0.0.1` or `[::1]`, not wildcard listeners or DNS hostnames. The local
Worker client disables HTTP proxies and redirects. Lambda and release mode
cannot enable these adapters, and the default Worker retains IAM enforcement
even when `AGENT_MODE=local` exists in its environment.

## Persistent local conversations

Agent migrations live in `agent/cmd/db-bootstrap/migrations/`, starting at
`000001`. They are embedded in the operator executable, so migration discovery
does not rely on the current directory. They use
`public.expense_agent_schema_migrations` with independent versioning and locking;
backend migrations continue to use `public.schema_migrations`. The two sequences
can share one PostgreSQL database. Backend migrations do not create Agent tables.

`cmd/db-bootstrap` reads operator settings and passes the embedded files to
`internal/databasebootstrap`, which applies migrations before provisioning the
restricted runtime role.

A separate `expense_agent` schema owns all Agent records; the Agent must not use
the Worker's database login. Before starting the Agent, provide
`AGENT_STATE_ADMIN_DATABASE_URL`, `AGENT_STATE_DB_USER` and
`AGENT_STATE_DB_PASSWORD` in a separate operator shell, then run:

```sh
AGENT_STATE_DB_USER=agent_state_runtime go run ./agent/cmd/db-bootstrap
```

This explicit local operator command applies forward migrations, then creates
or updates the restricted runtime role and grants on the resulting tables.
Rerunning it applies pending migrations and refreshes role credentials/grants.
It requires a DDL- and role-capable operator connection. Migrations bound
statements to 30 seconds and migration lock waits to 10 seconds; role setup has
a separate 15-second deadline. Dirty migration state or untracked existing
schemas stop bootstrap before role provisioning. Bootstrap never runs during
Agent startup or on requests. The version table stays outside `expense_agent`, so
runtime table grants do not grant migration-version access.

Role names must start with `agent_state_`. Bootstrap refuses to repurpose a
privileged role or one with memberships. It grants only database connection,
schema usage, and SELECT/INSERT/UPDATE on Agent tables; no domain access, schema
creation or physical-delete grant. Rerun it after adding Agent tables. The
Agent process receives only `AGENT_STATE_DATABASE_URL`, pointing to that scoped
login, and validates its isolation before serving. Missing/invalid DB setup
fails startup instead of falling back to memory. The synthetic fixture needs
no DB. The state pool has two open connections and one idle connection, with
three-second storage-operation/statement limits.

The persistence service implements ADK sessions and application execution
records, injected through `runtime.Dependencies.Sessions` and `runtime.Dependencies.Records`.
Runtime sessions start with empty state. The current state and each committed
event's state delta are stored in one transaction. Duplicate ADK event IDs do
not reapply state changes. `temp:`
state is excluded from session persistence; shared `app:`/`user:` state prefixes
are rejected because this runtime retains conversation-local history only.
Model context reads use the latest 256 ADK events, capped at 512 KiB; original
records remain in the DB. Existing model-context limits still apply. Oversized
required state/history fails explicitly; start a new conversation when needed.

All skills share three tables: `sessions` for owner-scoped metadata/state,
`events` for `JSONB data`, and `agui_ref` for external identity mappings.
Run start/terminal summaries are execution events (`run.started` and
`run.finished`); history derives status and user messages from those events.
There is no separate runs table, lease or heartbeat.
Events reference sessions by UUID; app/owner metadata is read from the session
after ownership validation. `agui_ref` retains app/owner columns to enforce
external thread/run ID uniqueness within each namespace, with a foreign key
that prevents a mismatch with the session owner.
Adding a skill does not require another table. Each event
has an author, `event_type`, session link, optional run link, invocation ID and
database creation timestamp. Diagnostic call IDs live inside
the JSON snapshot envelope as `callId`, not in a table column. The canonical
ADK payload retains its native event ID, branch, actions and state deltas;
the database event ID identifies the stored record independently. Session events
created outside a run may have no run link. Diagnostic payloads retain explicit
completeness/omission envelopes. Domain tool inputs/results are recorded as
execution diagnostics without a resource fingerprint lock.

Readers select explicit event types: `adk.event` restores canonical ADK
history and `output` serves visible replay. Canonical history is reserved for
its session writer; execution snapshots cannot use that type. Legacy
`resource.bound` rows are ignored by those readers and are no longer written.
Other execution records remain restricted
diagnostics. There is no payload schema-version column; readers decode by
event type, tolerate additional fields, and reject malformed required data.
See [event payload formats](docs/event-data.md) for the JSON structures.
Canonical event insertion and state updates are atomic. Each append stores a new
record; native ADK event IDs remain in JSON and are not used for deduplication.
The MVP has no automatic event retry or redelivery path.
Events are ordered by `created_at` and event ID. Recent history is selected
with `created_at DESC, id DESC`, then restored chronologically for ADK.
Event inserts do not update a session counter. Within the existing session row
lock, one INSERT uses a latest-event timestamp lookup and
`clock_timestamp()`; ties or backward clock movement advance creation time by
one microsecond. Native event timestamps remain unchanged in JSON data.
Read-committed transactions make the preceding writer visible after lock acquisition.
Conversation activity time is derived from session metadata and the latest event.

The runtime generates internal session UUIDs; `sessions.session_id` is the sole
primary key. A caller's `conversationId` is the external AG-UI thread selector,
not an ADK session ID. `agui_ref` has one thread anchor (`run_id` and
`runtime_run_id` are NULL) per app/owner/thread and one mapping for each UI run.
All runs of that owned thread resolve to the same internal session UUID.
The runtime serializes turns per owned thread, and run mappings are saved
atomically with the start event. Retained mappings prevent deleted threads from being recreated.
Foreign keys retain app/owner consistency even though session identity is a
single UUID primary key.

A request may supply `runId`; otherwise the runtime generates an external run ID.
Each UI run maps to an internal runtime run UUID. While the same runtime process
is alive, its in-memory guard rejects duplicate UI run/message IDs and concurrent
turns for that owned thread. A reused UI run ID after a process restart reuses
its lookup mapping and may execute again; mapping uniqueness is not resend
protection. Transport output and history use external IDs, while ADK/event
storage use internal identities.
History/replay/deletion resolve the mapping under current actor authentication;
client IDs never select arbitrary internal sessions or runs. The fixture keeps
equivalent owner-scoped session mappings in memory. ADK invocation IDs remain
distinct from both UI and internal run IDs; associations are read from events.

Bundled skills grant the `update_state` FunctionTool to the orchestrator and
balance specialist. It accepts generic JSON changes, for example
`{"set":{"selectedGroupId":"demo","filters":{"itemQuery":"milk"}},"unset":["previousQuery"]}`.
`set` replaces complete top-level values, including objects, arrays and null;
`unset` removes keys, and omitted keys remain. Skills choose their own keys
without workflow namespaces. Values live inside the session's `agent_context`
and are supplied as untrusted JSON context on later model calls. They cannot
change authenticated identity, tool grants or backend authorization, and are not
evidence of current balances. Existing credential filtering applies.

By default each run permits four state-update attempts separately from its two
backend reads; the environment settings above control those ceilings.
Each model response must call `update_state` alone; combine changes in one call.
This avoids conflicting deltas when ADK executes parallel function calls.
The resulting context is limited to 16 KiB, 64 top-level keys, 128-byte keys,
eight nesting levels and 256 JSON nodes. Invalid changes leave context unchanged.
The tool returns `status: staged`: ADK commits its `StateDelta` with the function
response event through the injected persistence service. A failed commit fails
the run; the tool does not write SQL or promise a commit itself. Fixture models
remain deterministic and do not choose state updates automatically; tool dispatch
and persistence are verified with synthetic model responses.

The MVP supports one runtime process, with one active turn per owned thread,
a four-active-turn cap and bounded in-memory message/run-ID tracking. A
60-second execution deadline covers each turn. This guard is used by both
fixture and persisted chat; no database admission locks, message-dedup rows,
run leases or heartbeats are used. A pre-start storage failure releases the
in-memory reservation so the request may be retried before inference begins.

Sessions, state, ADK history and UI mappings survive restarts and idle gaps;
returning later to the same owned thread loads the same conversation. This is
independent of resend protection. Cross-restart deduplication, distributed
admission and automatic crash detection are deferred. The runtime never
restarts model/tool execution from a history read. Missing terminal events are
shown as `unknown` when no matching turn is active in this process. Cancellation
attempts a bounded independent terminal write; completed results remain complete
even if their final network delivery fails.

Fresh preflight authentication is required for these loopback bearer-auth routes:
History/replay/deletion operations have a ten-second overall deadline.

| Method | Route | Result |
| --- | --- | --- |
| GET | `/chat/conversations` | Up to 100 readable owned conversations |
| GET | `/chat/conversations/{conversation}/runs` | Latest 100 runs, user messages and status |
| GET | `/chat/conversations/{conversation}/runs/{run}/events` | Up to 256 visible output records after an optional event-ID cursor, capped at 512 KiB |
| DELETE | `/chat/conversations/{conversation}` | Soft-delete the owned conversation |

The conversation ID resolves through `agui_ref` to the owned ADK session.
UI run IDs resolve to runtime run UUIDs, which link to ADK invocation IDs
separately; model and native tool call IDs remain correlated
within the run. Assistant message IDs and output event IDs survive restart.
Each output record may produce several AG-UI frames (for example message start
and content). SSE `id` is the persisted `eventId` for that output group.
For recovery, omit `after` to read from the start, or pass
`?after=<eventId>` for the same run. The server resolves the ID to its stored
`created_at`/ID pair and returns later records chronologically. Unknown,
foreign-run and diagnostic event IDs cannot be used as output cursors. Recovery
JSON includes `eventId` and UTC `createdAt`. Repeated recovery reads do not
execute tools or models.

Historical conversations remain available to their authenticated owner after
balance or group-membership changes. They are records of earlier results, not
current financial facts. Listing, reading or continuing a conversation does not
query domain data automatically. New balance requests go through get_balance
and the Worker authorizes each request; revoked access still denies new reads.
Sessions have no automatic TTL or scheduled purge. Soft deletion retains audit
records, excludes normal access and blocks automatic session recreation. A
process-local output/deletion gate allows already-started writes to finish before
deletion commits and prevents subsequent output; sent bytes cannot be retracted.

Detailed diagnostics are available only through privileged operator DB access,
with no diagnostic HTTP endpoint. They retain model requests after preparation,
provider responses (including available reasoning/summaries), tool data, timings,
versions, usage and call relationships. Each payload is bounded to 128 KiB;
oversized diagnostic data is omitted with original/retained sizes and a reason.
Required thought signatures and their part associations are preserved exactly;
oversized required history is rejected. Provider reasoning that is not exposed
cannot be captured. JWTs, cookies, API keys and passwords are excluded. Normal
chat/history and application logs never expose provider reasoning or signatures.

## Worker transport

`expenseapi.New` in `internal/adapters/expenseapi` accepts an HTTPS Worker base URL, AWS region,
credential provider, and optional HTTP transport. The composition root supplies
this client to `tools.NewCatalog` and injects the constructed tools through
`runtime.Dependencies.Tools`. Separately, `client.Authenticate` is injected through
`runtime.Dependencies.Authenticate` and runs before inference on every turn. A trusted chat transport must attach only the current access token
using `credentials.WithAccessToken`. Do not forward refresh tokens or cookies.
The client never stores access tokens on the shared client or in ADK state.

The Worker exposes two read-only routes under its configured API prefix:

| Method | Route | Result |
| --- | --- | --- |
| GET | `/internal/agent/v1/preflight` | Authenticated `actorId` and UTC `expiresAt` |
| GET | `/internal/agent/v1/balance/{groupId}` | Group ID and bounded balance rows |

Each row contains participant IDs in `from`/`to`, an exact decimal `amount`
string, and its original `currency`. No currency conversion or model calculation
is involved. Results contain at most 100 rows; oversized results fail explicitly.

Worker configuration `AGENT_TOOL_ROLE_ARN` identifies the approved Agent IAM
role for the default signed boundary. Missing or malformed configuration disables
access to that boundary. Every signed-boundary request requires
an approved assumed-role identity from trusted API Gateway HTTP API IAM context,
the exact registered GET route key, and a valid active user's access JWT in
`X-Agent-Access-Token`. `Authorization` belongs to SigV4. Cookies, bearer fallback,
caller-supplied actor/principal headers, `$default`, and greedy routes cannot
authorize this boundary. Balance reads recheck membership on every call; archived
groups remain readable by current members, as in ordinary balance reads.

The client signs the dedicated JWT header, uses temporary credentials supplied
by the runtime credential provider, rejects redirects, bounds response bytes,
and applies a five-second call deadline including credential retrieval. Transport
and Worker errors expose controlled error codes without raw failure details.
The unauthenticated local fixture does not inject this client.

## Local verification

Unit tests live beside their implementation. API tests that combine HTTP with
the Agent runtime or Worker live in `internal/api/integration_tests`; their shared
helpers stay in that directory too.

Run backend-related test binaries outside the repository when local dotenv
loading must be avoided. One wrapper for Go's `-exec` option is:

```sh
cat > /tmp/expense-agent-test-exec <<'SH'
#!/bin/sh
cd /tmp || exit 1
exec "$@"
SH
chmod +x /tmp/expense-agent-test-exec
go test -race -exec /tmp/expense-agent-test-exec ./agent/... ./backend/services/agenttools ./backend/internal/tracker
go vet ./agent/... ./backend/services/agenttools ./backend/internal/tracker ./internal/agentcontract
go build ./agent/... ./backend/...
```

`TestPostgresSignedClientWorkerBalance` and `TestLocalChatPostgresRoundTrip`
require an isolated PostgreSQL database
with both backend and Agent migrations applied. Supply `DB_PUBLIC_HOST`,
`DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, and `DB_SSLMODE` through the test process environment,
then set `AGENT_TEST_POSTGRES=1` and run the tests in
`./agent/internal/adapters/expenseapi` and `./agent/internal/api/integration_tests` with the same wrapper.
Use synthetic data and a disposable database.

Persistence tests in `internal/store/session` and `internal/runtime` also
require `AGENT_STATE_ADMIN_DATABASE_URL` and `AGENT_STATE_DATABASE_URL` in the
test environment. They provision the isolated synthetic `agent_state_test`
login and verify denial of expense-table access. The runtime tests inject
synthetic provider responses for reasoning/signature checks without inference.

The signed-transport test adapter fabricates Gateway context only inside tests.
The explicit local HTTP adapter skips IAM without fabricating a principal. Signature
construction, Worker authorization, and database results are verified separately;
they do not prove real Gateway IAM policy enforcement, deployed route/mapping
behavior, network reachability, or browser authentication. Those checks require
the deployed infrastructure. No local HTTP identity-header bypass is provided.

Agent bootstrap integration tests create and remove their own disposable database.
Role-provisioning coverage also creates and removes a unique test role.
Provide a test-only operator connection with `CREATEDB` and `CREATEROLE` in
`AGENT_MIGRATION_TEST_DATABASE_URL`, then run `go test ./agent/cmd/db-bootstrap ./agent/internal/databasebootstrap`.
They check independent backend/Agent version records, repeated application,
retained data, dirty-state rejection, refusal to adopt an existing schema,
and runtime grants denying backend access, DDL, deletion and version-table access.
