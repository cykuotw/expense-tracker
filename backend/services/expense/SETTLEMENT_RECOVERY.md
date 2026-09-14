# Settlement Recovery

Use authoritative database state when a group-settlement response is lost or a
client cannot determine whether the request committed. Do not infer the result
from the client message alone and do not apply ad hoc repair SQL.

## Diagnose

1. Capture the `X-Request-ID` response header when available.
2. Find the `expense_settlement` event with that `request_id`. The event contains
   only the actor and group identifiers, operation, terminal outcome, and safe
   stage classification; it does not contain expense contents or credentials.
3. From an approved environment with the normal database configuration loaded,
   run the read-only group-scoped check:

   ```sh
   go run ./backend/cmd/reconcile-group-settlement --group-id <group-uuid>
   ```

The command takes the same group accounting lock as settlement to obtain a
stable snapshot, but it performs no database writes. Its JSON output contains
row counts, not amounts or expense contents.

## Interpret

- `committed`: no authoritative unsettled ledgers or current open balances
  remain. Refresh the group overview and do not retry.
- `not_started`: current balances still match the balances derived from
  authoritative unsettled ledgers. A normal authorized retry is safe.
- `inconsistent`: the current balances do not match authoritative unsettled
  ledgers. Stop retries, retain the request ID and output, and escalate for an
  investigated, group-scoped repair. Do not manually edit rows.

For a definite server rejection, follow the same classification before retrying
if any part of the response or connection outcome was ambiguous. A database
backup restore remains a separately controlled operator fallback, not a normal
settlement recovery action.
