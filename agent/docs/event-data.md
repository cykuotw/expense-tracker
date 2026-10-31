# Event payload formats

`expense_agent.events.event_type` determines how to interpret the `data` JSONB
column. There is no `schema_version` or `call_id` column. Payload additions
normally need only writer/reader updates; existing readers ignore unknown JSON
fields. Changing or removing fields used to restore history still requires
compatible readers or conversion of the affected records.

## Execution and diagnostic events

Run lifecycle events, visible output, and model/tool diagnostics use this
envelope, defined by `Snapshot` in
[the shared models](../internal/models/persistence.go):

```go
type Snapshot struct {
    CallID        string         `json:"callId,omitempty"`
    Data          jsontext.Value `json:"data"`
    Complete      bool           `json:"complete"`
    OriginalBytes int            `json:"originalBytes"`
    RetainedBytes int            `json:"retainedBytes"`
    Reason        string         `json:"reason,omitempty"`
}
```

`CallID` is optional model/tool correlation metadata. It is extracted from the
writer's diagnostic map, sanitized, and stored only at the envelope level.
It survives content omission. `Data` contains the event-specific content:

| event_type | Snapshot.data contents |
| --- | --- |
| `run.started` | `Run`: `id`, `clientMessageId`, `messageId`, `status`, `text`, optional `errorCode`, `startedAt` |
| `run.finished` | `status` and optional `errorCode` |
| `output` | `Output`: output kind, external conversation/run/message IDs, invocation links, skill/model metadata, text delta and error code |
| `model.request` | Agent, branch, skill/model metadata and provider request |
| `model.response` | Provider response, reasoning availability and optional rejection reason |
| `model.finished` | Agent/skill/model metadata, duration and status |
| `tool.started` | Tool name/input and agent context |
| `tool.finished` | Tool result or error, timing and any tool-specific metadata |

Diagnostic content depends on its writer; it is not a fixed set of dashboard
fields. A dashboard correlates diagnostic events through
`events.data->>'callId'`. Diagnostic fields remain separate from normal chat
history and UI replay.

For example, an oversized diagnostic may be recorded as:

```json
{
  "callId": "example-call",
  "data": null,
  "complete": false,
  "originalBytes": 150000,
  "retainedBytes": 0,
  "reason": "payload_limit"
}
```

The byte counts measure the event-specific content before/after credential
sanitization, excluding the envelope and its correlation metadata. Required run
and output payloads fail if oversized; optional diagnostics can be omitted as
shown above.

`Run` is defined in the shared persistence models linked above; `Output` and
`ChatRequest` are defined in [the shared chat models](../internal/models/chat.go).
The `eventId`
and `createdAt` returned by output replay are populated from the database row;
the snapshot is captured before those stored identifiers are attached.

## ADK history

For `adk.event`, `events.data` contains the sanitized SDK `session.Event` directly,
without the `Snapshot` envelope. Its content includes the native event and
invocation IDs, author, timestamp, content parts, actions/state deltas, branch
and other SDK event metadata. Native function-call IDs remain in their content
parts. The runtime reads these events to restore model history. The native ADK
event ID exists only in this payload; there is no separate source-event column
or uniqueness check. Each append creates a new database row and applies its state
delta atomically, even if a caller supplies a previously used native ID. Automatic
event retry/redelivery is outside the MVP.

The definition is the installed `google.golang.org/adk/v2/session.Event`, which
embeds `model.LLMResponse` (currently ADK v2.6.0). There is no separate local
copy of this SDK contract. Inspect the exact definitions with:

```sh
go doc google.golang.org/adk/v2/session.Event
go doc google.golang.org/adk/v2/session.EventActions
go doc google.golang.org/adk/v2/model.LLMResponse
```

The persisted JSON can include these fields when populated:

| Group | JSON fields |
| --- | --- |
| Identity and agent context | `id`, `timestamp`, `invocationId`, `author`, `branch`, `isolationScope` |
| Message and tool content | `content.role`, `content.parts` including text, function calls/responses, thought flags and thought signatures |
| Actions | `actions.stateDelta`, `actions.artifactDelta`, `actions.requestedToolConfirmations`, `actions.skipSummarization`, `actions.transferToAgent`, `actions.escalate`, `actions.compaction` |
| Workflow metadata | `longRunningToolIds`, `routes`, `requestedInput`, `output`, `nodeInfo` |
| Provider metadata | `citationMetadata`, `groundingMetadata`, `usageMetadata`, `customMetadata`, `logprobsResult`, `inputTranscription`, `outputTranscription`, `modelVersion`, `avgLogprobs` |
| Completion and errors | `partial`, `turnComplete`, `interrupted`, `sessionResumptionHandle`, `errorCode`, `errorMessage`, `finishReason` |

This is sanitized execution history, not an unmodified provider archive:

- The session writer skips partial ADK events and removes `temp:` state deltas;
  shared `app:`/`user:` state deltas are rejected.
- Credential filtering applies to persisted JSON. Required history over 128 KiB
  fails instead of being silently clipped.
- Before ADK receives model responses, the model wrapper removes custom metadata
  and unsigned thought parts. Signed parts retain their signatures and original
  part association. Pre-projection `model.response` diagnostics retain the
  available response metadata and reasoning under the diagnostic size policy.
- `model.request` captures the prepared request's `Model`, `Contents` and `Config`,
  not the complete SDK request/client object. `model.response` captures ADK's
  `model.LLMResponse`, not the provider's raw HTTP response or all candidates.

Execution diagnostic payloads are currently writer-defined maps rather than one
typed union or JSON Schema. `Snapshot`, `Run` and `Output` are local Go
contracts; SDK types define ADK history and model response contents.

## Historical resource-binding records

Older databases may contain `resource.bound` records. They are no longer written
or consulted for history access or conversation continuation. Existing records
remain untouched; canonical ADK history and output readers select their own
explicit event types. Current domain data is obtained through authorized tools,
while saved output remains an owner-scoped historical snapshot.
