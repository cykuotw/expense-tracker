# Unified Serverless Deployment

This directory is the complete serverless deployment implementation. It owns the database host, Worker and Bootstrap Lambdas, API custom domain, and frontend infrastructure/publication through one Python entrypoint and one Terraform state.

## Operator configuration

Create the protected external configuration once:

```bash
make deploy ACTION=init
```

The default path is `~/.config/expense-tracker/deploy.json`. To keep it elsewhere, pass an absolute path when creating and using it:

```bash
DEPLOY_CONFIG_FILE=/home/your-user/exp-env/deploy.json make deploy ACTION=init
chmod 600 /home/your-user/exp-env/deploy.json
```

The config file, SSH key, and optional Google token must be regular non-symlink files outside the repository with mode `0600`. `ACTION=init` refuses to overwrite an existing file.

The JSON sections are `deployment`, `aws`, `database`, optional `backup`,
`backend`, `observability`, `frontend`, optional `first_admin`, and
`local_credentials`.
This is the sole human-edited serverless deployment source. Do not create
serverless `.tfvars`, PostgreSQL password files, or Lambda runtime JSON files;
the deployer creates protected temporary projections and removes them.

Release history does not add a deploy-config field. The deployer supplies the
internal Terraform `use_lambda_aliases` cutover variable itself; do not add it
to `deploy.json` or create a `.tfvars` file.

Complete example—replace every `REPLACE` value before deployment:

```json
{
    "deployment": {
        "account_id": "123456789012",
        "name_prefix": "expense-tracker",
        "environment": "production",
        "tags": {
            "Project": "expense-tracker",
            "Environment": "production"
        }
    },
    "aws": {
        "region": "ca-central-1",
        "vpc_id": "vpc-REPLACE",
        "subnet_id": "subnet-REPLACE",
        "key_pair_name": "REPLACE",
        "operator_ssh_cidr": "203.0.113.10/32",
        "hosted_zone_name": "example.com"
    },
    "database": {
        "name": "expense_tracker",
        "admin_user": "expense_admin",
        "admin_password": "REPLACE_WITH_RANDOM_ADMIN_SECRET",
        "migration_user": "expense_migration",
        "migration_password": "REPLACE_WITH_RANDOM_MIGRATION_SECRET",
        "runtime_user": "expense_runtime",
        "runtime_password": "REPLACE_WITH_RANDOM_RUNTIME_SECRET",
        "instance_type": "t4g.micro",
        "ami_id": null
    },
    "backend": {
        "api_hostname": "api.example.com",
        "google_client_id": "REPLACE.apps.googleusercontent.com",
        "jwt_secret": "REPLACE_WITH_AT_LEAST_32_RANDOM_CHARACTERS",
        "jwt_exp": 900,
        "refresh_jwt_secret": "REPLACE_WITH_AT_LEAST_32_RANDOM_CHARACTERS",
        "refresh_jwt_exp": 604800,
        "expenses_per_page": 25,
        "db_conn_max_lifetime_seconds": 300,
        "db_conn_max_idle_time_seconds": 60,
        "web_push_vapid_public_key": "REPLACE_WITH_VAPID_PUBLIC_KEY",
        "web_push_vapid_private_key": "REPLACE_WITH_VAPID_PRIVATE_KEY",
        "web_push_vapid_subject": "mailto:ops@example.com"
    },
    "observability": {
        "discord_webhook_url": null
    },
    "frontend": {
        "hostname": "expense.example.com"
    },
    "first_admin": {
        "email": "admin@example.com",
        "password": "REPLACE_WITH_STRONG_ADMIN_PASSWORD",
        "firstname": "Admin",
        "lastname": "User",
        "nickname": "admin"
    },
    "local_credentials": {
        "ssh_private_key_file": "/home/your-user/.expense-tracker.pem",
        "google_id_token_file": null
    }
}
```

Set `"first_admin": null` when no initial administrator should be created. `nickname` may be an empty string. The application generates the administrator's user ID; no ID belongs in this file.

