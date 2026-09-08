# Notifications

The API stores subscriptions and preferences. Expense creation records pending
notifications in PostgreSQL. A scheduled sender claims a batch, sends Web Push,
and acknowledges outcomes. The mobile API and service worker do not depend on
where sending runs.

| File | Responsibility |
| --- | --- |
| `routes.go` | Authenticated subscription/settings API and endpoint validation |
| `store.go` | Subscription and preference persistence |
| `delivery_store.go` | Claim/acknowledge contracts, PostgreSQL leases, cleanup, and guarded result updates |
| `sender.go` | One batch workflow, Web Push transport, provider response classification and retry timing |
| `lambda_store.go` | IAM Lambda transport implementing the same two-method `DeliveryStore` interface |

`backend/cmd/push-sender` wires Sender directly to PostgreSQL for local execution.
`backend/cmd/push-sender-serverless` wires the same Sender to LambdaStore.
`backend/cmd/push-delivery-serverless` dispatches claim/acknowledge requests to
PostgreSQL. These entrypoints configure dependencies; they contain no delivery
policy. LambdaStore has no database credentials, buffering, or retry policy.

Claims use `FOR UPDATE SKIP LOCKED` and a two-minute lease with a random token.
Acknowledgements must match an unexpired token; duplicate and stale reports
cannot overwrite newer work. Each claim rechecks membership, group activity,
mute preferences, subscription existence, and delivery expiry. Changes after a
claim may not affect an already-running send. The sender reserves time to
acknowledge completed outcomes before its execution deadline.

Keep the lease longer than the sender's total execution limit. Keep invocation
retries disabled in the sender's AWS client: a lost claim response should recover
through lease expiry, not claim an additional batch in a hidden SDK retry.
Provider acceptance followed by a lost acknowledgement can cause duplicates.

For deployment, secrets, concurrency, and recovery, see the
[operator guide](../../../deployment/serverless/README.md).
Notification infrastructure is grouped in
[`notifications.tf`](../../../deployment/serverless/infrastructure/tf/notifications.tf).
