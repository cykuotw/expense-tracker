# Agent runtime

The Agent is a read-only assistant runtime built with Go ADK. Bundled skills
register a Chat orchestrator and Task capabilities; Go code owns tool grants,
model selection, and execution limits. The current executable serves a
loopback-only synthetic HTTP/SSE fixture:

```sh
go run ./agent/cmd/agent
```

`internal/api` owns chat request handling and SSE output.
`internal/clients/worker` owns outbound Worker calls. `internal/runtime`
coordinates execution, while `internal/tools` exposes the approved model tools.

It listens on `127.0.0.1:8090`. `POST /chat` accepts `conversationId`, `messageId`,
and `text` in a JSON body. Synthetic requests such as `balance demo` exercise the
temporary balance capability without paid inference. The fixture rejects remote
peers and credential-bearing requests and cannot start in Lambda. Authenticated
browser cookie chat and Lambda streaming startup are not available in this executable.

## Local mode with real Worker authorization

Set `AGENT_MODE=local` on both standalone processes. The Worker must listen on a
literal loopback address, and the Agent must receive its URL (including `API_URL`
prefix, if configured):

```sh
# Terminal 1: provide your local PostgreSQL and application JWT configuration.
AGENT_MODE=local MODE=debug BACKEND_URL=127.0.0.1:8000 go run ./backend/cmd/tracker

# Terminal 2: no AWS credentials or IAM role is needed.
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
`messageId` per turn. The deterministic model recognizes `balance <group UUID>`
and streams real Worker balance results without paid inference. It is a local
authorization/transport test, not a live-model evaluation. The default fixture
mode remains available by omitting `AGENT_MODE` or setting it to `fixture` on
the Agent; use `balance demo` only with that synthetic fixture.

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

## Worker transport

`worker.New` in `internal/clients/worker` accepts an HTTPS Worker base URL, AWS region,
credential provider, and optional HTTP transport. Runtime construction injects
this client as `runtime.Config.Balance`; its preflight runs before inference on
every turn. A trusted chat transport must attach only the current access token
using `worker.WithAccessToken`. Do not forward refresh tokens or cookies.
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
with all migrations applied. Supply `DB_PUBLIC_HOST`, `DB_PORT`, `DB_USER`,
`DB_PASSWORD`, `DB_NAME`, and `DB_SSLMODE` through the test process environment,
then set `AGENT_TEST_POSTGRES=1` and run the tests in
`./agent/internal/clients/worker` and `./agent/internal/api` with the same wrapper.
Use synthetic data and a disposable database.

The signed-transport test adapter fabricates Gateway context only inside tests.
The explicit local HTTP adapter skips IAM without fabricating a principal. Signature
construction, Worker authorization, and database results are verified separately;
they do not prove real Gateway IAM policy enforcement, deployed route/mapping
behavior, network reachability, or browser authentication. Those checks require
the deployed infrastructure. No local HTTP identity-header bypass is provided.