Set `observability.discord_webhook_url` to the Discord webhook URL, or leave it
`null` to disable Discord error alerting. The protected deploy config is the
only human-managed source for this value. During `ACTION=deploy` or a backend
update, the deployer writes it directly to the notifier Lambda environment
through the same mode-`0600` protected temporary projection used for other
runtime secrets such as `jwt_secret`. Terraform receives only the enable/disable
boolean and never receives the URL. The notifier validates the configured URL
at startup and keeps the parsed value only in process memory.

When the field is `null`, Terraform omits the notifier Lambda, notifier log
group, IAM role, invoke permission, and Worker subscription. Disabling an
existing installation removes those managed resources but deliberately retains
the value in the protected external deploy config until the operator removes
it. This design adds no fixed-monthly-cost alerting service. Lambda and
CloudWatch Logs usage can still be billed if the account exceeds the applicable
shared free-tier allowances, so an absolute zero bill cannot be guaranteed.

The access-token lifetime must be between 60 and 86,400 seconds. The refresh
lifetime must be between 300 and 31,536,000 seconds and greater than the access
lifetime. The two JWT secrets must be different and at least 32 bytes;
length is a minimum safeguard, not proof of entropy. `expenses_per_page` is
limited to 1–1,000. Database connection lifetime and idle-time values are
limited to 0–86,400 seconds, where `0` disables the corresponding expiration.
The deployed Worker retains the stricter two-connection application pool budget.
The request-server runtime permits at most 100 open database connections, with
idle connections between zero and that configured open-connection limit.

## Local build-tool provisioning

The machine that runs `make deploy` builds and tests the frontend locally before
publication. It must provide the following commands on `PATH` before invoking
the deployer:

- Node `22.23.2` through Node `22.x`; `frontend/.node-version` identifies the
  tested baseline and `frontend/package.json` enforces the supported range.
- pnpm `11.25.0` through `12.x`; `frontend/package.json` pins the tested `12.4.2` version.

pnpm requires Node, and the deployer intentionally does not download or install
either tool. The serverless preflight checks that `node` and `pnpm` are
available and rejects unsupported Node or pnpm versions before it performs AWS
operations. On a new machine, provision Node and pnpm using the operator's
normal user-level or CI tool manager, then verify `node --version` and
`pnpm --version`. Do not add a Node binary to this repository or to frontend
application dependencies.

For each frontend publication, the deployer runs `pnpm install
--frozen-lockfile`, `pnpm run test:run`, and `pnpm run build` from `frontend/`.

## Commands

```bash
export DEPLOY_CONFIG_FILE=/home/your-user/exp-env/deploy.json

make deploy
make deploy ACTION=plan
make deploy ACTION=deploy
make deploy ACTION=update SCOPE=migrations
make deploy ACTION=update SCOPE=backend
make deploy ACTION=update SCOPE=frontend
make deploy ACTION=update SCOPE=all
make deploy ACTION=status
make deploy ACTION=history
make deploy ACTION=show RELEASE=<exact-release-id>
make deploy ACTION=rollback RELEASE=<exact-release-id> SCOPE=backend|frontend|all
make deploy ACTION=promote RELEASE=<exact-release-id> SCOPE=backend|frontend|all
make deploy ACTION=cleanup
make deploy ACTION=backup-configure
make deploy ACTION=restore-verify
make deploy ACTION=backup-status
make deploy ACTION=backup-cleanup
make deploy ACTION=destroy
```

`make deploy` deploys/resumes an incomplete environment and runs migrations → backend → frontend when the environment is complete. Initial infrastructure requires typing the configured name prefix. Destroy requires typing `destroy-<name_prefix>`.

`SCOPE=backend` also publishes and invokes the bootstrap function first. This applies pending migrations and verifies first-administrator reconciliation before marker-dependent Worker code is published. Use `SCOPE=migrations` when only the migration/bootstrap step should run.

## Release history and application rollback

