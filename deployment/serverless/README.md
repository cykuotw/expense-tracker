# Unified Serverless Deployment

This directory is the complete serverless deployment implementation. It owns the database host, Worker, OCR, and Bootstrap Lambdas, API custom domain, and frontend infrastructure/publication through one Python entrypoint and one Terraform state.

## AWS authentication

The serverless configuration requires AWS provider 6.23 or later within the
6.x series to support native `aws login` credentials. The checked-in Terraform
lock file selects the tested provider version. After pulling a provider upgrade,
run `terraform -chdir=deployment/serverless/infrastructure/tf init -input=false`
to install the locked version before deploying.

Authenticate with `aws login` (AWS CLI 2.32 or later), then run the deployment
command in the same environment. For a named profile, use `aws login --profile
<profile>` and set `AWS_PROFILE=<profile>` for deployment. Existing supported
credential sources, including IAM Identity Center profiles, remain usable.
Preflight uses the AWS CLI; Terraform resolves credentials independently through
its provider, so a successful preflight alone does not verify provider login.
See the [AWS login guide](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-sign-in.html).

After a provider upgrade, a targeted plan can retain older-schema state for
unrelated resources. If `terraform show -json` cannot serialize that prior state,
the deployer can inspect the actions from the successful plan's machine-readable
events instead. This fallback is limited to the prior-state schema-version error,
checks the saved plan's digest and event totals, and retains the same deletion
and replacement allowlists. It does not rewrite state, refresh unrelated
resources, or expand the deployment scope. Other inspection failures still stop
deployment. No manual state edits or full-scope deployment are required solely
to work around this inspection error.

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
        "ocr_capability_secret": "REPLACE_WITH_A_DISTINCT_RANDOM_32_BYTE_SECRET",
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
lifetime. The two JWT secrets and `ocr_capability_secret` must be pairwise
different and at least 32 bytes;
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
- Go with toolchain selection support. Lambda artifact builds select the exact
  version declared by `go.mod` (currently `1.27.3`) using `GOTOOLCHAIN`. Go may
  download that toolchain if it is not cached, so the build machine needs access
  to the configured Go toolchain download source.

Artifact builds clear inherited `GOROOT` for the build subprocess so the selected
Go driver uses its matching compiler, assembler and standard library. An old
shell `GOROOT` or `GOTOOLCHAIN=local` cannot force Lambda builds onto a different
installation. This does not modify the shell, system Go installation, or
deployment configuration. Errors such as `compile: version "go1.25.14" does not
match go tool version "go1.27.3"` indicate a mixed toolchain; use this checkout's
artifact builder rather than overriding the selected toolchain's `GOROOT`.

pnpm requires Node, and the deployer intentionally does not download or install
either tool. The serverless preflight checks that `node` and `pnpm` are
available and rejects unsupported Node or pnpm versions before it performs AWS
operations. On a new machine, provision Node and pnpm using the operator's
normal user-level or CI tool manager, then verify `node --version` and
`pnpm --version`. Do not add a Node binary to this repository or to frontend
application dependencies.

For each frontend publication, the deployer runs `pnpm install
--frozen-lockfile`, `pnpm run test:run`, and `pnpm run build` from `frontend/`.
Frontend-only updates do not build backend Lambda artifacts. Migration-only
updates build only Bootstrap; backend and all-scope updates build the six
independent Go entrypoints with bounded concurrency.

Update releases compare each candidate Lambda zip hash and exact runtime
environment with the currently active immutable version. An unchanged function
is carried forward in the new release manifest instead of publishing another
numbered version. Sender is drained for up to 60 seconds only when Sender or
Delivery code/configuration changes; other backend-only changes do not incur
that fixed wait. When Bootstrap is reused, one read-only migration-state probe
replaces its two idempotency invocations when the live database schema is clean
and current; that response is also reused in the release manifest.

Frontend versions are content-addressed. After a verified frontend build, an
update compares its complete snapshot descriptor with the active snapshot. An
identical build carries the active snapshot forward without S3 publication or
CloudFront invalidation; changed output still follows the full snapshot,
restore, invalidation, and verification path.

Major deployment steps, including preflight and automatic cleanup, print
`elapsed_seconds` on completion. Automatic cleanup reuses the outputs and
health checks validated by the successful deployment, batches stale S3 asset
deletion, and removes stale Lambda versions with bounded concurrency.

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

