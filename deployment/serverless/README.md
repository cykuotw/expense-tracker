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

The JSON sections are `deployment`, `aws`, `database`, `backend`, `frontend`, optional `first_admin`, and `local_credentials`. This is the sole human-edited serverless deployment source. Do not create serverless `.tfvars`, PostgreSQL password files, or Lambda runtime JSON files; the deployer creates protected temporary projections and removes them.

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
        "db_conn_max_idle_time_seconds": 60
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

## Local build-tool provisioning

The machine that runs `make deploy` builds and tests the frontend locally before
publication. It must provide the following commands on `PATH` before invoking
the deployer:

- Node `22.13.1` through Node `22.x`; `frontend/.node-version` identifies the
  tested baseline and `frontend/package.json` enforces the supported range.
- pnpm `10.23.0`, as declared in `frontend/package.json`.

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
make deploy ACTION=backup-configure
make deploy ACTION=restore-verify
make deploy ACTION=backup-status
make deploy ACTION=backup-cleanup
make deploy ACTION=destroy
```

`make deploy` deploys/resumes an incomplete environment and runs migrations → backend → frontend when the environment is complete. Initial infrastructure requires typing the configured name prefix. Destroy requires typing `destroy-<name_prefix>`.

`SCOPE=backend` also publishes and invokes the bootstrap function first. This applies pending migrations and verifies first-administrator reconciliation before marker-dependent Worker code is published. Use `SCOPE=migrations` when only the migration/bootstrap step should run.

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

## Safety boundary

- Terraform never receives database/JWT/first-admin secrets and never manages Lambda environments.
- The Worker begins at reserved concurrency `0`; Python publishes runtime configuration and activates it at `3`.
- The raw execute-api endpoint is disabled only after custom-domain and frontend checks pass.
- Normal updates use narrowly targeted, non-destructive Terraform plans only for explicitly supported API and CloudFront infrastructure changes; Lambda code and runtime environments remain owned by the deployment runtime after initial creation. The invitation-security release has a precise allowlist for retiring its two obsolete token-bearing routes; this is not a general permission to delete infrastructure.
- Before an update, the deployer removes only unmanaged Worker/Bootstrap runtime environments if an AWS provider response persisted them into local state, then verifies that no configured protected value remains anywhere in Terraform artifacts.
- Destroy deletes Lambdas first, waits for their owned ENIs, then runs `terraform destroy -refresh=false`.
- A deployment failure keeps persistent resources for an explicit resume; it never performs automatic rollback or destroy.