Successful deployments publish immutable numbered Lambda versions, route all
production invocations through stable `live` aliases, and write a compact
non-secret manifest to standard-tier SSM Parameter Store. Each manifest is a
complete resolved state: a scoped frontend update carries forward the current
backend versions, and a scoped backend update carries forward the current
frontend snapshot. `ACTION=status` shows the active and previous release;
`ACTION=history` lists retained releases; `ACTION=show` validates and prints one
exact manifest.

Release-producing commands require a clean Git worktree so the recorded commit
SHA identifies the exact deployed source. Git-ignored caches are allowed, but
staged, unstaged, and non-ignored untracked files must be committed or removed.

The first update of an existing installation must use `ACTION=update SCOPE=all`
(or plain `make deploy`, which selects the all-scope update for a complete
installation). It establishes the `live` aliases and adopts the installed
backend as a baseline before publishing the candidate. Scoped updates fail
before artifact construction until this one-time cutover succeeds. The adopted
baseline cannot restore the legacy frontend because it predates snapshots;
the successful all-scope update creates the first restorable frontend snapshot.
No deploy-config migration or new deploy-config field is required.

Rollback and promote both activate previously stored artifacts without
checking out or rebuilding the old commit. Use `rollback` when moving to an
older known-good release and `promote` when moving forward again. Both commands
require the full release ID, print the exact Lambda versions or frontend
snapshot, and require typing `rollback-<release-id>` or
`promote-<release-id>`. A successful activation writes a new immutable
composite manifest, so history records the actual resulting backend/frontend
combination without a second event-log subsystem.

Activation changes application artifacts only. It rejects a backend target
whose optional Error Notifier enabled/disabled topology differs from the
current release, because that transition requires a normal reviewed Terraform
deployment.

Database schema is forward-only during application rollback. The deployer
reads the current migration state and rejects dirty state, unknown migration
metadata, a target recorded against a newer schema, a repository manifest
digest mismatch, or any intervening migration whose
`rollback.application` is not `compatible`. It never runs a down migration,
forces a migration version, or restores a database backup as part of release
activation. Use a reviewed forward fix when compatibility cannot be proven.

Frontend releases keep content-hashed `assets/` objects shared and immutable.
Every other built file is copied into
`releases/<release-id>/frontend/root/`, with a digest-checked snapshot
descriptor. Activation verifies the descriptor, every referenced shared asset,
and each mutable snapshot object before copying files to the live root and
invalidating CloudFront. If a later deployment step fails, the deployer attempts
to restore the prior aliases and frontend snapshot before reporting the error.

The same bounded cleanup runs automatically after each release becomes current;
if it fails, the command reports that the release is already active and directs
the operator to repair cleanup separately. `ACTION=cleanup` previews that plan
without applying it. Cleanup always protects the active and immediately
previous releases, keeps at most five successes, expires other unprotected
successes after 30 days, and reports stale candidates, Lambda versions,
snapshots, and unreferenced assets. Review the exact plan, then run:

```bash
SERVERLESS_CLEANUP_APPLY=true make deploy ACTION=cleanup
```

Applied cleanup requires typing `cleanup-<name_prefix>`. Failed candidates are
eligible after 24 hours. Cleanup will not prune shared frontend assets while
the current release still represents an adopted legacy frontend without a
snapshot.

`ACTION=destroy` removes the deployment-scoped release manifests, candidates,
and current pointer only after the owned infrastructure is verified absent.
If infrastructure was already removed but those parameters remain, rerunning
destroy requires the normal `destroy-<name_prefix>` confirmation before
removing the stale metadata. PostgreSQL backup deletion remains a separate,
explicitly guarded operation.

The design adds no always-on service or fixed monthly charge. It reuses the
existing Lambda, S3, and CloudFront resources and standard-tier SSM parameters.
Cost remains an architecture and post-deployment billing review concern; the
deployer does not print or enforce a static monthly estimate.

## Frontend security headers