Normal updates migrate forward to the latest repository schema, then publish
the selected application components. They do not require compatibility with
the previous application. Migration validation still rejects dirty or unknown
database state, a database newer than this checkout, and pending migrations
that require maintenance.

If an update fails, applied migrations and application changes remain in place.
The deployer records the failed candidate and restores paused notification
concurrency, but does not restore older Lambda versions or frontend files.
Fix the reported cause and rerun `make deploy ACTION=update SCOPE=all` from a
clean committed checkout. Until that succeeds, the current release pointer
describes the last successful deployment and may differ from the live components.
All-scope updates report existing application health failures as warnings so a
forward fix can proceed; API and frontend verification must pass after publication.

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
deployment. It also rejects activation across the pre-OCR/OCR infrastructure
boundary. The first reviewed backend or all-scope update creates the OCR
resources without aliases, publishes the numbered OCR version and `live`
alias, then reapplies the route against that alias; later releases promote OCR
with the rest of the backend.

Database schema is forward-only during application rollback. The deployer
reads the current migration state and rejects dirty state, unknown migration
metadata, a target recorded against a newer schema, a repository manifest
digest mismatch, or any intervening migration whose
`rollback.application` is not `compatible`. It never runs a down migration,
forces a migration version, or restores a database backup as part of release
activation. A legacy current-release record whose schema was omitted as zero
is recovered only by reading the live Bootstrap migration state and validating
that it is clean and described by the repository manifest; the next successful
deployment records the corrected version without rewriting immutable history.
Use a reviewed forward fix when compatibility cannot be proven.

Frontend releases keep content-hashed `assets/` objects shared and immutable.
Every other built file is copied into
`releases/<release-id>/frontend/root/`, with a digest-checked snapshot
descriptor. Activation verifies the descriptor, every referenced shared asset,
and each mutable snapshot object before copying files to the live root and
invalidating CloudFront. Explicit `rollback` and `promote` commands retain their
compatibility checks and attempt to restore prior artifacts if activation fails;
normal updates use the forward-fix behavior described above.

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
Identity Services, and the Roboto font files used by the frontend. Receipt
previews additionally allow `blob:` only in `img-src`, so locally prepared
images can be reviewed without permitting blob scripts, workers, or network
connections. Saved receipt photos allow the exact configured S3 bucket origin
in both `img-src` and `connect-src`: Workbox forwards signed image requests via
`fetch()`, which requires the connection directive. Signed URLs use a no-store,
network-only route and never enter Workbox caches. Deployment verification rejects
missing origins in either directive, other buckets, and wildcard S3 origins.

Publish receipt CSP fixes through `make deploy ACTION=update SCOPE=frontend` so
CloudFront headers and the changed service-worker bundle are updated together.
Existing installations must activate the waiting worker through the normal app
update flow before reopening a receipt for a fresh short-lived URL. Validate
this browser contract locally after building the frontend with
`uv run python frontend/e2e/receipt_view_csp_browser_check.py`; the check uses only
synthetic local images and compares the old and corrected policies.

Production observation found that both the application UI libraries and Google Identity
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

### Optional database operating hours

To stop the database overnight, configure these non-secret sections in the
protected external `deploy.json`:

```json
"operating_hours": {
  "enabled": true,
  "start": "09:00",
  "stop": "22:00",
  "safety_stop": "01:00",
  "timezone": "America/Toronto"
},
"backup": {
  "time": "21:00:00"
}
```

`operating_hours.timezone` is the shared source for EC2 actions, monthly reviews,
backup timing/freshness, Sender policy, and frontend notices. Omit
`backup.timezone`; it inherits this value even when `enabled` is `false`.
Existing matching duplicate values remain compatible; conflicting duplicates
are rejected with instructions to remove `backup.timezone`. Always-on deployments
without an `operating_hours` block retain standalone `backup.timezone` support
and the existing UTC default.

Install application support first with the normal `make deploy ACTION=update
SCOPE=all`. A successful deployment applies the configured hours afterward.
Activation checks the deployed frontend and live Sender capability versions;
partial application upgrades cannot activate a schedule. Runtime installation
initializes a disabled public policy before Sender activation, including a
backend-only update of an always-on installation. Once both components are
installed, later enablement and time changes require no rebuild.
Subsequent configuration changes require only:

```sh
make deploy ACTION=hours-apply
```

This action builds no application artifacts and performs no application release.
Retain the local Terraform initialization and existing deployment artifacts used
by the supported deployer. It validates configuration before mutation, stages
Terraform-owned schedules disabled, waits two minutes for old schedule events to
expire, reconfigures the backup when policy/settings changed, publishes the public
policy, then enables schedules. Backup configuration uses the existing restricted
temporary SSH path and runs a measured backup. The apply refuses activation when
that run fails, its duration is unknown, or dump/upload exceeds 45 minutes. Allow
at least an hour before shutdown for a changed configuration. A failure reports
the incomplete stage; inspect schedule states and rerun the same action. Do not
assume a local edit has changed AWS. Reapplying uses stable resource names and
creates no duplicate schedules. The action does not wake a stopped database.

The public hours document is outside release snapshots, so application rollback
does not roll back operator schedules. Frontend notices reload it every minute
while visible; Sender reads it each tick. CloudFront invalidation is awaited
before schedules activate. Only public hours are published, never credentials or
the deployment configuration. Set `enabled` to `false` and rerun `hours-apply`
to remove the three EC2 schedules while retaining the backup and local-time
monthly-review schedules. Keep the block after first application; removing it
is rejected on application updates to prevent orphaned active schedules.