The existing CloudFront response headers policy enforces
`Referrer-Policy: strict-origin-when-cross-origin`,
`X-Content-Type-Options: nosniff`, and `X-Frame-Options: DENY`. It also enforces
a narrow `Content-Security-Policy` that was first validated in report-only mode
across the supported Chromium and Safari flows. The allowlist covers the
configured API origin, same-origin application and PWA resources, Google
Identity Services, and the Roboto font files used by the frontend. Production
observation found that both the application UI libraries and Google Identity
Services require inline CSS, so `style-src` permits `unsafe-inline`. That
exception applies only to styles; `script-src` remains free of `unsafe-inline`
and `unsafe-eval`. The policy has no reporting endpoint or broad wildcard.

After deploying a frontend infrastructure update, inspect the browser console
for CSP violations while testing local and Google sign-in, authenticated API
requests, PWA installation and updates, Web Push, and the supported Safari
flows. Enforced-policy violations can block those operations, so treat any new
violation as a failed deployment verification and restore the preceding
report-only configuration through Terraform.

Security headers are Terraform-managed infrastructure, not application release
artifacts. `ACTION=rollback` and `ACTION=promote` do not change them. To recover
from a bad header update, restore the preceding reviewed
`aws_cloudfront_response_headers_policy.frontend_security` configuration and
run `make deploy ACTION=update SCOPE=frontend`; then verify the live response
headers and affected browser flows again. HSTS is intentionally deferred until
the complete hostname scope is proven HTTPS-only.

The canonical [schema migration policy](../../backend/cmd/migrate/MIGRATION_POLICY.md)
defines the `000035` metadata baseline, expand/contract sequence, online
execution limits, and recovery rules. The `migrations`, `backend`, and `all`
scopes are all migration-bearing scopes and must satisfy that policy. Before
any remote or database mutation, the deployer validates the manifest. If the
manifest contains maintenance history, an update queries the currently deployed
Bootstrap through its read-only `migration-state` operation and rejects only
pending maintenance-required entries. Deploy this inspection capability before
adding the first such entry. Bootstrap artifact construction and the Bootstrap
runtime validate the same manifest again; runtime validation also rejects a
dirty database migration state before applying pending migrations.
Fresh plan and deploy operations apply the same normal-deployment policy before
creating infrastructure.

Web Push uses two small Lambdas. The scheduled Sender runs outside the VPC and
contacts public Push providers. It invokes the VPC-connected Delivery function
through the IAM-authorized Lambda API to claim pending work and acknowledge a
batch of results. Delivery connects only to PostgreSQL; it does not call the
public Lambda API. No NAT Gateway, paid VPC endpoint, or permanent public IPv4
is required for notifications. Costs are usage-based Lambda execution, logs, and
data transfer, with no additional fixed network charge.

Generate one VAPID key pair, keep the private key in the protected deployment
configuration, and set `web_push_vapid_subject` to an operator contact such as
`mailto:ops@example.com`. Existing VAPID configuration and mobile subscriptions
remain valid. The deployer derives the Delivery function name automatically.
Only Sender receives VAPID signing credentials; only Delivery receives database
credentials. The API Worker receives the public VAPID key.

Use `ACTION=update SCOPE=all` to publish migrations, both notification functions,
and the frontend. Updates apply migrations before pausing an existing Sender,
then wait up to its 60-second execution limit before replacing notification
code. Delivery is configured and activated before Sender. If a later update
step fails, the deployer restores the Sender's previous reserved concurrency
before returning the failure. The obsolete notification HTTPS security group
rule is the only notification resource explicitly allowed to be deleted during
this migration.