Scheduler uses the configured IANA zone, including EST/EDT changes. Fixed `EST`
and unknown zones are rejected. Daily times in a known spring-forward gap are
rejected against the next five years of installed timezone rules on each apply;
reapply if timezone legislation changes. On fall-back days Scheduler invokes a
repeated wall-clock time only once, before the clock change. Thus the 01:00 safety
stop runs once. Toronto's usual 11-hour overnight interval becomes 10 or 12 hours
on transition nights. See [AWS Scheduler time-zone behavior](https://docs.aws.amazon.com/scheduler/latest/UserGuide/schedule-types.html).
Validation also checks the actual UTC elapsed intervals on transition dates so
a skipped hour cannot consume the backup margin or safety-stop clearance.

These are discrete actions. A manual stop during the day stays stopped until the
next start. A manual start after 22:00 is caught by 01:00; a start after 01:00 runs
until that calendar day's 22:00 stop. There is no extension or draining mechanism.
Normal OS shutdown ends in-flight connections and rolls back uncommitted work;
committed database contents and EBS storage remain. No forced stop or termination
is used. Each action has zero retries and a 60-second event age; no overnight retry
can spill into the next operating period. Verify real EC2 already-running/stopped
API responses during the controlled rollout; local tests do not prove them.

Manual operations use the supported commands:

```sh
make deploy ACTION=database-start
make deploy ACTION=database-stop
```

Start waits at most five minutes for a database-backed Bootstrap migration-state
read and prints measured readiness seconds. Start time is not a promise of
immediate availability. PostgreSQL starts through its enabled systemd service;
the existing private network interface/IP is retained across stop/start. Stop
waits for normal EC2 shutdown. Deploy/update/bootstrap/backup-configure refuse a
stopped database before their database-dependent work; start and wait first.
`backup-status` and isolated `restore-verify` remain usable while it is stopped.

The 21:00 backup leaves 15 minutes for existing timer jitter and 45 minutes for
dump/upload before 22:00. The service has a 45-minute timeout. Its status records
actual duration. Freshness uses the latest due local backup plus that margin,
not a fixed 24-hour age that falsely alarms at DST changes. `Persistent=true`
remains: a missed 21:00 backup catches up at the next 09:00 boot and may compete
with morning traffic. The compatibility default `03:17 UTC` is rejected when
incompatible hours are configured. Verify actual production dump duration and
the installed timer after configuration; do not infer historical duration from
the new margin.

Monthly reviews publish daily 30 minutes after start (09:30 by default) in the
same zone. The old 05:15 UTC rule is disabled; Scheduler invokes Delivery using
its IAM role. Each invocation still processes at most 25 group/month candidates;
failure or remaining backlog catches up at the next daily run, without an
unbounded same-day retry. Publication deduplication and seven-day review push
expiry are unchanged. Sender quietly skips all off-hours ticks, including after
a manual out-of-hours start; it resumes normal database-backed claiming in hours.
In-hours failures remain Lambda errors in Sender/Delivery logs; they do not use
the Worker/OCR Discord subscriptions. Expense pushes still expire after 15
minutes, so pending pushes may expire overnight without losing committed
expenses. Asynchronous OCR usage telemetry remains best effort and may exhaust
retries during downtime; never repeat paid OCR to recover telemetry. The host
database-alert timer is silent while stopped, as expected.

During API/network failures the frontend shows possible scheduled downtime only
outside configured hours, otherwise unexpected service unavailability, with
administrator guidance. It preserves authentication on temporary refresh failures
and keeps an open authenticated page mounted. Initial PWA/browser/notification
entry shows a recovery screen at the original URL. Visible pages make at most
three automatic session checks at 15-second intervals, then require Try again or
foreground/online recovery. After the service recovers, Reload page refreshes
missing data explicitly; copy unsaved entries first, or choose Keep this page.
No mutation is automatically retried after a service
failure; inspect the saved result before resubmitting. Expired receipt attachments
need re-uploading. A successful manual out-of-hours start permits normal HTTP use.

Before enabling the schedule for production, perform a controlled normal stop/
start during a maintenance window. Record actual EC2 already-running/stopped API
responses, readiness duration, PostgreSQL automatic startup, and consecutive
authenticated reads plus a representative write after restart to check warm
pooled-connection recovery. Manually verify the notice/retry, preserved session,
notification entry, and foreground recovery on iOS Safari/PWA and macOS Safari.
Automated browser checks supplement these checks.

For failures, inspect Scheduler `TargetErrorCount`, `InvocationDroppedCount` and
`InvocationThrottleCount` metrics in `AWS/Scheduler` for the database-hours group;
inspect the group's `database-hours-failures` SQS queue (14-day retention). A
Scheduler Lambda target's successful acceptance does not establish successful
publication: also inspect Delivery Lambda errors. For a failed start/readiness,
check EC2 state/status checks and the PostgreSQL systemd journal, then rerun
`database-start`. For a failed stop, inspect state and retry `database-stop`;
never force-stop as an automatic recovery step. No extra always-on availability
service is introduced. Reduced compute runtime does not eliminate storage or
other ongoing charges; no dollar savings have been measured.

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

Frontend render failures (the occurrence ID is random correlation metadata;
the browser does not send exception messages, stacks, URLs, or form data):

```text
fields @timestamp, request_id, occurrence_id, route, error_category
| filter event = "frontend_render_error"
| sort @timestamp desc
| limit 100
```

The frontend submits this event only for authenticated production sessions.
The endpoint uses the normal CSRF and trusted-origin boundary, accepts only a
bounded UUID occurrence ID, suppresses same-user bursts and replayed IDs, and
has a dedicated API Gateway throttle. Reporting is best-effort and never
changes authentication or recovery behavior. Pre-authentication render errors
are intentionally not reported because the application does not expose an
anonymous alert-producing endpoint.

For an authorized smoke check, use a dedicated test account in a production
browser session and submit only a newly generated UUID through the normal
CSRF-protected endpoint. Confirm the first request returns `202`, an immediate
repeat returns `204`, and the Logs Insights query above finds exactly one
sanitized event. Then confirm Discord includes the occurrence ID but no browser
exception text, stack, URL, account identity, or request payload. Do not use a
real production exception or paste diagnostic content into the request.

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
- The protected deploy config owns a dedicated OCR capability secret. The
  deployer projects it only to Worker and OCR runtime environments; Terraform
  and release manifests never receive it.
- The Worker begins at reserved concurrency `0`; Python publishes runtime configuration and activates it at `5`. With the deployed `DB_MAX_OPEN_CONNS=2`, the Worker has a maximum application-side database pool budget of 10 connections.
- The OCR Lambda begins at reserved concurrency `0`; Python publishes its
  minimal environment and activates it at `1`. It has no VPC attachment,
  database variables, NAT Gateway, or paid VPC endpoint. Its data-plane
  permissions cover the conditional DynamoDB replay claim and narrowly scoped
  temporary S3 writes. Receipt images are validated and re-encoded before a
  bounded synchronous Gemini request. The prompt and extraction schema are
  versioned with the backend; model output is validated and remains editable.
  Gemini HTTP and handler provider-stage timeouts are 30 seconds. Lambda allows
  60 seconds for preprocessing and optional storage; the browser's overall
  capability/upload/response timer is also 60 seconds. API Gateway's HTTP API
  integration still caps the entire OCR request at
  [30 seconds](https://docs.aws.amazon.com/apigateway/latest/developerguide/http-api-quotas.html). These allowances
  cannot guarantee a full 30 seconds of model time through the public endpoint:
  a slow synchronous scan can return a gateway timeout while Lambda finishes.
  Direct provider benchmarks measure that boundary separately from browser/API
  latency. Do not automatically retry a timed-out generation; its billing and
  completion may be unknown.
- Gemini provider failures and timeouts emit content-free structured outcomes
  and the `OCRProviderFailure` metric. Full-path latency, usage and receipt
  correctness must be measured with the selected model before claiming quality
  or cost improvements; historical Textract per-page prices do not apply.
- The raw execute-api endpoint is disabled only after custom-domain and frontend checks pass.
- The private receipt bucket is unversioned, encrypted and blocked from public
  access. Temporary photos expire after one day. The Worker finalizes an explicit
  retention choice and authorizes each short-lived receipt view. Images and URLs
  stay outside application caches; metadata removal is not automatic pixel
  redaction. `backend.receipt_storage_enabled` controls retention in both Worker
  and OCR runtime projections; it defaults to `false` for existing configs.
- Improvement collection is retired. The historical evidence bucket, schema and
  operator replay role remain for cleanup and offline investigation. New evidence
  writes are denied. Application cleanup hooks are retired; historical storage
  and its IAM grants remain for a separately reviewed infrastructure retirement. Normal
  Gemini scanning and expense receipt photos do not collect improvement samples.

- Post-cutover API verification allows up to 60 seconds for API Gateway, Lambda alias, and invoke-permission propagation. It retries transient `5xx` health responses and verifies the OCR draft preflight specifically permits `POST` plus `Content-Type`, `Authorization`, `X-OCR-Account-ID`, and `X-OCR-Request-ID`; the pre-update health gate deliberately defers this new contract until the Worker has been promoted.
- Normal updates use narrowly targeted Terraform plans for supported API, CloudFront, notification, and database-support infrastructure changes; Lambda code and runtime environments remain owned by the deployment runtime after initial creation. Deletions are limited to the two retired invitation routes, the three retired improvement mutation routes, the obsolete notification HTTPS egress rule, and the eight conditional alerting resources when alerting is explicitly disabled. Replacement is allowed only for explicitly listed Lambda permissions and Worker/OCR log subscription filters whose destination changes during alias cutover; all unrelated replacements are rejected.
- Before an update and immediately after any targeted infrastructure apply, the deployer removes only unmanaged Worker/OCR/Bootstrap/Sender/Delivery/Error Notifier runtime environments if an AWS provider response persisted them into local state, then verifies that no configured protected value remains anywhere in Terraform artifacts.
- Destroy deletes Lambdas first, waits for their owned ENIs, then runs `terraform destroy -refresh=false`.
- A deployment failure keeps persistent resources for an explicit resume; it never performs automatic rollback or destroy.

## Receipt photo storage configuration

To enable saving receipt photos, add this boolean to the existing `backend`
object in your external deployment config:

```json
"receipt_storage_enabled": true
```

The deployer sets `RECEIPT_STORAGE_ENABLED=true` on both Worker and OCR, using
the Terraform-managed private receipt bucket. A missing bucket blocks enabled
runtime publication. Omitted settings default to `false`; quoted strings such as
`"true"` are rejected.

The VPC Worker also needs TCP 443 egress to the S3 prefix list through the existing
S3 gateway endpoint. Backend/all targeted updates include this narrow rule; no
general internet egress is required. IAM permissions and a gateway route alone
do not override a security group that allows only PostgreSQL. Without the rule,
receipt verification can time out after four seconds and return 503 before
expense creation. The same path is needed for finalization and photo cleanup.

Publish the updated runtime environments and versions using:

```bash
make deploy ACTION=update SCOPE=backend
```

On an existing installation, backend updates publish both Worker and OCR; changing
only the frontend or an unqualified Lambda environment does not update the live
aliases. Accounts with the `receipt_ocr` grant then receive
`capabilities.receiptRetention=true`, enabling Keep receipt and Add photo in the
expense editor. Verify scan/save/view/delete after rollout. Existing expenses
without a saved photo require attaching one; enabling storage does not recover
previously discarded images. Setting this flag to `false` prevents new retention;
it does not purge saved photos or disable authorized views of finalized receipts.

## Receipt bucket operator access

By default, the receipt bucket explicitly denies object reads from identities
other than the Worker role. An operator's IAM permissions alone cannot override
that deny. To grant named operators full S3 access to this bucket, add
`receipt_bucket_admin_arns` to the existing `aws` object in your external config:

```json
"receipt_bucket_admin_arns": ["<same-account IAM user or role ARN>"]
```

The default is an empty list. Only explicit user/role ARNs from the deployment
account are accepted; wildcard, root and cross-account principals are rejected.
Each configured principal receives `s3:*` on the receipt bucket and its objects,
including read, upload, deletion and bucket-policy management. This grants no
permissions to other buckets or AWS services. Public-access blocking and the
HTTPS-only restriction remain enabled, and the bucket is not made public.

Apply through the supported backend update workflow. If an emergency policy
change was made directly in AWS, add the matching allowlist entry before the next
Terraform deployment so it does not remove that grant. Removing an entry revokes
this bucket-policy grant and restores the object-read deny for that principal.

## Gemini OCR configuration and usage accounting

Set `backend.gemini_api_key` in the external deployment config before a backend
or all-scope deployment. An empty placeholder is accepted when preparing config,
but runtime publication is blocked until it is filled. Local OCR tooling uses
`GEMINI_API_KEY`. `backend.gemini_model` / `GEMINI_MODEL` select the model; the
current default is `gemini-3.8-flash`, listed as stable in the
[Gemini model catalogue](https://ai.google.dev/gemini-api/docs/models).
Deployment config is authoritative for deployed runtime values; local environment
values are for local tooling. The key is projected only into the OCR runtime,
never the Worker, frontend, Terraform state, or release manifests.

Each attempted Gemini HTTP call emits an `ocr_gemini_usage` JSON event in the OCR
Lambda log group. It contains `model`, `outcome`, `http_status`, latency,
`usage_status`, and `cost_status`; available usage adds `input_tokens`,
`output_tokens`, `thinking_tokens`, `cached_input_tokens`, `tool_input_tokens`,
and `total_tokens`. Input includes the image and prompt. Output counts exclude
thinking, which is logged separately and included in the cost estimate.
Rejected or truncated model output still logs usage when the complete provider
response includes valid usage metadata. Timeouts, HTTP errors, unreadable or
oversized responses may have unavailable usage; they are not reported as free.
Offline replay produces no billable-usage event.

Migration `000042` adds `ocr_usage` independently of receipts and expenses.
The deployer sets `OCR_USAGE_FUNCTION_NAME` to the Worker live alias; OCR's IAM
role permits invocation of that target only (unqualified during pre-alias
infrastructure preparation). The OCR runtime still has no database credentials.
Per-attempt records store account/request IDs, model, outcome, token counts,
latency, nullable estimated USD cost and pricing version. Raw images, prompts,
responses and credentials are excluded. `request_id` is unique, so retries do
not count a scan twice. Failed provider attempts retain whatever valid usage is
available; unavailable values remain NULL. No historical backfill is possible.

Submission waits up to two seconds for Lambda's asynchronous acceptance, then
returns to the scan flow. Processing failures retry under Lambda's bounded
asynchronous policy; `ocr_usage_enqueue_failed` and Worker errors must be
investigated. There is no durable failure queue in this implementation. An
accepted event is not proof of a completed database write, and cost totals may
be incomplete after exhausted retries or abrupt runtime termination. Logging
failure never triggers another paid provider request. Publish the compatible
Worker before enabling the new OCR version and verify a scan creates one row.

```sql
SELECT request_id, model, outcome, input_tokens, output_tokens, thinking_tokens,
       cached_input_tokens, total_tokens, estimated_cost_usd, cost_status,
       pricing_version, recorded_at
FROM ocr_usage
ORDER BY recorded_at DESC
LIMIT 20;
```


Successful draft responses also include an optional, per-request `usage`
summary. Receipt review shows it under a collapsed **Scan usage** disclosure,
with model, token counts, and an explicitly labeled USD estimate. This metadata
is transient: it clears on rescan/discard and is not saved to the expense or
included in persisted draft fields. Missing telemetry does not block
the receipt draft, and unavailable estimates are never displayed as zero.

For `gemini-3.8-flash`, numeric `estimated_cost_usd` uses the versioned standard
paid-tier [Google pricing](https://ai.google.dev/gemini-api/docs/pricing),
verified on 2026-10-20. The snapshot includes the published January 1, 2027 rate
transition. Cached input is charged once at its discounted rate; thinking is
charged at the output rate. Unknown models, nonstandard tiers, unsupported
tool usage, or missing/inconsistent usage omit the estimate and expose a
`cost_status` reason. Update the source-controlled pricing snapshot when
changing models or when Google revises its prices. No additional environment or
deployment config is required. This estimates Gemini generation charges only;
free-tier allowances, credits, discounts, taxes, and AWS/storage/log charges are
excluded. Google billing remains authoritative. Logs contain no receipt pixels,
merchant/item text, prompt, provider payload, or API key.

Example CloudWatch Logs Insights query for the OCR log group:

```text
fields @timestamp, event, model, input_tokens, output_tokens, thinking_tokens, estimated_cost_usd
| filter event = "ocr_gemini_usage" and cost_status = "estimated"
| stats count(*) as estimated_calls, sum(input_tokens) as input_tokens, sum(output_tokens) as output_tokens, sum(thinking_tokens) as thinking_tokens, sum(estimated_cost_usd) as estimated_gemini_usd by bin(1d), model
```

Check events with `cost_status != "estimated"` separately; the sum above excludes
those calls and must not be treated as a complete bill.

For provider HTTP errors, the same event includes allowlisted
`provider_error_status` and, when recognized, `provider_error_reason` (for
example `INVALID_ARGUMENT` / `API_KEY_INVALID`). Error bodies are bounded to
16 KiB; provider messages, request content, and error metadata are never logged.
The browser may see a generic 503 even when Gemini rejected the request with
400; inspect `http_status` and the safe category in the OCR log group before
assuming an upload problem or a temporary provider outage.

Keep the 200-item receipt limit in the server parser rather than adding
`maxItems: 200` to the schema sent to Gemini. In a controlled text-only check,
the full receipt schema returned 400 while the identical prompt/schema with
only that constraint removed returned 200. Plain generation and a one-field
JSON schema also succeeded. The server still accepts 200 items and rejects 201;
required fields, types, extra-field rejection, and money/date validation remain
enforced. This is a compatibility fix for this receipt schema/model combination,
not a claim that Gemini never supports `maxItems`.

Review the selected API tier's [data-use terms](https://ai.google.dev/gemini-api/terms)
before submitting receipts. Improvement sharing settings and evidence collection
are no longer part of the application. The obsolete
`backend.ocr_evidence_capture_enabled` config key is accepted but ignored for
compatibility with existing deployment files; remove it from your external config.
Generated templates omit it, and neither runtime receives a capture flag. The OCR
runtime receives no evidence bucket or write permission. The Worker no longer
receives `OCR_EVIDENCE_BUCKET` or queries the historical evidence tables. This
does not change Google's processing or retention terms.

Receipt-view URLs expire after 45 seconds and use `private, no-store`. A previously
issued URL remains a bearer capability until it expires. Existing group members
may view receipts in archived groups; restore the group before changing receipts.
Soft deletion hides the photo and schedules object cleanup. Permanent expense
purge must complete product receipt cleanup before removing associations.

Migration `000043` removes the historical improvement tables, two cleanup
triggers and their function. Deploy the compatible Worker without evidence
consumers before applying this contract release; the migration does not detect
which application binary is running. It takes brief exclusive locks on expense,
receipt and historical tables (subject to Bootstrap's five-second lock timeout),
rechecks both historical tables are empty, and performs the DDL atomically.
Nonempty tables or unexpected dependencies abort without dropping objects.

Use the supported `migrations`, `backend` or `all` update scope. After deployment,
verify `schema_migrations` reports version 43 without a dirty flag, both
`ocr_improvement_*` tables are absent, and receipt replacement/deletion and
`ocr_usage` insertion still work. A failed migration can leave the migration
version dirty even though its schema DDL rolled back; follow the migration
policy's reviewed recovery procedure rather than automatically forcing a version.
The down migration refuses automatic schema recreation. Any rollback binary must
be checked for retired-table consumers. Never edit applied migrations or use
`DROP ... CASCADE`. If historical rows exist, stop and plan retention/purge.
Historical S3 resources are unaffected, and offline replay remains file-only.

Offline replay accepts a locally downloaded Gemini response or compressed
historical evidence bundle. It needs no Gemini key, performs no provider call,
and writes normalized receipt data to stdout: keep that output in approved
private storage. Use the distinct replay role to read confirmed evidence;
normal receipt/application endpoints never expose bundles.

```bash
go run ./backend/cmd/ocr-evidence -action replay -file <private-local-response.json.gz>
```

Replay validates saved extraction through the supported adapter and current
schema. It does not rerun image understanding or measure a changed prompt/model.
Those comparisons require separately authorized, billable provider calls.


## Database connection alerts

Optional host monitoring checks all databases on the PostgreSQL instance once
per minute. At least 80% of `max_connections` for two consecutive fresh samples
emits ALARM; two readings below 80% emit recovery (OK). Both active and idle
client connections count, excluding the collector itself and internal PostgreSQL
workers. The threshold uses the configured total, not the smaller pool available
after administrative reserved slots. With `max_connections=20`, alarm begins at
16 clients; continue reserving connection budget for administration/migrations.

Two failed local queries emit CHECK_FAILED (usage unknown), and two subsequent
successful samples report the measured state. Normal checks do not create S3
objects or CloudWatch log entries. Local systemd journals still record service
activity/errors. This is not host-availability monitoring: stopped hosts, stopped
timers and broken S3 delivery require separate operator observation. Brief spikes
between samples may be missed.

Configure a new Discord webhook belonging to the desired database-alert channel
in the external deployment config:

```json
"observability": {
  "discord_webhook_url": "<existing application-error webhook>",
  "database_discord_webhook_url": "<database-alert channel webhook>"
}
```

Omit the new field or set it to null to disable the feature. The feature reuses
an existing enabled application error notifier, so its webhook must remain
configured. Identical webhook URLs are rejected; the operator must ensure the
new webhook belongs to a different channel. Both values are protected runtime
secrets, never Terraform variables, host files, logs or public frontend config.
Do not paste webhook URLs into support messages or commit them.

Deploy the backend/infrastructure first through the supported update workflow
(`ACTION=update SCOPE=all`). This publishes the compatible notifier, injects its
separate webhook and private bucket name, scopes host PUT/notifier GET permissions
to `database-alerts/*`, and adds an S3 notification targeting the notifier live
alias. Existing application-error subscriptions keep their existing channel.
Before applying, inspect the plan for expected scoped changes and no unrelated
replacement or deletion. Do not enable the timer before this deployment succeeds.

Install the timer using the same external config and supported deploy command:

```bash
make deploy ACTION=database-alerts-configure
```

This builds required local artifacts, verifies the existing database instance
profile, temporarily opens the existing guarded operator access path, installs
an immutable runtime script/unit, and removes temporary access in a finally
block. It requires the existing AWS CLI and PostgreSQL tools installed by database
setup. The host uses local peer authentication as postgres solely for the bounded
statistics query. No runtime application credentials are installed. Installation
must be separately authorized/performed by the operator.

The local state file has root-only permissions. Pending transition uploads are
retained across failures/reboots and retried with a stable event identifier.
S3 acceptance does not prove Discord delivery; S3/Lambda events are at-least-once,
can be delayed/reordered/duplicated, and exhausted retries are not a durable
failure queue. Notifications contain observation timestamps and event IDs.
Seven-day lifecycle expiration applies only to alert events; backup retention is
unchanged. These objects contain counts/status only, never account/query/expense
content. Healthy periodic checks incur no metric/alarm charge; transitions incur
small S3/Lambda/log usage plus any applicable data transfer.

To disable cleanly, set the new field to null, run
`ACTION=database-alerts-configure` to stop the host timer, then deploy the update
to remove only the four conditional alert resources/permissions and the separate
webhook. Local state remains for diagnosis; pending uploads are not discarded.
After changing the webhook channel, verify a controlled test in a non-production
environment before relying on production notifications.