The schedule runs every minute, claims up to 10 deliveries, and acknowledges
results in one batch. An empty poll makes one Delivery invocation; a nonempty
poll normally makes two. Leases expire after two minutes, transient failures
are retried up to three reported attempts within the delivery expiry, and
404/410 responses retire subscriptions. A push accepted before a lost
acknowledgement can be sent again; exactly-once delivery is not promised.
Concurrency is Worker `2`, Sender `1`, Delivery `1`, and Bootstrap `1` (five
reserved executions in total, subject to the account's reservation quota).

A second product-wide EventBridge rule invokes the existing Delivery Lambda at
05:15 UTC each day to publish at most 25 eligible closed Home-group months. It
does not create a new Lambda or scan from the every-minute Sender tick. Review
reads calculate current totals directly from PostgreSQL, while publication and
notification deduplication use one durable group/month marker.

`SCOPE=all` also creates and manages the database EC2 Instance Connect
Endpoint. It uses a dedicated security group with SSH-only access to the
database host; it does not create a public IP or an SSM interface endpoint.
Remove any manually created endpoint first because AWS permits only one
endpoint per VPC and subnet.

Use `SCOPE=all` for the account-linking release so the local-password capability migration, Google-authorized API route, Worker code, and Account Settings UI are published together. The migration preserves password capability for existing local accounts and classifies existing Google-created accounts as Google-only.

Use `SERVERLESS_AUTO_APPROVE=true` only in a controlled disposable test. `FORCE_DETACH_LAMBDA_ENI=true` permits the bounded owned-ENI force cleanup only after the normal 20-minute wait plus five additional minutes.

## PostgreSQL backup and restore verification

`make deploy ACTION=backup-configure` provisions the private logical-backup
bucket and the database instance's least-privilege upload role, briefly enables
the existing restricted SSH bootstrap path, installs a root-owned systemd timer,
and runs the first backup. The timer writes a PostgreSQL custom-format dump to
the bucket every day. S3 encrypts the objects at rest and deletes daily dumps
after 90 days.

The compatibility default is `03:17:00 UTC`. To schedule at a fixed local
time, add this non-secret block to the protected deployment configuration and
run `ACTION=backup-configure` again:

```json
"backup": {
  "time": "01:17:00",
  "timezone": "America/Toronto"
}
```

`timezone` must be an available IANA zone name. With the example above, the
timer remains at 1:17 AM Toronto time through daylight-saving transitions.

`make deploy ACTION=restore-verify` creates a temporary, isolated PostgreSQL
16 host. It downloads the newest retained dump, restores it locally, verifies
the core application schema and queryable data, writes the last successful
verification marker to the backup bucket, and removes the temporary host,
security group, and public IP even when the restore fails. It never writes to
the production database. The command requires typing
`restore-verify-<name_prefix>` unless `SERVERLESS_AUTO_APPROVE=true` is set for
a controlled disposable test.

`make deploy ACTION=backup-status` prints the newest retained backup object,
the latest backup status, and the timestamp of the latest successful
restore-verification marker. The backup timer records its failures in
`expense-tracker-postgres-backup.service` in the database host's systemd
journal; a stale latest-backup timestamp or status is therefore an actionable
failure signal. A failed interactive backup configuration also uploads a
diagnostic report for this status command.

Backups intentionally make application teardown a deliberate operation.
`ACTION=destroy` refuses before changing application resources while the backup
bucket contains objects. Retain the bucket for recovery, or run
`ACTION=backup-cleanup` only after the retention decision is approved; it checks
the deployment ownership tags and requires typing `backup-cleanup-<name_prefix>`
before clearing retained data. Then run `ACTION=destroy` separately. The bucket
is not configured for forced deletion.

## Worker log investigation

The Worker writes structured, line-delimited JSON to its stable CloudWatch log
group, `/aws/lambda/<name_prefix>-<environment>-worker`. Terraform retains this
log group across Worker code updates. Its retention is controlled by
`worker_log_retention_days`, which defaults to three days and accepts only
periods supported by CloudWatch Logs. The Bootstrap, Sender, and Delivery log
groups keep their existing retention settings.

Treat three days as the operational evidence window: start an investigation as
soon as an error is reported and do not rely on older events being available.
In CloudWatch Logs Insights, select only the Worker log group and the narrowest
useful time range. Structured JSON fields are discovered automatically. These
queries intentionally display allowlisted diagnostic fields instead of the raw
`@message`.

Recent alertable errors:

```text
fields @timestamp, level, event, request_id, route, status, error_code, error_category, diagnostic_message
| filter alertable = true
| sort @timestamp desc
| limit 100
```

All events for an application request ID:

```text
fields @timestamp, level, event, method, route, status, error_code, error_category
| filter request_id = "REPLACE_WITH_REQUEST_ID"
| sort @timestamp asc
| limit 100
```

Count 5xx failures by registered route and public error code:

```text
filter event = "unexpected_http_error" and status >= 500
| stats count(*) as failures by route, error_code
| sort failures desc
```

Recovered panics:

```text
fields @timestamp, request_id, route, error_category, diagnostic_message
| filter event = "panic_recovered"
| sort @timestamp desc
| limit 100
```

Correlate an API Gateway or Lambda request ID:

```text
fields @timestamp, level, event, request_id, api_gateway_request_id, aws_request_id, route, status, error_code
| filter api_gateway_request_id = "REPLACE_WITH_API_GATEWAY_REQUEST_ID"
    or aws_request_id = "REPLACE_WITH_AWS_REQUEST_ID"
| sort @timestamp asc
| limit 100
```

Request IDs are correlation keys, not proof of identity. Keep query results in
approved operational systems and avoid copying log records into tickets or chat
without checking them for sensitive data.

CloudWatch Logs subscription delivery and Discord posting are best-effort and
at-least-once. Retryable Discord failures can cause AWS to deliver a
batch again, so duplicate Discord posts are possible. The notifier has no
persistent deduplication store and does not claim exactly-once delivery.

### Authorized alerting smoke check

Run this only after explicit approval for a live deployment check. Create a
uniquely named temporary stream in the Worker log group and publish one
sanitized JSON event with `alertable=true`,
`event="unexpected_http_error"`, `status=500`, a synthetic `request_id`, and no
request body, headers, user data, credentials, or webhook material. Then:

1. Find the event by its synthetic `request_id` with the allowlisted Logs
   Insights query above.
2. Confirm Discord receives a sanitized message containing the same request ID.
   At-least-once delivery means duplicates are possible.
3. Inspect notifier logs for the request ID and confirm that neither logs nor
   Discord contain the webhook or any sensitive canary.
4. Set `discord_webhook_url` to `null`, run a backend update, confirm Terraform
   removes only the six conditional alerting resources, and repeat with a new
   synthetic ID to confirm no Discord delivery occurs.

The smoke event itself consumes CloudWatch Logs and, while enabled, Lambda
usage. Do not run it when the requirement is strictly no additional metered
usage, even though a single check will ordinarily fit within shared free-tier
allowances.

## Safety boundary

- Terraform never receives database/JWT/first-admin/webhook secrets and never manages Lambda environments.
- The Worker begins at reserved concurrency `0`; Python publishes runtime configuration and activates it at `5`. With the deployed `DB_MAX_OPEN_CONNS=2`, the Worker has a maximum application-side database pool budget of 10 connections.
- The raw execute-api endpoint is disabled only after custom-domain and frontend checks pass.
- Normal updates use narrowly targeted Terraform plans for supported API, CloudFront, notification, and database-support infrastructure changes; Lambda code and runtime environments remain owned by the deployment runtime after initial creation. Deletions are limited to the two retired invitation routes, the obsolete notification HTTPS egress rule, and the six conditional alerting resources when alerting is explicitly disabled; replacements and unrelated deletions are rejected.
- Before an update, the deployer removes only unmanaged Worker/Bootstrap/Sender/Delivery/Error Notifier runtime environments if an AWS provider response persisted them into local state, then verifies that no configured protected value remains anywhere in Terraform artifacts.
- Destroy deletes Lambdas first, waits for their owned ENIs, then runs `terraform destroy -refresh=false`.
- A deployment failure keeps persistent resources for an explicit resume; it never performs automatic rollback or destroy.
